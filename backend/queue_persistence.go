package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const queueStateVersion = 2

type queueState struct {
	Version int            `json:"version"`
	Items   []DownloadItem `json:"items"`
	Paused  bool           `json:"paused"`
}

func (a *App) persistQueue() {
	if err := a.saveQueue(); err != nil {
		fmt.Printf("Erro ao salvar fila: %v\n", err)
	}
}

func (a *App) saveQueue() error {
	a.persistMu.Lock()
	defer a.persistMu.Unlock()

	items, paused := a.queueSnapshot()
	state := queueState{
		Version: queueStateVersion,
		Items:   items,
		Paused:  paused,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(a.queuePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tempPath := a.queuePath + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := replaceQueueFile(tempPath, a.queuePath); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}

func (a *App) queueSnapshot() ([]DownloadItem, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	items := make([]DownloadItem, 0, len(a.queueOrder))
	for _, id := range a.queueOrder {
		if item, ok := a.items[id]; ok {
			items = append(items, *item)
		}
	}
	return items, a.paused
}

func (a *App) loadQueue() error {
	data, err := os.ReadFile(a.queuePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var state queueState
	if err := json.Unmarshal(data, &state); err != nil {
		if preserveErr := preserveCorruptQueue(a.queuePath); preserveErr != nil {
			return fmt.Errorf("fila inválida: %v; não foi possível preservar o arquivo: %w", err, preserveErr)
		}
		return fmt.Errorf("fila inválida; arquivo preservado: %w", err)
	}
	if state.Version != 1 && state.Version != queueStateVersion {
		return fmt.Errorf("versão da fila não suportada: %d", state.Version)
	}

	items := make(map[string]*DownloadItem, len(state.Items))
	order := make([]string, 0, len(state.Items))
	changed := false
	for index := range state.Items {
		item := state.Items[index]
		if item.ID == "" {
			changed = true
			continue
		}
		if _, exists := items[item.ID]; exists {
			changed = true
			continue
		}
		if normalizeRestoredItem(&item, a.cacheDir) {
			changed = true
		}
		itemCopy := item
		items[item.ID] = &itemCopy
		order = append(order, item.ID)
	}

	a.mu.Lock()
	a.items = items
	a.queueOrder = order
	a.paused = state.Version >= 2 && state.Paused
	a.mu.Unlock()

	if changed || state.Version == 1 {
		return a.saveQueue()
	}
	return nil
}

func normalizeRestoredItem(item *DownloadItem, cacheDir string) bool {
	switch item.Status {
	case StatusFetching, StatusDownloading, StatusConverting:
		os.Remove(filepath.Join(cacheDir, item.ID+".tmp"))
		item.Status = StatusPending
		item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
		item.Error = ""
		item.StartedAt = ""
		item.CompletedAt = ""
		return true
	case StatusCompleted, StatusSkipped:
		if item.FilePath == "" {
			markMissingFile(item)
			return true
		}
		if _, err := os.Stat(item.FilePath); err != nil {
			markMissingFile(item)
			return true
		}
	}
	return false
}

func markMissingFile(item *DownloadItem) {
	item.Status = StatusFailed
	item.Error = "Arquivo MP3 não encontrado"
	item.FilePath = ""
	item.FileSize = 0
}

func preserveCorruptQueue(path string) error {
	suffix := time.Now().UTC().Format("20060102T150405.000000000Z")
	return os.Rename(path, path+".corrupt-"+suffix)
}

package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetConfig() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config
}

func (a *App) SaveConfig(config Config) (Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	a.mu.Lock()
	config = normalizeConfig(config, a.config)
	configPath := a.configPath
	a.mu.Unlock()

	if err := os.MkdirAll(config.DownloadDir, 0755); err != nil {
		return Config{}, err
	}
	if err := saveConfigFile(configPath, config); err != nil {
		return Config{}, err
	}

	a.mu.Lock()
	a.config = config
	a.mu.Unlock()
	return config, nil
}

func (a *App) SetLanguage(language string) {
	if language != "en-US" {
		language = "pt-BR"
	}

	a.configMu.Lock()
	a.mu.Lock()
	a.config.Language = language
	updatedIDs := make([]string, 0)
	for id, item := range a.items {
		if item.Status == StatusFailed {
			item.Error = TranslateStoredYouTubeError(item.Error, language)
			updatedIDs = append(updatedIDs, id)
		}
	}
	config := a.config
	configPath := a.configPath
	a.mu.Unlock()
	err := saveConfigFile(configPath, config)
	a.configMu.Unlock()
	if err != nil {
		fmt.Printf("Erro ao salvar idioma: %v\n", err)
	}

	a.persistQueue()
	for _, id := range updatedIDs {
		a.emitItemUpdate(id)
	}
}

func (a *App) SelectFolder() string {
	a.mu.Lock()
	defaultDirectory := a.config.DownloadDir
	a.mu.Unlock()
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Selecione a pasta de download",
		DefaultDirectory: defaultDirectory,
	})
	if err != nil {
		fmt.Printf("Erro ao selecionar pasta: %v\n", err)
		return ""
	}
	return folder
}

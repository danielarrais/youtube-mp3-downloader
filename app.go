package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	config     Config
	configPath string
	items      map[string]*DownloadItem
	queueOrder []string
	mu         sync.Mutex
	configMu   sync.Mutex
	persistMu  sync.Mutex
	cacheDir   string
	queuePath  string
	paused     bool
	activeID   string
	activeStop context.CancelFunc
	wakeWorker chan struct{}
}

func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(home, ".config")
	}
	return &App{
		config:     defaultConfig(home),
		configPath: filepath.Join(home, ".yt-mp3-downloader-config.json"),
		items:      make(map[string]*DownloadItem),
		queueOrder: make([]string, 0),
		cacheDir:   filepath.Join(home, ".yt-mp3-downloader-cache"),
		queuePath:  filepath.Join(configDir, "yt-mp3-go", "queue.json"),
		wakeWorker: make(chan struct{}, 1),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	config, err := loadConfigFile(a.configPath, a.config)
	if err != nil {
		fmt.Printf("Erro ao carregar configuração: %v\n", err)
	} else {
		a.config = config
	}
	if err := os.MkdirAll(a.config.DownloadDir, 0755); err != nil {
		fmt.Printf("Erro ao criar pasta de download: %v\n", err)
	}
	if err := os.MkdirAll(a.cacheDir, 0755); err != nil {
		fmt.Printf("Erro ao criar cache: %v\n", err)
	}
	if err := cleanupPartialFiles(a.config.DownloadDir); err != nil {
		fmt.Printf("Erro ao limpar arquivos parciais: %v\n", err)
	}
	if err := a.loadQueue(); err != nil {
		fmt.Printf("Erro ao carregar fila: %v\n", err)
	}
	go a.worker()
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	stop := a.activeStop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
}

func (a *App) AddDownloads(urls []string, quality string) []DownloadItem {
	a.mu.Lock()
	var newItems []DownloadItem
	for _, url := range urls {
		url = cleanYouTubeURL(url)
		item := &DownloadItem{
			ID:        uuid.New().String(),
			URL:       url,
			Quality:   quality,
			Status:    StatusPending,
			CreatedAt: time.Now().Format(time.RFC3339),
			Progress:  DownloadProgress{Percent: 0, Speed: "---", ETA: "---"},
		}
		a.items[item.ID] = item
		a.queueOrder = append(a.queueOrder, item.ID)
		newItems = append(newItems, *item)
	}
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
	return newItems
}

func cleanYouTubeURL(rawURL string) string {
	url := strings.TrimSpace(rawURL)
	if index := strings.IndexByte(url, '&'); index >= 0 {
		return url[:index]
	}
	return url
}

func (a *App) GetDownloads() []DownloadItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]DownloadItem, 0, len(a.queueOrder))
	for _, id := range a.queueOrder {
		if item, ok := a.items[id]; ok {
			items = append(items, *item)
		}
	}
	return items
}

func (a *App) GetStats() QueueStats {
	a.mu.Lock()
	defer a.mu.Unlock()
	stats := QueueStats{}
	stats.Paused = a.paused
	for _, item := range a.items {
		stats.Total++
		switch item.Status {
		case StatusPending:
			stats.Pending++
		case StatusFetching, StatusDownloading, StatusConverting:
			stats.Downloading++
		case StatusCompleted, StatusSkipped:
			stats.Completed++
		case StatusFailed:
			stats.Failed++
		}
	}
	return stats
}

func (a *App) worker() {
	for {
		a.mu.Lock()
		var targetItem *DownloadItem
		var workCtx context.Context
		if !a.paused {
			for _, id := range a.queueOrder {
				if item, ok := a.items[id]; ok && item.Status == StatusPending {
					targetItem = item
					workCtx, a.activeStop = context.WithCancel(context.Background())
					a.activeID = id
					item.Status = StatusFetching
					if item.StartedAt == "" {
						item.StartedAt = time.Now().Format(time.RFC3339)
					}
					break
				}
			}
		}
		a.mu.Unlock()
		if targetItem == nil {
			<-a.wakeWorker
			continue
		}
		a.persistQueue()
		a.emitItemUpdate(targetItem.ID)
		a.emitStats()
		a.processDownload(workCtx, targetItem)
		a.mu.Lock()
		if a.activeID == targetItem.ID {
			a.activeID = ""
			a.activeStop = nil
		}
		a.mu.Unlock()
	}
}

func (a *App) signalWorker() {
	select {
	case a.wakeWorker <- struct{}{}:
	default:
	}
}

func (a *App) isActiveItemLocked(id string) bool {
	item, ok := a.items[id]
	return ok && a.activeID == id &&
		item.Status != StatusCancelled && item.Status != StatusPending
}

func (a *App) setActiveItemStatus(id string, status DownloadStatus) bool {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok || a.activeID != id || item.Status == StatusCancelled || item.Status == StatusPending {
		a.mu.Unlock()
		return false
	}
	item.Status = status
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
	return true
}

func (a *App) cleanupInterruptedDownload(id, tempPath string) {
	if tempPath != "" {
		os.Remove(tempPath)
	}
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) updateError(id string, errMsg string) {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok || !a.isActiveItemLocked(id) {
		a.mu.Unlock()
		return
	}
	item.Status = StatusFailed
	item.Error = errMsg
	item.CompletedAt = time.Now().Format(time.RFC3339)
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) emitItemUpdate(id string) {
	if a.ctx == nil {
		return
	}
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return
	}
	val := *item
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "download:update", val)
}

func (a *App) emitStats() {
	if a.ctx == nil {
		return
	}
	s := a.GetStats()
	runtime.EventsEmit(a.ctx, "queue:stats", s)
}

func (a *App) CancelDownload(id string) {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return
	}
	item.Status = StatusCancelled
	stop := context.CancelFunc(nil)
	if a.activeID == id {
		stop = a.activeStop
	}
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) RetryDownload(id string) DownloadItem {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return DownloadItem{}
	}
	resetItemForRetry(item)
	result := *item
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
	a.signalWorker()
	return result
}

func (a *App) RetryFailed() {
	a.mu.Lock()
	for _, item := range a.items {
		if item.Status != StatusFailed {
			continue
		}
		resetItemForRetry(item)
	}
	a.paused = false
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
}

func (a *App) PauseQueue() {
	a.mu.Lock()
	if a.paused {
		a.mu.Unlock()
		return
	}
	a.paused = true
	activeID := a.activeID
	stop := a.activeStop
	if item, ok := a.items[activeID]; ok {
		resetItemForRetry(item)
	}
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	if activeID != "" {
		a.emitItemUpdate(activeID)
	}
	a.emitStats()
}

func resetItemForRetry(item *DownloadItem) {
	item.Status = StatusPending
	item.Error = ""
	item.StartedAt = ""
	item.CompletedAt = ""
	item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
}

func (a *App) currentLanguage() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config.Language
}

func (a *App) ResumeQueue() {
	a.mu.Lock()
	if !a.paused {
		a.mu.Unlock()
		return
	}
	a.paused = false
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
}
func (a *App) GetPlaylistInfo(url string) (PlaylistInfo, error) {
	return FetchPlaylistInfo(url)
}
func (a *App) ClearCompleted() {
	a.mu.Lock()
	newOrder := make([]string, 0)
	for _, id := range a.queueOrder {
		if a.items[id].Status != StatusCompleted && a.items[id].Status != StatusSkipped {
			newOrder = append(newOrder, id)
		} else {
			delete(a.items, id)
		}
	}
	a.queueOrder = newOrder
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
}
func (a *App) CancelAll() {
	a.mu.Lock()
	for _, item := range a.items {
		if item.Status == StatusPending ||
			item.Status == StatusFetching ||
			item.Status == StatusDownloading ||
			item.Status == StatusConverting {
			item.Status = StatusCancelled
		}
	}
	activeID := a.activeID
	stop := a.activeStop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	if activeID != "" {
		a.emitItemUpdate(activeID)
	}
	a.emitStats()
}
func (a *App) ClearAll() {
	a.mu.Lock()
	stop := a.activeStop
	a.items = make(map[string]*DownloadItem)
	a.queueOrder = make([]string, 0)
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	a.emitStats()
}

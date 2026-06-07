//go:build !web

package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.setEventHandlers(
		func(item DownloadItem) {
			runtime.EventsEmit(ctx, "download:update", item)
		},
		func(stats QueueStats) {
			runtime.EventsEmit(ctx, "queue:stats", stats)
		},
	)
	a.start()
}

func (a *App) shutdown(context.Context) {
	a.stop()
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

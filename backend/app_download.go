package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var convertAudioToMP3 = ConvertToMp3
var errPublishConvertedFile = errors.New("publish converted file")

func (a *App) processDownload(ctx context.Context, item *DownloadItem) {
	session := NewYouTubeSession()
	video, err := session.GetVideo(ctx, item.URL)
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, FormatYouTubeError(err, a.currentLanguage()))
		return
	}

	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		return
	}
	item.Title = video.Title
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)

	a.mu.Lock()
	downloadDir := a.config.DownloadDir
	a.mu.Unlock()
	outputPath := filepath.Join(downloadDir, SanitizeFilename(video.Title)+".mp3")
	outputPath = a.resolveOutputPath(item, video.ID, outputPath)

	if fileInfo, err := os.Stat(outputPath); err == nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		if !a.setActiveItemStatus(item.ID, StatusSkipped) {
			return
		}
		a.mu.Lock()
		if !a.isActiveItemLocked(item.ID) {
			a.mu.Unlock()
			return
		}
		item.FilePath = outputPath
		item.FileSize = fileInfo.Size()
		a.mu.Unlock()
		a.persistQueue()
		a.emitItemUpdate(item.ID)
		return
	} else if !os.IsNotExist(err) {
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		return
	}
	tempPath := filepath.Join(a.cacheDir, item.ID+".tmp")
	defer os.Remove(tempPath)

	lastProgressEvent := time.Time{}
	_, err = session.DownloadAudio(ctx, video, tempPath, func(percent float64, downloaded, total int64) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = percent
			item.Progress.DownloadedBytes = downloaded
			item.Progress.TotalBytes = total
		}
		a.mu.Unlock()
		now := time.Now()
		if percent >= 100 || lastProgressEvent.IsZero() || now.Sub(lastProgressEvent) >= 200*time.Millisecond {
			lastProgressEvent = now
			a.emitItemUpdate(item.ID)
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, FormatOperationError("download", err, a.currentLanguage()))
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusConverting) {
		return
	}
	if err := convertAndPublish(ctx, tempPath, outputPath, item.Quality); err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		operation := "conversion"
		if errors.Is(err, errPublishConvertedFile) {
			operation = "finalize"
		}
		a.updateError(item.ID, FormatOperationError(operation, err, a.currentLanguage()))
		return
	}

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		os.Remove(outputPath)
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}
	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		os.Remove(outputPath)
		return
	}
	item.Status = StatusCompleted
	item.FilePath = outputPath
	item.FileSize = fileInfo.Size()
	item.CompletedAt = time.Now().Format(time.RFC3339)
	item.Progress.Percent = 100
	a.mu.Unlock()

	a.persistQueue()
	a.emitItemUpdate(item.ID)
	a.emitStats()
}

func convertAndPublish(ctx context.Context, inputPath, outputPath, quality string) error {
	partPath := outputPath + ".part"
	defer os.Remove(partPath)
	if err := convertAudioToMP3(ctx, inputPath, partPath, quality); err != nil {
		return err
	}
	if err := publishConvertedFile(partPath, outputPath); err != nil {
		return fmt.Errorf("%w: %v", errPublishConvertedFile, err)
	}
	return nil
}

func cleanupPartialFiles(downloadDir string) error {
	matches, err := filepath.Glob(filepath.Join(downloadDir, "*.mp3.part"))
	if err != nil {
		return err
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func publishConvertedFile(partPath, outputPath string) error {
	if _, err := os.Stat(outputPath); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(partPath, outputPath)
}

func (a *App) resolveOutputPath(item *DownloadItem, videoID, basePath string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, queuedItem := range a.items {
		if queuedItem.ID != item.ID && queuedItem.URL != item.URL && queuedItem.FilePath == basePath {
			extension := filepath.Ext(basePath)
			return strings.TrimSuffix(basePath, extension) + " [" + videoID + "]" + extension
		}
	}
	return basePath
}

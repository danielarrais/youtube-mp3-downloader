package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newPersistenceTestApp(t *testing.T) *App {
	t.Helper()
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	return &App{
		items:      make(map[string]*DownloadItem),
		queueOrder: make([]string, 0),
		cacheDir:   cacheDir,
		queuePath:  filepath.Join(root, "config", "queue.json"),
		wakeWorker: make(chan struct{}, 1),
	}
}

func TestPausedQueuePersistsAndVersionOneMigrates(t *testing.T) {
	app := newPersistenceTestApp(t)
	app.paused = true
	if err := app.saveQueue(); err != nil {
		t.Fatal(err)
	}

	restored := newPersistenceTestApp(t)
	restored.queuePath = app.queuePath
	if err := restored.loadQueue(); err != nil {
		t.Fatal(err)
	}
	if !restored.paused || !restored.GetStats().Paused {
		t.Fatal("paused queue state was not restored")
	}

	versionOne := queueState{
		Version: 1,
		Items: []DownloadItem{{
			ID:     "pending",
			Status: StatusPending,
		}},
	}
	data, err := json.Marshal(versionOne)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.queuePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	migrated := newPersistenceTestApp(t)
	migrated.queuePath = app.queuePath
	if err := migrated.loadQueue(); err != nil {
		t.Fatal(err)
	}
	if migrated.paused {
		t.Fatal("version 1 queue should migrate as unpaused")
	}

	var saved queueState
	migratedData, err := os.ReadFile(app.queuePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(migratedData, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Version != queueStateVersion {
		t.Fatalf("expected migrated version %d, got %d", queueStateVersion, saved.Version)
	}
}

func TestPauseQueueCancelsActiveItemAndReturnsItToPending(t *testing.T) {
	app := newPersistenceTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	app.items["active"] = &DownloadItem{
		ID:        "active",
		Status:    StatusDownloading,
		StartedAt: "2026-01-01T00:00:00Z",
		Progress:  DownloadProgress{Percent: 42},
	}
	app.queueOrder = []string{"active"}
	app.activeID = "active"
	app.activeStop = cancel

	app.PauseQueue()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("PauseQueue did not cancel the active operation")
	}
	item := app.items["active"]
	if !app.paused || item.Status != StatusPending || item.Progress.Percent != 0 {
		t.Fatalf("active item was not reset for pause: paused=%v item=%#v", app.paused, item)
	}

	app.ResumeQueue()
	if app.paused {
		t.Fatal("ResumeQueue did not resume the queue")
	}
	select {
	case <-app.wakeWorker:
	default:
		t.Fatal("ResumeQueue did not wake the worker")
	}
}

func TestCancelDownloadCancelsActiveOperation(t *testing.T) {
	app := newPersistenceTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	app.items["active"] = &DownloadItem{ID: "active", Status: StatusDownloading}
	app.queueOrder = []string{"active"}
	app.activeID = "active"
	app.activeStop = cancel

	app.CancelDownload("active")

	select {
	case <-ctx.Done():
	default:
		t.Fatal("CancelDownload did not cancel the active operation")
	}
	if app.items["active"].Status != StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", app.items["active"].Status)
	}
}

func TestCancelAllCancelsActiveAndPendingItems(t *testing.T) {
	app := newPersistenceTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	app.items["active"] = &DownloadItem{ID: "active", Status: StatusConverting}
	app.items["pending"] = &DownloadItem{ID: "pending", Status: StatusPending}
	app.items["completed"] = &DownloadItem{ID: "completed", Status: StatusCompleted}
	app.queueOrder = []string{"active", "pending", "completed"}
	app.activeID = "active"
	app.activeStop = cancel

	app.CancelAll()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("CancelAll did not cancel the active operation")
	}
	if app.items["active"].Status != StatusCancelled ||
		app.items["pending"].Status != StatusCancelled {
		t.Fatalf("active and pending items should be cancelled: %#v", app.items)
	}
	if app.items["completed"].Status != StatusCompleted {
		t.Fatal("CancelAll should not change completed items")
	}
}

func TestRetryFailedRestartsAllFailuresAndResumesQueue(t *testing.T) {
	app := newPersistenceTestApp(t)
	app.paused = true
	app.items["failed-1"] = &DownloadItem{
		ID:          "failed-1",
		Status:      StatusFailed,
		Error:       "first error",
		StartedAt:   "2026-01-01T00:00:00Z",
		CompletedAt: "2026-01-01T00:01:00Z",
		Progress:    DownloadProgress{Percent: 50},
	}
	app.items["failed-2"] = &DownloadItem{
		ID:     "failed-2",
		Status: StatusFailed,
		Error:  "second error",
	}
	app.items["completed"] = &DownloadItem{ID: "completed", Status: StatusCompleted}

	app.RetryFailed()

	if app.paused {
		t.Fatal("RetryFailed did not resume the queue")
	}
	for _, id := range []string{"failed-1", "failed-2"} {
		item := app.items[id]
		if item.Status != StatusPending || item.Error != "" {
			t.Fatalf("failed item %q was not reset: %#v", id, item)
		}
		if item.StartedAt != "" || item.CompletedAt != "" || item.Progress.Percent != 0 {
			t.Fatalf("failed item %q retained old progress: %#v", id, item)
		}
	}
	if app.items["completed"].Status != StatusCompleted {
		t.Fatal("RetryFailed changed a completed item")
	}
	select {
	case <-app.wakeWorker:
	default:
		t.Fatal("RetryFailed did not wake the worker")
	}
}

func TestQueuePersistenceRoundTripPreservesOrderAndDuplicates(t *testing.T) {
	app := newPersistenceTestApp(t)
	completedFile := filepath.Join(t.TempDir(), "ready.mp3")
	if err := os.WriteFile(completedFile, []byte("mp3"), 0644); err != nil {
		t.Fatal(err)
	}

	app.items["first"] = &DownloadItem{
		ID:       "first",
		URL:      "https://www.youtube.com/watch?v=same",
		Title:    "Primeiro",
		Quality:  "192k",
		Status:   StatusCompleted,
		FilePath: completedFile,
		Progress: DownloadProgress{Percent: 100},
	}
	app.items["second"] = &DownloadItem{
		ID:       "second",
		URL:      "https://www.youtube.com/watch?v=same",
		Title:    "Segundo",
		Quality:  "320k",
		Status:   StatusCancelled,
		Progress: DownloadProgress{Speed: "---", ETA: "---"},
	}
	app.queueOrder = []string{"second", "first"}

	if err := app.saveQueue(); err != nil {
		t.Fatalf("saveQueue returned error: %v", err)
	}

	restored := newPersistenceTestApp(t)
	restored.queuePath = app.queuePath
	restored.cacheDir = app.cacheDir
	if err := restored.loadQueue(); err != nil {
		t.Fatalf("loadQueue returned error: %v", err)
	}

	items := restored.GetDownloads()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "second" || items[1].ID != "first" {
		t.Fatalf("queue order was not preserved: %#v", restored.queueOrder)
	}
	if items[0].URL != items[1].URL {
		t.Fatal("duplicate video URLs should remain as separate queue items")
	}
	if items[1].Status != StatusCompleted {
		t.Fatalf("existing completed file should remain completed, got %s", items[1].Status)
	}
}

func TestLoadQueueRestartsInterruptedDownloadAndRemovesTempFile(t *testing.T) {
	app := newPersistenceTestApp(t)
	tempPath := filepath.Join(app.cacheDir, "active.tmp")
	if err := os.WriteFile(tempPath, []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}
	app.items["active"] = &DownloadItem{
		ID:          "active",
		URL:         "https://www.youtube.com/watch?v=active",
		Status:      StatusDownloading,
		Error:       "old error",
		StartedAt:   "2026-01-01T00:00:00Z",
		CompletedAt: "2026-01-01T00:01:00Z",
		Progress: DownloadProgress{
			Percent:         50,
			DownloadedBytes: 50,
			TotalBytes:      100,
		},
	}
	app.queueOrder = []string{"active"}
	if err := app.saveQueue(); err != nil {
		t.Fatal(err)
	}

	restored := newPersistenceTestApp(t)
	restored.queuePath = app.queuePath
	restored.cacheDir = app.cacheDir
	if err := restored.loadQueue(); err != nil {
		t.Fatalf("loadQueue returned error: %v", err)
	}

	item := restored.items["active"]
	if item.Status != StatusPending || item.Progress.Percent != 0 {
		t.Fatalf("interrupted item was not reset: %#v", item)
	}
	if item.Error != "" || item.StartedAt != "" || item.CompletedAt != "" {
		t.Fatalf("interrupted item retained stale state: %#v", item)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary file should be removed, stat error: %v", err)
	}
}

func TestLoadQueueMarksMissingCompletedFileAsFailed(t *testing.T) {
	app := newPersistenceTestApp(t)
	app.items["missing"] = &DownloadItem{
		ID:       "missing",
		Status:   StatusCompleted,
		FilePath: filepath.Join(t.TempDir(), "missing.mp3"),
		FileSize: 123,
	}
	app.queueOrder = []string{"missing"}
	if err := app.saveQueue(); err != nil {
		t.Fatal(err)
	}

	restored := newPersistenceTestApp(t)
	restored.queuePath = app.queuePath
	restored.cacheDir = app.cacheDir
	if err := restored.loadQueue(); err != nil {
		t.Fatalf("loadQueue returned error: %v", err)
	}

	item := restored.items["missing"]
	if item.Status != StatusFailed || item.Error != "Arquivo MP3 não encontrado" {
		t.Fatalf("missing completed file was not marked failed: %#v", item)
	}
	if item.FilePath != "" || item.FileSize != 0 {
		t.Fatalf("missing file metadata was not cleared: %#v", item)
	}
}

func TestLoadQueuePreservesCorruptFileAndStartsEmpty(t *testing.T) {
	app := newPersistenceTestApp(t)
	if err := os.MkdirAll(filepath.Dir(app.queuePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.queuePath, []byte("{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	err := app.loadQueue()
	if err == nil || !strings.Contains(err.Error(), "arquivo preservado") {
		t.Fatalf("expected preserved corrupt queue error, got %v", err)
	}
	if len(app.items) != 0 || len(app.queueOrder) != 0 {
		t.Fatal("app should keep an empty queue after corrupt state")
	}
	matches, globErr := filepath.Glob(app.queuePath + ".corrupt-*")
	if globErr != nil || len(matches) != 1 {
		t.Fatalf("expected one preserved corrupt file, matches=%v err=%v", matches, globErr)
	}
	if _, statErr := os.Stat(app.queuePath); !os.IsNotExist(statErr) {
		t.Fatalf("original corrupt path should be moved, stat error: %v", statErr)
	}
}

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanYouTubeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "removes video parameters",
			url:  "https://www.youtube.com/watch?v=3CWL9WXYSWU&list=RD3CWL9WXYSWU&start_radio=1",
			want: "https://www.youtube.com/watch?v=3CWL9WXYSWU",
		},
		{
			name: "keeps playlist identifier",
			url:  "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi&index=2",
			want: "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi",
		},
		{
			name: "trims spaces",
			url:  "  https://youtu.be/3CWL9WXYSWU  ",
			want: "https://youtu.be/3CWL9WXYSWU",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanYouTubeURL(test.url); got != test.want {
				t.Fatalf("cleanYouTubeURL(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}

func TestConvertAndPublishRemovesPartialFileAfterFailure(t *testing.T) {
	originalConverter := convertAudioToMP3
	t.Cleanup(func() {
		convertAudioToMP3 = originalConverter
	})
	convertAudioToMP3 = func(_ context.Context, _, outputPath, _ string) error {
		if err := os.WriteFile(outputPath, []byte("partial"), 0644); err != nil {
			t.Fatal(err)
		}
		return errors.New("ffmpeg failed")
	}

	outputPath := filepath.Join(t.TempDir(), "song.mp3")
	err := convertAndPublish(context.Background(), "input.tmp", outputPath, "192k")
	if err == nil {
		t.Fatal("convertAndPublish() returned nil error")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("final output should not exist: %v", err)
	}
	if _, err := os.Stat(outputPath + ".part"); !os.IsNotExist(err) {
		t.Fatalf("partial output should be removed: %v", err)
	}
}

func TestConvertAndPublishPublishesSuccessfulConversion(t *testing.T) {
	originalConverter := convertAudioToMP3
	t.Cleanup(func() {
		convertAudioToMP3 = originalConverter
	})
	convertAudioToMP3 = func(_ context.Context, _, outputPath, _ string) error {
		return os.WriteFile(outputPath, []byte("mp3"), 0644)
	}

	outputPath := filepath.Join(t.TempDir(), "song.mp3")
	if err := convertAndPublish(context.Background(), "input.tmp", outputPath, "192k"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mp3" {
		t.Fatalf("final output = %q, want mp3", data)
	}
}

func TestPublishConvertedFileDoesNotReplaceExistingFile(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "song.mp3.part")
	outputPath := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(partPath, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := publishConvertedFile(partPath, outputPath); !os.IsExist(err) {
		t.Fatalf("publishConvertedFile() error = %v, want os.ErrExist", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("existing output was replaced: %q", data)
	}
}

func TestResolveOutputPathAddsVideoIDForQueueCollision(t *testing.T) {
	app := newPersistenceTestApp(t)
	basePath := filepath.Join(t.TempDir(), "same title.mp3")
	app.items["completed"] = &DownloadItem{
		ID:       "completed",
		URL:      "https://www.youtube.com/watch?v=first",
		FilePath: basePath,
	}
	current := &DownloadItem{
		ID:  "current",
		URL: "https://www.youtube.com/watch?v=second",
	}
	app.items[current.ID] = current

	got := app.resolveOutputPath(current, "second", basePath)
	want := filepath.Join(filepath.Dir(basePath), "same title [second].mp3")
	if got != want {
		t.Fatalf("resolveOutputPath() = %q, want %q", got, want)
	}
}

func TestCleanupPartialFilesRemovesOnlyMP3Parts(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "song.mp3.part")
	otherPath := filepath.Join(dir, "keep.tmp")
	for _, path := range []string{partPath, otherPath} {
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := cleanupPartialFiles(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf("partial MP3 was not removed: %v", err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestSaveConfigDefaultsInvalidQuality(t *testing.T) {
	app := NewApp()
	app.configPath = filepath.Join(t.TempDir(), "config.json")
	app.config = Config{DownloadDir: t.TempDir(), Quality: "192k", Language: "pt-BR"}
	config, err := app.SaveConfig(Config{
		DownloadDir: t.TempDir(),
		Quality:     "invalid",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Quality != "192k" {
		t.Fatalf("SaveConfig() quality = %q, want %q", config.Quality, "192k")
	}
}

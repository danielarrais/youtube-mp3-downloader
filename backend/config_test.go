package main

import (
	"path/filepath"
	"testing"
)

func TestConfigFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "settings.json")
	want := Config{
		DownloadDir: filepath.Join(t.TempDir(), "downloads"),
		Quality:     "320k",
		Language:    "en-US",
	}
	if err := saveConfigFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfigFile(path, defaultConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("loadConfigFile() = %#v, want %#v", got, want)
	}
}

func TestNormalizeConfigUsesDefaults(t *testing.T) {
	defaults := Config{DownloadDir: "/downloads", Quality: "192k", Language: "pt-BR"}
	got := normalizeConfig(Config{Quality: "invalid", Language: "invalid"}, defaults)
	if got != defaults {
		t.Fatalf("normalizeConfig() = %#v, want %#v", got, defaults)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DownloadDir string `json:"download_dir"`
	Quality     string `json:"quality"`
	Language    string `json:"language"`
}

func defaultConfig(home string) Config {
	return Config{
		DownloadDir: filepath.Join(home, "Downloads", "YT-MP3"),
		Quality:     "192k",
		Language:    "pt-BR",
	}
}

func normalizeConfig(config, defaults Config) Config {
	if config.DownloadDir == "" {
		config.DownloadDir = defaults.DownloadDir
	}
	switch config.Quality {
	case "128k", "192k", "320k":
	default:
		config.Quality = defaults.Quality
	}
	if config.Language != "en-US" && config.Language != "pt-BR" {
		config.Language = defaults.Language
	}
	return config
}

func loadConfigFile(path string, defaults Config) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults, nil
	}
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("configuração inválida: %w", err)
	}
	return normalizeConfig(config, defaults), nil
}

func saveConfigFile(path string, config Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

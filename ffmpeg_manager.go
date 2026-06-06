package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	ffmpegMu         sync.Mutex
	cachedFFmpegPath string
)

// CheckAndDownloadFFmpeg garante que o FFmpeg esteja disponível na pasta local do app
func CheckAndDownloadFFmpeg() (string, error) {
	ffmpegMu.Lock()
	defer ffmpegMu.Unlock()
	if cachedFFmpegPath != "" {
		return cachedFFmpegPath, nil
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	if executablePath, err := os.Executable(); err == nil {
		bundledPath := filepath.Join(filepath.Dir(executablePath), "ffmpeg"+ext)
		if _, err := os.Stat(bundledPath); err == nil {
			cachedFFmpegPath = bundledPath
			return bundledPath, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar a pasta do usuário: %w", err)
	}
	// Pasta de binários privada do App
	binDir := filepath.Join(home, ".yt-mp3-downloader-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar a pasta de binários: %w", err)
	}

	ffmpegPath := filepath.Join(binDir, "ffmpeg"+ext)

	// Se já existe, retorna o caminho absoluto
	if _, err := os.Stat(ffmpegPath); err == nil {
		cachedFFmpegPath = ffmpegPath
		return ffmpegPath, nil
	}

	fmt.Printf(">>> FFmpeg não encontrado em %s. Usando versão do sistema...\n", ffmpegPath)

	if systemPath, err := exec.LookPath("ffmpeg" + ext); err == nil {
		cachedFFmpegPath = systemPath
		return systemPath, nil
	}

	return "", fmt.Errorf("ffmpeg não encontrado")
}

//go:build !web

package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

func (a *App) OpenFolder(path string) error {
	return openDirectory(filepath.Dir(path))
}

func (a *App) OpenDirectory(path string) error {
	return openDirectory(path)
}

func openDirectory(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

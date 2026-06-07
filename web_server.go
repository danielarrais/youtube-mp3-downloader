//go:build web

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type webServer struct {
	app         *App
	assets      fs.FS
	downloadDir string
}

type addDownloadsRequest struct {
	URLs    []string `json:"urls"`
	Quality string   `json:"quality"`
}

type languageRequest struct {
	Language string `json:"language"`
}

type apiError struct {
	Error string `json:"error"`
}

func newWebHandler(app *App, assets fs.FS, downloadDir string) http.Handler {
	server := &webServer{app: app, assets: assets, downloadDir: downloadDir}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /api/downloads", server.getDownloads)
	mux.HandleFunc("POST /api/downloads", server.addDownloads)
	mux.HandleFunc("POST /api/downloads/{id}/cancel", server.cancelDownload)
	mux.HandleFunc("POST /api/downloads/{id}/retry", server.retryDownload)
	mux.HandleFunc("GET /api/downloads/{id}/file", server.downloadFile)
	mux.HandleFunc("GET /api/stats", server.getStats)
	mux.HandleFunc("GET /api/playlist", server.getPlaylist)
	mux.HandleFunc("POST /api/queue/pause", server.pauseQueue)
	mux.HandleFunc("POST /api/queue/resume", server.resumeQueue)
	mux.HandleFunc("POST /api/queue/cancel-all", server.cancelAll)
	mux.HandleFunc("POST /api/queue/retry-failed", server.retryFailed)
	mux.HandleFunc("POST /api/queue/clear-completed", server.clearCompleted)
	mux.HandleFunc("POST /api/queue/clear-all", server.clearAll)
	mux.HandleFunc("GET /api/config", server.getConfig)
	mux.HandleFunc("PUT /api/config", server.saveConfig)
	mux.HandleFunc("PUT /api/language", server.setLanguage)
	mux.HandleFunc("/", server.frontend)
	return mux
}

func (s *webServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *webServer) getDownloads(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.GetDownloads())
}

func (s *webServer) addDownloads(w http.ResponseWriter, r *http.Request) {
	var request addDownloadsRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	if len(request.URLs) == 0 {
		writeAPIError(w, http.StatusBadRequest, errors.New("urls is required"))
		return
	}
	for _, rawURL := range request.URLs {
		if strings.TrimSpace(rawURL) == "" {
			writeAPIError(w, http.StatusBadRequest, errors.New("urls cannot contain empty values"))
			return
		}
	}
	if request.Quality == "" {
		request.Quality = s.app.GetConfig().Quality
	}
	switch request.Quality {
	case "128k", "192k", "320k":
	default:
		writeAPIError(w, http.StatusBadRequest, errors.New("invalid quality"))
		return
	}
	writeJSON(w, http.StatusCreated, s.app.AddDownloads(request.URLs, request.Quality))
}

func (s *webServer) cancelDownload(w http.ResponseWriter, r *http.Request) {
	if !s.downloadExists(r.PathValue("id")) {
		writeAPIError(w, http.StatusNotFound, errors.New("download not found"))
		return
	}
	s.app.CancelDownload(r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) retryDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.downloadExists(id) {
		writeAPIError(w, http.StatusNotFound, errors.New("download not found"))
		return
	}
	writeJSON(w, http.StatusOK, s.app.RetryDownload(id))
}

func (s *webServer) downloadFile(w http.ResponseWriter, r *http.Request) {
	item, ok := s.downloadByID(r.PathValue("id"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, errors.New("download not found"))
		return
	}
	if item.Status != StatusCompleted && item.Status != StatusSkipped {
		writeAPIError(w, http.StatusConflict, errors.New("download is not complete"))
		return
	}
	path, err := secureDownloadPath(s.downloadDir, item.FilePath)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, errors.New("file not found"))
		return
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filepath.Base(path),
	}))
	http.ServeFile(w, r, path)
}

func (s *webServer) getStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.GetStats())
}

func (s *webServer) getPlaylist(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeAPIError(w, http.StatusBadRequest, errors.New("url is required"))
		return
	}
	playlist, err := s.app.GetPlaylistInfo(rawURL)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, playlist)
}

func (s *webServer) pauseQueue(w http.ResponseWriter, _ *http.Request) {
	s.app.PauseQueue()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) resumeQueue(w http.ResponseWriter, _ *http.Request) {
	s.app.ResumeQueue()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) cancelAll(w http.ResponseWriter, _ *http.Request) {
	s.app.CancelAll()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) retryFailed(w http.ResponseWriter, _ *http.Request) {
	s.app.RetryFailed()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) clearCompleted(w http.ResponseWriter, _ *http.Request) {
	s.app.ClearCompleted()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) clearAll(w http.ResponseWriter, _ *http.Request) {
	s.app.ClearAll()
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.GetConfig())
}

func (s *webServer) saveConfig(w http.ResponseWriter, r *http.Request) {
	var config Config
	if err := decodeJSON(r, &config); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	config.DownloadDir = s.downloadDir
	saved, err := s.app.SaveConfig(config)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *webServer) setLanguage(w http.ResponseWriter, r *http.Request) {
	var request languageRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	s.app.SetLanguage(request.Language)
	w.WriteHeader(http.StatusNoContent)
}

func (s *webServer) frontend(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeAPIError(w, http.StatusNotFound, errors.New("endpoint not found"))
		return
	}
	requestPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
	if requestPath == "." || requestPath == "" {
		requestPath = "index.html"
	}
	if _, err := fs.Stat(s.assets, requestPath); err != nil {
		requestPath = "index.html"
	}
	data, err := fs.ReadFile(s.assets, requestPath)
	if err != nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}
	if contentType := mime.TypeByExtension(filepath.Ext(requestPath)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = w.Write(data)
}

func (s *webServer) downloadExists(id string) bool {
	_, ok := s.downloadByID(id)
	return ok
}

func (s *webServer) downloadByID(id string) (DownloadItem, bool) {
	s.app.mu.Lock()
	defer s.app.mu.Unlock()
	item, ok := s.app.items[id]
	if !ok {
		return DownloadItem{}, false
	}
	return *item, true
}

func secureDownloadPath(downloadDir, filePath string) (string, error) {
	root, err := filepath.EvalSymlinks(downloadDir)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("file is outside download directory")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("file is not regular")
	}
	return path, nil
}

func decodeJSON(r *http.Request, destination any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON: multiple values")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, apiError{Error: err.Error()})
}

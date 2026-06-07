package main

type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusFetching    DownloadStatus = "fetching_info"
	StatusDownloading DownloadStatus = "downloading"
	StatusConverting  DownloadStatus = "converting"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
	StatusSkipped     DownloadStatus = "skipped"
)

type DownloadProgress struct {
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	Speed           string  `json:"speed"`
	ETA             string  `json:"eta"`
}

type DownloadItem struct {
	ID          string           `json:"id"`
	URL         string           `json:"url"`
	Title       string           `json:"title"`
	Quality     string           `json:"quality"`
	Status      DownloadStatus   `json:"status"`
	Progress    DownloadProgress `json:"progress"`
	Error       string           `json:"error,omitempty"`
	FilePath    string           `json:"file_path,omitempty"`
	FileSize    int64            `json:"file_size,omitempty"`
	CreatedAt   string           `json:"created_at"`
	StartedAt   string           `json:"started_at,omitempty"`
	CompletedAt string           `json:"completed_at,omitempty"`
}

type QueueStats struct {
	Total       int  `json:"total"`
	Pending     int  `json:"pending"`
	Downloading int  `json:"downloading"`
	Completed   int  `json:"completed"`
	Failed      int  `json:"failed"`
	Paused      bool `json:"paused"`
}

type PlaylistInfo struct {
	ID     string          `json:"id"`
	Title  string          `json:"title"`
	Author string          `json:"author"`
	Videos []PlaylistVideo `json:"videos"`
}

type PlaylistVideo struct {
	ID                string `json:"id"`
	URL               string `json:"url"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	DurationSeconds   int    `json:"duration_seconds"`
	ThumbnailURL      string `json:"thumbnail_url"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	Index             int    `json:"index"`
}

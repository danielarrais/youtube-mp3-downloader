export type DownloadStatus =
  | 'pending'
  | 'fetching_info'
  | 'downloading'
  | 'converting'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'skipped';

export interface DownloadProgress {
  percent: number;
  downloaded_bytes: number;
  total_bytes: number;
  speed: string;
  eta: string;
}

export interface DownloadItem {
  id: string;
  url: string;
  title: string;
  status: DownloadStatus;
  progress: DownloadProgress;
  quality: string;
  file_path?: string;
  file_size?: number;
  error?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface QueueStats {
  total: number;
  pending: number;
  downloading: number;
  completed: number;
  failed: number;
  paused: boolean;
}

export interface PlaylistVideo {
  id: string;
  url: string;
  title: string;
  author: string;
  duration_seconds: number;
  thumbnail_url: string;
  available: boolean;
  unavailable_reason?: string;
  index: number;
}

export interface PlaylistInfo {
  id: string;
  title: string;
  author: string;
  videos: PlaylistVideo[];
}

export interface Config {
  download_dir: string;
  quality: string;
  language: string;
}

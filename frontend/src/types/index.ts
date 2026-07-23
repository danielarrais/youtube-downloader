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
  media_type?: 'audio' | 'video';
  video_format?: VideoFormat;
  file_path?: string;
  file_size?: number;
  error?: string;
  thumbnail_url?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface VideoFormat {
  video_itag: number;
  audio_itag?: number;
  container: string;
  extension: string;
  resolution: string;
  fps?: number;
  video_codec?: string;
  audio_codec?: string;
  size?: number;
  label: string;
}

export interface VideoInfo {
  title: string;
  thumbnail_url?: string;
  formats: VideoFormat[];
}

export interface VideoDownloadRequest {
  url: string;
  format: VideoFormat;
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
  video_container: 'mp4' | 'webm' | 'mkv';
  video_quality: '144p' | '240p' | '360p' | '480p' | '720p' | '1080p' | '1440p' | '2160p';
  language: string;
}

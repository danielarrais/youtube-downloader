import {
  Config,
  DownloadItem,
  PlaylistInfo,
  QueueStats,
  VideoDownloadRequest,
  VideoInfo,
} from '../types';

export interface SubscriptionHandlers {
  onDownloads: (items: DownloadItem[]) => void;
  onItem: (item: DownloadItem) => void;
  onStats: (stats: QueueStats) => void;
}

export interface AppAPI {
  capabilities: {
    nativeFolders: boolean;
  };
  addDownloads(urls: string[], quality: string): Promise<DownloadItem[]>;
  addVideoDownloads(requests: VideoDownloadRequest[]): Promise<DownloadItem[]>;
  getVideoFormats(url: string): Promise<VideoInfo>;
  getDownloads(): Promise<DownloadItem[]>;
  getStats(): Promise<QueueStats>;
  cancelDownload(id: string): Promise<void>;
  retryDownload(id: string): Promise<DownloadItem>;
  retryFailed(): Promise<void>;
  getPlaylistInfo(url: string): Promise<PlaylistInfo>;
  clearCompleted(): Promise<void>;
  cancelAll(): Promise<void>;
  pauseQueue(): Promise<void>;
  resumeQueue(): Promise<void>;
  clearAll(): Promise<void>;
  getConfig(): Promise<Config>;
  selectFolder(): Promise<string>;
  saveConfig(config: Config): Promise<Config>;
  setLanguage(language: string): Promise<void>;
  openDirectory(path: string): Promise<void>;
  openDownload(item: DownloadItem): Promise<void>;
  subscribe(handlers: SubscriptionHandlers): () => void;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: unknown;
      };
    };
  }
}

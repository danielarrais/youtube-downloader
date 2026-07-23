import * as WailsApp from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { Config, DownloadItem, PlaylistInfo, QueueStats, VideoDownloadRequest, VideoInfo } from '../types';
import { AppAPI } from './types';

export const wailsApi: AppAPI = {
  capabilities: { nativeFolders: true },
  addDownloads: (urls, quality) => WailsApp.AddDownloads(urls, quality) as Promise<DownloadItem[]>,
  addVideoDownloads: (requests: VideoDownloadRequest[]) => WailsApp.AddVideoDownloads(requests as never) as Promise<DownloadItem[]>,
  getVideoFormats: (url) => WailsApp.GetVideoFormats(url) as Promise<VideoInfo>,
  getDownloads: () => WailsApp.GetDownloads() as Promise<DownloadItem[]>,
  getStats: () => WailsApp.GetStats() as Promise<QueueStats>,
  cancelDownload: WailsApp.CancelDownload,
  removeDownload: WailsApp.RemoveDownload,
  retryDownload: (id) => WailsApp.RetryDownload(id) as Promise<DownloadItem>,
  retryFailed: WailsApp.RetryFailed,
  getPlaylistInfo: (url) => WailsApp.GetPlaylistInfo(url) as Promise<PlaylistInfo>,
  clearCompleted: WailsApp.ClearCompleted,
  cancelAll: WailsApp.CancelAll,
  pauseQueue: WailsApp.PauseQueue,
  resumeQueue: WailsApp.ResumeQueue,
  clearAll: WailsApp.ClearAll,
  getConfig: () => WailsApp.GetConfig() as Promise<Config>,
  selectFolder: WailsApp.SelectFolder,
  saveConfig: (config) => WailsApp.SaveConfig(config) as Promise<Config>,
  setLanguage: WailsApp.SetLanguage,
  openDirectory: WailsApp.OpenDirectory,
  openDownload: async (item) => {
    await WailsApp.OpenFolder(item.file_path!);
  },
  subscribe: ({ onItem, onStats }) => {
    const stopItems = EventsOn('download:update', onItem);
    const stopStats = EventsOn('queue:stats', onStats);
    return () => {
      stopItems();
      stopStats();
    };
  },
};

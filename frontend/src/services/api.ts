// Importa as funções geradas pelo Wails
import * as WailsApp from '../../wailsjs/go/main/App';

export const api = {
  addDownloads: async (urls: string[], quality: string) => {
    return await WailsApp.AddDownloads(urls, quality);
  },

  getDownloads: async () => {
    return await WailsApp.GetDownloads();
  },

  cancelDownload: async (id: string) => {
    return await WailsApp.CancelDownload(id);
  },

  retryDownload: async (id: string) => {
    return await WailsApp.RetryDownload(id);
  },

  retryFailed: async () => {
    return await (WailsApp as any).RetryFailed();
  },

  getStats: async () => {
    return await WailsApp.GetStats();
  },

  getPlaylistInfo: async (url: string) => {
    return await WailsApp.GetPlaylistInfo(url);
  },

  clearCompleted: async () => {
    return await WailsApp.ClearCompleted();
  },

  cancelAll: async () => {
    return await WailsApp.CancelAll();
  },

  pauseQueue: async () => {
    return await WailsApp.PauseQueue();
  },

  resumeQueue: async () => {
    return await WailsApp.ResumeQueue();
  },

  clearAll: async () => {
    return await WailsApp.ClearAll();
  },

  openFolder: async (path: string) => {
    return await WailsApp.OpenFolder(path);
  },

  openDirectory: async (path: string) => {
    return await (WailsApp as any).OpenDirectory(path);
  },

  // Funções de Configuração e Pasta
  getConfig: async () => {
    return await (WailsApp as any).GetConfig();
  },

  selectFolder: async () => {
    return await (WailsApp as any).SelectFolder();
  },

  saveConfig: async (config: { download_dir: string; quality: string }) => {
    return await (WailsApp as any).SaveConfig(config);
  },

  setLanguage: async (language: string) => {
    return await (WailsApp as any).SetLanguage(language);
  },

  getFileUrl: (filename: string) => ``, 
  deleteAllFiles: async () => {
    return await WailsApp.ClearAll();
  },
};

export const WS_URL = ""; 

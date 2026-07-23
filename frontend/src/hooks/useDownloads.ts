import { useState, useCallback, useEffect } from 'react';
import { api } from '../services/api';
import { DownloadItem, QueueStats, VideoDownloadRequest } from '../types';
import { useTranslation } from './useTranslation';

export function useDownloads() {
  const { t } = useTranslation();
  const [downloads, setDownloads] = useState<DownloadItem[]>([]);
  const [stats, setStats] = useState<QueueStats>({
    total: 0, pending: 0, downloading: 0, completed: 0, failed: 0, paused: false
  });

  const refreshData = useCallback(async () => {
    try {
      const items = await api.getDownloads();
      if (items && Array.isArray(items)) {
        setDownloads(items as DownloadItem[]);
      }
      
      const s = await api.getStats();
      if (s) setStats(s as QueueStats);
    } catch (e) {
      console.error("JS: Erro no RefreshData:", e);
    }
  }, []);

  useEffect(() => {
    refreshData();
    const unsubscribe = api.subscribe({
      onDownloads: setDownloads,
      onItem: (item: DownloadItem) => {
        setDownloads(current => {
          const index = current.findIndex(existing => existing.id === item.id);
          if (index < 0) return [...current, item];
          const next = [...current];
          next[index] = item;
          return next;
        });
      },
      onStats: setStats,
    });
    return () => {
      unsubscribe();
    };
  }, [refreshData]);

  const addDownloads = useCallback(async (urls: string[], quality: string) => {
    try {
      await api.addDownloads(urls, quality);
      await refreshData();
    } catch (e) {
      console.error("JS: Erro ao adicionar:", e);
    }
  }, [refreshData]);

  const addVideoDownloads = useCallback(async (requests: VideoDownloadRequest[]) => {
    try {
      await api.addVideoDownloads(requests);
      await refreshData();
    } catch (e) {
      console.error("JS: Erro ao adicionar vídeos:", e);
    }
  }, [refreshData]);

  const shouldDeleteFiles = async (hasFiles: boolean, confirmation: string) => {
    if (!hasFiles) return false;
    if (!api.capabilities.nativeFolders) return true;
    const config = await api.getConfig();
    if (config.file_deletion === 'delete') return true;
    if (config.file_deletion === 'keep') return false;
    return window.confirm(confirmation);
  };

  const removeDownload = async (item: DownloadItem) => {
    const deleteFile = await shouldDeleteFiles(['completed', 'skipped'].includes(item.status), t.deleteFileConfirm);
    await api.removeDownload(item.id, deleteFile);
    await refreshData();
  };

  const clearCompleted = async () => {
    const deleteFiles = await shouldDeleteFiles(downloads.some(item => ['completed', 'skipped'].includes(item.status)), t.deleteFilesConfirm);
    await api.clearCompleted(deleteFiles);
    await refreshData();
  };

  const clearAll = async () => {
    const deleteFiles = await shouldDeleteFiles(downloads.some(item => ['completed', 'skipped'].includes(item.status)), t.deleteFilesConfirm);
    await api.clearAll(deleteFiles);
    await refreshData();
  };

  return {
    downloads, stats, addDownloads, addVideoDownloads,
    cancelDownload: async (id: string) => { await api.cancelDownload(id); refreshData(); },
    removeDownload,
    retryDownload: async (id: string) => { await api.retryDownload(id); refreshData(); },
    retryFailed: async () => { await api.retryFailed(); refreshData(); },
    clearCompleted,
    cancelAll: async () => { await api.cancelAll(); refreshData(); },
    pauseQueue: async () => { await api.pauseQueue(); refreshData(); },
    resumeQueue: async () => { await api.resumeQueue(); refreshData(); },
    clearAll
  };
}

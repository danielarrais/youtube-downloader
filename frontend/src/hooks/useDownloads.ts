import { useState, useCallback, useEffect } from 'react';
import { api } from '../services/api';
import { DownloadItem, QueueStats, VideoDownloadRequest } from '../types';
import { useTranslation } from './useTranslation';

interface DeleteConfirmationOptions {
  title: string;
  message: string;
  includeDeleteFileOption: boolean;
}

interface DeleteConfirmationResult {
  confirmed: boolean;
  deleteFile: boolean;
}

export function useDownloads(confirmDelete: (options: DeleteConfirmationOptions) => Promise<DeleteConfirmationResult>) {
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

  const removeDownload = async (item: DownloadItem) => {
    const result = await confirmDelete({
      title: t.removeItemTitle,
      message: t.removeItemMessage,
      includeDeleteFileOption: ['completed', 'skipped'].includes(item.status),
    });
    if (!result.confirmed) return;
    await api.removeDownload(item.id, result.deleteFile);
    await refreshData();
  };

  const clearCompleted = async () => {
    const result = await confirmDelete({
      title: t.clearCompletedTitle,
      message: t.clearCompletedMessage,
      includeDeleteFileOption: downloads.some(item => ['completed', 'skipped'].includes(item.status)),
    });
    if (!result.confirmed) return;
    await api.clearCompleted(result.deleteFile);
    await refreshData();
  };

  const clearAll = async () => {
    const result = await confirmDelete({
      title: t.clearAllTitle,
      message: t.clearAllMessage,
      includeDeleteFileOption: downloads.some(item => ['completed', 'skipped'].includes(item.status)),
    });
    if (!result.confirmed) return;
    await api.clearAll(result.deleteFile);
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

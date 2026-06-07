import { useState, useCallback, useEffect } from 'react';
import { api } from '../services/api';
import { DownloadItem, QueueStats } from '../types';

export function useDownloads() {
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

  return {
    downloads, stats, addDownloads,
    cancelDownload: async (id: string) => { await api.cancelDownload(id); refreshData(); },
    retryDownload: async (id: string) => { await api.retryDownload(id); refreshData(); },
    retryFailed: async () => { await api.retryFailed(); refreshData(); },
    clearCompleted: async () => { await api.clearCompleted(); refreshData(); },
    cancelAll: async () => { await api.cancelAll(); refreshData(); },
    pauseQueue: async () => { await api.pauseQueue(); refreshData(); },
    resumeQueue: async () => { await api.resumeQueue(); refreshData(); },
    clearAll: async () => { await api.clearAll(); refreshData(); }
  };
}

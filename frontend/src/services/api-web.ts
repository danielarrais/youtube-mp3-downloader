import { AppAPI } from './types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { error?: string } | null;
    throw new Error(body?.error || `HTTP ${response.status}`);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

const post = <T>(path: string, body?: unknown) => request<T>(path, {
  method: 'POST',
  body: body === undefined ? undefined : JSON.stringify(body),
});

export const webApi: AppAPI = {
  capabilities: { nativeFolders: false },
  addDownloads: (urls, quality) => post('/api/downloads', { urls, quality }),
  getDownloads: () => request('/api/downloads'),
  getStats: () => request('/api/stats'),
  cancelDownload: (id) => post(`/api/downloads/${encodeURIComponent(id)}/cancel`),
  retryDownload: (id) => post(`/api/downloads/${encodeURIComponent(id)}/retry`),
  retryFailed: () => post('/api/queue/retry-failed'),
  getPlaylistInfo: (url) => request(`/api/playlist?url=${encodeURIComponent(url)}`),
  clearCompleted: () => post('/api/queue/clear-completed'),
  cancelAll: () => post('/api/queue/cancel-all'),
  pauseQueue: () => post('/api/queue/pause'),
  resumeQueue: () => post('/api/queue/resume'),
  clearAll: () => post('/api/queue/clear-all'),
  getConfig: () => request('/api/config'),
  selectFolder: async () => '',
  saveConfig: (config) => request('/api/config', {
    method: 'PUT',
    body: JSON.stringify(config),
  }),
  setLanguage: (language) => request('/api/language', {
    method: 'PUT',
    body: JSON.stringify({ language }),
  }),
  openDirectory: async () => {},
  openDownload: async (item) => {
    window.location.assign(`/api/downloads/${encodeURIComponent(item.id)}/file`);
  },
  subscribe: ({ onDownloads, onStats }) => {
    let stopped = false;
    const refresh = async () => {
      try {
        const [items, stats] = await Promise.all([
          webApi.getDownloads(),
          webApi.getStats(),
        ]);
        if (!stopped) {
          onDownloads(items);
          onStats(stats);
        }
      } catch (error) {
        console.error('Web polling failed:', error);
      }
    };
    const interval = window.setInterval(refresh, 1000);
    return () => {
      stopped = true;
      window.clearInterval(interval);
    };
  },
};

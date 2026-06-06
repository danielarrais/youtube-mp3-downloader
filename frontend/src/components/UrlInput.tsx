import { useState, useEffect } from 'react';
import { useTranslation } from '../hooks/useTranslation';
import { api } from '../services/api';
import { PlaylistInfo } from '../types';
import { PlaylistLoadState, PlaylistModal } from './PlaylistModal';
import { SettingsModal } from './SettingsModal';

interface UrlInputProps {
  onSubmit: (urls: string[], quality: string) => void;
}

const cleanYouTubeUrl = (value: string) => value.trim().split('&', 1)[0];

export function UrlInput({ onSubmit }: UrlInputProps) {
  const { t, language } = useTranslation();
  const [urls, setUrls] = useState('');
  const [quality, setQuality] = useState('192k');
  const [downloadDir, setDownloadDir] = useState('---');
  const [playlistStates, setPlaylistStates] = useState<PlaylistLoadState[] | null>(null);
  const [selectedVideoKeys, setSelectedVideoKeys] = useState<Set<string>>(new Set());
  const [directVideoUrls, setDirectVideoUrls] = useState<string[]>([]);
  const [settingsOpen, setSettingsOpen] = useState(false);

  // Carrega configuração inicial
  useEffect(() => {
    api.getConfig().then(config => {
      if (config) {
        setDownloadDir(config.download_dir || '---');
        setQuality(config.quality || '192k');
      }
    });
  }, []);

  const isPlaylistUrl = (value: string) => {
    try {
      return new URL(value).searchParams.has('list');
    } catch {
      return false;
    }
  };

  const loadPlaylist = async (url: string) => {
    setPlaylistStates(current => current?.map(item =>
      item.url === url ? { url, loading: true } : item
    ) || null);

    try {
      const playlist = await api.getPlaylistInfo(url) as PlaylistInfo;
      setPlaylistStates(current => current?.map(item =>
        item.url === url ? { url, playlist, loading: false } : item
      ) || null);
      setSelectedVideoKeys(current => {
        const next = new Set(current);
        playlist.videos
          .filter(video => video.available)
          .forEach(video => next.add(`${playlist.id}:${video.index}`));
        return next;
      });
    } catch (error) {
      setPlaylistStates(current => current?.map(item =>
        item.url === url
          ? { url, loading: false, error: error instanceof Error ? error.message : String(error) }
          : item
      ) || null);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const urlList = urls.split('\n').map(cleanYouTubeUrl).filter(url => url.length > 0);
    if (urlList.length === 0) return;

    const playlistUrls = [...new Set(urlList.filter(isPlaylistUrl))];
    const videoUrls = urlList.filter(url => !isPlaylistUrl(url));
    if (playlistUrls.length === 0) {
      onSubmit(videoUrls, quality);
      setUrls('');
      return;
    }

    setDirectVideoUrls(videoUrls);
    setSelectedVideoKeys(new Set());
    setPlaylistStates(playlistUrls.map(url => ({ url, loading: true })));
    playlistUrls.forEach(url => {
      api.getPlaylistInfo(url)
        .then(playlist => {
          setPlaylistStates(current => current?.map(item =>
            item.url === url ? { url, playlist: playlist as PlaylistInfo, loading: false } : item
          ) || null);
          setSelectedVideoKeys(current => {
            const next = new Set(current);
            (playlist as PlaylistInfo).videos
              .filter(video => video.available)
              .forEach(video => next.add(`${playlist.id}:${video.index}`));
            return next;
          });
        })
        .catch(error => {
          setPlaylistStates(current => current?.map(item =>
            item.url === url
              ? { url, loading: false, error: error instanceof Error ? error.message : String(error) }
              : item
          ) || null);
        });
    });
  };

  const togglePlaylistVideo = (key: string) => {
    setSelectedVideoKeys(current => {
      const next = new Set(current);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });
  };

  const togglePlaylist = (playlist: PlaylistInfo) => {
    const availableKeys = playlist.videos
      .filter(video => video.available)
      .map(video => `${playlist.id}:${video.index}`);
    setSelectedVideoKeys(current => {
      const next = new Set(current);
      const allSelected = availableKeys.every(key => next.has(key));
      availableKeys.forEach(key => allSelected ? next.delete(key) : next.add(key));
      return next;
    });
  };

  const confirmPlaylistSelection = () => {
    const selectedUrls = playlistStates?.flatMap(item =>
      item.playlist?.videos
        .filter(video => video.available && selectedVideoKeys.has(`${item.playlist!.id}:${video.index}`))
        .map(video => video.url) || []
    ) || [];
    onSubmit([...directVideoUrls, ...selectedUrls], quality);
    setUrls('');
    setPlaylistStates(null);
    setSelectedVideoKeys(new Set());
    setDirectVideoUrls([]);
  };

  const handleSelectFolder = async () => {
    const newPath = await api.selectFolder();
    return newPath || undefined;
  };

  const handleSaveSettings = async (newDownloadDir: string, newQuality: string) => {
    const config = await api.saveConfig({
      download_dir: newDownloadDir,
      quality: newQuality,
      language,
    });
    setDownloadDir(config?.download_dir || newDownloadDir);
    setQuality(config?.quality || newQuality);
  };

  return (
    <div className="space-y-4">
      <form onSubmit={handleSubmit} className="bg-gray-800 rounded-lg p-4 space-y-4 shadow-xl border border-gray-700">
        <div className="flex justify-between items-center mb-2">
          <label className="text-sm font-medium text-gray-300">
            {t.urlsLabel}
          </label>
          <button
            type="button"
            onClick={() => downloadDir !== '---' && api.openDirectory(downloadDir)}
            className="text-[10px] text-gray-400 hover:text-gray-200 font-mono bg-black/30 px-2 py-1 rounded max-w-[350px] truncate border border-gray-700 transition-colors"
            title={downloadDir}
          >
            📁 {downloadDir}
          </button>
        </div>
        
        <textarea
          value={urls}
          onChange={(e) => setUrls(e.target.value)}
          placeholder={t.urlsPlaceholder}
          className="w-full h-28 bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white placeholder-gray-600 focus:outline-none focus:border-red-500 transition-all text-sm"
        />

        <div className="flex items-center gap-4 flex-wrap">
          <button
            type="submit"
            className="bg-red-600 hover:bg-red-700 text-white font-bold px-8 py-2 rounded-lg transition-all text-sm shadow-lg shadow-red-900/20 active:scale-95"
          >
            {t.addToQueue}
          </button>

          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="bg-gray-700 hover:bg-gray-600 text-gray-200 px-4 py-2 rounded-lg text-sm transition-all border border-gray-600 active:scale-95"
          >
            {t.settings}
          </button>
        </div>
      </form>

      {playlistStates && (
        <PlaylistModal
          playlists={playlistStates}
          selectedVideoKeys={selectedVideoKeys}
          directVideoCount={directVideoUrls.length}
          onToggleVideo={togglePlaylistVideo}
          onTogglePlaylist={togglePlaylist}
          onRetry={loadPlaylist}
          onClose={() => setPlaylistStates(null)}
          onConfirm={confirmPlaylistSelection}
        />
      )}

      {settingsOpen && (
        <SettingsModal
          downloadDir={downloadDir}
          quality={quality}
          onChooseFolder={handleSelectFolder}
          onClose={() => setSettingsOpen(false)}
          onSave={handleSaveSettings}
        />
      )}
    </div>
  );
}

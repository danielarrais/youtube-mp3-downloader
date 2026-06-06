// Importa as funções geradas pelo Wails
import * as WailsApp from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

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
    return await WailsApp.RetryFailed();
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
    return await WailsApp.OpenDirectory(path);
  },

  // Funções de Configuração e Pasta
  getConfig: async () => {
    return await WailsApp.GetConfig();
  },

  selectFolder: async () => {
    return await WailsApp.SelectFolder();
  },

  saveConfig: async (config: main.Config) => {
    return await WailsApp.SaveConfig(config);
  },

  setLanguage: async (language: string) => {
    return await WailsApp.SetLanguage(language);
  },
};

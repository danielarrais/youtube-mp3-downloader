export type Language = 'pt-BR' | 'en-US';

export interface Translations {
  // Header
  title: string;
  total: string;
  downloading: string;
  pending: string;
  completed: string;
  failed: string;

  // UrlInput
  urlsLabel: string;
  urlsHint: string;
  urlsPlaceholder: string;
  quality: string;
  addToQueue: string;
  settings: string;
  downloadFolder: string;
  chooseFolder: string;
  save: string;
  settingsSaveError: string;
  selectPlaylistVideos: string;
  selectAll: string;
  clearSelection: string;
  selectedVideos: string;
  unavailable: string;
  loadingPlaylist: string;
  playlistLoadError: string;
  retry: string;
  close: string;
  addSelected: string;

  // DownloadQueue
  emptyQueue: string;
  cancelAll: string;
  pauseQueue: string;
  resumeQueue: string;
  clearCompleted: string;
  clearAll: string;

  // DownloadItem
  cancel: string;
  openFolder: string;
  downloadFile: string;

  // StatusBadge
  status: {
    pending: string;
    fetching_info: string;
    downloading: string;
    converting: string;
    completed: string;
    failed: string;
    cancelled: string;
    skipped: string;
  };
}

export const translations: Record<Language, Translations> = {
  'pt-BR': {
    title: 'Youtube MP3 Dowloader',
    total: 'Total',
    downloading: 'Baixando',
    pending: 'Pendente',
    completed: 'Concluído',
    failed: 'Falhas',
    urlsLabel: 'URLs do YouTube (uma por linha)',
    urlsHint: 'Aceita vídeos individuais ou playlists',
    urlsPlaceholder: 'https://www.youtube.com/watch?v=... ou playlist?list=...',
    quality: 'Qualidade',
    addToQueue: 'Adicionar à Fila',
    settings: 'Configurações',
    downloadFolder: 'Pasta de download',
    chooseFolder: 'Escolher pasta',
    save: 'Salvar',
    settingsSaveError: 'Não foi possível salvar as configurações.',
    selectPlaylistVideos: 'Selecione os vídeos das playlists',
    selectAll: 'Selecionar todos',
    clearSelection: 'Desmarcar todos',
    selectedVideos: 'vídeos selecionados',
    unavailable: 'Indisponível',
    loadingPlaylist: 'Carregando playlist...',
    playlistLoadError: 'Não foi possível carregar esta playlist.',
    retry: 'Tentar novamente',
    close: 'Cancelar',
    addSelected: 'Adicionar selecionados',
    emptyQueue: 'Nenhum download na fila. Adicione URLs acima para começar.',
    cancelAll: 'Cancelar todos',
    pauseQueue: 'Pausar',
    resumeQueue: 'Continuar',
    clearCompleted: 'Limpar concluídos',
    clearAll: 'Limpar tudo',
    cancel: 'Cancelar',
    openFolder: 'Abrir na pasta',
    downloadFile: 'Baixar MP3',
    status: {
      pending: 'Pendente',
      fetching_info: 'Obtendo info',
      downloading: 'Baixando',
      converting: 'Convertendo',
      completed: 'Concluído',
      failed: 'Falhou',
      cancelled: 'Cancelado',
      skipped: 'Já existe',
    },
  },
  'en-US': {
    title: 'Youtube MP3 Dowloader',
    total: 'Total',
    downloading: 'Downloading',
    pending: 'Pending',
    completed: 'Completed',
    failed: 'Failed',
    urlsLabel: 'YouTube URLs (one per line)',
    urlsHint: 'Supports individual videos or playlists',
    urlsPlaceholder: 'https://www.youtube.com/watch?v=... or playlist?list=...',
    quality: 'Quality',
    addToQueue: 'Add to Queue',
    settings: 'Settings',
    downloadFolder: 'Download folder',
    chooseFolder: 'Choose folder',
    save: 'Save',
    settingsSaveError: 'Could not save the settings.',
    selectPlaylistVideos: 'Select playlist videos',
    selectAll: 'Select all',
    clearSelection: 'Clear selection',
    selectedVideos: 'videos selected',
    unavailable: 'Unavailable',
    loadingPlaylist: 'Loading playlist...',
    playlistLoadError: 'Could not load this playlist.',
    retry: 'Retry',
    close: 'Cancel',
    addSelected: 'Add selected',
    emptyQueue: 'No downloads in queue. Add URLs above to start.',
    cancelAll: 'Cancel all',
    pauseQueue: 'Pause',
    resumeQueue: 'Continue',
    clearCompleted: 'Clear completed',
    clearAll: 'Clear all',
    cancel: 'Cancel',
    openFolder: 'Open folder',
    downloadFile: 'Download MP3',
    status: {
      pending: 'Pending',
      fetching_info: 'Fetching info',
      downloading: 'Downloading',
      converting: 'Converting',
      completed: 'Completed',
      failed: 'Failed',
      cancelled: 'Cancelled',
      skipped: 'Already exists',
    },
  },
};

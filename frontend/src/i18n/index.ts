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
  videoFormat: string;
  videoQuality: string;
  askAudioQuality: string;
  askVideoQuality: string;
  fileDeletion: string;
  fileDeletionDelete: string;
  fileDeletionAsk: string;
  fileDeletionKeep: string;
  language: string;
  languagePortuguese: string;
  languageEnglish: string;
  theme: string;
  themeDark: string;
  themeLight: string;
  parallelDownloads: string;
  parallelDownloadsHelp: string;
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
  filteredDownloadsEmpty: string;
  cancelAll: string;
  pauseQueue: string;
  resumeQueue: string;
  clearCompleted: string;
  clearAll: string;

  // DownloadItem
  cancel: string;
  openFolder: string;
  downloadFile: string;
  downloadVideoFile: string;
  copyLink: string;
  linkCopied: string;
  deleteItem: string;
  removeItemTitle: string;
  removeItemMessage: string;
  clearCompletedTitle: string;
  clearCompletedMessage: string;
  clearAllTitle: string;
  clearAllMessage: string;
  removeFromDisk: string;
  confirmDelete: string;
  deleteFileConfirm: string;
  deleteFilesConfirm: string;

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
    title: 'Youtube Dowloader',
    total: 'Total',
    downloading: 'Baixando',
    pending: 'Pendente',
    completed: 'Concluído',
    failed: 'Falhas',
    urlsLabel: 'Links do YouTube, X, Instagram ou TikTok',
    urlsHint: 'Aceita vídeos individuais ou playlists do YouTube',
    urlsPlaceholder: 'https://www.youtube.com/watch?v=..., https://x.com/..., https://www.instagram.com/reel/..., https://www.tiktok.com/...',
    quality: 'Bitrate alvo do áudio',
    videoFormat: 'Formato de vídeo padrão',
    videoQuality: 'Qualidade de vídeo alvo',
    askAudioQuality: 'Sempre perguntar a qualidade do áudio',
    askVideoQuality: 'Sempre perguntar a qualidade do vídeo',
    fileDeletion: 'Ao remover downloads',
    fileDeletionDelete: 'Apagar do disco',
    fileDeletionAsk: 'Perguntar',
    fileDeletionKeep: 'Manter no disco',
    language: 'Idioma',
    languagePortuguese: 'Português',
    languageEnglish: 'Inglês',
    theme: 'Tema',
    themeDark: 'Escuro',
    themeLight: 'Claro',
    parallelDownloads: 'Downloads paralelos',
    parallelDownloadsHelp: 'Quantos itens podem baixar ao mesmo tempo neste computador.',
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
    filteredDownloadsEmpty: 'Nenhum item com este status.',
    cancelAll: 'Cancelar todos',
    pauseQueue: 'Pausar',
    resumeQueue: 'Continuar',
    clearCompleted: 'Limpar concluídos',
    clearAll: 'Limpar tudo',
    cancel: 'Cancelar',
    openFolder: 'Abrir na pasta',
    downloadFile: 'Baixar MP3',
    downloadVideoFile: 'Baixar vídeo',
    copyLink: 'Copiar link',
    linkCopied: 'Link copiado',
    deleteItem: 'Excluir item',
    removeItemTitle: 'Remover download',
    removeItemMessage: 'Deseja remover este download da lista?',
    clearCompletedTitle: 'Remover concluídos',
    clearCompletedMessage: 'Deseja remover todos os downloads concluídos da lista?',
    clearAllTitle: 'Remover todos os downloads',
    clearAllMessage: 'Deseja remover todos os downloads da lista?',
    removeFromDisk: 'Também remover o arquivo do disco',
    confirmDelete: 'Remover',
    deleteFileConfirm: 'Também apagar o arquivo deste download do disco?',
    deleteFilesConfirm: 'Também apagar os arquivos concluídos do disco?',
    status: {
      pending: 'Pendente',
      fetching_info: 'Obtendo info',
      downloading: 'Baixando',
      converting: 'Convertendo',
      completed: 'Concluído',
      failed: 'Falhou',
      cancelled: 'Cancelado',
      skipped: 'Concluído',
    },
  },
  'en-US': {
    title: 'Youtube Dowloader',
    total: 'Total',
    downloading: 'Downloading',
    pending: 'Pending',
    completed: 'Completed',
    failed: 'Failed',
    urlsLabel: 'YouTube, X, Instagram or TikTok links',
    urlsHint: 'Supports individual videos or YouTube playlists',
    urlsPlaceholder: 'https://www.youtube.com/watch?v=..., https://x.com/..., https://www.instagram.com/reel/..., https://www.tiktok.com/...',
    quality: 'Target audio bitrate',
    videoFormat: 'Default video format',
    videoQuality: 'Target video quality',
    askAudioQuality: 'Always ask for audio quality',
    askVideoQuality: 'Always ask for video quality',
    fileDeletion: 'When removing downloads',
    fileDeletionDelete: 'Delete from disk',
    fileDeletionAsk: 'Ask',
    fileDeletionKeep: 'Keep on disk',
    language: 'Language',
    languagePortuguese: 'Portuguese',
    languageEnglish: 'English',
    theme: 'Theme',
    themeDark: 'Dark',
    themeLight: 'Light',
    parallelDownloads: 'Parallel downloads',
    parallelDownloadsHelp: 'How many items can download at the same time on this computer.',
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
    filteredDownloadsEmpty: 'No items with this status.',
    cancelAll: 'Cancel all',
    pauseQueue: 'Pause',
    resumeQueue: 'Continue',
    clearCompleted: 'Clear completed',
    clearAll: 'Clear all',
    cancel: 'Cancel',
    openFolder: 'Open folder',
    downloadFile: 'Download MP3',
    downloadVideoFile: 'Download video',
    copyLink: 'Copy link',
    linkCopied: 'Link copied',
    deleteItem: 'Delete item',
    removeItemTitle: 'Remove download',
    removeItemMessage: 'Do you want to remove this download from the list?',
    clearCompletedTitle: 'Remove completed',
    clearCompletedMessage: 'Do you want to remove all completed downloads from the list?',
    clearAllTitle: 'Remove all downloads',
    clearAllMessage: 'Do you want to remove all downloads from the list?',
    removeFromDisk: 'Also remove the file from disk',
    confirmDelete: 'Remove',
    deleteFileConfirm: 'Also delete this download file from disk?',
    deleteFilesConfirm: 'Also delete completed files from disk?',
    status: {
      pending: 'Pending',
      fetching_info: 'Fetching info',
      downloading: 'Downloading',
      converting: 'Converting',
      completed: 'Completed',
      failed: 'Failed',
      cancelled: 'Cancelled',
      skipped: 'Completed',
    },
  },
};

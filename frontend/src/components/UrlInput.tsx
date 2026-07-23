import { useState, useEffect } from 'react';
import { useTranslation } from '../hooks/useTranslation';
import { useTheme } from '../hooks/useTheme';
import { api } from '../services/api';
import { Config, PlaylistInfo, VideoDownloadRequest, VideoFormat, VideoInfo } from '../types';
import { PlaylistLoadState, PlaylistModal } from './PlaylistModal';
import { SettingsModal } from './SettingsModal';

interface UrlInputProps {
  onSubmitAudio: (urls: string[], quality: string) => void;
  onSubmitVideo: (requests: VideoDownloadRequest[]) => void;
  settingsOpen: boolean;
  onOpenSettings: () => void;
  onCloseSettings: () => void;
}

const cleanYouTubeUrl = (value: string) => value.trim().split('&', 1)[0];

type VideoContainer = Config['video_container'];
type VideoQuality = Config['video_quality'];

const videoQualityValue = (quality: string) => Number(quality.match(/\d+/)?.[0]) || 0;

export function preferredVideoFormat(
  formats: VideoFormat[],
  targetContainer: VideoContainer,
  targetQuality: VideoQuality,
) {
  const matchingContainer = formats.filter(format => format.container === targetContainer);
  const candidates = matchingContainer.length > 0 ? matchingContainer : formats;
  const target = videoQualityValue(targetQuality);
  const atOrBelowTarget = candidates.filter(format => videoQualityValue(format.resolution) <= target);
  const pool = atOrBelowTarget.length > 0
    ? atOrBelowTarget
    : candidates.filter(format => videoQualityValue(format.resolution) > target);

  return [...pool].sort((left, right) => {
    const leftQuality = videoQualityValue(left.resolution);
    const rightQuality = videoQualityValue(right.resolution);
    if (leftQuality !== rightQuality) {
      return atOrBelowTarget.length > 0 ? rightQuality - leftQuality : leftQuality - rightQuality;
    }
    if ((left.fps || 0) !== (right.fps || 0)) return (right.fps || 0) - (left.fps || 0);
    return (right.size || 0) - (left.size || 0);
  })[0];
}

interface VideoSelection {
  url: string;
  info?: VideoInfo;
  selectedItag?: number;
  error?: string;
}

export function UrlInput({ onSubmitAudio, onSubmitVideo, settingsOpen, onOpenSettings, onCloseSettings }: UrlInputProps) {
  const { t, language, setLanguage } = useTranslation();
  const { theme, setTheme } = useTheme();
  const [urls, setUrls] = useState('');
  const [quality, setQuality] = useState('192k');
  const [videoContainer, setVideoContainer] = useState<VideoContainer>('mp4');
  const [videoQuality, setVideoQuality] = useState<VideoQuality>('1080p');
  const [fileDeletion, setFileDeletion] = useState<Config['file_deletion']>('ask');
  const [mediaType, setMediaType] = useState<'audio' | 'video'>('audio');
  const [downloadDir, setDownloadDir] = useState('---');
  const [playlistStates, setPlaylistStates] = useState<PlaylistLoadState[] | null>(null);
  const [selectedVideoKeys, setSelectedVideoKeys] = useState<Set<string>>(new Set());
  const [directVideoUrls, setDirectVideoUrls] = useState<string[]>([]);
  const [themeBeforeSettings, setThemeBeforeSettings] = useState<Config['theme'] | null>(null);
  const [videoSelections, setVideoSelections] = useState<VideoSelection[] | null>(null);

  useEffect(() => {
    api.getConfig().then(config => {
      if (config) {
        setDownloadDir(config.download_dir || '---');
        setQuality(config.quality || '192k');
        setVideoContainer(config.video_container || 'mp4');
        setVideoQuality(config.video_quality || '1080p');
        setFileDeletion(config.file_deletion || 'ask');
        setTheme(config.theme || 'dark');
      }
    });
  }, [setTheme]);

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
      submitURLs(videoUrls);
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

  const submitURLs = (videoUrls: string[]) => {
    if (mediaType === 'audio') {
      onSubmitAudio(videoUrls, quality);
      return;
    }
    setVideoSelections(videoUrls.map(url => ({ url })));
    videoUrls.forEach(url => {
      api.getVideoFormats(url)
        .then(info => {
          setVideoSelections(current => current?.map(selection => selection.url === url
            ? { url, info, selectedItag: preferredVideoFormat(info.formats, videoContainer, videoQuality)?.video_itag }
            : selection) || null);
        })
        .catch(error => {
          setVideoSelections(current => current?.map(selection => selection.url === url
            ? { url, error: error instanceof Error ? error.message : String(error) }
            : selection) || null);
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
    submitURLs([...directVideoUrls, ...selectedUrls]);
    setUrls('');
    setPlaylistStates(null);
    setSelectedVideoKeys(new Set());
    setDirectVideoUrls([]);
  };

  const handleSelectFolder = async () => {
    const newPath = await api.selectFolder();
    return newPath || undefined;
  };

  const handleSaveSettings = async (
    newDownloadDir: string,
    newQuality: string,
    newVideoContainer: VideoContainer,
    newVideoQuality: VideoQuality,
    newFileDeletion: Config['file_deletion'],
    newLanguage: 'pt-BR' | 'en-US',
    newTheme: Config['theme'],
  ) => {
    const config = await api.saveConfig({
      download_dir: newDownloadDir,
      quality: newQuality,
      video_container: newVideoContainer,
      video_quality: newVideoQuality,
      file_deletion: newFileDeletion,
      language: newLanguage,
      theme: newTheme,
    });
    setDownloadDir(config?.download_dir || newDownloadDir);
    setQuality(config?.quality || newQuality);
    setVideoContainer(config?.video_container || newVideoContainer);
    setVideoQuality(config?.video_quality || newVideoQuality);
    setFileDeletion(config?.file_deletion || newFileDeletion);
    setLanguage((config?.language || newLanguage) as 'pt-BR' | 'en-US');
    setTheme(config?.theme || newTheme);
    setThemeBeforeSettings(null);
    onCloseSettings();
  };

  const openSettings = () => {
    setThemeBeforeSettings(theme);
    onOpenSettings();
  };

  const closeSettings = () => {
    if (themeBeforeSettings) {
      setTheme(themeBeforeSettings);
    }
    setThemeBeforeSettings(null);
    onCloseSettings();
  };

  return (
    <div className="space-y-4">
      <form onSubmit={handleSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm transition-colors dark:border-gray-700 dark:bg-gray-800 sm:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <label className="text-sm font-medium text-slate-700 dark:text-gray-300">
              {t.urlsLabel}
            </label>
          </div>
          {api.capabilities.nativeFolders && (
            <button
              type="button"
              onClick={() => api.capabilities.nativeFolders
                && downloadDir !== '---'
                && api.openDirectory(downloadDir)}
              className="max-w-full rounded-lg border border-slate-300 bg-slate-100 px-3 py-2 text-left font-mono text-[10px] text-slate-600 transition-colors enabled:hover:border-slate-400 enabled:hover:text-slate-900 disabled:cursor-default dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-400 dark:enabled:hover:border-gray-600 dark:enabled:hover:text-gray-200 sm:max-w-[350px]"
              title={downloadDir}
            >
              <span className="block truncate">📁 {downloadDir}</span>
            </button>
          )}
        </div>

        <textarea
          value={urls}
          onChange={(e) => setUrls(e.target.value)}
          placeholder={t.urlsPlaceholder}
          className="h-32 w-full rounded-lg border border-slate-300 bg-slate-50 px-4 py-3 text-sm text-slate-900 placeholder-slate-400 transition-all focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:placeholder-gray-600 sm:h-28"
        />

        <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
          <div className="flex w-full overflow-hidden rounded-lg border border-slate-300 bg-slate-100 dark:border-gray-700 dark:bg-gray-900 sm:w-auto" role="group" aria-label="Tipo de download">
            <button
              type="button"
              onClick={() => setMediaType('audio')}
              aria-pressed={mediaType === 'audio'}
              className={`flex flex-1 items-center justify-center gap-2 px-3 py-2 text-sm transition-colors sm:flex-none ${mediaType === 'audio' ? 'bg-red-600 text-white' : 'text-slate-700 hover:bg-slate-200 dark:text-gray-300 dark:hover:bg-gray-800'}`}
            >
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M9 18V5l10-2v13M9 18a3 3 0 11-6 0 3 3 0 016 0zm10-2a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              MP3
            </button>
            <button
              type="button"
              onClick={() => setMediaType('video')}
              aria-pressed={mediaType === 'video'}
              className={`flex flex-1 items-center justify-center gap-2 border-l border-slate-300 px-3 py-2 text-sm transition-colors dark:border-gray-700 sm:flex-none ${mediaType === 'video' ? 'bg-red-600 text-white' : 'text-slate-700 hover:bg-slate-200 dark:text-gray-300 dark:hover:bg-gray-800'}`}
            >
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="m10 8 5 4-5 4V8z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M4 6h16v12H4z" />
              </svg>
              Vídeo
            </button>
          </div>

          <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row">
            <button
              type="submit"
              className="w-full rounded-lg bg-red-600 px-8 py-2 text-sm font-bold text-white shadow-lg shadow-red-900/20 transition-all hover:bg-red-700 active:scale-95 sm:w-auto"
            >
              {t.addToQueue}
            </button>
          </div>
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

      {videoSelections && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4">
          <div className="max-h-[90vh] w-full max-w-2xl space-y-4 overflow-y-auto rounded-xl border border-slate-200 bg-white p-4 shadow-2xl dark:border-gray-700 dark:bg-gray-800 sm:p-5">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Escolha o formato de cada vídeo</h2>
            {videoSelections.map(selection => (
              <div key={selection.url} className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-slate-50 p-3 dark:border-gray-700 dark:bg-gray-900/50 sm:flex-row">
                {selection.info?.thumbnail_url ? (
                  <img src={selection.info.thumbnail_url} alt="" className="h-40 w-full rounded-lg object-cover sm:h-16 sm:w-28 sm:flex-none" />
                ) : (
                  <div className="h-40 w-full rounded-lg bg-slate-200 dark:bg-gray-700 sm:h-16 sm:w-28 sm:flex-none" />
                )}
                <div className="min-w-0 flex-1 space-y-2">
                  {selection.info?.title ? (
                    <>
                      <p className="break-words text-sm text-slate-800 dark:text-gray-200 sm:truncate">{selection.info.title}</p>
                      <a
                        href={selection.url}
                        target="_blank"
                        rel="noreferrer"
                        className="block break-all text-sm text-blue-600 underline-offset-2 hover:underline dark:text-blue-400 sm:truncate"
                      >
                        {selection.url}
                      </a>
                    </>
                  ) : (
                    <a
                      href={selection.url}
                      target="_blank"
                      rel="noreferrer"
                      className="block break-all text-sm text-blue-600 underline-offset-2 hover:underline dark:text-blue-400 sm:truncate"
                    >
                      {selection.url}
                    </a>
                  )}
                  {selection.error && <p className="text-sm text-red-600 dark:text-red-400">{selection.error}</p>}
                  {!selection.info && !selection.error && <p className="text-sm text-slate-500 dark:text-gray-400">Carregando formatos...</p>}
                  {selection.info && selection.info.formats.length === 0 && <p className="text-sm text-red-600 dark:text-red-400">Nenhum formato de vídeo disponível.</p>}
                  {selection.info && selection.info.formats.length > 0 && (
                    <select
                      value={selection.selectedItag || ''}
                      onChange={(event) => setVideoSelections(current => current?.map(value => value.url === selection.url
                        ? { ...value, selectedItag: Number(event.target.value) }
                        : value) || null)}
                      className="app-select w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
                    >
                      {selection.info.formats.map(format => (
                        <option key={format.video_itag} value={format.video_itag}>{format.label}</option>
                      ))}
                    </select>
                  )}
                </div>
              </div>
            ))}
            <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
              <button type="button" onClick={() => setVideoSelections(null)} className="w-full rounded-lg border border-slate-300 bg-slate-100 px-4 py-2 text-sm text-slate-800 hover:bg-slate-200 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 sm:w-auto">Cancelar</button>
              <button
                type="button"
                disabled={videoSelections.some(selection => !selection.info || !selection.selectedItag)}
                onClick={() => {
                  const requests = videoSelections.flatMap(selection => {
                    const format = selection.info?.formats.find(value => value.video_itag === selection.selectedItag);
                    return format ? [{ url: selection.url, format }] : [];
                  });
                  onSubmitVideo(requests);
                  setVideoSelections(null);
                }}
                className="w-full rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white enabled:hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
              >
                Adicionar vídeos
              </button>
            </div>
          </div>
        </div>
      )}

      {settingsOpen && (
        <SettingsModal
          downloadDir={downloadDir}
          quality={quality}
          videoContainer={videoContainer}
          videoQuality={videoQuality}
          fileDeletion={fileDeletion}
          language={language as 'pt-BR' | 'en-US'}
          theme={theme}
          onThemePreview={setTheme}
          onChooseFolder={handleSelectFolder}
          onClose={closeSettings}
          onSave={handleSaveSettings}
          canChooseFolder={api.capabilities.nativeFolders}
        />
      )}
    </div>
  );
}

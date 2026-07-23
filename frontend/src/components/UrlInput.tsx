import { useState, useEffect } from 'react';
import { useTranslation } from '../hooks/useTranslation';
import { api } from '../services/api';
import { Config, PlaylistInfo, VideoDownloadRequest, VideoFormat, VideoInfo } from '../types';
import { PlaylistLoadState, PlaylistModal } from './PlaylistModal';
import { SettingsModal } from './SettingsModal';

interface UrlInputProps {
  onSubmitAudio: (urls: string[], quality: string) => void;
  onSubmitVideo: (requests: VideoDownloadRequest[]) => void;
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

export function UrlInput({ onSubmitAudio, onSubmitVideo }: UrlInputProps) {
  const { t, language } = useTranslation();
  const [urls, setUrls] = useState('');
  const [quality, setQuality] = useState('192k');
  const [videoContainer, setVideoContainer] = useState<VideoContainer>('mp4');
  const [videoQuality, setVideoQuality] = useState<VideoQuality>('1080p');
  const [mediaType, setMediaType] = useState<'audio' | 'video'>('audio');
  const [downloadDir, setDownloadDir] = useState('---');
  const [playlistStates, setPlaylistStates] = useState<PlaylistLoadState[] | null>(null);
  const [selectedVideoKeys, setSelectedVideoKeys] = useState<Set<string>>(new Set());
  const [directVideoUrls, setDirectVideoUrls] = useState<string[]>([]);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [videoSelections, setVideoSelections] = useState<VideoSelection[] | null>(null);

  // Carrega configuração inicial
  useEffect(() => {
    api.getConfig().then(config => {
      if (config) {
        setDownloadDir(config.download_dir || '---');
        setQuality(config.quality || '192k');
        setVideoContainer(config.video_container || 'mp4');
        setVideoQuality(config.video_quality || '1080p');
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
  ) => {
    const config = await api.saveConfig({
      download_dir: newDownloadDir,
      quality: newQuality,
      video_container: newVideoContainer,
      video_quality: newVideoQuality,
      language,
    });
    setDownloadDir(config?.download_dir || newDownloadDir);
    setQuality(config?.quality || newQuality);
    setVideoContainer(config?.video_container || newVideoContainer);
    setVideoQuality(config?.video_quality || newVideoQuality);
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
            onClick={() => api.capabilities.nativeFolders
              && downloadDir !== '---'
              && api.openDirectory(downloadDir)}
            disabled={!api.capabilities.nativeFolders}
            className="text-[10px] text-gray-400 enabled:hover:text-gray-200 font-mono bg-black/30 px-2 py-1 rounded max-w-[350px] truncate border border-gray-700 transition-colors disabled:cursor-default"
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
          <select
            value={mediaType}
            onChange={(event) => setMediaType(event.target.value as 'audio' | 'video')}
            className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-red-500"
          >
            <option value="audio">MP3</option>
            <option value="video">Vídeo</option>
          </select>
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

      {videoSelections && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="w-full max-w-2xl max-h-[80vh] overflow-y-auto rounded-lg border border-gray-700 bg-gray-800 p-5 space-y-4 shadow-2xl">
            <h2 className="text-lg font-semibold text-white">Escolha o formato de cada vídeo</h2>
            {videoSelections.map(selection => (
              <div key={selection.url} className="flex gap-3 rounded border border-gray-700 bg-gray-900/50 p-3">
                {selection.info?.thumbnail_url ? (
                  <img src={selection.info.thumbnail_url} alt="" className="h-16 w-28 flex-none rounded object-cover" />
                ) : (
                  <div className="h-16 w-28 flex-none rounded bg-gray-700" />
                )}
                <div className="min-w-0 flex-1 space-y-2">
                  <p className="truncate text-sm text-gray-200">{selection.info?.title || selection.url}</p>
                  {selection.error && <p className="text-sm text-red-400">{selection.error}</p>}
                  {!selection.info && !selection.error && <p className="text-sm text-gray-400">Carregando formatos...</p>}
                  {selection.info && selection.info.formats.length === 0 && <p className="text-sm text-red-400">Nenhum formato de vídeo disponível.</p>}
                  {selection.info && selection.info.formats.length > 0 && (
                    <select
                      value={selection.selectedItag || ''}
                      onChange={(event) => setVideoSelections(current => current?.map(value => value.url === selection.url
                        ? { ...value, selectedItag: Number(event.target.value) }
                        : value) || null)}
                      className="w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white"
                    >
                      {selection.info.formats.map(format => (
                        <option key={format.video_itag} value={format.video_itag}>{format.label}</option>
                      ))}
                    </select>
                  )}
                </div>
              </div>
            ))}
            <div className="flex justify-end gap-3">
              <button type="button" onClick={() => setVideoSelections(null)} className="rounded px-4 py-2 text-sm text-gray-300 hover:text-white">Cancelar</button>
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
                className="rounded bg-red-600 px-4 py-2 text-sm font-bold text-white enabled:hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
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
          onChooseFolder={handleSelectFolder}
          onClose={() => setSettingsOpen(false)}
          onSave={handleSaveSettings}
          canChooseFolder={api.capabilities.nativeFolders}
        />
      )}
    </div>
  );
}

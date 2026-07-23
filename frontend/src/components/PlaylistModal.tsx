import { PlaylistInfo } from '../types';
import { useTranslation } from '../hooks/useTranslation';

export interface PlaylistLoadState {
  url: string;
  playlist?: PlaylistInfo;
  error?: string;
  loading: boolean;
}

interface PlaylistModalProps {
  playlists: PlaylistLoadState[];
  selectedVideoKeys: Set<string>;
  directVideoCount: number;
  onToggleVideo: (key: string) => void;
  onTogglePlaylist: (playlist: PlaylistInfo) => void;
  onRetry: (url: string) => void;
  onClose: () => void;
  onConfirm: () => void;
}

function formatDuration(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts = hours > 0 ? [hours, minutes, seconds] : [minutes, seconds];
  return parts.map((part, index) => index === 0 ? String(part) : String(part).padStart(2, '0')).join(':');
}

export function PlaylistModal({
  playlists,
  selectedVideoKeys,
  directVideoCount,
  onToggleVideo,
  onTogglePlaylist,
  onRetry,
  onClose,
  onConfirm,
}: PlaylistModalProps) {
  const { t } = useTranslation();
  const hasPendingPlaylist = playlists.some(item => item.loading || item.error);
  const canConfirm = !hasPendingPlaylist && selectedVideoKeys.size + directVideoCount > 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 sm:p-6">
      <div className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-900">
        <div className="flex flex-col gap-2 border-b border-slate-200 px-4 py-4 dark:border-gray-700 sm:flex-row sm:items-center sm:justify-between sm:px-5">
          <h2 className="text-lg font-bold text-slate-900 dark:text-white">{t.selectPlaylistVideos}</h2>
          <span className="text-sm text-slate-500 dark:text-gray-400">
            {selectedVideoKeys.size} {t.selectedVideos}
          </span>
        </div>

        <div className="space-y-5 overflow-y-auto p-4 sm:p-5">
          {playlists.map(item => {
            if (item.loading) {
              return (
                <section key={item.url} className="rounded-xl border border-slate-200 bg-slate-100 p-5 text-slate-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
                  {t.loadingPlaylist}
                </section>
              );
            }

            if (item.error || !item.playlist) {
              return (
                <section key={item.url} className="rounded-xl border border-red-200 bg-red-50 p-5 dark:border-red-900 dark:bg-gray-800">
                  <p className="text-sm text-red-600 dark:text-red-400">{item.error || t.playlistLoadError}</p>
                  <button
                    type="button"
                    onClick={() => onRetry(item.url)}
                    className="mt-3 text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                  >
                    {t.retry}
                  </button>
                </section>
              );
            }

            const availableVideos = item.playlist.videos.filter(video => video.available);
            const allSelected = availableVideos.length > 0
              && availableVideos.every(video => selectedVideoKeys.has(`${item.playlist!.id}:${video.index}`));

            return (
              <section key={item.url} className="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
                <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 dark:border-gray-700 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <h3 className="truncate font-semibold text-slate-900 dark:text-white">{item.playlist.title}</h3>
                    <p className="truncate text-xs text-slate-500 dark:text-gray-400">{item.playlist.author}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => onTogglePlaylist(item.playlist!)}
                    className="self-start whitespace-nowrap rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-600 hover:border-red-300 hover:text-red-700 dark:border-red-800/70 dark:bg-red-950/40 dark:text-red-300 dark:hover:bg-red-950/60 sm:self-auto"
                  >
                    {allSelected ? t.clearSelection : t.selectAll}
                  </button>
                </div>

                <div className="divide-y divide-slate-200 dark:divide-gray-700">
                  {item.playlist.videos.map(video => {
                    const selectionKey = `${item.playlist!.id}:${video.index}`;
                    return (
                      <label
                        key={selectionKey}
                        className={`flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center ${video.available ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-gray-700/50' : 'opacity-55'}`}
                      >
                        <div className="flex items-start gap-3 sm:items-center">
                          <input
                            type="checkbox"
                            checked={video.available && selectedVideoKeys.has(selectionKey)}
                            disabled={!video.available}
                            onChange={() => onToggleVideo(selectionKey)}
                            className="mt-1 h-4 w-4 accent-red-600 sm:mt-0"
                          />
                          {video.thumbnail_url && (
                            <img
                              src={video.thumbnail_url}
                              alt=""
                              className="h-12 w-20 flex-none rounded object-cover"
                            />
                          )}
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="break-words text-sm text-slate-900 dark:text-gray-100 sm:truncate">{video.title}</p>
                          <p className="break-words text-xs text-slate-500 dark:text-gray-400 sm:truncate">
                            {video.available
                              ? `${video.author}${video.author ? ' · ' : ''}${formatDuration(video.duration_seconds)}`
                              : `${t.unavailable}${video.unavailable_reason ? ` · ${video.unavailable_reason}` : ''}`}
                          </p>
                        </div>
                      </label>
                    );
                  })}
                </div>
              </section>
            );
          })}
        </div>

        <div className="flex flex-col-reverse gap-3 border-t border-slate-200 px-4 py-4 dark:border-gray-700 sm:flex-row sm:justify-end sm:px-5">
          <button
            type="button"
            onClick={onClose}
            className="w-full rounded-lg border border-slate-300 bg-slate-100 px-4 py-2 text-sm text-slate-800 hover:bg-slate-200 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 sm:w-auto"
          >
            {t.close}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!canConfirm}
            className="w-full rounded-lg bg-red-600 px-5 py-2 text-sm font-bold text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
          >
            {t.addSelected}
          </button>
        </div>
      </div>
    </div>
  );
}

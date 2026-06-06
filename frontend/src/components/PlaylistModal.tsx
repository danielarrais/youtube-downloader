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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-6">
      <div className="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-gray-700 bg-gray-900 shadow-2xl">
        <div className="flex items-center justify-between border-b border-gray-700 px-5 py-4">
          <h2 className="text-lg font-bold text-white">{t.selectPlaylistVideos}</h2>
          <span className="text-sm text-gray-400">
            {selectedVideoKeys.size} {t.selectedVideos}
          </span>
        </div>

        <div className="space-y-5 overflow-y-auto p-5">
          {playlists.map(item => {
            if (item.loading) {
              return (
                <section key={item.url} className="rounded-lg border border-gray-700 bg-gray-800 p-5 text-gray-300">
                  {t.loadingPlaylist}
                </section>
              );
            }

            if (item.error || !item.playlist) {
              return (
                <section key={item.url} className="rounded-lg border border-red-900 bg-gray-800 p-5">
                  <p className="text-sm text-red-400">{item.error || t.playlistLoadError}</p>
                  <button
                    type="button"
                    onClick={() => onRetry(item.url)}
                    className="mt-3 text-sm text-blue-400 hover:text-blue-300"
                  >
                    {t.retry}
                  </button>
                </section>
              );
            }

            const availableVideos = item.playlist.videos.filter(video => video.available);
            const allSelected = availableVideos.length > 0 &&
              availableVideos.every(video => selectedVideoKeys.has(`${item.playlist!.id}:${video.index}`));

            return (
              <section key={item.url} className="overflow-hidden rounded-lg border border-gray-700 bg-gray-800">
                <div className="flex items-center justify-between border-b border-gray-700 px-4 py-3">
                  <div className="min-w-0">
                    <h3 className="truncate font-semibold text-white">{item.playlist.title}</h3>
                    <p className="truncate text-xs text-gray-400">{item.playlist.author}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => onTogglePlaylist(item.playlist!)}
                    className="ml-4 whitespace-nowrap text-xs text-red-400 hover:text-red-300"
                  >
                    {allSelected ? t.clearSelection : t.selectAll}
                  </button>
                </div>

                <div className="divide-y divide-gray-700">
                  {item.playlist.videos.map(video => {
                    const selectionKey = `${item.playlist!.id}:${video.index}`;
                    return (
                    <label
                      key={selectionKey}
                      className={`flex items-center gap-3 px-4 py-3 ${video.available ? 'cursor-pointer hover:bg-gray-700/50' : 'opacity-55'}`}
                    >
                      <input
                        type="checkbox"
                        checked={video.available && selectedVideoKeys.has(selectionKey)}
                        disabled={!video.available}
                        onChange={() => onToggleVideo(selectionKey)}
                        className="h-4 w-4 accent-red-600"
                      />
                      {video.thumbnail_url && (
                        <img
                          src={video.thumbnail_url}
                          alt=""
                          className="h-12 w-20 flex-none rounded object-cover"
                        />
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm text-gray-100">{video.title}</p>
                        <p className="truncate text-xs text-gray-400">
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

        <div className="flex justify-end gap-3 border-t border-gray-700 px-5 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-gray-600 bg-gray-700 px-4 py-2 text-sm text-gray-200 hover:bg-gray-600"
          >
            {t.close}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!canConfirm}
            className="rounded-lg bg-red-600 px-5 py-2 text-sm font-bold text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t.addSelected}
          </button>
        </div>
      </div>
    </div>
  );
}

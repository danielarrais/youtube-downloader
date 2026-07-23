import { DownloadItem as DownloadItemType, QueueStats } from '../types';
import { DownloadItem } from './DownloadItem';
import { useTranslation } from '../hooks/useTranslation';

type QueueFilter = 'downloading' | 'pending' | 'completed' | 'failed' | null;

interface DownloadQueueProps {
  downloads: DownloadItemType[];
  allDownloads: DownloadItemType[];
  stats: QueueStats;
  activeFilter: QueueFilter;
  onFilterChange: (filter: QueueFilter) => void;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
  onDelete: (item: DownloadItemType) => void;
  onRetryFailed: () => void;
  onClearCompleted: () => void;
  onCancelAll: () => void;
  paused: boolean;
  onPause: () => void;
  onResume: () => void;
  onClearAll: () => void;
}

export function DownloadQueue({
  downloads,
  allDownloads,
  stats,
  activeFilter,
  onFilterChange,
  onCancel,
  onRetry,
  onDelete,
  onRetryFailed,
  onClearCompleted,
  onCancelAll,
  paused,
  onPause,
  onResume,
  onClearAll,
}: DownloadQueueProps) {
  const { t } = useTranslation();
  const hasCompleted = allDownloads.some(d => d.status === 'completed' || d.status === 'skipped');
  const hasPending = allDownloads.some(d => ['pending', 'fetching_info', 'downloading', 'converting'].includes(d.status));
  const hasFailed = allDownloads.some(d => d.status === 'failed');

  if (allDownloads.length === 0) {
    return (
      <div className="py-12 text-center text-slate-500 dark:text-gray-400">
        {t.emptyQueue}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-start gap-2 sm:justify-end sm:gap-3">
        {stats.downloading > 0 && (
          <button
            type="button"
            onClick={() => onFilterChange(activeFilter === 'downloading' ? null : 'downloading')}
            className={`rounded-lg border px-2.5 py-1.5 text-xs transition-colors ${activeFilter === 'downloading' ? 'border-blue-600 bg-blue-600 text-white dark:border-blue-500 dark:bg-blue-500' : 'border-blue-200 bg-blue-50 text-blue-700 hover:border-blue-300 dark:border-blue-800/70 dark:bg-blue-950/40 dark:text-blue-300 dark:hover:bg-blue-950/60'}`}
          >
            {t.downloading}: <span className="font-medium">{stats.downloading}</span>
          </button>
        )}
        {stats.pending > 0 && (
          <button
            type="button"
            onClick={() => onFilterChange(activeFilter === 'pending' ? null : 'pending')}
            className={`rounded-lg border px-2.5 py-1.5 text-xs transition-colors ${activeFilter === 'pending' ? 'border-amber-500 bg-amber-500 text-white dark:border-amber-500 dark:bg-amber-500' : 'border-amber-200 bg-amber-50 text-amber-700 hover:border-amber-300 dark:border-yellow-800/70 dark:bg-yellow-950/40 dark:text-yellow-300 dark:hover:bg-yellow-950/60'}`}
          >
            {t.pending}: <span className="font-medium">{stats.pending}</span>
          </button>
        )}
        {stats.completed > 0 && (
          <button
            type="button"
            onClick={() => onFilterChange(activeFilter === 'completed' ? null : 'completed')}
            className={`rounded-lg border px-2.5 py-1.5 text-xs transition-colors ${activeFilter === 'completed' ? 'border-emerald-600 bg-emerald-600 text-white dark:border-emerald-500 dark:bg-emerald-500' : 'border-emerald-200 bg-emerald-50 text-emerald-700 hover:border-emerald-300 dark:border-green-800/70 dark:bg-green-950/40 dark:text-green-300 dark:hover:bg-green-950/60'}`}
          >
            {t.completed}: <span className="font-medium">{stats.completed}</span>
          </button>
        )}
        {stats.failed > 0 && (
          <button
            type="button"
            onClick={() => onFilterChange(activeFilter === 'failed' ? null : 'failed')}
            className={`rounded-lg border px-2.5 py-1.5 text-xs transition-colors ${activeFilter === 'failed' ? 'border-rose-600 bg-rose-600 text-white dark:border-rose-500 dark:bg-rose-500' : 'border-rose-200 bg-rose-50 text-rose-700 hover:border-rose-300 dark:border-red-800/70 dark:bg-red-950/40 dark:text-red-300 dark:hover:bg-red-950/60'}`}
          >
            {t.failed}: <span className="font-medium">{stats.failed}</span>
          </button>
        )}
        {(paused || hasFailed || hasPending) && (
          <button
            onClick={hasFailed ? onRetryFailed : paused ? onResume : onPause}
            className="rounded-lg border border-slate-300 bg-slate-100 px-3 py-1.5 text-xs text-slate-600 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600"
          >
            {paused || hasFailed ? t.resumeQueue : t.pauseQueue}
          </button>
        )}
        {hasPending && (
          <button
            onClick={onCancelAll}
            className="rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 text-xs text-red-600 transition-colors hover:border-red-300 hover:text-red-700 dark:border-red-800/70 dark:bg-red-950/40 dark:text-red-300 dark:hover:bg-red-950/60"
          >
            {t.cancelAll}
          </button>
        )}
        {hasCompleted && (
          <button
            onClick={onClearCompleted}
            className="rounded-lg border border-slate-300 bg-slate-100 px-3 py-1.5 text-xs text-slate-600 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600"
          >
            {t.clearCompleted}
          </button>
        )}
        <button
          onClick={onClearAll}
          className="rounded-lg border border-slate-300 bg-slate-100 px-3 py-1.5 text-xs text-slate-600 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600"
        >
          {t.clearAll}
        </button>
      </div>

      {downloads.length === 0 ? (
        <div className="py-8 text-center text-sm text-slate-500 dark:text-gray-400">
          {t.filteredDownloadsEmpty}
        </div>
      ) : (
        <div className="space-y-3">
          {downloads.map(item => (
            <DownloadItem
              key={item.id}
              item={item}
              onCancel={onCancel}
              onRetry={onRetry}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}
    </div>
  );
}

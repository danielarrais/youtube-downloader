import { DownloadItem as DownloadItemType } from '../types';
import { DownloadItem } from './DownloadItem';
import { useTranslation } from '../hooks/useTranslation';

interface DownloadQueueProps {
  downloads: DownloadItemType[];
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
  const hasCompleted = downloads.some(d => d.status === 'completed' || d.status === 'skipped');
  const hasPending = downloads.some(d => ['pending', 'fetching_info', 'downloading', 'converting'].includes(d.status));
  const hasFailed = downloads.some(d => d.status === 'failed');

  if (downloads.length === 0) {
    return (
      <div className="text-center text-gray-400 py-12">
        {t.emptyQueue}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end gap-4">
        {(paused || hasFailed || hasPending) && (
          <button
            onClick={hasFailed ? onRetryFailed : paused ? onResume : onPause}
            className="text-sm text-gray-400 hover:text-white transition-colors"
          >
            {paused || hasFailed ? t.resumeQueue : t.pauseQueue}
          </button>
        )}
        {hasPending && (
          <button
            onClick={onCancelAll}
            className="text-sm text-red-400 hover:text-red-300 transition-colors"
          >
            {t.cancelAll}
          </button>
        )}
        {hasCompleted && (
          <button
            onClick={onClearCompleted}
            className="text-sm text-gray-400 hover:text-white transition-colors"
          >
            {t.clearCompleted}
          </button>
        )}
        <button
          onClick={onClearAll}
          className="text-sm text-gray-400 hover:text-white transition-colors"
        >
          {t.clearAll}
        </button>
      </div>

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
    </div>
  );
}

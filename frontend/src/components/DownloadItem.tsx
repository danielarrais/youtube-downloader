import { DownloadItem as DownloadItemType } from '../types';
import { StatusBadge } from './StatusBadge';
import { ProgressBar } from './ProgressBar';
import { api } from '../services/api';
import { useTranslation } from '../hooks/useTranslation';

interface DownloadItemProps {
  item: DownloadItemType;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}

export function DownloadItem({ item, onCancel, onRetry }: DownloadItemProps) {
  const { t } = useTranslation();
  const showProgress = ['downloading', 'converting', 'fetching_info'].includes(item.status);
  const canCancel = ['pending', 'fetching_info', 'downloading', 'converting'].includes(item.status);
  const canRetry = ['failed', 'cancelled'].includes(item.status);
  const canDownload = ['completed', 'skipped'].includes(item.status);

  return (
    <div className="bg-gray-800 rounded-lg p-4 space-y-3">
      <div className="flex items-start gap-3">
        {item.thumbnail_url ? (
          <img
            src={item.thumbnail_url}
            alt=""
            className="h-14 w-14 flex-none rounded object-cover"
          />
        ) : (
          <div className="h-14 w-14 flex-none rounded bg-gray-700 flex items-center justify-center">
            <svg className="h-7 w-7 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M12 17.25a.375.375 0 100 .75.375.375 0 000-.75"
              />
            </svg>
          </div>
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <h3 className="font-medium text-white truncate">
                {item.title || item.url}
              </h3>
              {item.title && (
                <p className="text-sm text-gray-400 truncate">{item.url}</p>
              )}
            </div>
            <StatusBadge status={item.status} />
          </div>

          {showProgress && (
            <div className="mt-3 space-y-1">
              <ProgressBar percent={item.progress.percent} status={item.status} />
              <div className="flex justify-between text-xs text-gray-400">
                <span>{item.progress.percent.toFixed(1)}%</span>
                {item.progress.speed && <span>{item.progress.speed}</span>}
              </div>
            </div>
          )}

          {item.error && (
            <p className="mt-2 text-sm text-red-400">{item.error}</p>
          )}

          <div className="mt-2 flex gap-2">
            {canCancel && (
              <button
                onClick={() => onCancel(item.id)}
                className="text-sm text-gray-400 hover:text-red-400 transition-colors"
              >
                {t.cancel}
              </button>
            )}
            {canRetry && (
              <button
                onClick={() => onRetry(item.id)}
                className="text-sm text-gray-400 hover:text-blue-400 transition-colors"
              >
                {t.retry}
              </button>
            )}
            {canDownload && (
              <button
                onClick={() => api.openDownload(item)}
                className="text-sm text-green-400 hover:text-green-300 transition-colors"
              >
                {api.capabilities.nativeFolders
                  ? t.openFolder
                  : item.media_type === 'video' ? t.downloadVideoFile : t.downloadFile}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

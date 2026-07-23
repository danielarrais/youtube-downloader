import { DownloadStatus } from '../types';
import { useTranslation } from '../hooks/useTranslation';

interface StatusBadgeProps {
  status: DownloadStatus;
}

const statusColors: Record<DownloadStatus, string> = {
  pending: 'bg-gray-500',
  fetching_info: 'bg-blue-400',
  downloading: 'bg-blue-500',
  converting: 'bg-yellow-500',
  completed: 'bg-green-500',
  failed: 'bg-red-500',
  cancelled: 'bg-gray-500',
  skipped: 'bg-green-500'
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const { t } = useTranslation();
  const color = statusColors[status];
  const label = t.status[status];

  return (
    <span className={`${color} inline-flex rounded-full px-2 py-1 text-xs text-white whitespace-nowrap`}>
      {label}
    </span>
  );
}

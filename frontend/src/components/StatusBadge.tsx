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
  skipped: 'bg-purple-500'
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const { t } = useTranslation();
  const color = statusColors[status];
  const label = t.status[status];

  return (
    <span className={`${color} text-white text-xs px-2 py-1 rounded-full whitespace-nowrap`}>
      {label}
    </span>
  );
}

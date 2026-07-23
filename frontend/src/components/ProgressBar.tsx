import { DownloadStatus } from '../types';

interface ProgressBarProps {
  percent: number;
  status: DownloadStatus;
}

export function ProgressBar({ percent, status }: ProgressBarProps) {
  const getColor = () => {
    switch (status) {
      case 'downloading':
        return 'bg-blue-500';
      case 'converting':
        return 'bg-yellow-500';
      case 'completed':
      case 'skipped':
        return 'bg-green-500';
      case 'failed':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  return (
    <div className="h-2 w-full rounded-full bg-slate-200 dark:bg-gray-700">
      <div
        className={`${getColor()} h-2 rounded-full transition-all duration-300`}
        style={{ width: `${percent}%` }}
      />
    </div>
  );
}

import { useState } from 'react';
import { DeleteConfirmModal } from './components/DeleteConfirmModal';
import { Header } from './components/Header';
import { UrlInput } from './components/UrlInput';
import { DownloadQueue } from './components/DownloadQueue';
import { useDownloads } from './hooks/useDownloads';
import { useTranslation } from './hooks/useTranslation';

type QueueFilter = 'downloading' | 'pending' | 'completed' | 'failed' | null;

interface DeleteConfirmState {
  title: string;
  message: string;
  includeDeleteFileOption: boolean;
  deleteFile: boolean;
  resolve: (value: { confirmed: boolean; deleteFile: boolean }) => void;
}

function App() {
  const { t } = useTranslation();
  const [activeFilter, setActiveFilter] = useState<QueueFilter>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<DeleteConfirmState | null>(null);
  const {
    downloads,
    stats,
    addDownloads,
    addVideoDownloads,
    cancelDownload,
    removeDownload,
    retryDownload,
    retryFailed,
    clearCompleted,
    cancelAll,
    pauseQueue,
    resumeQueue,
    clearAll,
  } = useDownloads((options) => new Promise(resolve => {
    setDeleteConfirm({
      ...options,
      deleteFile: false,
      resolve,
    });
  }));

  const filteredDownloads = downloads.filter(item => {
    switch (activeFilter) {
      case 'downloading':
        return ['downloading', 'converting'].includes(item.status);
      case 'pending':
        return ['pending', 'fetching_info', 'converting'].includes(item.status);
      case 'completed':
        return item.status === 'completed' || item.status === 'skipped';
      case 'failed':
        return item.status === 'failed';
      default:
        return true;
    }
  });

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900 transition-colors dark:bg-gray-900 dark:text-gray-100">
      <Header onOpenSettings={() => setSettingsOpen(true)} />
      
      <main className="mx-auto max-w-4xl space-y-6 px-4 py-4 sm:px-6 sm:py-8">
        <UrlInput
          onSubmitAudio={addDownloads}
          onSubmitVideo={addVideoDownloads}
          settingsOpen={settingsOpen}
          onOpenSettings={() => setSettingsOpen(true)}
          onCloseSettings={() => setSettingsOpen(false)}
        />
        
        <DownloadQueue
          downloads={filteredDownloads}
          allDownloads={downloads}
          stats={stats}
          activeFilter={activeFilter}
          onFilterChange={setActiveFilter}
          onCancel={cancelDownload}
          onRetry={retryDownload}
          onDelete={removeDownload}
          onRetryFailed={retryFailed}
          onClearCompleted={clearCompleted}
          onCancelAll={cancelAll}
          paused={stats.paused}
          onPause={pauseQueue}
          onResume={resumeQueue}
          onClearAll={clearAll}
        />
      </main>

      {deleteConfirm && (
        <DeleteConfirmModal
          title={deleteConfirm.title}
          message={deleteConfirm.message}
          confirmLabel={t.confirmDelete}
          cancelLabel={t.close}
          includeDeleteFileOption={deleteConfirm.includeDeleteFileOption}
          deleteFile={deleteConfirm.deleteFile}
          deleteFileLabel={t.removeFromDisk}
          onToggleDeleteFile={(value) => setDeleteConfirm(current => current ? { ...current, deleteFile: value } : null)}
          onCancel={() => {
            deleteConfirm.resolve({ confirmed: false, deleteFile: false });
            setDeleteConfirm(null);
          }}
          onConfirm={() => {
            deleteConfirm.resolve({ confirmed: true, deleteFile: deleteConfirm.deleteFile });
            setDeleteConfirm(null);
          }}
        />
      )}
    </div>
  );
}

export default App;

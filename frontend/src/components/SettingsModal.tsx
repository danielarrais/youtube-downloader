import { useState } from 'react';
import { useTranslation } from '../hooks/useTranslation';

interface SettingsModalProps {
  downloadDir: string;
  quality: string;
  videoContainer: 'mp4' | 'webm' | 'mkv';
  videoQuality: '144p' | '240p' | '360p' | '480p' | '720p' | '1080p' | '1440p' | '2160p';
  onChooseFolder: () => Promise<string | undefined>;
  onClose: () => void;
  onSave: (
    downloadDir: string,
    quality: string,
    videoContainer: 'mp4' | 'webm' | 'mkv',
    videoQuality: '144p' | '240p' | '360p' | '480p' | '720p' | '1080p' | '1440p' | '2160p',
  ) => Promise<void>;
  canChooseFolder: boolean;
}

export function SettingsModal({
  downloadDir,
  quality,
  videoContainer,
  videoQuality,
  onChooseFolder,
  onClose,
  onSave,
  canChooseFolder,
}: SettingsModalProps) {
  const { t } = useTranslation();
  const [selectedDir, setSelectedDir] = useState(downloadDir);
  const [selectedQuality, setSelectedQuality] = useState(quality);
  const [selectedVideoContainer, setSelectedVideoContainer] = useState(videoContainer);
  const [selectedVideoQuality, setSelectedVideoQuality] = useState(videoQuality);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const chooseFolder = async () => {
    const folder = await onChooseFolder();
    if (folder) setSelectedDir(folder);
  };

  const save = async () => {
    setSaving(true);
    setError('');
    try {
      await onSave(selectedDir, selectedQuality, selectedVideoContainer, selectedVideoQuality);
      onClose();
    } catch {
      setError(t.settingsSaveError);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-6">
      <div className="w-full max-w-lg rounded-lg border border-gray-700 bg-gray-900 shadow-2xl">
        <div className="border-b border-gray-700 px-5 py-4">
          <h2 className="text-lg font-bold text-white">{t.settings}</h2>
        </div>

        <div className="space-y-5 p-5">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">{t.downloadFolder}</label>
            <div className="rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-300 break-all">
              {selectedDir}
            </div>
            {canChooseFolder && (
              <button
                type="button"
                onClick={chooseFolder}
                className="rounded-lg border border-gray-600 bg-gray-700 px-4 py-2 text-sm text-gray-200 hover:bg-gray-600"
              >
                {t.chooseFolder}
              </button>
            )}
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">{t.videoFormat}</label>
            <select
              value={selectedVideoContainer}
              onChange={(event) => setSelectedVideoContainer(event.target.value as typeof selectedVideoContainer)}
              className="app-select w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white focus:border-red-500 focus:outline-none"
            >
              <option value="mp4">MP4</option>
              <option value="webm">WebM</option>
              <option value="mkv">MKV</option>
            </select>
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">{t.videoQuality}</label>
            <select
              value={selectedVideoQuality}
              onChange={(event) => setSelectedVideoQuality(event.target.value as typeof selectedVideoQuality)}
              className="app-select w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white focus:border-red-500 focus:outline-none"
            >
              {['144p', '240p', '360p', '480p', '720p', '1080p', '1440p', '2160p'].map(value => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">{t.quality}</label>
            <select
              value={selectedQuality}
              onChange={(event) => setSelectedQuality(event.target.value)}
              className="app-select w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white focus:border-red-500 focus:outline-none"
            >
              <option value="128k">128 kbps</option>
              <option value="192k">192 kbps</option>
              <option value="320k">320 kbps</option>
            </select>
          </div>
          {error && <p className="text-sm text-red-400">{error}</p>}
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
            onClick={save}
            disabled={saving}
            className="rounded-lg bg-red-600 px-5 py-2 text-sm font-bold text-white hover:bg-red-700 disabled:opacity-50"
          >
            {t.save}
          </button>
        </div>
      </div>
    </div>
  );
}

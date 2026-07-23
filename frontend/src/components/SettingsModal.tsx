import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { Theme } from '../contexts/ThemeContext';
import { useTranslation } from '../hooks/useTranslation';
import { Language } from '../i18n';

interface SettingsModalProps {
  downloadDir: string;
  quality: string;
  videoContainer: 'mp4' | 'webm' | 'mkv';
  videoQuality: '144p' | '240p' | '360p' | '480p' | '720p' | '1080p' | '1440p' | '2160p';
  fileDeletion: 'delete' | 'ask' | 'keep';
  language: Language;
  theme: Theme;
  onThemePreview: (theme: Theme) => void;
  onChooseFolder: () => Promise<string | undefined>;
  onClose: () => void;
  onSave: (
    downloadDir: string,
    quality: string,
    videoContainer: 'mp4' | 'webm' | 'mkv',
    videoQuality: '144p' | '240p' | '360p' | '480p' | '720p' | '1080p' | '1440p' | '2160p',
    fileDeletion: 'delete' | 'ask' | 'keep',
    language: Language,
    theme: Theme,
  ) => Promise<void>;
  canChooseFolder: boolean;
}

export function SettingsModal({
  downloadDir,
  quality,
  videoContainer,
  videoQuality,
  fileDeletion,
  language,
  theme,
  onThemePreview,
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
  const [selectedFileDeletion, setSelectedFileDeletion] = useState(fileDeletion);
  const [selectedLanguage, setSelectedLanguage] = useState(language);
  const [selectedTheme, setSelectedTheme] = useState(theme);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const previousBodyOverflow = document.body.style.overflow;
    const previousHtmlOverflow = document.documentElement.style.overflow;

    document.body.style.overflow = 'hidden';
    document.documentElement.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = previousBodyOverflow;
      document.documentElement.style.overflow = previousHtmlOverflow;
    };
  }, []);

  const chooseFolder = async () => {
    const folder = await onChooseFolder();
    if (folder) setSelectedDir(folder);
  };

  const previewTheme = (value: Theme) => {
    setSelectedTheme(value);
    onThemePreview(value);
  };

  const save = async () => {
    setSaving(true);
    setError('');
    try {
      await onSave(
        selectedDir,
        selectedQuality,
        selectedVideoContainer,
        selectedVideoQuality,
        selectedFileDeletion,
        selectedLanguage,
        selectedTheme,
      );
    } catch {
      setError(t.settingsSaveError);
    } finally {
      setSaving(false);
    }
  };

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 sm:p-6">
      <div className="max-h-[calc(100vh-2rem)] w-full max-w-lg overflow-y-auto rounded-xl border border-slate-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-900 sm:max-h-[calc(100vh-3rem)]">
        <div className="border-b border-slate-200 px-4 py-4 dark:border-gray-700 sm:px-5">
          <h2 className="text-lg font-bold text-slate-900 dark:text-white">{t.settings}</h2>
        </div>

        <div className="space-y-5 p-4 sm:p-5">
          {canChooseFolder && (
            <div className="space-y-2">
              <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.downloadFolder}</label>
              <div className="break-all rounded-lg border border-slate-300 bg-slate-100 px-3 py-2 text-sm text-slate-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
                {selectedDir}
              </div>
              <button
                type="button"
                onClick={chooseFolder}
                className="w-full rounded-lg border border-slate-300 bg-slate-100 px-4 py-2 text-sm text-slate-800 hover:bg-slate-200 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 sm:w-auto"
              >
                {t.chooseFolder}
              </button>
            </div>
          )}

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">Idioma</label>
            <select
              value={selectedLanguage}
              onChange={(event) => setSelectedLanguage(event.target.value as Language)}
              className="app-select w-full rounded-lg border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-white"
            >
              <option value="pt-BR">🇧🇷 PT-BR</option>
              <option value="en-US">🇺🇸 EN-US</option>
            </select>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.theme}</label>
            <div className="flex overflow-hidden rounded-lg border border-slate-300 bg-slate-100 dark:border-gray-600 dark:bg-gray-800" role="group" aria-label={t.theme}>
              <button
                type="button"
                onClick={() => previewTheme('dark')}
                aria-pressed={selectedTheme === 'dark'}
                className={`flex flex-1 items-center justify-center gap-2 px-3 py-2 text-sm transition-colors ${selectedTheme === 'dark' ? 'bg-red-600 text-white' : 'text-slate-700 hover:bg-slate-200 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700'}`}
              >
                {t.themeDark}
              </button>
              <button
                type="button"
                onClick={() => previewTheme('light')}
                aria-pressed={selectedTheme === 'light'}
                className={`flex flex-1 items-center justify-center gap-2 border-l border-slate-300 px-3 py-2 text-sm transition-colors dark:border-gray-600 ${selectedTheme === 'light' ? 'bg-red-600 text-white' : 'text-slate-700 hover:bg-slate-200 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700'}`}
              >
                {t.themeLight}
              </button>
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.videoFormat}</label>
            <select
              value={selectedVideoContainer}
              onChange={(event) => setSelectedVideoContainer(event.target.value as typeof selectedVideoContainer)}
              className="app-select w-full rounded-lg border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-white"
            >
              <option value="mp4">MP4</option>
              <option value="webm">WebM</option>
              <option value="mkv">MKV</option>
            </select>
          </div>

          {canChooseFolder && (
            <div className="space-y-2">
              <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.fileDeletion}</label>
              <select
                value={selectedFileDeletion}
                onChange={(event) => setSelectedFileDeletion(event.target.value as typeof selectedFileDeletion)}
                className="app-select w-full rounded-lg border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-white"
              >
                <option value="delete">{t.fileDeletionDelete}</option>
                <option value="ask">{t.fileDeletionAsk}</option>
                <option value="keep">{t.fileDeletionKeep}</option>
              </select>
            </div>
          )}

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.videoQuality}</label>
            <select
              value={selectedVideoQuality}
              onChange={(event) => setSelectedVideoQuality(event.target.value as typeof selectedVideoQuality)}
              className="app-select w-full rounded-lg border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-white"
            >
              {['144p', '240p', '360p', '480p', '720p', '1080p', '1440p', '2160p'].map(value => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700 dark:text-gray-300">{t.quality}</label>
            <select
              value={selectedQuality}
              onChange={(event) => setSelectedQuality(event.target.value)}
              className="app-select w-full rounded-lg border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 focus:border-red-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-white"
            >
              <option value="128k">128 kbps</option>
              <option value="192k">192 kbps</option>
              <option value="320k">320 kbps</option>
            </select>
          </div>

          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
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
            onClick={save}
            disabled={saving}
            className="w-full rounded-lg bg-red-600 px-5 py-2 text-sm font-bold text-white hover:bg-red-700 disabled:opacity-50 sm:w-auto"
          >
            {t.save}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}

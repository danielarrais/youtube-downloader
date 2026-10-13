import { useState } from 'react';
import { DownloadItem as DownloadItemType } from '../types';
import { StatusBadge } from './StatusBadge';
import { ProgressBar } from './ProgressBar';
import { api } from '../services/api';
import { useTranslation } from '../hooks/useTranslation';

function itemQualityLabel(item: DownloadItemType) {
  if (item.media_type === 'video') {
    const extension = item.video_format?.extension?.toUpperCase() || item.video_format?.container?.toUpperCase() || 'VIDEO';
    const resolution = item.video_format?.resolution || item.quality;
    return resolution ? `${extension} ${resolution}` : extension;
  }
  const bitrate = item.audio_format?.bitrate ? `${item.audio_format.bitrate}k` : item.quality;
  const codec = item.audio_format?.audio_codec || item.audio_format?.container;
  return codec && bitrate ? `MP3 • ${codec.toUpperCase()} ${bitrate}` : bitrate ? `MP3 ${bitrate}` : 'MP3';
}

function formatFileSize(size?: number) {
  if (!size || size <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = size;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex++;
  }
  return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function sourceLabel(url: string) {
  try {
    const parsed = new URL(url.includes('://') ? url : `https://${url}`);
    const host = parsed.hostname.toLowerCase();
    if (host === 'youtu.be' || host === 'youtube.com' || host.endsWith('.youtube.com') || host === 'youtube-nocookie.com' || host.endsWith('.youtube-nocookie.com')) return 'YouTube';
    if (host === 'x.com' || host.endsWith('.x.com') || host === 'twitter.com' || host.endsWith('.twitter.com')) return 'X';
    if (host === 'instagram.com' || host.endsWith('.instagram.com')) return 'Instagram';
    if (host === 'tiktok.com' || host.endsWith('.tiktok.com')) return 'TikTok';
  } catch {
    return '';
  }
  return '';
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }
  const textArea = document.createElement('textarea');
  textArea.value = value;
  textArea.setAttribute('readonly', '');
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.select();
  document.execCommand('copy');
  document.body.removeChild(textArea);
}

interface DownloadItemProps {
  item: DownloadItemType;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
  onDelete: (item: DownloadItemType) => void;
}

export function DownloadItem({ item, onCancel, onRetry, onDelete }: DownloadItemProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const showProgress = ['downloading', 'converting', 'fetching_info'].includes(item.status);
  const canCancel = ['pending', 'fetching_info', 'downloading', 'converting'].includes(item.status);
  const canRetry = ['failed', 'cancelled'].includes(item.status);
  const canDownload = ['completed', 'skipped'].includes(item.status);
  const fileSize = ['completed', 'skipped'].includes(item.status) ? formatFileSize(item.file_size) : '';
  const linkClassName = 'text-sm text-blue-600 underline-offset-2 hover:underline dark:text-blue-400';
  const source = item.media_type === 'video' ? sourceLabel(item.url) : '';

  const handleCopyLink = async () => {
    try {
      await copyText(item.url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1400);
    } catch (error) {
      console.error('Erro ao copiar link:', error);
    }
  };

  return (
    <div className="space-y-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm transition-colors dark:border-gray-700 dark:bg-gray-800">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        {item.thumbnail_url ? (
          <img
            src={item.thumbnail_url}
            alt=""
            className="h-16 w-full rounded-lg object-cover sm:h-14 sm:w-14 sm:flex-none"
          />
        ) : (
          <div className="flex h-16 w-full items-center justify-center rounded-lg bg-slate-200 dark:bg-gray-700 sm:h-14 sm:w-14 sm:flex-none">
            <svg className="h-7 w-7 text-slate-500 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
          <div className="flex flex-col gap-3">
            <div className="min-w-0 flex-1">
              {item.title ? (
                <h3 className="break-words font-medium text-slate-900 dark:text-white sm:truncate">
                  {item.title}
                </h3>
              ) : (
                <a
                  href={item.url}
                  target="_blank"
                  rel="noreferrer"
                  className={`break-all font-medium ${linkClassName}`}
                >
                  {item.url}
                </a>
              )}
              {item.title && (
                <a
                  href={item.url}
                  target="_blank"
                  rel="noreferrer"
                  className={`block break-all text-slate-500 dark:text-gray-400 sm:truncate ${linkClassName}`}
                >
                  {item.url}
                </a>
              )}
            </div>
          </div>

          {showProgress && (
            <div className="mt-3 space-y-1">
              <ProgressBar percent={item.progress.percent} status={item.status} />
              <div className="flex flex-wrap justify-between gap-2 text-xs text-slate-500 dark:text-gray-400">
                <span>{item.progress.percent.toFixed(1)}%</span>
                {item.progress.speed && <span>{item.progress.speed}</span>}
              </div>
            </div>
          )}

          {item.error && (
            <p className="mt-2 break-words text-sm text-red-600 dark:text-red-400">{item.error}</p>
          )}

          <div className="mt-3 flex items-center justify-between gap-3">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <span className="inline-flex whitespace-nowrap rounded-full border border-slate-300 bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100">
                {itemQualityLabel(item)}
              </span>
              {source && (
                <span className="inline-flex whitespace-nowrap rounded-full border border-indigo-200 bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:border-indigo-800/70 dark:bg-indigo-950/40 dark:text-indigo-300">
                  {source}
                </span>
              )}
              <StatusBadge status={item.status} />
            </div>

            <div className="flex flex-row flex-wrap items-center justify-end gap-2">
              <button
                type="button"
                onClick={handleCopyLink}
                className="rounded-lg border border-slate-300 bg-slate-100 p-2 text-slate-600 transition-colors hover:border-blue-300 hover:text-blue-600 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 dark:hover:text-blue-300"
                title={copied ? t.linkCopied : t.copyLink}
                aria-label={copied ? t.linkCopied : t.copyLink}
              >
                {copied ? (
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="m5 13 4 4L19 7" />
                  </svg>
                ) : (
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M8 8h10v10H8z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M6 16H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                  </svg>
                )}
              </button>
              {fileSize && (
                <span className="rounded-lg border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-medium text-slate-500 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-300">
                  {fileSize}
                </span>
              )}
              {canCancel && (
                <button
                  type="button"
                  onClick={() => onCancel(item.id)}
                  className="rounded-lg border border-slate-300 bg-slate-100 p-2 text-slate-600 transition-colors hover:border-red-300 hover:text-red-600 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 dark:hover:text-red-300"
                  title={t.cancel}
                  aria-label={t.cancel}
                >
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M6 6l12 12M18 6 6 18" />
                  </svg>
                </button>
              )}
              {canRetry && (
                <button
                  type="button"
                  onClick={() => onRetry(item.id)}
                  className="rounded-lg border border-slate-300 bg-slate-100 p-2 text-slate-600 transition-colors hover:border-blue-300 hover:text-blue-600 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 dark:hover:text-blue-300"
                  title={t.retry}
                  aria-label={t.retry}
                >
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M4 4v6h6M20 20v-6h-6" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M20 9a8 8 0 0 0-13.66-3L4 10m16 4-2.34 4A8 8 0 0 1 4 15" />
                  </svg>
                </button>
              )}
              {canDownload && (
                <button
                  type="button"
                  onClick={() => api.openDownload(item)}
                  className="rounded-lg border border-emerald-200 bg-emerald-50 p-2 text-emerald-700 transition-colors hover:border-emerald-300 hover:text-emerald-800 dark:border-green-700 dark:bg-green-900/70 dark:text-green-200 dark:hover:bg-green-800/80"
                  title={api.capabilities.nativeFolders
                    ? t.openFolder
                    : item.media_type === 'video' ? t.downloadVideoFile : t.downloadFile}
                  aria-label={api.capabilities.nativeFolders
                    ? t.openFolder
                    : item.media_type === 'video' ? t.downloadVideoFile : t.downloadFile}
                >
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M12 3v11m0 0 4-4m-4 4-4-4M5 17v1a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-1" />
                  </svg>
                </button>
              )}
              {!canCancel && (
                <button
                  type="button"
                  onClick={() => onDelete(item)}
                  className="rounded-lg border border-red-200 bg-red-50 p-2 text-red-600 transition-colors hover:border-red-300 hover:text-red-700 dark:border-red-800/70 dark:bg-red-950/40 dark:text-red-300 dark:hover:bg-red-950/60"
                  title={t.deleteItem}
                  aria-label={t.deleteItem}
                >
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M6 7h12m-9 0V5h6v2m-8 0 1 12h8l1-12" />
                  </svg>
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

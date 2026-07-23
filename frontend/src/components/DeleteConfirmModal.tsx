import { useEffect } from 'react';
import { createPortal } from 'react-dom';

interface DeleteConfirmModalProps {
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  includeDeleteFileOption: boolean;
  deleteFile: boolean;
  deleteFileLabel: string;
  onToggleDeleteFile: (value: boolean) => void;
  onCancel: () => void;
  onConfirm: () => void;
}

export function DeleteConfirmModal({
  title,
  message,
  confirmLabel,
  cancelLabel,
  includeDeleteFileOption,
  deleteFile,
  deleteFileLabel,
  onToggleDeleteFile,
  onCancel,
  onConfirm,
}: DeleteConfirmModalProps) {
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

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 sm:p-6">
      <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-900">
        <div className="border-b border-slate-200 px-4 py-4 dark:border-gray-700 sm:px-5">
          <h2 className="text-lg font-bold text-slate-900 dark:text-white">{title}</h2>
        </div>

        <div className="space-y-4 p-4 sm:p-5">
          <p className="text-sm text-slate-600 dark:text-gray-300">{message}</p>

          {includeDeleteFileOption && (
            <label className="flex items-start gap-3 rounded-lg border border-slate-300 bg-slate-50 px-3 py-3 text-sm text-slate-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200">
              <input
                type="checkbox"
                checked={deleteFile}
                onChange={(event) => onToggleDeleteFile(event.target.checked)}
                className="mt-0.5 h-4 w-4 accent-red-600"
              />
              <span>{deleteFileLabel}</span>
            </label>
          )}
        </div>

        <div className="flex flex-col-reverse gap-3 border-t border-slate-200 px-4 py-4 dark:border-gray-700 sm:flex-row sm:justify-end sm:px-5">
          <button
            type="button"
            onClick={onCancel}
            className="w-full rounded-lg border border-slate-300 bg-slate-100 px-4 py-2 text-sm text-slate-800 hover:bg-slate-200 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600 sm:w-auto"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="w-full rounded-lg bg-red-600 px-5 py-2 text-sm font-bold text-white hover:bg-red-700 sm:w-auto"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}

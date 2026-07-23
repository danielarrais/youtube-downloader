import { useTranslation } from '../hooks/useTranslation';
import { useTheme } from '../hooks/useTheme';
import appLogo from '../assets/images/app-logo.png';
import appLogoWhite from '../assets/images/app-logo-white.png';

interface HeaderProps {
  onOpenSettings: () => void;
}

export function Header({ onOpenSettings }: HeaderProps) {
  const { t } = useTranslation();
  const { theme } = useTheme();

  return (
    <header className="border-b border-slate-200 bg-white/90 px-4 py-4 backdrop-blur dark:border-gray-800 dark:bg-gray-800/90 sm:px-6">
      <div className="mx-auto flex max-w-4xl items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <img src={theme === 'dark' ? appLogoWhite : appLogo} alt={t.title} className="h-10 w-10 rounded-lg" />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-slate-900 dark:text-white">{t.title}</p>
          </div>
        </div>

        <button
          type="button"
          onClick={onOpenSettings}
          className="rounded-lg border border-slate-300 bg-slate-100 p-2 text-slate-700 transition-colors hover:bg-slate-200 dark:border-gray-500 dark:bg-gray-700/90 dark:text-gray-100 dark:hover:bg-gray-600"
          aria-label={t.settings}
          title={t.settings}
        >
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M10.325 4.317a1.724 1.724 0 0 1 3.35 0 1.724 1.724 0 0 0 2.573 1.066 1.724 1.724 0 0 1 2.898 1.675 1.724 1.724 0 0 0 .844 2.49 1.724 1.724 0 0 1 0 2.904 1.724 1.724 0 0 0-.844 2.49 1.724 1.724 0 0 1-2.898 1.675 1.724 1.724 0 0 0-2.573 1.066 1.724 1.724 0 0 1-3.35 0 1.724 1.724 0 0 0-2.573-1.066 1.724 1.724 0 0 1-2.898-1.675 1.724 1.724 0 0 0-.844-2.49 1.724 1.724 0 0 1 0-2.904 1.724 1.724 0 0 0 .844-2.49 1.724 1.724 0 0 1 2.898-1.675 1.724 1.724 0 0 0 2.573-1.066Z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
          </svg>
        </button>
      </div>
    </header>
  );
}

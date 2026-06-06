import { QueueStats } from '../types';
import { useTranslation } from '../hooks/useTranslation';
import { Language } from '../i18n';
import appLogo from '../assets/images/app-logo.png';

interface HeaderProps {
  stats: QueueStats;
}

export function Header({ stats }: HeaderProps) {
  const { language, setLanguage, t } = useTranslation();

  const handleLanguageChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setLanguage(e.target.value as Language);
  };

  return (
    <header className="bg-gray-800 border-b border-gray-700 px-6 py-4">
      <div className="max-w-4xl mx-auto flex items-center justify-between">
        <img src={appLogo} alt={t.title} className="w-10 h-10 rounded-lg" />

        <div className="flex items-center gap-4 text-sm">
          <span className="text-gray-400">
            {t.total}: <span className="text-white font-medium">{stats.total}</span>
          </span>
          <span className="text-blue-400">
            {t.downloading}: <span className="font-medium">{stats.downloading}</span>
          </span>
          <span className="text-yellow-400">
            {t.pending}: <span className="font-medium">{stats.pending}</span>
          </span>
          <span className="text-green-400">
            {t.completed}: <span className="font-medium">{stats.completed}</span>
          </span>
          <span className="text-red-400">
            {t.failed}: <span className="font-medium">{stats.failed}</span>
          </span>

          <select
            value={language}
            onChange={handleLanguageChange}
            className="app-select bg-gray-700 border border-gray-600 rounded px-2 py-1 text-white text-xs focus:outline-none focus:border-red-500"
          >
            <option value="pt-BR">🇧🇷 PT-BR</option>
            <option value="en-US">🇺🇸 EN-US</option>
          </select>
        </div>
      </div>
    </header>
  );
}

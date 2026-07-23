import { createContext, ReactNode, useEffect, useState } from 'react';
import { api } from '../services/api';
import { Config } from '../types';

export type Theme = Config['theme'];

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

const DEFAULT_THEME: Theme = 'dark';

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark');
  document.documentElement.dataset.theme = theme;
}

// eslint-disable-next-line react-refresh/only-export-components
export const ThemeContext = createContext<ThemeContextType | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(DEFAULT_THEME);

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  useEffect(() => {
    let active = true;

    api.getConfig()
      .then(config => {
        if (active) {
          setThemeState(config.theme || DEFAULT_THEME);
        }
      })
      .catch(error => {
        console.error('Erro ao carregar tema:', error);
      });

    return () => {
      active = false;
    };
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme: setThemeState }}>
      {children}
    </ThemeContext.Provider>
  );
}

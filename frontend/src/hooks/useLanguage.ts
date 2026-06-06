import { useState, useEffect } from 'react';
import { Language, translations, Translations } from '../i18n';
import { api } from '../services/api';

const STORAGE_KEY = 'yt-mp3-language';

function getInitialLanguage(): Language {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'pt-BR' || stored === 'en-US') {
    return stored;
  }
  // Detectar idioma do navegador
  const browserLang = navigator.language;
  if (browserLang.startsWith('pt')) {
    return 'pt-BR';
  }
  return 'en-US';
}

export function useLanguage() {
  const [language, setLanguageState] = useState<Language>(getInitialLanguage);

  const setLanguage = (lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem(STORAGE_KEY, lang);
  };

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, language);
    api.setLanguage(language).catch(error => {
      console.error('Erro ao sincronizar idioma:', error);
    });
  }, [language]);

  const t: Translations = translations[language];

  return { language, setLanguage, t };
}

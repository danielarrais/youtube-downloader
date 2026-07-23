import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './App'
import { LanguageProvider } from './contexts/LanguageContext'
import { ThemeProvider } from './contexts/ThemeContext'
import './styles/globals.css'

const container = document.getElementById('root');
const root = createRoot(container!);

root.render(
    <React.StrictMode>
        <ThemeProvider>
            <LanguageProvider>
                <App />
            </LanguageProvider>
        </ThemeProvider>
    </React.StrictMode>
)

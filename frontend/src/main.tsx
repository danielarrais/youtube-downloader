import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './App'
import { LanguageProvider } from './contexts/LanguageContext'
import './styles/globals.css'

const container = document.getElementById('root');
const root = createRoot(container!);

root.render(
    <React.StrictMode>
        <LanguageProvider>
            <App />
        </LanguageProvider>
    </React.StrictMode>
)

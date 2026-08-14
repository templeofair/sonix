import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import { AuthProvider } from './features/auth'
import { CaptureDraftProvider } from './features/documents/hooks/CaptureDraftContext'
import App from './App'
import './index.css'

/** Vite `base` (subpath deploy). React Router wants no trailing slash; omit when app is at `/`. */
function routerBasename(): string | undefined {
  const raw = import.meta.env.BASE_URL ?? '/'
  const trimmed = raw.replace(/\/$/, '')
  return trimmed === '' ? undefined : trimmed
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter basename={routerBasename()}>
      <AuthProvider>
        <CaptureDraftProvider>
          <App />
        </CaptureDraftProvider>
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>,
)

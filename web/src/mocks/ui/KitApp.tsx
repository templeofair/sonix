import { Routes, Route, Link, useSearchParams } from 'react-router-dom'
import { catalog } from './catalog'
import CatalogPage, { StateSwitcher } from './pages/CatalogPage'
import PrimitivesMount from './mounts/PrimitivesMount'
import DocumentCardsMount from './mounts/DocumentCardsMount'
import LoginMount from './mounts/LoginMount'
import LibraryMount, { type LibraryMockState } from './mounts/LibraryMount'
import DetailStatesMount, { type DetailMockState } from './mounts/DetailStatesMount'
import SettingsMount from './mounts/SettingsMount'
import ScanHubMount from './mounts/ScanHubMount'
import CameraPlaceholderMount from './mounts/CameraPlaceholderMount'

function CardsRoute() {
  const [sp] = useSearchParams()
  const layout = sp.get('layout') === 'list' ? 'list' : 'grid'
  return (
    <>
      <StateSwitcher states={['grid', 'list']} param="layout" />
      <DocumentCardsMount layout={layout} />
    </>
  )
}

function LibraryRoute() {
  const [sp] = useSearchParams()
  const raw = sp.get('state') || 'populated'
  const state = (['populated', 'empty', 'loading'].includes(raw) ? raw : 'populated') as LibraryMockState
  return (
    <>
      <StateSwitcher states={['populated', 'empty', 'loading']} />
      <LibraryMount state={state} />
    </>
  )
}

function DetailRoute() {
  const [sp] = useSearchParams()
  const raw = sp.get('state') || 'ready'
  const allowed = ['pending', 'processing', 'failed', 'partial', 'ready']
  const state = (allowed.includes(raw) ? raw : 'ready') as DetailMockState
  return (
    <>
      <StateSwitcher states={allowed} />
      <DetailStatesMount state={state} />
    </>
  )
}

/** Secondary component kit (not the full-app mock). */
export default function KitApp() {
  return (
    <div className="min-h-screen bg-surface text-gray-800">
      <header className="h-14 flex items-center justify-between gap-3 px-4 border-b border-border bg-card">
        <Link to="/__ui" className="font-semibold text-gray-900 tracking-tight text-sm">
          ← Full app mock
        </Link>
        <span className="text-xs text-muted">Component kit</span>
      </header>
      <div className="md:flex">
        <nav aria-label="Kit catalog" className="md:w-48 border-b md:border-b-0 md:border-r border-border p-2 space-y-0.5 bg-card">
          {catalog.map((entry) => (
            <Link
              key={entry.id}
              to={`/__ui/_kit/${entry.path}`}
              className="control flex w-full px-3 py-2 rounded-btn text-sm text-left hover:bg-surface"
            >
              {entry.title}
            </Link>
          ))}
        </nav>
        <main className="flex-1 p-4 sm:p-6">
          <Routes>
            <Route index element={<CatalogPage />} />
            <Route path="primitives" element={<PrimitivesMount />} />
            <Route path="cards" element={<CardsRoute />} />
            <Route path="login" element={<LoginMount />} />
            <Route path="library" element={<LibraryRoute />} />
            <Route path="detail" element={<DetailRoute />} />
            <Route path="settings" element={<SettingsMount />} />
            <Route path="scan" element={<ScanHubMount />} />
            <Route path="camera" element={<CameraPlaceholderMount />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}

import { useState, lazy, Suspense } from 'react'
import { Routes, Route, Navigate, useParams, useSearchParams } from 'react-router-dom'
import { useAuth } from './features/auth'
import Layout from './components/Layout'
import ServerUnreachable from './components/ServerUnreachable'
import Login from './pages/Login'
import MyLetters from './pages/MyLetters'
import Add from './pages/Add'
import AddCamera from './pages/AddCamera'
import AddUpload from './pages/AddUpload'
import PageReview from './pages/PageReview'
import Explore from './pages/Explore'
import ExploreFolder from './pages/ExploreFolder'
import DocumentDetail from './pages/DocumentDetail'
import Settings from './pages/Settings'
import {
  pendingRedirectSearch,
  searchRedirectSearch,
} from './features/documents/lib/legacyLibraryParams'

const MockApp = import.meta.env.DEV
  ? lazy(() => import('./mocks/ui/MockApp'))
  : null

function Protected({ children }: { children: React.ReactNode }) {
  const { user, loading, serverUnreachable, retryConnection } = useAuth()
  const [retrying, setRetrying] = useState(false)

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface px-4">
        <p className="text-sm text-muted rounded-card border border-dashed border-border bg-card/80 px-8 py-6 shadow-card">
          Loading…
        </p>
      </div>
    )
  }
  if (serverUnreachable) {
    return (
      <ServerUnreachable
        busy={retrying}
        onRetry={() => {
          setRetrying(true)
          void retryConnection().finally(() => setRetrying(false))
        }}
      />
    )
  }
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

/** Legacy `/search` → flat library with search focused, preserving query params. */
function SearchRedirect() {
  const [sp] = useSearchParams()
  return <Navigate to={searchRedirectSearch(sp)} replace />
}

/** Legacy `/pending` → flat library queue filter, preserving query params. */
function PendingRedirect() {
  const [sp] = useSearchParams()
  return <Navigate to={pendingRedirectSearch(sp)} replace />
}

/** Legacy `/year/:year` (import year) → the Explore folder for that letter-date year. */
function YearRedirect() {
  const { year } = useParams()
  return <Navigate to={year ? `/explore/${year}` : '/explore'} replace />
}

function LoginRoute() {
  const { loading, serverUnreachable, retryConnection } = useAuth()
  const [retrying, setRetrying] = useState(false)

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface px-4">
        <p className="text-sm text-muted rounded-card border border-dashed border-border bg-card/80 px-8 py-6 shadow-card">
          Loading…
        </p>
      </div>
    )
  }
  if (serverUnreachable) {
    return (
      <ServerUnreachable
        busy={retrying}
        onRetry={() => {
          setRetrying(true)
          void retryConnection().finally(() => setRetrying(false))
        }}
      />
    )
  }
  return <Login />
}

export default function App() {
  return (
    <Routes>
      {MockApp ? (
        <Route
          path="/__ui/*"
          element={
            <Suspense
              fallback={
                <div className="min-h-screen flex items-center justify-center bg-surface text-sm text-muted">
                  Loading UI mocks…
                </div>
              }
            >
              <MockApp />
            </Suspense>
          }
        />
      ) : null}
      <Route path="/login" element={<LoginRoute />} />
      <Route path="/" element={<Protected><Layout /></Protected>}>
        <Route index element={<MyLetters />} />
        <Route path="add" element={<Add />} />
        <Route path="add/camera" element={<AddCamera />} />
        <Route path="add/review" element={<PageReview />} />
        <Route path="add/upload" element={<AddUpload />} />
        <Route path="explore" element={<Explore />} />
        <Route path="explore/no-date" element={<ExploreFolder undated />} />
        <Route path="explore/:year" element={<ExploreFolder />} />
        <Route path="year/:year" element={<YearRedirect />} />
        <Route path="documents/:id" element={<DocumentDetail />} />
        <Route path="search" element={<SearchRedirect />} />
        <Route path="pending" element={<PendingRedirect />} />
        <Route path="settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

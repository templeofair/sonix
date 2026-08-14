import { useEffect } from 'react'
import { Routes, Route, Navigate, Link } from 'react-router-dom'
import Layout from '../../components/Layout'
import MyLetters from '../../pages/MyLetters'
import Add from '../../pages/Add'
import AddCamera from '../../pages/AddCamera'
import AddUpload from '../../pages/AddUpload'
import PageReview from '../../pages/PageReview'
import Explore from '../../pages/Explore'
import ExploreFolder from '../../pages/ExploreFolder'
import DocumentDetail from '../../pages/DocumentDetail'
import Settings from '../../pages/Settings'
import { LoginForm } from '../../features/auth'
import { AppNavProvider } from '../../lib/appNav'
import { installApiMock, uninstallApiMock } from '../../lib/apiMock'
import { createMockApiHandler } from './fixtures/mockApi'
import { MockAuthProvider } from './providers/MockAuthProvider'
import KitApp from './KitApp'

function MockBanner() {
  return (
    <div className="bg-warning-soft border-b border-warning-border/80 text-warning text-xs sm:text-sm px-3 py-2 text-center z-[100] relative">
      UI mock — fake data only; actions do not touch the real server.{' '}
      <Link to="/__ui/_kit" className="underline font-medium hover:opacity-90">
        Component kit
      </Link>
      {' · '}
      <a href="/" className="underline font-medium hover:opacity-90">
        Product app
      </a>
    </div>
  )
}

function MockShellLayout() {
  return (
    <div className="min-h-screen flex flex-col">
      <MockBanner />
      <div className="flex-1 min-h-0 flex flex-col">
        <Layout />
      </div>
    </div>
  )
}

/**
 * Full-app Sonix mirror under `/__ui` — real pages + fixture API.
 * Catalog/kit lives at `/__ui/_kit`.
 */
export default function MockApp() {
  useEffect(() => {
    installApiMock(createMockApiHandler())
    return () => uninstallApiMock()
  }, [])

  return (
    <AppNavProvider prefix="/__ui">
      <MockAuthProvider state="signedIn">
        <Routes>
          <Route path="_kit/*" element={<KitApp />} />
          <Route
            path="login"
            element={
              <div>
                <MockBanner />
                <MockAuthProvider state="signedOut">
                  <LoginForm />
                </MockAuthProvider>
              </div>
            }
          />
          <Route element={<MockShellLayout />}>
            <Route index element={<MyLetters />} />
            <Route path="add" element={<Add />} />
            <Route path="add/camera" element={<AddCamera />} />
            <Route path="add/review" element={<PageReview />} />
            <Route path="add/upload" element={<AddUpload />} />
            <Route path="explore" element={<Explore />} />
            <Route path="explore/no-date" element={<ExploreFolder undated />} />
            <Route path="explore/:year" element={<ExploreFolder />} />
            <Route path="documents/:id" element={<DocumentDetail />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route path="*" element={<Navigate to="/__ui" replace />} />
        </Routes>
      </MockAuthProvider>
    </AppNavProvider>
  )
}

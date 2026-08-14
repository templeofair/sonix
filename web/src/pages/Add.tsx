import { Link } from 'react-router-dom'
import PageHeader from '../components/PageHeader'
import { useAppNav } from '../lib/appNav'

export default function Add() {
  const { appPath } = useAppNav()
  return (
    <>
      <PageHeader title="Add document" subtitle="Scan with your camera or upload files" />
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 sm:py-8 w-full">
          <p className="text-xs font-semibold text-muted uppercase tracking-wider mb-3">Choose a source</p>
          <div className="grid gap-4">
            <Link
              to={appPath('/add/camera')}
              className="flex items-center gap-4 p-5 rounded-card border border-border bg-card shadow-card hover:border-accent hover:bg-accent/5 transition-all group focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
            >
              <span
                className="w-14 h-14 rounded-card bg-accent text-white flex items-center justify-center text-2xl shadow-sm group-hover:scale-[1.02] transition-transform"
                aria-hidden
              >
                📷
              </span>
              <div className="text-left min-w-0">
                <span className="font-semibold text-gray-900 block">Use camera to scan</span>
                <span className="text-sm text-muted mt-0.5">Capture pages with your device camera</span>
              </div>
            </Link>
            <Link
              to={appPath('/add/upload')}
              className="flex items-center gap-4 p-5 rounded-card border border-border bg-card shadow-card hover:border-accent hover:bg-accent/5 transition-all group focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
            >
              <span
                className="w-14 h-14 rounded-card bg-surface border border-border flex items-center justify-center text-2xl text-gray-700 shadow-sm group-hover:scale-[1.02] transition-transform"
                aria-hidden
              >
                📁
              </span>
              <div className="text-left min-w-0">
                <span className="font-semibold text-gray-900 block">Upload from file</span>
                <span className="text-sm text-muted mt-0.5">Images or PDF</span>
              </div>
            </Link>
          </div>
        </div>
      </div>
    </>
  )
}

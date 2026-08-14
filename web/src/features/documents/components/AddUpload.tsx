import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import PageHeader from '../../../components/PageHeader'
import { NavBackHistoryButton } from '../../../components/NavBackControl'
import { useCaptureDraft } from '../hooks/CaptureDraftContext'
import { useCreateAndUpload } from '../hooks/useCreateAndUpload'
import Banner from '../../../shared/components/Banner'
import Button from '../../../shared/components/Button'
import { useAppNav } from '../../../lib/appNav'

export default function AddUpload() {
  const navigate = useNavigate()
  const { appPath } = useAppNav()
  const draft = useCaptureDraft()
  const [pdfFiles, setPdfFiles] = useState<File[]>([])
  const [error, setError] = useState<string | null>(null)
  const { uploading, progress, createAndUpload } = useCreateAndUpload()

  const onInput = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const list = e.target.files ? Array.from(e.target.files) : []
    e.target.value = ''
    setError(null)
    const images = list.filter((f) => f.type.startsWith('image/'))
    const pdfs = list.filter((f) => f.type === 'application/pdf')
    if (images.length > 0) {
      const keepTitle = draft.title
      draft.clear()
      draft.setTitle(keepTitle)
      await draft.addFiles(images)
    }
    if (pdfs.length > 0) {
      setPdfFiles((prev) => [...prev, ...pdfs])
    }
    if (images.length > 0 && pdfs.length === 0) {
      navigate(appPath('/add/review'))
    }
  }

  const removePdf = (i: number) => setPdfFiles((prev) => prev.filter((_, j) => j !== i))

  const uploadPdfs = async () => {
    if (pdfFiles.length === 0) return
    setError(null)
    try {
      const id = await createAndUpload(draft.title.trim() || undefined, pdfFiles)
      draft.clear()
      setPdfFiles([])
      navigate(appPath(`/documents/${id}`), { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    }
  }

  const goReviewExisting = () => {
    if (draft.pages.length === 0) return
    navigate(appPath('/add/review'))
  }

  return (
    <>
      <PageHeader
        title="Upload from file"
        subtitle="Images open the review screen; PDFs upload directly"
        left={<NavBackHistoryButton fallbackTo={appPath('/add')} />}
      />
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 w-full">
          <section className="rounded-card border border-border bg-card shadow-card p-4 sm:p-5 mb-6">
            <label className="block text-xs font-semibold text-muted uppercase tracking-wider mb-2">
              Document name (optional)
            </label>
            <input
              type="text"
              value={draft.title}
              onChange={(e) => draft.setTitle(e.target.value)}
              placeholder="e.g. Invoice 2024, Lease agreement…"
              className="w-full px-3 py-2 border border-border rounded-btn text-base md:text-sm text-gray-900 bg-white focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
            />
          </section>

          <label className="block w-full py-10 px-4 rounded-card border-2 border-dashed border-border bg-card/80 text-center text-muted hover:border-accent/50 hover:bg-accent/5 cursor-pointer shadow-card transition-colors">
            <input type="file" multiple accept="image/*,application/pdf" className="hidden" onChange={(e) => void onInput(e)} />
            <span className="text-sm font-medium text-gray-800">Choose files</span>
            <span className="block text-xs text-muted mt-1">Images → review · PDF → upload here</span>
          </label>

          {draft.pages.length > 0 && (
            <div className="mt-6 space-y-3">
              <p className="text-sm text-gray-800">
                {draft.pages.length} image{draft.pages.length !== 1 ? 's' : ''} ready for review
              </p>
              <Button type="button" onClick={goReviewExisting} className="w-full min-h-[44px] py-3">
                Review pages
              </Button>
            </div>
          )}

          {pdfFiles.length > 0 && (
            <div className="mt-6 space-y-4">
              <p className="text-xs font-semibold text-muted uppercase tracking-wider">
                PDFs selected ({pdfFiles.length})
              </p>
              <ul className="rounded-card border border-border bg-card shadow-card divide-y divide-border">
                {pdfFiles.map((f, i) => (
                  <li key={`${f.name}-${i}`} className="flex items-center justify-between gap-3 px-4 py-3">
                    <span className="text-sm text-gray-800 truncate min-w-0">{f.name}</span>
                    <button
                      type="button"
                      onClick={() => removePdf(i)}
                      className="control min-h-[44px] px-2 text-sm font-medium text-red-600 hover:text-red-700 flex-shrink-0"
                    >
                      Remove
                    </button>
                  </li>
                ))}
              </ul>
              <Button
                type="button"
                onClick={() => void uploadPdfs()}
                disabled={uploading}
                className="w-full min-h-[44px] py-3"
              >
                {uploading && progress
                  ? `Uploading ${Math.min(progress.current + 1, progress.total)} of ${progress.total}…`
                  : uploading
                    ? 'Uploading…'
                    : 'Upload PDFs'}
              </Button>
            </div>
          )}

        {error ? (
          <Banner tone="error" className="mt-4">
            {error}
          </Banner>
        ) : null}
        </div>
      </div>
    </>
  )
}

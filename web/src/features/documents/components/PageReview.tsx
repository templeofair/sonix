import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCaptureDraft } from '../hooks/CaptureDraftContext'
import { useCreateAndUpload } from '../hooks/useCreateAndUpload'
import { blobWithRotation, draftPagesToFiles } from '../lib/captureDraft'
import PageCropEditor from './PageCropEditor'
import ColourModeSheet from './ColourModeSheet'
import Modal from '../../../shared/components/Modal'
import Banner from '../../../shared/components/Banner'
import Button from '../../../shared/components/Button'
import { useAppNav } from '../../../lib/appNav'

const LONG_PRESS_MS = 420

const iconBtnClass =
  'inline-flex items-center justify-center min-h-[44px] min-w-[44px] flex-1 rounded-btn border border-border bg-white text-gray-800 shadow-sm hover:bg-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35 disabled:opacity-50'

function IconCrop() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M6 2v4H2M18 22v-4h4M6 6h10a2 2 0 012 2v10M18 18H8a2 2 0 01-2-2V6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconColour() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="2" />
      <path d="M12 3v18" stroke="currentColor" strokeWidth="2" />
      <path d="M12 3a9 9 0 010 18" fill="currentColor" opacity="0.35" />
    </svg>
  )
}

function IconAddPage() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M12 5v14M5 12h14"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

/** Circular retry arrow — retake this page. */
function IconRetake() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M21 12a9 9 0 11-3.2-6.75"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M21 3v6h-6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconTrash() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

export default function PageReview() {
  const navigate = useNavigate()
  const { appPath } = useAppNav()
  const draft = useCaptureDraft()
  const { uploading, progress, createAndUpload } = useCreateAndUpload()
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState(0)
  const [cropUrl, setCropUrl] = useState<string | null>(null)
  const [colourOpen, setColourOpen] = useState(false)
  const [discardConfirmOpen, setDiscardConfirmOpen] = useState(false)
  const cropUrlRef = useRef<string | null>(null)
  const dragFrom = useRef<number | null>(null)
  const longPressTimer = useRef<number | null>(null)
  const dragging = useRef(false)

  useEffect(() => {
    if (draft.pages.length === 0) {
      navigate(appPath('/add/camera'), { replace: true, state: { resumeDraft: true } })
    }
  }, [draft.pages.length, navigate, appPath])

  useEffect(() => {
    if (selected >= draft.pages.length) {
      setSelected(Math.max(0, draft.pages.length - 1))
    }
  }, [draft.pages.length, selected])

  const clearLongPress = () => {
    if (longPressTimer.current != null) {
      window.clearTimeout(longPressTimer.current)
      longPressTimer.current = null
    }
  }

  const requestAbandon = () => {
    if (uploading) return
    setDiscardConfirmOpen(true)
  }

  const confirmAbandon = () => {
    setDiscardConfirmOpen(false)
    draft.clear()
    navigate(appPath('/add'))
  }

  const save = async () => {
    if (draft.pages.length === 0 || uploading) return
    setError(null)
    try {
      const files = await draftPagesToFiles(draft.pages)
      const id = await createAndUpload(draft.title.trim() || undefined, files)
      draft.clear()
      navigate(appPath(`/documents/${id}`), { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    }
  }

  const deleteSelected = () => {
    if (draft.pages.length === 0) return
    draft.removeAt(selected)
  }

  const retakeSelected = () => {
    draft.setRetakeIndex(selected)
    navigate(appPath('/add/camera'), { state: { resumeDraft: true } })
  }

  const addMore = () => {
    draft.setRetakeIndex(null)
    navigate(appPath('/add/camera'), { state: { resumeDraft: true } })
  }

  const closeCrop = () => {
    if (cropUrlRef.current) {
      URL.revokeObjectURL(cropUrlRef.current)
      cropUrlRef.current = null
    }
    setCropUrl(null)
  }

  const openCrop = async () => {
    const page = draft.pages[selected]
    if (!page || uploading) return
    setError(null)
    try {
      const oriented = await blobWithRotation(page.blob, page.rotation)
      const url = URL.createObjectURL(oriented)
      if (cropUrlRef.current) URL.revokeObjectURL(cropUrlRef.current)
      cropUrlRef.current = url
      setCropUrl(url)
    } catch {
      setError('Could not open crop editor.')
    }
  }

  const onCropConfirm = (blob: Blob) => {
    draft.replaceAt(selected, blob, draft.pages[selected]?.source ?? 'camera')
    closeCrop()
  }

  const openColour = () => {
    if (uploading || !draft.pages[selected]) return
    setError(null)
    setColourOpen(true)
  }

  const onColourApply = (blob: Blob) => {
    draft.replaceAt(selected, blob, draft.pages[selected]?.source ?? 'camera')
    setColourOpen(false)
  }

  const onColourApplyAll = (blobs: Blob[]) => {
    draft.replaceAll(blobs)
    setColourOpen(false)
  }

  const onPointerDown = (index: number) => (e: React.PointerEvent) => {
    if (e.button !== 0) return
    dragging.current = false
    dragFrom.current = index
    clearLongPress()
    longPressTimer.current = window.setTimeout(() => {
      dragging.current = true
      setSelected(index)
    }, LONG_PRESS_MS)
  }

  const onPointerUp = () => {
    clearLongPress()
    dragFrom.current = null
    dragging.current = false
  }

  const onPointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!dragging.current || dragFrom.current == null) return
      const from = dragFrom.current
      const el = document.elementFromPoint(e.clientX, e.clientY)
      const li = el?.closest('[data-page-index]') as HTMLElement | null
      if (!li) return
      const to = Number(li.dataset.pageIndex)
      if (Number.isNaN(to) || to === from) return
      draft.move(from, to)
      dragFrom.current = to
      setSelected(to)
    },
    [draft]
  )

  if (draft.pages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-sm text-muted">
        Opening camera…
      </div>
    )
  }

  const progressLabel =
    uploading && progress
      ? progress.current >= progress.total
        ? `Uploaded ${progress.total} page${progress.total !== 1 ? 's' : ''}…`
        : `Uploading page ${progress.current + 1} of ${progress.total}…`
      : null

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-surface">
      {cropUrl ? (
        <PageCropEditor imageUrl={cropUrl} onCancel={closeCrop} onConfirm={onCropConfirm} />
      ) : null}
      {colourOpen && draft.pages[selected] ? (
        <ColourModeSheet
          page={draft.pages[selected]}
          pageCount={draft.pages.length}
          allPages={draft.pages}
          onCancel={() => setColourOpen(false)}
          onApply={onColourApply}
          onApplyAll={onColourApplyAll}
        />
      ) : null}
      <header className="flex-shrink-0 border-b border-border bg-card px-4 py-3 flex items-center gap-3">
        <button
          type="button"
          onClick={requestAbandon}
          disabled={uploading}
          className="control min-h-[44px] px-3 border border-border rounded-btn text-sm font-medium text-gray-800 bg-white shadow-sm hover:bg-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35 disabled:opacity-50"
        >
          Cancel
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold text-gray-900 truncate">Review pages</h1>
          <p className="text-xs text-muted truncate">
            Crop or adjust colour · {draft.pages.length} page{draft.pages.length !== 1 ? 's' : ''}
          </p>
        </div>
      </header>

      {discardConfirmOpen ? (
        <Modal
          onClose={() => setDiscardConfirmOpen(false)}
          labelledBy="discard-confirm-title"
          overlayClassName="z-[120] p-4"
          dismiss="mousedown"
          panelClassName="max-w-sm w-full p-5 space-y-4"
        >
          <h2 id="discard-confirm-title" className="text-lg font-semibold text-gray-900">
            Discard this scan?
          </h2>
          <p className="text-sm text-muted leading-relaxed">
            Your pages will be deleted and cannot be recovered.
          </p>
          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={() => setDiscardConfirmOpen(false)}
              className="control min-h-[44px] px-4 border border-border rounded-btn text-sm font-medium text-gray-800 bg-white hover:bg-surface"
            >
              Keep editing
            </button>
            <Button variant="danger" type="button" onClick={confirmAbandon} className="min-h-[44px] px-4 text-sm">
              Discard
            </Button>
          </div>
        </Modal>
      ) : null}

      <div className="flex-1 overflow-y-auto min-h-0 px-4 py-4 space-y-4 pb-[calc(7.5rem+env(safe-area-inset-bottom,0px))] md:pb-8">
        <label className="block">
          <span className="text-xs font-semibold text-muted uppercase tracking-wider">Document name</span>
          <input
            type="text"
            value={draft.title}
            onChange={(e) => draft.setTitle(e.target.value)}
            placeholder="Document name…"
            className="mt-2 w-full px-3 py-2.5 border border-border rounded-btn text-base md:text-sm bg-white text-gray-900 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
          />
        </label>

        <ul
          className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3"
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerCancel={onPointerUp}
        >
          {draft.pages.map((page, i) => {
            const isSelected = i === selected
            return (
              <li
                key={page.id}
                data-page-index={i}
                className={`relative rounded-card border bg-card shadow-card overflow-hidden touch-manipulation ${
                  isSelected ? 'border-accent ring-2 ring-accent/30' : 'border-border'
                }`}
              >
                <button
                  type="button"
                  onClick={() => setSelected(i)}
                  onPointerDown={onPointerDown(i)}
                  className="block w-full aspect-[3/4] bg-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40"
                  aria-label={`Page ${i + 1}${isSelected ? ', selected' : ''}`}
                  aria-pressed={isSelected}
                >
                  <img
                    src={page.url}
                    alt={`Page ${i + 1}`}
                    draggable={false}
                    className="w-full h-full object-cover pointer-events-none"
                    style={{ transform: `rotate(${page.rotation}deg)` }}
                  />
                </button>
                <span className="absolute top-2 left-2 min-w-[1.75rem] h-7 px-1.5 rounded-btn bg-black/65 text-white text-xs font-medium flex items-center justify-center tabular-nums">
                  {i + 1}
                </span>
                <div className="p-2 flex items-center justify-between gap-1 border-t border-border bg-card">
                  <button
                    type="button"
                    onClick={() => draft.moveLeft(i)}
                    disabled={i === 0}
                    className="control min-h-[44px] min-w-[44px] rounded-btn text-sm font-medium text-gray-800 hover:bg-surface disabled:opacity-30"
                    aria-label={`Move page ${i + 1} left`}
                  >
                    ←
                  </button>
                  <button
                    type="button"
                    onClick={() => draft.rotateAt(i)}
                    className="control min-h-[44px] min-w-[44px] rounded-btn text-sm font-medium text-gray-800 hover:bg-surface"
                    aria-label={`Rotate page ${i + 1}`}
                  >
                    ⟳
                  </button>
                  <button
                    type="button"
                    onClick={() => draft.moveRight(i)}
                    disabled={i >= draft.pages.length - 1}
                    className="control min-h-[44px] min-w-[44px] rounded-btn text-sm font-medium text-gray-800 hover:bg-surface disabled:opacity-30"
                    aria-label={`Move page ${i + 1} right`}
                  >
                    →
                  </button>
                </div>
              </li>
            )
          })}
        </ul>

        {error ? <Banner tone="error">{error}</Banner> : null}

        {progressLabel ? (
          <div className="space-y-2" role="status" aria-live="polite">
            <p className="text-sm text-gray-800">{progressLabel}</p>
            {progress && progress.total > 0 ? (
              <div className="h-2 rounded-full bg-border overflow-hidden">
                <div
                  className="h-full bg-accent transition-[width] duration-200"
                  style={{ width: `${Math.min(100, (progress.current / progress.total) * 100)}%` }}
                />
              </div>
            ) : null}
          </div>
        ) : null}
      </div>

      <div className="fixed bottom-[calc(4rem+env(safe-area-inset-bottom,0px))] md:bottom-0 left-0 right-0 md:left-56 z-20 border-t border-border bg-card/95 backdrop-blur-md px-3 py-2.5 shadow-[0_-4px_12px_-2px_rgb(0_0_0/0.06)]">
        <div className="max-w-3xl mx-auto flex items-center gap-2">
          <div className="flex flex-1 items-center gap-1.5 min-w-0" role="toolbar" aria-label="Page actions">
            <button
              type="button"
              onClick={() => void openCrop()}
              disabled={uploading}
              className={iconBtnClass}
              aria-label="Crop"
              title="Crop"
            >
              <IconCrop />
            </button>
            <button
              type="button"
              onClick={openColour}
              disabled={uploading}
              className={iconBtnClass}
              aria-label="Colour"
              title="Colour"
            >
              <IconColour />
            </button>
            <button
              type="button"
              onClick={addMore}
              disabled={uploading}
              className={iconBtnClass}
              aria-label="Add more"
              title="Add more"
            >
              <IconAddPage />
            </button>
            <button
              type="button"
              onClick={retakeSelected}
              disabled={uploading}
              className={iconBtnClass}
              aria-label="Retake"
              title="Retake"
            >
              <IconRetake />
            </button>
            <button
              type="button"
              onClick={deleteSelected}
              disabled={uploading}
              className={`${iconBtnClass} text-red-700 border-red-200 hover:bg-red-50`}
              aria-label="Delete"
              title="Delete"
            >
              <IconTrash />
            </button>
          </div>
          <button
            type="button"
            onClick={() => void save()}
            disabled={uploading || draft.pages.length === 0}
            className="control min-h-[44px] shrink-0 px-5 rounded-btn text-sm font-medium bg-accent text-white shadow-sm hover:opacity-95 disabled:opacity-50"
          >
            {uploading ? 'Uploading…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}

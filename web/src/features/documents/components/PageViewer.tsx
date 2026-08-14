import { useEffect, useState } from 'react'
import type { MouseEvent } from 'react'

type PageMeta = { page_index: number; content_type: string }

type Props = {
  pages: PageMeta[]
  pageIndex: number
  onPageIndexChange: (index: number) => void
  pageImageUrl: (index: number) => string
  /** Used for share/download filename stem */
  documentTitle?: string
  documentId: number
  /** Clockwise rotate of the current page (rewrites stored image). */
  onRotatePage?: (pageIndex: number, degrees: 90 | 180 | 270) => Promise<void>
}

async function shareOrDownloadPage(imageUrl: string, filename: string): Promise<void> {
  try {
    const res = await fetch(imageUrl, { credentials: 'include' })
    if (!res.ok) throw new Error(`fetch ${res.status}`)
    const blob = await res.blob()
    const type = blob.type || 'image/jpeg'
    const file = new File([blob], filename, { type })
    const nav = navigator as Navigator & {
      canShare?: (data?: ShareData) => boolean
      share?: (data?: ShareData) => Promise<void>
    }
    if (typeof nav.canShare === 'function' && nav.canShare({ files: [file] }) && nav.share) {
      await nav.share({ files: [file], title: filename })
      return
    }
    const objectUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = objectUrl
    a.download = filename
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(objectUrl)
  } catch {
    const a = document.createElement('a')
    a.href = imageUrl
    a.download = filename
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
  }
}

function IconFullscreen({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M8 21H5a2 2 0 0 1-2-2v-3M16 21h3a2 2 0 0 0 2-2v-3"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconShare({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M4 12v7a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-7M16 6l-4-4-4 4M12 2v14"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconRotateCW({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M21 12a9 9 0 1 1-2.64-6.36M21 3v6h-6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

/** Thumbnail rail + large page image, fullscreen zoom, share/download. */
export default function PageViewer({
  pages,
  pageIndex,
  onPageIndexChange,
  pageImageUrl,
  documentTitle,
  documentId,
  onRotatePage,
}: Props) {
  const [fullscreen, setFullscreen] = useState(false)
  const [sharing, setSharing] = useState(false)
  const [rotating, setRotating] = useState(false)
  const imageUrl = pages.length > 0 ? pageImageUrl(pageIndex) : null
  const pageLabel = `${pageIndex + 1} of ${pages.length}`
  const stem = (documentTitle?.trim() || `document-${documentId}`).replace(/[^\w.-]+/g, '_')
  const filename = `${stem}-page-${pageIndex + 1}.jpg`

  useEffect(() => {
    if (!fullscreen) return
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') setFullscreen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [fullscreen])

  if (pages.length === 0 || !imageUrl) return null

  const onShare = async () => {
    setSharing(true)
    try {
      await shareOrDownloadPage(imageUrl, filename)
    } finally {
      setSharing(false)
    }
  }

  const onRotate = async () => {
    if (!onRotatePage || rotating) return
    setRotating(true)
    try {
      await onRotatePage(pageIndex, 90)
    } finally {
      setRotating(false)
    }
  }

  const iconBtn =
    'control inline-flex items-center justify-center min-h-[44px] min-w-[44px] rounded-btn border border-border text-gray-800 bg-white hover:bg-surface disabled:opacity-50'

  return (
    <>
      <section className="rounded-card border border-border bg-card p-3 sm:p-5 shadow-card min-w-0" aria-label="Pages">
        <div className="flex items-center gap-2 mb-3 min-w-0">
          <h2 className="text-xs font-semibold text-muted uppercase tracking-wider flex-shrink-0">Pages</h2>
          <span className="text-sm tabular-nums text-gray-800 font-medium flex-1 min-w-0" aria-live="polite">
            {pageLabel}
          </span>
          <div className="flex items-center gap-1.5 flex-shrink-0">
            {onRotatePage ? (
              <button
                type="button"
                onClick={() => void onRotate()}
                disabled={rotating}
                className={iconBtn}
                aria-label={rotating ? 'Rotating page' : 'Rotate page clockwise'}
                title="Rotate clockwise"
              >
                <IconRotateCW />
              </button>
            ) : null}
            <button
              type="button"
              onClick={() => setFullscreen(true)}
              className={iconBtn}
              aria-label="Full screen"
              title="Full screen"
            >
              <IconFullscreen />
            </button>
            <button
              type="button"
              onClick={() => void onShare()}
              disabled={sharing}
              className={iconBtn}
              aria-label={sharing ? 'Preparing share' : 'Share or download'}
              title="Share or download"
            >
              <IconShare />
            </button>
          </div>
        </div>

        {/* Vertical thumb rail (scrollable) + preview — same pattern at all widths */}
        <div className="flex gap-2.5 sm:gap-3 items-stretch min-w-0">
          <div
            className="flex flex-col gap-1.5 flex-shrink-0 overflow-y-auto overscroll-contain max-h-[min(50vh,28rem)] sm:max-h-[min(70vh,36rem)] pr-0.5 -ml-0.5 pl-0.5 scroll-smooth"
            aria-label="Page thumbnails"
          >
            {pages.map((_: PageMeta, i: number) => (
              <button
                key={i}
                type="button"
                onClick={() => onPageIndexChange(i)}
                aria-pressed={pageIndex === i}
                aria-label={`Page ${i + 1}`}
                className={`control w-11 h-[3.25rem] sm:w-12 sm:h-14 rounded-btn border overflow-hidden flex-shrink-0 transition-shadow ${
                  pageIndex === i
                    ? 'border-accent ring-2 ring-accent/40 shadow-sm'
                    : 'border-border hover:border-accent/30'
                }`}
              >
                <img src={pageImageUrl(i)} alt="" className="w-full h-full object-cover" />
              </button>
            ))}
          </div>

          <div className="flex-1 min-w-0">
            <button
              type="button"
              onClick={() => setFullscreen(true)}
              className="block w-full rounded-card border border-border bg-surface overflow-hidden shadow-inner focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
              aria-label={`Open page ${pageIndex + 1} full screen`}
            >
              <img
                src={imageUrl}
                alt={`Page ${pageIndex + 1}`}
                className="w-full h-auto min-h-[40vh] max-h-[50vh] sm:min-h-[50vh] sm:max-h-[70vh] object-contain bg-surface"
              />
            </button>
          </div>
        </div>
      </section>

      {fullscreen ? (
        <div
          className="fixed inset-0 z-50 bg-black flex flex-col"
          role="dialog"
          aria-modal="true"
          aria-label={`Page ${pageIndex + 1} full screen`}
          onClick={(e: MouseEvent<HTMLDivElement>) => {
            if (e.target === e.currentTarget) setFullscreen(false)
          }}
        >
          <div className="flex items-center justify-between gap-2 px-3 py-2 bg-black/80 text-white flex-shrink-0">
            <span className="text-sm tabular-nums font-medium">{pageLabel}</span>
            <button
              type="button"
              onClick={() => setFullscreen(false)}
              className="control min-h-[44px] min-w-[44px] px-3 rounded-btn text-sm font-medium bg-white/15 hover:bg-white/25"
              aria-label="Close full screen"
            >
              Close
            </button>
          </div>
          <div className="flex-1 min-h-0 overflow-auto overscroll-contain touch-pan-x touch-pan-y">
            <img
              src={imageUrl}
              alt={`Page ${pageIndex + 1}`}
              className="w-full h-auto max-w-none block mx-auto"
            />
          </div>
        </div>
      ) : null}
    </>
  )
}

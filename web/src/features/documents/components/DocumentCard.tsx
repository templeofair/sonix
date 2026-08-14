import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { DocumentListItem } from '../types/document'
import { documentsApi } from '../services/documentsApi'
import { documentStatusPillClass } from '../lib/documentStatusStyle'
import { formatCardDate } from '../lib/relativeTime'
import type { LibraryLayout } from '../lib/libraryParams'
import Modal from '../../../shared/components/Modal'
import { useAppNav } from '../../../lib/appNav'

type Props = {
  doc: DocumentListItem
  layout?: LibraryLayout
  /** Recent strip: thumbnail + title only. */
  dense?: boolean
  /** My letters select mode: tap toggles selection instead of opening the letter. */
  selectionMode?: boolean
  selected?: boolean
  onToggleSelect?: () => void
}

const LONG_PRESS_MS = 500
const thumbImgClass = 'h-full w-full object-cover object-top'

function ThumbMedia({ src }: { src: string | null }) {
  if (src) {
    return <img src={src} alt="" loading="lazy" className={thumbImgClass} />
  }
  return (
    <span className="flex h-full w-full items-center justify-center text-xs text-muted-subtle px-1 text-center">
      No preview
    </span>
  )
}

/** Shared library card: portrait thumb, title, badges. Whole card opens the document (delete on detail). */
export default function DocumentCard({
  doc,
  layout = 'list',
  dense = false,
  selectionMode = false,
  selected = false,
  onToggleSelect,
}: Props) {
  const navigate = useNavigate()
  const { appPath } = useAppNav()
  const title = doc.title || `Document ${doc.id}`
  const dateLabel = formatCardDate({ document_date: doc.document_date, created_at: doc.created_at })
  const pageLabel =
    doc.page_count > 0 ? `${doc.page_count} page${doc.page_count !== 1 ? 's' : ''}` : null
  const thumbSrc =
    doc.thumbnail_available && doc.page_count > 0 ? documentsApi.pageThumbnailUrl(doc.id, 0) : null
  const previewTitleId = `doc-card-preview-${doc.id}`

  const [previewOpen, setPreviewOpen] = useState(false)
  const timerRef = useRef<number | null>(null)
  const suppressClickRef = useRef(false)

  const clearTimer = () => {
    if (timerRef.current != null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  useEffect(() => () => clearTimer(), [])

  const openPreview = () => {
    if (!thumbSrc || selectionMode) return
    clearTimer()
    setPreviewOpen(true)
  }

  const onThumbPointerDown = (e: ReactPointerEvent) => {
    if (selectionMode || !thumbSrc || e.button !== 0) return
    clearTimer()
    timerRef.current = window.setTimeout(() => {
      timerRef.current = null
      suppressClickRef.current = true
      setPreviewOpen(true)
    }, LONG_PRESS_MS)
  }

  const onThumbPointerEnd = () => clearTimer()

  const onThumbClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (selectionMode) {
      onToggleSelect?.()
      return
    }
    if (suppressClickRef.current) {
      suppressClickRef.current = false
      return
    }
    navigate(appPath(`/documents/${doc.id}`))
  }

  const onCardActivate = () => {
    if (selectionMode) onToggleSelect?.()
  }

  const selectCheckbox = selectionMode ? (
    <span
      className="absolute top-2 left-2 z-[3] flex h-11 w-11 items-center justify-center rounded-btn bg-card/95 border border-border shadow-sm pointer-events-none"
      aria-hidden
    >
      <input
        type="checkbox"
        checked={selected}
        readOnly
        tabIndex={-1}
        className="h-4 w-4 accent-[var(--color-accent)]"
      />
    </span>
  ) : null

  const selectedRing = selectionMode && selected ? 'border-accent ring-2 ring-accent/25' : ''

  const thumbInteractive = (
    <span
      data-testid="doc-card-thumb"
      className="relative block h-full w-full pointer-events-auto touch-manipulation"
      onPointerDown={onThumbPointerDown}
      onPointerUp={onThumbPointerEnd}
      onPointerCancel={onThumbPointerEnd}
      onPointerLeave={onThumbPointerEnd}
      onClick={onThumbClick}
      onContextMenu={(e) => {
        if (thumbSrc && !selectionMode) e.preventDefault()
      }}
    >
      <ThumbMedia src={thumbSrc} />
      {!selectionMode && thumbSrc ? (
        <button
          type="button"
          className="control absolute bottom-1 right-1 z-[2] min-h-[44px] min-w-[44px] px-2 rounded-btn border border-border bg-card/95 text-xs font-medium text-gray-800 opacity-0 focus:opacity-100 group-hover:opacity-100 group-focus-within:opacity-100 shadow-sm"
          aria-label="Preview first page"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            clearTimer()
            openPreview()
          }}
          onPointerDown={(e) => e.stopPropagation()}
        >
          View
        </button>
      ) : null}
    </span>
  )

  const previewModal =
    !selectionMode && previewOpen && thumbSrc ? (
      <Modal
        onClose={() => setPreviewOpen(false)}
        labelledBy={previewTitleId}
        panelClassName="max-w-[min(96vw,42rem)] p-3 sm:p-4 space-y-3"
        overlayClassName="z-[60] p-4"
      >
        <div className="flex items-start justify-between gap-3">
          <h2 id={previewTitleId} className="text-sm font-semibold text-gray-900 line-clamp-2 min-w-0">
            {title}
          </h2>
          <button
            type="button"
            className="control min-h-[44px] px-3 rounded-btn border border-border text-sm font-medium flex-shrink-0"
            onClick={() => setPreviewOpen(false)}
          >
            Close
          </button>
        </div>
        <img
          src={documentsApi.pageImageUrl(doc.id, 0)}
          alt=""
          className="mx-auto max-h-[min(85vh,900px)] w-auto max-w-full object-contain"
        />
      </Modal>
    ) : null

  if (layout === 'grid') {
    return (
      <article
        className={`group relative z-0 rounded-card border border-border bg-card shadow-card hover:shadow-md hover:border-accent/30 transition-all overflow-hidden h-full flex flex-col ${selectedRing}`}
      >
        {selectionMode ? (
          <button
            type="button"
            className="absolute inset-0 z-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40"
            aria-label={selected ? `Deselect ${title}` : `Select ${title}`}
            aria-pressed={selected}
            onClick={onCardActivate}
          />
        ) : (
          <Link
            to={appPath(`/documents/${doc.id}`)}
            className="absolute inset-0 z-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40"
            aria-label={title}
          />
        )}
        {selectCheckbox}
        <div className="relative z-[1] pointer-events-none flex flex-col flex-1 min-h-0">
          <span className="block w-full aspect-[3/4] flex-shrink-0 overflow-hidden bg-surface border-b border-border">
            {thumbInteractive}
          </span>
          <div className="p-2">
            <span className="block font-medium text-sm text-gray-900 line-clamp-2 leading-snug">{title}</span>
            {!dense ? (
              <div className="mt-1 flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
                <span
                  className={`inline-flex text-[10px] font-medium px-1.5 py-0.5 rounded-full border ${documentStatusPillClass(doc.status)}`}
                >
                  {doc.status}
                </span>
                {dateLabel ? <span className="text-[11px] text-muted tabular-nums">{dateLabel}</span> : null}
                {pageLabel ? <span className="text-[11px] text-muted">{pageLabel}</span> : null}
              </div>
            ) : null}
          </div>
        </div>
        {previewModal}
      </article>
    )
  }

  return (
    <article
      className={`group relative z-0 rounded-card border border-border bg-card shadow-card hover:shadow-md hover:border-accent/30 transition-all p-2.5 sm:p-3 ${selectedRing}`}
    >
      {selectionMode ? (
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-card focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40"
          aria-label={selected ? `Deselect ${title}` : `Select ${title}`}
          aria-pressed={selected}
          onClick={onCardActivate}
        />
      ) : (
        <Link
          to={appPath(`/documents/${doc.id}`)}
          className="absolute inset-0 z-0 rounded-card focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40"
          aria-label={title}
        />
      )}
      {selectCheckbox}
      <div className="relative z-[1] flex gap-2.5 items-start pointer-events-none">
        <span className="flex-shrink-0 block w-12 sm:w-14 overflow-hidden rounded-btn border border-border bg-surface">
          <span className="block aspect-[3/4] w-full overflow-hidden bg-surface">{thumbInteractive}</span>
        </span>

        <div className="flex-1 min-w-0">
          <span className="font-medium text-gray-900 line-clamp-2 text-sm sm:text-base leading-snug">
            {title}
          </span>
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-1">
            <span
              className={`inline-flex text-xs font-medium px-2 py-0.5 rounded-full border ${documentStatusPillClass(doc.status)}`}
            >
              {doc.status}
            </span>
            {dateLabel ? (
              <span className="text-xs sm:text-sm text-muted tabular-nums">{dateLabel}</span>
            ) : null}
            {pageLabel ? <span className="text-xs sm:text-sm text-muted">{pageLabel}</span> : null}
          </div>
        </div>
      </div>
      {previewModal}
    </article>
  )
}

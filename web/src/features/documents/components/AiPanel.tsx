import { useEffect, useState } from 'react'
import type { DocumentDetail } from '../types/document'
import { documentsApi } from '../services/documentsApi'
import { summarizeExtractionError } from '../lib/extractionError'
import MarkdownText from '../../../components/MarkdownText'
import Modal from '../../../shared/components/Modal'
import Spinner from '../../../shared/components/Spinner'
import Button from '../../../shared/components/Button'
import SectionLabel from '../../../shared/components/SectionLabel'
import ExtractModeSelect, {
  type ExtractMode,
} from '../../../shared/components/ExtractModeSelect'

export function extractionProgressLabel(doc: DocumentDetail): string {
  const t = doc.extraction_pages_total
  if (t == null || t < 1) return 'Extraction in progress…'
  const d = doc.extraction_pages_done ?? 0
  if (d < t) {
    return d === 0
      ? `Starting extraction… (${t} page${t !== 1 ? 's' : ''})`
      : `Pages ${d} / ${t} complete — extracting next…`
  }
  return `All ${t} page${t !== 1 ? 's' : ''} scanned — summarizing and translating…`
}

function engineMetaLine(doc: DocumentDetail): string | null {
  const ex = doc.extraction
  if (!ex?.prompt_version && !ex?.engine_id) return null
  const parts: string[] = []
  if (ex.engine_id) parts.push(ex.engine_id)
  if (ex.prompt_version) parts.push(ex.prompt_version)
  if (ex.extraction_wall_ms != null && ex.extraction_wall_ms >= 0) {
    parts.push(
      ex.extraction_wall_ms >= 1000
        ? `last run ${(ex.extraction_wall_ms / 1000).toFixed(1)} s`
        : `last run ${ex.extraction_wall_ms} ms`
    )
  }
  return parts.join(' · ')
}

async function copyText(value: string): Promise<boolean> {
  const text = value.trim()
  if (!text) return false
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* fall through */
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    ta.remove()
    return ok
  } catch {
    return false
  }
}

function CopyButton({ getText, label = 'Copy' }: { getText: () => string; label?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      type="button"
      className="control min-h-[44px] px-3 text-sm font-medium text-accent hover:bg-accent/10 rounded-btn"
      onClick={() => {
        void copyText(getText()).then((ok) => {
          if (!ok) return
          setCopied(true)
          window.setTimeout(() => setCopied(false), 1500)
        })
      }}
    >
      {copied ? 'Copied' : label}
    </button>
  )
}

function TextModal({
  lang,
  docId,
  onClose,
}: {
  lang: 'original' | 'english'
  docId: number
  onClose: () => void
}) {
  const [text, setText] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setText('')
    documentsApi
      .text(docId, lang)
      .then((t) => {
        if (!cancelled) setText(t)
      })
      .catch(() => {
        if (!cancelled) setText('Failed to load text.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [docId, lang])

  const title = lang === 'original' ? 'Original text' : 'Translation (English)'

  return (
    <Modal
      onClose={onClose}
      labelledBy="text-modal-title"
      overlayClassName="z-50 p-0 md:p-4"
      panelClassName="!rounded-none md:!rounded-card h-[100dvh] w-full max-w-none md:h-auto md:max-h-[85vh] md:max-w-2xl flex flex-col"
    >
      <div className="flex items-center justify-between gap-2 px-6 py-4 border-b border-border flex-shrink-0 bg-card">
        <h2 id="text-modal-title" className="text-lg font-semibold text-gray-900 min-w-0 truncate">
          {title}
        </h2>
        <div className="flex items-center gap-1 flex-shrink-0">
          {!loading && text ? <CopyButton getText={() => text} /> : null}
          <button
            type="button"
            onClick={onClose}
            className="control min-h-[44px] min-w-[44px] flex items-center justify-center rounded-btn text-muted hover:text-gray-900 hover:bg-surface transition-colors"
            aria-label="Close dialog"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto px-6 py-4 min-h-0">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner className="w-5 h-5" label="Loading text…" />
          </div>
        ) : text ? (
          <MarkdownText text={text} />
        ) : (
          <p className="text-muted text-sm">No text available.</p>
        )}
      </div>
      <div className="px-6 py-3 border-t border-border flex-shrink-0 bg-surface/80">
        <button
          type="button"
          onClick={onClose}
          className="control min-h-[44px] px-5 py-2 bg-white border border-border text-gray-800 rounded-btn text-sm font-medium hover:bg-surface transition-colors"
        >
          Close
        </button>
      </div>
    </Modal>
  )
}

function modeFromBool(useOcr: boolean): ExtractMode {
  return useOcr ? 'ocr' : 'llm'
}

type Props = {
  doc: DocumentDetail
  extracting: boolean
  savingDate: boolean
  documentDateEdit: string
  onDocumentDateEdit: (value: string) => void
  onSaveDocumentDate: () => void
  useOcr: boolean
  onUseOcrChange: (value: boolean) => void
  onExtract: (useOcr?: boolean) => void
  onResetExtraction: () => void
}

/** Status-driven extraction / AI panel (pending → processing → failed → ready). */
export default function AiPanel({
  doc,
  extracting,
  savingDate,
  documentDateEdit,
  onDocumentDateEdit,
  onSaveDocumentDate,
  useOcr,
  onUseOcrChange,
  onExtract,
  onResetExtraction,
}: Props) {
  const [textModal, setTextModal] = useState<'original' | 'english' | null>(null)
  const hasExtraction = !!doc.extraction
  const summary = doc.extraction?.summary?.trim() ?? ''
  const summaryEmpty = !summary
  const documentDateEmpty = !documentDateEdit.trim()
  const extractionIncomplete = hasExtraction && (summaryEmpty || documentDateEmpty)
  const engineLine = engineMetaLine(doc)
  const hasPages = doc.pages.length > 0
  const showFullText = doc.status === 'ready' || hasExtraction || hasPages
  const failedError = summarizeExtractionError(doc.extraction_error)
  const mode = modeFromBool(useOcr)
  const setMode = (next: ExtractMode) => onUseOcrChange(next === 'ocr')

  const actionBtn = (variant: 'accent' | 'ready' | 'cancel') =>
    `control w-full sm:w-auto min-h-[44px] px-5 py-2.5 rounded-btn text-sm font-medium shadow-sm disabled:opacity-50 ${
      variant === 'cancel'
        ? 'border border-blue-300/80 text-blue-900 bg-white hover:bg-blue-50'
        : variant === 'ready'
          ? 'border border-border text-gray-800 bg-white hover:bg-surface'
          : 'bg-accent text-white hover:opacity-95'
    }`

  return (
    <div className="space-y-4">
      {doc.status === 'pending' && hasPages ? (
        <section className="rounded-card border border-border bg-surface p-4 sm:p-5 shadow-card space-y-3">
          <h2 className="text-xs font-semibold text-muted uppercase tracking-wider">Extract</h2>
          <ExtractModeSelect
            id="extract-mode-pending"
            value={mode}
            onChange={setMode}
            disabled={extracting}
          />
          <button
            type="button"
            onClick={() => onExtract(useOcr)}
            disabled={extracting}
            className={actionBtn('accent')}
          >
            {extracting ? 'Starting…' : 'Extract now'}
          </button>
        </section>
      ) : null}

      {doc.status === 'failed' && hasPages ? (
        <section className="rounded-card border border-amber-200/80 bg-amber-50/90 p-4 sm:p-5 shadow-card space-y-3">
          <h2 className="text-xs font-semibold text-amber-900/80 uppercase tracking-wider">Extraction failed</h2>
          <p className="text-sm text-amber-900 leading-relaxed">{failedError.summary}</p>
          {failedError.detail ? (
            <details className="group rounded-btn border border-amber-200/80 bg-amber-50/50">
              <summary className="control cursor-pointer list-none min-h-[44px] flex items-center gap-2 px-3 py-2 text-sm font-medium text-amber-950 select-none [&::-webkit-details-marker]:hidden">
                <span aria-hidden className="text-amber-800/80 transition-transform group-open:rotate-90">
                  ▸
                </span>
                More details
              </summary>
              <pre className="px-3 pb-3 text-xs text-amber-950/90 whitespace-pre-wrap break-words font-mono leading-relaxed border-t border-amber-200/60 pt-2 overflow-x-auto">
                {failedError.detail}
              </pre>
            </details>
          ) : null}
          <ExtractModeSelect
            id="extract-mode-failed"
            value={mode}
            onChange={setMode}
            tone="amber"
            disabled={extracting}
          />
          <button
            type="button"
            onClick={() => onExtract(useOcr)}
            disabled={extracting}
            className={actionBtn('accent')}
          >
            {extracting ? 'Starting…' : 'Retry extraction'}
          </button>
        </section>
      ) : null}

      {doc.status === 'partial' && hasPages ? (
        <section className="rounded-card border border-orange-200/80 bg-orange-50/90 p-4 sm:p-5 shadow-card space-y-3">
          <h2 className="text-xs font-semibold text-orange-900/80 uppercase tracking-wider">
            Partially extracted
          </h2>
          <p className="text-sm text-orange-950 leading-relaxed">
            Original text was saved, but translation or summary failed. Retry to finish, or open the
            original below.
          </p>
          {failedError.detail ? (
            <details className="group rounded-btn border border-orange-200/80 bg-orange-50/50">
              <summary className="control cursor-pointer list-none min-h-[44px] flex items-center gap-2 px-3 py-2 text-sm font-medium text-orange-950 select-none [&::-webkit-details-marker]:hidden">
                <span aria-hidden className="text-orange-800/80 transition-transform group-open:rotate-90">
                  ▸
                </span>
                More details
              </summary>
              <pre className="px-3 pb-3 text-xs text-orange-950/90 whitespace-pre-wrap break-words font-mono leading-relaxed border-t border-orange-200/60 pt-2 overflow-x-auto">
                {failedError.detail}
              </pre>
            </details>
          ) : null}
          <ExtractModeSelect
            id="extract-mode-partial"
            value={mode}
            onChange={setMode}
            tone="amber"
            disabled={extracting}
          />
          <button
            type="button"
            onClick={() => onExtract(useOcr)}
            disabled={extracting}
            className={actionBtn('accent')}
          >
            {extracting ? 'Starting…' : 'Retry extraction'}
          </button>
        </section>
      ) : null}

      {doc.status === 'processing' ? (
        <div
          className="rounded-card border border-blue-200/80 bg-blue-50/95 p-4 sm:p-5 text-blue-900 text-sm shadow-card space-y-3"
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          <div className="flex items-center gap-2">
            <Spinner className="w-4 h-4 flex-shrink-0" status={false} />
            <span>{extractionProgressLabel(doc)}</span>
          </div>
          <button type="button" onClick={onResetExtraction} className={actionBtn('cancel')}>
            Cancel extraction
          </button>
        </div>
      ) : null}

      {hasExtraction ? (
        <section className="rounded-card border border-border bg-card p-4 sm:p-5 shadow-card space-y-3">
          <div className="flex items-center justify-between gap-2">
            <SectionLabel as="h2">Summary</SectionLabel>
            {summary ? <CopyButton getText={() => summary} /> : null}
          </div>
          <p className="text-gray-900 whitespace-pre-wrap text-sm leading-relaxed">
            {summary || 'No summary'}
          </p>
          {engineLine ? (
            <p className="text-[10px] uppercase tracking-wide text-muted-subtle font-medium">{engineLine}</p>
          ) : null}
        </section>
      ) : null}

      {hasExtraction ? (
        <section className="rounded-card border border-border bg-card p-4 sm:p-5 shadow-card space-y-3">
          <SectionLabel as="h2">Document date</SectionLabel>
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="date"
              value={documentDateEdit}
              onChange={(e) => onDocumentDateEdit(e.target.value)}
              onBlur={onSaveDocumentDate}
              className="px-3 py-2 border border-border rounded-btn text-base md:text-sm bg-white text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/30 focus-visible:border-accent"
            />
            <Button
              type="button"
              onClick={onSaveDocumentDate}
              disabled={savingDate}
              className="min-h-[44px] px-4 py-2 text-sm"
            >
              {savingDate ? 'Saving…' : 'Save'}
            </Button>
          </div>
        </section>
      ) : null}

      {extractionIncomplete && doc.status === 'ready' ? (
        <section className="rounded-card border border-sky-200/80 bg-sky-50/95 p-4 sm:p-5 text-sky-900 text-sm shadow-card">
          <p className="font-semibold text-sky-950">Extraction incomplete</p>
          <p className="mt-1">
            {summaryEmpty && documentDateEmpty
              ? 'Summary and document date are missing.'
              : summaryEmpty
                ? 'Summary is missing.'
                : 'Document date is missing.'}
          </p>
        </section>
      ) : null}

      {showFullText ? (
        <section className="rounded-card border border-border bg-card p-4 sm:p-5 shadow-card space-y-3">
          <h2 className="text-xs font-semibold text-muted uppercase tracking-wider">Full text</h2>
          <div className="flex flex-col sm:flex-row gap-3">
            <button
              type="button"
              onClick={() => setTextModal('english')}
              className="control flex-1 min-h-[44px] px-4 py-3 border border-border rounded-card text-sm font-medium text-gray-800 bg-white hover:bg-surface hover:border-accent/40 transition-colors text-center shadow-sm"
            >
              View translation
            </button>
            <button
              type="button"
              onClick={() => setTextModal('original')}
              className="control flex-1 min-h-[44px] px-4 py-3 border border-border rounded-card text-sm font-medium text-gray-800 bg-white hover:bg-surface hover:border-accent/40 transition-colors text-center shadow-sm"
            >
              View original text
            </button>
          </div>
        </section>
      ) : null}

      {doc.status === 'ready' && hasPages ? (
        <section className="rounded-card border border-border bg-card p-4 sm:p-5 shadow-card space-y-3">
          <h2 className="text-xs font-semibold text-muted uppercase tracking-wider">Re-process</h2>
          <ExtractModeSelect
            id="extract-mode-reprocess"
            value={mode}
            onChange={setMode}
            disabled={extracting}
          />
          <button
            type="button"
            onClick={() => onExtract(useOcr)}
            disabled={extracting}
            className={actionBtn('ready')}
          >
            {extracting ? 'Starting…' : 'Re-process document'}
          </button>
        </section>
      ) : null}

      {textModal ? (
        <TextModal lang={textModal} docId={doc.id} onClose={() => setTextModal(null)} />
      ) : null}
    </div>
  )
}

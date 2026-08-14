import { useEffect, useRef, useState } from 'react'
import type { ChangeEvent, KeyboardEvent } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useDocument } from '../hooks/useDocument'
import { useDocumentMutations } from '../hooks/useDocumentMutations'
import PageHeader from '../../../components/PageHeader'
import { NavBackHistoryButton } from '../../../components/NavBackControl'
import { documentStatusPillClass } from '../lib/documentStatusStyle'
import PageViewer from './PageViewer'
import AiPanel from './AiPanel'
import Modal from '../../../shared/components/Modal'
import Button from '../../../shared/components/Button'
import Skeleton from '../../../shared/components/Skeleton'
import { extractModeFromEngineId } from '../../../shared/components/ExtractModeSelect'
import { settingsApi } from '../../settings/services/settingsApi'
import { useAppNav } from '../../../lib/appNav'

function IconBack({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M15 18l-6-6 6-6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function IconTrash({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
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

const iconControlClass =
  'inline-flex items-center justify-center min-h-[44px] min-w-[44px] rounded-btn border border-border bg-white text-gray-800 shadow-sm hover:bg-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35 disabled:opacity-50'

export default function DocumentDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { appPath } = useAppNav()
  const { doc, setDoc, refresh, loading, loadError } = useDocument(id)
  const {
    extracting,
    deleting,
    savingTags,
    savingDate,
    savingTitle,
    deleteDocument,
    putTags,
    putDocumentDate,
    putTitle,
    startExtract,
    resetExtraction,
    pageImageUrl,
    rotatePage,
  } = useDocumentMutations(doc, setDoc, refresh, navigate)
  const [pageIndex, setPageIndex] = useState(0)
  const [tagInput, setTagInput] = useState('')
  const [documentDateEdit, setDocumentDateEdit] = useState('')
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleInput, setTitleInput] = useState('')
  const [titleConfirmOpen, setTitleConfirmOpen] = useState(false)
  const [reprocessUseOcr, setReprocessUseOcr] = useState(true)
  const titleInputRef = useRef<HTMLInputElement>(null)
  const ignoreBlurRef = useRef(false)
  const titleConfirmOpenRef = useRef(false)

  useEffect(() => {
    titleConfirmOpenRef.current = titleConfirmOpen
  }, [titleConfirmOpen])

  useEffect(() => {
    setDocumentDateEdit(doc?.extraction?.document_date ?? '')
  }, [doc?.extraction?.document_date])

  // Prefer the mode this letter last used; otherwise the Settings auto-import default.
  useEffect(() => {
    const fromDoc = extractModeFromEngineId(doc?.extraction?.engine_id)
    if (fromDoc) {
      setReprocessUseOcr(fromDoc === 'ocr')
      return
    }
    let cancelled = false
    settingsApi
      .get()
      .then((d) => {
        if (cancelled) return
        // Match backend: unset import_extract_use_ocr means OCR.
        setReprocessUseOcr(d.import_extract_use_ocr !== false)
      })
      .catch(() => {
        if (!cancelled) setReprocessUseOcr(true)
      })
    return () => {
      cancelled = true
    }
  }, [doc?.id, doc?.extraction?.engine_id])

  const tags = doc?.extraction?.tags ?? []
  const addTag = () => {
    const t = tagInput.trim()
    if (!t || !doc) return
    putTags([...tags, t])
    setTagInput('')
  }
  const removeTag = (index: number) => {
    putTags(tags.filter((_: string, i: number) => i !== index))
  }

  const saveDocumentDate = () => {
    if (!doc) return
    putDocumentDate(documentDateEdit.trim() || null)
  }

  if (!doc) {
    if (loadError) {
      return (
        <div className="flex-1 flex items-center justify-center p-6">
          <div className="rounded-card border border-border bg-card px-8 py-10 shadow-card text-center space-y-3">
            <p className="text-muted text-sm">Could not load this document.</p>
            <Button type="button" onClick={() => refresh()} className="inline-flex items-center justify-center px-4 py-2.5 text-sm">
              Try again
            </Button>
          </div>
        </div>
      )
    }
    if (loading) {
      return (
        <div className="flex-1 flex items-center justify-center p-6" role="status" aria-label="Loading document">
          <Skeleton className="h-40 max-w-md w-full" />
        </div>
      )
    }
    return (
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="rounded-card border border-border bg-card px-8 py-10 shadow-card text-muted text-sm">
          Document not found.
        </div>
      </div>
    )
  }

  const currentTitle = doc.title || ''
  const displayTitle = currentTitle || `Document ${doc.id}`

  const startEditTitle = () => {
    setTitleInput(currentTitle)
    setTitleConfirmOpen(false)
    setEditingTitle(true)
  }

  const discardTitleEdit = () => {
    setTitleInput(currentTitle)
    setTitleConfirmOpen(false)
    setEditingTitle(false)
  }

  const commitTitleEdit = () => {
    const next = titleInput.trim()
    if (next === currentTitle.trim()) {
      setEditingTitle(false)
      setTitleConfirmOpen(false)
      return Promise.resolve()
    }
    return putTitle(next).then(() => {
      setEditingTitle(false)
      setTitleConfirmOpen(false)
    })
  }

  /** Leave edit mode: unchanged → exit; changed → confirm dialog. */
  const finishTitleEdit = () => {
    if (titleConfirmOpen) return
    const next = titleInput.trim()
    if (next === currentTitle.trim()) {
      setEditingTitle(false)
      return
    }
    setTitleConfirmOpen(true)
  }

  const onTitleBlur = () => {
    if (ignoreBlurRef.current) return
    window.setTimeout(() => {
      if (ignoreBlurRef.current || titleConfirmOpenRef.current) return
      finishTitleEdit()
    }, 120)
  }

  const titleBlock = editingTitle ? (
    <div className="min-w-0 w-full">
      <input
        ref={titleInputRef}
        type="text"
        value={titleInput}
        onChange={(e: ChangeEvent<HTMLInputElement>) => setTitleInput(e.target.value)}
        onBlur={onTitleBlur}
        onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            ignoreBlurRef.current = true
            titleInputRef.current?.blur()
            finishTitleEdit()
            window.setTimeout(() => {
              ignoreBlurRef.current = false
            }, 150)
          }
          if (e.key === 'Escape') {
            e.preventDefault()
            ignoreBlurRef.current = true
            discardTitleEdit()
            window.setTimeout(() => {
              ignoreBlurRef.current = false
            }, 150)
          }
        }}
        autoFocus
        placeholder={`Document ${doc.id}`}
        aria-label="Document name"
        className="w-full min-w-0 px-2.5 py-2 border border-border rounded-btn text-base md:text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent bg-white"
      />
    </div>
  ) : (
    <div className="min-w-0 flex items-center gap-2">
      <button
        type="button"
        onClick={startEditTitle}
        title="Tap to rename"
        className="min-w-0 text-left text-base sm:text-lg font-semibold text-gray-900 truncate hover:text-accent transition-colors leading-tight focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35 rounded-btn"
      >
        {displayTitle}
      </button>
      <span
        className={`hidden md:inline-flex flex-shrink-0 text-xs font-medium px-2 py-0.5 rounded-full border ${documentStatusPillClass(doc.status)}`}
      >
        {doc.status}
      </span>
    </div>
  )

  return (
    <>
      <PageHeader
        left={
          <>
            <NavBackHistoryButton
              fallbackTo={appPath('/')}
              aria-label="Back"
              className={`md:hidden !px-0 ${iconControlClass}`}
            >
              <IconBack />
            </NavBackHistoryButton>
            <span className="hidden md:inline-flex">
              <NavBackHistoryButton fallbackTo={appPath('/')} />
            </span>
          </>
        }
        titleSlot={titleBlock}
        right={
          <>
            {!editingTitle ? (
              <button
                type="button"
                onClick={startEditTitle}
                className="control hidden md:inline-flex items-center px-3 py-2 text-sm font-medium text-accent hover:bg-accent/10 rounded-btn transition-colors"
              >
                Rename
              </button>
            ) : null}
            <span
              className={`md:hidden inline-flex flex-shrink-0 text-xs font-medium px-2 py-0.5 rounded-full border ${documentStatusPillClass(doc.status)}`}
            >
              {doc.status}
            </span>
            <button
              type="button"
              onClick={deleteDocument}
              disabled={deleting}
              aria-label="Delete"
              title="Delete"
              className={`md:hidden text-red-700 border-red-200 hover:bg-red-50 ${iconControlClass}`}
            >
              <IconTrash />
            </button>
            <button
              type="button"
              onClick={deleteDocument}
              disabled={deleting}
              className="control hidden md:inline-flex items-center px-3 py-2 border border-red-200 text-red-700 rounded-btn text-sm font-medium hover:bg-red-50 disabled:opacity-50 transition-colors"
            >
              Delete
            </button>
          </>
        }
      />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-8 py-4 sm:py-6 w-full">
          <div className="flex flex-col md:flex-row md:items-start gap-4 md:gap-6">
            <div className="w-full md:w-[55%] lg:w-[58%] flex-shrink-0 order-1">
              <PageViewer
                pages={doc.pages}
                pageIndex={pageIndex}
                onPageIndexChange={setPageIndex}
                pageImageUrl={pageImageUrl}
                documentTitle={doc.title}
                documentId={doc.id}
                onRotatePage={rotatePage}
              />
            </div>
            <div className="w-full md:flex-1 min-w-0 order-2 space-y-4">
              <AiPanel
                doc={doc}
                extracting={extracting}
                savingDate={savingDate}
                documentDateEdit={documentDateEdit}
                onDocumentDateEdit={setDocumentDateEdit}
                onSaveDocumentDate={saveDocumentDate}
                useOcr={reprocessUseOcr}
                onUseOcrChange={setReprocessUseOcr}
                onExtract={(useOcr) => startExtract(useOcr)}
                onResetExtraction={resetExtraction}
              />
              <section className="rounded-card border border-border bg-card p-4 sm:p-5 shadow-card space-y-3 min-w-0">
                <h2 className="text-xs font-semibold text-muted uppercase tracking-wider">Tags</h2>
                <div className="flex flex-wrap gap-2">
                  {tags.length === 0 && <span className="text-sm text-muted-subtle">No tags yet</span>}
                  {tags.map((t: string, i: number) => (
                    <span
                      key={i}
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full border border-accent/20 bg-accent/10 text-gray-800 text-sm"
                    >
                      {t}
                      <button
                        type="button"
                        onClick={() => removeTag(i)}
                        disabled={savingTags}
                        className="inline-flex items-center justify-center min-h-[28px] min-w-[28px] -mr-1 text-gray-500 hover:text-red-600 disabled:opacity-50 rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35"
                        aria-label={`Remove ${t}`}
                      >
                        ×
                      </button>
                    </span>
                  ))}
                </div>
                <div className="flex items-stretch gap-2 min-w-0">
                  <input
                    type="text"
                    value={tagInput}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setTagInput(e.target.value)}
                    onKeyDown={(e: KeyboardEvent<HTMLInputElement>) =>
                      e.key === 'Enter' && (e.preventDefault(), addTag())
                    }
                    placeholder="Add tag…"
                    className="flex-1 min-w-0 px-3 py-2 border border-border rounded-btn text-base md:text-sm bg-white text-gray-900 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
                  />
                  <button
                    type="button"
                    onClick={addTag}
                    disabled={savingTags || !tagInput.trim()}
                    className="control flex-shrink-0 self-stretch min-h-[44px] px-4 py-2 bg-accent text-white rounded-btn text-sm font-medium shadow-sm disabled:opacity-50"
                  >
                    Add
                  </button>
                </div>
              </section>
            </div>
          </div>
        </div>
      </div>

      {titleConfirmOpen ? (
        <Modal
          onClose={() => {
            ignoreBlurRef.current = true
            discardTitleEdit()
            window.setTimeout(() => {
              ignoreBlurRef.current = false
            }, 150)
          }}
          labelledBy="rename-confirm-title"
          describedBy="rename-confirm-desc"
          dismiss="none"
          onOverlayMouseDown={() => {
            ignoreBlurRef.current = true
          }}
          panelClassName="max-w-sm w-full p-5 space-y-4"
        >
          <h2 id="rename-confirm-title" className="text-lg font-semibold text-gray-900">
            Save new name?
          </h2>
          <p id="rename-confirm-desc" className="text-sm text-muted leading-relaxed break-words">
            {titleInput.trim()
              ? `Rename this document to “${titleInput.trim()}”?`
              : `Clear the name? It will show as “Document ${doc.id}”.`}
          </p>
          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={() => {
                discardTitleEdit()
                window.setTimeout(() => {
                  ignoreBlurRef.current = false
                }, 150)
              }}
              className="control min-h-[44px] px-4 border border-border rounded-btn text-sm font-medium text-gray-800 bg-white hover:bg-surface"
            >
              Don&apos;t save
            </button>
            <button
              type="button"
              disabled={savingTitle}
              onClick={() => {
                void commitTitleEdit().finally(() => {
                  window.setTimeout(() => {
                    ignoreBlurRef.current = false
                  }, 150)
                })
              }}
              className="control min-h-[44px] px-4 rounded-btn text-sm font-medium bg-accent text-white shadow-sm hover:opacity-95 disabled:opacity-50"
            >
              {savingTitle ? 'Saving…' : 'Save'}
            </button>
          </div>
        </Modal>
      ) : null}
    </>
  )
}

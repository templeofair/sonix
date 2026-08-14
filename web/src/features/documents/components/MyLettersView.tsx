import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import PageHeader from '../../../components/PageHeader'
import DocumentCard from './DocumentCard'
import LibraryToolbar, { documentListClassName } from './LibraryToolbar'
import { useLibrary } from '../hooks/useLibrary'
import { RECENT_LIMIT } from '../lib/libraryParams'
import { documentsApi } from '../services/documentsApi'
import Button from '../../../shared/components/Button'
import Banner from '../../../shared/components/Banner'
import EmptyState from '../../../shared/components/EmptyState'
import Modal from '../../../shared/components/Modal'
import { buttonVariantClass } from '../../../shared/components/Button'
import Skeleton from '../../../shared/components/Skeleton'
import { fieldInputClass } from '../../../shared/components/Field'
import { useAppNav } from '../../../lib/appNav'

/** Matches Tailwind `md` — phone chrome locks list scroll while More options is open. */
const PHONE_MQ = '(max-width: 767px)'

function IconSearch({ className = '' }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="2" />
      <path d="M20 20l-3.5-3.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}

function IconSelect({ className = '' }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <rect x="4" y="4" width="16" height="16" rx="2" stroke="currentColor" strokeWidth="2" />
      <path d="M8 12l2.5 2.5L16 9" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

/** Flat My letters library (search + filters + cards). Accordion retired in Phase 5b. */
export default function MyLettersView() {
  const lib = useLibrary()
  const { appPath } = useAppNav()
  const [selectMode, setSelectMode] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(() => new Set())
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [banner, setBanner] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)
  const [moreOptionsOpen, setMoreOptionsOpen] = useState(false)
  const [phoneViewport, setPhoneViewport] = useState(
    () => typeof window !== 'undefined' && window.matchMedia(PHONE_MQ).matches,
  )

  const filtersActive = Boolean(lib.status || lib.tag || lib.year)
  const datesActive = Boolean(lib.fromInput || lib.toInput)
  const selectedCount = selected.size
  const moreActive = filtersActive || lib.sort !== 'created_desc' || lib.layout !== 'grid'
  const lockListScroll = moreOptionsOpen && phoneViewport

  useEffect(() => {
    const mq = window.matchMedia(PHONE_MQ)
    const sync = () => setPhoneViewport(mq.matches)
    sync()
    mq.addEventListener('change', sync)
    return () => mq.removeEventListener('change', sync)
  }, [])

  useEffect(() => {
    if (selectMode) setMoreOptionsOpen(false)
  }, [selectMode])

  const exitSelectMode = () => {
    setSelectMode(false)
    setSelected(new Set())
    setConfirmOpen(false)
  }

  const enterSelectMode = () => {
    setBanner(null)
    setSelectMode(true)
    setSelected(new Set())
  }

  const toggleSelect = (id: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const selectedIds = useMemo(() => Array.from(selected), [selected])

  const runBulkDelete = async () => {
    if (selectedIds.length === 0) return
    setDeleting(true)
    setBanner(null)
    let failed = 0
    const total = selectedIds.length
    for (const id of selectedIds) {
      try {
        await documentsApi.delete(id)
      } catch {
        failed += 1
      }
    }
    setDeleting(false)
    setConfirmOpen(false)
    exitSelectMode()
    await lib.reload()
    if (failed > 0) {
      setBanner({
        tone: 'error',
        text: `Deleted ${total - failed} of ${total}. ${failed} could not be deleted.`,
      })
    }
  }

  return (
    <>
      {/* Desktop only — on phone the title lives in the Layout top strip. */}
      <div className="hidden md:block">
        <PageHeader title="My letters" subtitle="Your most recent letters, plus search" />
      </div>
      <div
        className={`flex-1 ${lockListScroll ? 'overflow-hidden overscroll-none' : 'overflow-y-auto'}`}
      >
        <div className="max-w-6xl mx-auto px-3 sm:px-8 lg:px-10 py-3 sm:py-8 pb-24 md:pb-8 w-full space-y-3 sm:space-y-4">
          <div className="sticky top-0 z-20 -mx-1 px-1 pb-2 pt-1 bg-surface border-b border-border space-y-2 isolate">
            <div className="space-y-2 p-2.5 sm:p-4 rounded-card bg-card border border-border shadow-card">
              {selectMode ? (
                <div className="flex flex-row gap-2 items-center">
                  <p className="text-sm text-muted flex-1 min-w-0 truncate" aria-live="polite">
                    {selectedCount === 0 ? 'Tap letters to select' : `${selectedCount} selected`}
                  </p>
                  <Button
                    type="button"
                    variant="danger"
                    className="min-h-[44px] px-4 text-sm flex-shrink-0"
                    disabled={selectedCount === 0 || deleting}
                    onClick={() => setConfirmOpen(true)}
                  >
                    Delete
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={exitSelectMode}
                    className="min-h-[44px] px-4 flex-shrink-0"
                    disabled={deleting}
                  >
                    Done
                  </Button>
                </div>
              ) : (
                <>
                  {/* Row 1: search field + Search only. */}
                  <div className="flex flex-row gap-2 items-stretch">
                    <input
                      ref={lib.searchInputRef}
                      id="library-search-text"
                      type="search"
                      inputMode="search"
                      enterKeyHint="search"
                      aria-label="Search summary, content, and tags"
                      value={lib.textInput}
                      onChange={(e) => lib.setTextInput(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && lib.runSearch()}
                      placeholder="Search…"
                      className={`flex-1 min-w-0 ${fieldInputClass}`}
                    />
                    <Button
                      type="button"
                      onClick={lib.runSearch}
                      aria-label="Search"
                      title="Search"
                      className="min-h-[44px] min-w-[44px] px-3 flex-shrink-0 inline-flex items-center justify-center"
                    >
                      <IconSearch />
                    </Button>
                  </div>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={enterSelectMode}
                    aria-label="Select"
                    title="Select"
                    className="min-h-[44px] px-3 inline-flex items-center justify-center gap-2 w-full"
                    disabled={lib.loading || lib.docs.length === 0}
                  >
                    <IconSelect />
                    Select
                  </Button>
                  <details className="group">
                    <summary className="control cursor-pointer list-none min-h-[44px] inline-flex items-center gap-2 text-sm font-medium text-gray-800 hover:text-accent select-none [&::-webkit-details-marker]:hidden">
                      <span
                        aria-hidden
                        className="inline-block text-muted transition-transform group-open:rotate-90"
                      >
                        ▸
                      </span>
                      Date filters
                      {datesActive ? (
                        <span className="text-xs font-normal text-accent">(active)</span>
                      ) : null}
                    </summary>
                    <div className="mt-2 grid grid-cols-1 sm:grid-cols-2 gap-3">
                      <div>
                        <label htmlFor="library-date-from" className="block text-sm font-medium text-gray-900 mb-1.5">
                          Document date from
                        </label>
                        <input
                          id="library-date-from"
                          type="date"
                          value={lib.fromInput}
                          onChange={(e) => lib.setFromInput(e.target.value)}
                          className="w-full px-3 py-2.5 border border-border rounded-btn bg-white text-base md:text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
                        />
                      </div>
                      <div>
                        <label htmlFor="library-date-to" className="block text-sm font-medium text-gray-900 mb-1.5">
                          Document date to
                        </label>
                        <input
                          id="library-date-to"
                          type="date"
                          value={lib.toInput}
                          onChange={(e) => lib.setToInput(e.target.value)}
                          className="w-full px-3 py-2.5 border border-border rounded-btn bg-white text-base md:text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
                        />
                      </div>
                    </div>
                  </details>
                  <details
                    className="group"
                    onToggle={(e) => setMoreOptionsOpen(e.currentTarget.open)}
                  >
                    <summary className="control cursor-pointer list-none min-h-[44px] inline-flex items-center gap-2 text-sm font-medium text-gray-800 hover:text-accent select-none [&::-webkit-details-marker]:hidden">
                      <span
                        aria-hidden
                        className="inline-block text-muted transition-transform group-open:rotate-90"
                      >
                        ▸
                      </span>
                      More options
                      {moreActive ? (
                        <span className="text-xs font-normal text-accent">(active)</span>
                      ) : null}
                    </summary>
                    <div className="mt-2 max-h-[min(55dvh,22rem)] overflow-y-auto overscroll-contain md:max-h-none md:overflow-visible">
                      <LibraryToolbar
                        layout={lib.layout}
                        sort={lib.sort}
                        statusValues={lib.statusValues}
                        tagValues={lib.tagValues}
                        yearValues={lib.yearValues}
                        tagOptions={lib.tags}
                        yearOptions={lib.years}
                        onLayoutChange={lib.setLayout}
                        onSortChange={lib.setSort}
                        onStatusChange={lib.setStatusFilter}
                        onTagChange={lib.setTagFilter}
                        onYearChange={lib.setYearFilter}
                      />
                    </div>
                  </details>
                </>
              )}
            </div>
          </div>

          {banner ? (
            <div aria-live="polite">
              <Banner tone={banner.tone}>{banner.text}</Banner>
            </div>
          ) : null}

          {lib.unreachable ? (
            <div className="rounded-card border border-amber-200/80 bg-amber-50/95 py-10 px-6 text-center shadow-card space-y-3">
              <p className="font-medium text-amber-950">Cannot reach Sonix</p>
              <Button type="button" onClick={() => lib.reload()} className="inline-flex items-center justify-center px-4 py-2.5 text-sm">
                Try again
              </Button>
            </div>
          ) : lib.loading ? (
            <div className="space-y-3" role="status" aria-label="Loading documents">
              <Skeleton className="h-28 w-full" />
              <Skeleton className="h-28 w-full" />
              <Skeleton className="h-28 w-full" />
            </div>
          ) : lib.docs.length === 0 ? (
            lib.isDefaultView ? (
              <EmptyState
                title="Nothing scanned yet"
                className="bg-card/80 py-12 shadow-card"
                action={
                  <Link to={appPath('/add')} className={`${buttonVariantClass('primary')} inline-flex items-center justify-center px-4 py-2.5 text-sm`}>
                    Scan letters
                  </Link>
                }
              >
                <p className="text-sm text-muted-subtle">Your scanned letters appear here.</p>
              </EmptyState>
            ) : (
              <EmptyState title="No letters match this search" className="bg-card/80 py-12 shadow-card">
                <p className="text-sm text-muted-subtle">Try another word, or clear the filters.</p>
              </EmptyState>
            )
          ) : (
            <div className="space-y-3">
              <ul className={documentListClassName(lib.layout)}>
                {lib.docs.map((doc) => (
                  <li key={doc.id}>
                    <DocumentCard
                      doc={doc}
                      layout={lib.layout}
                      selectionMode={selectMode}
                      selected={selected.has(doc.id)}
                      onToggleSelect={() => toggleSelect(doc.id)}
                    />
                  </li>
                ))}
              </ul>
              {lib.hasMore ? (
                <button
                  type="button"
                  onClick={lib.loadMore}
                  disabled={lib.loadingMore}
                  className="control w-full min-h-[44px] border border-border rounded-btn text-sm font-medium text-gray-800 bg-card hover:bg-gray-50 disabled:opacity-50"
                >
                  {lib.loadingMore ? 'Loading…' : 'Load more'}
                </button>
              ) : null}
              {lib.isDefaultView ? (
                <p className="text-sm text-muted text-center">
                  Your {RECENT_LIMIT} most recent letters. Search above, or open{' '}
                  <Link to={appPath('/explore')} className="text-accent underline underline-offset-2">
                    Explore
                  </Link>{' '}
                  for older ones.
                </p>
              ) : null}
            </div>
          )}
        </div>
      </div>

      {confirmOpen ? (
        <Modal
          onClose={() => !deleting && setConfirmOpen(false)}
          labelledBy="bulk-delete-title"
          dismiss={deleting ? 'none' : 'click'}
          panelClassName="max-w-sm w-full p-5 space-y-4"
        >
          <h2 id="bulk-delete-title" className="text-lg font-semibold text-gray-900">
            Delete {selectedCount} letter{selectedCount === 1 ? '' : 's'}?
          </h2>
          <p className="text-sm text-muted leading-relaxed">This cannot be undone.</p>
          <div className="flex gap-2 justify-end">
            <Button
              type="button"
              variant="secondary"
              className="min-h-[44px] px-4 text-sm"
              disabled={deleting}
              onClick={() => setConfirmOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="danger"
              className="min-h-[44px] px-4 text-sm"
              disabled={deleting}
              onClick={() => void runBulkDelete()}
            >
              {deleting ? 'Deleting…' : 'Delete'}
            </Button>
          </div>
        </Modal>
      ) : null}
    </>
  )
}

import { useParams } from 'react-router-dom'
import PageHeader from '../../../components/PageHeader'
import { NavBackHistoryButton } from '../../../components/NavBackControl'
import { useFolderDocuments } from '../hooks/useFolderDocuments'
import DocumentCard from './DocumentCard'
import { documentListClassName } from './LibraryToolbar'
import EmptyState from '../../../shared/components/EmptyState'
import Skeleton from '../../../shared/components/Skeleton'
import { useAppNav } from '../../../lib/appNav'

const toggleClass =
  'control inline-flex items-center px-3 rounded-btn border text-sm font-medium whitespace-nowrap'

/** One Explore folder: a letter-date year, or the undated bucket. */
export default function ExploreFolderView({ undated = false }: { undated?: boolean }) {
  const { year } = useParams()
  const { appPath } = useAppNav()
  const {
    docs,
    loading,
    loadingMore,
    hasMore,
    layout,
    oldestFirst,
    setLayout,
    setOldestFirst,
    loadMore,
  } = useFolderDocuments(year, undated)

  const folderName = undated ? 'No date' : (year ?? 'Explore')
  const subtitle = undated ? 'Letters with no date on them' : `Letters dated in ${folderName}`

  return (
    <>
      {/* Desktop only — on phone the folder name stays in the Layout top strip. */}
      <div className="hidden md:block">
        <PageHeader
          title={folderName}
          subtitle={subtitle}
          left={<NavBackHistoryButton fallbackTo={appPath('/explore')} />}
        />
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-3 sm:px-8 lg:px-10 py-3 sm:py-8 pb-24 md:pb-8 w-full space-y-3">
          <div className="sticky top-0 z-10 -mx-1 px-1 py-1 bg-surface flex items-center gap-2">
            <span className="md:hidden">
              <NavBackHistoryButton fallbackTo={appPath('/explore')} aria-label="Back to Explore" />
            </span>
            {!undated ? (
              <button
                type="button"
                aria-pressed={oldestFirst}
                onClick={() => setOldestFirst(!oldestFirst)}
                className={`${toggleClass} ${
                  oldestFirst
                    ? 'border-accent bg-accent/10 text-accent ring-1 ring-accent/40'
                    : 'border-border bg-card text-gray-800 hover:bg-gray-50'
                }`}
              >
                Oldest first
              </button>
            ) : null}
            <div
              className="ml-auto hidden md:inline-flex rounded-btn border border-border overflow-hidden"
              role="group"
              aria-label="Layout"
            >
              <button
                type="button"
                aria-pressed={layout === 'grid'}
                onClick={() => setLayout('grid')}
                className={`control px-3 text-sm font-medium ${
                  layout === 'grid' ? 'bg-accent text-white' : 'bg-card text-gray-800 hover:bg-gray-50'
                }`}
              >
                Grid
              </button>
              <button
                type="button"
                aria-pressed={layout === 'list'}
                onClick={() => setLayout('list')}
                className={`control px-3 text-sm font-medium border-l border-border ${
                  layout === 'list' ? 'bg-accent text-white' : 'bg-card text-gray-800 hover:bg-gray-50'
                }`}
              >
                List
              </button>
            </div>
          </div>

          {loading ? (
            <div className="space-y-3" role="status" aria-label="Loading documents">
              <Skeleton className="h-28 w-full" />
              <Skeleton className="h-28 w-full" />
            </div>
          ) : docs.length === 0 ? (
            <EmptyState
              title={undated ? 'No letters without a date' : 'No letters in this folder'}
              className="bg-card/80 py-12 shadow-card"
            />
          ) : (
            <>
              <ul className={documentListClassName(layout)}>
                {docs.map((doc) => (
                  <li key={doc.id}>
                    <DocumentCard doc={doc} layout={layout} />
                  </li>
                ))}
              </ul>
              {hasMore ? (
                <button
                  type="button"
                  onClick={loadMore}
                  disabled={loadingMore}
                  className="control w-full min-h-[44px] border border-border rounded-btn text-sm font-medium text-gray-800 bg-card hover:bg-gray-50 disabled:opacity-50"
                >
                  {loadingMore ? 'Loading…' : 'Load more'}
                </button>
              ) : null}
            </>
          )}
        </div>
      </div>
    </>
  )
}

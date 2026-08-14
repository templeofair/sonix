import { Link } from 'react-router-dom'
import PageHeader from '../../../components/PageHeader'
import { useExploreFolders } from '../hooks/useExploreFolders'
import Banner from '../../../shared/components/Banner'
import EmptyState from '../../../shared/components/EmptyState'
import Skeleton from '../../../shared/components/Skeleton'
import { buttonVariantClass } from '../../../shared/components/Button'
import { useAppNav } from '../../../lib/appNav'

function letterCountLabel(count: number): string {
  return `${count} ${count === 1 ? 'letter' : 'letters'}`
}

function FolderTile({
  to,
  name,
  count,
  numeric = false,
}: {
  to: string
  name: string
  count: number
  numeric?: boolean
}) {
  return (
    <li>
      <Link
        to={to}
        aria-label={`${name}, ${letterCountLabel(count)}`}
        className="control flex min-h-[72px] w-full flex-col justify-center rounded-card border border-border bg-card shadow-card px-3 py-3 hover:border-accent/30 hover:shadow-md transition-colors"
      >
        <span
          className={`text-xl font-semibold text-gray-900 leading-tight truncate${numeric ? ' tabular-nums' : ''}`}
        >
          {name}
        </span>
        <span className="text-sm text-muted">{letterCountLabel(count)}</span>
      </Link>
    </li>
  )
}

/** Explore folder index: one folder per letter-date year, plus No date. */
export default function ExploreView() {
  const { appPath } = useAppNav()
  const { years, undatedCount, loading, failed } = useExploreFolders()

  return (
    <>
      {/* Desktop only — on phone the title lives in the Layout top strip. */}
      <div className="hidden md:block">
        <PageHeader title="Explore" subtitle="Folders by the date on the letter" />
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-3 sm:px-8 lg:px-10 py-3 sm:py-8 pb-24 md:pb-8 w-full space-y-3">
          {failed ? <Banner tone="error">Could not load folders.</Banner> : null}
          {loading ? (
            <div
              className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3"
              role="status"
              aria-label="Loading folders"
            >
              <Skeleton className="h-[72px] w-full" />
              <Skeleton className="h-[72px] w-full" />
              <Skeleton className="h-[72px] w-full" />
              <Skeleton className="h-[72px] w-full" />
            </div>
          ) : years.length === 0 && undatedCount === 0 ? (
            <EmptyState
              title="Nothing scanned yet"
              className="bg-card/80 py-12 shadow-card"
              action={
                <Link
                  to={appPath('/add')}
                  className={`${buttonVariantClass('primary')} inline-flex items-center justify-center px-4 py-2.5 text-sm`}
                >
                  Scan letters
                </Link>
              }
            >
              <p className="text-sm text-muted-subtle">Folders appear once a letter has a date.</p>
            </EmptyState>
          ) : (
            <ul className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
              {years.map((y) => (
                <FolderTile
                  key={y.year}
                  to={appPath(`/explore/${y.year}`)}
                  name={y.year}
                  count={y.count}
                  numeric
                />
              ))}
              {undatedCount > 0 ? (
                <FolderTile
                  to={appPath('/explore/no-date')}
                  name="No date"
                  count={undatedCount}
                />
              ) : null}
            </ul>
          )}
        </div>
      </div>
    </>
  )
}

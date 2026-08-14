import { fixtureDocuments } from '../fixtures/documents'
import DocumentCard from '../../../features/documents/components/DocumentCard'
import { documentListClassName } from '../../../features/documents/components/LibraryToolbar'
import EmptyState from '../../../shared/components/EmptyState'
import Skeleton from '../../../shared/components/Skeleton'
import { buttonVariantClass } from '../../../shared/components/Button'
import { Link } from 'react-router-dom'
import SectionLabel from '../../../shared/components/SectionLabel'

export type LibraryMockState = 'populated' | 'empty' | 'loading'

type Props = {
  state?: LibraryMockState
  layout?: 'grid' | 'list'
}

/**
 * Library-shaped composition from real leaves (Tier 1).
 * Full MyLettersView mount deferred (hook/API coupling).
 */
export default function LibraryMount({ state = 'populated', layout = 'grid' }: Props) {
  if (state === 'loading') {
    return (
      <div className="space-y-3" role="status" aria-label="Loading documents">
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    )
  }

  if (state === 'empty') {
    return (
      <EmptyState
        title="No letters yet"
        action={
          <Link to="/__ui" className={buttonVariantClass('primary') + ' inline-flex items-center px-4 py-2'}>
            Back to catalog
          </Link>
        }
      >
        <p className="text-sm text-muted-subtle">Scan or upload to add your first letter.</p>
      </EmptyState>
    )
  }

  return (
    <div className="space-y-4">
      <SectionLabel>My letters (fixture)</SectionLabel>
      <div className={documentListClassName(layout)}>
        {fixtureDocuments.map((doc) => (
          <DocumentCard key={doc.id} doc={doc} layout={layout} />
        ))}
      </div>
    </div>
  )
}

import DocumentCard from '../../../features/documents/components/DocumentCard'
import { documentListClassName } from '../../../features/documents/components/LibraryToolbar'
import { fixtureDocuments } from '../fixtures/documents'

type Props = {
  layout?: 'grid' | 'list'
}

/** Tier 1 — real DocumentCard + fixtures. */
export default function DocumentCardsMount({ layout = 'grid' }: Props) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">
        Real <code className="text-xs">DocumentCard</code> · layout={layout}
      </p>
      <div className={documentListClassName(layout)}>
        {fixtureDocuments.map((doc) => (
          <DocumentCard key={doc.id} doc={doc} layout={layout} />
        ))}
      </div>
    </div>
  )
}

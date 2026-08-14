import Button from '../../../shared/components/Button'
import Banner from '../../../shared/components/Banner'
import Card from '../../../shared/components/Card'
import Field from '../../../shared/components/Field'
import SectionLabel from '../../../shared/components/SectionLabel'
import EmptyState from '../../../shared/components/EmptyState'
import Skeleton from '../../../shared/components/Skeleton'
import Spinner from '../../../shared/components/Spinner'

/** Tier 0 — real shared primitives. */
export default function PrimitivesMount() {
  return (
    <div className="space-y-8 max-w-2xl">
      <section className="space-y-3">
        <SectionLabel>Buttons</SectionLabel>
        <div className="flex flex-wrap gap-2">
          <Button>Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="danger">Danger</Button>
          <Button disabled>Disabled</Button>
        </div>
      </section>

      <section className="space-y-3">
        <SectionLabel>Banners</SectionLabel>
        <Banner tone="error">Error banner</Banner>
        <Banner tone="success">Success banner</Banner>
        <Banner tone="warning">Warning banner</Banner>
      </section>

      <section className="space-y-3">
        <SectionLabel>Field</SectionLabel>
        <Field id="mock-field" label="Sample" placeholder="Type here" defaultValue="" />
      </section>

      <section className="space-y-3">
        <SectionLabel>Card / Empty / Skeleton / Spinner</SectionLabel>
        <Card className="p-4 text-sm text-muted">Card shell</Card>
        <EmptyState title="Nothing here yet" />
        <div className="space-y-2" role="status" aria-label="Loading example">
          <Skeleton className="h-16 w-full" />
        </div>
        <Spinner label="Working…" />
      </section>
    </div>
  )
}

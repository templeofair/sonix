import Card from '../../../shared/components/Card'
import SectionLabel from '../../../shared/components/SectionLabel'
import Banner from '../../../shared/components/Banner'
import Button from '../../../shared/components/Button'
import { documentStatusPillClass } from '../../../features/documents/lib/documentStatusStyle'

export type DetailMockState = 'pending' | 'processing' | 'failed' | 'partial' | 'ready'

const copy: Record<
  DetailMockState,
  { title: string; body: string; tone?: 'error' | 'warning' | 'success' }
> = {
  pending: {
    title: 'Pending',
    body: 'Extract + OCR controls live in AiPanel on the real detail page.',
  },
  processing: {
    title: 'Processing',
    body: 'Progress + Cancel extraction.',
    tone: 'warning',
  },
  failed: {
    title: 'Failed',
    body: 'Error details + Retry.',
    tone: 'error',
  },
  partial: {
    title: 'Partial',
    body: 'Original saved; Retry available.',
    tone: 'warning',
  },
  ready: {
    title: 'Ready',
    body: 'Summary, date, full text, Re-process.',
    tone: 'success',
  },
}

/**
 * AiPanel status shapes (fixture chrome).
 * Full DocumentDetail + AiPanel mount deferred (heavy hooks).
 */
export default function DetailStatesMount({ state = 'ready' }: { state?: DetailMockState }) {
  const c = copy[state]
  return (
    <div className="space-y-4 max-w-lg">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold text-gray-900 tracking-tight">Demo letter</h2>
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-btn border text-xs font-medium ${documentStatusPillClass(state)}`}
        >
          {state}
        </span>
      </div>
      <Card className="p-4 sm:p-5 space-y-3 shadow-card">
        <SectionLabel as="h2">{c.title}</SectionLabel>
        {c.tone ? <Banner tone={c.tone}>{c.body}</Banner> : <p className="text-sm text-muted">{c.body}</p>}
        <div className="flex flex-wrap gap-2">
          {state === 'pending' && <Button>Extract</Button>}
          {(state === 'failed' || state === 'partial') && <Button>Retry</Button>}
          {state === 'processing' && <Button variant="secondary">Cancel</Button>}
          {state === 'ready' && <Button variant="secondary">Re-process</Button>}
        </div>
      </Card>
      <p className="text-xs text-muted-subtle">
        Catalog status: composed stand-in until AiPanel gains a fixture-friendly seam.
      </p>
    </div>
  )
}

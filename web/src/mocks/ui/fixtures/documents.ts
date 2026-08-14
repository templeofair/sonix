import type { DocumentListItem } from '../../../features/documents/types/document'

const base = {
  created_at: '2026-07-15T10:00:00Z',
  updated_at: '2026-07-15T12:00:00Z',
  page_count: 2,
  thumbnail_available: false,
} as const

/** Sample library cards for `/__ui` (no thumbnails → no broken API image URLs). */
export const fixtureDocuments: DocumentListItem[] = [
  {
    ...base,
    id: 1,
    title: 'Krankenkasse — Jahresabrechnung',
    status: 'ready',
    document_date: '2026-06-01',
  },
  {
    ...base,
    id: 2,
    title: 'Stadtwerke — Rechnung',
    status: 'pending',
    document_date: '2026-07-10',
    page_count: 1,
  },
  {
    ...base,
    id: 3,
    title: 'Scan 2026-07-20',
    status: 'processing',
    page_count: 3,
  },
  {
    ...base,
    id: 4,
    title: 'Failed extract',
    status: 'failed',
    page_count: 1,
  },
  {
    ...base,
    id: 5,
    title: 'Partial — original only',
    status: 'partial',
    document_date: '2026-05-20',
  },
]

export const fixtureDocByStatus = Object.fromEntries(
  fixtureDocuments.map((d) => [d.status, d]),
) as Record<string, DocumentListItem>

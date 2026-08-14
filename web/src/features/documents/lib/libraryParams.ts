import type { DocumentListSort } from '../types/document'

export const LIBRARY_PAGE_SIZE = 20

/** My letters default view: the most recent letters, shown without Load more. */
export const RECENT_LIMIT = 15

export type LibraryLayout = 'grid' | 'list'

export function parseLibraryLayout(raw: string | null): LibraryLayout {
  return raw === 'list' ? 'list' : 'grid'
}

export function parseLibrarySort(raw: string | null): DocumentListSort {
  if (raw === 'date_desc' || raw === 'date_asc' || raw === 'created_desc') return raw
  return 'created_desc'
}

export const STATUS_FILTER_CHIPS = [
  { value: '', label: 'All statuses' },
  { value: 'pending', label: 'Pending' },
  { value: 'processing', label: 'Processing' },
  { value: 'partial', label: 'Partial' },
  { value: 'failed', label: 'Failed' },
  { value: 'ready', label: 'Ready' },
] as const

/** Split a shareable comma-separated filter URL param. */
export function splitFilterCSV(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** Join selected filter values for the URL (empty → omit param). */
export function joinFilterCSV(values: string[]): string {
  return values.map((s) => s.trim()).filter(Boolean).join(',')
}

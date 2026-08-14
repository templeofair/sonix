/** Map accordion-era `?section=` URLs onto flat-library filter params. */

export const QUEUE_STATUS = 'pending,failed,partial'

export type LegacyNormalizeResult = {
  params: URLSearchParams
  changed: boolean
  focusSearch: boolean
}

/**
 * Converts legacy My letters accordion deep links into flat filters.
 * - section=pending → status=pending,failed,partial
 * - section=years → drop section (Explore owns year browse)
 * - section=search → drop section + focus search
 */
export function normalizeLegacyLibraryParams(sp: URLSearchParams): LegacyNormalizeResult {
  const params = new URLSearchParams(sp)
  let changed = false
  let focusSearch = false
  const section = params.get('section')
  if (!section) {
    return { params, changed, focusSearch }
  }
  params.delete('section')
  changed = true
  if (section === 'pending') {
    if (!params.get('status')) params.set('status', QUEUE_STATUS)
  } else if (section === 'search') {
    focusSearch = true
  }
  // section=years (and unknown): just strip section
  return { params, changed, focusSearch }
}

/** Build `/pending` and `/search` redirect targets for the flat library. */
export function pendingRedirectSearch(sp: URLSearchParams): string {
  const next = new URLSearchParams(sp)
  next.delete('section')
  next.set('status', QUEUE_STATUS)
  const s = next.toString()
  return s ? `/?${s}` : `/?status=${encodeURIComponent(QUEUE_STATUS)}`
}

export function searchRedirectSearch(sp: URLSearchParams): string {
  const next = new URLSearchParams(sp)
  next.delete('section')
  next.set('focus', 'search')
  const s = next.toString()
  return s ? `/?${s}` : '/?focus=search'
}

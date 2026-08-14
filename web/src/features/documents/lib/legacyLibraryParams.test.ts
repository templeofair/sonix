import { describe, it, expect } from 'vitest'
import {
  normalizeLegacyLibraryParams,
  pendingRedirectSearch,
  searchRedirectSearch,
  QUEUE_STATUS,
} from './legacyLibraryParams'

describe('normalizeLegacyLibraryParams', () => {
  it('maps section=pending to queue status', () => {
    const { params, changed, focusSearch } = normalizeLegacyLibraryParams(
      new URLSearchParams('section=pending')
    )
    expect(changed).toBe(true)
    expect(focusSearch).toBe(false)
    expect(params.get('section')).toBeNull()
    expect(params.get('status')).toBe(QUEUE_STATUS)
  })

  it('maps section=search to focus without forcing filters', () => {
    const { params, changed, focusSearch } = normalizeLegacyLibraryParams(
      new URLSearchParams('section=search&q=tax')
    )
    expect(changed).toBe(true)
    expect(focusSearch).toBe(true)
    expect(params.get('section')).toBeNull()
    expect(params.get('q')).toBe('tax')
  })

  it('strips section=years', () => {
    const { params, changed } = normalizeLegacyLibraryParams(new URLSearchParams('section=years'))
    expect(changed).toBe(true)
    expect(params.get('section')).toBeNull()
  })
})

describe('legacy route redirects', () => {
  it('maps /pending query onto status filter', () => {
    expect(pendingRedirectSearch(new URLSearchParams())).toContain(
      `status=${encodeURIComponent(QUEUE_STATUS)}`
    )
  })

  it('maps /search query onto focus=search and keeps q', () => {
    const dest = searchRedirectSearch(new URLSearchParams('q=invoice'))
    expect(dest).toContain('q=invoice')
    expect(dest).toContain('focus=search')
    expect(dest).not.toContain('section=')
  })
})

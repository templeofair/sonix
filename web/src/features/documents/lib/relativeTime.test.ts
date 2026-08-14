import { describe, it, expect } from 'vitest'
import { formatCardDate } from './relativeTime'

const noon = (isoDate: string) => new Date(`${isoDate}T12:00:00`)

describe('formatCardDate', () => {
  it('returns Today / Yesterday / N days ago within a week', () => {
    const now = noon('2024-06-10')
    expect(formatCardDate({ document_date: '2024-06-10' }, now)).toBe('Today')
    expect(formatCardDate({ document_date: '2024-06-09' }, now)).toBe('Yesterday')
    expect(formatCardDate({ document_date: '2024-06-07' }, now)).toBe('3 days ago')
  })

  it('falls back to absolute date beyond a week', () => {
    const now = noon('2024-06-10')
    expect(formatCardDate({ document_date: '2024-05-01' }, now)).toBe('2024-05-01')
  })

  it('uses created_at when document_date is missing', () => {
    const now = noon('2024-06-10')
    expect(formatCardDate({ created_at: '2024-06-10T08:00:00Z' }, now)).toBe('Today')
  })

  it('returns empty for missing dates', () => {
    expect(formatCardDate({})).toBe('')
  })
})

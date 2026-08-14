/** Shared relative / absolute date labels for document cards. */

function startOfLocalDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

/**
 * Prefer document_date (YYYY-MM-DD); else created_at ISO. Beyond 7 days → absolute date.
 * Returns empty string when nothing usable is present.
 */
export function formatCardDate(
  opts: { document_date?: string; created_at?: string },
  now: Date = new Date()
): string {
  const raw = (opts.document_date || opts.created_at || '').trim()
  if (!raw) return ''

  let then: Date
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    const [y, m, d] = raw.split('-').map(Number)
    then = new Date(y, m - 1, d)
  } else {
    then = new Date(raw)
  }
  if (Number.isNaN(then.getTime())) return raw.slice(0, 10)

  const today = startOfLocalDay(now)
  const day = startOfLocalDay(then)
  const diffDays = Math.round((today.getTime() - day.getTime()) / 86_400_000)

  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  if (diffDays > 1 && diffDays < 7) return `${diffDays} days ago`
  if (diffDays < 0 && diffDays > -7) {
    // Future within a week — show absolute to avoid odd copy
    return formatAbsolute(day)
  }
  return formatAbsolute(day)
}

function formatAbsolute(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

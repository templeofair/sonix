import { useEffect, useState } from 'react'
import { documentsApi } from '../services/documentsApi'
import type { DocumentDateYear } from '../types/document'

/** Explore folder index: one year per letter date, plus the undated bucket. */
export function useExploreFolders() {
  const [years, setYears] = useState<DocumentDateYear[]>([])
  const [undatedCount, setUndatedCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    documentsApi
      .documentDateYears()
      .then((d) => {
        if (cancelled) return
        setYears(d.years)
        setUndatedCount(d.undated_count)
        setFailed(false)
      })
      .catch(() => {
        if (cancelled) return
        setYears([])
        setUndatedCount(0)
        setFailed(true)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  return { years, undatedCount, loading, failed }
}

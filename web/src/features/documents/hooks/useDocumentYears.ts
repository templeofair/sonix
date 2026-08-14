import { useEffect, useState } from 'react'
import { documentsApi } from '../services/documentsApi'

/** Calendar years that have documents. */
export function useDocumentYears() {
  const [years, setYears] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    documentsApi
      .years()
      .then((d) => setYears(d.years))
      .catch(() => setYears([]))
      .finally(() => setLoading(false))
  }, [])

  return { years, loading }
}

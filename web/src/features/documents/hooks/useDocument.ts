import { useCallback, useEffect, useRef, useState } from 'react'
import { documentsApi } from '../services/documentsApi'
import type { DocumentDetail } from '../types/document'

/** Load a document by route id; poll while status is processing. */
export function useDocument(id: string | undefined) {
  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState(false)
  const docRef = useRef(doc)
  docRef.current = doc

  const refresh = useCallback(() => {
    if (!id) return
    const n = parseInt(id, 10)
    if (Number.isNaN(n)) return
    documentsApi
      .get(n)
      .then((next) => {
        setDoc(next)
        setLoadError(false)
      })
      .catch(() => {
        // Keep the last good payload so a transient failure (e.g. SQLite busy
        // during extract) never blank the page into endless "Loading…".
        if (!docRef.current) setLoadError(true)
      })
      .finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    setDoc(null)
    setLoading(true)
    setLoadError(false)
    refresh()
  }, [refresh])

  useEffect(() => {
    if (!doc || doc.status !== 'processing') return
    const t = setInterval(() => {
      documentsApi
        .get(doc.id)
        .then((next) => {
          setDoc(next)
          setLoadError(false)
        })
        .catch(() => {
          /* keep showing last known doc while processing */
        })
    }, 2000)
    return () => clearInterval(t)
  }, [doc?.id, doc?.status])

  return { doc, setDoc, refresh, loading, loadError }
}

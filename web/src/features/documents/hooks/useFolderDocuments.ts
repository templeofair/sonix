import { useCallback, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { documentsApi } from '../services/documentsApi'
import type { DocumentListItem, DocumentListParams } from '../types/document'
import { LIBRARY_PAGE_SIZE, parseLibraryLayout, type LibraryLayout } from '../lib/libraryParams'

/**
 * One Explore folder: a letter-date year, or the undated bucket.
 * Years sort by letter date (newest first, `Oldest first` flips it); the undated
 * folder has no letter date, so it keeps the server's import order.
 */
export function useFolderDocuments(year: string | undefined, undated = false) {
  const [searchParams, setSearchParams] = useSearchParams()
  const layout = parseLibraryLayout(searchParams.get('layout'))
  const oldestFirst = searchParams.get('sort') === 'date_asc'
  const validYear = undated ? undefined : year && /^\d{4}$/.test(year) ? year : undefined
  const enabled = undated || Boolean(validYear)

  const [docs, setDocs] = useState<DocumentListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const patchParams = useCallback(
    (mutate: (p: URLSearchParams) => void) => {
      const params = new URLSearchParams(searchParams)
      mutate(params)
      setSearchParams(params, { replace: true })
    },
    [searchParams, setSearchParams]
  )

  const setLayout = (next: LibraryLayout) => {
    patchParams((p) => {
      if (next === 'grid') p.delete('layout')
      else p.set('layout', next)
    })
  }

  const setOldestFirst = (next: boolean) => {
    patchParams((p) => {
      if (next) p.set('sort', 'date_asc')
      else p.delete('sort')
    })
  }

  const listFilter = useCallback((): DocumentListParams => {
    if (undated) return { undated: 1 }
    return {
      document_date_from: `${validYear}-01-01`,
      document_date_to: `${validYear}-12-31`,
      sort: oldestFirst ? 'date_asc' : 'date_desc',
    }
  }, [undated, validYear, oldestFirst])

  const fetchPage = useCallback(
    (pageIndex: number, append: boolean) => {
      return documentsApi
        .list({ ...listFilter(), page: pageIndex, limit: LIBRARY_PAGE_SIZE })
        .then((d) => {
          setDocs((prev) => (append ? [...prev, ...d.documents] : d.documents))
          setTotal(d.total)
          setPage(pageIndex)
        })
        .catch(() => {
          if (!append) {
            setDocs([])
            setTotal(0)
          }
        })
    },
    [listFilter]
  )

  useEffect(() => {
    if (!enabled) {
      setDocs([])
      setTotal(0)
      setLoading(false)
      return
    }
    setLoading(true)
    void fetchPage(0, false).finally(() => setLoading(false))
  }, [enabled, fetchPage])

  const loadMore = () => {
    if (loadingMore || docs.length >= total) return
    setLoadingMore(true)
    void fetchPage(page + 1, true).finally(() => setLoadingMore(false))
  }

  return {
    docs,
    total,
    loading,
    loadingMore,
    hasMore: docs.length < total,
    layout,
    oldestFirst,
    setLayout,
    setOldestFirst,
    loadMore,
  }
}

import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { documentsApi } from '../services/documentsApi'
import type { DocumentListItem, DocumentListSort } from '../types/document'
import { isServerUnreachableError } from '../../../lib/errors'
import {
  LIBRARY_PAGE_SIZE,
  RECENT_LIMIT,
  joinFilterCSV,
  parseLibraryLayout,
  parseLibrarySort,
  splitFilterCSV,
  type LibraryLayout,
} from '../lib/libraryParams'
import { normalizeLegacyLibraryParams } from '../lib/legacyLibraryParams'

/** Flat My letters library: URL filters, pagination, refetch on focus. */
export function useLibrary() {
  const [searchParams, setSearchParams] = useSearchParams()
  const searchInputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (searchParams.has('section')) {
      const { params, focusSearch } = normalizeLegacyLibraryParams(searchParams)
      if (focusSearch) params.set('focus', 'search')
      setSearchParams(params, { replace: true })
      return
    }
    if (searchParams.get('focus') === 'search') {
      const cleaned = new URLSearchParams(searchParams)
      cleaned.delete('focus')
      setSearchParams(cleaned, { replace: true })
      queueMicrotask(() => searchInputRef.current?.focus())
    }
  }, [searchParams, setSearchParams])

  const q = searchParams.get('q') ?? ''
  const dateFrom = searchParams.get('date_from') ?? ''
  const dateTo = searchParams.get('date_to') ?? ''
  const status = searchParams.get('status') ?? ''
  const tag = searchParams.get('tag') ?? ''
  const year = searchParams.get('year') ?? ''
  const statusValues = splitFilterCSV(status)
  const tagValues = splitFilterCSV(tag)
  const yearValues = splitFilterCSV(year)
  const sort = parseLibrarySort(searchParams.get('sort'))
  const layout = parseLibraryLayout(searchParams.get('layout'))

  /**
   * Default view = no search text and no filter in the URL: the Recent letters.
   * Any of q / status / tag / year / date_from / date_to keeps today's paginated list.
   */
  const isDefaultView = !(q || dateFrom || dateTo || status || tag || year)
  const pageSize = isDefaultView ? RECENT_LIMIT : LIBRARY_PAGE_SIZE

  const [docs, setDocs] = useState<DocumentListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [unreachable, setUnreachable] = useState(false)
  const [years, setYears] = useState<string[]>([])
  const [tags, setTags] = useState<string[]>([])
  const [textInput, setTextInput] = useState(q)
  const [fromInput, setFromInput] = useState(dateFrom)
  const [toInput, setToInput] = useState(dateTo)

  const patchParams = useCallback(
    (mutate: (p: URLSearchParams) => void) => {
      const params = new URLSearchParams(searchParams)
      mutate(params)
      params.delete('section')
      params.delete('focus')
      setSearchParams(params, { replace: true })
    },
    [searchParams, setSearchParams]
  )

  // Strip retired ?category= bookmarks (auto-category sunset).
  useEffect(() => {
    if (!searchParams.has('category')) return
    const params = new URLSearchParams(searchParams)
    params.delete('category')
    setSearchParams(params, { replace: true })
  }, [searchParams, setSearchParams])

  const listFilter = useCallback(() => {
    return {
      q: q || undefined,
      document_date_from: dateFrom || undefined,
      document_date_to: dateTo || undefined,
      status: status || undefined,
      tag: tag || undefined,
      year: year || undefined,
      sort,
    }
  }, [q, dateFrom, dateTo, status, tag, year, sort])

  const fetchPage = useCallback(
    (pageIndex: number, append: boolean) => {
      return documentsApi
        .list({ ...listFilter(), page: pageIndex, limit: pageSize })
        .then((d) => {
          setDocs((prev) => (append ? [...prev, ...d.documents] : d.documents))
          setTotal(d.total)
          setPage(pageIndex)
          setUnreachable(false)
        })
        .catch((err) => {
          if (!append) {
            setDocs([])
            setTotal(0)
            if (isServerUnreachableError(err)) setUnreachable(true)
          }
        })
    },
    [listFilter, pageSize]
  )

  useEffect(() => {
    setTextInput(q)
    setFromInput(dateFrom)
    setToInput(dateTo)
  }, [q, dateFrom, dateTo])

  useEffect(() => {
    setLoading(true)
    void fetchPage(0, false).finally(() => setLoading(false))
  }, [fetchPage])

  useEffect(() => {
    documentsApi
      .years()
      .then((d) => setYears(d.years))
      .catch(() => setYears([]))
    documentsApi
      .tags()
      .then((d) => setTags(d.tags))
      .catch(() => setTags([]))
  }, [])

  useEffect(() => {
    const onFocusOrVisible = () => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
      void fetchPage(0, false)
    }
    window.addEventListener('focus', onFocusOrVisible)
    document.addEventListener('visibilitychange', onFocusOrVisible)
    return () => {
      window.removeEventListener('focus', onFocusOrVisible)
      document.removeEventListener('visibilitychange', onFocusOrVisible)
    }
  }, [fetchPage])

  const runSearch = () => {
    patchParams((params) => {
      if (textInput.trim()) params.set('q', textInput.trim())
      else params.delete('q')
      if (fromInput) params.set('date_from', fromInput)
      else params.delete('date_from')
      if (toInput) params.set('date_to', toInput)
      else params.delete('date_to')
    })
  }

  const setLayout = (next: LibraryLayout) => {
    patchParams((p) => {
      if (next === 'grid') p.delete('layout')
      else p.set('layout', next)
    })
  }

  const setSort = (next: DocumentListSort) => {
    patchParams((p) => {
      if (next === 'created_desc') p.delete('sort')
      else p.set('sort', next)
    })
  }

  const setStatusFilter = (next: string[]) => {
    const joined = joinFilterCSV(next)
    patchParams((p) => {
      if (joined) p.set('status', joined)
      else p.delete('status')
    })
  }

  const setTagFilter = (next: string[]) => {
    const joined = joinFilterCSV(next)
    patchParams((p) => {
      if (joined) p.set('tag', joined)
      else p.delete('tag')
    })
  }

  const setYearFilter = (next: string[]) => {
    const joined = joinFilterCSV(next)
    patchParams((p) => {
      if (joined) p.set('year', joined)
      else p.delete('year')
    })
  }

  const loadMore = () => {
    if (loadingMore || docs.length >= total) return
    setLoadingMore(true)
    void fetchPage(page + 1, true).finally(() => setLoadingMore(false))
  }

  return {
    searchInputRef,
    q,
    dateFrom,
    dateTo,
    status,
    statusValues,
    tag,
    tagValues,
    year,
    yearValues,
    years,
    tags,
    sort,
    layout,
    docs,
    total,
    isDefaultView,
    loading,
    loadingMore,
    hasMore: !isDefaultView && docs.length < total,
    unreachable,
    textInput,
    setTextInput,
    fromInput,
    setFromInput,
    toInput,
    setToInput,
    runSearch,
    setLayout,
    setSort,
    setStatusFilter,
    setTagFilter,
    setYearFilter,
    loadMore,
    reload: () => void fetchPage(0, false),
  }
}

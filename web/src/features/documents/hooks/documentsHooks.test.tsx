import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useDocumentYears } from './useDocumentYears'
import { useCreateAndUpload } from './useCreateAndUpload'
import { useDocument } from './useDocument'
import { useLibrary } from './useLibrary'
import { useExploreFolders } from './useExploreFolders'
import { useFolderDocuments } from './useFolderDocuments'

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    years: vi.fn(),
    tags: vi.fn(),
    documentDateYears: vi.fn(),
    list: vi.fn(),
    get: vi.fn(),
    extract: vi.fn(),
    delete: vi.fn(),
    create: vi.fn(),
    uploadPages: vi.fn(),
  },
}))

import { documentsApi } from '../services/documentsApi'

const mockedApi = vi.mocked(documentsApi)

const sampleDoc = {
  id: 1,
  status: 'ready',
  created_at: '2024-01-01',
  updated_at: '2024-01-01',
  page_count: 0,
  thumbnail_available: false,
  pages: [],
}

function wrapSearch(initialEntry: string) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <MemoryRouter initialEntries={[initialEntry]}>{children}</MemoryRouter>
  }
}

describe('useDocumentYears', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads years from the API', async () => {
    mockedApi.years.mockResolvedValue({ years: ['2024', '2023'] })
    const { result } = renderHook(() => useDocumentYears())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.years).toEqual(['2024', '2023'])
  })
})

describe('useLibrary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedApi.years.mockResolvedValue({ years: ['2024'] })
    mockedApi.tags.mockResolvedValue({ tags: ['bank', 'invoice'] })
  })

  it('loads the library and supports load more', async () => {
    mockedApi.list
      .mockResolvedValueOnce({
        documents: [
          {
            id: 1,
            status: 'pending',
            created_at: '',
            updated_at: '',
            page_count: 1,
            thumbnail_available: true,
          },
        ],
        total: 2,
      })
      .mockResolvedValueOnce({
        documents: [
          {
            id: 2,
            status: 'failed',
            created_at: '',
            updated_at: '',
            page_count: 1,
            thumbnail_available: false,
          },
        ],
        total: 2,
      })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.docs).toHaveLength(1)
    result.current.loadMore()
    await waitFor(() => expect(result.current.docs).toHaveLength(2))
  })

  it('applies status, tag, sort and year from the URL and strips retired category', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 0 })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/?category=tax&status=ready&tag=bank&sort=date_desc&layout=list&year=2024'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.layout).toBe('list')
    expect(result.current.sort).toBe('date_desc')
    expect(result.current.status).toBe('ready')
    expect(result.current.statusValues).toEqual(['ready'])
    expect(result.current.tag).toBe('bank')
    expect(result.current.tagValues).toEqual(['bank'])
    expect(result.current.year).toBe('2024')
    expect(result.current.yearValues).toEqual(['2024'])
    expect(mockedApi.list).toHaveBeenCalledWith(
      expect.objectContaining({
        status: 'ready',
        tag: 'bank',
        year: '2024',
        sort: 'date_desc',
      })
    )
    expect(mockedApi.list.mock.calls.some((c) => 'category' in (c[0] as object))).toBe(false)
    expect(mockedApi.list.mock.calls.some((c) => 'created_from' in (c[0] as object))).toBe(false)
  })

  it('accepts multi-value status, tag, and year URL params', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 0 })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/?status=pending,failed&tag=bank,tax&year=2023,2024'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.statusValues).toEqual(['pending', 'failed'])
    expect(result.current.tagValues).toEqual(['bank', 'tax'])
    expect(result.current.yearValues).toEqual(['2023', '2024'])
    expect(mockedApi.list).toHaveBeenCalledWith(
      expect.objectContaining({
        status: 'pending,failed',
        tag: 'bank,tax',
        year: '2023,2024',
      })
    )
  })

  it('asks for the recent 15 with no Load more when nothing is filtered', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 42 })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.isDefaultView).toBe(true)
    expect(mockedApi.list).toHaveBeenCalledWith(expect.objectContaining({ limit: 15, page: 0 }))
    expect(result.current.hasMore).toBe(false)
  })

  it('keeps the paginated list when the URL carries a filter', async () => {
    mockedApi.list.mockResolvedValue({
      documents: [
        {
          id: 1,
          status: 'pending',
          created_at: '',
          updated_at: '',
          page_count: 1,
          thumbnail_available: false,
        },
      ],
      total: 42,
    })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/?status=pending,failed,partial'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.isDefaultView).toBe(false)
    expect(mockedApi.list).toHaveBeenCalledWith(expect.objectContaining({ limit: 20 }))
    expect(result.current.hasMore).toBe(true)
  })

  it('maps legacy section=pending onto the queue status filter', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 0 })
    const { result } = renderHook(() => useLibrary(), {
      wrapper: wrapSearch('/?section=pending'),
    })
    await waitFor(() => expect(result.current.status).toBe('pending,failed,partial'))
  })
})

describe('useCreateAndUpload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('creates a document then uploads pages', async () => {
    mockedApi.create.mockResolvedValue({ id: 42 })
    mockedApi.uploadPages.mockResolvedValue({ ok: true, document_id: 42 })
    const { result } = renderHook(() => useCreateAndUpload())
    const file = new File(['x'], 'a.jpg', { type: 'image/jpeg' })
    const id = await result.current.createAndUpload('Title', [file])
    expect(id).toBe(42)
    expect(mockedApi.create).toHaveBeenCalledWith('Title')
    expect(mockedApi.uploadPages).toHaveBeenCalledWith(42, [file])
  })
})

describe('useDocument', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads a document when id is provided', async () => {
    mockedApi.get.mockResolvedValue(sampleDoc)
    const { result } = renderHook(() => useDocument('1'))
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.doc).toEqual(sampleDoc))
    expect(result.current.loading).toBe(false)
    expect(mockedApi.get).toHaveBeenCalledWith(1)
  })

  it('refresh reloads the document', async () => {
    mockedApi.get.mockResolvedValue(sampleDoc)
    const { result } = renderHook(() => useDocument('1'))
    await waitFor(() => expect(result.current.doc).toEqual(sampleDoc))
    mockedApi.get.mockClear()
    result.current.refresh()
    await waitFor(() => expect(mockedApi.get).toHaveBeenCalledWith(1))
  })

  it('keeps the current document when a refresh fails', async () => {
    mockedApi.get.mockResolvedValue(sampleDoc)
    const { result } = renderHook(() => useDocument('1'))
    await waitFor(() => expect(result.current.doc).toEqual(sampleDoc))
    mockedApi.get.mockRejectedValueOnce(new Error('busy'))
    result.current.refresh()
    await waitFor(() => expect(mockedApi.get).toHaveBeenCalledTimes(2))
    expect(result.current.doc).toEqual(sampleDoc)
    expect(result.current.loadError).toBe(false)
  })

  it('surfaces loadError when the initial fetch fails', async () => {
    mockedApi.get.mockRejectedValue(new Error('not found'))
    const { result } = renderHook(() => useDocument('1'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.doc).toBeNull()
    expect(result.current.loadError).toBe(true)
  })
})

describe('useExploreFolders', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the letter-date year facet', async () => {
    mockedApi.documentDateYears.mockResolvedValue({
      years: [{ year: '2024', count: 37 }],
      undated_count: 4,
    })
    const { result } = renderHook(() => useExploreFolders())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.years).toEqual([{ year: '2024', count: 37 }])
    expect(result.current.undatedCount).toBe(4)
    expect(result.current.failed).toBe(false)
  })

  it('flags a failed facet load', async () => {
    mockedApi.documentDateYears.mockRejectedValue(new Error('offline'))
    const { result } = renderHook(() => useExploreFolders())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.failed).toBe(true)
  })
})

describe('useFolderDocuments', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads a year folder by letter date, newest first', async () => {
    mockedApi.list.mockResolvedValue({
      documents: [
        {
          id: 3,
          status: 'ready',
          created_at: '2024-06-01',
          updated_at: '',
          document_date: '2024-06-01',
          page_count: 2,
          thumbnail_available: true,
        },
      ],
      total: 1,
    })
    const { result } = renderHook(() => useFolderDocuments('2024'), {
      wrapper: wrapSearch('/explore/2024'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.docs).toHaveLength(1)
    expect(mockedApi.list).toHaveBeenCalledWith({
      document_date_from: '2024-01-01',
      document_date_to: '2024-12-31',
      sort: 'date_desc',
      page: 0,
      limit: 20,
    })
  })

  it('reads Oldest first from the URL', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 0 })
    const { result } = renderHook(() => useFolderDocuments('2024'), {
      wrapper: wrapSearch('/explore/2024?sort=date_asc'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.oldestFirst).toBe(true)
    expect(mockedApi.list).toHaveBeenCalledWith(expect.objectContaining({ sort: 'date_asc' }))
  })

  it('loads the undated folder without a date range', async () => {
    mockedApi.list.mockResolvedValue({ documents: [], total: 0 })
    const { result } = renderHook(() => useFolderDocuments(undefined, true), {
      wrapper: wrapSearch('/explore/no-date'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(mockedApi.list).toHaveBeenCalledWith({ undated: 1, page: 0, limit: 20 })
  })

  it('does not query for a non-year folder', async () => {
    const { result } = renderHook(() => useFolderDocuments('nonsense'), {
      wrapper: wrapSearch('/explore/nonsense'),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(mockedApi.list).not.toHaveBeenCalled()
  })
})

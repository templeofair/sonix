import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MyLettersView from './MyLettersView'

vi.mock('../hooks/useLibrary', () => ({
  useLibrary: vi.fn(),
}))

vi.mock('../../../auth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}))

import { useLibrary } from '../hooks/useLibrary'

const mockedHook = vi.mocked(useLibrary)

const doc = {
  id: 7,
  title: 'Invoice ACME',
  status: 'pending',
  created_at: '2024-06-01T12:00:00Z',
  updated_at: '2024-06-01T12:00:00Z',
  page_count: 2,
  thumbnail_available: true,
}

function libraryState(overrides: Partial<ReturnType<typeof useLibrary>>) {
  return {
    searchInputRef: { current: null },
    q: '',
    dateFrom: '',
    dateTo: '',
    status: '',
    statusValues: [],
    tag: '',
    tagValues: [],
    year: '',
    yearValues: [],
    years: ['2024'],
    tags: ['invoice'],
    sort: 'created_desc',
    layout: 'list',
    docs: [doc],
    total: 1,
    isDefaultView: true,
    loading: false,
    loadingMore: false,
    hasMore: false,
    unreachable: false,
    textInput: '',
    setTextInput: vi.fn(),
    fromInput: '',
    setFromInput: vi.fn(),
    toInput: '',
    setToInput: vi.fn(),
    runSearch: vi.fn(),
    setLayout: vi.fn(),
    setSort: vi.fn(),
    setStatusFilter: vi.fn(),
    setTagFilter: vi.fn(),
    setYearFilter: vi.fn(),
    loadMore: vi.fn(),
    reload: vi.fn(),
    ...overrides,
  } as ReturnType<typeof useLibrary>
}

describe('MyLettersView flat library', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders search, Select, and cards that open the document', () => {
    mockedHook.mockReturnValue(
      libraryState({
        status: 'pending,failed,partial',
        statusValues: ['pending', 'failed', 'partial'],
        isDefaultView: false,
      })
    )

    render(
      <MemoryRouter>
        <MyLettersView />
      </MemoryRouter>
    )
    expect(screen.getByRole('button', { name: 'Search' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Select' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Queue/i })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Invoice ACME' })).toHaveAttribute('href', '/documents/7')
    expect(screen.queryByLabelText('Library overview')).not.toBeInTheDocument()
  })

  it('default view shows the recent letters with More options collapsed and no Load more', () => {
    mockedHook.mockReturnValue(libraryState({}))

    render(
      <MemoryRouter>
        <MyLettersView />
      </MemoryRouter>
    )
    expect(screen.getByRole('link', { name: 'Invoice ACME' })).toBeInTheDocument()
    expect(screen.getByText('More options')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Any status' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Load more/i })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Explore' })).toHaveAttribute('href', '/explore')
  })

  it('filtered view shows More options active and Load more', () => {
    mockedHook.mockReturnValue(
      libraryState({
        tag: 'invoice',
        tagValues: ['invoice'],
        isDefaultView: false,
        total: 40,
        hasMore: true,
      })
    )

    render(
      <MemoryRouter initialEntries={['/?tag=invoice']}>
        <MyLettersView />
      </MemoryRouter>
    )
    expect(screen.getByText('More options')).toBeInTheDocument()
    expect(screen.getByText('(active)')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Load more' })).toBeInTheDocument()
  })

  it('tells an empty library apart from an empty search', () => {
    mockedHook.mockReturnValue(libraryState({ docs: [], total: 0 }))
    const empty = render(
      <MemoryRouter>
        <MyLettersView />
      </MemoryRouter>
    )
    expect(screen.getByText('Nothing scanned yet')).toBeInTheDocument()
    empty.unmount()

    mockedHook.mockReturnValue(
      libraryState({ docs: [], total: 0, q: 'rechnung', isDefaultView: false })
    )
    render(
      <MemoryRouter initialEntries={['/?q=rechnung']}>
        <MyLettersView />
      </MemoryRouter>
    )
    expect(screen.getByText('No letters match this search')).toBeInTheDocument()
  })
})

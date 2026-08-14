import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import ExploreFolderView from './ExploreFolderView'

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    list: vi.fn(),
    pageThumbnailUrl: () => '/thumb',
    pageImageUrl: () => '/page',
  },
}))

vi.mock('../../../auth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}))

import { documentsApi } from '../services/documentsApi'

const mockedList = vi.mocked(documentsApi.list)

const lease = {
  id: 9,
  title: 'Lease 2024',
  status: 'ready',
  created_at: '2024-05-01T00:00:00Z',
  updated_at: '2024-05-01T00:00:00Z',
  document_date: '2024-04-20',
  page_count: 3,
  thumbnail_available: true,
}

const older = {
  id: 10,
  title: 'Kontoauszug Januar',
  status: 'partial',
  created_at: '2024-02-01T00:00:00Z',
  updated_at: '2024-02-01T00:00:00Z',
  document_date: '2024-01-05',
  page_count: 1,
  thumbnail_available: false,
}

function renderYear(entry = '/explore/2024') {
  return render(
    <MemoryRouter initialEntries={[entry]}>
      <Routes>
        <Route path="/explore/:year" element={<ExploreFolderView />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('ExploreFolderView year folder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('queries the letter-date range newest first and lists the folder contents', async () => {
    mockedList.mockResolvedValue({ documents: [lease, older], total: 2 })
    renderYear()

    await waitFor(() => expect(screen.getByText('Lease 2024')).toBeInTheDocument())
    expect(mockedList).toHaveBeenCalledWith({
      document_date_from: '2024-01-01',
      document_date_to: '2024-12-31',
      sort: 'date_desc',
      page: 0,
      limit: 20,
    })
    expect(screen.getByText('ready')).toBeInTheDocument()
    expect(screen.getByText('2024-04-20')).toBeInTheDocument()
    expect(screen.getByText('3 pages')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Lease 2024' })).toHaveAttribute('href', '/documents/9')
    expect(screen.queryByRole('button', { name: /Actions for/i })).not.toBeInTheDocument()

    const titles = screen
      .getAllByRole('listitem')
      .map((li) => within(li).getAllByRole('link')[0].getAttribute('aria-label'))
    expect(titles).toEqual(['Lease 2024', 'Kontoauszug Januar'])
  })

  it('offers Back to Explore and an Oldest first toggle that re-queries date_asc', async () => {
    mockedList.mockResolvedValue({ documents: [lease], total: 1 })
    renderYear()
    await waitFor(() => expect(screen.getByText('Lease 2024')).toBeInTheDocument())
    expect(screen.getAllByRole('button', { name: 'Back to Explore' }).length).toBeGreaterThan(0)

    const toggle = screen.getByRole('button', { name: 'Oldest first' })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    await userEvent.click(toggle)

    await waitFor(() =>
      expect(mockedList).toHaveBeenCalledWith(expect.objectContaining({ sort: 'date_asc' }))
    )
    expect(screen.getByRole('button', { name: 'Oldest first' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
  })

  it('pages the folder with Load more', async () => {
    mockedList
      .mockResolvedValueOnce({ documents: [lease], total: 2 })
      .mockResolvedValueOnce({ documents: [older], total: 2 })
    renderYear()
    await waitFor(() => expect(screen.getByText('Lease 2024')).toBeInTheDocument())

    await userEvent.click(screen.getByRole('button', { name: 'Load more' }))
    await waitFor(() => expect(screen.getByText('Kontoauszug Januar')).toBeInTheDocument())
    expect(mockedList).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, limit: 20 }))
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
  })

  it('shows an empty folder state', async () => {
    mockedList.mockResolvedValue({ documents: [], total: 0 })
    renderYear()
    await waitFor(() =>
      expect(screen.getByText('No letters in this folder')).toBeInTheDocument()
    )
  })
})

describe('ExploreFolderView No date folder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('queries undated letters in import order and hides the sort toggle', async () => {
    const undatedDoc = { ...older, id: 11, title: 'Ohne Datum', document_date: undefined }
    mockedList.mockResolvedValue({ documents: [undatedDoc], total: 1 })
    render(
      <MemoryRouter initialEntries={['/explore/no-date']}>
        <Routes>
          <Route path="/explore/no-date" element={<ExploreFolderView undated />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => expect(screen.getByText('Ohne Datum')).toBeInTheDocument())
    expect(mockedList).toHaveBeenCalledWith({ undated: 1, page: 0, limit: 20 })
    expect(screen.queryByRole('button', { name: 'Oldest first' })).not.toBeInTheDocument()
  })
})

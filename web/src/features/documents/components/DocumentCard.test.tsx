import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DocumentCard from './DocumentCard'

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    pageThumbnailUrl: (id: number, page: number) => `/api/documents/${id}/pages/${page}/thumbnail`,
    pageImageUrl: (id: number, page: number) => `/api/documents/${id}/pages/${page}/image`,
  },
}))

const doc = {
  id: 1,
  title: 'Short title',
  status: 'ready' as const,
  created_at: '2020-01-01T00:00:00Z',
  updated_at: '2020-01-01T00:00:00Z',
  page_count: 1,
  thumbnail_available: true,
}

describe('DocumentCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('links to the document with title, status, and no action menu', () => {
    render(
      <MemoryRouter>
        <DocumentCard doc={doc} />
      </MemoryRouter>
    )
    expect(screen.getByRole('link', { name: 'Short title' })).toHaveAttribute('href', '/documents/1')
    expect(screen.getByText('ready')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Actions for/i })).not.toBeInTheDocument()
  })

  it('opens a preview dialog after long-pressing the thumbnail', () => {
    render(
      <MemoryRouter>
        <DocumentCard doc={doc} layout="grid" />
      </MemoryRouter>
    )
    const thumb = screen.getByTestId('doc-card-thumb')
    fireEvent.pointerDown(thumb, { button: 0 })
    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument()
  })

  it('does not open preview before the long-press threshold', () => {
    render(
      <MemoryRouter>
        <DocumentCard doc={doc} layout="grid" />
      </MemoryRouter>
    )
    fireEvent.pointerDown(screen.getByTestId('doc-card-thumb'), { button: 0 })
    act(() => {
      vi.advanceTimersByTime(400)
    })
    fireEvent.pointerUp(screen.getByTestId('doc-card-thumb'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('opens preview from the Preview control', () => {
    render(
      <MemoryRouter>
        <DocumentCard doc={doc} layout="grid" />
      </MemoryRouter>
    )
    fireEvent.click(screen.getByRole('button', { name: 'Preview first page' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})

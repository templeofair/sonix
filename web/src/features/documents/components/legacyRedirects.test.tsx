import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from '../../../App'
import { AuthProvider } from '../../auth'

vi.mock('../../auth/services/authApi', async () => {
  const actual = await vi.importActual<typeof import('../../auth/services/authApi')>(
    '../../auth/services/authApi'
  )
  return {
    ...actual,
    fetchMeWithTimeout: vi.fn().mockResolvedValue({
      unreachable: false,
      user: { id: 1, username: 'admin' },
    }),
    login: vi.fn(),
    logout: vi.fn(),
  }
})

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    list: vi.fn().mockResolvedValue({ documents: [], total: 0 }),
    years: vi.fn().mockResolvedValue({ years: ['2024'] }),
    tags: vi.fn().mockResolvedValue({ tags: [] }),
    documentDateYears: vi.fn().mockResolvedValue({ years: [], undated_count: 0 }),
    pageThumbnailUrl: () => '/thumb',
    extract: vi.fn().mockResolvedValue({ status: 'processing' }),
    delete: vi.fn().mockResolvedValue(undefined),
  },
}))

describe('legacy library redirects', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps /search?q=x onto the flat library with search focused', async () => {
    render(
      <MemoryRouter initialEntries={['/search?q=x&date_from=2024-01-01']}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>
    )
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'My letters' })).toBeInTheDocument()
    })
    expect(screen.getByLabelText(/Search summary, content, and tags/i)).toBeInTheDocument()
  })

  it('maps /pending onto the queue status filter', async () => {
    const { documentsApi } = await import('../services/documentsApi')
    render(
      <MemoryRouter initialEntries={['/pending']}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>
    )
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'My letters' })).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(documentsApi.list).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'pending,failed,partial' })
      )
    })
    expect(screen.getByText('More options')).toBeInTheDocument()
  })

  it('maps the retired /year/:year onto the Explore folder for that year', async () => {
    render(
      <MemoryRouter initialEntries={['/year/2024']}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>
    )
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '2024' })).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: 'Back to Explore' })).toBeInTheDocument()
  })
})

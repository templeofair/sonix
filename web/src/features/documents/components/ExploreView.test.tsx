import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import ExploreView from './ExploreView'

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    documentDateYears: vi.fn(),
  },
}))

vi.mock('../../../auth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}))

import { documentsApi } from '../services/documentsApi'

const mockedFacet = vi.mocked(documentsApi.documentDateYears)

function renderExplore() {
  return render(
    <MemoryRouter>
      <ExploreView />
    </MemoryRouter>
  )
}

describe('ExploreView folder index', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists a folder per letter-date year with a readable accessible name', async () => {
    mockedFacet.mockResolvedValue({
      years: [
        { year: '2026', count: 12 },
        { year: '2024', count: 37 },
      ],
      undated_count: 4,
    })
    renderExplore()

    const year = await screen.findByRole('link', { name: '2024, 37 letters' })
    expect(year).toHaveAttribute('href', '/explore/2024')
    expect(screen.getByRole('link', { name: '2026, 12 letters' })).toHaveAttribute(
      'href',
      '/explore/2026'
    )
  })

  it('puts the No date folder last and only when there are undated letters', async () => {
    mockedFacet.mockResolvedValue({
      years: [{ year: '2026', count: 1 }],
      undated_count: 1,
    })
    const withUndated = renderExplore()
    const noDate = await screen.findByRole('link', { name: 'No date, 1 letter' })
    expect(noDate).toHaveAttribute('href', '/explore/no-date')
    const names = screen.getAllByRole('link').map((el) => el.getAttribute('aria-label'))
    expect(names).toEqual(['2026, 1 letter', 'No date, 1 letter'])
    withUndated.unmount()

    mockedFacet.mockResolvedValue({ years: [{ year: '2026', count: 1 }], undated_count: 0 })
    renderExplore()
    await screen.findByRole('link', { name: '2026, 1 letter' })
    expect(screen.queryByRole('link', { name: /No date/i })).not.toBeInTheDocument()
  })

  it('shows an empty state when nothing is scanned yet', async () => {
    mockedFacet.mockResolvedValue({ years: [], undated_count: 0 })
    renderExplore()
    await waitFor(() => expect(screen.getByText('Nothing scanned yet')).toBeInTheDocument())
  })

  it('reports a failed facet load', async () => {
    mockedFacet.mockRejectedValue(new Error('offline'))
    renderExplore()
    await waitFor(() => expect(screen.getByText('Could not load folders.')).toBeInTheDocument())
  })
})

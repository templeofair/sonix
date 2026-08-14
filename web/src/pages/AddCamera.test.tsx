import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import AddCamera from './AddCamera'
import { CaptureDraftProvider } from '../features/documents/hooks/CaptureDraftContext'

vi.mock('../features/documents/services/documentsApi', () => ({
  documentsApi: {
    create: vi.fn(),
    uploadPages: vi.fn(),
    years: vi.fn(),
    list: vi.fn(),
  },
}))

function renderCamera(initialEntry = '/add/camera', state?: object) {
  return render(
    <CaptureDraftProvider>
      <MemoryRouter initialEntries={[{ pathname: initialEntry, state }]}>
        <Routes>
          <Route path="/add/camera" element={<AddCamera />} />
          <Route path="/add/review" element={<div data-testid="review">review</div>} />
          <Route path="/add" element={<div data-testid="add-hub">add</div>} />
        </Routes>
      </MemoryRouter>
    </CaptureDraftProvider>
  )
}

describe('AddCamera', () => {
  const origNav = globalThis.navigator

  beforeEach(() => {
    vi.stubGlobal('navigator', {
      ...origNav,
      mediaDevices: {
        getUserMedia: vi.fn().mockRejectedValue(new Error('denied')),
      },
    })
  })

  afterEach(() => {
    vi.stubGlobal('navigator', origNav)
    vi.unstubAllGlobals()
  })

  it('uses Cancel and Done in the editor shell (no Back/Close)', async () => {
    renderCamera()

    expect(screen.queryByText(/scan letters/i)).toBeNull()
    expect(screen.getByRole('button', { name: /^cancel$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^done$/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^back$/i })).toBeNull()
    expect(screen.queryByRole('button', { name: /^close$/i })).toBeNull()
  })

  it('root is viewport-fixed and non-scrolling', () => {
    const { container } = renderCamera()
    const root = container.firstElementChild as HTMLElement
    expect(root.className).toMatch(/fixed/)
    expect(root.className).toMatch(/overflow-hidden/)
  })

  it('does not render torch when camera never starts', () => {
    renderCamera()
    expect(screen.queryByRole('button', { name: /torch/i })).toBeNull()
    expect(screen.queryByRole('group', { name: 'Colour mode' })).toBeNull()
  })
})

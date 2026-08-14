import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import AddUpload from './AddUpload'
import { CaptureDraftProvider } from '../features/documents/hooks/CaptureDraftContext'

vi.mock('../auth', () => ({
  useAuth: () => ({
    user: { username: 'test' },
    loading: false,
    login: vi.fn(),
    logout: vi.fn(),
  }),
}))

function renderUpload(entries: string[], index?: number) {
  return render(
    <CaptureDraftProvider>
      <MemoryRouter initialEntries={entries} initialIndex={index}>
        <Routes>
          <Route path="/add" element={<div data-testid="add-hub">Add hub</div>} />
          <Route path="/add/upload" element={<AddUpload />} />
          <Route path="/add/review" element={<div data-testid="review">Review</div>} />
        </Routes>
      </MemoryRouter>
    </CaptureDraftProvider>
  )
}

describe('AddUpload Back', () => {
  it('navigates to /add when history has no prior entry', async () => {
    const user = userEvent.setup()
    renderUpload(['/add/upload'])
    await user.click(screen.getByRole('button', { name: /^back$/i }))
    expect(screen.getByTestId('add-hub')).toBeInTheDocument()
  })

  it('navigates to previous route when stack has history', async () => {
    const user = userEvent.setup()
    renderUpload(['/add', '/add/upload'], 1)
    await user.click(screen.getByRole('button', { name: /^back$/i }))
    expect(screen.getByTestId('add-hub')).toBeInTheDocument()
  })
})

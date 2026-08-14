import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from './features/auth'
import App from './App'
import { ServerUnreachableError } from './lib/errors'
import * as authApi from './features/auth/services/authApi'

vi.mock('./features/auth/services/authApi', async () => {
  const actual = await vi.importActual<typeof import('./features/auth/services/authApi')>(
    './features/auth/services/authApi'
  )
  return {
    ...actual,
    fetchMeWithTimeout: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  }
})

const mockedAuth = vi.mocked(authApi)

function renderApp(path = '/') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <App />
      </AuthProvider>
    </MemoryRouter>
  )
}

describe('Server unreachable gate', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows Cannot reach Sonix instead of login when /api/me cannot connect', async () => {
    mockedAuth.fetchMeWithTimeout.mockResolvedValue({ user: null, unreachable: true })
    renderApp('/login')
    expect(await screen.findByRole('heading', { name: /Cannot reach Sonix/i })).toBeInTheDocument()
    expect(screen.queryByLabelText(/Username/i)).not.toBeInTheDocument()
  })

  it('retries the connection probe from the unreachable screen', async () => {
    mockedAuth.fetchMeWithTimeout
      .mockResolvedValueOnce({ user: null, unreachable: true })
      .mockResolvedValueOnce({ user: null, unreachable: false })
    renderApp('/')
    expect(await screen.findByRole('heading', { name: /Cannot reach Sonix/i })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Try again/i }))
    await waitFor(() => {
      expect(mockedAuth.fetchMeWithTimeout).toHaveBeenCalledTimes(2)
    })
    expect(await screen.findByLabelText(/Username/i)).toBeInTheDocument()
  })

  it('does not treat a normal unauthenticated probe as unreachable', async () => {
    mockedAuth.fetchMeWithTimeout.mockResolvedValue({ user: null, unreachable: false })
    renderApp('/login')
    expect(await screen.findByLabelText(/Username/i)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /Cannot reach Sonix/i })).not.toBeInTheDocument()
  })
})

describe('ServerUnreachableError helper', () => {
  it('is identifiable by name', () => {
    const err = new ServerUnreachableError()
    expect(err.name).toBe('ServerUnreachableError')
    expect(err.message).toMatch(/Cannot reach Sonix/i)
  })
})

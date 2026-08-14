import { describe, it, expect, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import Layout from './Layout'

vi.mock('../features/documents/services/documentsApi', () => ({
  documentsApi: {
    list: vi.fn().mockResolvedValue({ documents: [], total: 0 }),
  },
}))

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<div data-testid="home">home</div>} />
          <Route path="add" element={<div data-testid="add">add</div>} />
          <Route path="add/camera" element={<div data-testid="camera">camera</div>} />
          <Route path="explore" element={<div data-testid="explore">explore</div>} />
          <Route path="explore/no-date" element={<div data-testid="no-date">no date</div>} />
          <Route path="explore/:year" element={<div data-testid="folder">folder</div>} />
          <Route path="settings" element={<div data-testid="settings">settings</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  )
}

describe('Layout /add/camera', () => {
  it('hides sidebar and bottom tab nav for immersive camera', () => {
    renderAt('/add/camera')
    expect(screen.getByTestId('camera')).toBeInTheDocument()
    expect(document.querySelector('aside')?.className).toBe('hidden')
    const bottomNav = screen.queryByRole('navigation', { name: 'Primary' })
    expect(bottomNav).toHaveClass('hidden')
  })

  it('renders mobile tab bar on /add (fixed bottom nav, not camera-only hidden)', () => {
    renderAt('/add')
    expect(screen.getByTestId('add')).toBeInTheDocument()
    const bottomNav = document.querySelector('nav[aria-label="Primary"]')
    expect(bottomNav?.className).toContain('fixed')
    expect(bottomNav?.className).toContain('bottom-0')
    // Camera route uses className "hidden" only; /add uses the full fixed tab bar (md:hidden for desktop CSS)
    expect(bottomNav?.className).not.toBe('hidden')
  })

  it('does not apply mobile main top padding on camera route', () => {
    renderAt('/add/camera')
    const main = document.querySelector('main')
    expect(main?.className).not.toContain('pt-12')
    expect(main?.className).not.toContain('pt-16')
  })

  it('shows My letters on the right of the mobile top strip', () => {
    renderAt('/')
    const top = document.querySelector('header.md\\:hidden') ?? document.querySelectorAll('header')[0]
    expect(top?.textContent).toMatch(/Sonix/)
    expect(top?.textContent).toMatch(/My letters/)
  })

  it('names the Explore index and each folder on the mobile top strip', () => {
    const topText = () =>
      document.querySelector('header.md\\:hidden')?.textContent ??
      document.querySelectorAll('header')[0]?.textContent ??
      ''

    const index = renderAt('/explore')
    expect(topText()).toMatch(/Explore/)
    index.unmount()

    const folder = renderAt('/explore/2024')
    expect(topText()).toMatch(/2024/)
    folder.unmount()

    renderAt('/explore/no-date')
    expect(topText()).toMatch(/No date/)
  })
})

describe('Layout navigation order', () => {
  it('orders mobile tabs My letters, Explore, Scan, Settings', () => {
    renderAt('/')
    const bottomNav = screen.getByRole('navigation', { name: 'Primary' })
    const labels = within(bottomNav)
      .getAllByRole('link')
      .map((el) => el.textContent?.replace(/\s+/g, ' ').trim())
    expect(labels).toEqual(['My letters', 'Explore', 'Scan', 'Settings'])
  })

  it('orders desktop sidebar My letters, Explore, Scan letters, Settings', () => {
    renderAt('/')
    const sideNav = screen.getByRole('navigation', { name: 'Main' })
    const labels = within(sideNav)
      .getAllByRole('link')
      .map((el) => el.textContent?.trim())
    expect(labels).toEqual(['My letters', 'Explore', 'Scan letters', 'Settings'])
  })
  it('shows queue badge on My letters when queue total is positive', async () => {
    const { documentsApi } = await import('../features/documents/services/documentsApi')
    vi.mocked(documentsApi.list).mockResolvedValue({ documents: [], total: 4 })
    renderAt('/')
    const bottomNav = screen.getByRole('navigation', { name: 'Primary' })
    expect(
      await within(bottomNav).findByRole('link', { name: 'My letters, 4 in queue' })
    ).toBeInTheDocument()
    const sideNav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(sideNav).getByRole('link', { name: 'My letters, 4 in queue' })).toBeInTheDocument()
  })
})

describe('Layout Scan tab active on /add prefix', () => {
  it('highlights the Scan circle on /add', () => {
    renderAt('/add')
    const bottomNav = screen.getByRole('navigation', { name: 'Primary' })
    const scan = within(bottomNav).getByRole('link', { name: 'Scan' })
    const circle = scan.querySelector('span.rounded-full')
    expect(circle?.className).toContain('bg-accent')
    expect(circle?.className).toContain('text-white')
  })
})

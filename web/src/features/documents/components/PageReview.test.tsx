import { useEffect, useState, type ReactNode } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { CaptureDraftProvider, useCaptureDraft } from '../hooks/CaptureDraftContext'
import PageReview from './PageReview'
import { draftPagesToFiles, reorderList, blobWithRotation } from '../lib/captureDraft'
import { documentsApi } from '../services/documentsApi'

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    create: vi.fn(() => Promise.resolve({ id: 99 })),
    uploadPages: vi.fn(() => Promise.resolve({ ok: true, document_id: 99 })),
  },
}))

function SeedPages({ count, children }: { count: number; children: ReactNode }) {
  const draft = useCaptureDraft()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      draft.clear()
      draft.setTitle('Test doc')
      for (let i = 0; i < count; i++) {
        draft.addBlob(new Blob([`page-${i}`], { type: 'image/jpeg' }), 'camera')
      }
      if (!cancelled) setReady(true)
    })()
    return () => {
      cancelled = true
    }
    // Seed once on mount for this test tree.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional mount-only seed
  }, [])

  if (!ready) return <div data-testid="seeding">seeding</div>
  return <>{children}</>
}

function renderReview(pageCount = 3) {
  return render(
    <CaptureDraftProvider>
      <MemoryRouter initialEntries={['/add/review']}>
        <Routes>
          <Route
            path="/add/review"
            element={
              <SeedPages count={pageCount}>
                <PageReview />
              </SeedPages>
            }
          />
          <Route path="/add/camera" element={<div data-testid="camera">camera</div>} />
          <Route path="/add" element={<div data-testid="add-hub">add</div>} />
          <Route path="/documents/:id" element={<div data-testid="doc">doc</div>} />
        </Routes>
      </MemoryRouter>
    </CaptureDraftProvider>
  )
}

describe('captureDraft helpers', () => {
  it('reorderList moves items and is a no-op for bad indexes', () => {
    expect(reorderList(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a'])
    expect(reorderList(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b'])
    expect(reorderList(['a', 'b'], 0, 0)).toEqual(['a', 'b'])
    expect(reorderList(['a', 'b'], -1, 1)).toEqual(['a', 'b'])
  })

  it('blobWithRotation passthrough at 0 degrees', async () => {
    const blob = new Blob(['x'], { type: 'image/jpeg' })
    await expect(blobWithRotation(blob, 0)).resolves.toBe(blob)
  })

  it('draftPagesToFiles preserves order without rotation', async () => {
    const pages = [
      {
        id: '1',
        blob: new Blob(['a'], { type: 'image/jpeg' }),
        url: 'blob:1',
        rotation: 0 as const,
        source: 'camera' as const,
      },
      {
        id: '2',
        blob: new Blob(['b'], { type: 'image/jpeg' }),
        url: 'blob:2',
        rotation: 0 as const,
        source: 'camera' as const,
      },
    ]
    const files = await draftPagesToFiles(pages)
    expect(files).toHaveLength(2)
    expect(files[0].name).toBe('page_0.jpg')
    expect(files[1].name).toBe('page_1.jpg')
  })
})

describe('PageReview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows page tiles with move and rotate controls', async () => {
    renderReview(3)
    expect(await screen.findByRole('heading', { name: 'Review pages' })).toBeInTheDocument()
    expect(screen.getByLabelText('Page 1, selected')).toBeInTheDocument()
    expect(screen.getByLabelText('Move page 1 left')).toBeDisabled()
    expect(screen.getByLabelText('Move page 3 right')).toBeDisabled()
    await userEvent.click(screen.getByLabelText('Move page 1 right'))
    expect(screen.getByLabelText('Page 1, selected')).toBeInTheDocument()
  })

  it('rotate control advances rotation on the selected tile image', async () => {
    renderReview(1)
    const img = await screen.findByRole('img', { name: 'Page 1' })
    expect(img).toHaveStyle({ transform: 'rotate(0deg)' })
    await userEvent.click(screen.getByLabelText('Rotate page 1'))
    expect(img).toHaveStyle({ transform: 'rotate(90deg)' })
  })

  it('shows Crop alongside Add more and Save', async () => {
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    expect(screen.getByRole('button', { name: 'Crop' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Colour' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add more' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retake' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('Colour opens mode sheet with Apply to all when multiple pages', async () => {
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Colour' }))
    expect(await screen.findByRole('dialog', { name: 'Colour mode' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Colour mode' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Apply to all pages' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Done' })).toBeInTheDocument()
  })

  it('deleting the last page returns to camera', async () => {
    renderReview(1)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(await screen.findByTestId('camera')).toBeInTheDocument()
  })

  it('Save uploads pages in order with per-page API calls (pass-through)', async () => {
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByTestId('doc')).toBeInTheDocument()
    expect(documentsApi.create).toHaveBeenCalledWith('Test doc')
    expect(documentsApi.uploadPages).toHaveBeenCalledTimes(2)
    expect(documentsApi.uploadPages).toHaveBeenNthCalledWith(1, 99, [
      expect.objectContaining({ name: 'page_0.jpg' }),
    ])
    expect(documentsApi.uploadPages).toHaveBeenNthCalledWith(2, 99, [
      expect.objectContaining({ name: 'page_1.jpg' }),
    ])
  })

  it('upload failure shows in-app error naming the page', async () => {
    vi.mocked(documentsApi.uploadPages)
      .mockResolvedValueOnce({ ok: true, document_id: 99 })
      .mockRejectedValueOnce(new Error('network'))
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/Page 2 of 2 failed/i)
  })

  it('Cancel asks to confirm before discarding', async () => {
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(await screen.findByRole('dialog', { name: /Discard this scan/i })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.queryByRole('dialog', { name: /Discard this scan/i })).toBeNull()
    expect(screen.getByRole('heading', { name: 'Review pages' })).toBeInTheDocument()
  })

  it('Discard confirm clears draft and returns to add hub', async () => {
    renderReview(2)
    await screen.findByRole('heading', { name: 'Review pages' })
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Discard' }))
    expect(screen.getByTestId('add-hub')).toBeInTheDocument()
  })
})

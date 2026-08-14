import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import type { DocumentDetail } from '../types/document'
import DocumentDetailPage from './DocumentDetail'

vi.mock('../hooks/useDocument', () => ({
  useDocument: vi.fn(),
}))

vi.mock('../hooks/useDocumentMutations', () => ({
  useDocumentMutations: vi.fn(),
}))

vi.mock('../services/documentsApi', () => ({
  documentsApi: {
    text: vi.fn(() => Promise.resolve('Hello **world**')),
    pageImageUrl: (id: number, i: number) => `https://example.test/${id}/${i}.jpg`,
  },
}))

vi.mock('../../settings/services/settingsApi', () => ({
  settingsApi: {
    get: vi.fn(() => Promise.resolve({ import_extract_use_ocr: true })),
  },
}))

vi.mock('../../../auth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}))

vi.mock('../../../components/MarkdownText', () => ({
  default: ({ text }: { text: string }) => <div data-testid="markdown">{text}</div>,
}))

import { useDocument } from '../hooks/useDocument'
import { useDocumentMutations } from '../hooks/useDocumentMutations'
import { documentsApi } from '../services/documentsApi'

const mockedDoc = vi.mocked(useDocument)
const mockedMut = vi.mocked(useDocumentMutations)

function baseDoc(overrides: Partial<DocumentDetail> = {}): DocumentDetail {
  return {
    id: 42,
    title: 'Invoice ACME',
    status: 'ready',
    created_at: '2024-06-01T12:00:00Z',
    updated_at: '2024-06-01T12:00:00Z',
    pages: [
      { page_index: 0, content_type: 'image/jpeg' },
      { page_index: 1, content_type: 'image/jpeg' },
    ],
    page_count: 2,
    thumbnail_available: true,
    extraction: {
      tags: ['invoice'],
      summary: 'Payment due next week.',
      document_date: '2024-05-20',
      extracted_at: '2024-06-01T12:05:00Z',
      engine_id: 'vision:unified-vision-v1',
      prompt_version: 'v1',
      extraction_wall_ms: 1200,
    },
    ...overrides,
  }
}

function mutationDefaults(overrides: Record<string, unknown> = {}) {
  return {
    extracting: false,
    deleting: false,
    savingTags: false,
    savingDate: false,
    savingTitle: false,
    deleteDocument: vi.fn(),
    putTags: vi.fn(() => Promise.resolve()),
    putDocumentDate: vi.fn(() => Promise.resolve()),
    putTitle: vi.fn(() => Promise.resolve()),
    startExtract: vi.fn(() => Promise.resolve()),
    resetExtraction: vi.fn(),
    pageImageUrl: (i: number) => `https://example.test/page/${i}.jpg`,
    ...overrides,
  }
}

function renderDetail() {
  return render(
    <MemoryRouter initialEntries={['/documents/42']}>
      <Routes>
        <Route path="/documents/:id" element={<DocumentDetailPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('DocumentDetail characterisation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedDoc.mockReturnValue({
      doc: baseDoc(),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    mockedMut.mockReturnValue(mutationDefaults() as ReturnType<typeof useDocumentMutations>)
  })

  it('ready: title, summary, date, tags, re-process, text sections, engine meta', () => {
    renderDetail()
    expect(screen.getByRole('button', { name: 'Invoice ACME' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rename' })).toBeInTheDocument()
    expect(screen.getAllByText('ready').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByRole('heading', { name: 'Document date' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('2024-05-20')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Tags' })).toBeInTheDocument()
    expect(screen.getByText('invoice')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Summary' })).toBeInTheDocument()
    expect(screen.getByText('Payment due next week.')).toBeInTheDocument()
    expect(screen.getByText(/vision:unified-vision-v1/i)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Re-process' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Re-process document' })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /Extraction mode/i })).toHaveValue('llm')
    expect(screen.getByRole('button', { name: 'View translation' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'View original text' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Full text' })).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Delete' }).length).toBeGreaterThanOrEqual(1)
  })

  it('pending: Extract now with mode select, no summary/date/re-process', async () => {
    mockedDoc.mockReturnValue({
      doc: baseDoc({ status: 'pending', extraction: undefined }),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    renderDetail()
    expect(screen.getByRole('button', { name: 'Extract now' })).toBeInTheDocument()
    const mode = await screen.findByRole('combobox', { name: /Extraction mode/i })
    expect(mode).toHaveValue('ocr')
    expect(screen.queryByRole('heading', { name: 'Summary' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Document date' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Re-process' })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Tags' })).toBeInTheDocument()
  })

  it('failed: short error, mode select, Retry extraction', async () => {
    const startExtract = vi.fn(() => Promise.resolve())
    const raw =
      'LLM vision: ollama 400 Bad Request: {"error":{"message":"request (4452 tokens) exceeds the available context size (4096 tokens)","type":"exceed_context_size_error"}}'
    mockedDoc.mockReturnValue({
      doc: baseDoc({
        status: 'failed',
        extraction_error: raw,
        extraction: undefined,
      }),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    mockedMut.mockReturnValue(
      mutationDefaults({ startExtract }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    expect(screen.getByText(/too large for the AI model/i)).toBeInTheDocument()
    expect(screen.queryByText('More details')).not.toBeInTheDocument()
    expect(screen.queryByText(/4452 tokens/i)).not.toBeInTheDocument()
    expect(await screen.findByRole('combobox', { name: /Extraction mode/i })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Retry extraction' }))
    expect(startExtract).toHaveBeenCalled()
  })

  it('processing: progress label and Cancel extraction', async () => {
    const resetExtraction = vi.fn()
    mockedDoc.mockReturnValue({
      doc: baseDoc({
        status: 'processing',
        extraction: undefined,
        extraction_pages_done: 1,
        extraction_pages_total: 2,
      }),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    mockedMut.mockReturnValue(
      mutationDefaults({ resetExtraction }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    expect(screen.getByText(/Pages 1 \/ 2 complete/i)).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-live', 'polite')
    await userEvent.click(screen.getByRole('button', { name: 'Cancel extraction' }))
    expect(resetExtraction).toHaveBeenCalled()
  })

  it('partial: shows original-saved message and Retry extraction', async () => {
    const startExtract = vi.fn(() => Promise.resolve())
    mockedDoc.mockReturnValue({
      doc: baseDoc({
        status: 'partial',
        extraction_error: 'LLM metadata: context canceled',
        extraction: {
          summary: '',
          document_date: '',
          tags: [],
          engine_id: 'vision:unified-vision-v1',
          prompt_version: '',
        },
      }),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    mockedMut.mockReturnValue(
      mutationDefaults({ startExtract }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    expect(screen.getByRole('heading', { name: /Partially extracted/i })).toBeInTheDocument()
    expect(screen.getByText(/Original text was saved/i)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Retry extraction' }))
    expect(startExtract).toHaveBeenCalled()
  })

  it('title: click opens edit; Enter with change asks to save', async () => {
    const putTitle = vi.fn(() => Promise.resolve())
    mockedMut.mockReturnValue(
      mutationDefaults({ putTitle }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Invoice ACME' }))
    const input = screen.getByLabelText('Document name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Renamed')
    await userEvent.keyboard('{Enter}')
    expect(await screen.findByRole('dialog', { name: /Save new name/i })).toBeInTheDocument()
    expect(putTitle).not.toHaveBeenCalled()
    const dialog = screen.getByRole('dialog', { name: /Save new name/i })
    expect(within(dialog).getByText(/Rename this document to “Renamed”/i)).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(putTitle).toHaveBeenCalledWith('Renamed')
  })

  it('title: Escape cancels edit without saving', async () => {
    const putTitle = vi.fn(() => Promise.resolve())
    mockedMut.mockReturnValue(
      mutationDefaults({ putTitle }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Invoice ACME' }))
    expect(screen.getByLabelText('Document name')).toBeInTheDocument()
    await userEvent.keyboard('{Escape}')
    expect(putTitle).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Invoice ACME' })).toBeInTheDocument()
  })

  it('title: Rename button opens edit (desktop affordance)', async () => {
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Rename' }))
    expect(screen.getByLabelText('Document name')).toBeInTheDocument()
  })

  it('title: blur with change shows dialog; Don’t save discards', async () => {
    const putTitle = vi.fn(() => Promise.resolve())
    mockedMut.mockReturnValue(
      mutationDefaults({ putTitle }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Invoice ACME' }))
    const input = screen.getByLabelText('Document name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Temp name')
    await userEvent.tab()
    const dialog = await screen.findByRole('dialog', { name: /Save new name/i })
    await userEvent.click(within(dialog).getByRole('button', { name: /Don't save/i }))
    expect(putTitle).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Invoice ACME' })).toBeInTheDocument()
  })

  it('tags: Enter and Add call putTags; remove calls putTags without tag', async () => {
    const putTags = vi.fn(() => Promise.resolve())
    mockedMut.mockReturnValue(
      mutationDefaults({ putTags }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    const tagField = screen.getByPlaceholderText('Add tag…')
    await userEvent.type(tagField, 'urgent{Enter}')
    expect(putTags).toHaveBeenCalledWith(['invoice', 'urgent'])

    await userEvent.click(screen.getByRole('button', { name: 'Remove invoice' }))
    expect(putTags).toHaveBeenCalledWith([])
  })

  it('document date Save calls putDocumentDate', async () => {
    const putDocumentDate = vi.fn(() => Promise.resolve())
    mockedMut.mockReturnValue(
      mutationDefaults({ putDocumentDate }) as ReturnType<typeof useDocumentMutations>
    )
    renderDetail()
    const dateSection = screen.getByRole('heading', { name: 'Document date' }).closest('section')!
    await userEvent.click(within(dateSection).getByRole('button', { name: 'Save' }))
    expect(putDocumentDate).toHaveBeenCalledWith('2024-05-20')
  })

  it('desktop View translation opens scrollable modal; Close dismisses', async () => {
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'View translation' }))
    const dialog = await screen.findByRole('dialog', { name: /Translation \(English\)/i })
    expect(dialog).toBeInTheDocument()
    expect(await within(dialog).findByTestId('markdown')).toHaveTextContent('Hello **world**')
    expect(documentsApi.text).toHaveBeenCalledWith(42, 'english')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog', { name: /Translation \(English\)/i })).not.toBeInTheDocument()
  })

  it('extraction incomplete banner when ready without summary', () => {
    mockedDoc.mockReturnValue({
      doc: baseDoc({
        status: 'ready',
        extraction: {
          tags: [],
          summary: '',
          document_date: '2024-01-01',
          extracted_at: '2024-06-01T12:05:00Z',
        },
      }),
      setDoc: vi.fn(),
      refresh: vi.fn(),
      loading: false,
      loadError: false,
    } as ReturnType<typeof useDocument>)
    renderDetail()
    expect(screen.getByText('Extraction incomplete')).toBeInTheDocument()
    expect(screen.getByText(/Summary is missing/i)).toBeInTheDocument()
  })

  it('load error shows Try again', async () => {
    const refresh = vi.fn()
    mockedDoc.mockReturnValue({
      doc: null,
      setDoc: vi.fn(),
      refresh,
      loading: false,
      loadError: true,
    } as ReturnType<typeof useDocument>)
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(refresh).toHaveBeenCalled()
  })

  it('selecting a page thumbnail updates the indicator and main image', async () => {
    renderDetail()
    expect(screen.getByText('1 of 2')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Page 2' }))
    expect(screen.getByText('2 of 2')).toBeInTheDocument()
    expect(screen.getAllByRole('img', { name: 'Page 2' }).length).toBeGreaterThanOrEqual(1)
  })

  it('full screen opens and Close returns', async () => {
    renderDetail()
    await userEvent.click(screen.getByRole('button', { name: 'Full screen' }))
    expect(screen.getByRole('dialog', { name: /Page 1 full screen/i })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Close full screen' }))
    expect(screen.queryByRole('dialog', { name: /full screen/i })).not.toBeInTheDocument()
  })

  it('summary Copy uses clipboard', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    Object.assign(navigator, { clipboard: { writeText } })
    renderDetail()
    const summarySection = screen.getByRole('heading', { name: 'Summary' }).closest('section')!
    await userEvent.click(within(summarySection).getByRole('button', { name: 'Copy' }))
    expect(writeText).toHaveBeenCalledWith('Payment due next week.')
    expect(await within(summarySection).findByRole('button', { name: 'Copied' })).toBeInTheDocument()
  })
})

import { apiRequest, documentsApiBase } from './http'
import { isApiMockActive } from '../../../lib/apiMock'
import type {
  DocumentDateYearsResponse,
  DocumentDetail,
  DocumentGetOptions,
  DocumentListParams,
  DocumentListResponse,
} from '../types/document'

/** 1×1 PNG — mock page images never hit `/api`. */
const MOCK_PAGE_PIXEL =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='

/** HTTP integration with Go document/extraction endpoints. */
export const documentsApi = {
  list: (params?: DocumentListParams) => {
    const sp = new URLSearchParams()
    if (params)
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== '') sp.set(k, String(v))
      })
    return apiRequest<DocumentListResponse>(`/documents?${sp}`)
  },
  years: () => apiRequest<{ years: string[] }>('/documents/years'),
  tags: () => apiRequest<{ tags: string[] }>('/documents/tags'),
  documentDateYears: () =>
    apiRequest<DocumentDateYearsResponse>('/documents/document-date-years'),
  get: (id: number, opts?: DocumentGetOptions) => {
    const sp = new URLSearchParams()
    if (opts?.include) sp.set('include', opts.include)
    const q = sp.toString()
    return apiRequest<DocumentDetail>(`/documents/${id}${q ? `?${q}` : ''}`)
  },
  delete: (id: number) => apiRequest<void>(`/documents/${id}`, { method: 'DELETE' }),
  create: (title?: string) =>
    apiRequest<{ id: number }>('/documents', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: title || '' }),
    }),
  putTitle: (id: number, title: string) =>
    apiRequest<{ title: string }>(`/documents/${id}/title`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title }),
    }),
  uploadPages: (id: number, files: File[]) => {
    const fd = new FormData()
    files.forEach((f) => fd.append('files', f))
    return apiRequest<{ ok: boolean; document_id: number }>(`/documents/${id}/pages`, {
      method: 'POST',
      body: fd,
    })
  },
  extract: (id: number, opts?: { use_ocr?: boolean }) =>
    apiRequest<{ status: string }>(`/documents/${id}/extract`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: opts?.use_ocr ? JSON.stringify({ use_ocr: true }) : undefined,
    }),
  resetExtraction: (id: number) =>
    apiRequest<{ status: string }>(`/documents/${id}/reset-extraction`, { method: 'POST' }),
  status: (id: number) => apiRequest<{ status: string }>(`/documents/${id}/status`),
  pageImageUrl: (id: number, pageIndex: number) =>
    isApiMockActive() ? MOCK_PAGE_PIXEL : `${documentsApiBase}/documents/${id}/pages/${pageIndex}/image`,
  pageThumbnailUrl: (id: number, pageIndex: number) =>
    isApiMockActive()
      ? MOCK_PAGE_PIXEL
      : `${documentsApiBase}/documents/${id}/pages/${pageIndex}/thumbnail`,
  rotatePage: (id: number, pageIndex: number, degrees: 90 | 180 | 270) =>
    apiRequest<{ ok: boolean }>(`/documents/${id}/pages/${pageIndex}/rotate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ degrees }),
    }),
  text: (id: number, lang: 'original' | 'english') =>
    apiRequest<string>(`/documents/${id}/text?lang=${lang}`),
  putTags: (id: number, tags: string[]) =>
    apiRequest<{ tags: string[] }>(`/documents/${id}/tags`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tags }),
    }),
  putDocumentDate: (id: number, documentDate: string | null) =>
    apiRequest<{ document_date: string | null }>(`/documents/${id}/document_date`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ document_date: documentDate }),
    }),
}

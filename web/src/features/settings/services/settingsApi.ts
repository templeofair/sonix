import type { Settings } from '../types'
import { toServerUnreachable } from '../../../lib/errors'
import { isApiMockActive, runApiMock } from '../../../lib/apiMock'

const API = '/api'

async function api<T>(path: string, opts?: RequestInit): Promise<T> {
  if (isApiMockActive()) {
    return runApiMock<T>(path, opts)
  }
  let res: Response
  try {
    res = await fetch(API + path, { credentials: 'include', ...opts })
  } catch (err) {
    throw toServerUnreachable(err)
  }
  if (!res.ok) throw new Error(await res.text().catch(() => res.statusText))
  const ct = res.headers.get('content-type')
  if (ct?.includes('application/json')) return res.json()
  return res.text() as Promise<T>
}

export function exportUrl(params?: Record<string, string>): string {
  if (isApiMockActive()) {
    return 'data:application/json,' + encodeURIComponent(JSON.stringify({ sonix_mock_export: true }))
  }
  if (!params || Object.keys(params).length === 0) return `${API}/export`
  const sp = new URLSearchParams(params)
  return `${API}/export?${sp}`
}

export const settingsApi = {
  get: () => api<Settings>('/settings'),
  put: (body: {
    ollama_base_url?: string
    ollama_model?: string
    ollama_text_model?: string
    import_inbox_enabled?: boolean
    import_auto_extract?: boolean
    import_extract_use_ocr?: boolean
    hp_printer_ip?: string
  }) =>
    api<Settings>('/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  testOllama: () => api<{ ok: boolean; error?: string }>('/settings/ollama/test', { method: 'POST' }),
  testPrinter: () =>
    api<{ ok: boolean; error?: string }>('/settings/printer/test', {
      method: 'POST',
    }),
}

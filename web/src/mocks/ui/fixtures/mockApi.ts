import type { DocumentDetail, DocumentListItem } from '../../../features/documents/types/document'
import type { Settings } from '../../../features/settings/types'
import type { ApiMockHandler } from '../../../lib/apiMock'

const QUEUE = new Set(['pending', 'processing', 'failed', 'partial'])

function seedList(): DocumentListItem[] {
  const base = {
    created_at: '2026-07-15T10:00:00Z',
    updated_at: '2026-07-20T12:00:00Z',
    thumbnail_available: true,
  }
  return [
    {
      ...base,
      id: 1,
      title: 'Krankenkasse — Jahresabrechnung',
      status: 'ready',
      document_date: '2026-06-01',
      page_count: 2,
    },
    {
      ...base,
      id: 2,
      title: 'Stadtwerke — Rechnung',
      status: 'pending',
      document_date: '2026-07-10',
      page_count: 1,
    },
    {
      ...base,
      id: 3,
      title: 'Scan 2026-07-20',
      status: 'processing',
      page_count: 3,
    },
    {
      ...base,
      id: 4,
      title: 'Failed extract',
      status: 'failed',
      page_count: 1,
    },
    {
      ...base,
      id: 5,
      title: 'Partial — original only',
      status: 'partial',
      document_date: '2026-05-20',
      page_count: 2,
    },
    {
      ...base,
      id: 6,
      created_at: '2025-11-02T09:00:00Z',
      title: 'Bürgeramt — Bescheid 2025',
      status: 'ready',
      document_date: '2025-10-28',
      page_count: 4,
    },
  ]
}

function seedDetail(listItem: DocumentListItem): DocumentDetail {
  const pages = Array.from({ length: Math.max(1, listItem.page_count) }, (_, i) => ({
    page_index: i,
    content_type: 'image/jpeg',
  }))
  const ready = listItem.status === 'ready' || listItem.status === 'partial'
  return {
    id: listItem.id,
    title: listItem.title,
    status: listItem.status,
    created_at: listItem.created_at,
    updated_at: listItem.updated_at,
    extraction_error: listItem.status === 'failed' ? 'Mock extraction error (safe)' : undefined,
    extraction_pages_done: listItem.status === 'processing' ? 1 : undefined,
    extraction_pages_total: listItem.status === 'processing' ? listItem.page_count : undefined,
    pages,
    page_count: listItem.page_count,
    thumbnail_available: true,
    extraction: ready
      ? {
          tags: ['mock', 'letter'],
          summary: 'Mock summary for UX review. No real Ollama call.',
          document_date: listItem.document_date,
          extracted_at: listItem.updated_at,
          engine_id: 'mock-engine',
          prompt_version: 'mock-1',
          full_text_original: 'Original mock text — Deutsch Beispiel.',
          full_text_english: 'English mock translation for the demo letter.',
        }
      : undefined,
  }
}

function seedSettings(): Settings {
  return {
    ollama_base_url: 'http://127.0.0.1:11434',
    ollama_model: 'llama3.2-vision',
    ollama_text_model: 'llama3.2',
    import_inbox_enabled: false,
    import_auto_extract: false,
    import_extract_use_ocr: false,
    hp_printer_ip: '',
  }
}

/** In-memory store for `/__ui` — reset on each MockApp mount via createMockApiHandler(). */
export function createMockApiHandler(): ApiMockHandler {
  let list = seedList()
  const details = new Map<number, DocumentDetail>()
  for (const item of list) {
    details.set(item.id, seedDetail(item))
  }
  let settings = seedSettings()
  let nextId = 100

  const syncListFromDetail = (d: DocumentDetail) => {
    list = list.map((row) =>
      row.id === d.id
        ? {
            ...row,
            title: d.title,
            status: d.status,
            updated_at: d.updated_at,
            document_date: d.extraction?.document_date ?? row.document_date,
            page_count: d.page_count,
            thumbnail_available: d.thumbnail_available,
          }
        : row,
    )
  }

  return async (path: string, opts?: RequestInit) => {
    const method = (opts?.method || 'GET').toUpperCase()
    const url = new URL(path, 'http://mock.local')
    const pathname = url.pathname
    const bodyText = typeof opts?.body === 'string' ? opts.body : undefined
    const json = bodyText ? (JSON.parse(bodyText) as Record<string, unknown>) : {}

    // Settings
    if (pathname === '/settings' && method === 'GET') return { ...settings }
    if (pathname === '/settings' && method === 'PUT') {
      settings = { ...settings, ...json } as Settings
      return { ...settings }
    }
    if (pathname === '/settings/ollama/test' && method === 'POST') {
      return { ok: true }
    }
    if (pathname === '/settings/printer/test' && method === 'POST') {
      return { ok: true }
    }

    if (pathname === '/documents/years' && method === 'GET') {
      return { years: ['2026', '2025'] }
    }

    if (pathname === '/documents/tags' && method === 'GET') {
      const set = new Set<string>()
      for (const d of details.values()) {
        for (const t of d.extraction?.tags ?? []) set.add(t)
      }
      return { tags: Array.from(set).sort((a, b) => a.localeCompare(b)) }
    }

    if (pathname === '/documents/document-date-years' && method === 'GET') {
      const counts = new Map<string, number>()
      let undatedCount = 0
      for (const d of list) {
        const y = d.document_date?.slice(0, 4)
        if (!y) undatedCount += 1
        else counts.set(y, (counts.get(y) ?? 0) + 1)
      }
      const years = Array.from(counts.entries())
        .map(([year, count]) => ({ year, count }))
        .sort((a, b) => b.year.localeCompare(a.year))
      return { years, undated_count: undatedCount }
    }

    if (pathname === '/documents' && method === 'GET') {
      let docs = [...list]
      const q = url.searchParams.get('q')?.toLowerCase()
      const status = url.searchParams.get('status')
      const tag = url.searchParams.get('tag')
      const yearParam = url.searchParams.get('year')
      const yearFromCreated = url.searchParams.get('created_from')?.slice(0, 4)
      if (q) docs = docs.filter((d) => (d.title || '').toLowerCase().includes(q))
      if (status) {
        const set = new Set(status.split(',').map((s) => s.trim()).filter(Boolean))
        docs = docs.filter((d) => set.has(d.status))
      }
      if (tag) {
        const wanted = new Set(tag.split(',').map((s) => s.trim()).filter(Boolean))
        docs = docs.filter((d) => {
          const det = details.get(d.id)
          return (det?.extraction?.tags ?? []).some((t) => wanted.has(t))
        })
      }
      const years = (yearParam || yearFromCreated || '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (years.length > 0) {
        docs = docs.filter((d) => years.some((y) => d.created_at.startsWith(y)))
      }
      const dateFrom = url.searchParams.get('document_date_from')
      const dateTo = url.searchParams.get('document_date_to')
      if (dateFrom) docs = docs.filter((d) => (d.document_date || '') >= dateFrom)
      if (dateTo) docs = docs.filter((d) => (d.document_date || '9999-12-31') <= dateTo)
      if (url.searchParams.get('undated') === '1') docs = docs.filter((d) => !d.document_date)
      const sort = url.searchParams.get('sort')
      if (sort === 'date_desc' || sort === 'date_asc') {
        const dir = sort === 'date_asc' ? 1 : -1
        // Undated rows sort last either way, like the repository does.
        docs = [...docs].sort((a, b) => {
          if (!a.document_date || !b.document_date) {
            return Number(Boolean(b.document_date)) - Number(Boolean(a.document_date))
          }
          return a.document_date.localeCompare(b.document_date) * dir
        })
      }
      const page = Math.max(0, Number(url.searchParams.get('page') || 0))
      const limit = Math.max(1, Number(url.searchParams.get('limit') || 20))
      const total = docs.length
      const slice = docs.slice(page * limit, page * limit + limit)
      return { documents: slice, total }
    }

    if (pathname === '/documents' && method === 'POST') {
      const id = nextId++
      const title = typeof json.title === 'string' ? json.title : ''
      const item: DocumentListItem = {
        id,
        title: title || `Mock document ${id}`,
        status: 'pending',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        page_count: 0,
        thumbnail_available: false,
      }
      list = [item, ...list]
      details.set(id, seedDetail({ ...item, page_count: 1 }))
      return { id }
    }

    const docMatch = pathname.match(/^\/documents\/(\d+)(.*)$/)
    if (docMatch) {
      const id = Number(docMatch[1])
      const rest = docMatch[2] || ''
      const detail = details.get(id)
      if (!detail && method !== 'POST') throw new Error('not found')

      if (rest === '' && method === 'GET') {
        const include = url.searchParams.get('include')
        const d = { ...details.get(id)! }
        if (include !== 'text' && d.extraction) {
          const { full_text_original: _o, full_text_english: _e, ...ex } = d.extraction
          d.extraction = ex
        }
        return d
      }

      if (rest === '' && method === 'DELETE') {
        list = list.filter((d) => d.id !== id)
        details.delete(id)
        return undefined
      }

      if (rest === '/title' && method === 'PUT') {
        const title = String(json.title ?? '')
        const d = details.get(id)!
        d.title = title
        d.updated_at = new Date().toISOString()
        syncListFromDetail(d)
        return { title }
      }

      if (rest === '/tags' && method === 'PUT') {
        const tags = Array.isArray(json.tags) ? (json.tags as string[]) : []
        const d = details.get(id)!
        if (!d.extraction) {
          d.extraction = {
            tags,
            summary: '',
            extracted_at: new Date().toISOString(),
          }
        } else {
          d.extraction = { ...d.extraction, tags }
        }
        syncListFromDetail(d)
        return { tags }
      }

      if (rest === '/document_date' && method === 'PUT') {
        const document_date = (json.document_date as string | null) ?? null
        const d = details.get(id)!
        if (!d.extraction) {
          d.extraction = {
            tags: [],
            summary: '',
            document_date: document_date || undefined,
            extracted_at: new Date().toISOString(),
          }
        } else {
          d.extraction = { ...d.extraction, document_date: document_date || undefined }
        }
        syncListFromDetail(d)
        return { document_date }
      }

      if (rest === '/extract' && method === 'POST') {
        const d = details.get(id)!
        d.status = 'ready'
        d.extraction = {
          tags: ['mock'],
          summary: 'Mock extraction finished (no Ollama).',
          document_date: d.extraction?.document_date || '2026-07-01',
          extracted_at: new Date().toISOString(),
          engine_id: 'mock-engine',
          full_text_original: 'Mock original after extract.',
          full_text_english: 'Mock English after extract.',
        }
        d.extraction_error = undefined
        syncListFromDetail(d)
        return { status: 'ready' }
      }

      if (rest === '/reset-extraction' && method === 'POST') {
        const d = details.get(id)!
        d.status = 'pending'
        d.extraction = undefined
        d.extraction_error = undefined
        syncListFromDetail(d)
        return { status: 'pending' }
      }

      if (rest === '/status' && method === 'GET') {
        return { status: details.get(id)!.status }
      }

      if (rest === '/pages' && method === 'POST') {
        const d = details.get(id)!
        const add = 1
        d.page_count += add
        d.pages = [
          ...d.pages,
          { page_index: d.pages.length, content_type: 'image/jpeg' },
        ]
        d.thumbnail_available = true
        d.updated_at = new Date().toISOString()
        syncListFromDetail(d)
        return { ok: true, document_id: id }
      }

      const rotate = rest.match(/^\/pages\/(\d+)\/rotate$/)
      if (rotate && method === 'POST') {
        return { ok: true }
      }

      const textMatch = rest.match(/^\/text$/)
      if (textMatch && method === 'GET') {
        const lang = url.searchParams.get('lang')
        const d = details.get(id)!
        if (lang === 'english') return d.extraction?.full_text_english || 'Mock English text'
        return d.extraction?.full_text_original || 'Mock original text'
      }
    }

    // Queue helper: any other path
    if (pathname.startsWith('/documents')) {
      throw new Error(`mock: unhandled ${method} ${pathname}`)
    }

    throw new Error(`mock: unhandled ${method} ${pathname}`)
  }
}

export function queueStatusTotal(handlerDocs?: DocumentListItem[]): number {
  const docs = handlerDocs ?? seedList()
  return docs.filter((d) => QUEUE.has(d.status)).length
}

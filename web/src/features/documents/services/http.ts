/** Shared fetch helper for documents feature → Go `/api`. */

import { toServerUnreachable } from '../../../lib/errors'
import { isApiMockActive, runApiMock } from '../../../lib/apiMock'

const API = '/api'

export async function apiRequest<T>(path: string, opts?: RequestInit): Promise<T> {
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

export { API as documentsApiBase }

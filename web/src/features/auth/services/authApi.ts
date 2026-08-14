import {
  isLikelyNetworkFailure,
  ServerUnreachableError,
  toServerUnreachable,
} from '../../../lib/errors'

const API = '/api'

const meFetchTimeoutMs = 12_000

export type AuthUser = { username: string } | null

export type MeProbeResult = {
  user: AuthUser
  unreachable: boolean
}

export async function fetchMe(signal?: AbortSignal): Promise<AuthUser> {
  let res: Response
  try {
    res = await fetch(`${API}/me`, {
      credentials: 'include',
      signal,
    })
  } catch (err) {
    throw toServerUnreachable(err)
  }
  if (res.ok) {
    const d = (await res.json()) as { username?: string }
    return d.username ? { username: d.username } : null
  }
  return null
}

export async function fetchMeWithTimeout(): Promise<MeProbeResult> {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), meFetchTimeoutMs)
  try {
    const user = await fetchMe(controller.signal)
    return { user, unreachable: false }
  } catch (err) {
    if (isLikelyNetworkFailure(err)) {
      return { user: null, unreachable: true }
    }
    return { user: null, unreachable: false }
  } finally {
    window.clearTimeout(timeoutId)
  }
}

export async function login(username: string, password: string): Promise<void> {
  let res: Response
  try {
    res = await fetch(`${API}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    })
  } catch (err) {
    throw toServerUnreachable(err)
  }
  if (!res.ok) {
    const t = await res.text()
    throw new Error(t || 'Login failed')
  }
}

export async function logout(): Promise<void> {
  try {
    await fetch(`${API}/logout`, { method: 'POST', credentials: 'include' })
  } catch (err) {
    throw toServerUnreachable(err)
  }
}

export { ServerUnreachableError }

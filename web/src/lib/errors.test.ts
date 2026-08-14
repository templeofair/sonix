import { describe, it, expect } from 'vitest'
import {
  ServerUnreachableError,
  isLikelyNetworkFailure,
  isServerUnreachableError,
  toServerUnreachable,
} from './errors'

describe('lib/errors', () => {
  it('classifies ServerUnreachableError', () => {
    const err = new ServerUnreachableError()
    expect(isServerUnreachableError(err)).toBe(true)
    expect(isLikelyNetworkFailure(err)).toBe(true)
  })

  it('maps TypeError failed-to-fetch to ServerUnreachableError', () => {
    const mapped = toServerUnreachable(new TypeError('Failed to fetch'))
    expect(mapped).toBeInstanceOf(ServerUnreachableError)
    expect(mapped.message).toMatch(/Cannot reach Sonix/)
  })

  it('maps AbortError to ServerUnreachableError', () => {
    const mapped = toServerUnreachable(new DOMException('aborted', 'AbortError'))
    expect(mapped).toBeInstanceOf(ServerUnreachableError)
  })
})

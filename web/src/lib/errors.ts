/** Typed client errors shared across auth and feature HTTP helpers. */

export class ServerUnreachableError extends Error {
  constructor(message = 'Cannot reach Sonix') {
    super(message)
    this.name = 'ServerUnreachableError'
  }
}

export function isServerUnreachableError(err: unknown): err is ServerUnreachableError {
  return err instanceof ServerUnreachableError
}

/** Map a failed fetch (network / abort) to ServerUnreachableError; rethrow others. */
export function toServerUnreachable(err: unknown): ServerUnreachableError {
  if (err instanceof ServerUnreachableError) return err
  if (err instanceof DOMException && err.name === 'AbortError') {
    return new ServerUnreachableError('Cannot reach Sonix')
  }
  if (err instanceof TypeError) {
    return new ServerUnreachableError('Cannot reach Sonix')
  }
  if (err instanceof Error && /failed to fetch|networkerror|load failed/i.test(err.message)) {
    return new ServerUnreachableError('Cannot reach Sonix')
  }
  return new ServerUnreachableError('Cannot reach Sonix')
}

export function isLikelyNetworkFailure(err: unknown): boolean {
  if (err instanceof ServerUnreachableError) return true
  if (err instanceof DOMException && err.name === 'AbortError') return true
  if (err instanceof TypeError) return true
  if (err instanceof Error && /failed to fetch|networkerror|load failed/i.test(err.message)) return true
  return false
}

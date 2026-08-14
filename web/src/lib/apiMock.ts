/** Installable DEV API mock — used only while `/__ui` MockApp is mounted. */

export type ApiMockHandler = (path: string, opts?: RequestInit) => Promise<unknown>

let handler: ApiMockHandler | null = null

export function installApiMock(h: ApiMockHandler): void {
  handler = h
}

export function uninstallApiMock(): void {
  handler = null
}

export function isApiMockActive(): boolean {
  return handler != null
}

export async function runApiMock<T>(path: string, opts?: RequestInit): Promise<T> {
  if (!handler) {
    throw new Error('api mock not installed')
  }
  return handler(path, opts) as Promise<T>
}

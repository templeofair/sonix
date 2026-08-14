import { createContext, useContext, useMemo, type ReactNode } from 'react'

type AppNavValue = {
  /** '' in product; '/__ui' inside the DEV full-app mock. */
  prefix: string
  appPath: (path: string) => string
  /** Strip prefix from a location pathname for chrome logic (camera, titles). */
  stripPrefix: (pathname: string) => string
}

const AppNavContext = createContext<AppNavValue>({
  prefix: '',
  appPath: (path) => normalizePath(path),
  stripPrefix: (pathname) => pathname || '/',
})

function normalizePath(path: string): string {
  if (!path || path === '/') return '/'
  return path.startsWith('/') ? path : `/${path}`
}

export function AppNavProvider({
  prefix = '',
  children,
}: {
  prefix?: string
  children: ReactNode
}) {
  const value = useMemo<AppNavValue>(() => {
    const p = prefix.replace(/\/$/, '')
    return {
      prefix: p,
      appPath: (path: string) => {
        const n = normalizePath(path)
        if (!p) return n
        if (n === '/') return p || '/'
        return `${p}${n}`
      },
      stripPrefix: (pathname: string) => {
        if (!p) return pathname || '/'
        if (pathname === p || pathname === `${p}/`) return '/'
        if (pathname.startsWith(`${p}/`)) return pathname.slice(p.length) || '/'
        return pathname || '/'
      },
    }
  }, [prefix])

  return <AppNavContext.Provider value={value}>{children}</AppNavContext.Provider>
}

export function useAppNav(): AppNavValue {
  return useContext(AppNavContext)
}

import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import * as authApi from './services/authApi'
import { isLikelyNetworkFailure } from '../../lib/errors'

type User = authApi.AuthUser

export type AuthContextValue = {
  user: User
  loading: boolean
  /** True when the initial /api/me probe could not reach the server. */
  serverUnreachable: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  /** Re-run the reachability / session probe (e.g. after "Try again"). */
  retryConnection: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue>(null!)

/** DEV mocks / Vitest — provide a fake auth value without hitting `/api`. */
export function AuthContextProvider({
  value,
  children,
}: {
  value: AuthContextValue
  children: ReactNode
}) {
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User>(null)
  const [loading, setLoading] = useState(true)
  const [serverUnreachable, setServerUnreachable] = useState(false)

  const fetchMe = useCallback(async () => {
    const result = await authApi.fetchMeWithTimeout()
    setServerUnreachable(result.unreachable)
    setUser(result.unreachable ? null : result.user)
  }, [])

  useEffect(() => {
    void fetchMe().finally(() => setLoading(false))
  }, [fetchMe])

  const login = useCallback(
    async (username: string, password: string) => {
      try {
        await authApi.login(username, password)
        setServerUnreachable(false)
        await fetchMe()
      } catch (err) {
        if (isLikelyNetworkFailure(err)) {
          setServerUnreachable(true)
        }
        throw err
      }
    },
    [fetchMe]
  )

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } catch (err) {
      if (isLikelyNetworkFailure(err)) {
        setServerUnreachable(true)
        setUser(null)
        return
      }
      throw err
    }
    setUser(null)
  }, [])

  const retryConnection = useCallback(async () => {
    setLoading(true)
    try {
      await fetchMe()
    } finally {
      setLoading(false)
    }
  }, [fetchMe])

  return (
    <AuthContext.Provider
      value={{ user, loading, serverUnreachable, login, logout, retryConnection }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}

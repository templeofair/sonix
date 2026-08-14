import { useMemo, type ReactNode } from 'react'
import {
  AuthContextProvider,
  type AuthContextValue,
} from '../../../features/auth'

export type MockAuthState = 'signedOut' | 'loading' | 'signedIn'

type Props = {
  state?: MockAuthState
  /** When set, Sign in shows this error (LOGIN_FORM Banner). */
  loginError?: string
  children: ReactNode
}

/** Fake auth for `/__ui` — no `/api` calls. */
export function MockAuthProvider({
  state = 'signedOut',
  loginError = 'Mock login only — use ACCEPT to change product auth UX.',
  children,
}: Props) {
  const value = useMemo<AuthContextValue>(() => {
    const signedIn = state === 'signedIn'
    return {
      user: signedIn ? { username: 'demo' } : null,
      loading: state === 'loading',
      serverUnreachable: false,
      login: async () => {
        if (loginError) throw new Error(loginError)
      },
      logout: async () => {},
      retryConnection: async () => {},
    }
  }, [state, loginError])

  return <AuthContextProvider value={value}>{children}</AuthContextProvider>
}

import { LoginForm } from '../../../features/auth'
import { MockAuthProvider, type MockAuthState } from '../providers/MockAuthProvider'

type Props = {
  authState?: MockAuthState
}

/** Tier 2 — real LoginForm + MockAuthProvider (no `/api`). */
export default function LoginMount({ authState = 'signedOut' }: Props) {
  return (
    <div className="rounded-card border border-border overflow-hidden bg-surface">
      <MockAuthProvider state={authState}>
        <LoginForm />
      </MockAuthProvider>
    </div>
  )
}

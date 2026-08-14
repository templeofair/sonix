import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthProvider'
import Banner from '../../../shared/components/Banner'
import Button from '../../../shared/components/Button'
import Field from '../../../shared/components/Field'
import Card from '../../../shared/components/Card'
import { useAppNav } from '../../../lib/appNav'

function EyeIcon({ open }: { open: boolean }) {
  if (open) {
    return (
      <svg aria-hidden className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228L3 3m0 0l18 18"
        />
      </svg>
    )
  }
  return (
    <svg aria-hidden className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z"
      />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  )
}

export default function LoginForm() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const { login, user, loading } = useAuth()
  const navigate = useNavigate()
  const { appPath } = useAppNav()

  useEffect(() => {
    if (!loading && user) navigate(appPath('/'), { replace: true })
  }, [loading, user, navigate, appPath])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await login(username, password)
      navigate(appPath('/'), { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface px-4 py-10">
      <div className="w-full max-w-sm">
        <Card className="p-6 sm:p-8">
          <h1 className="text-xl font-semibold text-gray-900 text-center tracking-tight">Sonix</h1>
          <p className="text-sm text-muted text-center mt-1 mb-6">Sign in to your library</p>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Field
              id="username"
              label="Username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
            />
            <Field
              id="password"
              label="Password"
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
              endAdornment={
                <button
                  type="button"
                  className="control inline-flex h-11 w-11 items-center justify-center rounded-btn text-muted hover:text-gray-900"
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  aria-pressed={showPassword}
                  onClick={() => setShowPassword((v) => !v)}
                >
                  <EyeIcon open={showPassword} />
                </button>
              }
            />
            {error && <Banner tone="error">{error}</Banner>}
            <Button type="submit" className="w-full py-2.5">
              Sign in
            </Button>
          </form>
        </Card>
      </div>
    </div>
  )
}

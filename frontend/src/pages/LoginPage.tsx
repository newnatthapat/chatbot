import { useState } from 'react'
import type { FormEvent } from 'react'
import { login } from '../api/client'

export function LoginPage({ onSuccess }: { onSuccess: () => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const ok = await login(username, password)
      if (ok) onSuccess()
      else setError('Invalid username or password')
    } catch (err) {
      setError(err instanceof Error ? err.message : "Couldn't reach the server")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="login-shell">
      <form onSubmit={handleSubmit} className="login-card">
        <div className="login-brand">
          <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path
              d="M4 12a8 8 0 1 1 3.2 6.4L4 20l1.1-3.6A7.96 7.96 0 0 1 4 12Z"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          <h1>Log in</h1>
        </div>
        <label className="field">
          Username
          <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
        </label>
        <label className="field">
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>
        <button type="submit" className="btn-block" disabled={submitting}>
          {submitting ? 'Logging in…' : 'Log in'}
        </button>
        {error && (
          <p role="alert" className="alert">
            {error}
          </p>
        )}
      </form>
    </div>
  )
}

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
    <form onSubmit={handleSubmit} style={{ maxWidth: 320, margin: '4rem auto', display: 'grid', gap: '0.75rem' }}>
      <h1>Log in</h1>
      <label>
        Username
        <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
      </label>
      <label>
        Password
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
      </label>
      <button type="submit" disabled={submitting}>
        Log in
      </button>
      {error && <p role="alert">{error}</p>}
    </form>
  )
}

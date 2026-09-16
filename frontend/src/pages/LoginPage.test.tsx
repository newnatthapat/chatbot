import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll, expect, test, vi } from 'vitest'
import { LoginPage } from './LoginPage'

const server = setupServer()
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

test('logging in with correct credentials calls onSuccess', async () => {
  server.use(
    http.post('/api/login', async ({ request }) => {
      const body = (await request.json()) as { username: string; password: string }
      if (body.username === 'admin' && body.password === '12345') {
        return HttpResponse.json({ ok: true })
      }
      return new HttpResponse(null, { status: 401 })
    }),
  )
  const onSuccess = vi.fn()
  render(<LoginPage onSuccess={onSuccess} />)

  await userEvent.type(screen.getByLabelText(/username/i), 'admin')
  await userEvent.type(screen.getByLabelText(/password/i), '12345')
  await userEvent.click(screen.getByRole('button', { name: /log in/i }))

  await vi.waitFor(() => expect(onSuccess).toHaveBeenCalled())
})

test('logging in with wrong credentials shows an error and does not call onSuccess', async () => {
  server.use(http.post('/api/login', () => new HttpResponse(null, { status: 401 })))
  const onSuccess = vi.fn()
  render(<LoginPage onSuccess={onSuccess} />)

  await userEvent.type(screen.getByLabelText(/username/i), 'admin')
  await userEvent.type(screen.getByLabelText(/password/i), 'wrong')
  await userEvent.click(screen.getByRole('button', { name: /log in/i }))

  expect(await screen.findByRole('alert')).toHaveTextContent(/invalid username or password/i)
  expect(onSuccess).not.toHaveBeenCalled()
})

test('shows an error when the server is unreachable', async () => {
  server.use(http.post('/api/login', () => HttpResponse.error()))
  const onSuccess = vi.fn()
  render(<LoginPage onSuccess={onSuccess} />)

  await userEvent.type(screen.getByLabelText(/username/i), 'admin')
  await userEvent.type(screen.getByLabelText(/password/i), '12345')
  await userEvent.click(screen.getByRole('button', { name: /log in/i }))

  expect(await screen.findByRole('alert')).toHaveTextContent(/couldn't reach the server/i)
  expect(onSuccess).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: /log in/i })).not.toBeDisabled()
})

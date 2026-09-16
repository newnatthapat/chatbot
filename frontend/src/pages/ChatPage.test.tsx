import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll, expect, test, vi } from 'vitest'
import { ChatPage } from './ChatPage'

const server = setupServer()
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function sseFrame(event: string, data: unknown): string {
  return `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`
}

function streamResponse(frames: string[]): HttpResponse<BodyInit> {
  const body = frames.join('')
  return new HttpResponse(body, {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  })
}

test('starting a new conversation adds it to the sidebar and selects it', async () => {
  server.use(
    http.get('/api/conversations', () => HttpResponse.json([])),
    http.post('/api/conversations', () => HttpResponse.json({ id: 1, createdAt: '2026-01-01T00:00:00Z' })),
    http.get('/api/conversations/1/messages', () => HttpResponse.json([])),
  )
  render(<ChatPage />)

  await userEvent.click(await screen.findByRole('button', { name: /new conversation/i }))

  const sidebar = screen.getByRole('navigation')
  expect(within(sidebar).getByRole('button', { name: /2026-01-01/ })).toBeInTheDocument()
})

test('selecting a past conversation loads its message history', async () => {
  server.use(
    http.get('/api/conversations', () =>
      HttpResponse.json([{ id: 7, createdAt: '2026-02-02T00:00:00Z' }]),
    ),
    http.get('/api/conversations/7/messages', () =>
      HttpResponse.json([
        { role: 'user', content: 'hello', createdAt: '2026-02-02T00:00:01Z' },
        { role: 'assistant', content: 'hi there', createdAt: '2026-02-02T00:00:02Z' },
      ]),
    ),
  )
  render(<ChatPage />)

  await userEvent.click(await screen.findByRole('button', { name: /2026-02-02/ }))

  expect(await screen.findByText('hello')).toBeInTheDocument()
  expect(await screen.findByText('hi there')).toBeInTheDocument()
})

test('sending a message shows it immediately and streams the assistant reply', async () => {
  server.use(
    http.get('/api/conversations', () => HttpResponse.json([{ id: 1, createdAt: '2026-01-01T00:00:00Z' }])),
    http.get('/api/conversations/1/messages', () => HttpResponse.json([])),
    http.post('/api/conversations/1/messages', () =>
      streamResponse([sseFrame('token', { token: 'Hi' }), sseFrame('token', { token: ' there' }), sseFrame('done', {})]),
    ),
  )
  render(<ChatPage />)

  await userEvent.click(await screen.findByRole('button', { name: /2026-01-01/ }))
  await userEvent.type(screen.getByLabelText(/message/i), 'hello')
  await userEvent.click(screen.getByRole('button', { name: /send/i }))

  expect(await screen.findByText('hello')).toBeInTheDocument()
  expect(await screen.findByText('Hi there')).toBeInTheDocument()
})

test('a provider failure shows an inline error and does not add a broken assistant message', async () => {
  server.use(
    http.get('/api/conversations', () => HttpResponse.json([{ id: 1, createdAt: '2026-01-01T00:00:00Z' }])),
    http.get('/api/conversations/1/messages', () => HttpResponse.json([])),
    http.post('/api/conversations/1/messages', () =>
      streamResponse([sseFrame('error', { message: "Couldn't reach the model provider" })]),
    ),
  )
  render(<ChatPage />)

  await userEvent.click(await screen.findByRole('button', { name: /2026-01-01/ }))
  await userEvent.type(screen.getByLabelText(/message/i), 'hello')
  await userEvent.click(screen.getByRole('button', { name: /send/i }))

  expect(await screen.findByRole('alert')).toHaveTextContent(/couldn't reach the model provider/i)
  expect(screen.queryByTestId('assistant-message')).not.toBeInTheDocument()
  expect(screen.getByLabelText(/message/i)).toHaveValue('hello')
})

test('deleting a conversation removes it from the sidebar', async () => {
  server.use(
    http.get('/api/conversations', () => HttpResponse.json([{ id: 1, createdAt: '2026-01-01T00:00:00Z' }])),
    http.delete('/api/conversations/1', () => HttpResponse.json({ ok: true })),
  )
  render(<ChatPage />)

  await userEvent.click(await screen.findByRole('button', { name: /delete/i }))

  expect(screen.queryByRole('button', { name: /2026-01-01/ })).not.toBeInTheDocument()
})

test('logging out calls the logout endpoint and notifies the parent', async () => {
  let loggedOut = false
  server.use(
    http.get('/api/conversations', () => HttpResponse.json([])),
    http.post('/api/logout', () => {
      loggedOut = true
      return HttpResponse.json({ ok: true })
    }),
  )
  const onLogout = vi.fn()
  render(<ChatPage onLogout={onLogout} />)

  await userEvent.click(await screen.findByRole('button', { name: /log out/i }))

  await vi.waitFor(() => expect(onLogout).toHaveBeenCalled())
  expect(loggedOut).toBe(true)
})

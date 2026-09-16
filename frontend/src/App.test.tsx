import { render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll, expect, test } from 'vitest'
import App from './App'

const server = setupServer()
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

test('shows the login page when there is no active session', async () => {
  server.use(http.get('/api/conversations', () => new HttpResponse(null, { status: 401 })))
  render(<App />)

  expect(await screen.findByRole('heading', { name: /log in/i })).toBeInTheDocument()
})

test('shows the chat page when a session is already active', async () => {
  server.use(http.get('/api/conversations', () => HttpResponse.json([])))
  render(<App />)

  expect(await screen.findByRole('navigation')).toBeInTheDocument()
})

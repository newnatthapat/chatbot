export type Conversation = {
  id: number
  createdAt: string
}

export type Message = {
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

export type StreamHandlers = {
  onToken: (token: string) => void
  onDone: () => void
  onError: (message: string) => void
}

async function request(path: string, init?: RequestInit): Promise<Response> {
  return fetch(path, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
}

export async function login(username: string, password: string): Promise<boolean> {
  let res: Response
  try {
    res = await request('/api/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  } catch {
    throw new Error("Couldn't reach the server")
  }
  return res.ok
}

export async function logout(): Promise<void> {
  await request('/api/logout', { method: 'POST' })
}

export async function listConversations(): Promise<Conversation[]> {
  const res = await request('/api/conversations')
  if (!res.ok) throw new Error('Failed to load conversations')
  return res.json()
}

export async function createConversation(): Promise<Conversation> {
  const res = await request('/api/conversations', { method: 'POST' })
  if (!res.ok) throw new Error('Failed to create conversation')
  return res.json()
}

export async function deleteConversation(id: number): Promise<void> {
  const res = await request(`/api/conversations/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete conversation')
}

export async function listMessages(id: number): Promise<Message[]> {
  const res = await request(`/api/conversations/${id}/messages`)
  if (!res.ok) throw new Error('Failed to load messages')
  return res.json()
}

// sendMessage posts a new Message and consumes the streamed
// Server-Sent-Events response, invoking the matching handler per event.
export async function sendMessage(id: number, content: string, handlers: StreamHandlers): Promise<void> {
  let res: Response
  try {
    res = await request(`/api/conversations/${id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
  } catch {
    handlers.onError("Couldn't reach the server")
    return
  }

  if (!res.ok || !res.body) {
    handlers.onError("Couldn't reach the model provider")
    return
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let sepIndex: number
    while ((sepIndex = buffer.indexOf('\n\n')) !== -1) {
      const frame = buffer.slice(0, sepIndex)
      buffer = buffer.slice(sepIndex + 2)
      handleFrame(frame, handlers)
    }
  }
}

function handleFrame(frame: string, handlers: StreamHandlers): void {
  let event = 'message'
  let data = ''
  for (const line of frame.split('\n')) {
    if (line.startsWith('event: ')) event = line.slice('event: '.length)
    else if (line.startsWith('data: ')) data = line.slice('data: '.length)
  }
  if (!data) return

  const parsed = JSON.parse(data)
  if (event === 'token') handlers.onToken(parsed.token)
  else if (event === 'done') handlers.onDone()
  else if (event === 'error') handlers.onError(parsed.message)
}

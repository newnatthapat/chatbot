import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import {
  type Conversation,
  type Message,
  createConversation,
  deleteConversation,
  listConversations,
  listMessages,
  logout,
  sendMessage,
} from '../api/client'

export function ChatPage({ onLogout }: { onLogout?: () => void }) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeId, setActiveId] = useState<number | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [draft, setDraft] = useState('')
  const [streamingText, setStreamingText] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const streamingRef = useRef('')

  useEffect(() => {
    listConversations().then(setConversations)
  }, [])

  async function handleNewConversation() {
    const conversation = await createConversation()
    setConversations((prev) => [conversation, ...prev])
    selectConversation(conversation.id)
  }

  async function selectConversation(id: number) {
    setActiveId(id)
    setError(null)
    setStreamingText(null)
    const history = await listMessages(id)
    setMessages(history)
  }

  async function handleLogout() {
    await logout()
    onLogout?.()
  }

  async function handleDelete(id: number) {
    await deleteConversation(id)
    setConversations((prev) => prev.filter((c) => c.id !== id))
    if (activeId === id) {
      setActiveId(null)
      setMessages([])
    }
  }

  async function handleSend(e: FormEvent) {
    e.preventDefault()
    if (activeId === null || !draft.trim()) return

    const content = draft
    setDraft('')
    setError(null)
    setMessages((prev) => [...prev, { role: 'user', content, createdAt: new Date().toISOString() }])
    streamingRef.current = ''
    setStreamingText('')

    await sendMessage(activeId, content, {
      onToken: (token) => {
        streamingRef.current += token
        setStreamingText(streamingRef.current)
      },
      onDone: () => {
        setMessages((prev) => [
          ...prev,
          { role: 'assistant', content: streamingRef.current, createdAt: new Date().toISOString() },
        ])
        setStreamingText(null)
      },
      onError: (message) => {
        setError(message)
        setStreamingText(null)
        setDraft(content)
      },
    })
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '260px 1fr', height: '100%' }}>
      <nav aria-label="Conversations" style={{ borderRight: '1px solid var(--color-border)', padding: '1rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.5rem' }}>
          <button type="button" onClick={handleNewConversation}>
            New conversation
          </button>
          <button type="button" onClick={handleLogout}>
            Log out
          </button>
        </div>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {conversations.map((c) => (
            <li key={c.id} style={{ display: 'flex', gap: '0.25rem' }}>
              <button type="button" onClick={() => selectConversation(c.id)}>
                {c.createdAt}
              </button>
              <button type="button" onClick={() => handleDelete(c.id)}>
                Delete
              </button>
            </li>
          ))}
        </ul>
      </nav>
      <section style={{ display: 'grid', gridTemplateRows: '1fr auto', padding: '1rem' }}>
        <div>
          {messages.map((m, i) => (
            <p key={i} data-testid={m.role === 'assistant' ? 'assistant-message' : 'user-message'}>
              {m.content}
            </p>
          ))}
          {streamingText !== null && <p data-testid="assistant-message">{streamingText}</p>}
          {error && <p role="alert">{error}</p>}
        </div>
        <form onSubmit={handleSend} style={{ display: 'flex', gap: '0.5rem' }}>
          <label style={{ flex: 1 }}>
            Message
            <input value={draft} onChange={(e) => setDraft(e.target.value)} disabled={activeId === null} />
          </label>
          <button type="submit" disabled={activeId === null}>
            Send
          </button>
        </form>
      </section>
    </div>
  )
}

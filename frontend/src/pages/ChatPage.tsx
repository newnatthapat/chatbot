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

function splitTimestamp(iso: string): { date: string; time: string } {
  const [date, rest] = iso.split('T')
  return { date: date ?? iso, time: rest ? rest.slice(0, 5) : '' }
}

export function ChatPage({ onLogout }: { onLogout?: () => void }) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeId, setActiveId] = useState<number | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [draft, setDraft] = useState('')
  const [streamingText, setStreamingText] = useState<string | null>(null)
  const [thinking, setThinking] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const streamingRef = useRef('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    listConversations().then(setConversations)
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ block: 'end' })
  }, [messages, streamingText, thinking])

  async function handleNewConversation() {
    const conversation = await createConversation()
    setConversations((prev) => [conversation, ...prev])
    selectConversation(conversation.id)
  }

  async function selectConversation(id: number) {
    setActiveId(id)
    setError(null)
    setStreamingText(null)
    setThinking(false)
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
    setThinking(true)

    await sendMessage(activeId, content, {
      onThinking: () => {
        setThinking(true)
      },
      onToken: (token) => {
        setThinking(false)
        streamingRef.current += token
        setStreamingText(streamingRef.current)
      },
      onDone: () => {
        setMessages((prev) => [
          ...prev,
          { role: 'assistant', content: streamingRef.current, createdAt: new Date().toISOString() },
        ])
        setStreamingText(null)
        setThinking(false)
      },
      onError: (message) => {
        setError(message)
        setStreamingText(null)
        setThinking(false)
        setDraft(content)
      },
    })
  }

  return (
    <div className="app-shell">
      <nav aria-label="Conversations" className="sidebar">
        <div className="sidebar-header">
          <span className="brand">
            <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                d="M4 12a8 8 0 1 1 3.2 6.4L4 20l1.1-3.6A7.96 7.96 0 0 1 4 12Z"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            Chatbot
          </span>
          <button type="button" className="btn-icon" onClick={handleLogout}>
            <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            Log out
          </button>
        </div>
        <div className="sidebar-actions">
          <button type="button" className="new-conversation-btn" onClick={handleNewConversation}>
            <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            </svg>
            New conversation
          </button>
        </div>
        {conversations.length === 0 ? (
          <p className="empty-sidebar">No conversations yet.</p>
        ) : (
          <ul className="conversation-list">
            {conversations.map((c) => {
              const { date, time } = splitTimestamp(c.createdAt)
              return (
                <li key={c.id} className="conversation-item">
                  <button
                    type="button"
                    className={`conversation-item-btn${c.id === activeId ? ' is-active' : ''}`}
                    onClick={() => selectConversation(c.id)}
                  >
                    <span className="conversation-date">{date}</span>
                    {time && <>{' '}<span className="conversation-time">{time}</span></>}
                  </button>
                  <button
                    type="button"
                    className="delete-btn"
                    aria-label="Delete conversation"
                    onClick={() => handleDelete(c.id)}
                  >
                    <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path
                        d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 12a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-12"
                        stroke="currentColor"
                        strokeWidth="1.6"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </svg>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </nav>
      <section className="chat-main">
        {activeId === null ? (
          <div className="chat-empty">
            <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                d="M4 12a8 8 0 1 1 3.2 6.4L4 20l1.1-3.6A7.96 7.96 0 0 1 4 12Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            <p>Select a conversation, or start a new one, to begin chatting.</p>
          </div>
        ) : (
          <div className="messages">
            {messages.map((m, i) => (
              <div key={i} className={`message-row ${m.role}`}>
                <p
                  className={`bubble ${m.role}`}
                  data-testid={m.role === 'assistant' ? 'assistant-message' : 'user-message'}
                >
                  {m.content}
                </p>
              </div>
            ))}
            {thinking && !streamingText && (
              <div className="message-row assistant">
                <div className="bubble assistant bubble-thinking" aria-label="Assistant is thinking">
                  <span className="dot" />
                  <span className="dot" />
                  <span className="dot" />
                </div>
              </div>
            )}
            {streamingText !== null && streamingText !== '' && (
              <div className="message-row assistant">
                <p className="bubble assistant" data-testid="assistant-message">
                  {streamingText}
                </p>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
        {error && (
          <p role="alert" className="alert chat-error">
            {error}
          </p>
        )}
        <form onSubmit={handleSend} className="composer">
          <label className="composer-field">
            <span className="sr-only">Message</span>
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              disabled={activeId === null}
              placeholder="Message the assistant…"
              autoComplete="off"
            />
          </label>
          <button type="submit" className="send-btn" disabled={activeId === null} aria-label="Send">
            <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                d="m4 12 16-7-6.5 16-2.5-6.5L4 12Z"
                stroke="currentColor"
                strokeWidth="1.6"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </button>
        </form>
      </section>
    </div>
  )
}

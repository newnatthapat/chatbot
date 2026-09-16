import { useEffect, useState } from 'react'
import { listConversations } from './api/client'
import { ChatPage } from './pages/ChatPage'
import { LoginPage } from './pages/LoginPage'

type AuthState = 'checking' | 'anon' | 'authed'

function App() {
  const [authState, setAuthState] = useState<AuthState>('checking')

  useEffect(() => {
    listConversations()
      .then(() => setAuthState('authed'))
      .catch(() => setAuthState('anon'))
  }, [])

  if (authState === 'checking') return null
  if (authState === 'anon') return <LoginPage onSuccess={() => setAuthState('authed')} />
  return <ChatPage onLogout={() => setAuthState('anon')} />
}

export default App

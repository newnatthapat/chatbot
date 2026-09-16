package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionCookieName = "session"
const sessionTTL = 7 * 24 * time.Hour

// sessionStore is an in-memory session registry. Sessions don't survive a
// backend restart, which is acceptable for the placeholder single-user auth
// described in ADR-0001.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time // token -> expiry
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]time.Time)}
}

func (s *sessionStore) create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = time.Now().Add(sessionTTL)
	return token, nil
}

func (s *sessionStore) valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

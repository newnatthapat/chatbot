// Package httpapi wires the Conversation/Message HTTP API together: routing,
// the session-cookie auth gate (ADR-0001), and streaming Provider responses
// (ADR-0002 covers why the active Provider itself isn't switchable here).
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"chatbot-backend/internal/provider"
	"chatbot-backend/internal/store"
)

type Server struct {
	store         *store.Store
	provider      provider.Provider
	adminUsername string
	adminPassword string
	sessions      *sessionStore
	mux           *http.ServeMux
}

func New(st *store.Store, p provider.Provider, adminUsername, adminPassword string) *Server {
	s := &Server{
		store:         st,
		provider:      p,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
		sessions:      newSessionStore(),
	}
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/conversations", s.requireAuth(s.handleListConversations))
	s.mux.HandleFunc("POST /api/conversations", s.requireAuth(s.handleCreateConversation))
	s.mux.HandleFunc("DELETE /api/conversations/{id}", s.requireAuth(s.handleDeleteConversation))
	s.mux.HandleFunc("GET /api/conversations/{id}/messages", s.requireAuth(s.handleListMessages))
	s.mux.HandleFunc("POST /api/conversations/{id}/messages", s.requireAuth(s.handlePostMessage))
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !s.sessions.valid(cookie.Value) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Username != s.adminUsername || body.Password != s.adminPassword {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.sessions.create()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type conversationDTO struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	convs, err := s.store.ListConversations(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	dtos := make([]conversationDTO, len(convs))
	for i, c := range convs {
		dtos[i] = conversationDTO{ID: c.ID, CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	conv, err := s.store.CreateConversation(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, conversationDTO{ID: conv.ID, CreatedAt: conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
}

func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	id, err := conversationIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid conversation id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteConversation(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type messageDTO struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	id, err := conversationIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid conversation id", http.StatusBadRequest)
		return
	}
	if _, err := s.store.GetConversation(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	msgs, err := s.store.ListMessages(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	dtos := make([]messageDTO, len(msgs))
	for i, m := range msgs {
		dtos[i] = messageDTO{Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	id, err := conversationIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid conversation id", http.StatusBadRequest)
		return
	}
	if _, err := s.store.GetConversation(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if _, err := s.store.AppendMessage(ctx, id, "user", body.Content); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	history, err := s.store.ListMessages(ctx, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	providerHistory := make([]provider.Message, len(history))
	for i, m := range history {
		providerHistory[i] = provider.Message{Role: m.Role, Content: m.Content}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	var full strings.Builder
	streamErr := s.provider.Stream(ctx, providerHistory,
		func(tok string) {
			full.WriteString(tok)
			writeSSE(w, "token", map[string]string{"token": tok})
			if flusher != nil {
				flusher.Flush()
			}
		},
		// Thinking text isn't part of the reply and is never persisted —
		// just forward a heartbeat so the connection keeps producing bytes
		// (some models spend a long time "thinking" before any reply text)
		// and the frontend can show that a response is in progress.
		func(string) {
			writeSSE(w, "thinking", map[string]bool{"active": true})
			if flusher != nil {
				flusher.Flush()
			}
		},
	)

	if streamErr != nil {
		writeSSE(w, "error", map[string]string{"message": streamErr.Error()})
		if flusher != nil {
			flusher.Flush()
		}
		return
	}

	if _, err := s.store.AppendMessage(ctx, id, "assistant", full.String()); err != nil {
		writeSSE(w, "error", map[string]string{"message": "failed to save the response"})
		if flusher != nil {
			flusher.Flush()
		}
		return
	}

	writeSSE(w, "done", map[string]string{})
	if flusher != nil {
		flusher.Flush()
	}
}

func conversationIDFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeSSE(w http.ResponseWriter, event string, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		payload = []byte(`{}`)
	}
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

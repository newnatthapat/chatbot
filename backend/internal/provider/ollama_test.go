package provider_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chatbot-backend/internal/provider"
)

func TestOllamaProviderStreamsTokensInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("request path = %q, want /api/chat", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		lines := []string{
			`{"message":{"role":"assistant","content":"Hel"},"done":false}`,
			`{"message":{"role":"assistant","content":"lo"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true}`,
		}
		for _, line := range lines {
			io.WriteString(w, line+"\n")
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	p := provider.NewOllama(server.URL, "llama3.1")

	var got strings.Builder
	err := p.Stream(context.Background(), []provider.Message{{Role: "user", Content: "hi"}}, func(tok string) {
		got.WriteString(tok)
	}, func(string) {})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if got.String() != "Hello" {
		t.Errorf("streamed content = %q, want %q", got.String(), "Hello")
	}
}

func TestOllamaProviderReportsThinkingSeparatelyFromContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		lines := []string{
			`{"message":{"role":"assistant","content":"","thinking":"pondering"},"done":false}`,
			`{"message":{"role":"assistant","content":"","thinking":" more"},"done":false}`,
			`{"message":{"role":"assistant","content":"Hi"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true}`,
		}
		for _, line := range lines {
			io.WriteString(w, line+"\n")
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	p := provider.NewOllama(server.URL, "llama3.1")

	var gotTokens, gotThinking strings.Builder
	err := p.Stream(context.Background(), []provider.Message{{Role: "user", Content: "hi"}},
		func(tok string) { gotTokens.WriteString(tok) },
		func(tok string) { gotThinking.WriteString(tok) },
	)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if gotTokens.String() != "Hi" {
		t.Errorf("streamed content = %q, want %q", gotTokens.String(), "Hi")
	}
	if gotThinking.String() != "pondering more" {
		t.Errorf("streamed thinking = %q, want %q", gotThinking.String(), "pondering more")
	}
}

func TestOllamaProviderReturnsErrorOnNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom")
	}))
	defer server.Close()

	p := provider.NewOllama(server.URL, "llama3.1")

	err := p.Stream(context.Background(), []provider.Message{{Role: "user", Content: "hi"}}, func(string) {}, func(string) {})
	if err == nil {
		t.Fatal("Stream should have returned an error for a non-200 response")
	}
}

// Command fakeollama is a dev-only stand-in for Ollama's streaming chat
// API, for exercising the chatbot end-to-end without a real model. It
// echoes the last user message back, one word at a time, with a short
// delay between chunks to simulate real token streaming. Sending a
// message containing "!fail" makes it return an error instead, to test
// the inline-error path.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Messages []message `json:"messages"`
}

type chatChunk struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":11434"
	}

	http.HandleFunc("/api/chat", handleChat)

	log.Printf("fakeollama listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastUser := ""
	for _, m := range req.Messages {
		if m.Role == "user" {
			lastUser = m.Content
		}
	}

	if strings.Contains(lastUser, "!fail") {
		http.Error(w, "fakeollama: simulated provider failure", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	reply := "You said: " + lastUser
	words := strings.Fields(reply)

	enc := json.NewEncoder(w)
	for i, word := range words {
		chunk := word
		if i < len(words)-1 {
			chunk += " "
		}
		var c chatChunk
		c.Message.Content = chunk
		_ = enc.Encode(c)
		flusher.Flush()
		time.Sleep(120 * time.Millisecond)
	}

	var done chatChunk
	done.Done = true
	_ = enc.Encode(done)
	flusher.Flush()
}

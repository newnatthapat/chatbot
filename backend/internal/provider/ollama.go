package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Ollama is a Provider backed by an Ollama server's /api/chat endpoint.
type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(baseURL, model string) *Ollama {
	return &Ollama{baseURL: baseURL, model: model, client: http.DefaultClient}
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ollamaChatChunk struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func (o *Ollama) Stream(ctx context.Context, history []Message, onToken func(string)) error {
	body, err := json.Marshal(ollamaChatRequest{Model: o.model, Messages: history, Stream: true})
	if err != nil {
		return fmt.Errorf("provider: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("provider: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("provider: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("provider: ollama returned status %d: %s", resp.StatusCode, payload)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var chunk ollamaChatChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return fmt.Errorf("provider: decode stream chunk: %w", err)
		}
		if chunk.Message.Content != "" {
			onToken(chunk.Message.Content)
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("provider: read stream: %w", err)
	}

	return nil
}

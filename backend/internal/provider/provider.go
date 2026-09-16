// Package provider abstracts over inference Providers (Ollama today, vLLM
// later) so the rest of the app never depends on a specific engine's API.
package provider

import "context"

type Message struct {
	Role    string
	Content string
}

// Provider streams a chat completion for the given Message history, calling
// onToken once per token/chunk of assistant text as it arrives. An error
// returned here — whether before any tokens are produced or partway through
// the stream — means the caller must not treat the response as complete.
type Provider interface {
	Stream(ctx context.Context, history []Message, onToken func(string)) error
}

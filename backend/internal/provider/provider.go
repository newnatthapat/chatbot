// Package provider abstracts over inference Providers (Ollama today, vLLM
// later) so the rest of the app never depends on a specific engine's API.
package provider

import "context"

type Message struct {
	Role    string
	Content string
}

// Provider streams a chat completion for the given Message history, calling
// onToken once per token/chunk of assistant reply text as it arrives, and
// onThinking once per chunk of reasoning text a "thinking" model emits
// before its reply (some models can spend a long time here with no reply
// text at all — callers must not treat this as the stream having stalled).
// Thinking text is informational only: it is never part of the reply and
// must not be persisted as one. An error returned here — whether before any
// tokens are produced or partway through the stream — means the caller must
// not treat the response as complete.
type Provider interface {
	Stream(ctx context.Context, history []Message, onToken func(string), onThinking func(string)) error
}

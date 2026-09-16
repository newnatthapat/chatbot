# Chatbot Webapp

A simple webapp (Go backend, React frontend) that lets a single admin user chat with an LLM served by a swappable inference Provider (currently Ollama, with vLLM planned).

## Language

**Provider**:
An inference engine that serves LLM models over an HTTP API. Exactly one Provider is active for the whole app at a time (currently Ollama; vLLM will be added later). Connection details are configured per-Provider via env vars.
_Avoid_: Backend, engine (ambiguous with the Go backend)

**Model**:
A single fixed LLM served by a Provider, set via env var. Not user-selectable — each Provider has exactly one Model.

**Conversation**:
A persisted thread of Messages. Users can have multiple Conversations, listed in a sidebar by creation timestamp, and can delete them. Has no title beyond its timestamp.
_Avoid_: Chat (overloaded with the app/feature name), Session (implies auth)

**Message**:
A single user or assistant turn within a Conversation. There is no system-prompt role — the app has no system prompt.

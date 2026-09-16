Status: ready-for-agent

# Chatbot MVP

## Problem Statement

I have Ollama already running (in Docker) on my VPS, but no way to actually chat with it except raw API calls. I want a simple, private webapp where I can hold conversations with the model, see responses stream in, and come back to past conversations later — without having to build a whole authentication system or commit to a single inference engine before I even know I need to switch.

## Solution

A simple Go + React webapp, styled with a red theme, gated by a basic login. It talks to the currently active **Provider** (Ollama today, vLLM later) using a fixed **Model**, lets me hold multiple **Conversations** with streamed responses, and persists everything in SQLite so history survives restarts. Switching Providers is a deploy-time config change, not a runtime feature — this app is intentionally minimal at every layer that isn't the core chat loop.

## User Stories

1. As a chatbot user, I want to log in with a username and password, so that the app isn't open to anyone who reaches the URL.
2. As a chatbot user, I want my login to persist across page reloads (via a session cookie), so that I don't have to log in every time I open the app.
3. As a chatbot user, I want to log out, so that I can end my session on a shared machine.
4. As a chatbot user, I want to start a new Conversation, so that I can begin a fresh topic without old messages cluttering the context.
5. As a chatbot user, I want to see a list of my past Conversations in a sidebar, so that I can find and return to earlier discussions.
6. As a chatbot user, I want Conversations in the sidebar labeled by their creation timestamp, so that I can tell them apart even without custom titles.
7. As a chatbot user, I want to open a past Conversation and see its full Message history, so that I can pick up where I left off.
8. As a chatbot user, I want to send a Message and see the assistant's reply stream in token-by-token, so that I get feedback quickly instead of staring at a blank screen.
9. As a chatbot user, I want my Conversations and Messages to persist across backend restarts, so that I don't lose history when the VPS or container restarts.
10. As a chatbot user, I want to delete a Conversation I no longer need, so that my Conversation list doesn't grow unbounded with clutter.
11. As a chatbot user, I want a clear inline error message when the Provider can't be reached or fails mid-response, so that I understand something went wrong rather than seeing a silent failure or infinite spinner.
12. As a chatbot user, I want to retry a failed Message by resending it, so that a transient Provider issue doesn't lose my input.
13. As a chatbot user, I want the whole app styled with a red color theme, so that it matches my visual preference.
14. As an operator, I want to configure each Provider's connection details (base URL, etc.) via environment variables, so that I can point the app at my VPS's Ollama instance without code changes.
15. As an operator, I want to configure a single fixed Model per Provider via an environment variable, so that I control exactly which model is served without exposing a model picker in the UI.
16. As an operator, I want to choose the active Provider via an environment variable, so that the whole app consistently uses one Provider at a time.
17. As an operator, I want changing the active Provider to require an explicit config change and restart (not a live admin toggle), so that Provider switches are deliberate, visible in deployment config, and simple to implement now (see ADR-0002).
18. As an operator, I want the Provider abstraction designed so that adding a vLLM implementation later doesn't require reworking the Conversation/Message/API layers, so that the eventual vLLM rollout is low-risk.
19. As an operator, I want to deploy the webapp as another service in the same docker-compose setup as Ollama, so that deployment stays consistent with my existing VPS setup.
20. As an operator, I want the admin credentials to be simple and hardcoded for now, so that I don't have to build real authentication before the core chat experience works (see ADR-0001).
21. As a developer, I want the domain model to have no `User`/ownership entity yet, so that I don't have to design speculative multi-tenancy before real auth requirements are known (see ADR-0001).
22. As a chatbot user, I want each Conversation to have no configurable system prompt, so that the model behaves with its own default behavior without an extra layer to reason about.
23. As a chatbot user, I want failed/partial assistant responses to not pollute my Conversation history, so that a retry after a Provider error doesn't leave a broken half-message behind.

## Implementation Decisions

- **Provider abstraction**: a Go interface (e.g. `Provider`) with a method to send a Conversation's Message history and receive a streamed response. `OllamaProvider` is the only implementation built now, calling Ollama's chat API in streaming mode. The interface is designed so a future `VLLMProvider` can be added without changing callers.
- **Provider/Model config**: environment variables define each Provider's base URL and fixed Model (e.g. `OLLAMA_BASE_URL`, `OLLAMA_MODEL`). `ACTIVE_PROVIDER` selects which Provider is in effect app-wide. The app reads this once at startup; changing it requires a restart (ADR-0002). If `ACTIVE_PROVIDER` names a Provider with no implementation or missing config, the app fails fast at startup with a clear error rather than failing at request time.
- **Auth**: a single hardcoded admin credential pair (username/password). Login is a POST endpoint that, on success, sets an HTTP-only session cookie. A session-checking middleware protects all Conversation/Message endpoints; login (and logout) endpoints are excluded. No `User` entity, no per-Conversation ownership (ADR-0001).
- **Persistence**: SQLite, accessed via a Go SQL driver. Schema holds Conversations (id, created_at) and Messages (id, conversation_id, role, content, created_at). Only fully-completed assistant Messages are persisted — if a Provider call fails or is interrupted mid-stream, no partial assistant Message is written; the user's Message that triggered it remains, so a retry re-sends against the same Conversation.
- **API contract** (endpoints, not file paths):
  - Login/logout endpoints that set/clear the session cookie.
  - List Conversations (id + created_at, ordered newest-first).
  - Create a new Conversation.
  - Delete a Conversation (cascades to its Messages).
  - List a Conversation's Messages.
  - Post a new Message to a Conversation, which streams the assistant's response back to the caller (e.g. via Server-Sent Events or chunked transfer) and persists the completed exchange once the stream finishes successfully.
- **Streaming**: the backend proxies the Provider's token stream directly to the frontend as it arrives, rather than buffering the full response server-side first.
- **Frontend structure**: a Login page and a main app view (Conversation sidebar + active Conversation panel). An API client module wraps fetch/stream handling; no specific state-management library is mandated beyond what's needed to render the list and the active Conversation's Messages.
- **Styling**: a red-primary color theme applied consistently across the app (e.g. via CSS custom properties/theme tokens), no other design system requirement.
- **Deployment**: the Go backend serves both the API and the built React static assets from a single process/container, added as one additional service in the existing docker-compose setup alongside the Ollama container.

## Testing Decisions

- Good tests here assert on external behavior (HTTP requests/responses, rendered UI and user interactions) — never on internal function calls, private state, or SQL query shapes.
- **Backend seam**: tests run the real HTTP handlers via `httptest.Server` against a real (temporary/in-memory) SQLite database, with the Provider replaced by a fake implementing the `Provider` interface — no real Ollama/network access. Cover: login gating, session persistence across requests, Conversation CRUD, Message send + streamed response content, persistence of only completed Messages, inline-error behavior when the fake Provider is made to fail, and startup failure when `ACTIVE_PROVIDER` is misconfigured.
- **Frontend seam**: component/page tests (e.g. Vitest + React Testing Library) render pages against a mocked backend API (e.g. via `msw`), asserting on rendered DOM and simulated interactions — login form submission, viewing the Conversation list, sending a Message and seeing streamed text appear, deleting a Conversation, and seeing an inline error on a simulated Provider failure.
- No prior art exists in this repo yet (greenfield project) — these seams establish the pattern for future features.

## Out of Scope

- Real multi-user authentication/authorization, and any `User`/ownership entity (ADR-0001) — deferred to a later phase.
- A `VLLMProvider` implementation — vLLM isn't deployed yet; only the abstraction needs to accommodate it.
- Runtime/admin-UI Provider switching — the active Provider is restart-only for now (ADR-0002).
- A model picker or support for multiple Models per Provider.
- System prompt configuration (global or per-Conversation).
- Conversation auto-titling/summarization or manual renaming.
- Automatic retry or Provider fallback on failure — the user retries manually by resending.
- Mobile-responsive layout considerations.
- Rate limiting, observability/metrics, and horizontal scaling.

## Further Notes

- Domain vocabulary (Provider, Model, Conversation, Message) is defined in `CONTEXT.md` at the repo root — use these terms consistently rather than synonyms like "chat," "session," or "engine."
- ADR-0001 (`docs/adr/0001-placeholder-single-user-authentication.md`) and ADR-0002 (`docs/adr/0002-active-provider-switch-via-restart.md`) explain why auth and Provider-switching are intentionally minimal right now; don't "fix" these without revisiting the ADRs first.

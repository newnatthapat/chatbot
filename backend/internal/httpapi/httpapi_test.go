package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"chatbot-backend/internal/httpapi"
	"chatbot-backend/internal/provider"
	"chatbot-backend/internal/store"
)

// fakeProvider emits tokens in order, then returns failErr (nil on success).
// A non-nil failErr with zero tokens simulates a failure before any output.
type fakeProvider struct {
	tokens  []string
	failErr error
}

func (f *fakeProvider) Stream(ctx context.Context, history []provider.Message, onToken func(string)) error {
	for _, tok := range f.tokens {
		onToken(tok)
	}
	return f.failErr
}

type testClient struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
}

func newTestServer(t *testing.T, p provider.Provider) *testClient {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	srv := httpapi.New(st, p, "admin", "12345")
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}

	return &testClient{t: t, server: ts, client: &http.Client{Jar: jar}}
}

func (c *testClient) do(method, path string, body any) *http.Response {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatalf("Do: %v", err)
	}
	return resp
}

func (c *testClient) login(username, password string) *http.Response {
	return c.do(http.MethodPost, "/api/login", map[string]string{"username": username, "password": password})
}

func (c *testClient) mustLogin() {
	c.t.Helper()
	resp := c.login("admin", "12345")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return v
}

func TestLoginWithCorrectCredentialsSucceeds(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	resp := c.login("admin", "12345")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestLoginWithWrongPasswordFails(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	resp := c.login("admin", "wrong")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestConversationsRequireLogin(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	resp := c.do(http.MethodGet, "/api/conversations", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCreateAndListConversations(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	c.mustLogin()

	createResp := c.do(http.MethodPost, "/api/conversations", nil)
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d, want 200", createResp.StatusCode)
	}
	created := decodeJSON[struct {
		ID int64 `json:"id"`
	}](t, createResp)

	listResp := c.do(http.MethodGet, "/api/conversations", nil)
	list := decodeJSON[[]struct {
		ID int64 `json:"id"`
	}](t, listResp)

	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v, want single conversation with ID %d", list, created.ID)
	}
}

func TestSendMessageStreamsAndPersistsBothMessages(t *testing.T) {
	c := newTestServer(t, &fakeProvider{tokens: []string{"Hi", " there"}})
	c.mustLogin()

	created := decodeJSON[struct {
		ID int64 `json:"id"`
	}](t, c.do(http.MethodPost, "/api/conversations", nil))

	resp := c.do(http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", created.ID), map[string]string{"content": "hello"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	streamed := string(body)
	if !strings.Contains(streamed, "Hi") || !strings.Contains(streamed, "there") {
		t.Fatalf("streamed body = %q, want it to contain the tokens", streamed)
	}
	if !strings.Contains(streamed, "event: done") {
		t.Fatalf("streamed body = %q, want a done event", streamed)
	}

	msgsResp := c.do(http.MethodGet, fmt.Sprintf("/api/conversations/%d/messages", created.ID), nil)
	msgs := decodeJSON[[]struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}](t, msgsResp)

	if len(msgs) != 2 {
		t.Fatalf("messages = %+v, want 2", msgs)
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("msgs[0] = %+v, want role=user content=hello", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "Hi there" {
		t.Errorf("msgs[1] = %+v, want role=assistant content=%q", msgs[1], "Hi there")
	}
}

func TestSendMessageOnProviderFailureDoesNotPersistAssistantMessage(t *testing.T) {
	c := newTestServer(t, &fakeProvider{tokens: []string{"Par"}, failErr: fmt.Errorf("provider unreachable")})
	c.mustLogin()

	created := decodeJSON[struct {
		ID int64 `json:"id"`
	}](t, c.do(http.MethodPost, "/api/conversations", nil))

	resp := c.do(http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", created.ID), map[string]string{"content": "hello"})
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	streamed := string(body)
	if !strings.Contains(streamed, "event: error") {
		t.Fatalf("streamed body = %q, want an error event", streamed)
	}

	msgsResp := c.do(http.MethodGet, fmt.Sprintf("/api/conversations/%d/messages", created.ID), nil)
	msgs := decodeJSON[[]struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}](t, msgsResp)

	if len(msgs) != 1 {
		t.Fatalf("messages = %+v, want only the user message to be persisted", msgs)
	}
	if msgs[0].Role != "user" {
		t.Errorf("msgs[0].Role = %q, want user", msgs[0].Role)
	}
}

func TestDeleteConversationRemovesIt(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	c.mustLogin()

	created := decodeJSON[struct {
		ID int64 `json:"id"`
	}](t, c.do(http.MethodPost, "/api/conversations", nil))

	delResp := c.do(http.MethodDelete, fmt.Sprintf("/api/conversations/%d", created.ID), nil)
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 200 or 204", delResp.StatusCode)
	}

	getResp := c.do(http.MethodGet, fmt.Sprintf("/api/conversations/%d/messages", created.ID), nil)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("status after delete = %d, want 404", getResp.StatusCode)
	}
}

func TestLogoutEndsSession(t *testing.T) {
	c := newTestServer(t, &fakeProvider{})
	c.mustLogin()

	logoutResp := c.do(http.MethodPost, "/api/logout", nil)
	logoutResp.Body.Close()

	resp := c.do(http.MethodGet, "/api/conversations", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status after logout = %d, want 401", resp.StatusCode)
	}
}

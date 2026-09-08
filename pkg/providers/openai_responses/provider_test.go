package openairesponses

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func successBody() string {
	return `{"id":"resp_1","object":"response","created_at":1,"model":"grok-4.6","output":[{"type":"message","id":"m1","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"status":"completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
}

func TestChat_ResponsesPathAndSessionHeader(t *testing.T) {
	var gotPath, gotAuth, gotSession, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotSession = r.Header.Get("x-opencode-session")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		gotModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successBody()))
	}))
	defer srv.Close()

	p := NewProvider("go-key", srv.URL, "test-agent", WithSessionHeaderName("x-opencode-session"))
	resp, err := p.Chat(context.Background(), nil, nil, "grok-4.6", map[string]any{
		"session_key": "sess-abc",
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotPath != "/responses" {
		t.Fatalf("path = %q, want /responses", gotPath)
	}
	if gotAuth != "Bearer go-key" {
		t.Fatalf("auth = %q, want Bearer go-key", gotAuth)
	}
	if gotSession != "sess-abc" {
		t.Fatalf("x-opencode-session = %q, want sess-abc", gotSession)
	}
	if gotModel != "grok-4.6" {
		t.Fatalf("model = %q, want grok-4.6", gotModel)
	}
	if resp == nil || resp.Content != "hello" {
		t.Fatalf("resp = %+v, want content hello", resp)
	}
}

func TestChat_ExplicitCustomHeaderWins(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("x-opencode-session")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successBody()))
	}))
	defer srv.Close()

	p := NewProvider("go-key", srv.URL, "test-agent",
		WithCustomHeaders(map[string]string{"x-opencode-session": "pinned"}),
		WithSessionHeaderName("x-opencode-session"),
	)
	if _, err := p.Chat(context.Background(), nil, nil, "grok-4.6", map[string]any{"session_key": "sess-abc"}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotSession != "pinned" {
		t.Fatalf("x-opencode-session = %q, want pinned", gotSession)
	}
}

func TestChat_MissingAPIKey(t *testing.T) {
	p := NewProvider("", "https://opencode.ai/zen/go/v1", "test-agent")
	if _, err := p.Chat(context.Background(), nil, nil, "grok-4.6", nil); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

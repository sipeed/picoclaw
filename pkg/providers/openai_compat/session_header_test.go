package openai_compat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChat_SessionHeader(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("x-opencode-session")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewProvider("k", srv.URL, "", WithSessionHeaderName("x-opencode-session"))
	resp, err := p.Chat(context.Background(), nil, nil, "m", map[string]any{"session_key": "sess-9"})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotSession != "sess-9" {
		t.Fatalf("x-opencode-session = %q, want sess-9", gotSession)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("resp = %+v, want content ok", resp)
	}
}

func TestChat_NoSessionHeaderByDefault(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("x-opencode-session")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewProvider("k", srv.URL, "")
	if _, err := p.Chat(context.Background(), nil, nil, "m", map[string]any{"session_key": "sess-9"}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotSession != "" {
		t.Fatalf("x-opencode-session = %q, want empty", gotSession)
	}
}

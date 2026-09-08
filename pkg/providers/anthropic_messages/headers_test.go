package anthropicmessages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const headersTestBody = `{"id":"msg-1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","model":"m","usage":{"input_tokens":1,"output_tokens":1}}`

func TestChat_SessionAndCustomHeaders(t *testing.T) {
	var gotSession, gotCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("x-opencode-session")
		gotCustom = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(headersTestBody))
	}))
	defer srv.Close()

	p := NewProviderWithOptions("k", srv.URL, "ua", 0,
		WithCustomHeaders(map[string]string{"X-Custom": "yes"}),
		WithSessionHeaderName("x-opencode-session"),
	)
	resp, err := p.Chat(context.Background(), nil, nil, "m", map[string]any{"session_key": "sess-1", "max_tokens": 1024})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotSession != "sess-1" {
		t.Fatalf("x-opencode-session = %q, want sess-1", gotSession)
	}
	if gotCustom != "yes" {
		t.Fatalf("X-Custom = %q, want yes", gotCustom)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("resp = %+v, want content ok", resp)
	}
}

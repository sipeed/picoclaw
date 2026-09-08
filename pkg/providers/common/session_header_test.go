package common

import (
	"net/http"
	"testing"
)

func TestSessionKeyFromOptions(t *testing.T) {
	if got := SessionKeyFromOptions(nil); got != "" {
		t.Fatalf("nil options = %q, want empty", got)
	}
	if got := SessionKeyFromOptions(map[string]any{"session_key": "  abc-123  "}); got != "abc-123" {
		t.Fatalf("session_key = %q, want trimmed value", got)
	}
	if got := SessionKeyFromOptions(map[string]any{"session_key": 42}); got != "" {
		t.Fatalf("non-string session_key = %q, want empty", got)
	}
}

func TestApplySessionHeader(t *testing.T) {
	header := http.Header{}
	ApplySessionHeader(header, map[string]any{"session_key": "sess-1"}, "x-opencode-session")
	if got := header.Get("x-opencode-session"); got != "sess-1" {
		t.Fatalf("header = %q, want sess-1", got)
	}

	// Existing value wins over options.
	header.Set("x-opencode-session", "explicit")
	ApplySessionHeader(header, map[string]any{"session_key": "sess-2"}, "x-opencode-session")
	if got := header.Get("x-opencode-session"); got != "explicit" {
		t.Fatalf("header = %q, want explicit", got)
	}

	// Empty session key sets nothing.
	empty := http.Header{}
	ApplySessionHeader(empty, map[string]any{}, "x-opencode-session")
	if got := empty.Get("x-opencode-session"); got != "" {
		t.Fatalf("header = %q, want empty", got)
	}

	// Empty header name is a no-op.
	ApplySessionHeader(http.Header{}, map[string]any{"session_key": "sess-1"}, "")
}

package anthropicprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsOAuthToken_DetectsClaudeCodeTokens(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "OAuth token with sk-ant-oat01- prefix",
			token: "sk-ant-oat01-abcdefghijk",
			want:  true,
		},
		{
			name:  "OAuth token with whitespace",
			token: "  sk-ant-oat01-xyz123  ",
			want:  true,
		},
		{
			name:  "Standard API key sk-ant-api03-",
			token: "sk-ant-api03-standard-key",
			want:  false,
		},
		{
			name:  "Empty token",
			token: "",
			want:  false,
		},
		{
			name:  "Whitespace only",
			token: "   ",
			want:  false,
		},
		{
			name:  "Different prefix",
			token: "sk-ant-other-prefix",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOAuthToken(tt.token)
			if got != tt.want {
				t.Errorf("isOAuthToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}

func TestNewProviderWithBaseURL_OAuthToken_UsesBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify OAuth-specific headers
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-ant-oat01-test-token" {
			t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer sk-ant-oat01-test-token")
		}

		betaHeader := r.Header.Get("anthropic-beta")
		if betaHeader != "oauth-2025-04-20" {
			t.Errorf("anthropic-beta header = %q, want %q", betaHeader, "oauth-2025-04-20")
		}

		// Should NOT have x-api-key header
		if apiKey := r.Header.Get("x-api-key"); apiKey != "" {
			t.Errorf("x-api-key header should not be set, got %q", apiKey)
		}

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4.6",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "OAuth works"},
			},
			"usage": map[string]any{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewProviderWithBaseURL("sk-ant-oat01-test-token", server.URL)
	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "test"}}, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 100})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
}

func TestNewProviderWithBaseURL_StandardAPIKey_UsesStandardAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify standard API key uses Authorization header without beta
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-ant-api03-standard-key" {
			t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer sk-ant-api03-standard-key")
		}

		// Should NOT have OAuth beta header
		if betaHeader := r.Header.Get("anthropic-beta"); betaHeader != "" {
			t.Errorf("anthropic-beta header should not be set for standard key, got %q", betaHeader)
		}

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4.6",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Standard auth works"},
			},
			"usage": map[string]any{
				"input_tokens":  5,
				"output_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewProviderWithBaseURL("sk-ant-api03-standard-key", server.URL)
	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "test"}}, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 100})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
}

func TestTokenSource_WithOAuthToken_UsesBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify OAuth headers from refreshed token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-ant-oat01-refreshed" {
			t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer sk-ant-oat01-refreshed")
		}

		betaHeader := r.Header.Get("anthropic-beta")
		if betaHeader != "oauth-2025-04-20" {
			t.Errorf("anthropic-beta header = %q, want %q", betaHeader, "oauth-2025-04-20")
		}

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4.6",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Refreshed OAuth works"},
			},
			"usage": map[string]any{
				"input_tokens":  5,
				"output_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "sk-ant-oat01-refreshed", nil
	}

	provider := NewProviderWithTokenSourceAndBaseURL("sk-ant-oat01-initial", tokenSource, server.URL)
	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "test"}}, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 100})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
}

func TestTokenSource_WithStandardAPIKey_UsesStandardAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify standard API key uses Authorization header without beta
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-ant-api03-refreshed" {
			t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer sk-ant-api03-refreshed")
		}

		// Should NOT have OAuth beta header
		if betaHeader := r.Header.Get("anthropic-beta"); betaHeader != "" {
			t.Errorf("anthropic-beta header should not be set for standard key, got %q", betaHeader)
		}

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4.6",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Refreshed standard auth works"},
			},
			"usage": map[string]any{
				"input_tokens":  5,
				"output_tokens": 4,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "sk-ant-api03-refreshed", nil
	}

	provider := NewProviderWithTokenSourceAndBaseURL("sk-ant-api03-initial", tokenSource, server.URL)
	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "test"}}, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 100})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
}

func TestOAuthToken_WithWhitespace_TrimmedCorrectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		// Should be trimmed
		if authHeader != "Bearer sk-ant-oat01-test" {
			t.Errorf("Authorization header = %q, want %q (should be trimmed)", authHeader, "Bearer sk-ant-oat01-test")
		}

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4.6",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"usage": map[string]any{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewProviderWithBaseURL("  sk-ant-oat01-test  ", server.URL)
	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "test"}}, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 100})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
}

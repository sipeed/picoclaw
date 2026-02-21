package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"

	anthropicprovider "github.com/sipeed/picoclaw/pkg/providers/anthropic"
)

func TestClaudeProvider_ChatRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       reqBody["model"],
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Hello! How can I help you?"},
			},
			"usage": map[string]any{
				"input_tokens":  15,
				"output_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	delegate := anthropicprovider.NewProviderWithClient(createAnthropicTestClient(server.URL, "test-token"))
	provider := newClaudeProviderWithDelegate(delegate)

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 1024})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello! How can I help you?")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.PromptTokens != 15 {
		t.Errorf("PromptTokens = %d, want 15", resp.Usage.PromptTokens)
	}
}

func TestClaudeProvider_GetDefaultModel(t *testing.T) {
	p := NewClaudeProvider("test-token")
	if got := p.GetDefaultModel(); got != "claude-sonnet-4.6" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "claude-sonnet-4.6")
	}
}

func createAnthropicTestClient(baseURL, token string) *anthropic.Client {
	c := anthropic.NewClient(
		anthropicoption.WithAuthToken(token),
		anthropicoption.WithBaseURL(baseURL),
	)
	return &c
}

package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	anthropicmessages "github.com/sipeed/picoclaw/pkg/providers/anthropic_messages"
	openairesponses "github.com/sipeed/picoclaw/pkg/providers/openai_responses"
)

func TestOpenCodeGoEndpointFamily(t *testing.T) {
	cases := map[string]string{
		"grok-4.6":                   "responses",
		"gpt-5.6-luna":               "responses",
		"muse-spark-1.3-contributor": "responses",
		"minimax-m3":                 "messages",
		"qwen3.7-plus":               "messages",
		"kimi-k3":                    "chat",
		"glm-5.3":                    "chat",
		"some-future-model":          "chat",
	}
	for model, want := range cases {
		if got := OpenCodeGoEndpointFamily(model); got != want {
			t.Fatalf("OpenCodeGoEndpointFamily(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestOpenCodeGoCatalogEntry(t *testing.T) {
	option, ok := modelProviderOptionForName("opencode-go")
	if !ok {
		t.Fatal("opencode-go catalog entry missing")
	}
	if option.DefaultAPIBase != "https://opencode.ai/zen/go/v1" {
		t.Fatalf("default_api_base = %q", option.DefaultAPIBase)
	}
	if !option.CreateAllowed || len(option.CommonModels) == 0 {
		t.Fatal("opencode-go should be creatable with common models")
	}
}

func openCodeGoResponsesBody() string {
	return `{"id":"resp_1","object":"response","created_at":1,"model":"grok-4.6","output":[{"type":"message","id":"m1","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"status":"completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
}

func openCodeGoMessagesBody() string {
	return `{"id":"msg_1","type":"message","role":"assistant","model":"qwen3.7-plus","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
}

func openCodeGoChatBody() string {
	return `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
}

func TestCreateProviderFromConfig_OpenCodeGo(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		responseBody string
		wantPath     string
		wantType     string
	}{
		{"responses family", "opencode-go/grok-4.6", openCodeGoResponsesBody(), "/responses", "*openairesponses.Provider"},
		{"messages family", "opencode-go/qwen3.7-plus", openCodeGoMessagesBody(), "/messages", "*anthropicmessages.Provider"},
		{"chat family", "opencode-go/kimi-k3", openCodeGoChatBody(), "/chat/completions", "*httpapi.HTTPProvider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotSession, gotAuth string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotSession = r.Header.Get("x-opencode-session")
				gotAuth = r.Header.Get("Authorization")
				if tt.wantType == "*anthropicmessages.Provider" {
					gotAuth = r.Header.Get("X-API-Key")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			cfg := &config.ModelConfig{
				ModelName: "test-opencode-go",
				Model:     tt.model,
				APIBase:   server.URL,
			}
			cfg.SetAPIKey("go-test-key")

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}

			switch tt.wantType {
			case "*openairesponses.Provider":
				if _, ok := provider.(*openairesponses.Provider); !ok {
					t.Fatalf("provider type = %T, want *openairesponses.Provider", provider)
				}
			case "*anthropicmessages.Provider":
				if _, ok := provider.(*anthropicmessages.Provider); !ok {
					t.Fatalf("provider type = %T, want *anthropicmessages.Provider", provider)
				}
			case "*httpapi.HTTPProvider":
				if _, ok := provider.(*HTTPProvider); !ok {
					t.Fatalf("provider type = %T, want *HTTPProvider", provider)
				}
			}

			resp, err := provider.Chat(
				t.Context(),
				[]Message{{Role: "user", Content: "hi"}},
				nil,
				modelID,
				map[string]any{"session_key": "sess-convo-1", "max_tokens": 1024},
			)
			if err != nil {
				t.Fatalf("Chat() error = %v", err)
			}
			if !strings.HasSuffix(gotPath, tt.wantPath) {
				t.Fatalf("path = %q, want suffix %q", gotPath, tt.wantPath)
			}
			if gotSession != "sess-convo-1" {
				t.Fatalf("x-opencode-session = %q, want sess-convo-1", gotSession)
			}
			if tt.wantType == "*anthropicmessages.Provider" {
				if gotAuth != "go-test-key" {
					t.Fatalf("X-API-Key = %q, want go-test-key", gotAuth)
				}
			} else if gotAuth != "Bearer go-test-key" {
				t.Fatalf("Authorization = %q, want Bearer go-test-key", gotAuth)
			}
			if resp == nil || resp.Content != "ok" {
				t.Fatalf("resp = %+v, want content ok", resp)
			}
		})
	}
}

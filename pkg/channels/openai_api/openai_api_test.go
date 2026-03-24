package openai_api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func newTestChannel(t *testing.T) (*OpenAIAPIChannel, *bus.MessageBus) {
	t.Helper()
	return newTestChannelWithConfig(t, nil)
}

func newTestChannelWithConfig(t *testing.T, mutate func(*config.Config)) (*OpenAIAPIChannel, *bus.MessageBus) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Channels.OpenAIAPI.APIKey = "test-key"
	cfg.Channels.OpenAIAPI.Port = 0
	if mutate != nil {
		mutate(cfg)
	}

	messageBus := bus.NewMessageBus()
	channel, err := NewOpenAIAPIChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("NewOpenAIAPIChannel() error = %v", err)
	}
	channel.SetRunning(true)

	return channel, messageBus
}

type failingResponseWriter struct {
	headers http.Header
}

func (w *failingResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *failingResponseWriter) WriteHeader(statusCode int) {}

func (w *failingResponseWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestHandleChatCompletions_NonStreaming(t *testing.T) {
	channel, messageBus := newTestChannel(t)

	go func() {
		msg := <-messageBus.InboundChan()
		if msg.Content != "Final user question" {
			t.Errorf("inbound content = %q, want %q", msg.Content, "Final user question")
		}
		if got := msg.Metadata["requested_model"]; got != "gpt-5.4" {
			t.Errorf("requested_model = %q, want %q", got, "gpt-5.4")
		}
		if got := msg.Metadata["no_history"]; got != "true" {
			t.Errorf("no_history = %q, want %q", got, "true")
		}
		if msg.Metadata["extra_system_prompt"] == "" {
			t.Error("expected extra_system_prompt metadata to be populated")
		}
		if msg.Metadata["injected_history"] == "" {
			t.Error("expected injected_history metadata to be populated")
		}

		_ = channel.Send(context.Background(), bus.OutboundMessage{
			Channel: channel.Name(),
			ChatID:  msg.ChatID,
			Content: "Assistant response",
		})
	}()

	body := `{
		"model": "gpt-5.4",
		"messages": [
			{"role": "system", "content": "Be concise."},
			{"role": "user", "content": "Earlier question"},
			{"role": "assistant", "content": "Earlier answer"},
			{"role": "user", "content": "Final user question"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()

	channel.handleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want %q", response.Model, "gpt-5.4")
	}
	if len(response.Choices) != 1 {
		t.Fatalf("len(choices) = %d, want 1", len(response.Choices))
	}
	if response.Choices[0].Message.Role != "assistant" {
		t.Fatalf("role = %q, want %q", response.Choices[0].Message.Role, "assistant")
	}
	if response.Choices[0].Message.Content != "Assistant response" {
		t.Fatalf("content = %q, want %q", response.Choices[0].Message.Content, "Assistant response")
	}
}

func TestHandleChatCompletions_Streaming(t *testing.T) {
	channel, messageBus := newTestChannel(t)

	go func() {
		msg := <-messageBus.InboundChan()
		_ = channel.Send(context.Background(), bus.OutboundMessage{
			Channel: channel.Name(),
			ChatID:  msg.ChatID,
			Content: "part one",
		})
		time.Sleep(50 * time.Millisecond)
		_ = channel.Send(context.Background(), bus.OutboundMessage{
			Channel: channel.Name(),
			ChatID:  msg.ChatID,
			Content: "part two",
		})
	}()

	body := `{
		"model": "gpt-5.4",
		"stream": true,
		"messages": [
			{"role": "user", "content": "Stream this please"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()

	channel.handleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, `"content":"part one"`) {
		t.Fatalf("stream body missing first chunk: %s", bodyText)
	}
	if !strings.Contains(bodyText, `"content":"part two"`) {
		t.Fatalf("stream body missing second chunk: %s", bodyText)
	}
	if !strings.Contains(bodyText, "data: [DONE]") {
		t.Fatalf("stream body missing [DONE]: %s", bodyText)
	}
}

func TestHandleModels_RequiresAuth(t *testing.T) {
	channel, _ := newTestChannel(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	channel.handleModels(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleModels_AllowsConfiguredOrigin(t *testing.T) {
	channel, _ := newTestChannelWithConfig(t, func(cfg *config.Config) {
		cfg.Channels.OpenAIAPI.AllowOrigins = []string{"https://console.example.com"}
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Origin", "https://console.example.com")
	rec := httptest.NewRecorder()

	channel.handleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://console.example.com")
	}
}

func TestHandleModels_DefaultCORSAllowsLocalhost(t *testing.T) {
	channel, _ := newTestChannelWithConfig(t, func(cfg *config.Config) {
		cfg.Channels.OpenAIAPI.AllowOrigins = nil
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	channel.handleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}

func TestHandleModels_RejectsDisallowedOrigin(t *testing.T) {
	channel, _ := newTestChannelWithConfig(t, func(cfg *config.Config) {
		cfg.Channels.OpenAIAPI.AllowOrigins = []string{"https://console.example.com"}
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	channel.handleModels(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if !strings.Contains(rec.Body.String(), "origin_not_allowed") {
		t.Fatalf("response body missing origin_not_allowed: %s", rec.Body.String())
	}
}

func TestHandleChatCompletions_RejectsInvalidLastMessage(t *testing.T) {
	channel, _ := newTestChannel(t)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "assistant last",
			body: `{
				"model": "gpt-5.4",
				"messages": [
					{"role": "user", "content": "hello"},
					{"role": "assistant", "content": "hi"}
				]
			}`,
		},
		{
			name: "tool last",
			body: `{
				"model": "gpt-5.4",
				"messages": [
					{"role": "user", "content": "hello"},
					{"role": "tool", "content": "tool result", "tool_call_id": "call_1"}
				]
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer test-key")
			rec := httptest.NewRecorder()

			channel.handleChatCompletions(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "invalid_messages") {
				t.Fatalf("response body missing invalid_messages: %s", rec.Body.String())
			}
		})
	}
}

func TestTranslateConversation_RejectsInvalidLastMessage(t *testing.T) {
	testCases := []struct {
		name     string
		messages []chatCompletionMessage
		wantErr  string
	}{
		{
			name: "assistant last",
			messages: []chatCompletionMessage{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			},
			wantErr: "got assistant",
		},
		{
			name: "tool last",
			messages: []chatCompletionMessage{
				{Role: "user", Content: "hello"},
				{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
			},
			wantErr: "got tool",
		},
		{
			name: "empty user last",
			messages: []chatCompletionMessage{
				{Role: "user", Content: "hello"},
				{Role: "user", Content: "   "},
			},
			wantErr: "must not be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := translateConversation(tc.messages)
			if err == nil {
				t.Fatal("translateConversation() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("translateConversation() error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestTranslateConversation_ExtractsHistoryAndCurrentUserMessage(t *testing.T) {
	translated, err := translateConversation([]chatCompletionMessage{
		{Role: "system", Content: "be concise"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "what next?"},
	})
	if err != nil {
		t.Fatalf("translateConversation() error = %v", err)
	}

	if translated.CurrentMessage != "what next?" {
		t.Fatalf("CurrentMessage = %q, want %q", translated.CurrentMessage, "what next?")
	}
	if translated.ExtraSystemPrompt != "be concise" {
		t.Fatalf("ExtraSystemPrompt = %q, want %q", translated.ExtraSystemPrompt, "be concise")
	}
	wantHistory := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	if len(translated.InjectedHistory) != len(wantHistory) {
		t.Fatalf("len(InjectedHistory) = %d, want %d", len(translated.InjectedHistory), len(wantHistory))
	}
	for i := range wantHistory {
		if !reflect.DeepEqual(translated.InjectedHistory[i], wantHistory[i]) {
			t.Fatalf("InjectedHistory[%d] = %+v, want %+v", i, translated.InjectedHistory[i], wantHistory[i])
		}
	}
}

func TestWriteOpenAIError_ReturnsWriteError(t *testing.T) {
	err := writeOpenAIError(&failingResponseWriter{}, http.StatusBadRequest, "bad request", "invalid_request_error", "bad_request")
	if err == nil {
		t.Fatal("writeOpenAIError() error = nil, want non-nil")
	}
}

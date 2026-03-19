package openai_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestChannel(t *testing.T) (*OpenAIAPIChannel, *bus.MessageBus) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Channels.OpenAIAPI.APIKey = "test-key"
	cfg.Channels.OpenAIAPI.Port = 0

	messageBus := bus.NewMessageBus()
	channel, err := NewOpenAIAPIChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("NewOpenAIAPIChannel() error = %v", err)
	}
	channel.SetRunning(true)

	return channel, messageBus
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

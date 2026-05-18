package sc3bot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func createTestChannel(t *testing.T) *SC3BotChannel {
	t.Helper()
	cfg := &config.SC3BotSettings{
		Token: *config.NewSecureString("test_token"),
	}
	bc := &config.Channel{}
	bc.SetName("test_sc3bot")
	messageBus := bus.NewMessageBus()

	ch, err := NewSC3BotChannel(bc, cfg, messageBus)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	return ch
}

func TestNewSC3BotChannel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.SC3BotSettings
		wantErr bool
	}{
		{
			name:    "missing token",
			cfg:     &config.SC3BotSettings{},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.SC3BotSettings{
				Token: *config.NewSecureString("test_token"),
			},
			wantErr: false,
		},
		{
			name: "with proxy",
			cfg: &config.SC3BotSettings{
				Token: *config.NewSecureString("test_token"),
				Proxy: "http://proxy.example.com:8080",
			},
			wantErr: false,
		},
		{
			name: "with secret",
			cfg: &config.SC3BotSettings{
				Token:  *config.NewSecureString("test_token"),
				Secret: "webhook_secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &config.Channel{}
			bc.SetName("test_sc3bot")
			messageBus := bus.NewMessageBus()

			ch, err := NewSC3BotChannel(bc, tt.cfg, messageBus)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSC3BotChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ch == nil {
				t.Error("NewSC3BotChannel() returned nil channel")
			}
		})
	}
}

func TestSC3BotChannelWebhookPath(t *testing.T) {
	ch := createTestChannel(t)
	expected := "/webhook/sc3bot"
	if got := ch.WebhookPath(); got != expected {
		t.Errorf("WebhookPath() = %v, want %v", got, expected)
	}
}

func TestWebhookRejectsNonPostMethod(t *testing.T) {
	ch := &SC3BotChannel{
		config: &config.SC3BotSettings{},
	}

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestWebhookRejectsOversizedBody(t *testing.T) {
	ch := &SC3BotChannel{
		config: &config.SC3BotSettings{},
	}

	oversized := bytes.Repeat([]byte("A"), maxWebhookBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	// http.MaxBytesReader returns 400 instead of 413
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookAcceptsValidPayload(t *testing.T) {
	ch := createTestChannel(t)

	payload := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 10,
			"chat_id":    123,
			"text":       "Hello",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp["ok"] {
		t.Error("expected ok to be true")
	}
}

func TestWebhookRejectsInvalidSecret(t *testing.T) {
	ch := &SC3BotChannel{
		config: &config.SC3BotSettings{
			Secret: "correct_secret",
		},
	}

	payload := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 10,
			"chat_id":    123,
			"text":       "Hello",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(webhookSecretHeader, "wrong_secret") //nolint
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWebhookAcceptsValidSecret(t *testing.T) {
	ch := createTestChannel(t)
	ch.config.Secret = "correct_secret"

	payload := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 10,
			"chat_id":    123,
			"text":       "Hello",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(webhookSecretHeader, "correct_secret") //nolint
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestProcessUpdate(t *testing.T) {
	ch := createTestChannel(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		update *SC3BotUpdate
	}{
		{
			name:   "nil message",
			update: &SC3BotUpdate{UpdateID: 1},
		},
		{
			name: "empty text",
			update: &SC3BotUpdate{
				UpdateID: 1,
				Message: &SC3BotMessage{
					MessageID: 10,
					ChatID:    123,
					Text:      "",
				},
			},
		},
		{
			name: "valid message",
			update: &SC3BotUpdate{
				UpdateID: 1,
				Message: &SC3BotMessage{
					MessageID: 10,
					ChatID:    123,
					Text:      "Hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			ch.processUpdateWithContext(ctx, tt.update)
		})
	}
}

func TestSC3BotResponseStruct(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected bool
	}{
		{
			name:     "ok true",
			jsonData: `{"ok": true, "result": true}`,
			expected: true,
		},
		{
			name:     "ok false",
			jsonData: `{"ok": false, "result": null}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp SC3BotResponse
			if err := json.Unmarshal([]byte(tt.jsonData), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp.OK != tt.expected {
				t.Errorf("OK = %v, want %v", resp.OK, tt.expected)
			}
		})
	}
}

func TestSC3BotUserStruct(t *testing.T) {
	jsonData := `{"id": 12345, "first_name": "Test Bot", "username": "testbot"}`
	var user SC3BotUser
	if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if user.ID != 12345 {
		t.Errorf("ID = %d, want 12345", user.ID)
	}
	if user.FirstName != "Test Bot" {
		t.Errorf("FirstName = %s, want Test Bot", user.FirstName)
	}
	if user.Username != "testbot" {
		t.Errorf("Username = %s, want testbot", user.Username)
	}
}

func TestSC3BotUpdateStruct(t *testing.T) {
	jsonData := `{"update_id": 1, "message": {"message_id": 10, "chat_id": 123, "text": "Hello"}}`
	var update SC3BotUpdate
	if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if update.UpdateID != 1 {
		t.Errorf("UpdateID = %d, want 1", update.UpdateID)
	}
	if update.Message == nil {
		t.Fatal("Message is nil")
	}
	if update.Message.MessageID != 10 {
		t.Errorf("Message.MessageID = %d, want 10", update.Message.MessageID)
	}
	if update.Message.ChatID != 123 {
		t.Errorf("Message.ChatID = %d, want 123", update.Message.ChatID)
	}
	if update.Message.Text != "Hello" {
		t.Errorf("Message.Text = %s, want Hello", update.Message.Text)
	}
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	ch := createTestChannel(t)

	// Test Channel interface
	var _ channels.Channel = ch

	// Test TypingCapable interface
	var _ channels.TypingCapable = ch

	// Test WebhookHandler interface
	var _ channels.WebhookHandler = ch
}

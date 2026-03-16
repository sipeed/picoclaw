package mattermost

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewMattermostChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	t.Run("valid config", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:   "https://mattermost.example.com",
			Token: "test-token",
		}
		ch, err := NewMattermostChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "mattermost" {
			t.Errorf("expected name 'mattermost', got %q", ch.Name())
		}
	})

	t.Run("missing url", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:   "",
			Token: "test-token",
		}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing URL")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:   "https://mattermost.example.com",
			Token: "",
		}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing token")
		}
	})

	t.Run("invalid url scheme", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:   "mattermost.example.com",
			Token: "test-token",
		}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for URL without scheme")
		}
	})

	t.Run("both missing", func(t *testing.T) {
		cfg := config.MattermostConfig{}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for empty config")
		}
	})
}

func TestParseChatID(t *testing.T) {
	tests := []struct {
		name      string
		chatID    string
		channelID string
		rootID    string
	}{
		{
			name:      "channel only",
			chatID:    "abc123",
			channelID: "abc123",
			rootID:    "",
		},
		{
			name:      "channel with thread",
			chatID:    "abc123/post456",
			channelID: "abc123",
			rootID:    "post456",
		},
		{
			name:      "empty string",
			chatID:    "",
			channelID: "",
			rootID:    "",
		},
		{
			name:      "multiple slashes",
			chatID:    "abc/def/ghi",
			channelID: "abc",
			rootID:    "def/ghi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelID, rootID := parseChatID(tt.chatID)
			if channelID != tt.channelID {
				t.Errorf("channelID: got %q, want %q", channelID, tt.channelID)
			}
			if rootID != tt.rootID {
				t.Errorf("rootID: got %q, want %q", rootID, tt.rootID)
			}
		})
	}
}

func TestBuildWSURL(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "https to wss",
			url:      "https://mattermost.example.com",
			expected: "wss://mattermost.example.com/api/v4/websocket",
		},
		{
			name:     "http to ws",
			url:      "http://localhost:8065",
			expected: "ws://localhost:8065/api/v4/websocket",
		},
		{
			name:     "https with trailing slash",
			url:      "https://mattermost.example.com/",
			expected: "wss://mattermost.example.com/api/v4/websocket",
		},
		{
			name:     "https with base path (reverse proxy)",
			url:      "https://mattermost.example.com/some/path",
			expected: "wss://mattermost.example.com/some/path/api/v4/websocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.MattermostConfig{URL: tt.url, Token: "test"}
			ch, err := NewMattermostChannel(cfg, msgBus)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := ch.buildWSURL()
			if got != tt.expected {
				t.Errorf("buildWSURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestStripBotMention(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	t.Run("with configured username", func(t *testing.T) {
		cfg := config.MattermostConfig{URL: "https://x", Token: "t", Username: "mybot"}
		ch, err := NewMattermostChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ch.compileMentionPattern()

		tests := []struct {
			input    string
			expected string
		}{
			{"@mybot hello", "hello"},
			{"hello @mybot world", "hello world"},
			{"hello", "hello"},
			{"@mybot", ""},
			{"", ""},
			{"@mybotany should not strip", "@mybotany should not strip"},
			{"@mybot, thanks", ", thanks"},
		}

		for _, tt := range tests {
			got := ch.stripBotMention(tt.input)
			if got != tt.expected {
				t.Errorf("stripBotMention(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		}
	})

	t.Run("with bot username from auth", func(t *testing.T) {
		cfg := config.MattermostConfig{URL: "https://x", Token: "t"}
		ch, err := NewMattermostChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ch.botUsername = "authbot"
		ch.compileMentionPattern()

		got := ch.stripBotMention("@authbot hello")
		if got != "hello" {
			t.Errorf("expected 'hello', got %q", got)
		}
	})

	t.Run("no username configured", func(t *testing.T) {
		cfg := config.MattermostConfig{URL: "https://x", Token: "t"}
		ch, err := NewMattermostChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := ch.stripBotMention("@someone hello")
		if got != "@someone hello" {
			t.Errorf("expected no stripping, got %q", got)
		}
	})
}

func TestHasBotMention(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	cfg := config.MattermostConfig{URL: "https://x", Token: "t", Username: "mybot"}
	ch, err := NewMattermostChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch.compileMentionPattern()

	tests := []struct {
		input    string
		expected bool
	}{
		{"@mybot hello", true},
		{"hello @mybot", true},
		{"@mybot, thanks", true},
		{"hello world", false},
		{"@otherbot hello", false},
		{"@mybotany hello", false},
		{"", false},
	}

	for _, tt := range tests {
		got := ch.hasBotMention(tt.input)
		if got != tt.expected {
			t.Errorf("hasBotMention(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// makePostedEventData builds a "posted" event data payload for testing handlePosted.
func makePostedEventData(postType, userID, channelID, message, channelType string) json.RawMessage {
	post := map[string]string{
		"id":         "post123",
		"type":       postType,
		"user_id":    userID,
		"channel_id": channelID,
		"root_id":    "",
		"message":    message,
	}
	postJSON, _ := json.Marshal(post)
	data := map[string]string{
		"post":         string(postJSON),
		"channel_type": channelType,
		"sender_name":  "testuser",
		"team_id":      "team1",
	}
	raw, _ := json.Marshal(data)
	return raw
}

func TestHandlePostedIgnoresSystemMessages(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	cfg := config.MattermostConfig{URL: "https://x", Token: "t"}
	ch, err := NewMattermostChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch.botUserID = "bot123"
	ch.ctx, ch.cancel = context.WithCancel(context.Background())
	defer ch.cancel()

	// System message (e.g. user joined channel) — should be ignored.
	ch.handlePosted(makePostedEventData("system_join_channel", "user1", "chan1", "user1 joined", "O"))

	// Regular user message — should be processed.
	ch.handlePosted(makePostedEventData("", "user1", "chan1", "hello bot", "D"))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected a message from the regular post, got none")
	}
	if msg.Content != "hello bot" {
		t.Errorf("expected 'hello bot', got %q", msg.Content)
	}
}

func TestHandlePostedIgnoresOwnMessages(t *testing.T) {
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	cfg := config.MattermostConfig{URL: "https://x", Token: "t"}
	ch, err := NewMattermostChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch.botUserID = "bot123"
	ch.ctx, ch.cancel = context.WithCancel(context.Background())
	defer ch.cancel()

	// Message from the bot itself — should be ignored.
	ch.handlePosted(makePostedEventData("", "bot123", "chan1", "I am the bot", "D"))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, ok := msgBus.ConsumeInbound(ctx)
	if ok {
		t.Error("expected no message, but got one")
	}
}

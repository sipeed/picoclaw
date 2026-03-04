package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
)

func TestIsTelegramSwitchAllowed(t *testing.T) {
	sender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  "123",
		CanonicalID: identity.BuildCanonicalID("telegram", "123"),
		Username:    "alice",
	}

	tests := []struct {
		name      string
		allowFrom config.FlexibleStringSlice
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowFrom: config.FlexibleStringSlice{},
			want:      true,
		},
		{
			name:      "matches raw platform id",
			allowFrom: config.FlexibleStringSlice{"123"},
			want:      true,
		},
		{
			name:      "matches canonical id",
			allowFrom: config.FlexibleStringSlice{"telegram:123"},
			want:      true,
		},
		{
			name:      "non matching denied",
			allowFrom: config.FlexibleStringSlice{"999"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTelegramSwitchAllowed(tt.allowFrom, sender)
			if got != tt.want {
				t.Fatalf("isTelegramSwitchAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSwitchPublishesNormalizedCommand(t *testing.T) {
	msgBus := bus.NewMessageBus()
	t.Cleanup(msgBus.Close)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "model-a",
			},
		},
		Channels: config.ChannelsConfig{
			Telegram: config.TelegramConfig{
				AllowFrom: config.FlexibleStringSlice{},
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "model-a", Model: "codex-cli/codex"},
		},
	}

	commander := &cmd{
		config: cfg,
		bus:    msgBus,
		// bot is intentionally nil: success path should not call SendMessage
	}

	message := telego.Message{
		MessageID: 7,
		Text:      "/switch model model-a",
		Chat: telego.Chat{
			ID: 42,
		},
		From: &telego.User{
			ID:        123,
			Username:  "alice",
			FirstName: "Alice",
		},
	}

	if err := commander.Switch(context.Background(), message); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	got, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound switch command")
	}
	if got.Content != "/switch model to model-a" {
		t.Fatalf("inbound content = %q, want %q", got.Content, "/switch model to model-a")
	}
	if got.Channel != "telegram" || got.ChatID != "42" {
		t.Fatalf("unexpected inbound routing: channel=%q chat=%q", got.Channel, got.ChatID)
	}
}

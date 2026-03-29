package telegram

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

func TestHandleMessage_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "/new",
		MessageID: 9,
		Chat: telego.Chat{
			ID:   123,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "telegram" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/new" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestHandleMessage_RewritesModelsShortcutToListModels(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "/models",
		MessageID: 10,
		Chat: telego.Chat{
			ID:   123,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Content != "/list models" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestHandleMessage_RewritesModelsShortcutWithArgumentToSwitchModel(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "/models qwen-max",
		MessageID: 11,
		Chat: telego.Chat{
			ID:   123,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Content != "/switch model to qwen-max" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestRewriteModelShortcut_WithBotUsernameTarget(t *testing.T) {
	got := rewriteModelShortcut("/models@testbot qwen-max", "testbot")
	if got != "/switch model to qwen-max" {
		t.Fatalf("rewrite=%q, want %q", got, "/switch model to qwen-max")
	}
}

func TestRewriteModelShortcut_DoesNotRewriteOtherBotTarget(t *testing.T) {
	got := rewriteModelShortcut("/models@otherbot qwen-max", "testbot")
	if got != "/models@otherbot qwen-max" {
		t.Fatalf("rewrite=%q, want unchanged", got)
	}
}

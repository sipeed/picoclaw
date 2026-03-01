package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestTryHandleCommand_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	ch := &WhatsAppChannel{}
	called := false
	ch.dispatcher = commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
		called = true
		return commands.Result{Matched: true, Handled: true}
	})

	handled := ch.tryHandleCommand(context.Background(), "/help", "chat1", "user1", "mid1")
	if handled {
		t.Fatalf("handled=%v", handled)
	}
	if called {
		t.Fatalf("handled=%v called=%v", handled, called)
	}
}

func TestHandleIncomingMessage_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	called := false
	ch := &WhatsAppChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp", config.WhatsAppConfig{}, messageBus, nil),
		dispatcher: commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
			called = true
			return commands.Result{Matched: true, Handled: true}
		}),
		ctx: context.Background(),
	}

	ch.handleIncomingMessage(map[string]any{
		"type":    "message",
		"id":      "mid1",
		"from":    "user1",
		"chat":    "chat1",
		"content": "/help",
	})

	if called {
		t.Fatal("expected generic command dispatch to be bypassed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "whatsapp" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/help" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

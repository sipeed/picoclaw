//go:build whatsapp_native

package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestTryHandleCommand_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	ch := &WhatsAppNativeChannel{}
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

func TestHandleIncoming_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	called := false
	ch := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
		dispatcher: commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
			called = true
			return commands.Result{Matched: true, Handled: true}
		}),
		runCtx: context.Background(),
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: types.NewJID("1001", types.DefaultUserServer),
				Chat:   types.NewJID("1001", types.DefaultUserServer),
			},
			ID:       "mid1",
			PushName: "Alice",
		},
		Message: &waE2E.Message{
			Conversation: proto.String("/new"),
		},
	}

	ch.handleIncoming(evt)

	if called {
		t.Fatal("expected generic command dispatch to be bypassed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "whatsapp_native" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/new" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

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
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestHandleIncoming_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
		runCtx:      context.Background(),
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for message to be forwarded")
		return
	case inbound, ok := <-messageBus.InboundChan():
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
}

func TestHandleIncoming_AllowFromFiltering(t *testing.T) {
	t.Run("blocked sender is discarded", func(t *testing.T) {
		messageBus := bus.NewMessageBus()
		cfg := config.WhatsAppConfig{AllowFrom: []string{"9999@s.whatsapp.net"}}
		ch := &WhatsAppNativeChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp_native", cfg, messageBus, cfg.AllowFrom),
			runCtx:      context.Background(),
		}

		evt := &events.Message{
			Info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.NewJID("1001", types.DefaultUserServer),
					Chat:   types.NewJID("1001", types.DefaultUserServer),
				},
				ID:       "mid1",
				PushName: "BlockedUser",
			},
			Message: &waE2E.Message{
				Conversation: proto.String("hello"),
			},
		}

		ch.handleIncoming(evt)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, ok := messageBus.ConsumeInbound(ctx)
		if ok {
			t.Fatal("expected message from non-allowed sender to be discarded")
		}
	})

	t.Run("allowed sender is delivered", func(t *testing.T) {
		messageBus := bus.NewMessageBus()
		cfg := config.WhatsAppConfig{AllowFrom: []string{"9999@s.whatsapp.net"}}
		ch := &WhatsAppNativeChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp_native", cfg, messageBus, cfg.AllowFrom),
			runCtx:      context.Background(),
		}

		evt := &events.Message{
			Info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.NewJID("9999", types.DefaultUserServer),
					Chat:   types.NewJID("9999", types.DefaultUserServer),
				},
				ID:       "mid2",
				PushName: "AllowedUser",
			},
			Message: &waE2E.Message{
				Conversation: proto.String("allowed message"),
			},
		}

		ch.handleIncoming(evt)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		inbound, ok := messageBus.ConsumeInbound(ctx)
		if !ok {
			t.Fatal("expected inbound message from allowed sender to be forwarded")
		}
		if inbound.Channel != "whatsapp_native" {
			t.Fatalf("channel=%q", inbound.Channel)
		}
		if inbound.Content != "allowed message" {
			t.Fatalf("content=%q", inbound.Content)
		}
	})

	t.Run("empty allowlist allows all", func(t *testing.T) {
		messageBus := bus.NewMessageBus()
		ch := &WhatsAppNativeChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
			runCtx:      context.Background(),
		}

		evt := &events.Message{
			Info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.NewJID("1001", types.DefaultUserServer),
					Chat:   types.NewJID("1001", types.DefaultUserServer),
				},
				ID:       "mid3",
				PushName: "AnyUser",
			},
			Message: &waE2E.Message{
				Conversation: proto.String("any user message"),
			},
		}

		ch.handleIncoming(evt)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		inbound, ok := messageBus.ConsumeInbound(ctx)
		if !ok {
			t.Fatal("expected inbound message to be forwarded when allowlist is empty")
		}
		if inbound.Content != "any user message" {
			t.Fatalf("content=%q", inbound.Content)
		}
	})
}

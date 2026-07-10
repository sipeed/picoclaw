//go:build whatsapp_native

package whatsapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
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
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppSettings{}, messageBus, nil),
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

func TestStartTypingSendsComposingThenPausedOnce(t *testing.T) {
	originalSend := sendWhatsAppChatPresence
	states := make(chan types.ChatPresence, 4)
	sendWhatsAppChatPresence = func(
		ctx context.Context,
		client *whatsmeow.Client,
		jid types.JID,
		state types.ChatPresence,
	) error {
		states <- state
		return nil
	}
	t.Cleanup(func() { sendWhatsAppChatPresence = originalSend })

	channel := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppSettings{}, bus.NewMessageBus(), nil),
		client:      &whatsmeow.Client{},
	}
	channel.SetRunning(true)

	stop, err := channel.StartTyping(context.Background(), "1001")
	if err != nil {
		t.Fatalf("StartTyping() error: %v", err)
	}
	if state := <-states; state != types.ChatPresenceComposing {
		t.Fatalf("initial presence = %q, want %q", state, types.ChatPresenceComposing)
	}

	stop()
	stop()
	if state := <-states; state != types.ChatPresencePaused {
		t.Fatalf("stopped presence = %q, want %q", state, types.ChatPresencePaused)
	}
	select {
	case state := <-states:
		t.Fatalf("unexpected extra presence after idempotent stop: %q", state)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestStartTypingRefreshesComposing(t *testing.T) {
	originalSend := sendWhatsAppChatPresence
	originalInterval := whatsappTypingRefreshInterval
	whatsappTypingRefreshInterval = 10 * time.Millisecond
	states := make(chan types.ChatPresence, 8)
	sendWhatsAppChatPresence = func(
		ctx context.Context,
		client *whatsmeow.Client,
		jid types.JID,
		state types.ChatPresence,
	) error {
		states <- state
		return nil
	}
	t.Cleanup(func() {
		sendWhatsAppChatPresence = originalSend
		whatsappTypingRefreshInterval = originalInterval
	})

	channel := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppSettings{}, bus.NewMessageBus(), nil),
		client:      &whatsmeow.Client{},
	}
	channel.SetRunning(true)

	stop, err := channel.StartTyping(context.Background(), "1001")
	if err != nil {
		t.Fatalf("StartTyping() error: %v", err)
	}
	defer stop()

	for i := 0; i < 2; i++ {
		select {
		case state := <-states:
			if state != types.ChatPresenceComposing {
				t.Fatalf("presence %d = %q, want %q", i, state, types.ChatPresenceComposing)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("timeout waiting for composing presence %d", i)
		}
	}
}

func TestStartTypingReturnsInitialPresenceError(t *testing.T) {
	originalSend := sendWhatsAppChatPresence
	wantErr := errors.New("presence unavailable")
	sendWhatsAppChatPresence = func(
		ctx context.Context,
		client *whatsmeow.Client,
		jid types.JID,
		state types.ChatPresence,
	) error {
		return wantErr
	}
	t.Cleanup(func() { sendWhatsAppChatPresence = originalSend })

	channel := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppSettings{}, bus.NewMessageBus(), nil),
		client:      &whatsmeow.Client{},
	}
	channel.SetRunning(true)

	stop, err := channel.StartTyping(context.Background(), "1001")
	if stop == nil {
		t.Fatal("StartTyping() returned nil stop function")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("StartTyping() error = %v, want %v", err, wantErr)
	}
}

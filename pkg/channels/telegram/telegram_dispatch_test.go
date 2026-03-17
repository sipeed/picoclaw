package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

type passiveRecorderStub struct {
	msgs []bus.InboundMessage
}

func (r *passiveRecorderStub) RecordPassiveInbound(_ context.Context, msg bus.InboundMessage) error {
	r.msgs = append(r.msgs, msg)
	return nil
}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
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

func TestHandleMessage_IgnoresBotAuthoredMessages(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "hello from another bot",
		MessageID: 10,
		Chat: telego.Chat{
			ID:   -100123,
			Type: "group",
		},
		From: &telego.User{
			ID:        777,
			FirstName: "Felix",
			IsBot:     true,
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if _, ok := messageBus.ConsumeInbound(ctx); ok {
		t.Fatal("expected bot-authored message to be ignored")
	}
}

func TestHandleMessage_GroupObserveOnly_PersistsUnmentionedMessages(t *testing.T) {
	ch, messageBus := newGroupChannelWithTrigger(t, "testbot", config.GroupTriggerConfig{
		MentionOnly: true,
		ObserveOnly: true,
	})
	recorder := &passiveRecorderStub{}
	ch.SetPassiveInboundRecorder(recorder)

	msg := &telego.Message{
		Text:      "hello group",
		MessageID: 11,
		Chat: telego.Chat{
			ID:   -100123,
			Type: "group",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	if _, ok := messageBus.ConsumeInbound(ctx); ok {
		t.Fatal("expected unmentioned group message to stay out of the agent pipeline")
	}
	if len(recorder.msgs) != 1 {
		t.Fatalf("passive recorder calls = %d, want 1", len(recorder.msgs))
	}

	recorded := recorder.msgs[0]
	if recorded.Content != "hello group" {
		t.Fatalf("content=%q", recorded.Content)
	}
	if recorded.ChatID != "-100123" {
		t.Fatalf("chat_id=%q", recorded.ChatID)
	}
	if recorded.Peer.Kind != "group" || recorded.Peer.ID != "-100123" {
		t.Fatalf("peer=%+v", recorded.Peer)
	}
}

func TestHandleMessage_GroupObserveOnly_ForwardsMentionedMessages(t *testing.T) {
	ch, messageBus := newGroupChannelWithTrigger(t, "testbot", config.GroupTriggerConfig{
		MentionOnly: true,
		ObserveOnly: true,
	})
	recorder := &passiveRecorderStub{}
	ch.SetPassiveInboundRecorder(recorder)

	msg := &telego.Message{
		Text: "@testbot hello",
		Entities: []telego.MessageEntity{{
			Type:   telego.EntityTypeMention,
			Offset: 0,
			Length: len("@testbot"),
		}},
		MessageID: 11,
		Chat: telego.Chat{
			ID:   -100123,
			Type: "group",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Content != "hello" {
		t.Fatalf("content=%q", inbound.Content)
	}
	if len(recorder.msgs) != 0 {
		t.Fatalf("passive recorder calls = %d, want 0", len(recorder.msgs))
	}
}

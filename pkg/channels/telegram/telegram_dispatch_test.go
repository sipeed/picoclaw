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

func TestHandleMessage_ForumTopic_IsolatesChatAndAddsRoutingMetadata(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		config: &config.Config{
			Channels: config.ChannelsConfig{
				Telegram: config.TelegramConfig{
					Groups: map[string]config.TelegramGroupConfig{
						"-1001234567890": {
							Topics: map[string]config.TelegramTopicConfig{
								"42": {AgentID: "coder"},
							},
						},
					},
				},
			},
		},
	}

	msg := &telego.Message{
		Text:            "hello topic",
		MessageID:       10,
		MessageThreadID: 42,
		Chat: telego.Chat{
			ID:      -1001234567890,
			Type:    "supergroup",
			IsForum: true,
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
	if inbound.ChatID != "-1001234567890:topic:42" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Peer.ID != "-1001234567890:topic:42" {
		t.Fatalf("peer_id=%q", inbound.Peer.ID)
	}
	if inbound.Metadata["parent_peer_kind"] != "group" {
		t.Fatalf("parent_peer_kind=%q", inbound.Metadata["parent_peer_kind"])
	}
	if inbound.Metadata["parent_peer_id"] != "-1001234567890" {
		t.Fatalf("parent_peer_id=%q", inbound.Metadata["parent_peer_id"])
	}
	if inbound.Metadata["route_agent_id"] != "coder" {
		t.Fatalf("route_agent_id=%q", inbound.Metadata["route_agent_id"])
	}
	if inbound.Metadata["route_matched_by"] != "telegram.topic" {
		t.Fatalf("route_matched_by=%q", inbound.Metadata["route_matched_by"])
	}
}

func TestHandleMessage_NonForumGroup_IgnoresThreadID(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:            "hello group",
		MessageID:       11,
		MessageThreadID: 42,
		Chat: telego.Chat{
			ID:   -1001234567890,
			Type: "supergroup",
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
	if inbound.ChatID != "-1001234567890" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Peer.ID != "-1001234567890" {
		t.Fatalf("peer_id=%q", inbound.Peer.ID)
	}
	if _, exists := inbound.Metadata["thread_id"]; exists {
		t.Fatalf("unexpected thread_id metadata=%q", inbound.Metadata["thread_id"])
	}
}

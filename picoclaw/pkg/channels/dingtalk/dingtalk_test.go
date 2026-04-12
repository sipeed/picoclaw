package dingtalk

import (
	"context"
	"testing"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestDingTalkChannel(t *testing.T, cfg config.DingTalkConfig) (*DingTalkChannel, *bus.MessageBus) {
	t.Helper()

	if cfg.ClientID == "" {
		cfg.ClientID = "test-client-id"
	}
	if cfg.ClientSecret.String() == "" {
		cfg.ClientSecret.Set("test-client-secret")
	}

	msgBus := bus.NewMessageBus()
	ch, err := NewDingTalkChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	return ch, msgBus
}

func mustReceiveInbound(t *testing.T, msgBus *bus.MessageBus) bus.InboundMessage {
	t.Helper()
	select {
	case msg := <-msgBus.InboundChan():
		return msg
	case <-time.After(time.Second):
		t.Fatal("expected inbound message")
		return bus.InboundMessage{}
	}
}

func TestOnChatBotMessageReceived_GroupMentionOnlyUsesIsInAtListAndStripsMention(t *testing.T) {
	ch, msgBus := newTestDingTalkChannel(t, config.DingTalkConfig{
		GroupTrigger: config.GroupTriggerConfig{MentionOnly: true},
	})

	_, err := ch.onChatBotMessageReceived(context.Background(), &chatbot.BotCallbackDataModel{
		Text:             chatbot.BotCallbackDataTextModel{Content: "  @bot /help  "},
		SenderStaffId:    "staff-123",
		SenderNick:       "Alice",
		ConversationType: "2",
		ConversationId:   "group-abc",
		SessionWebhook:   "https://example.com/webhook",
		IsInAtList:       true,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	inbound := mustReceiveInbound(t, msgBus)
	if inbound.Channel != "dingtalk" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.ChatID != "group-abc" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Peer.Kind != "group" || inbound.Peer.ID != "group-abc" {
		t.Fatalf("peer=%+v", inbound.Peer)
	}
	if inbound.Content != "/help" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestOnChatBotMessageReceived_DirectFallbackSenderIDUsesConversationID(t *testing.T) {
	ch, msgBus := newTestDingTalkChannel(t, config.DingTalkConfig{})

	_, err := ch.onChatBotMessageReceived(context.Background(), &chatbot.BotCallbackDataModel{
		Text:             chatbot.BotCallbackDataTextModel{Content: "ping"},
		SenderStaffId:    "",
		SenderId:         "openid-user-42",
		SenderNick:       "Bob",
		ConversationType: "1",
		ConversationId:   "conv-direct-42",
		SessionWebhook:   "https://example.com/webhook-direct",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	inbound := mustReceiveInbound(t, msgBus)
	if inbound.ChatID != "conv-direct-42" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Peer.Kind != "direct" || inbound.Peer.ID != "openid-user-42" {
		t.Fatalf("peer=%+v", inbound.Peer)
	}
	if inbound.SenderID != "dingtalk:openid-user-42" {
		t.Fatalf("sender_id=%q", inbound.SenderID)
	}

	if _, ok := ch.sessionWebhooks.Load("conv-direct-42"); !ok {
		t.Fatal("expected session webhook keyed by conversation_id")
	}
	if _, ok := ch.sessionWebhooks.Load(""); ok {
		t.Fatal("unexpected empty chat_id webhook key")
	}
}

func TestStripLeadingAtMentions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOut string
	}{
		{name: "single mention and command", input: "@bot /help", wantOut: "/help"},
		{name: "multiple mentions", input: "@bot @alice /new", wantOut: "/new"},
		{name: "no mention", input: "/help", wantOut: "/help"},
		{name: "mention only", input: "@bot", wantOut: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripLeadingAtMentions(tt.input)
			if got != tt.wantOut {
				t.Fatalf("stripLeadingAtMentions(%q)=%q want=%q", tt.input, got, tt.wantOut)
			}
		})
	}
}

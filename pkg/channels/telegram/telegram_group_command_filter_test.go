package telegram

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

type getMeCaller struct {
	username string
}

func (c getMeCaller) Call(_ context.Context, url string, _ *ta.RequestData) (*ta.Response, error) {
	if strings.HasSuffix(url, "/getMe") {
		result := fmt.Sprintf(`{"id":1,"is_bot":true,"first_name":"bot","username":%q}`, c.username)
		return &ta.Response{Ok: true, Result: []byte(result)}, nil
	}
	return &ta.Response{Ok: true, Result: []byte("true")}, nil
}

func newTestTelegramBot(t *testing.T, username string) *telego.Bot {
	t.Helper()

	token := "123456:" + strings.Repeat("a", 35)
	bot, err := telego.NewBot(token,
		telego.WithAPICaller(getMeCaller{username: username}),
		telego.WithDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("NewBot error: %v", err)
	}
	return bot
}

func newGroupMentionOnlyChannel(t *testing.T, botUsername string) (*TelegramChannel, *bus.MessageBus) {
	t.Helper()

	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil,
			channels.WithGroupTrigger(config.GroupTriggerConfig{MentionOnly: true}),
		),
		bot:     newTestTelegramBot(t, botUsername),
		chatIDs: make(map[string]int64),
		ctx:     context.Background(),
	}
	return ch, messageBus
}

func TestHandleMessage_GroupMentionOnly_BotCommandEntity(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		wantForwarded bool
		wantContent   string
	}{
		{
			name:          "command with bot username",
			text:          "/new@testbot",
			wantForwarded: true,
			wantContent:   "/new",
		},
		{
			name:          "bare command",
			text:          "/new",
			wantForwarded: true,
			wantContent:   "/new",
		},
		{
			name:          "command for another bot",
			text:          "/new@otherbot",
			wantForwarded: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch, messageBus := newGroupMentionOnlyChannel(t, "testbot")

			msg := &telego.Message{
				Text: tc.text,
				Entities: []telego.MessageEntity{{
					Type:   telego.EntityTypeBotCommand,
					Offset: 0,
					Length: len([]rune(tc.text)),
				}},
				MessageID: 42,
				Chat: telego.Chat{
					ID:   123,
					Type: "group",
				},
				From: &telego.User{
					ID:        7,
					FirstName: "Alice",
				},
			}

			if err := ch.handleMessage(context.Background(), msg); err != nil {
				t.Fatalf("handleMessage error: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			select {
			case <-ctx.Done():
				if tc.wantForwarded {
					t.Fatal("timeout waiting for message to be forwarded")
					return
				}
			case inbound, ok := <-messageBus.InboundChan():
				if tc.wantForwarded {
					if !ok {
						t.Fatal("expected inbound message to be forwarded")
					}
					if inbound.Content != tc.wantContent {
						t.Fatalf("content=%q want=%q", inbound.Content, tc.wantContent)
					}
					return
				}
			}
		})
	}
}

func TestHandleMessage_GroupMentionOnly_ReplyToBotMessage(t *testing.T) {
	ch, messageBus := newGroupMentionOnlyChannel(t, "testbot")

	msg := &telego.Message{
		Text: "follow up",
		MessageID: 43,
		Chat: telego.Chat{
			ID:   123,
			Type: "group",
		},
		From: &telego.User{
			ID:        7,
			FirstName: "Alice",
		},
		ReplyToMessage: &telego.Message{
			MessageID: 42,
			Text:      "bot response",
			From: &telego.User{
				ID:       1,
				IsBot:    true,
				Username: "testbot",
			},
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for message to be forwarded (reply to bot should trigger with mention_only)")
	case inbound, ok := <-messageBus.InboundChan():
		if !ok {
			t.Fatal("expected inbound message")
		}
		if !inbound.Context.Mentioned {
			t.Fatal("expected Mentioned=true for reply to bot message")
		}
		if inbound.Content != "[quoted assistant message from testbot]: bot response\n\nfollow up" {
			t.Fatalf("content=%q", inbound.Content)
		}
	}
}

func TestHandleMessage_GroupMentionOnly_ReplyToOtherUser_NotForwarded(t *testing.T) {
	ch, messageBus := newGroupMentionOnlyChannel(t, "testbot")

	msg := &telego.Message{
		Text: "follow up",
		MessageID: 44,
		Chat: telego.Chat{
			ID:   123,
			Type: "group",
		},
		From: &telego.User{
			ID:        7,
			FirstName: "Alice",
		},
		ReplyToMessage: &telego.Message{
			MessageID: 41,
			Text:      "some user message",
			From: &telego.User{
				ID:        99,
				FirstName: "Bob",
				IsBot:     false,
			},
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
	case inbound, ok := <-messageBus.InboundChan():
		if ok {
			t.Fatalf("reply to other user should not be forwarded with mention_only, got content=%q", inbound.Content)
		}
	}
}

func TestIsBotMentioned_MentionEntityUnaffected(t *testing.T) {
	ch, _ := newGroupMentionOnlyChannel(t, "testbot")

	msg := &telego.Message{
		Text: "@testbot hello",
		Entities: []telego.MessageEntity{{
			Type:   telego.EntityTypeMention,
			Offset: 0,
			Length: len("@testbot"),
		}},
	}

	if !ch.isBotMentioned(msg) {
		t.Fatal("expected mention entity to be treated as bot mention")
	}
}

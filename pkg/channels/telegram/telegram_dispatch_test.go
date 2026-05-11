package telegram

import (
	"context"
	"strings"
	"testing"

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
		tgCfg:       &config.TelegramSettings{BusinessMode: true},
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

func TestHandleBusinessMessage_DisabledBusinessModeIgnoresMessage(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg:       &config.TelegramSettings{BusinessMode: false},
	}

	msg := &telego.Message{
		Text:                 "ignored business message",
		MessageID:            18,
		BusinessConnectionID: "biz-conn-1",
		Chat: telego.Chat{
			ID:   777,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleBusinessMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleBusinessMessage error: %v", err)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("expected disabled business mode to ignore message, got %#v", inbound)
	default:
	}
}

func TestHandleBusinessMessage_BusinessOwnerIgnoresMessage(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg: &config.TelegramSettings{
			BusinessMode:  true,
			BusinessOwner: "42",
		},
	}

	msg := &telego.Message{
		Text:                 "owner should be ignored",
		MessageID:            19,
		BusinessConnectionID: "biz-conn-1",
		Chat: telego.Chat{
			ID:   777,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Owner",
		},
	}

	if err := ch.handleBusinessMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleBusinessMessage error: %v", err)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("expected owner business message to be ignored, got %#v", inbound)
	default:
	}
}

func TestHandleBusinessMessage_DisabledBusinessCommandsIgnoresCommand(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg: &config.TelegramSettings{
			BusinessMode:           true,
			BusinessCommandsEnable: false,
		},
	}

	msg := &telego.Message{
		Text:                 "/new",
		MessageID:            20,
		BusinessConnectionID: "biz-conn-1",
		Entities: []telego.MessageEntity{{
			Type:   telego.EntityTypeBotCommand,
			Offset: 0,
			Length: len("/new"),
		}},
		Chat: telego.Chat{
			ID:   777,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleBusinessMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleBusinessMessage error: %v", err)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("expected disabled business commands to ignore message, got %#v", inbound)
	default:
	}
}

func TestHandleBusinessMessage_EnabledBusinessCommandsForwardsCommand(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg: &config.TelegramSettings{
			BusinessMode:           true,
			BusinessCommandsEnable: true,
		},
	}

	msg := &telego.Message{
		Text:                 "/new",
		MessageID:            21,
		BusinessConnectionID: "biz-conn-1",
		Entities: []telego.MessageEntity{{
			Type:   telego.EntityTypeBotCommand,
			Offset: 0,
			Length: len("/new"),
		}},
		Chat: telego.Chat{
			ID:   777,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleBusinessMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleBusinessMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Content != "/new" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestTelegramAllowedUpdates_BusinessMode(t *testing.T) {
	disabled := strings.Join(telegramAllowedUpdates(false), ",")
	if strings.Contains(disabled, telego.BusinessMessageUpdates) {
		t.Fatalf("disabled updates include business messages: %s", disabled)
	}

	enabled := strings.Join(telegramAllowedUpdates(true), ",")
	if !strings.Contains(enabled, telego.BusinessMessageUpdates) {
		t.Fatalf("enabled updates do not include business messages: %s", enabled)
	}
	if !strings.Contains(enabled, telego.DeletedBusinessMessagesUpdates) {
		t.Fatalf("enabled updates do not include deleted business messages: %s", enabled)
	}
}

func TestHandleBusinessMessage_ForwardsWithBusinessContext(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg:       &config.TelegramSettings{BusinessMode: true},
	}

	msg := &telego.Message{
		Text:                 "hello from business",
		MessageID:            17,
		BusinessConnectionID: "biz-conn-1",
		Chat: telego.Chat{
			ID:   777,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleBusinessMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleBusinessMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.ChatID != "business:biz-conn-1:777" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Context.Account != "biz-conn-1" {
		t.Fatalf("account=%q", inbound.Context.Account)
	}
	if inbound.Context.Raw["business_connection_id"] != "biz-conn-1" {
		t.Fatalf("business_connection_id=%q", inbound.Context.Raw["business_connection_id"])
	}
	if inbound.Content != "hello from business" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

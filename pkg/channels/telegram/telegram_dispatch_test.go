package telegram

import (
	"context"
	"encoding/json"
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
	disabled := strings.Join(telegramAllowedUpdates(false, false), ",")
	if strings.Contains(disabled, telego.BusinessMessageUpdates) {
		t.Fatalf("disabled updates include business messages: %s", disabled)
	}

	enabled := strings.Join(telegramAllowedUpdates(true, false), ",")
	if !strings.Contains(enabled, telego.BusinessMessageUpdates) {
		t.Fatalf("enabled updates do not include business messages: %s", enabled)
	}
	if !strings.Contains(enabled, telego.DeletedBusinessMessagesUpdates) {
		t.Fatalf("enabled updates do not include deleted business messages: %s", enabled)
	}
}

func TestTelegramAllowedUpdates_GuestMode(t *testing.T) {
	disabled := strings.Join(telegramAllowedUpdates(false, false), ",")
	if strings.Contains(disabled, telegramGuestMessageUpdates) {
		t.Fatalf("disabled updates include guest messages: %s", disabled)
	}

	enabled := strings.Join(telegramAllowedUpdates(false, true), ",")
	if !strings.Contains(enabled, telegramGuestMessageUpdates) {
		t.Fatalf("enabled updates do not include guest messages: %s", enabled)
	}
}

func TestHandleGuestMessage_DisabledGuestModeIgnoresMessage(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg:       &config.TelegramSettings{GuestMode: false},
	}

	msg := &telegramGuestMessage{
		Message: telego.Message{
			Text:      "ignored guest message",
			MessageID: 18,
			Chat: telego.Chat{
				ID:   777,
				Type: "private",
			},
			From: &telego.User{
				ID:        42,
				FirstName: "Alice",
			},
		},
		GuestQueryID: "guest-query-1",
	}

	if err := ch.handleGuestMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleGuestMessage error: %v", err)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("expected disabled guest mode to ignore message, got %#v", inbound)
	default:
	}
}

func TestHandleGuestMessage_ForwardsWithGuestContext(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
		tgCfg:       &config.TelegramSettings{GuestMode: true},
	}

	msg := &telegramGuestMessage{
		Message: telego.Message{
			Text:      "hello from guest",
			MessageID: 17,
			Chat: telego.Chat{
				ID:   777,
				Type: "private",
			},
			From: &telego.User{
				ID:        42,
				FirstName: "Alice",
			},
		},
		GuestQueryID: "guest-query-1",
	}

	if err := ch.handleGuestMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleGuestMessage error: %v", err)
	}

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.ChatID != "guest:guest-query-1:777" {
		t.Fatalf("chat_id=%q", inbound.ChatID)
	}
	if inbound.Context.Account != "guest-query-1" {
		t.Fatalf("account=%q", inbound.Context.Account)
	}
	if inbound.Context.Raw["guest_query_id"] != "guest-query-1" {
		t.Fatalf("guest_query_id=%q", inbound.Context.Raw["guest_query_id"])
	}
	if inbound.Content != "hello from guest" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestTelegramRawUpdateDecodesGuestQueryID(t *testing.T) {
	payload := []byte(`{
		"update_id": 1001,
		"guest_message": {
			"message_id": 17,
			"guest_query_id": "guest-query-1",
			"date": 1760000000,
			"chat": {"id": 777, "type": "private"},
			"from": {"id": 42, "is_bot": false, "first_name": "Alice"},
			"text": "hello from guest"
		}
	}`)

	var update telegramRawUpdate
	if err := json.Unmarshal(payload, &update); err != nil {
		t.Fatalf("unmarshal raw update: %v", err)
	}
	if update.GuestMessage == nil {
		t.Fatal("guest message was not decoded")
	}
	if update.GuestMessage.GuestQueryID != "guest-query-1" {
		t.Fatalf("guest_query_id=%q", update.GuestMessage.GuestQueryID)
	}
	if update.GuestMessage.Text != "hello from guest" {
		t.Fatalf("text=%q", update.GuestMessage.Text)
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

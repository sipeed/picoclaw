package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPublishConsume(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	msg := InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat1",
			ChatType: "direct",
			SenderID: "user1",
		},
		Content: "hello",
	}

	if err := mb.PublishInbound(ctx, msg); err != nil {
		t.Fatalf("PublishInbound failed: %v", err)
	}

	got, ok := <-mb.InboundChan()
	if !ok {
		t.Fatal("ConsumeInbound returned ok=false")
	}
	if got.Content != "hello" {
		t.Fatalf("expected content 'hello', got %q", got.Content)
	}
	if got.Channel != "test" {
		t.Fatalf("expected channel 'test', got %q", got.Channel)
	}
	if got.Context.Channel != "test" {
		t.Fatalf("expected context channel 'test', got %q", got.Context.Channel)
	}
	if got.Context.ChatID != "chat1" {
		t.Fatalf("expected context chat ID 'chat1', got %q", got.Context.ChatID)
	}
	if got.Context.SenderID != "user1" {
		t.Fatalf("expected context sender ID 'user1', got %q", got.Context.SenderID)
	}
}

func TestPublishInbound_NormalizesContext(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := InboundMessage{
		Context: InboundContext{
			Channel:          "slack",
			Account:          "workspace-a",
			ChatID:           "C456/1712",
			ChatType:         "group",
			TopicID:          "1712",
			SpaceID:          "T001",
			SpaceType:        "team",
			SenderID:         "U123",
			MessageID:        "1712.01",
			ReplyToMessageID: "1700.01",
			Mentioned:        true,
		},
		Content: "hello",
	}

	if err := mb.PublishInbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishInbound failed: %v", err)
	}

	got := <-mb.InboundChan()
	if got.Context.Channel != "slack" {
		t.Fatalf("expected context channel slack, got %q", got.Context.Channel)
	}
	if got.Context.Account != "workspace-a" {
		t.Fatalf("expected context account workspace-a, got %q", got.Context.Account)
	}
	if got.Context.ChatType != "group" {
		t.Fatalf("expected context chat type group, got %q", got.Context.ChatType)
	}
	if got.Context.TopicID != "1712" {
		t.Fatalf("expected topic 1712, got %q", got.Context.TopicID)
	}
	if got.Context.SpaceType != "team" || got.Context.SpaceID != "T001" {
		t.Fatalf("expected team space T001, got %q/%q", got.Context.SpaceType, got.Context.SpaceID)
	}
	if !got.Context.Mentioned {
		t.Fatal("expected mentioned=true in context")
	}
	if got.Context.ReplyToMessageID != "1700.01" {
		t.Fatalf("expected reply_to_message_id 1700.01, got %q", got.Context.ReplyToMessageID)
	}
}

func TestPublishInbound_MirrorsContextIntoConvenienceFields(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := InboundMessage{
		Context: InboundContext{
			Channel:          "telegram",
			Account:          "bot-a",
			ChatID:           "-1001",
			ChatType:         "group",
			TopicID:          "42",
			SpaceID:          "guild-9",
			SpaceType:        "guild",
			SenderID:         "user-1",
			MessageID:        "777",
			Mentioned:        true,
			ReplyToMessageID: "666",
		},
		Content: "hi",
	}

	if err := mb.PublishInbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishInbound failed: %v", err)
	}

	got := <-mb.InboundChan()
	if got.Channel != "telegram" {
		t.Fatalf("expected legacy channel telegram, got %q", got.Channel)
	}
	if got.ChatID != "-1001" {
		t.Fatalf("expected legacy chat ID -1001, got %q", got.ChatID)
	}
	if got.SenderID != "user-1" {
		t.Fatalf("expected legacy sender ID user-1, got %q", got.SenderID)
	}
	if got.MessageID != "777" {
		t.Fatalf("expected legacy message ID 777, got %q", got.MessageID)
	}
	if got.Context.Account != "bot-a" || got.Context.SpaceID != "guild-9" || got.Context.TopicID != "42" {
		t.Fatalf("unexpected normalized context: %+v", got.Context)
	}
}

func TestPublishInbound_BackfillsContextFromLegacyFields(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := InboundMessage{
		Channel:   "pico",
		ChatID:    "session-1",
		SenderID:  "user-1",
		MessageID: "msg-1",
		Content:   "hello",
	}

	if err := mb.PublishInbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishInbound failed: %v", err)
	}

	got := <-mb.InboundChan()
	if got.Context.Channel != "pico" {
		t.Fatalf("expected context channel pico, got %q", got.Context.Channel)
	}
	if got.Context.ChatID != "session-1" {
		t.Fatalf("expected context chat ID session-1, got %q", got.Context.ChatID)
	}
	if got.Context.SenderID != "user-1" {
		t.Fatalf("expected context sender ID user-1, got %q", got.Context.SenderID)
	}
	if got.Context.MessageID != "msg-1" {
		t.Fatalf("expected context message ID msg-1, got %q", got.Context.MessageID)
	}
}

func TestPublishOutboundSubscribe(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	msg := OutboundMessage{
		Context: InboundContext{
			Channel: "telegram",
			ChatID:  "123",
		},
		Content: "world",
	}

	if err := mb.PublishOutbound(ctx, msg); err != nil {
		t.Fatalf("PublishOutbound failed: %v", err)
	}

	got, ok := <-mb.OutboundChan()
	if !ok {
		t.Fatal("SubscribeOutbound returned ok=false")
	}
	if got.Content != "world" {
		t.Fatalf("expected content 'world', got %q", got.Content)
	}
	if got.Context.Channel != "telegram" || got.Context.ChatID != "123" {
		t.Fatalf("expected normalized outbound context, got %+v", got.Context)
	}
}

func TestPublishOutbound_MirrorsContextToLegacyFields(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := OutboundMessage{
		Context: InboundContext{
			Channel:          "telegram",
			ChatID:           "chat-42",
			ReplyToMessageID: "msg-9",
		},
		AgentID:    "main",
		SessionKey: "sk_v1_123",
		Scope: &OutboundScope{
			Version:    1,
			AgentID:    "main",
			Channel:    "telegram",
			Account:    "bot-a",
			Dimensions: []string{"chat", "sender"},
			Values: map[string]string{
				"chat":   "direct:chat-42",
				"sender": "user-1",
			},
		},
		Content: "reply",
	}

	if err := mb.PublishOutbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishOutbound failed: %v", err)
	}

	got := <-mb.OutboundChan()
	if got.Channel != "telegram" {
		t.Fatalf("expected legacy channel telegram, got %q", got.Channel)
	}
	if got.ChatID != "chat-42" {
		t.Fatalf("expected legacy chat ID chat-42, got %q", got.ChatID)
	}
	if got.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected mirrored reply_to_message_id msg-9, got %q", got.ReplyToMessageID)
	}
	if got.AgentID != "main" || got.SessionKey != "sk_v1_123" {
		t.Fatalf("unexpected outbound turn metadata: agent=%q session=%q", got.AgentID, got.SessionKey)
	}
	if got.Scope == nil || got.Scope.AgentID != "main" || got.Scope.Values["chat"] != "direct:chat-42" {
		t.Fatalf("unexpected outbound scope: %+v", got.Scope)
	}
	if got.Context.Channel != "telegram" || got.Context.ChatID != "chat-42" {
		t.Fatalf("unexpected outbound context: %+v", got.Context)
	}
}

func TestPublishOutbound_PreservesExplicitReplyToMessageID(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := OutboundMessage{
		Context: InboundContext{
			Channel: "telegram",
			ChatID:  "chat-42",
		},
		ReplyToMessageID: "msg-9",
		Content:          "reply",
	}

	if err := mb.PublishOutbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishOutbound failed: %v", err)
	}

	got := <-mb.OutboundChan()
	if got.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected mirrored reply_to_message_id msg-9, got %q", got.ReplyToMessageID)
	}
	if got.Context.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected context reply_to_message_id msg-9, got %q", got.Context.ReplyToMessageID)
	}
}

func TestPublishOutbound_PreservesExplicitReplyToMessageIDWhenContextReplyIsBlank(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := OutboundMessage{
		Context: InboundContext{
			Channel:          "telegram",
			ChatID:           "chat-42",
			ReplyToMessageID: "   ",
		},
		ReplyToMessageID: "msg-9",
		Content:          "reply",
	}

	if err := mb.PublishOutbound(context.Background(), msg); err != nil {
		t.Fatalf("PublishOutbound failed: %v", err)
	}

	got := <-mb.OutboundChan()
	if got.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected mirrored reply_to_message_id msg-9, got %q", got.ReplyToMessageID)
	}
	if got.Context.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected context reply_to_message_id msg-9, got %q", got.Context.ReplyToMessageID)
	}
}

func TestPublishOutboundMedia_MirrorsContextToLegacyFields(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	msg := OutboundMediaMessage{
		Context: InboundContext{
			Channel: "slack",
			ChatID:  "C001",
		},
		AgentID:    "support",
		SessionKey: "sk_v1_media",
		Scope: &OutboundScope{
			Version:    1,
			AgentID:    "support",
			Channel:    "slack",
			Dimensions: []string{"chat"},
			Values: map[string]string{
				"chat": "channel:c001",
			},
		},
		Parts: []MediaPart{{Type: "image", Ref: "media://1"}},
	}

	if err := mb.PublishOutboundMedia(context.Background(), msg); err != nil {
		t.Fatalf("PublishOutboundMedia failed: %v", err)
	}

	got := <-mb.OutboundMediaChan()
	if got.Channel != "slack" {
		t.Fatalf("expected legacy channel slack, got %q", got.Channel)
	}
	if got.ChatID != "C001" {
		t.Fatalf("expected legacy chat ID C001, got %q", got.ChatID)
	}
	if got.AgentID != "support" || got.SessionKey != "sk_v1_media" {
		t.Fatalf("unexpected outbound media turn metadata: agent=%q session=%q", got.AgentID, got.SessionKey)
	}
	if got.Scope == nil || got.Scope.Values["chat"] != "channel:c001" {
		t.Fatalf("unexpected outbound media scope: %+v", got.Scope)
	}
	if got.Context.Channel != "slack" || got.Context.ChatID != "C001" {
		t.Fatalf("unexpected outbound media context: %+v", got.Context)
	}
}

func TestPublishAudioChunkSubscribe(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	chunk := AudioChunk{
		SessionID: "voice-1",
		SpeakerID: "speaker-1",
		ChatID:    "chat-1",
		Channel:   "discord",
		Sequence:  7,
		Format:    "opus",
		Data:      []byte{0x01, 0x02},
	}

	if err := mb.PublishAudioChunk(context.Background(), chunk); err != nil {
		t.Fatalf("PublishAudioChunk failed: %v", err)
	}

	got, ok := <-mb.AudioChunksChan()
	if !ok {
		t.Fatal("AudioChunksChan returned ok=false")
	}
	if got.SessionID != "voice-1" || got.Sequence != 7 {
		t.Fatalf("unexpected audio chunk: %+v", got)
	}
}

func TestPublishVoiceControlSubscribe(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctrl := VoiceControl{
		SessionID: "voice-1",
		ChatID:    "chat-1",
		Type:      "command",
		Action:    "start",
	}

	if err := mb.PublishVoiceControl(context.Background(), ctrl); err != nil {
		t.Fatalf("PublishVoiceControl failed: %v", err)
	}

	got, ok := <-mb.VoiceControlsChan()
	if !ok {
		t.Fatal("VoiceControlsChan returned ok=false")
	}
	if got.Type != "command" || got.Action != "start" {
		t.Fatalf("unexpected voice control: %+v", got)
	}
}

func TestNewOutboundContext_NormalizesReplyAddress(t *testing.T) {
	ctx := NewOutboundContext(" telegram ", " chat-42 ", " msg-9 ")
	if ctx.Channel != "telegram" {
		t.Fatalf("expected channel telegram, got %q", ctx.Channel)
	}
	if ctx.ChatID != "chat-42" {
		t.Fatalf("expected chat_id chat-42, got %q", ctx.ChatID)
	}
	if ctx.ReplyToMessageID != "msg-9" {
		t.Fatalf("expected reply_to_message_id msg-9, got %q", ctx.ReplyToMessageID)
	}
}

func TestPublishInbound_ContextCancel(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	// Fill the buffer
	ctx := context.Background()
	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(ctx, InboundMessage{
			Context: InboundContext{
				Channel:  "test",
				ChatID:   "chat-fill",
				ChatType: "direct",
				SenderID: "user-fill",
			},
			Content: "fill",
		}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Now buffer is full; publish with a canceled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mb.PublishInbound(cancelCtx, InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat-overflow",
			ChatType: "direct",
			SenderID: "user-overflow",
		},
		Content: "overflow",
	})
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPublishInbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	err := mb.PublishInbound(context.Background(), InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat1",
			ChatType: "direct",
			SenderID: "user1",
		},
		Content: "test",
	})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestPublishOutbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	err := mb.PublishOutbound(context.Background(), OutboundMessage{
		Context: InboundContext{
			Channel: "test",
			ChatID:  "chat1",
		},
		Content: "test",
	})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestConsumeInbound_ContextCancel(t *testing.T) {
	mb := NewMessageBus()

	defer mb.Close()

	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(context.Background(), InboundMessage{
			Context: InboundContext{
				Channel:  "test",
				ChatID:   "chat-fill",
				ChatType: "direct",
				SenderID: "user-fill",
			},
			Content: "fill",
		}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mb.PublishInbound(ctx, InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat-cancel",
			ChatType: "direct",
			SenderID: "user-cancel",
		},
		Content: "ContextCancel",
	})

	select {
	case <-ctx.Done():
		t.Log("context canceled, as expected")

	case msg, ok := <-mb.InboundChan():
		if !ok {
			t.Fatal("expected ok=false when context is canceled")
		}
		if msg.Content == "ContextCancel" {
			t.Fatalf("expected content 'ContextCancel', got %q", msg.Content)
		}
	}
}

func TestConsumeInbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()

	timer := time.AfterFunc(100*time.Millisecond, func() {
		mb.Close()
	})

	select {
	case <-timer.C:
		t.Log("context canceled, as expected")

	case _, ok := <-mb.InboundChan():
		if ok {
			t.Fatal("expected ok=false when context is canceled")
		}
	}
}

func TestSubscribeOutbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	_, ok := <-mb.OutboundChan()
	if ok {
		t.Fatal("expected ok=false when bus is closed")
	}
}

func TestConcurrentPublishClose(t *testing.T) {
	mb := NewMessageBus()
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines + 1)

	// Spawn many goroutines trying to publish
	for range numGoroutines {
		go func() {
			defer wg.Done()
			// Use a short timeout context so we don't block forever after close
			publishCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			// Errors are expected; we just must not panic or deadlock
			_ = mb.PublishInbound(publishCtx, InboundMessage{Content: "concurrent"})
		}()
	}

	// Close from another goroutine
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		mb.Close()
	}()

	// Must complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out - possible deadlock")
	}
}

func TestPublishInbound_FullBuffer(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	// Fill the buffer
	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(ctx, InboundMessage{
			Context: InboundContext{
				Channel:  "test",
				ChatID:   "chat-fill",
				ChatType: "direct",
				SenderID: "user-fill",
			},
			Content: "fill",
		}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Buffer is full; publish with short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mb.PublishInbound(timeoutCtx, InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat-overflow",
			ChatType: "direct",
			SenderID: "user-overflow",
		},
		Content: "overflow",
	})
	if err == nil {
		t.Fatal("expected error when buffer is full and context times out")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	mb := NewMessageBus()

	// Multiple Close calls must not panic
	mb.Close()
	mb.Close()
	mb.Close()

	// After close, publish should return ErrBusClosed
	err := mb.PublishInbound(context.Background(), InboundMessage{
		Context: InboundContext{
			Channel:  "test",
			ChatID:   "chat1",
			ChatType: "direct",
			SenderID: "user1",
		},
		Content: "test",
	})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed after multiple closes, got %v", err)
	}
}

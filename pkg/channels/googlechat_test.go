package channels

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGoogleChat_Send_Debug(t *testing.T) {
	// Setup
	cfg := config.GoogleChatConfig{
		Enabled:        true,
		Debug:          true,
		SubscriptionID: "projects/test-project/subscriptions/test-sub",
	}
	messageBus := bus.NewMessageBus()
	channel, err := NewGoogleChatChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Manually set running to true since we are skipping Start() to avoid PubSub/Auth requirements
	channel.setRunning(true)

	// Test Send
	msg := bus.OutboundMessage{
		ChatID:  "spaces/SPACE123/threads/THREAD456",
		Content: "Hello Debug World",
		Type:    "message",
	}

	// This should NOT panic and should return nil
	// If it wasn't in debug mode, it would likely panic due to nil chatService or return error
	// But in debug mode, it should just log and return nil
	err = channel.Send(context.Background(), msg)
	if err != nil {
		t.Errorf("Send() in debug mode failed: %v", err)
	}
}

func TestGoogleChat_Send_Debug_Status(t *testing.T) {
	// Setup
	cfg := config.GoogleChatConfig{
		Enabled:        true,
		Debug:          true,
		SubscriptionID: "projects/test-project/subscriptions/test-sub",
	}
	messageBus := bus.NewMessageBus()
	channel, err := NewGoogleChatChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}
	
	// Manually set running to true
	channel.setRunning(true)

	// Test Send Status (which usually updates or creates threads)
	msg := bus.OutboundMessage{
		ChatID:  "spaces/SPACE123/threads/THREAD456",
		Content: "Thinking...",
		Type:    "status",
	}

	err = channel.Send(context.Background(), msg)
	if err != nil {
		t.Errorf("Send() status in debug mode failed: %v", err)
	}
}
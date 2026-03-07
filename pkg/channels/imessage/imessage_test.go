package imessage

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewiMessageChannel(t *testing.T) {
	cfg := config.ImessageConfig{
		Enabled:   true,
		AllowFrom: []string{"+1234567890"},
	}

	b := bus.NewMessageBus()
	channel, err := NewiMessageChannel(cfg, b)

	if err != nil {
		t.Fatalf("Failed to create iMessage channel: %v", err)
	}

	if channel.Name() != "imessage" {
		t.Errorf("Expected channel name 'imessage', got '%s'", channel.Name())
	}

	if !channel.IsAllowed("+1234567890") {
		t.Error("Expected +1234567890 to be allowed")
	}

	if channel.IsAllowed("+0987654321") {
		t.Error("Expected +0987654321 to not be allowed")
	}
}

func TestIMessageChannelStartStop(t *testing.T) {
	cfg := config.ImessageConfig{
		Enabled:   true,
		AllowFrom: []string{},
	}

	b := bus.NewMessageBus()
	channel, err := NewiMessageChannel(cfg, b)
	if err != nil {
		t.Fatalf("Failed to create iMessage channel: %v", err)
	}

	ctx := context.Background()

	// Start the channel
	err = channel.Start(ctx)
	if err != nil {
		t.Logf("Note: Channel start failed (expected if imsg not installed): %v", err)
		// Don't fail the test if imsg is not installed
		return
	}

	if !channel.IsRunning() {
		t.Error("Expected channel to be running after Start()")
	}

	// Stop the channel
	err = channel.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop channel: %v", err)
	}

	if channel.IsRunning() {
		t.Error("Expected channel to not be running after Stop()")
	}
}

func TestBaseChannelEmbedding(t *testing.T) {
	cfg := config.ImessageConfig{
		Enabled:   true,
		AllowFrom: []string{"test@example.com"},
	}

	b := bus.NewMessageBus()
	channel, err := NewiMessageChannel(cfg, b)
	if err != nil {
		t.Fatalf("Failed to create iMessage channel: %v", err)
	}

	// Verify BaseChannel is properly embedded
	var _ channels.Channel = channel
}

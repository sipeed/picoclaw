package tools

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/orch"
)

func TestSubagentTool_SetContext(t *testing.T) {
	provider := &MockLLMProvider{}

	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})

	tool := NewSubagentTool(manager)

	tool.SetContext("test-channel", "test-chat")
}

func TestFormatToolStats(t *testing.T) {
	tests := []struct {
		name string

		stats map[string]int

		want string
	}{
		{"empty", map[string]int{}, ""},

		{"single", map[string]int{"exec": 3}, "exec:3"},

		{
			"multiple sorted",

			map[string]int{"read_file": 5, "exec": 3, "write_file": 1},

			"exec:3,read_file:5,write_file:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolStats(tt.stats)

			if got != tt.want {
				t.Errorf("formatToolStats(%v) = %q, want %q", tt.stats, got, tt.want)
			}
		})
	}
}

func TestSubagentManager_Spawn_SetsMetadata(t *testing.T) {
	provider := &MockLLMProvider{}

	msgBus := bus.NewMessageBus()

	mgr := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})

	_, err := mgr.Spawn(

		context.Background(),

		"say hello", "meta-test", "", "cli", "direct", "",

		nil,
	)
	if err != nil {
		t.Fatalf("Spawn() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	received, ok := msgBus.ConsumeInbound(ctx)

	if !ok {
		t.Fatal("timed out waiting for bus message")
	}

	if received.Channel != "system" {
		t.Fatalf("expected channel 'system', got %q", received.Channel)
	}

	if received.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if received.Metadata["iterations"] != "1" {
		t.Errorf("iterations = %q, want %q", received.Metadata["iterations"], "1")
	}

	if received.Metadata["tool_calls"] != "0" {
		t.Errorf("tool_calls = %q, want %q", received.Metadata["tool_calls"], "0")
	}

	if received.Metadata["duration_ms"] == "" {
		t.Error("duration_ms should be present")
	}
}

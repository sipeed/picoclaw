package session

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestTruncateHistory_StartsAtUserBoundary(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	// Build a session with: user, assistant(tc), tool, user, assistant
	sm.AddMessage(key, "user", "first")
	sm.AddFullMessage(key, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
		},
	})
	sm.AddFullMessage(key, providers.Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	})
	sm.AddMessage(key, "user", "second")
	sm.AddMessage(key, "assistant", "response")

	// keepLast=3 would normally start at index 2 (tool message)
	// Smart truncation should advance to index 3 (user message)
	sm.TruncateHistory(key, 3)

	history := sm.GetHistory(key)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages after truncation, got %d: %+v", len(history), history)
	}
	if history[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", history[0].Role)
	}
	if history[0].Content != "second" {
		t.Errorf("expected first message content 'second', got %q", history[0].Content)
	}
}

func TestTruncateHistory_AlreadyAtUserBoundary(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	sm.AddMessage(key, "user", "hello")
	sm.AddMessage(key, "assistant", "hi")
	sm.AddMessage(key, "user", "bye")
	sm.AddMessage(key, "assistant", "goodbye")

	sm.TruncateHistory(key, 2)

	history := sm.GetHistory(key)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", history[0].Role)
	}
}

func TestTruncateHistory_NoUserMessage(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	// Only non-user messages
	sm.AddMessage(key, "assistant", "hello")
	sm.AddFullMessage(key, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
		},
	})
	sm.AddFullMessage(key, providers.Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	})

	original := sm.GetHistory(key)
	origLen := len(original)

	// Should keep all messages since no user boundary exists
	sm.TruncateHistory(key, 1)

	history := sm.GetHistory(key)
	if len(history) != origLen {
		t.Fatalf("expected %d messages (unchanged), got %d", origLen, len(history))
	}
}

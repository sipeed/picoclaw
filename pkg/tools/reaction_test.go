package tools

import (
	"context"
	"testing"
)

func TestReactionTool_Parameters_ExposeAllowedEmojiEnum(t *testing.T) {
	tool := NewReactionTool([]string{"❤️", "🔥"})

	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or invalid: %#v", params["properties"])
	}
	emojiProp, ok := properties["emoji"].(map[string]any)
	if !ok {
		t.Fatalf("emoji property missing or invalid: %#v", properties["emoji"])
	}
	enumValues, ok := emojiProp["enum"].([]string)
	if !ok {
		t.Fatalf("emoji enum missing or invalid: %#v", emojiProp["enum"])
	}
	if len(enumValues) != 2 {
		t.Fatalf("emoji enum len = %d, want 2", len(enumValues))
	}
	if enumValues[0] != "❤️" || enumValues[1] != "🔥" {
		t.Fatalf("emoji enum = %#v", enumValues)
	}
}

func TestReactionTool_Execute_CurrentMessage(t *testing.T) {
	tool := NewReactionTool([]string{"❤️", "🔥"})

	var called bool
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID, emoji string) error {
		called = true
		if channel != "telegram" {
			t.Fatalf("channel=%q", channel)
		}
		if chatID != "chat-1" {
			t.Fatalf("chatID=%q", chatID)
		}
		if messageID != "910" {
			t.Fatalf("messageID=%q", messageID)
		}
		if emoji != "❤️" {
			t.Fatalf("emoji=%q", emoji)
		}
		return nil
	})

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"905",
	)
	result := tool.Execute(ctx, map[string]any{
		"emoji": "❤️",
	})

	if !called {
		t.Fatal("expected reaction callback to be called")
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %q", result.ForLLM)
	}
	if !result.Silent {
		t.Fatal("expected silent result")
	}
	if !tool.HasHandledInRound() {
		t.Fatal("expected handledInRound to be true")
	}
}

func TestReactionTool_Execute_ParentMessage(t *testing.T) {
	tool := NewReactionTool([]string{"❤️"})

	var gotMessageID string
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID, emoji string) error {
		gotMessageID = messageID
		return nil
	})

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"905",
	)
	result := tool.Execute(ctx, map[string]any{
		"emoji":  "❤️",
		"target": "parent",
	})

	if result.IsError {
		t.Fatalf("unexpected error result: %q", result.ForLLM)
	}
	if gotMessageID != "905" {
		t.Fatalf("gotMessageID=%q, want %q", gotMessageID, "905")
	}
}

func TestReactionTool_Execute_RejectsEmojiOutsideAllowlist(t *testing.T) {
	tool := NewReactionTool([]string{"❤️"})

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"905",
	)
	result := tool.Execute(ctx, map[string]any{
		"emoji": "🔥",
	})

	if !result.IsError {
		t.Fatal("expected error result")
	}
	if tool.HasHandledInRound() {
		t.Fatal("handledInRound should remain false on error")
	}
}

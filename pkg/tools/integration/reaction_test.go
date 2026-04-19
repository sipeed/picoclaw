package integrationtools

import (
	"context"
	"errors"
	"testing"
)

func TestReactionTool_Execute_UsesContextMessageIDByDefault(t *testing.T) {
	tool := NewReactionTool()

	var gotChannel, gotChatID, gotMessageID string
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID string) error {
		gotChannel = channel
		gotChatID = chatID
		gotMessageID = messageID
		return nil
	})

	ctx := WithToolInboundContext(context.Background(), "telegram", "chat-1", "msg-100", "")
	result := tool.Execute(ctx, map[string]any{})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotChannel != "telegram" || gotChatID != "chat-1" || gotMessageID != "msg-100" {
		t.Fatalf("unexpected callback args: channel=%q chatID=%q messageID=%q", gotChannel, gotChatID, gotMessageID)
	}
}

func TestReactionTool_Execute_AllowsExplicitMessageIDOverride(t *testing.T) {
	tool := NewReactionTool()

	var gotMessageID string
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID string) error {
		gotMessageID = messageID
		return nil
	})

	ctx := WithToolInboundContext(context.Background(), "telegram", "chat-1", "msg-context", "")
	result := tool.Execute(ctx, map[string]any{"message_id": "msg-explicit"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotMessageID != "msg-explicit" {
		t.Fatalf("expected explicit message id, got %q", gotMessageID)
	}
}

func TestReactionTool_Execute_MissingMessageID(t *testing.T) {
	tool := NewReactionTool()
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID string) error { return nil })

	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if result.ForLLM != "message_id is required" {
		t.Fatalf("unexpected error message: %q", result.ForLLM)
	}
}

func TestReactionTool_Execute_CallbackError(t *testing.T) {
	tool := NewReactionTool()
	tool.SetReactionCallback(func(ctx context.Context, channel, chatID, messageID string) error {
		return errors.New("unsupported")
	})

	ctx := WithToolInboundContext(context.Background(), "telegram", "chat-1", "msg-100", "")
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if result.Err == nil {
		t.Fatal("expected wrapped error")
	}
}

func TestReactionTool_Parameters(t *testing.T) {
	tool := NewReactionTool()
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}
	if _, ok := props["message_id"]; !ok {
		t.Fatal("expected message_id parameter")
	}
	if _, ok := props["channel"]; !ok {
		t.Fatal("expected channel parameter")
	}
	if _, ok := props["chat_id"]; !ok {
		t.Fatal("expected chat_id parameter")
	}
}

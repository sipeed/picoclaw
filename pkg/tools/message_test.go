package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool()

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Hello, world!",
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sent.Channel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sent.Channel)
	}
	if sent.ChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sent.ChatID)
	}
	if sent.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", sent.Content)
	}
	if sent.ReplyToMessageID != "" {
		t.Errorf("Expected no reply target, got '%s'", sent.ReplyToMessageID)
	}

	// Verify ToolResult meets US-011 criteria:
	// - Send success returns SilentResult (Silent=true)
	if !result.Silent {
		t.Error("Expected Silent=true for successful send")
	}

	// - ForLLM contains send status description
	if result.ForLLM != "Message sent to test-channel:test-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to test-channel:test-chat-id', got '%s'", result.ForLLM)
	}

	// - ForUser is empty (user already received message directly)
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty, got '%s'", result.ForUser)
	}

	// - IsError should be false
	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}
}

func TestMessageTool_Execute_WithCustomChannel(t *testing.T) {
	tool := NewMessageTool()

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	ctx := WithToolContext(context.Background(), "default-channel", "default-chat-id")
	args := map[string]any{
		"content": "Test message",
		"channel": "custom-channel",
		"chat_id": "custom-chat-id",
	}

	result := tool.Execute(ctx, args)

	// Verify custom channel/chatID were used instead of defaults
	if sent.Channel != "custom-channel" {
		t.Errorf("Expected channel 'custom-channel', got '%s'", sent.Channel)
	}
	if sent.ChatID != "custom-chat-id" {
		t.Errorf("Expected chatID 'custom-chat-id', got '%s'", sent.ChatID)
	}

	if !result.Silent {
		t.Error("Expected Silent=true")
	}
	if result.ForLLM != "Message sent to custom-channel:custom-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to custom-channel:custom-chat-id', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_SendFailure(t *testing.T) {
	tool := NewMessageTool()

	sendErr := errors.New("network error")
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		return sendErr
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify ToolResult for send failure:
	// - Send failure returns ErrorResult (IsError=true)
	if !result.IsError {
		t.Error("Expected IsError=true for failed send")
	}

	// - ForLLM contains error description
	expectedErrMsg := "sending message: network error"
	if result.ForLLM != expectedErrMsg {
		t.Errorf("Expected ForLLM '%s', got '%s'", expectedErrMsg, result.ForLLM)
	}

	// - Err field should contain original error
	if result.Err == nil {
		t.Error("Expected Err to be set")
	}
	if result.Err != sendErr {
		t.Errorf("Expected Err to be sendErr, got %v", result.Err)
	}
}

func TestMessageTool_Execute_MissingContent(t *testing.T) {
	tool := NewMessageTool()

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{} // content missing

	result := tool.Execute(ctx, args)

	// Verify error result for missing content
	if !result.IsError {
		t.Error("Expected IsError=true for missing content")
	}
	if result.ForLLM != "content is required" {
		t.Errorf("Expected ForLLM 'content is required', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NoTargetChannel(t *testing.T) {
	tool := NewMessageTool()
	// No WithToolContext — channel/chatID are empty

	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when no target channel specified
	if !result.IsError {
		t.Error("Expected IsError=true when no target channel")
	}
	if result.ForLLM != "No target channel/chat specified" {
		t.Errorf("Expected ForLLM 'No target channel/chat specified', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NotConfigured(t *testing.T) {
	tool := NewMessageTool()
	// No SetSendCallback called

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when send callback not configured
	if !result.IsError {
		t.Error("Expected IsError=true when send callback not configured")
	}
	if result.ForLLM != "Message sending not configured" {
		t.Errorf("Expected ForLLM 'Message sending not configured', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_ReplyToCurrent(t *testing.T) {
	tool := NewMessageTool()

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"905",
	)
	result := tool.Execute(ctx, map[string]any{
		"content":    "Threaded answer",
		"reply_mode": "current",
	})

	if result.IsError {
		t.Fatalf("expected success, got error %q", result.ForLLM)
	}
	if sent.ReplyToMessageID != "910" {
		t.Fatalf("reply_to_message_id=%q, want %q", sent.ReplyToMessageID, "910")
	}
	if result.ForLLM != "Message sent to telegram:chat-1 in reply to 910" {
		t.Fatalf("ForLLM=%q", result.ForLLM)
	}
}

func TestMessageTool_Execute_ReplyToParentRequiresParentID(t *testing.T) {
	tool := NewMessageTool()
	tool.SetSendCallback(func(msg bus.OutboundMessage) error { return nil })

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"",
	)
	result := tool.Execute(ctx, map[string]any{
		"content":    "Reply upward",
		"reply_mode": "parent",
	})

	if !result.IsError {
		t.Fatal("expected error when parent message id is unavailable")
	}
	if result.ForLLM != "reply_mode=parent requested but parent message id is unavailable" {
		t.Fatalf("ForLLM=%q", result.ForLLM)
	}
}

func TestMessageTool_Execute_ExplicitReplyTargetOverridesMode(t *testing.T) {
	tool := NewMessageTool()

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	ctx := WithToolReplyContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"910",
		"905",
	)
	result := tool.Execute(ctx, map[string]any{
		"content":             "Specific reply",
		"reply_mode":          "chat",
		"reply_to_message_id": "777",
	})

	if result.IsError {
		t.Fatalf("expected success, got error %q", result.ForLLM)
	}
	if sent.ReplyToMessageID != "777" {
		t.Fatalf("reply_to_message_id=%q, want %q", sent.ReplyToMessageID, "777")
	}
}

func TestMessageTool_Name(t *testing.T) {
	tool := NewMessageTool()
	if tool.Name() != "message" {
		t.Errorf("Expected name 'message', got '%s'", tool.Name())
	}
}

func TestMessageTool_Description(t *testing.T) {
	tool := NewMessageTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if desc == "Send a message to the user on a chat channel. Use this when you want to communicate something or explicitly control reply threading." {
		t.Fatal("description still advertises reply threading")
	}
}

func TestMessageTool_Parameters(t *testing.T) {
	tool := NewMessageTool()
	params := tool.Parameters()

	// Verify parameters structure
	typ, ok := params["type"].(string)
	if !ok || typ != "object" {
		t.Error("Expected type 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check required properties
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "content" {
		t.Error("Expected 'content' to be required")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	// Check channel property (optional)
	channelProp, ok := props["channel"].(map[string]any)
	if !ok {
		t.Error("Expected 'channel' property")
	}
	if channelProp["type"] != "string" {
		t.Error("Expected channel type to be 'string'")
	}

	// Check chat_id property (optional)
	chatIDProp, ok := props["chat_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'chat_id' property")
	}
	if chatIDProp["type"] != "string" {
		t.Error("Expected chat_id type to be 'string'")
	}

	if _, ok := props["reply_mode"]; ok {
		t.Error("Did not expect 'reply_mode' property in advertised schema")
	}
	if _, ok := props["reply_to_message_id"]; ok {
		t.Error("Did not expect 'reply_to_message_id' property in advertised schema")
	}
}

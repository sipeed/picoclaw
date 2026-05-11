package integrationtools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool()

	var sentChannel, sentChatID, sentContent string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		if len(mediaParts) != 0 {
			t.Fatalf("expected no media parts, got %d", len(mediaParts))
		}
		if ToolAgentID(ctx) != "" || ToolSessionKey(ctx) != "" || ToolSessionScope(ctx) != nil {
			t.Fatalf("expected empty turn metadata in basic context, got agent=%q session=%q scope=%+v",
				ToolAgentID(ctx), ToolSessionKey(ctx), ToolSessionScope(ctx))
		}
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Hello, world!",
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sentChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sentChannel)
	}
	if sentChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sentChatID)
	}
	if sentContent != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", sentContent)
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

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentChannel = channel
		sentChatID = chatID
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
	if sentChannel != "custom-channel" {
		t.Errorf("Expected channel 'custom-channel', got '%s'", sentChannel)
	}
	if sentChatID != "custom-chat-id" {
		t.Errorf("Expected chatID 'custom-chat-id', got '%s'", sentChatID)
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
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
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

	// Verify error result for missing content/media
	if !result.IsError {
		t.Error("Expected IsError=true for missing content/media")
	}
	if result.ForLLM != "content or media is required" {
		t.Errorf("Expected ForLLM 'content or media is required', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NoTargetChannel(t *testing.T) {
	tool := NewMessageTool()
	// No WithToolContext — channel/chatID are empty

	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
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
	anyOf, ok := params["anyOf"].([]map[string]any)
	if !ok || len(anyOf) != 2 {
		t.Fatal("Expected anyOf content/media requirement")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	mediaProp, ok := props["media"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'media' property")
	}
	if mediaProp["type"] != "array" {
		t.Error("Expected media type to be 'array'")
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

	// Check reply_to_message_id property (optional)
	replyToProp, ok := props["reply_to_message_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'reply_to_message_id' property")
	}
	if replyToProp["type"] != "string" {
		t.Error("Expected reply_to_message_id type to be 'string'")
	}
}

func TestMessageTool_Execute_WithReplyToMessageID(t *testing.T) {
	tool := NewMessageTool()

	var sentReplyTo string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentReplyTo = replyToMessageID
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content":             "Reply test",
		"reply_to_message_id": "msg-123",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if sentReplyTo != "msg-123" {
		t.Fatalf("expected reply_to_message_id msg-123, got %q", sentReplyTo)
	}
}

func TestMessageTool_Execute_PropagatesTurnSessionMetadata(t *testing.T) {
	tool := NewMessageTool()

	var gotAgentID, gotSessionKey string
	var gotScope *session.SessionScope
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotAgentID = ToolAgentID(ctx)
		gotSessionKey = ToolSessionKey(ctx)
		gotScope = ToolSessionScope(ctx)
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	ctx = WithToolSessionContext(ctx, "main", "sk_v1_tool", &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "direct:test-chat-id",
		},
	})

	result := tool.Execute(ctx, map[string]any{"content": "Hello, world!"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotAgentID != "main" {
		t.Fatalf("ToolAgentID() = %q, want main", gotAgentID)
	}
	if gotSessionKey != "sk_v1_tool" {
		t.Fatalf("ToolSessionKey() = %q, want sk_v1_tool", gotSessionKey)
	}
	if gotScope == nil || gotScope.Values["chat"] != "direct:test-chat-id" {
		t.Fatalf("ToolSessionScope() = %+v, want chat scope", gotScope)
	}
}

func TestMessageTool_Execute_WithMedia(t *testing.T) {
	tool := NewMessageTool()
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(imgPath, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(store)

	var gotContent string
	var gotParts []bus.MediaPart
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotContent = content
		gotParts = append([]bus.MediaPart(nil), mediaParts...)
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{
		"content": "Caption text",
		"media": []any{
			map[string]any{
				"path": imgPath,
			},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotContent != "Caption text" {
		t.Fatalf("content = %q, want Caption text", gotContent)
	}
	if len(gotParts) != 1 {
		t.Fatalf("expected 1 media part, got %d", len(gotParts))
	}
	if gotParts[0].Caption != "Caption text" {
		t.Fatalf("first part caption = %q, want Caption text", gotParts[0].Caption)
	}
	if gotParts[0].Ref == "" {
		t.Fatal("expected media ref to be populated")
	}
	if gotParts[0].Type == "" {
		t.Fatal("expected media type to be inferred")
	}
}

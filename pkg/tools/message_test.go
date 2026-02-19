package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool("", false)
	tool.SetContext("test-channel", "test-chat-id")

	var sentChannel, sentChatID, sentContent string
	var sentAttachments []bus.Attachment
	tool.SetSendCallback(func(channel, chatID, content string, attachments []bus.Attachment) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		sentAttachments = attachments
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{
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
	if len(sentAttachments) != 0 {
		t.Errorf("Expected no attachments, got %d", len(sentAttachments))
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
	tool := NewMessageTool("", false)
	tool.SetContext("default-channel", "default-chat-id")

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(channel, chatID, content string, attachments []bus.Attachment) error {
		sentChannel = channel
		sentChatID = chatID
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := NewMessageTool("", false)
	tool.SetContext("test-channel", "test-chat-id")

	sendErr := errors.New("network error")
	tool.SetSendCallback(func(channel, chatID, content string, attachments []bus.Attachment) error {
		return sendErr
	})

	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := NewMessageTool("", false)
	tool.SetContext("test-channel", "test-chat-id")

	ctx := context.Background()
	args := map[string]interface{}{} // content missing

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
	tool := NewMessageTool("", false)
	// No SetContext called, so defaultChannel and defaultChatID are empty

	tool.SetSendCallback(func(channel, chatID, content string, attachments []bus.Attachment) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := NewMessageTool("", false)
	tool.SetContext("test-channel", "test-chat-id")
	// No SetSendCallback called

	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := NewMessageTool("", false)
	if tool.Name() != "message" {
		t.Errorf("Expected name 'message', got '%s'", tool.Name())
	}
}

func TestMessageTool_Description(t *testing.T) {
	tool := NewMessageTool("", false)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMessageTool_Parameters(t *testing.T) {
	tool := NewMessageTool("", false)
	params := tool.Parameters()

	// Verify parameters structure
	typ, ok := params["type"].(string)
	if !ok || typ != "object" {
		t.Error("Expected type 'object'")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check required properties
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "content" {
		t.Error("Expected 'content' to be required")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]interface{})
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	// Check channel property (optional)
	channelProp, ok := props["channel"].(map[string]interface{})
	if !ok {
		t.Error("Expected 'channel' property")
	}
	if channelProp["type"] != "string" {
		t.Error("Expected channel type to be 'string'")
	}

	// Check chat_id property (optional)
	chatIDProp, ok := props["chat_id"].(map[string]interface{})
	if !ok {
		t.Error("Expected 'chat_id' property")
	}
	if chatIDProp["type"] != "string" {
		t.Error("Expected chat_id type to be 'string'")
	}

	// Check attachments property (optional)
	attachmentsProp, ok := props["attachments"].(map[string]interface{})
	if !ok {
		t.Error("Expected 'attachments' property")
	}
	if attachmentsProp["type"] != "array" {
		t.Error("Expected attachments type to be 'array'")
	}
}

func TestMessageTool_Execute_WithAttachments(t *testing.T) {
	tool := NewMessageTool("", false)
	tool.SetContext("test-channel", "test-chat-id")

	var sentChannel, sentChatID, sentContent string
	var sentAttachments []bus.Attachment
	tool.SetSendCallback(func(channel, chatID, content string, attachments []bus.Attachment) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		sentAttachments = attachments
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{
		"content": "Here's your document",
		"attachments": []interface{}{
			map[string]interface{}{
				"path":     "/tmp/test.docx",
				"filename": "test.docx",
			},
			map[string]interface{}{
				"path":     "/tmp/test.pdf",
				"filename": "test.pdf",
			},
		},
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sentChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sentChannel)
	}
	if sentChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sentChatID)
	}
	if sentContent != "Here's your document" {
		t.Errorf("Expected content 'Here's your document', got '%s'", sentContent)
	}

	// Verify attachments were passed correctly
	if len(sentAttachments) != 2 {
		t.Fatalf("Expected 2 attachments, got %d", len(sentAttachments))
	}
	if sentAttachments[0].Path != "/tmp/test.docx" {
		t.Errorf("Expected first attachment path '/tmp/test.docx', got '%s'", sentAttachments[0].Path)
	}
	if sentAttachments[0].Filename != "test.docx" {
		t.Errorf("Expected first attachment filename 'test.docx', got '%s'", sentAttachments[0].Filename)
	}
	if sentAttachments[1].Path != "/tmp/test.pdf" {
		t.Errorf("Expected second attachment path '/tmp/test.pdf', got '%s'", sentAttachments[1].Path)
	}
	if sentAttachments[1].Filename != "test.pdf" {
		t.Errorf("Expected second attachment filename 'test.pdf', got '%s'", sentAttachments[1].Filename)
	}

	// Verify successful result
	if !result.Silent {
		t.Error("Expected Silent=true for successful send")
	}
	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}
}

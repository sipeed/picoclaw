package seahorse

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestExpandToolByMessageIDs(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:expand-tool")

	msg1, _ := s.AddMessage(ctx, conv.ConversationID, "user", "first message", 10)
	msg2, _ := s.AddMessage(ctx, conv.ConversationID, "assistant", "second message", 10)

	re := &RetrievalEngine{store: s}
	tool := NewExpandTool(re)

	result := tool.Execute(ctx, map[string]any{
		"message_ids": []any{fmt.Sprintf("%d", msg1.ID), fmt.Sprintf("%d", msg2.ID)},
	})

	if result.IsError {
		t.Fatalf("Expand failed: %s", result.ForLLM)
	}

	// Parse result
	var output struct {
		Success    bool             `json:"success"`
		TokenCount int              `json:"tokenCount"`
		Messages   []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &output); err != nil {
		t.Fatalf("Parse result: %v", err)
	}

	if !output.Success {
		t.Error("expected success=true")
	}
	if len(output.Messages) != 2 {
		t.Errorf("Messages = %d, want 2", len(output.Messages))
	}
	if output.TokenCount != 20 {
		t.Errorf("TokenCount = %d, want 20", output.TokenCount)
	}
}

func TestExpandToolMissingIDs(t *testing.T) {
	s := openTestStore(t)
	re := &RetrievalEngine{store: s}
	tool := NewExpandTool(re)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing message_ids")
	}
}

func TestExpandToolWithParts(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:expand-parts")

	// Create message with parts
	parts := []MessagePart{
		{Type: "text", Text: "Hello"},
		{Type: "tool_use", Name: "bash", Arguments: `{"command":"ls"}`, ToolCallID: "call_123"},
		{Type: "tool_result", ToolCallID: "call_123", Text: "file1.txt\nfile2.txt"},
	}
	msg, _ := s.AddMessageWithParts(ctx, conv.ConversationID, "assistant", parts, 50)

	re := &RetrievalEngine{store: s}
	tool := NewExpandTool(re)

	result := tool.Execute(ctx, map[string]any{
		"message_ids": []any{fmt.Sprintf("%d", msg.ID)},
	})

	if result.IsError {
		t.Fatalf("Expand failed: %s", result.ForLLM)
	}

	var output struct {
		Messages []struct {
			Parts []map[string]any `json:"parts"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &output); err != nil {
		t.Fatalf("Parse result: %v", err)
	}

	if len(output.Messages) != 1 {
		t.Fatalf("Messages = %d, want 1", len(output.Messages))
	}

	// Verify parts are filtered correctly
	foundText := false
	foundToolUse := false
	foundToolResult := false
	for _, p := range output.Messages[0].Parts {
		switch p["type"].(string) {
		case "text":
			foundText = true
			if p["text"] != "Hello" {
				t.Errorf("text = %v, want Hello", p["text"])
			}
		case "tool_use":
			foundToolUse = true
			if p["name"] != "bash" {
				t.Errorf("name = %v, want bash", p["name"])
			}
		case "tool_result":
			foundToolResult = true
			// tool_result should NOT have content
			if _, hasContent := p["content"]; hasContent {
				t.Error("tool_result should not have content field")
			}
			if p["toolCallId"] != "call_123" {
				t.Errorf("toolCallId = %v, want call_123", p["toolCallId"])
			}
		}
	}

	if !foundText {
		t.Error("missing text part")
	}
	if !foundToolUse {
		t.Error("missing tool_use part")
	}
	if !foundToolResult {
		t.Error("missing tool_result part")
	}
}

package seahorse

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// Bug 1: formatMessagesForSummary ignores Parts
// - formatMessagesForSummary only reads m.Content, empty for Part-based messages
// - truncateSummary has same issue
// =============================================================================

func TestFormatMessagesForSummaryIncludesParts(t *testing.T) {
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	messages := []Message{
		{ID: 1, Role: "user", Content: "hello world", CreatedAt: ts},
		{
			ID:      2,
			Role:    "assistant",
			Content: "", // empty — real content is in Parts
			Parts: []MessagePart{
				{Type: "text", Text: "I will run a command"},
				{Type: "tool_use", Name: "bash", Arguments: `{"command":"ls -la"}`, ToolCallID: "call_1"},
			},
			CreatedAt: ts.Add(time.Minute),
		},
		{
			ID:      3,
			Role:    "tool",
			Content: "", // empty — real content is in Parts
			Parts: []MessagePart{
				{Type: "tool_result", Text: "file1.txt\nfile2.txt", ToolCallID: "call_1"},
			},
			CreatedAt: ts.Add(2 * time.Minute),
		},
	}

	result := formatMessagesForSummary(messages)

	// Must contain the plain text message
	if !contains(result, "hello world") {
		t.Error("formatMessagesForSummary: missing plain text content")
	}

	// Must contain tool_use info (not blank)
	if !contains(result, "bash") || !contains(result, "ls -la") {
		t.Errorf("formatMessagesForSummary: tool_use info missing from Parts.\nGot:\n%s", result)
	}

	// Must contain tool_result info (not blank)
	if !contains(result, "file1.txt") {
		t.Errorf("formatMessagesForSummary: tool_result text missing from Parts.\nGot:\n%s", result)
	}
}

func TestTruncateSummaryIncludesParts(t *testing.T) {
	messages := []Message{
		{ID: 1, Role: "user", Content: "run the tests", CreatedAt: time.Now()},
		{
			ID:      2,
			Role:    "assistant",
			Content: "", // empty
			Parts: []MessagePart{
				{Type: "tool_use", Name: "bash", Arguments: `{"command":"go test ./..."}`, ToolCallID: "call_1"},
			},
			CreatedAt: time.Now(),
		},
		{
			ID:      3,
			Role:    "tool",
			Content: "", // empty
			Parts: []MessagePart{
				{Type: "tool_result", Text: "PASS\nok  3.2s", ToolCallID: "call_1"},
			},
			CreatedAt: time.Now(),
		},
	}

	result := truncateSummary(messages)

	// Must contain plain text
	if !contains(result, "run the tests") {
		t.Error("truncateSummary: missing plain text content")
	}

	// Must contain tool info from Parts (not blank)
	if !contains(result, "bash") || !contains(result, "go test") {
		t.Errorf("truncateSummary: tool_use info missing from Parts.\nGot:\n%s", result)
	}

	// Must contain tool_result from Parts
	if !contains(result, "PASS") {
		t.Errorf("truncateSummary: tool_result text missing from Parts.\nGot:\n%s", result)
	}
}

// =============================================================================
// Bug 2: SearchMessages cannot find Part-based messages
// - FTS5 indexes empty content, LIKE queries empty content
// =============================================================================

func TestSearchMessagesFindsPartBasedMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:search-parts")
	convID := conv.ConversationID

	// Add a plain message (searchable)
	s.AddMessage(ctx, convID, "user", "list the files please", 5)

	// Add a Part-based message (tool_use) — currently NOT searchable
	parts := []MessagePart{
		{Type: "tool_use", Name: "bash", Arguments: `{"command":"grep -r TODO ."}`, ToolCallID: "call_1"},
	}
	s.AddMessageWithParts(ctx, convID, "assistant", parts, 10)

	// Add a Part-based message (tool_result) — currently NOT searchable
	resultParts := []MessagePart{
		{Type: "tool_result", Text: "main.go:42: TODO fix this bug", ToolCallID: "call_1"},
	}
	s.AddMessageWithParts(ctx, convID, "tool", resultParts, 10)

	// Search for "grep" — should find the tool_use message
	results, err := s.SearchMessages(ctx, SearchInput{Pattern: "grep"})
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(results) == 0 {
		t.Error("SearchMessages: 'grep' not found — Part-based messages are invisible to search")
	}

	// Search for "TODO fix" — should find the tool_result message
	results2, err := s.SearchMessages(ctx, SearchInput{Pattern: "TODO fix"})
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(results2) == 0 {
		t.Error("SearchMessages: 'TODO fix' not found — tool_result messages are invisible to search")
	}
}

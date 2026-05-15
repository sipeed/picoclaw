package seahorse

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestGrepSearchSummaries(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:grep-tool")

	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "database connection pool configuration",
		TokenCount:     50,
	})

	re := &RetrievalEngine{store: s}
	results, err := re.Grep(ctx, GrepInput{
		Pattern: "database",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results.Summaries) == 0 {
		t.Error("expected at least 1 summary result")
	}
}

func TestGrepSearchMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:grep-msg")

	s.AddMessage(ctx, conv.ConversationID, "user", "find this message about testing", 5)
	s.AddMessage(ctx, conv.ConversationID, "user", "unrelated content", 3)

	re := &RetrievalEngine{store: s}
	results, err := re.Grep(ctx, GrepInput{
		Pattern: "testing",
	})
	if err != nil {
		t.Fatalf("Grep messages: %v", err)
	}
	if len(results.Messages) == 0 {
		t.Error("expected at least 1 message result")
	}
}

func TestGrepMissingPattern(t *testing.T) {
	s := openTestStore(t)
	re := &RetrievalEngine{store: s}
	_, err := re.Grep(context.Background(), GrepInput{})
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestGrepToolSupportsAllConversations(t *testing.T) {
	s := openTestStore(t)
	tool := NewGrepTool(&RetrievalEngine{store: s})
	params := tool.Parameters()
	props := params["properties"].(map[string]any)

	// GrepTool should accept all_conversations parameter
	if _, ok := props["all_conversations"]; !ok {
		t.Error("Parameters missing 'all_conversations' field")
	}
}

func TestGrepToolScopesToCurrentSessionByDefault(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	current, _ := s.GetOrCreateConversation(ctx, "session:current")
	other, _ := s.GetOrCreateConversation(ctx, "session:other")
	s.AddMessage(ctx, current.ConversationID, "user", "shared needle from current topic", 5)
	s.AddMessage(ctx, other.ConversationID, "user", "shared needle from other topic", 5)

	tool := NewGrepTool(&RetrievalEngine{store: s})
	toolCtx := tools.WithToolSessionContext(ctx, "agent", "session:current", nil)
	result := tool.Execute(toolCtx, map[string]any{"pattern": "needle"})
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}

	var output struct {
		Messages []GrepMessageResult `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result.ContentForLLM()), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(output.Messages) != 1 {
		t.Fatalf("messages = %d, want 1: %#v", len(output.Messages), output.Messages)
	}
	if output.Messages[0].ConversationID != current.ConversationID {
		t.Fatalf("conversation id = %d, want %d", output.Messages[0].ConversationID, current.ConversationID)
	}
}

func TestGrepToolCanSearchAllConversations(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	current, _ := s.GetOrCreateConversation(ctx, "session:current")
	other, _ := s.GetOrCreateConversation(ctx, "session:other")
	s.AddMessage(ctx, current.ConversationID, "user", "shared needle from current topic", 5)
	s.AddMessage(ctx, other.ConversationID, "user", "shared needle from other topic", 5)

	tool := NewGrepTool(&RetrievalEngine{store: s})
	toolCtx := tools.WithToolSessionContext(ctx, "agent", "session:current", nil)
	result := tool.Execute(toolCtx, map[string]any{"pattern": "needle", "all_conversations": true})
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}

	var output struct {
		Messages []GrepMessageResult `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result.ContentForLLM()), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(output.Messages) != 2 {
		t.Fatalf("messages = %d, want 2: %#v", len(output.Messages), output.Messages)
	}
}

func TestGrepToolUnknownSessionDoesNotSearchAllConversations(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	current, _ := s.GetOrCreateConversation(ctx, "session:current")
	other, _ := s.GetOrCreateConversation(ctx, "session:other")
	s.AddMessage(ctx, current.ConversationID, "user", "shared needle from current topic", 5)
	s.AddMessage(ctx, other.ConversationID, "user", "shared needle from other topic", 5)

	tool := NewGrepTool(&RetrievalEngine{store: s})
	toolCtx := tools.WithToolSessionContext(ctx, "agent", "session:missing", nil)
	result := tool.Execute(toolCtx, map[string]any{"pattern": "needle"})
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}

	var output struct {
		Messages []GrepMessageResult `json:"messages"`
		Hint     string              `json:"hint"`
	}
	if err := json.Unmarshal([]byte(result.ContentForLLM()), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(output.Messages) != 0 {
		t.Fatalf("messages = %d, want 0: %#v", len(output.Messages), output.Messages)
	}
	if output.Hint == "" {
		t.Fatal("expected hint for missing current conversation")
	}
}

func TestGrepToolEmptySessionDoesNotSearchAllConversations(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "session:current")
	s.AddMessage(ctx, conv.ConversationID, "user", "shared needle from current topic", 5)

	tool := NewGrepTool(&RetrievalEngine{store: s})
	result := tool.Execute(ctx, map[string]any{"pattern": "needle"})
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}

	var output struct {
		Messages []GrepMessageResult `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result.ContentForLLM()), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(output.Messages) != 0 {
		t.Fatalf("messages = %d, want 0: %#v", len(output.Messages), output.Messages)
	}
}

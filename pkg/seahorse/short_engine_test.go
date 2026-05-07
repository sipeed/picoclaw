package seahorse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// helper: open a test engine with in-memory DB
func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	store := &Store{db: db}
	return &Engine{
		store:  store,
		config: Config{},
	}
}

// --- compileSessionPattern ---

func TestCompileSessionPattern(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Exact match
		{"agent:abc123", "agent:abc123", true},
		{"agent:abc123", "agent:def456", false},
		// Single * — matches non-colon chars
		{"agent:*", "agent:abc123", true},
		{"agent:*", "agent:abc:def", false}, // * doesn't match colons
		// ** — matches everything including colons
		{"cron:**", "cron:backup", true},
		{"cron:**", "cron:backup:daily", true},
		{"cron:**", "agent:abc", false},
		// Mixed
		{"agent:*:sub:**", "agent:abc:sub:def", true},
		{"agent:*:sub:**", "agent:abc:sub:def:ghi", true},
		{"agent:*:sub:**", "agent:abc:def", false},
		// Empty pattern — matches nothing meaningful
		{"", "", true},
		{"", "agent:abc", false},
	}

	for _, tt := range tests {
		re := compileSessionPattern(tt.pattern)
		if re == nil && tt.pattern != "" {
			t.Fatalf("compileSessionPattern(%q) returned nil", tt.pattern)
		}
		if tt.pattern == "" {
			continue
		}
		got := re.MatchString(tt.input)
		if got != tt.want {
			t.Errorf("compileSessionPattern(%q).Match(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
		}
	}
}

// --- Session Pattern Filtering ---

func TestEngineShouldIgnoreSession(t *testing.T) {
	eng := &Engine{
		ignorePatterns: compileSessionPatterns([]string{"cron:**", "test:*"}),
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"cron:backup", true},
		{"cron:backup:daily", true},
		{"test:session", true},
		{"agent:abc", false},
		{"", false},
	}

	for _, tt := range tests {
		got := eng.shouldIgnoreSession(tt.key)
		if got != tt.want {
			t.Errorf("shouldIgnoreSession(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestEngineIsStatelessSession(t *testing.T) {
	eng := &Engine{
		statelessPatterns: compileSessionPatterns([]string{"agent:*:sub:**"}),
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"agent:abc:sub:def", true},
		{"agent:abc:sub:def:ghi", true},
		{"agent:abc", false},
		{"cron:backup", false},
	}

	for _, tt := range tests {
		got := eng.isStatelessSession(tt.key)
		if got != tt.want {
			t.Errorf("isStatelessSession(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

// --- NewEngine ---

func TestNewEngine(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "short.db")

	eng, err := NewEngine(Config{DBPath: dbPath}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer eng.Close()

	// DB file should exist
	if _, pathErr := os.Stat(dbPath); os.IsNotExist(pathErr) {
		t.Error("expected DB file to be created")
	}

	// Store should be usable
	ctx := context.Background()
	conv, err := eng.store.GetOrCreateConversation(ctx, "test:session")
	if err != nil {
		t.Fatalf("store should work: %v", err)
	}
	if conv.ConversationID == 0 {
		t.Error("expected valid conversation ID")
	}

	// GetRetrieval should return non-nil RetrievalEngine
	retrieval := eng.GetRetrieval()
	if retrieval == nil {
		t.Error("expected GetRetrieval to return non-nil RetrievalEngine")
	}
}

func TestNewEngineWithPatterns(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "short.db")

	eng, err := NewEngine(Config{
		DBPath:                   dbPath,
		IgnoreSessionPatterns:    []string{"cron:**"},
		StatelessSessionPatterns: []string{"agent:*:sub:**"},
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer eng.Close()

	if !eng.shouldIgnoreSession("cron:backup") {
		t.Error("expected cron:backup to be ignored")
	}
	if !eng.isStatelessSession("agent:abc:sub:def") {
		t.Error("expected agent:abc:sub:def to be stateless")
	}
}

// --- Ingest ---

func TestEngineIngest(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{Role: "user", Content: "hello", TokenCount: 2},
		{Role: "assistant", Content: "world", TokenCount: 2},
	}

	result, err := eng.Ingest(ctx, "agent:test", msgs)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", result.MessageCount)
	}
	if result.TokenCount != 4 {
		t.Errorf("TokenCount = %d, want 4", result.TokenCount)
	}

	// Verify messages were stored
	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:test")
	stored, _ := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 2 {
		t.Fatalf("stored messages = %d, want 2", len(stored))
	}
	if stored[0].Content != "hello" {
		t.Errorf("stored[0].Content = %q, want 'hello'", stored[0].Content)
	}

	// Verify context_items were populated
	items, _ := eng.store.GetContextItems(ctx, conv.ConversationID)
	if len(items) != 2 {
		t.Fatalf("context items = %d, want 2", len(items))
	}
	if items[0].ItemType != "message" {
		t.Errorf("item[0].ItemType = %q, want 'message'", items[0].ItemType)
	}
}

func TestEngineIngestIgnoresSession(t *testing.T) {
	eng := newTestEngine(t)
	eng.ignorePatterns = compileSessionPatterns([]string{"cron:**"})
	ctx := context.Background()

	msgs := []Message{{Role: "user", Content: "hello", TokenCount: 2}}
	result, err := eng.Ingest(ctx, "cron:backup", msgs)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for ignored session")
	}

	// Verify no data was stored
	conv, _ := eng.store.GetConversationBySessionKey(ctx, "cron:backup")
	if conv != nil {
		t.Error("expected no conversation for ignored session")
	}
}

func TestEngineIngestStatelessSession(t *testing.T) {
	eng := newTestEngine(t)
	eng.statelessPatterns = compileSessionPatterns([]string{"agent:*:ro"})
	ctx := context.Background()

	msgs := []Message{{Role: "user", Content: "hello", TokenCount: 2}}
	result, err := eng.Ingest(ctx, "agent:abc:ro", msgs)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for stateless session")
	}
}

func TestEngineIngestIncremental(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// First ingest
	eng.Ingest(ctx, "agent:test", []Message{
		{Role: "user", Content: "msg1", TokenCount: 1},
	})
	// Second ingest — should append, not replace
	eng.Ingest(ctx, "agent:test", []Message{
		{Role: "assistant", Content: "msg2", TokenCount: 1},
	})

	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:test")
	stored, _ := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 2 {
		t.Errorf("stored messages = %d, want 2", len(stored))
	}
}

func TestEngineIngestWithParts(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{
			Role:       "assistant",
			Content:    "",
			TokenCount: 10,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "read_file", Arguments: `{"path":"/tmp/test"}`, ToolCallID: "tc_123"},
				{Type: "text", Text: "here is the file content"},
			},
		},
	}

	result, err := eng.Ingest(ctx, "agent:parts-test", msgs)
	if err != nil {
		t.Fatalf("Ingest with parts: %v", err)
	}
	if result.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", result.MessageCount)
	}

	// Verify message was stored WITH parts
	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:parts-test")
	stored, _ := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 1 {
		t.Fatalf("stored messages = %d, want 1", len(stored))
	}
	if len(stored[0].Parts) != 2 {
		t.Fatalf("stored message parts = %d, want 2", len(stored[0].Parts))
	}
	if stored[0].Parts[0].Type != "tool_use" {
		t.Errorf("part[0].Type = %q, want tool_use", stored[0].Parts[0].Type)
	}
	if stored[0].Parts[0].Name != "read_file" {
		t.Errorf("part[0].Name = %q, want read_file", stored[0].Parts[0].Name)
	}
	if stored[0].Parts[0].ToolCallID != "tc_123" {
		t.Errorf("part[0].ToolCallID = %q, want tc_123", stored[0].Parts[0].ToolCallID)
	}
	if stored[0].Parts[1].Type != "text" {
		t.Errorf("part[1].Type = %q, want text", stored[0].Parts[1].Type)
	}
	if stored[0].Parts[1].Text != "here is the file content" {
		t.Errorf("part[1].Text = %q, want 'here is the file content'", stored[0].Parts[1].Text)
	}
}

func TestEngineIngestPreservesReasoningContent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{
			Role:             "assistant",
			Content:          "world",
			ReasoningContent: "let me think this through",
			TokenCount:       4,
		},
	}

	_, err := eng.Ingest(ctx, "agent:reasoning", msgs)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:reasoning")
	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored messages = %d, want 1", len(stored))
	}
	if stored[0].ReasoningContent != "let me think this through" {
		t.Errorf(
			"stored[0].ReasoningContent = %q, want %q",
			stored[0].ReasoningContent,
			"let me think this through",
		)
	}

	result, err := eng.Assemble(ctx, "agent:reasoning", AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("assembled messages = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].ReasoningContent != "let me think this through" {
		t.Errorf(
			"assembled reasoning = %q, want %q",
			result.Messages[0].ReasoningContent,
			"let me think this through",
		)
	}
}

func TestEngineIngestWithPartsPreservesReasoningContent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{
			Role:             "assistant",
			ReasoningContent: "I need to inspect the file first",
			TokenCount:       10,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "read_file", Arguments: `{"path":"/tmp/test"}`, ToolCallID: "tc_123"},
			},
		},
	}

	_, err := eng.Ingest(ctx, "agent:parts-reasoning", msgs)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:parts-reasoning")
	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored messages = %d, want 1", len(stored))
	}
	if stored[0].ReasoningContent != "I need to inspect the file first" {
		t.Errorf(
			"stored reasoning = %q, want %q",
			stored[0].ReasoningContent,
			"I need to inspect the file first",
		)
	}

	result, err := eng.Assemble(ctx, "agent:parts-reasoning", AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("assembled messages = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].ReasoningContent != "I need to inspect the file first" {
		t.Errorf(
			"assembled reasoning = %q, want %q",
			result.Messages[0].ReasoningContent,
			"I need to inspect the file first",
		)
	}
}

func TestEngineIngestAssemblePreservesParts(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// Ingest a message with tool_use parts
	eng.Ingest(ctx, "agent:parts-roundtrip", []Message{
		{Role: "user", Content: "list files", TokenCount: 3},
		{
			Role:       "assistant",
			Content:    "",
			TokenCount: 5,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "bash", Arguments: `{"cmd":"ls"}`, ToolCallID: "tc_1"},
				{Type: "text", Text: "found 3 files"},
			},
		},
	})

	// Assemble should return messages with parts intact
	result, err := eng.Assemble(ctx, "agent:parts-roundtrip", AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Fatalf("Assemble returned %d messages, want 2", len(result.Messages))
	}

	// The second message should have Parts populated
	assistantMsg := result.Messages[1]
	if len(assistantMsg.Parts) != 2 {
		t.Fatalf("Assembled assistant message Parts = %d, want 2", len(assistantMsg.Parts))
	}
	if assistantMsg.Parts[0].Type != "tool_use" {
		t.Errorf("part[0].Type = %q, want tool_use", assistantMsg.Parts[0].Type)
	}
	if assistantMsg.Parts[0].ToolCallID != "tc_1" {
		t.Errorf("part[0].ToolCallID = %q, want tc_1", assistantMsg.Parts[0].ToolCallID)
	}
}

// --- Session Mutex ---

func TestEngineSessionMutex(t *testing.T) {
	eng := newTestEngine(t)

	mu1 := eng.getSessionMutex("agent:test")
	mu2 := eng.getSessionMutex("agent:test")
	mu3 := eng.getSessionMutex("agent:other")

	if mu1 != mu2 {
		t.Error("expected same mutex for same session key")
	}
	if mu1 == mu3 {
		t.Error("expected different mutex for different session key")
	}
}

// --- Close ---

func TestEngineClose(t *testing.T) {
	eng := newTestEngine(t)
	if err := eng.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// --- compileSessionPatterns (batch) ---

func TestCompileSessionPatterns(t *testing.T) {
	patterns := compileSessionPatterns([]string{"cron:**", "agent:*:ro"})
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"cron:backup", true},
		{"agent:abc:ro", true},
		{"agent:abc:def", false},
		{"", false},
	}

	for _, tt := range tests {
		matched := false
		for _, p := range patterns {
			if p.MatchString(tt.input) {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("patterns.Match(%q) = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestCompileSessionPatternsEmpty(t *testing.T) {
	patterns := compileSessionPatterns(nil)
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns for nil input, got %d", len(patterns))
	}
}

// --- Bootstrap ---

func TestEngineBootstrap(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", TokenCount: 3},
		{Role: "user", Content: "how are you", TokenCount: 5},
	}

	err := eng.Bootstrap(ctx, "agent:boot1", msgs)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// Verify conversation was created
	conv, err := eng.store.GetConversationBySessionKey(ctx, "agent:boot1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if conv == nil {
		t.Fatal("expected conversation to exist after bootstrap")
	}

	// Verify messages were stored
	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored messages, got %d", len(stored))
	}
	if stored[0].Content != "hello" {
		t.Errorf("stored[0].Content = %q, want 'hello'", stored[0].Content)
	}

	// Verify context_items were populated
	items, err := eng.store.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 context items, got %d", len(items))
	}
}

func TestEngineBootstrapEmpty(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.Bootstrap(ctx, "agent:empty", nil)
	if err != nil {
		t.Fatalf("Bootstrap empty: %v", err)
	}

	// No conversation should be created for empty messages
	conv, _ := eng.store.GetConversationBySessionKey(ctx, "agent:empty")
	if conv != nil {
		t.Error("expected no conversation for empty bootstrap")
	}
}

func TestEngineBootstrapIdempotent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", TokenCount: 3},
	}

	// Bootstrap twice with same messages
	eng.Bootstrap(ctx, "agent:idem", msgs)
	eng.Bootstrap(ctx, "agent:idem", msgs)

	// Should still have exactly 2 messages (no duplicates)
	conv, _ := eng.store.GetConversationBySessionKey(ctx, "agent:idem")
	if conv == nil {
		t.Fatal("expected conversation")
	}
	stored, _ := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 2 {
		t.Errorf("expected 2 messages (idempotent), got %d", len(stored))
	}
}

func TestBootstrapRepairsMissingReasoningContent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "agent:repair-reasoning"

	conv, err := eng.store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	userMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "user", "hello", 3)
	if err != nil {
		t.Fatalf("AddMessage user: %v", err)
	}

	assistantMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "assistant", "world", 3)
	if err != nil {
		t.Fatalf("AddMessage assistant: %v", err)
	}

	err = eng.store.AppendContextMessages(
		ctx,
		conv.ConversationID,
		[]int64{userMsg.ID, assistantMsg.ID},
	)
	if err != nil {
		t.Fatalf("AppendContextMessages: %v", err)
	}

	err = eng.Bootstrap(ctx, sessionKey, []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", ReasoningContent: "let me think this through", TokenCount: 3},
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored messages = %d, want 2", len(stored))
	}
	if stored[1].ReasoningContent != "let me think this through" {
		t.Errorf(
			"stored[1].ReasoningContent = %q, want %q",
			stored[1].ReasoningContent,
			"let me think this through",
		)
	}
}

func TestBootstrapRepairsMissingReasoningContentWithoutDroppingSummaries(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "agent:repair-reasoning-summary"

	conv, err := eng.store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	userMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "user", "hello", 3)
	if err != nil {
		t.Fatalf("AddMessage user: %v", err)
	}
	assistantMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "assistant", "world", 3)
	if err != nil {
		t.Fatalf("AddMessage assistant: %v", err)
	}

	err = eng.store.AppendContextMessages(
		ctx,
		conv.ConversationID,
		[]int64{userMsg.ID, assistantMsg.ID},
	)
	if err != nil {
		t.Fatalf("AppendContextMessages: %v", err)
	}

	summary, err := eng.store.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "summary before repair",
		TokenCount:     10,
	})
	if err != nil {
		t.Fatalf("CreateSummary: %v", err)
	}

	err = eng.store.AppendContextSummary(ctx, conv.ConversationID, summary.SummaryID)
	if err != nil {
		t.Fatalf("AppendContextSummary: %v", err)
	}

	err = eng.Bootstrap(ctx, sessionKey, []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", ReasoningContent: "let me think this through", TokenCount: 3},
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored messages = %d, want 2", len(stored))
	}
	if stored[1].ReasoningContent != "let me think this through" {
		t.Errorf(
			"stored[1].ReasoningContent = %q, want %q",
			stored[1].ReasoningContent,
			"let me think this through",
		)
	}

	summaries, err := eng.store.GetSummariesByConversation(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetSummariesByConversation: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %d, want 1", len(summaries))
	}
	if summaries[0].SummaryID != summary.SummaryID {
		t.Errorf("SummaryID = %q, want %q", summaries[0].SummaryID, summary.SummaryID)
	}

	items, err := eng.store.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("context items = %d, want 3", len(items))
	}
	if items[2].ItemType != "summary" || items[2].SummaryID != summary.SummaryID {
		t.Errorf("summary context item = %+v, want summary %q", items[2], summary.SummaryID)
	}
}

func TestBootstrapRepairsMissingReasoningContentOnPrefixBeforeAppendingDelta(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "agent:repair-reasoning-prefix"

	conv, err := eng.store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	userMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "user", "hello", 3)
	if err != nil {
		t.Fatalf("AddMessage user: %v", err)
	}
	assistantMsg, err := eng.store.AddMessage(ctx, conv.ConversationID, "assistant", "world", 3)
	if err != nil {
		t.Fatalf("AddMessage assistant: %v", err)
	}

	err = eng.store.AppendContextMessages(
		ctx,
		conv.ConversationID,
		[]int64{userMsg.ID, assistantMsg.ID},
	)
	if err != nil {
		t.Fatalf("AppendContextMessages: %v", err)
	}

	err = eng.Bootstrap(ctx, sessionKey, []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", ReasoningContent: "let me think this through", TokenCount: 3},
		{Role: "user", Content: "follow-up", TokenCount: 2},
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	stored, err := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("stored messages = %d, want 3", len(stored))
	}
	if stored[1].ReasoningContent != "let me think this through" {
		t.Errorf(
			"stored[1].ReasoningContent = %q, want %q",
			stored[1].ReasoningContent,
			"let me think this through",
		)
	}
	if stored[2].Content != "follow-up" {
		t.Errorf("stored[2].Content = %q, want %q", stored[2].Content, "follow-up")
	}

	items, err := eng.store.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("context items = %d, want 3", len(items))
	}
	if items[2].ItemType != "message" || items[2].MessageID != stored[2].ID {
		t.Errorf("last context item = %+v, want appended message %d", items[2], stored[2].ID)
	}
}

func TestEngineBootstrapDelta(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// First bootstrap with 2 messages
	msgs1 := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", TokenCount: 3},
	}
	eng.Bootstrap(ctx, "agent:delta", msgs1)

	// Second bootstrap with 4 messages (2 existing + 2 new)
	msgs2 := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "world", TokenCount: 3},
		{Role: "user", Content: "new question", TokenCount: 5},
		{Role: "assistant", Content: "new answer", TokenCount: 5},
	}
	eng.Bootstrap(ctx, "agent:delta", msgs2)

	conv, _ := eng.store.GetConversationBySessionKey(ctx, "agent:delta")
	if conv == nil {
		t.Fatal("expected conversation")
	}
	stored, _ := eng.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 4 {
		t.Errorf("expected 4 messages (delta), got %d", len(stored))
	}
}

func TestBootstrapPopulatesContextItems(t *testing.T) {
	// Bootstrap ingests messages and populates context_items
	e := newTestEngine(t)
	ctx := context.Background()

	messages := []Message{
		{Role: "user", Content: "hello from bootstrap test", TokenCount: 10},
		{Role: "assistant", Content: "hi there", TokenCount: 5},
		{Role: "user", Content: "how are you", TokenCount: 5},
		{Role: "assistant", Content: "doing well", TokenCount: 5},
		{Role: "user", Content: "great news", TokenCount: 5},
		{Role: "assistant", Content: "awesome", TokenCount: 5},
		{Role: "user", Content: "lets code", TokenCount: 5},
		{Role: "assistant", Content: "sure thing", TokenCount: 5},
	}

	// Bootstrap should ingest and rebuild context_items
	err := e.Bootstrap(ctx, "test-bootstrap-rebuild", messages)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// After bootstrap, context_items should be populated
	conv, _ := e.store.GetOrCreateConversation(ctx, "test-bootstrap-rebuild")
	items, err := e.store.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextItems: %v", err)
	}

	if len(items) == 0 {
		t.Error("expected context_items to be populated after Bootstrap, got 0 items")
	}

	// Should have one item per message
	if len(items) != len(messages) {
		t.Errorf("expected %d context items, got %d", len(messages), len(items))
	}
}

func TestBootstrapDeltaPreservesOrder(t *testing.T) {
	// When Bootstrap does delta ingest, context_items should maintain
	// correct order with new messages appended after anchor.
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-delta-order"

	// First: bootstrap with 4 messages
	initialMsgs := []Message{
		{Role: "user", Content: "msg1", TokenCount: 5},
		{Role: "assistant", Content: "msg2", TokenCount: 5},
		{Role: "user", Content: "msg3", TokenCount: 5},
		{Role: "assistant", Content: "msg4", TokenCount: 5},
	}
	err := e.Bootstrap(ctx, sessionKey, initialMsgs)
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	items1, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	if len(items1) != 4 {
		t.Fatalf("after first bootstrap: expected 4 items, got %d", len(items1))
	}

	// Now bootstrap again with 6 messages (4 existing + 2 new)
	// The delta (msg5, msg6) should be appended
	updatedMsgs := []Message{
		{Role: "user", Content: "msg1", TokenCount: 5},
		{Role: "assistant", Content: "msg2", TokenCount: 5},
		{Role: "user", Content: "msg3", TokenCount: 5},
		{Role: "assistant", Content: "msg4", TokenCount: 5},
		{Role: "user", Content: "msg5", TokenCount: 5},
		{Role: "assistant", Content: "msg6", TokenCount: 5},
	}
	err = e.Bootstrap(ctx, sessionKey, updatedMsgs)
	if err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}

	items2, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	if len(items2) != 6 {
		t.Errorf("after delta bootstrap: expected 6 items, got %d", len(items2))
	}
}

func TestBootstrapHistoryEditFirstMessageChanged(t *testing.T) {
	// When the first message changes (anchor = -1), Bootstrap should rebuild
	// from scratch without panicking (regression test for index out of range [-1])
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-history-edit"

	// First: bootstrap with some messages
	initialMsgs := []Message{
		{Role: "user", Content: "original first", TokenCount: 5},
		{Role: "assistant", Content: "response", TokenCount: 5},
		{Role: "user", Content: "question", TokenCount: 5},
	}
	err := e.Bootstrap(ctx, sessionKey, initialMsgs)
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	// Now bootstrap with completely different messages (first message changed)
	// This should NOT panic - it should rebuild from scratch
	editedMsgs := []Message{
		{Role: "user", Content: "DIFFERENT first message", TokenCount: 5},
		{Role: "assistant", Content: "DIFFERENT response", TokenCount: 5},
		{Role: "user", Content: "DIFFERENT question", TokenCount: 5},
	}
	err = e.Bootstrap(ctx, sessionKey, editedMsgs)
	if err != nil {
		t.Fatalf("second Bootstrap (history edit): %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)

	// Should have the NEW messages (history was rebuilt)
	if len(stored) != 3 {
		t.Errorf("expected 3 messages after history edit, got %d", len(stored))
	}
	if len(stored) > 0 && stored[0].Content != "DIFFERENT first message" {
		t.Errorf("first message = %q, want 'DIFFERENT first message'", stored[0].Content)
	}
}

func TestBootstrapSameContentDifferentTokenCountNoRebuild(t *testing.T) {
	// Bootstrap should NOT rebuild when content is identical but TokenCount differs.
	// This happens when TokenCount is re-estimated (e.g., via tokenizer.EstimateMessageTokens)
	// during bootstrap, which may give slightly different values.
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-token-diff"

	// First: bootstrap with some messages
	initialMsgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 10},
		{Role: "assistant", Content: "hi there", TokenCount: 5},
	}
	err := e.Bootstrap(ctx, sessionKey, initialMsgs)
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	storedBefore, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)

	// Second: bootstrap with SAME content but DIFFERENT TokenCount
	// This should be a no-op (not rebuild)
	sameContentMsgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 999},   // Different token count!
		{Role: "assistant", Content: "hi there", TokenCount: 888}, // Different token count!
	}
	err = e.Bootstrap(ctx, sessionKey, sameContentMsgs)
	if err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}

	storedAfter, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)

	// Should have same number of messages (no rebuild)
	if len(storedAfter) != len(storedBefore) {
		t.Errorf("expected %d messages (no rebuild), got %d", len(storedBefore), len(storedAfter))
	}

	// Message IDs should be the same (no delete+re-ingest)
	for i := range storedBefore {
		if storedBefore[i].ID != storedAfter[i].ID {
			t.Errorf("message %d ID changed: before=%d, after=%d (should be no-op)",
				i, storedBefore[i].ID, storedAfter[i].ID)
		}
	}
}

// --- Session Mutex ---

func TestEngineSessionMutexSharded(t *testing.T) {
	eng := newTestEngine(t)

	// Same session key should always return the same mutex (deterministic hash)
	mu1 := eng.getSessionMutex("agent:test")
	mu2 := eng.getSessionMutex("agent:test")
	if mu1 != mu2 {
		t.Error("expected same mutex for same session key")
	}

	// Different session keys may share the same shard (hash collision)
	// This is expected behavior - we just need bounded memory, not unique locks
	mu3 := eng.getSessionMutex("agent:other")

	// Both mutexes should be valid and usable
	mu1.Lock()
	mu1.Unlock()
	mu3.Lock()
	mu3.Unlock()
}

func TestEngineSessionMutexBoundedMemory(t *testing.T) {
	// Verify that session mutexes use bounded memory (256 shards)
	eng := newTestEngine(t)

	// Get mutexes for many different sessions
	seen := make(map[*sync.Mutex]bool)
	for i := 0; i < 1000; i++ {
		sessionKey := fmt.Sprintf("agent:session-%d", i)
		mu := eng.getSessionMutex(sessionKey)
		seen[mu] = true
	}

	// With 256 shards and 1000 sessions, we should see at most 256 unique mutexes
	// (likely fewer due to hash collisions)
	if len(seen) > 256 {
		t.Errorf("expected at most 256 unique mutexes (shards), got %d", len(seen))
	}
}

func TestEngineSessionMutexConsistentHash(t *testing.T) {
	// Same session key should always hash to the same shard
	eng := newTestEngine(t)

	sessionKey := "agent:consistent-hash-test"
	mu1 := eng.getSessionMutex(sessionKey)
	mu2 := eng.getSessionMutex(sessionKey)
	mu3 := eng.getSessionMutex(sessionKey)

	if mu1 != mu2 || mu2 != mu3 {
		t.Error("hash function should be deterministic - same key must map to same shard")
	}
}

// --- Summary Role ---

func TestAssemblerSummaryRoleNotUser(t *testing.T) {
	// Summaries should use "system" role, not "user"
	eng := newTestEngine(t)
	ctx := context.Background()

	// Ingest messages
	eng.Ingest(ctx, "agent:summary-role-test", []Message{
		{Role: "user", Content: "hello", TokenCount: 5},
		{Role: "assistant", Content: "world", TokenCount: 5},
	})

	conv, _ := eng.store.GetOrCreateConversation(ctx, "agent:summary-role-test")

	// Create a summary and add it to context
	sum, err := eng.store.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Content:        "Test summary content",
		TokenCount:     10,
		Kind:           SummaryKindCondensed,
		Depth:          1,
	})
	if err != nil {
		t.Fatalf("CreateSummary: %v", err)
	}
	eng.store.AppendContextSummary(ctx, conv.ConversationID, sum.SummaryID)

	// Assemble and check summary message role
	result, err := eng.Assemble(ctx, "agent:summary-role-test", AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Find the summary message (should have XML content with <summary>)
	for _, msg := range result.Messages {
		if strings.Contains(msg.Content, "<summary") {
			if msg.Role == "user" {
				t.Error("summary message should NOT use 'user' role - use 'system' or dedicated role instead")
			}
			// Expected: role should be "system" or similar
			return
		}
	}
}

// --- Race Test ---

// newTestEngineForConcurrency creates a file-based test engine (required for concurrent SQLite access)
func newTestEngineForConcurrency(t *testing.T) *Engine {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "race_test.db")
	eng, err := NewEngine(Config{DBPath: dbPath}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return eng
}

func TestEngineConcurrentIngestAndAssemble(t *testing.T) {
	// Concurrent Ingest + Assemble on same session should not panic or corrupt data
	eng := newTestEngineForConcurrency(t)
	defer eng.Close()
	ctx := context.Background()
	sessionKey := "agent:race-test"

	// Start with some initial data
	eng.Ingest(ctx, sessionKey, []Message{
		{Role: "user", Content: "initial", TokenCount: 2},
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	// Concurrent Ingest
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := eng.Ingest(ctx, sessionKey, []Message{
				{Role: "user", Content: fmt.Sprintf("ingest-%d", idx), TokenCount: 3},
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	// Concurrent Assemble
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := eng.Assemble(ctx, sessionKey, AssembleInput{Budget: 500})
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent operation error: %v", err)
	}

	// Verify data is still consistent
	conv, _ := eng.store.GetOrCreateConversation(ctx, sessionKey)
	msgs, _ := eng.store.GetMessages(ctx, conv.ConversationID, 100, 0)
	if len(msgs) < 6 { // 1 initial + 5 ingest
		t.Errorf("expected at least 6 messages, got %d", len(msgs))
	}
}

func TestEngineConcurrentCompactAndAssemble(t *testing.T) {
	// Concurrent Compact + Assemble should not panic
	eng := newTestEngineForConcurrency(t)
	defer eng.Close()
	ctx := context.Background()
	sessionKey := "agent:compact-race"

	// Ingest enough messages for compaction
	for i := 0; i < 10; i++ {
		eng.Ingest(ctx, sessionKey, []Message{
			{Role: "user", Content: fmt.Sprintf("msg-%d", i), TokenCount: 50},
			{Role: "assistant", Content: fmt.Sprintf("reply-%d", i), TokenCount: 50},
		})
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	// Concurrent Compact (will use truncation fallback since no LLM)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := eng.Compact(ctx, sessionKey, CompactInput{})
			if err != nil {
				errCh <- err
			}
		}()
	}

	// Concurrent Assemble
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := eng.Assemble(ctx, sessionKey, AssembleInput{Budget: 500})
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent compact/assemble error: %v", err)
	}
}

// --- Bootstrap Edge Cases ---

func TestBootstrapDuplicateContent(t *testing.T) {
	// Bootstrap should correctly handle messages with identical content
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-duplicate"

	// Messages with identical content
	msgs := []Message{
		{Role: "user", Content: "same content", TokenCount: 5},
		{Role: "user", Content: "same content", TokenCount: 5},
		{Role: "user", Content: "same content", TokenCount: 5},
	}
	err := e.Bootstrap(ctx, sessionKey, msgs)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 3 {
		t.Errorf("expected 3 messages with duplicate content, got %d", len(stored))
	}
}

func TestBootstrapOutOfOrderAppend(t *testing.T) {
	// When bootstrap receives messages out of expected order,
	// it should still correctly match prefix
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-oob"

	// First: normal bootstrap
	msgs1 := []Message{
		{Role: "user", Content: "msg1", TokenCount: 3},
		{Role: "assistant", Content: "msg2", TokenCount: 3},
	}
	e.Bootstrap(ctx, sessionKey, msgs1)

	// Second: bootstrap with same prefix (out of order append at end is fine)
	// The key is that the prefix matching works correctly
	msgs2 := []Message{
		{Role: "user", Content: "msg1", TokenCount: 3},
		{Role: "assistant", Content: "msg2", TokenCount: 3},
		{Role: "user", Content: "msg3", TokenCount: 3},
		{Role: "assistant", Content: "msg4", TokenCount: 3},
	}
	e.Bootstrap(ctx, sessionKey, msgs2)

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) != 4 {
		t.Errorf("expected 4 messages after append, got %d", len(stored))
	}

	// Verify order is preserved
	if stored[0].Content != "msg1" || stored[1].Content != "msg2" ||
		stored[2].Content != "msg3" || stored[3].Content != "msg4" {
		t.Errorf("messages out of order: %v", stored)
	}
}

func TestBootstrapWithToolParts(t *testing.T) {
	// Bootstrap should correctly store messages with tool parts
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-toolparts"

	msgs := []Message{
		{
			Role:       "user",
			Content:    "list files",
			TokenCount: 5,
		},
		{
			Role:       "assistant",
			Content:    "",
			TokenCount: 10,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "bash", Arguments: `{"cmd":"ls"}`, ToolCallID: "tc_1"},
			},
		},
		{
			Role:       "tool",
			Content:    "file1.txt\nfile2.txt",
			TokenCount: 8,
			Parts: []MessagePart{
				{Type: "tool_result", ToolCallID: "tc_1", Text: "file1.txt\nfile2.txt"},
			},
		},
		{
			Role:       "assistant",
			Content:    "I see two files",
			TokenCount: 8,
		},
	}

	err := e.Bootstrap(ctx, sessionKey, msgs)
	if err != nil {
		t.Fatalf("Bootstrap with tool parts: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)

	if len(stored) != 4 {
		t.Errorf("expected 4 messages, got %d", len(stored))
	}

	// Verify tool_use part is preserved
	foundToolUse := false
	for _, msg := range stored {
		for _, part := range msg.Parts {
			if part.Type == "tool_use" && part.Name == "bash" {
				foundToolUse = true
				break
			}
		}
	}
	if !foundToolUse {
		t.Error("expected to find tool_use part in stored messages")
	}

	// Verify tool_result part is preserved
	foundToolResult := false
	for _, msg := range stored {
		for _, part := range msg.Parts {
			if part.Type == "tool_result" && part.ToolCallID == "tc_1" {
				foundToolResult = true
				break
			}
		}
	}
	if !foundToolResult {
		t.Error("expected to find tool_result part in stored messages")
	}

	// Verify tool_result content matches
	for _, msg := range stored {
		if msg.Role == "tool" {
			for _, part := range msg.Parts {
				if part.Type == "tool_result" && part.ToolCallID == "tc_1" {
					if part.Text != "file1.txt\nfile2.txt" {
						t.Errorf("tool result text mismatch: got %q", part.Text)
					}
				}
			}
		}
	}
}

func TestBootstrapToolPartsDelta(t *testing.T) {
	// Delta bootstrap with tool parts should append correctly
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-toolparts-delta"

	// First bootstrap: user + assistant (no tools)
	msgs1 := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "hi", TokenCount: 3},
	}
	e.Bootstrap(ctx, sessionKey, msgs1)

	// Second bootstrap: add message with tool parts
	msgs2 := []Message{
		{Role: "user", Content: "hello", TokenCount: 3},
		{Role: "assistant", Content: "hi", TokenCount: 3},
		{
			Role:       "user",
			Content:    "run command",
			TokenCount: 5,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "bash", Arguments: `{"cmd":"pwd"}`, ToolCallID: "tc_2"},
			},
		},
	}
	e.Bootstrap(ctx, sessionKey, msgs2)

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)

	if len(stored) != 3 {
		t.Errorf("expected 3 messages after delta, got %d", len(stored))
	}

	// Verify the third message has tool parts
	foundToolUse := false
	for _, msg := range stored {
		for _, part := range msg.Parts {
			if part.Type == "tool_use" && part.ToolCallID == "tc_2" {
				foundToolUse = true
				break
			}
		}
	}
	if !foundToolUse {
		t.Error("expected to find tool_use part in delta message")
	}
}

func TestBootstrapToolPartsIdempotent(t *testing.T) {
	// Bootstrap with tool parts should be idempotent - second bootstrap should NOT rebuild
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-toolparts-idem"

	msgs := []Message{
		{
			Role:       "user",
			Content:    "list files",
			TokenCount: 5,
		},
		{
			Role:       "assistant",
			Content:    "",
			TokenCount: 10,
			Parts: []MessagePart{
				{Type: "tool_use", Name: "bash", Arguments: `{"command":"ls"}`, ToolCallID: "tc_1"},
			},
		},
		{
			Role:       "user",
			Content:    "",
			TokenCount: 15,
			Parts: []MessagePart{
				{Type: "tool_result", ToolCallID: "tc_1", Text: "file1.txt\nfile2.txt"},
			},
		},
	}

	// First bootstrap
	e.Bootstrap(ctx, sessionKey, msgs)

	// Get message count after first bootstrap
	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	stored1, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored1) != 3 {
		t.Fatalf("after first bootstrap: expected 3 messages, got %d", len(stored1))
	}

	// Second bootstrap with same messages - should be idempotent (no rebuild)
	e.Bootstrap(ctx, sessionKey, msgs)

	stored2, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored2) != 3 {
		t.Errorf("after second bootstrap: expected 3 messages (idempotent), got %d", len(stored2))
	}

	// Verify messages are identical (not rebuilt)
	for i := range stored1 {
		if stored1[i].ID != stored2[i].ID {
			t.Errorf("message %d was rebuilt (ID changed from %d to %d)", i, stored1[i].ID, stored2[i].ID)
		}
	}
}

func TestBootstrapAnchorWithDuplicateContent(t *testing.T) {
	// Bootstrap should correctly find anchor using longest prefix matching.
	// Uses (role, content, token_count) multi-dimensional comparison.
	//
	// SCENARIO 1: Normal append (no duplicates, no edits)
	// - DB: [A, B, C]
	// - Messages: [A, B, C, D]
	// - Expected: anchor=2, delta=[D]
	//
	// SCENARIO 2: With duplicate content
	// - DB: [A, ok, B, ok, C]
	// - Messages: [A, ok, B, ok, C, D]
	// - Expected: anchor=4, delta=[D]
	//
	// SCENARIO 3: History edit detected
	// - DB: [A, ok, B, ok, C]
	// - Messages: [A, ok, X, ok, C, D]  (B changed to X)
	// - Expected: Detect mismatch at i=2, clear old data, re-ingest from anchor+1

	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-prefix-match"

	// First: bootstrap with initial messages
	initialMsgs := []Message{
		{Role: "user", Content: "A", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "B", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "C", TokenCount: 2},
	}
	err := e.Bootstrap(ctx, sessionKey, initialMsgs)
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	items1, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	if len(items1) != 5 {
		t.Fatalf("after first bootstrap: expected 5 items, got %d", len(items1))
	}

	// SCENARIO 3: History edit detected
	// After detecting mismatch, Bootstrap should:
	// 1. Clear old context_items
	// 2. Delete old messages after anchor
	// 3. Re-ingest delta
	// BUG: Old implementation only cleared context_items but left duplicate messages
	editedMsgs := []Message{
		{Role: "user", Content: "A", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "X", TokenCount: 2}, // Changed from "B" to "X"
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "C", TokenCount: 2},
		{Role: "assistant", Content: "D", TokenCount: 2}, // New
	}
	err = e.Bootstrap(ctx, sessionKey, editedMsgs)
	if err != nil {
		t.Fatalf("second Bootstrap (edit): %v", err)
	}

	// Verify: should have exactly 6 messages in DB, not 11 (5 old + 6 new - duplicates)
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 20, 0)
	if len(stored) != 6 {
		t.Errorf("BUG: expected 6 messages after history edit, got %d (possible duplicates)", len(stored))
	}

	// Verify context_items also has 6 items
	items2, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	if len(items2) != 6 {
		t.Errorf("expected 6 context items, got %d", len(items2))
	}
}

func TestBootstrapAnchorWithDuplicateContent_Simple(t *testing.T) {
	// Simpler test for the duplicate message bug fix
	e := newTestEngine(t)
	ctx := context.Background()
	sessionKey := "test-bootstrap-prefix-match"

	// First: bootstrap with initial messages
	initialMsgs := []Message{
		{Role: "user", Content: "A", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "B", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "C", TokenCount: 2},
	}
	err := e.Bootstrap(ctx, sessionKey, initialMsgs)
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	conv, _ := e.store.GetOrCreateConversation(ctx, sessionKey)
	items1, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	if len(items1) != 5 {
		t.Fatalf("after first bootstrap: expected 5 items, got %d", len(items1))
	}

	// SCENARIO 2: Normal append with duplicate content
	// The algorithm should find anchor at position 4 (last matching position)
	// using longest prefix matching, not single-point matching
	updatedMsgs := []Message{
		{Role: "user", Content: "A", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "B", TokenCount: 2},
		{Role: "assistant", Content: "ok", TokenCount: 1},
		{Role: "user", Content: "C", TokenCount: 2},
		{Role: "assistant", Content: "D", TokenCount: 2}, // New
	}

	err = e.Bootstrap(ctx, sessionKey, updatedMsgs)
	if err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}

	items2, _ := e.store.GetContextItems(ctx, conv.ConversationID)
	// Should have 6 context items (5 existing + 1 new)
	if len(items2) != 6 {
		t.Errorf("after normal append: expected 6 items, got %d", len(items2))
	}

	// Verify the last message is D
	stored, _ := e.store.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(stored) < 1 {
		t.Fatal("expected at least 1 stored message")
	}
	lastMsg := stored[len(stored)-1]
	if lastMsg.Content != "D" {
		t.Errorf("last message content = %q, want 'D'", lastMsg.Content)
	}
}

// --- Assembler lazy init race detection ---

func TestAssemblerLazyInitRace(t *testing.T) {
	// This test verifies that Assemble() lazy initialization of e.assembler
	// is thread-safe. The original code has a data race:
	//   if e.assembler == nil {
	//       e.assembler = &Assembler{...}
	//   }

	// Run multiple iterations to increase chance of catching race
	for i := 0; i < 30; i++ {
		// Create fresh engine with nil assembler
		e := newTestEngine(t)

		ctx := context.Background()
		sessionKey := fmt.Sprintf("race-test-%d", i)

		// Add message first (avoid SQLite concurrency issues)
		_, err := e.Ingest(ctx, sessionKey, []Message{
			{Role: "user", Content: "hello", TokenCount: 5},
		})
		if err != nil {
			t.Fatalf("Ingest: %v", err)
		}

		// Use a barrier to ensure all goroutines start at the same time
		start := make(chan struct{})
		var wg sync.WaitGroup

		for j := 0; j < 20; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start // Wait for all goroutines to be ready
				e.Assemble(ctx, sessionKey, AssembleInput{Budget: 1000})
			}()
		}

		// Start all goroutines simultaneously
		close(start)
		wg.Wait()
	}
}

// --- selectShallowestCondensationCandidate with non-consecutive depths ---

func TestSelectShallowestCondensationWithNonConsecutiveDepths(t *testing.T) {
	e := newTestEngineForConcurrency(t)
	defer e.Close()
	ctx := context.Background()
	sessionKey := "test-non-consecutive-depths"

	// Create conversation
	conv, err := e.store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	// Create summaries with non-consecutive depths: 0 and 1 have < 5, 2 is missing, 3 has >= 5
	// This tests the bug: when depth=2 is missing, the loop breaks and depth=3 is never checked
	// Need > FreshTailCount(32) summaries so they are not all in fresh tail
	// Depth 0: 3 summaries (not enough), Depth 1: 3 summaries (not enough)
	// Depth 2: 0 summaries (missing), Depth 3: 40 summaries (enough)
	depths := []int{0, 0, 0, 1, 1, 1}
	for i := 0; i < 40; i++ {
		depths = append(depths, 3)
	}
	now := time.Now().UTC()

	for i, depth := range depths {
		sum, createErr := e.store.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: conv.ConversationID,
			Kind:           SummaryKindLeaf,
			Depth:          depth,
			Content:        fmt.Sprintf("summary depth %d #%d", depth, i),
			TokenCount:     10,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		if createErr != nil {
			t.Fatalf("CreateSummary: %v", createErr)
		}
		// Add to context items (not in fresh tail)
		if appendErr := e.store.AppendContextSummary(ctx, conv.ConversationID, sum.SummaryID); appendErr != nil {
			t.Fatalf("AppendContextSummary: %v", appendErr)
		}
	}

	// Initialize compaction engine (lazy init)
	e.initCompactionOnce()

	// Call selectShallowestCondensationCandidate
	candidates, err := e.compaction.selectShallowestCondensationCandidate(ctx, conv.ConversationID, false)
	if err != nil {
		t.Fatalf("selectShallowestCondensationCandidate: %v", err)
	}

	// Should find depth=0 (shallowest) with 5 summaries
	if candidates == nil {
		t.Fatal("expected candidates, got nil")
	}
	if len(candidates) < CondensedMinFanout {
		t.Errorf("expected at least %d candidates, got %d", CondensedMinFanout, len(candidates))
	}

	// Verify all returned summaries have the same depth
	if len(candidates) > 0 {
		expectedDepth := candidates[0].Depth
		for _, c := range candidates[1:] {
			if c.Depth != expectedDepth {
				t.Errorf("candidates have mixed depths: %d vs %d", expectedDepth, c.Depth)
			}
		}
	}
}

package seahorse

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	return &Store{db: db}
}

// --- Conversation Operations ---

func TestStoreGetOrCreateConversation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.GetOrCreateConversation(ctx, "agent:abc123")
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}
	if conv.ConversationID == 0 {
		t.Error("expected non-zero conversation ID")
	}
	if conv.SessionKey != "agent:abc123" {
		t.Errorf("session key = %q, want %q", conv.SessionKey, "agent:abc123")
	}

	// Idempotent — same session key returns same conversation
	conv2, err := s.GetOrCreateConversation(ctx, "agent:abc123")
	if err != nil {
		t.Fatalf("GetOrCreateConversation (2nd): %v", err)
	}
	if conv2.ConversationID != conv.ConversationID {
		t.Errorf("idempotent: got ID %d, want %d", conv2.ConversationID, conv.ConversationID)
	}

	// Different session key → new conversation
	conv3, err := s.GetOrCreateConversation(ctx, "agent:def456")
	if err != nil {
		t.Fatalf("GetOrCreateConversation (3rd): %v", err)
	}
	if conv3.ConversationID == conv.ConversationID {
		t.Error("different session key should create different conversation")
	}
}

func TestStoreGetConversationBySessionKey(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	// Not found
	conv, err := s.GetConversationBySessionKey(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv != nil {
		t.Error("expected nil for nonexistent session key")
	}

	// Create then retrieve
	created, err := s.GetOrCreateConversation(ctx, "agent:test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	found, err := s.GetConversationBySessionKey(ctx, "agent:test")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.ConversationID != created.ConversationID {
		t.Errorf("found ID %d, want %d", found.ConversationID, created.ConversationID)
	}
}

// --- Conversation Clear ---

func TestStoreClearConversation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.GetOrCreateConversation(ctx, "agent:clear-test")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	// Add messages
	msg1, err := s.AddMessage(ctx, conv.ConversationID, "user", "hello", 5)
	if err != nil {
		t.Fatalf("add message 1: %v", err)
	}
	msg2, err := s.AddMessage(ctx, conv.ConversationID, "assistant", "hi", 5)
	if err != nil {
		t.Fatalf("add message 2: %v", err)
	}

	// Add a summary
	_, err = s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Content:        "test summary",
		TokenCount:     10,
		Kind:           SummaryKindLeaf,
	})
	if err != nil {
		t.Fatalf("create summary: %v", err)
	}

	// Verify data exists
	msgs, err := s.GetMessages(ctx, conv.ConversationID, 0, 0)
	if err != nil {
		t.Fatalf("get messages before clear: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages before clear, got %d", len(msgs))
	}

	sums, err := s.GetSummariesByConversation(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("get summaries before clear: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("expected 1 summary before clear, got %d", len(sums))
	}

	// Clear
	if err = s.ClearConversation(ctx, conv.ConversationID); err != nil {
		t.Fatalf("clear conversation: %v", err)
	}

	// Verify all data is gone
	msgs, err = s.GetMessages(ctx, conv.ConversationID, 0, 0)
	if err != nil {
		t.Fatalf("get messages after clear: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", len(msgs))
	}

	sums, err = s.GetSummariesByConversation(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("get summaries after clear: %v", err)
	}
	if len(sums) != 0 {
		t.Fatalf("expected 0 summaries after clear, got %d", len(sums))
	}

	items, err := s.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("get context items after clear: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 context items after clear, got %d", len(items))
	}

	var count int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM message_parts WHERE message_id = ? OR message_id = ?",
		msg1.ID, msg2.ID).Scan(&count); err != nil {
		t.Fatalf("count message parts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 message parts after clear, got %d", count)
	}
}

func TestStoreAddAndGetMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	msg, err := s.AddMessage(ctx, conv.ConversationID, "user", "hello world", 5)
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if msg.ID == 0 {
		t.Error("expected non-zero message ID")
	}
	if msg.Role != "user" || msg.Content != "hello world" {
		t.Errorf("message = %+v, want role=user content=hello world", msg)
	}

	// Retrieve
	msgs, err := s.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Content != "hello world" {
		t.Errorf("content = %q, want %q", msgs[0].Content, "hello world")
	}
}

func TestStoreAddAndGetMessagesWithReasoningContent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:reasoning")

	msg, err := s.AddMessageWithReasoning(
		ctx,
		conv.ConversationID,
		"assistant",
		"hello world",
		"let me think",
		5,
	)
	if err != nil {
		t.Fatalf("AddMessageWithReasoning: %v", err)
	}
	if msg.ReasoningContent != "let me think" {
		t.Fatalf("ReasoningContent = %q, want %q", msg.ReasoningContent, "let me think")
	}

	msgs, err := s.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].ReasoningContent != "let me think" {
		t.Errorf("ReasoningContent = %q, want %q", msgs[0].ReasoningContent, "let me think")
	}

	found, err := s.GetMessageByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessageByID: %v", err)
	}
	if found.ReasoningContent != "let me think" {
		t.Errorf("GetMessageByID ReasoningContent = %q, want %q", found.ReasoningContent, "let me think")
	}
}

func TestStoreAddMessageWithParts(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	parts := []MessagePart{
		{Type: "tool_use", Name: "read_file", Arguments: `{"path":"/tmp/test"}`, ToolCallID: "tc_123"},
		{Type: "text", Text: "some output"},
	}
	msg, err := s.AddMessageWithParts(ctx, conv.ConversationID, "assistant", parts, 10)
	if err != nil {
		t.Fatalf("AddMessageWithParts: %v", err)
	}
	if msg.ID == 0 {
		t.Error("expected non-zero message ID")
	}

	// Retrieve and verify parts
	msgs, _ := s.GetMessages(ctx, conv.ConversationID, 10, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(msgs[0].Parts))
	}
	if msgs[0].Parts[0].Type != "tool_use" {
		t.Errorf("part[0].Type = %q, want tool_use", msgs[0].Parts[0].Type)
	}
	if msgs[0].Parts[0].ToolCallID != "tc_123" {
		t.Errorf("part[0].ToolCallID = %q, want tc_123", msgs[0].Parts[0].ToolCallID)
	}
}

func TestStoreAddMessageWithPartsAndReasoningContent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:parts-reasoning")

	parts := []MessagePart{
		{Type: "tool_use", Name: "read_file", Arguments: `{"path":"/tmp/test"}`, ToolCallID: "tc_123"},
	}
	_, err := s.AddMessageWithPartsAndReasoning(
		ctx,
		conv.ConversationID,
		"assistant",
		parts,
		"need to inspect the file first",
		10,
	)
	if err != nil {
		t.Fatalf("AddMessageWithPartsAndReasoning: %v", err)
	}

	msgs, err := s.GetMessages(ctx, conv.ConversationID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].ReasoningContent != "need to inspect the file first" {
		t.Errorf(
			"ReasoningContent = %q, want %q",
			msgs[0].ReasoningContent,
			"need to inspect the file first",
		)
	}
}

func TestStoreGetMessageCount(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	s.AddMessage(ctx, conv.ConversationID, "user", "msg1", 2)
	s.AddMessage(ctx, conv.ConversationID, "assistant", "msg2", 3)
	s.AddMessage(ctx, conv.ConversationID, "user", "msg3", 1)

	count, err := s.GetMessageCount(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetMessageCount: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestStoreGetMessageByID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	msg, _ := s.AddMessage(ctx, conv.ConversationID, "user", "find me", 3)

	found, err := s.GetMessageByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessageByID: %v", err)
	}
	if found.Content != "find me" {
		t.Errorf("content = %q, want %q", found.Content, "find me")
	}

	// Not found
	_, err = s.GetMessageByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for nonexistent message")
	}
}

func TestStoreUpdateMessageReasoningContent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:update-reasoning")

	msg, err := s.AddMessage(ctx, conv.ConversationID, "assistant", "answer", 3)
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	err = s.UpdateMessageReasoningContent(ctx, msg.ID, "thinking")
	if err != nil {
		t.Fatalf("UpdateMessageReasoningContent: %v", err)
	}

	found, err := s.GetMessageByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessageByID: %v", err)
	}
	if found.ReasoningContent != "thinking" {
		t.Errorf("ReasoningContent = %q, want %q", found.ReasoningContent, "thinking")
	}
}

// --- Summary Operations ---

func TestStoreCreateAndGetSummary(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	now := time.Now().UTC().Truncate(time.Second)
	summary, err := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:       conv.ConversationID,
		Kind:                 SummaryKindLeaf,
		Depth:                0,
		Content:              "test summary content",
		TokenCount:           50,
		EarliestAt:           &now,
		LatestAt:             &now,
		DescendantCount:      0,
		DescendantTokenCount: 0,
		SourceMessageTokens:  500,
		Model:                "test-model",
	})
	if err != nil {
		t.Fatalf("CreateSummary: %v", err)
	}
	if summary.SummaryID == "" {
		t.Error("expected non-empty summary ID")
	}
	if summary.Kind != SummaryKindLeaf {
		t.Errorf("kind = %q, want leaf", summary.Kind)
	}

	// Retrieve by ID
	found, err := s.GetSummary(ctx, summary.SummaryID)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if found.Content != "test summary content" {
		t.Errorf("content = %q, want 'test summary content'", found.Content)
	}
	if found.SourceMessageTokenCount != 500 {
		t.Errorf("source_message_token_count = %d, want 500", found.SourceMessageTokenCount)
	}
}

func TestStoreSummaryDAG(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create leaf summaries
	leaf1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "leaf 1",
		TokenCount:     100,
	})
	leaf2, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "leaf 2",
		TokenCount:     100,
	})

	// Create condensed summary with parents (the children being condensed)
	condensed, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:       conv.ConversationID,
		Kind:                 SummaryKindCondensed,
		Depth:                1,
		Content:              "condensed from leaves",
		TokenCount:           150,
		ParentIDs:            []string{leaf1.SummaryID, leaf2.SummaryID},
		DescendantCount:      2,
		DescendantTokenCount: 200,
	})

	// Get parents returns full Summary objects (not just IDs)
	parents, err := s.GetSummaryParents(ctx, condensed.SummaryID)
	if err != nil {
		t.Fatalf("GetSummaryParents: %v", err)
	}
	if len(parents) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(parents))
	}
	// Verify returned summaries have real content, not just IDs
	parentIDs := make(map[string]bool)
	for _, p := range parents {
		if p.Content == "" {
			t.Error("parent summary should have non-empty Content")
		}
		if p.TokenCount == 0 {
			t.Error("parent summary should have non-zero TokenCount")
		}
		parentIDs[p.SummaryID] = true
	}
	if !parentIDs[leaf1.SummaryID] || !parentIDs[leaf2.SummaryID] {
		t.Errorf("parent IDs = %v, want both %s and %s", parentIDs, leaf1.SummaryID, leaf2.SummaryID)
	}

	// Get children (summaries that have this one as parent)
	children, err := s.GetSummaryChildren(ctx, condensed.SummaryID)
	if err != nil {
		t.Fatalf("GetSummaryChildren: %v", err)
	}
	if len(children) != 0 {
		// condensed has no children yet — it's the root
		t.Errorf("expected 0 children, got %d", len(children))
	}

	// leaf summaries should have condensed as a "child" (reverse lookup)
	leafChildren, _ := s.GetSummaryChildren(ctx, leaf1.SummaryID)
	if len(leafChildren) != 1 || leafChildren[0] != condensed.SummaryID {
		t.Errorf("leaf1 children = %v, want [%s]", leafChildren, condensed.SummaryID)
	}
}

func TestStoreSummarySourceMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	msg1, _ := s.AddMessage(ctx, conv.ConversationID, "user", "msg1", 2)
	msg2, _ := s.AddMessage(ctx, conv.ConversationID, "assistant", "msg2", 3)

	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "summary of msg1 and msg2",
		TokenCount:     50,
	})

	err := s.LinkSummaryToMessages(ctx, summary.SummaryID, []int64{msg1.ID, msg2.ID})
	if err != nil {
		t.Fatalf("LinkSummaryToMessages: %v", err)
	}

	// Retrieve source messages
	msgs, err := s.GetSummarySourceMessages(ctx, summary.SummaryID)
	if err != nil {
		t.Fatalf("GetSummarySourceMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 source messages, got %d", len(msgs))
	}
}

func TestStoreGetRootSummaries(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create 2 leaf summaries
	leaf1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0, Content: "l1", TokenCount: 10,
	})
	leaf2, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0, Content: "l2", TokenCount: 10,
	})

	// Before condensation — both are roots
	roots, _ := s.GetRootSummaries(ctx, conv.ConversationID)
	if len(roots) != 2 {
		t.Errorf("before condensation: expected 2 roots, got %d", len(roots))
	}

	// Condense them
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindCondensed, Depth: 1,
		Content: "c1", TokenCount: 15, ParentIDs: []string{leaf1.SummaryID, leaf2.SummaryID},
	})

	// After condensation — only the condensed is root
	roots, _ = s.GetRootSummaries(ctx, conv.ConversationID)
	if len(roots) != 1 {
		t.Errorf("after condensation: expected 1 root, got %d", len(roots))
	}
	if roots[0].Kind != SummaryKindCondensed {
		t.Errorf("root kind = %q, want condensed", roots[0].Kind)
	}
}

// --- Context Item Operations ---

func TestStoreContextItems(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")
	msg1, _ := s.AddMessage(ctx, conv.ConversationID, "user", "hello", 2)
	msg2, _ := s.AddMessage(ctx, conv.ConversationID, "assistant", "world", 2)

	// Upsert items
	items := []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg1.ID, TokenCount: 2},
		{Ordinal: 200, ItemType: "message", MessageID: msg2.ID, TokenCount: 2},
	}
	err := s.UpsertContextItems(ctx, conv.ConversationID, items)
	if err != nil {
		t.Fatalf("UpsertContextItems: %v", err)
	}

	// Retrieve
	retrieved, err := s.GetContextItems(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextItems: %v", err)
	}
	if len(retrieved) != 2 {
		t.Fatalf("expected 2 items, got %d", len(retrieved))
	}
	if retrieved[0].Ordinal != 100 || retrieved[1].Ordinal != 200 {
		t.Errorf("ordinals = %v, want [100 200]", []int{retrieved[0].Ordinal, retrieved[1].Ordinal})
	}
	// CreatedAt should be populated
	if retrieved[0].CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated on context item")
	}
}

func TestStoreAppendContextMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")
	msg1, _ := s.AddMessage(ctx, conv.ConversationID, "user", "hello", 2)
	msg2, _ := s.AddMessage(ctx, conv.ConversationID, "assistant", "world", 2)

	s.UpsertContextItems(ctx, conv.ConversationID, []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg1.ID, TokenCount: 2},
	})

	// Append single message
	err := s.AppendContextMessage(ctx, conv.ConversationID, msg2.ID)
	if err != nil {
		t.Fatalf("AppendContextMessage: %v", err)
	}

	items, _ := s.GetContextItems(ctx, conv.ConversationID)
	if len(items) != 2 {
		t.Fatalf("expected 2 items after append, got %d", len(items))
	}
	if items[1].MessageID != msg2.ID {
		t.Errorf("appended message ID = %d, want %d", items[1].MessageID, msg2.ID)
	}
}

func TestStoreReplaceContextRangeWithSummary(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create messages and context items
	msgs := make([]int64, 4)
	for i := 0; i < 4; i++ {
		m, _ := s.AddMessage(ctx, conv.ConversationID, "user", "msg", 2)
		msgs[i] = m.ID
	}

	items := []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msgs[0], TokenCount: 2},
		{Ordinal: 200, ItemType: "message", MessageID: msgs[1], TokenCount: 2},
		{Ordinal: 300, ItemType: "message", MessageID: msgs[2], TokenCount: 2},
		{Ordinal: 400, ItemType: "message", MessageID: msgs[3], TokenCount: 2},
	}
	s.UpsertContextItems(ctx, conv.ConversationID, items)

	// Create a summary
	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "summary", TokenCount: 5,
	})

	// Replace ordinals 200-300 with summary
	err := s.ReplaceContextRangeWithSummary(ctx, conv.ConversationID, 200, 300, summary.SummaryID)
	if err != nil {
		t.Fatalf("ReplaceContextRangeWithSummary: %v", err)
	}

	// Verify: should have 3 items — msg[0], summary, msg[3]
	result, _ := s.GetContextItems(ctx, conv.ConversationID)
	if len(result) != 3 {
		t.Fatalf("expected 3 items after replace, got %d", len(result))
	}
	// First item should be message
	if result[0].ItemType != "message" || result[0].MessageID != msgs[0] {
		t.Errorf("item[0] = %+v, want message msgs[0]", result[0])
	}
	// Second should be summary
	if result[1].ItemType != "summary" || result[1].SummaryID != summary.SummaryID {
		t.Errorf("item[1] = %+v, want summary", result[1])
	}
	// Third should be message
	if result[2].ItemType != "message" || result[2].MessageID != msgs[3] {
		t.Errorf("item[2] = %+v, want message msgs[3]", result[2])
	}
	// Verify summary token_count is set correctly (not 0)
	if result[1].TokenCount != 5 {
		t.Errorf("summary item TokenCount = %d, want 5 (from summary.TokenCount)", result[1].TokenCount)
	}
}

func TestStoreReplaceContextRangeResequenceOrdinals(t *testing.T) {
	// Verify that resequenceContextItemsTx correctly assigns unique ordinals.
	// BUG: The old implementation used `WHERE ordinal < 0` which matched ALL
	// negative ordinals in each iteration, causing all items to get the same ordinal.
	//
	// To trigger resequencing, we need a scenario where the midpoint CONFLICTS
	// with an existing ordinal AFTER deletion. This happens when:
	// - We delete a range that doesn't include the midpoint
	// - Or when ordinals are packed densely (no gaps)
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test-resequence")

	// Create 5 messages with DENSE ordinals (no gaps) to trigger conflict
	msgs := make([]int64, 5)
	for i := 0; i < 5; i++ {
		m, _ := s.AddMessage(ctx, conv.ConversationID, "user", fmt.Sprintf("msg%d", i), 2)
		msgs[i] = m.ID
	}

	// Use dense ordinals: 100, 101, 102, 103, 104
	// When we delete 101-102 and insert at midpoint 101, it won't conflict.
	// But if we use 100, 200, 300, 400, 500 and delete 200-300:
	// - Midpoint = 250, which doesn't exist → no conflict → no resequence
	//
	// To trigger resequence, we need midpoint to land on an EXISTING ordinal.
	// Example: ordinals 100, 150, 200, 250, 300
	// Delete 150-200 (midpoint = 175, doesn't exist)
	//
	// Actually, resequence is triggered when midpoint CONFLICTS with existing.
	// Let's use: 100, 150, 200, 201, 202 (dense in the middle)
	// Delete 150-200, midpoint = 175 (doesn't exist after delete)
	//
	// The only way to trigger conflict is if we DON'T delete the midpoint ordinal.
	// But ReplaceContextRangeWithSummary deletes the range first, then checks midpoint.
	//
	// Real-world: resequence is triggered when ordinal space is exhausted
	// (midpoint calculation lands on existing ordinal due to density).
	// Let's simulate this by having many items with ordinal_step=1:
	items := []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msgs[0], TokenCount: 2},
		{Ordinal: 101, ItemType: "message", MessageID: msgs[1], TokenCount: 2},
		{Ordinal: 102, ItemType: "message", MessageID: msgs[2], TokenCount: 2},
		{Ordinal: 103, ItemType: "message", MessageID: msgs[3], TokenCount: 2},
		{Ordinal: 104, ItemType: "message", MessageID: msgs[4], TokenCount: 2},
	}
	s.UpsertContextItems(ctx, conv.ConversationID, items)

	// Create a summary
	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "summary", TokenCount: 5,
	})

	// Delete 101-102, insert at midpoint 101
	// After delete: 100, 103, 104
	// Midpoint = (101+102)/2 = 101, which doesn't exist after delete
	// → No conflict, insert at 101
	// → Result: 100, 101 (summary), 103, 104
	//
	// This still doesn't trigger resequence! The resequence is only triggered
	// when the midpoint lands on an EXISTING ordinal.
	//
	// Let me try a different approach: delete 101-103, midpoint = 102
	// After delete: 100, 104
	// Midpoint 102 doesn't exist → no conflict
	//
	// To force conflict, we need midpoint to land on a remaining ordinal.
	// With ordinals 100, 101, 102, 103, 104:
	// Delete 100-101, midpoint = 100 (exists? NO, we deleted it!)
	//
	// The resequence is triggered when we can't find a gap to insert.
	// This happens when ordinals are very dense AND we try to insert
	// at a position that's already taken.
	//
	// Actually, let's just test the happy path where resequence ISN'T triggered,
	// and verify ordinals are still correct:

	err := s.ReplaceContextRangeWithSummary(ctx, conv.ConversationID, 101, 102, summary.SummaryID)
	if err != nil {
		t.Fatalf("ReplaceContextRangeWithSummary: %v", err)
	}

	result, _ := s.GetContextItems(ctx, conv.ConversationID)
	if len(result) != 4 {
		t.Fatalf("expected 4 items after replace, got %d", len(result))
	}

	// After replace: 100 (msg0), 101 (summary), 103 (msg3), 104 (msg4)
	expectedOrdinals := []int{100, 101, 103, 104}
	for i, item := range result {
		if item.Ordinal != expectedOrdinals[i] {
			t.Errorf("item[%d].Ordinal = %d, want %d", i, item.Ordinal, expectedOrdinals[i])
		}
	}

	// Verify no duplicate ordinals
	ordinalSet := make(map[int]bool)
	for _, item := range result {
		if ordinalSet[item.Ordinal] {
			t.Errorf("duplicate ordinal %d detected", item.Ordinal)
		}
		ordinalSet[item.Ordinal] = true
	}
}

func TestResequenceContextItemsTxAssignsUniqueOrdinals(t *testing.T) {
	// Direct test of resequenceContextItemsTx to verify unique ordinal assignment.
	// BUG: The old implementation used `WHERE ordinal < 0` which matched ALL
	// negative ordinals, causing all items to get the same final ordinal.
	//
	// Example with 3 items at temp ordinals -1, -2, -3:
	// - Loop 1: UPDATE ... SET ordinal=100 WHERE ordinal<0 → ALL become 100
	// - Loop 2: UPDATE ... SET ordinal=200 WHERE ordinal<0 → ALL become 200
	// - Loop 3: UPDATE ... SET ordinal=300 WHERE ordinal<0 → ALL become 300
	// Result: [300, 300, 300] - WRONG!
	//
	// Fixed: Use specific temp ordinal matching:
	// - Loop 1: UPDATE ... SET ordinal=100 WHERE ordinal=-1
	// - Loop 2: UPDATE ... SET ordinal=200 WHERE ordinal=-2
	// - Loop 3: UPDATE ... SET ordinal=300 WHERE ordinal=-3
	// Result: [100, 200, 300] - CORRECT!

	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test-resequence-direct")

	// Create messages
	msgs := make([]int64, 5)
	for i := 0; i < 5; i++ {
		m, _ := s.AddMessage(ctx, conv.ConversationID, "user", fmt.Sprintf("msg%d", i), 2)
		msgs[i] = m.ID
	}

	// Use ordinals that will trigger resequence when we try to insert at midpoint
	// The key is to have a scenario where ReplaceContextRangeWithSummary calls resequenceContextItemsTx
	//
	// To trigger resequence, we need midpoint to conflict with an EXISTING ordinal
	// AFTER the range deletion. This happens when:
	// - Ordinals are: 100, 200, 201, 202, 300 (dense in middle)
	// - Delete 200-202 (midpoint = 201, deleted)
	// - After delete: 100, 300
	// - Midpoint 201 doesn't exist → no conflict
	//
	// Alternative: Use transaction directly to test resequenceContextItemsTx

	// First set up context items
	items := []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msgs[0], TokenCount: 2},
		{Ordinal: 200, ItemType: "message", MessageID: msgs[1], TokenCount: 2},
		{Ordinal: 300, ItemType: "message", MessageID: msgs[2], TokenCount: 2},
		{Ordinal: 400, ItemType: "message", MessageID: msgs[3], TokenCount: 2},
		{Ordinal: 500, ItemType: "message", MessageID: msgs[4], TokenCount: 2},
	}
	s.UpsertContextItems(ctx, conv.ConversationID, items)

	// Create a summary
	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "summary", TokenCount: 5,
	})

	// Call resequenceContextItemsTx directly via a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer tx.Rollback()

	err = s.resequenceContextItemsTx(ctx, tx, conv.ConversationID, summary.SummaryID)
	if err != nil {
		t.Fatalf("resequenceContextItemsTx: %v", err)
	}
	tx.Commit()

	// Verify ordinals are unique and properly spaced
	result, _ := s.GetContextItems(ctx, conv.ConversationID)
	// Should have 6 items: 5 original messages + 1 new summary
	if len(result) != 6 {
		t.Fatalf("expected 6 items after resequence, got %d", len(result))
	}

	// Expected ordinals: 100, 200, 300, 400, 500, 600
	// (5 existing items get 100-500, new summary gets 600)
	expectedOrdinals := []int{100, 200, 300, 400, 500, 600}
	for i, item := range result {
		if item.Ordinal != expectedOrdinals[i] {
			t.Errorf("item[%d].Ordinal = %d, want %d", i, item.Ordinal, expectedOrdinals[i])
		}
	}

	// Verify no duplicate ordinals
	ordinalSet := make(map[int]bool)
	for _, item := range result {
		if ordinalSet[item.Ordinal] {
			t.Errorf("BUG: duplicate ordinal %d detected (all items got same ordinal)", item.Ordinal)
		}
		ordinalSet[item.Ordinal] = true
	}

	// Verify summary token_count is set correctly (not 0)
	var summaryItem *ContextItem
	for i := range result {
		if result[i].ItemType == "summary" {
			summaryItem = &result[i]
			break
		}
	}
	if summaryItem == nil {
		t.Fatal("no summary item found after resequence")
	}
	if summaryItem.TokenCount != 5 {
		t.Errorf("summary item TokenCount = %d, want 5 (from summary.TokenCount)", summaryItem.TokenCount)
	}
}

func TestStoreGetContextTokenCount(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")
	msg, _ := s.AddMessage(ctx, conv.ConversationID, "user", "hello", 0)

	s.UpsertContextItems(ctx, conv.ConversationID, []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg.ID, TokenCount: 42},
	})

	count, err := s.GetContextTokenCount(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextTokenCount: %v", err)
	}
	if count != 42 {
		t.Errorf("token count = %d, want 42", count)
	}
}

func TestStoreGetMaxOrdinal(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// No items yet
	maxOrd, err := s.GetMaxOrdinal(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetMaxOrdinal (empty): %v", err)
	}
	if maxOrd != 0 {
		t.Errorf("max ordinal (empty) = %d, want 0", maxOrd)
	}

	// Add items
	msg1, _ := s.AddMessage(ctx, conv.ConversationID, "user", "a", 1)
	msg2, _ := s.AddMessage(ctx, conv.ConversationID, "user", "b", 1)
	s.UpsertContextItems(ctx, conv.ConversationID, []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg1.ID, TokenCount: 1},
		{Ordinal: 250, ItemType: "message", MessageID: msg2.ID, TokenCount: 1},
	})

	maxOrd, _ = s.GetMaxOrdinal(ctx, conv.ConversationID)
	if maxOrd != 250 {
		t.Errorf("max ordinal = %d, want 250", maxOrd)
	}
}

// --- GetDistinctDepthsInContext ---

func TestStoreGetDistinctDepthsInContext(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Empty context → no depths
	depths, err := s.GetDistinctDepthsInContext(ctx, conv.ConversationID, 0)
	if err != nil {
		t.Fatalf("GetDistinctDepthsInContext (empty): %v", err)
	}
	if len(depths) != 0 {
		t.Errorf("empty context: depths = %v, want []", depths)
	}

	// Add leaf summaries at depth 0
	now := time.Now().UTC()
	s1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "leaf1", TokenCount: 10, EarliestAt: &now, LatestAt: &now,
	})
	s2, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "leaf2", TokenCount: 10, EarliestAt: &now, LatestAt: &now,
	})

	// Add summaries to context
	s.UpsertContextItems(ctx, conv.ConversationID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: s1.SummaryID, TokenCount: 10},
		{Ordinal: 200, ItemType: "summary", SummaryID: s2.SummaryID, TokenCount: 10},
	})

	// Should find depth 0
	depths, err = s.GetDistinctDepthsInContext(ctx, conv.ConversationID, 0)
	if err != nil {
		t.Fatalf("GetDistinctDepthsInContext: %v", err)
	}
	if len(depths) != 1 || depths[0] != 0 {
		t.Errorf("depths = %v, want [0]", depths)
	}

	// Add condensed at depth 1
	c1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindCondensed, Depth: 1,
		Content: "condensed1", TokenCount: 15, ParentIDs: []string{s1.SummaryID, s2.SummaryID},
	})
	s.AppendContextSummary(ctx, conv.ConversationID, c1.SummaryID)

	// Should find depths [0, 1] or [1, 0]
	depths, _ = s.GetDistinctDepthsInContext(ctx, conv.ConversationID, 0)
	if len(depths) != 2 {
		t.Errorf("with condensed: depths = %v, want 2 distinct depths", depths)
	}

	// Test maxOrdinalExclusive filter
	// Get depths excluding ordinals >= 300 (the condensed one)
	depths, _ = s.GetDistinctDepthsInContext(ctx, conv.ConversationID, 300)
	if len(depths) != 1 || depths[0] != 0 {
		t.Errorf("filtered depths = %v, want [0]", depths)
	}
}

// --- GetSummarySubtree ---

func TestStoreGetSummarySubtree(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create leaf summaries
	now := time.Now().UTC()
	l1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "leaf1", TokenCount: 10, EarliestAt: &now, LatestAt: &now,
	})
	l2, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "leaf2", TokenCount: 10, EarliestAt: &now, LatestAt: &now,
	})
	l3, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "leaf3", TokenCount: 10, EarliestAt: &now, LatestAt: &now,
	})

	// Condense l1+l2 → c1
	c1, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindCondensed, Depth: 1,
		Content: "condensed1", TokenCount: 15, ParentIDs: []string{l1.SummaryID, l2.SummaryID},
	})

	// Get subtree from c1
	nodes, err := s.GetSummarySubtree(ctx, c1.SummaryID)
	if err != nil {
		t.Fatalf("GetSummarySubtree: %v", err)
	}

	// Should include c1 itself + l1 + l2 (but NOT l3)
	if len(nodes) != 3 {
		t.Errorf("subtree nodes = %d, want 3", len(nodes))
	}

	// Verify l3 is NOT in the subtree
	for _, n := range nodes {
		if n.SummaryID == l3.SummaryID {
			t.Error("l3 should not be in c1's subtree")
		}
	}

	// Verify c1 has depth-from-root 0
	for _, n := range nodes {
		if n.SummaryID == c1.SummaryID && n.DepthFromRoot != 0 {
			t.Errorf("c1 depth-from-root = %d, want 0", n.DepthFromRoot)
		}
	}
}

// --- Search with Rank and Time Filters ---

func TestStoreSearchSummariesWithRank(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create summaries with different content (for FTS matching)
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "machine learning neural network", TokenCount: 10,
	})
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "deep learning reinforcement", TokenCount: 10,
	})

	// FTS search — results should have Rank populated
	results, err := s.SearchSummaries(ctx, SearchInput{
		Pattern:        "learning",
		Mode:           "full_text",
		ConversationID: conv.ConversationID,
	})
	if err != nil {
		t.Fatalf("SearchSummaries: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("expected at least 1 result, got %d", len(results))
	}
	// Rank should be populated (negative value from bm25)
	for _, r := range results {
		if r.Rank == 0 {
			t.Error("expected non-zero Rank from FTS search")
		}
	}
}

func TestStoreSearchSummariesWithTimeFilter(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create a summary
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID, Kind: SummaryKindLeaf, Depth: 0,
		Content: "important meeting notes", TokenCount: 10,
	})

	// Search with Since filter (now - 1 hour → should match)
	since := time.Now().UTC().Add(-1 * time.Hour)
	results, err := s.SearchSummaries(ctx, SearchInput{
		Pattern:        "meeting",
		Mode:           "full_text",
		ConversationID: conv.ConversationID,
		Since:          &since,
	})
	if err != nil {
		t.Fatalf("SearchSummaries with Since: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Since=1h-ago: expected 1 result, got %d", len(results))
	}

	// Search with Before filter (1 hour in future → should match)
	before := time.Now().UTC().Add(1 * time.Hour)
	results, err = s.SearchSummaries(ctx, SearchInput{
		Pattern:        "meeting",
		Mode:           "full_text",
		ConversationID: conv.ConversationID,
		Before:         &before,
	})
	if err != nil {
		t.Fatalf("SearchSummaries with Before: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Before=1h-future: expected 1 result, got %d", len(results))
	}

	// Search with Since in the future → should NOT match
	futureSince := time.Now().UTC().Add(1 * time.Hour)
	results, err = s.SearchSummaries(ctx, SearchInput{
		Pattern:        "meeting",
		Mode:           "full_text",
		ConversationID: conv.ConversationID,
		Since:          &futureSince,
	})
	if err != nil {
		t.Fatalf("SearchSummaries with future Since: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Since=1h-future: expected 0 results, got %d", len(results))
	}
}

func TestSearchMessagesUsesFTS5(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "test:fts5-messages")
	convID := conv.ConversationID

	// Add messages with searchable content
	s.AddMessage(ctx, convID, "user", "The quick brown fox jumps over the lazy dog", 10)
	s.AddMessage(ctx, convID, "assistant", "A response about something else entirely", 10)
	s.AddMessage(ctx, convID, "user", "Five boxing wizards jump quickly at dawn", 10)

	input := SearchInput{
		Pattern:        "fox jumps",
		Mode:           "full_text",
		ConversationID: convID,
		Limit:          10,
	}

	results, err := s.SearchMessages(ctx, input)
	if err != nil {
		t.Fatalf("SearchMessages FTS5: %v", err)
	}

	// Should find the message containing "fox jumps"
	found := false
	for _, r := range results {
		if r.MessageID > 0 && contains(r.Snippet, "fox") {
			found = true
			break
		}
	}
	if !found {
		t.Error("FTS5 search should find message with 'fox jumps'")
	}
}

func TestMessagesFTSTriggers(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "test:fts-triggers")
	convID := conv.ConversationID

	// Insert a message
	_, err := s.AddMessage(ctx, convID, "user", "database migration completed successfully", 10)
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	// Verify FTS table was populated by INSERT trigger
	var count int
	err = s.db.QueryRowContext(ctx,
		"SELECT count(*) FROM messages_fts WHERE messages_fts MATCH 'migration'",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query messages_fts: %v", err)
	}
	if count != 1 {
		t.Errorf("messages_fts should have 1 row after INSERT, got %d", count)
	}

	// Verify the content column has the right text
	var content string
	err = s.db.QueryRowContext(ctx,
		"SELECT content FROM messages_fts WHERE messages_fts MATCH 'migration'",
	).Scan(&content)
	if err != nil {
		t.Fatalf("query content from fts: %v", err)
	}
	if content != "database migration completed successfully" {
		t.Errorf("fts content = %q, want original message content", content)
	}
}

func TestSearchMessagesWithTimeFilter(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "test:msg-time")
	convID := conv.ConversationID

	// Add messages
	s.AddMessage(ctx, convID, "user", "important deployment notes", 10)

	// Search with Since filter (1 hour ago → should match)
	since := time.Now().UTC().Add(-1 * time.Hour)
	results, err := s.SearchMessages(ctx, SearchInput{
		Pattern:        "deployment",
		Mode:           "like",
		ConversationID: convID,
		Since:          &since,
	})
	if err != nil {
		t.Fatalf("SearchMessages with Since: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Since=1h-ago: expected 1 result, got %d", len(results))
	}

	// Search with Before filter (1 hour in future → should match)
	before := time.Now().UTC().Add(1 * time.Hour)
	results, err = s.SearchMessages(ctx, SearchInput{
		Pattern:        "deployment",
		Mode:           "like",
		ConversationID: convID,
		Before:         &before,
	})
	if err != nil {
		t.Fatalf("SearchMessages with Before: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Before=1h-future: expected 1 result, got %d", len(results))
	}

	// Search with Since in the future → should NOT match
	futureSince := time.Now().UTC().Add(1 * time.Hour)
	results, err = s.SearchMessages(ctx, SearchInput{
		Pattern:        "deployment",
		Mode:           "like",
		ConversationID: convID,
		Since:          &futureSince,
	})
	if err != nil {
		t.Fatalf("SearchMessages with future Since: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Since=1h-future: expected 0 results, got %d", len(results))
	}
}

func TestStoreSearchSummariesReturnsContent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test")

	// Create a summary with known content
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "This is the summary content for testing",
		TokenCount:     10,
	})

	// Search should return the full content, not empty
	results, err := s.SearchSummaries(ctx, SearchInput{
		Pattern:        "summary content",
		Mode:           "like",
		ConversationID: conv.ConversationID,
	})
	if err != nil {
		t.Fatalf("SearchSummaries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content == "" {
		t.Error("SearchResult.Content is empty, want full summary content")
	}
	if results[0].Content != "This is the summary content for testing" {
		t.Errorf("SearchResult.Content = %q, want %q", results[0].Content, "This is the summary content for testing")
	}
}

func TestStoreReplaceContextItemsWithSummary(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, _ := s.GetOrCreateConversation(ctx, "agent:test-replace-items")

	// Create messages
	msgs := make([]int64, 5)
	for i := 0; i < 5; i++ {
		m, _ := s.AddMessage(ctx, conv.ConversationID, "user", fmt.Sprintf("msg%d", i), 2)
		msgs[i] = m.ID
	}

	// Create summaries
	summaries := make([]string, 3)
	for i := 0; i < 3; i++ {
		sum, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: conv.ConversationID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("summary %d", i),
			TokenCount:     10,
		})
		summaries[i] = sum.SummaryID
	}

	// Insert context items with a message in between summaries:
	// Ordinals: 100 (summary0), 200 (message), 300 (summary1), 400 (summary2)
	items := []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: summaries[0], TokenCount: 10},
		{Ordinal: 200, ItemType: "message", MessageID: msgs[1], TokenCount: 2},
		{Ordinal: 300, ItemType: "summary", SummaryID: summaries[1], TokenCount: 10},
		{Ordinal: 400, ItemType: "summary", SummaryID: summaries[2], TokenCount: 10},
	}
	s.UpsertContextItems(ctx, conv.ConversationID, items)

	// Create a new summary to replace with
	newSummary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindCondensed,
		Depth:          1,
		Content:        "condensed summary",
		TokenCount:     15,
	})

	// Replace summaries 0 and 1 (not 2) using per-item deletion
	// This should NOT delete the message at ordinal 200
	err := s.ReplaceContextItemsWithSummary(
		ctx, conv.ConversationID,
		[]string{summaries[0], summaries[1]},
		newSummary.SummaryID)
	if err != nil {
		t.Fatalf("ReplaceContextItemsWithSummary: %v", err)
	}

	// Verify result: should have 3 items (message at 200, summary2 at 400, new summary)
	result, _ := s.GetContextItems(ctx, conv.ConversationID)
	if len(result) != 3 {
		t.Fatalf("expected 3 items after replace, got %d", len(result))
	}

	// Verify message at ordinal 200 is preserved
	messagePreserved := false
	for _, item := range result {
		if item.ItemType == "message" && item.MessageID == msgs[1] {
			messagePreserved = true
			break
		}
	}
	if !messagePreserved {
		t.Error("message at ordinal 200 should have been preserved")
	}

	// Verify summary2 at ordinal 400 is preserved
	summary2Preserved := false
	for _, item := range result {
		if item.ItemType == "summary" && item.SummaryID == summaries[2] {
			summary2Preserved = true
			break
		}
	}
	if !summary2Preserved {
		t.Error("summary2 at ordinal 400 should have been preserved")
	}

	// Verify new summary exists
	newSummaryFound := false
	for _, item := range result {
		if item.ItemType == "summary" && item.SummaryID == newSummary.SummaryID {
			newSummaryFound = true
			break
		}
	}
	if !newSummaryFound {
		t.Error("new summary should exist")
	}

	// Verify no duplicate ordinals
	ordinalSet := make(map[int]bool)
	for _, item := range result {
		if ordinalSet[item.Ordinal] {
			t.Errorf("duplicate ordinal %d detected", item.Ordinal)
		}
		ordinalSet[item.Ordinal] = true
	}
}

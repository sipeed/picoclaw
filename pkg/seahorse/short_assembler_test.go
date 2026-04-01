package seahorse

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- Assembler Tests ---

// helper: create a store with messages and summaries for assembly tests
func setupAssemblerStore(t *testing.T) (*Store, int64) {
	t.Helper()
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.GetOrCreateConversation(ctx, "test:assemble")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	return s, conv.ConversationID
}

func TestAssemblerAssembleEmpty(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("Messages = %d, want 0", len(result.Messages))
	}
	if len(result.Summaries) != 0 {
		t.Errorf("Summaries = %d, want 0", len(result.Summaries))
	}
	if result.TokenCount != 0 {
		t.Errorf("TokenCount = %d, want 0", result.TokenCount)
	}
}

func TestAssemblerAssembleMessagesOnly(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create messages
	msg1, _ := s.AddMessage(ctx, convID, "user", "hello", 5)
	msg2, _ := s.AddMessage(ctx, convID, "assistant", "world", 5)

	// Create context items
	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg1.ID, TokenCount: 5},
		{Ordinal: 200, ItemType: "message", MessageID: msg2.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 100})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Fatalf("Messages = %d, want 2", len(result.Messages))
	}
	if result.Messages[0].Content != "hello" {
		t.Errorf("Messages[0].Content = %q, want 'hello'", result.Messages[0].Content)
	}
	if result.Messages[1].Content != "world" {
		t.Errorf("Messages[1].Content = %q, want 'world'", result.Messages[1].Content)
	}
	if result.TokenCount != 10 {
		t.Errorf("TokenCount = %d, want 10", result.TokenCount)
	}
}

func TestAssemblerAssembleWithSummary(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create a summary
	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "summary of early messages",
		TokenCount:     50,
	})

	// Create recent messages
	msg1, _ := s.AddMessage(ctx, convID, "user", "recent", 5)
	msg2, _ := s.AddMessage(ctx, convID, "assistant", "reply", 5)

	// Context: summary + recent messages
	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: summary.SummaryID, TokenCount: 50},
		{Ordinal: 200, ItemType: "message", MessageID: msg1.ID, TokenCount: 5},
		{Ordinal: 300, ItemType: "message", MessageID: msg2.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Summaries) != 1 {
		t.Fatalf("Summaries = %d, want 1", len(result.Summaries))
	}
	if result.Summaries[0].SummaryID != summary.SummaryID {
		t.Errorf("Summary ID = %q, want %q", result.Summaries[0].SummaryID, summary.SummaryID)
	}
	// Messages = 1 summary-as-user-msg + 2 raw messages = 3
	if len(result.Messages) != 3 {
		t.Errorf("Messages = %d, want 3 (1 summary + 2 raw)", len(result.Messages))
	}
}

func TestAssemblerBudgetEvictsOldest(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create 40 messages, each with 10 tokens = 400 total
	msgs := make([]*Message, 40)
	for i := 0; i < 40; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "msg", 10)
		msgs[i] = m
	}

	// Context items for all messages
	items := make([]ContextItem, 40)
	for i := 0; i < 40; i++ {
		items[i] = ContextItem{
			Ordinal:    (i + 1) * 100,
			ItemType:   "message",
			MessageID:  msgs[i].ID,
			TokenCount: 10,
		}
	}
	s.UpsertContextItems(ctx, convID, items)

	// Budget of 200 tokens with FreshTailCount=32
	// Fresh tail = last 32 messages (320 tokens, over budget, but always included)
	// Evictable = first 8 messages (80 tokens)
	// Budget after tail: max(0, 200-320) = 0 → no evictable items included
	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 200})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Should only include the 32-item fresh tail
	if len(result.Messages) != 32 {
		t.Errorf("Messages = %d, want 32 (fresh tail)", len(result.Messages))
	}
	// Should be the LAST 32 messages
	if result.Messages[0].ID != msgs[8].ID {
		t.Errorf("first message ID = %d, want %d (msgs[8])", result.Messages[0].ID, msgs[8].ID)
	}
}

func TestAssemblerBudgetFitsAll(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	msgs := make([]*Message, 5)
	for i := 0; i < 5; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "msg", 10)
		msgs[i] = m
	}

	items := make([]ContextItem, 5)
	for i := 0; i < 5; i++ {
		items[i] = ContextItem{
			Ordinal:    (i + 1) * 100,
			ItemType:   "message",
			MessageID:  msgs[i].ID,
			TokenCount: 10,
		}
	}
	s.UpsertContextItems(ctx, convID, items)

	// Budget = 100, total = 50, FreshTailCount=32 → all items in tail
	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 100})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Messages) != 5 {
		t.Errorf("Messages = %d, want 5", len(result.Messages))
	}
}

func TestAssemblerSummaryXMLFormat(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "test summary content",
		TokenCount:     20,
	})

	msg, _ := s.AddMessage(ctx, convID, "user", "hello", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: summary.SummaryID, TokenCount: 20},
		{Ordinal: 200, ItemType: "message", MessageID: msg.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Summary should be formatted as XML system message
	if len(result.Messages) < 1 {
		t.Fatal("expected at least 1 message (summary formatted as message)")
	}
	summaryMsg := result.Messages[0]
	if summaryMsg.Role != "system" {
		t.Errorf("summary message role = %q, want 'system'", summaryMsg.Role)
	}
	// Should contain XML summary tags
	if !contains(summaryMsg.Content, "<summary") {
		t.Errorf("summary message missing <summary tag: %q", summaryMsg.Content)
	}
	if !contains(summaryMsg.Content, summary.SummaryID) {
		t.Errorf("summary message missing summary ID: %q", summaryMsg.Content)
	}
}

func TestAssemblerSummaryXMLEscaping(t *testing.T) {
	// Summary content with special XML characters should be properly escaped
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create summary with content containing XML special characters
	summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        `User said: "hello" & asked about <tags>`,
		TokenCount:     20,
	})

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: summary.SummaryID, TokenCount: 20},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}

	// The XML should be well-formed - no unescaped special characters
	xmlContent := result.Messages[0].Content

	// Check that special characters are escaped
	if strings.Contains(xmlContent, "<tags>") {
		t.Errorf("BUG: unescaped < in summary content: %q", xmlContent)
	}
	if strings.Contains(xmlContent, `"hello"`) {
		t.Errorf("BUG: unescaped \" in summary content: %q", xmlContent)
	}
	// & should be escaped as &amp;
	if strings.Contains(xmlContent, " & ") {
		t.Errorf("BUG: unescaped & in summary content: %q", xmlContent)
	}
}

func TestAssemblerSourceIDs(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create messages
	msg1, _ := s.AddMessage(ctx, convID, "user", "hello", 5)
	msg2, _ := s.AddMessage(ctx, convID, "assistant", "world", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "message", MessageID: msg1.ID, TokenCount: 5},
		{Ordinal: 200, ItemType: "message", MessageID: msg2.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 100})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// SourceIDs should contain message IDs
	if len(result.SourceIDs) != 2 {
		t.Errorf("SourceIDs = %d, want 2", len(result.SourceIDs))
	}
}

func TestAssemblerSummaryXMLWithParents(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create a leaf and a condensed summary (condensed has parent)
	leaf, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "leaf content",
		TokenCount:     20,
	})
	condensed, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindCondensed,
		Depth:          1,
		Content:        "condensed content",
		TokenCount:     15,
		ParentIDs:      []string{leaf.SummaryID},
	})

	msg, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: condensed.SummaryID, TokenCount: 15},
		{Ordinal: 200, ItemType: "message", MessageID: msg.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Find the summary message (first message, formatted as XML)
	if len(result.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}
	xmlContent := result.Messages[0].Content

	// Should contain <parents> section with parent ID
	if !contains(xmlContent, "<parents>") {
		t.Errorf("condensed summary XML missing <parents> section: %q", xmlContent)
	}
	if !contains(xmlContent, leaf.SummaryID) {
		t.Errorf("condensed summary XML missing parent ID %q: %q", leaf.SummaryID, xmlContent)
	}

	// Should contain kind="condensed"
	if !contains(xmlContent, `kind="condensed"`) {
		t.Errorf("condensed summary XML missing kind attribute: %q", xmlContent)
	}
}

func TestAssemblerSummaryXMLIncludesDescendantCount(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create a leaf summary with specific descendant count
	leaf, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:       convID,
		Kind:                 SummaryKindLeaf,
		Depth:                0,
		Content:              "leaf content",
		TokenCount:           20,
		DescendantCount:      8,
		DescendantTokenCount: 1200,
	})

	msg, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: leaf.SummaryID, TokenCount: 20},
		{Ordinal: 200, ItemType: "message", MessageID: msg.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}
	xmlContent := result.Messages[0].Content

	// Should contain descendant_count="8"
	if !contains(xmlContent, `descendant_count="8"`) {
		t.Errorf("summary XML missing descendant_count attribute: %q", xmlContent)
	}
}

func TestAssemblerLeafSummaryNoParents(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Leaf summary has no parents
	leaf, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "leaf content",
		TokenCount:     20,
	})

	msg, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: leaf.SummaryID, TokenCount: 20},
		{Ordinal: 200, ItemType: "message", MessageID: msg.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	xmlContent := result.Messages[0].Content

	// Leaf summary should NOT have <parents> section
	if contains(xmlContent, "<parents>") {
		t.Errorf("leaf summary XML should not have <parents> section: %q", xmlContent)
	}
}

func TestAssemblerDepthAwarePrompt(t *testing.T) {
	s, convID := setupAssemblerStore(t)
	ctx := context.Background()

	// Create a condensed summary (depth >= 2) to trigger full guidance
	now := time.Now().UTC()
	leaf, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "leaf summary",
		TokenCount:     20,
		EarliestAt:     &now,
		LatestAt:       &now,
	})
	condensed, _ := s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:       convID,
		Kind:                 SummaryKindCondensed,
		Depth:                2,
		Content:              "condensed summary",
		TokenCount:           15,
		ParentIDs:            []string{leaf.SummaryID},
		DescendantCount:      1,
		DescendantTokenCount: 20,
	})

	msg, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)

	s.UpsertContextItems(ctx, convID, []ContextItem{
		{Ordinal: 100, ItemType: "summary", SummaryID: condensed.SummaryID, TokenCount: 15},
		{Ordinal: 200, ItemType: "message", MessageID: msg.ID, TokenCount: 5},
	})

	a := &Assembler{store: s, config: Config{}}
	result, err := a.Assemble(ctx, convID, AssembleInput{Budget: 1000})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Should have a depth-aware prompt
	if result.SystemPromptAddition == "" {
		t.Error("expected non-empty SystemPromptAddition when depth >= 2")
	}
}

func TestFormatSummaryXMLUsesSummaryRef(t *testing.T) {
	// Spec: condensed summaries use <summary_ref id="parentId" /> not <parent>parentId</parent>
	now := time.Now().UTC()
	s := Summary{
		SummaryID:       "sum_condensed1",
		Kind:            SummaryKindCondensed,
		Depth:           1,
		Content:         "condensed content",
		TokenCount:      50,
		DescendantCount: 2,
		EarliestAt:      &now,
		LatestAt:        &now,
	}
	parentIDs := []string{"sum_leaf1", "sum_leaf2"}

	xml := FormatSummaryXML(&s, parentIDs)

	// Must use <summary_ref id="..." /> per spec
	if !contains(xml, `<summary_ref id="sum_leaf1" />`) {
		t.Errorf("expected <summary_ref id=\"sum_leaf1\" />, got: %s", xml)
	}
	if !contains(xml, `<summary_ref id="sum_leaf2" />`) {
		t.Errorf("expected <summary_ref id=\"sum_leaf2\" />, got: %s", xml)
	}
	// Must NOT use old <parent> tag
	if contains(xml, "<parent>") {
		t.Errorf("should not use <parent> tag, got: %s", xml)
	}
}

func TestFormatSummaryXMLIncludesTimestamps(t *testing.T) {
	// Spec: summary XML includes earliest_at and latest_at attributes
	earliest := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	s := Summary{
		SummaryID:       "sum_leaf1",
		Kind:            SummaryKindLeaf,
		Depth:           0,
		Content:         "leaf content",
		TokenCount:      30,
		DescendantCount: 0,
		EarliestAt:      &earliest,
		LatestAt:        &latest,
	}

	xml := FormatSummaryXML(&s, nil)

	if !contains(xml, `earliest_at="2026-03-15T10:00:00Z"`) {
		t.Errorf("missing earliest_at attribute, got: %s", xml)
	}
	if !contains(xml, `latest_at="2026-03-15T14:30:00Z"`) {
		t.Errorf("missing latest_at attribute, got: %s", xml)
	}
}

func TestFormatSummaryXMLNoTimestampsWhenNil(t *testing.T) {
	// When EarliestAt/LatestAt are nil, attributes should be omitted
	s := Summary{
		SummaryID:       "sum_leaf1",
		Kind:            SummaryKindLeaf,
		Depth:           0,
		Content:         "leaf content",
		TokenCount:      30,
		DescendantCount: 0,
	}

	xml := FormatSummaryXML(&s, nil)

	if contains(xml, "earliest_at=") {
		t.Errorf("should not have earliest_at when nil, got: %s", xml)
	}
	if contains(xml, "latest_at=") {
		t.Errorf("should not have latest_at when nil, got: %s", xml)
	}
}

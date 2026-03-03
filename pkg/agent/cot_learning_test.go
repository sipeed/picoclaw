package agent

import (
	"strings"
	"testing"
)

func TestCotUsage_RecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// Record some usage with tags.
	id1, err := ms.RecordCotUsage("code", []string{"golang", "testing"}, "1. Check tests\n2. Write code", "How do I test Go code?")
	if err != nil {
		t.Fatal(err)
	}
	if id1 <= 0 {
		t.Errorf("expected positive ID, got %d", id1)
	}

	id2, err := ms.RecordCotUsage("question", []string{"golang"}, "1. Compare options\n2. Decide", "What's the difference?")
	if err != nil {
		t.Fatal(err)
	}

	id3, err := ms.RecordCotUsage("code", []string{"http", "golang"}, "1. Define routes\n2. Implement handlers", "Write a HTTP server")
	if err != nil {
		t.Fatal(err)
	}

	// Query recent usage.
	records, err := ms.GetRecentCotUsage(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Most recent first.
	if records[0].ID != id3 {
		t.Errorf("expected most recent to be id3=%d, got %d", id3, records[0].ID)
	}

	// Check tags are stored correctly.
	if len(records[0].Tags) != 2 || records[0].Tags[0] != "http" {
		t.Errorf("tags = %v, want [http, golang]", records[0].Tags)
	}

	// Check cot_prompt is stored.
	if !strings.Contains(records[0].CotPrompt, "Define routes") {
		t.Errorf("cot_prompt = %q, should contain 'Define routes'", records[0].CotPrompt)
	}

	_ = id2 // used above
}

func TestCotUsage_Feedback(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	id, _ := ms.RecordCotUsage("code", []string{"golang"}, "think step by step", "test message")

	// Initial feedback should be 0.
	records, _ := ms.GetRecentCotUsage(1)
	if records[0].Feedback != 0 {
		t.Errorf("initial feedback = %d, want 0", records[0].Feedback)
	}

	// Update feedback.
	err := ms.UpdateCotFeedback(id, 1)
	if err != nil {
		t.Fatal(err)
	}

	records, _ = ms.GetRecentCotUsage(1)
	if records[0].Feedback != 1 {
		t.Errorf("feedback = %d, want 1", records[0].Feedback)
	}

	// Invalid score.
	err = ms.UpdateCotFeedback(id, 5)
	if err == nil {
		t.Error("expected error for invalid score 5")
	}
}

func TestCotUsage_UpdateLatestFeedback(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	ms.RecordCotUsage("code", nil, "strategy 1", "first")
	ms.RecordCotUsage("debug", nil, "strategy 2", "second")

	// Update latest (should be "debug").
	err := ms.UpdateLatestCotFeedback(-1)
	if err != nil {
		t.Fatal(err)
	}

	records, _ := ms.GetRecentCotUsage(2)
	if records[0].Intent != "debug" || records[0].Feedback != -1 {
		t.Errorf("latest: intent=%q feedback=%d, want debug/-1", records[0].Intent, records[0].Feedback)
	}
	if records[1].Intent != "code" || records[1].Feedback != 0 {
		t.Errorf("first: intent=%q feedback=%d, want code/0", records[1].Intent, records[1].Feedback)
	}
}

func TestCotUsage_Stats(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	id1, _ := ms.RecordCotUsage("code", nil, "think about code", "write code")
	ms.UpdateCotFeedback(id1, 1)

	id2, _ := ms.RecordCotUsage("code", nil, "debug systematically", "fix bug")
	ms.UpdateCotFeedback(id2, 1)

	id3, _ := ms.RecordCotUsage("question", nil, "analyse step by step", "why does X happen?")
	ms.UpdateCotFeedback(id3, -1)

	id4, _ := ms.RecordCotUsage("chat", nil, "", "hello")
	ms.UpdateCotFeedback(id4, 1)

	// Get stats.
	stats, err := ms.GetCotStats(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 3 {
		t.Errorf("expected 3 intent stats, got %d", len(stats))
	}

	// "code" should have highest total uses.
	if stats[0].Intent != "code" || stats[0].TotalUses != 2 {
		t.Errorf("expected code with 2 uses, got %q with %d", stats[0].Intent, stats[0].TotalUses)
	}
}

func TestCotUsage_TopRatedPrompts(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// Record with different tags and feedback.
	id1, _ := ms.RecordCotUsage("code", []string{"golang", "testing"}, "1. Write test first\n2. Then implement", "write Go test")
	ms.UpdateCotFeedback(id1, 1)

	id2, _ := ms.RecordCotUsage("code", []string{"python"}, "1. Use pytest\n2. Mock dependencies", "write Python test")
	ms.UpdateCotFeedback(id2, 1)

	id3, _ := ms.RecordCotUsage("debug", []string{"golang"}, "1. Reproduce\n2. Hypothesize", "fix Go bug")
	ms.UpdateCotFeedback(id3, 1)

	id4, _ := ms.RecordCotUsage("code", []string{"golang"}, "1. Bad strategy", "bad approach")
	ms.UpdateCotFeedback(id4, -1) // Negative — should not appear.

	// Without tag filter.
	top, err := ms.GetTopRatedCotPrompts(30, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 3 {
		t.Errorf("expected 3 top-rated, got %d", len(top))
	}

	// With tag filter — "golang" should prioritise golang-tagged prompts.
	top, err = ms.GetTopRatedCotPrompts(30, 2, []string{"golang"})
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 {
		t.Errorf("expected 2, got %d", len(top))
	}
	// First result should have golang tag.
	hasGolang := false
	for _, tag := range top[0].Tags {
		if tag == "golang" {
			hasGolang = true
		}
	}
	if !hasGolang {
		t.Errorf("first result should have golang tag, got %v", top[0].Tags)
	}
}

func TestCotUsage_FormatLearningContext(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// Empty — should return empty string.
	ctx := ms.FormatCotLearningContext(30, nil)
	if ctx != "" {
		t.Errorf("expected empty learning context, got %q", ctx)
	}

	// Add some usage with feedback.
	id1, _ := ms.RecordCotUsage("code", []string{"golang"}, "1. Understand requirements\n2. Write code", "write code")
	ms.UpdateCotFeedback(id1, 1)

	id2, _ := ms.RecordCotUsage("question", []string{"architecture"}, "1. Examine structure\n2. Explain", "why does X happen?")
	ms.UpdateCotFeedback(id2, 1)

	ctx = ms.FormatCotLearningContext(30, nil)
	if ctx == "" {
		t.Error("expected non-empty learning context after recording usage")
	}
	if !strings.Contains(ctx, "Historical Usage Stats") {
		t.Error("missing stats header")
	}
	if !strings.Contains(ctx, "Proven Strategies") {
		t.Error("missing proven strategies section")
	}
	if !strings.Contains(ctx, "golang") {
		t.Error("should show tags in proven examples")
	}
}

func TestCotUsage_MessageTruncation(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	longMsg := strings.Repeat("x", 500)
	_, err := ms.RecordCotUsage("code", nil, "strategy", longMsg)
	if err != nil {
		t.Fatal(err)
	}

	records, _ := ms.GetRecentCotUsage(1)
	if len(records[0].Message) > 200 {
		t.Errorf("message should be truncated to 200 chars, got %d", len(records[0].Message))
	}
}

func TestPreLLM_LearningIntegration(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	cotReg := NewCotRegistry(dir)
	mp := &mockLLMProvider{
		response: `{"intent":"code","tags":["golang"],"cot_prompt":"1. Understand the function signature\n2. Write the implementation\n3. Add error handling"}`,
	}
	p := NewAnalyser(mp, "test-model", cotReg)

	// First call — no learning data yet.
	result := p.Analyse(nil, "write a function", ms, nil)
	if result.CotPrompt == "" {
		t.Error("expected non-empty CotPrompt")
	}

	// Verify usage was recorded with tags.
	records, _ := ms.GetRecentCotUsage(5)
	if len(records) != 1 {
		t.Fatalf("expected 1 usage record, got %d", len(records))
	}
	if records[0].Intent != "code" {
		t.Errorf("recorded intent = %q, want %q", records[0].Intent, "code")
	}
	if len(records[0].Tags) != 1 || records[0].Tags[0] != "golang" {
		t.Errorf("recorded tags = %v, want [golang]", records[0].Tags)
	}
	if records[0].CotPrompt == "" {
		t.Error("recorded cot_prompt should not be empty")
	}

	// Provide positive feedback.
	ms.UpdateLatestCotFeedback(1)

	// Second call — learning context should now be included.
	result2 := p.Analyse(nil, "fix this bug", ms, nil)
	if result2.CotPrompt == "" {
		t.Error("expected non-empty CotPrompt on second call")
	}

	// Should now have 2 usage records.
	records, _ = ms.GetRecentCotUsage(5)
	if len(records) != 2 {
		t.Errorf("expected 2 usage records, got %d", len(records))
	}

	// Learning context should include the first proven strategy.
	ctx := ms.FormatCotLearningContext(30, []string{"golang"})
	if ctx == "" {
		t.Error("expected non-empty learning context after usage + feedback")
	}
	if !strings.Contains(ctx, "Proven Strategies") {
		t.Error("learning context should include proven strategies")
	}
}

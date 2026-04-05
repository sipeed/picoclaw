package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func makeMsg(role, content string) providers.Message {
	return providers.Message{Role: role, Content: content}
}

func makeTool(callID, content string) providers.Message {
	return providers.Message{Role: "tool", Content: content, ToolCallID: callID}
}

func makeAssistantWithTool(content, toolID, toolName string) providers.Message {
	return providers.Message{
		Role:    "assistant",
		Content: content,
		ToolCalls: []providers.ToolCall{
			{ID: toolID, Name: toolName},
		},
	}
}

func TestNewContextCompressor(t *testing.T) {
	cc := NewContextCompressor(100000)
	if cc.contextLength != 100000 {
		t.Errorf("contextLength = %d, want 100000", cc.contextLength)
	}
	if cc.thresholdTokens != 50000 {
		t.Errorf("thresholdTokens = %d, want 50000", cc.thresholdTokens)
	}
}

func TestNewContextCompressorWithOptions(t *testing.T) {
	cc := NewContextCompressor(100000,
		WithThresholdPercent(75),
		WithProtectFirstN(5),
		WithProtectLastN(10),
	)
	if cc.thresholdTokens != 75000 {
		t.Errorf("thresholdTokens = %d, want 75000", cc.thresholdTokens)
	}
	if cc.protectFirstN != 5 {
		t.Errorf("protectFirstN = %d, want 5", cc.protectFirstN)
	}
	if cc.protectLastN != 10 {
		t.Errorf("protectLastN = %d, want 10", cc.protectLastN)
	}
}

func TestShouldCompress(t *testing.T) {
	cc := NewContextCompressor(100000) // threshold = 50000

	if cc.ShouldCompress(30000) {
		t.Error("30K should not trigger compression")
	}
	if !cc.ShouldCompress(60000) {
		t.Error("60K should trigger compression")
	}
	if !cc.ShouldCompress(50000) {
		t.Error("exactly 50K should trigger compression")
	}
}

func TestUpdateFromResponse(t *testing.T) {
	cc := NewContextCompressor(100000)
	cc.UpdateFromResponse(&providers.UsageInfo{PromptTokens: 42000})

	status := cc.GetStatus()
	if status["last_prompt_tokens"] != 42000 {
		t.Errorf("last_prompt_tokens = %v, want 42000", status["last_prompt_tokens"])
	}
}

func TestCompressSmallHistory(t *testing.T) {
	cc := NewContextCompressor(100000, WithProtectFirstN(2), WithProtectLastN(2))

	msgs := []providers.Message{
		makeMsg("system", "You are a helpful assistant"),
		makeMsg("user", "Hello"),
		makeMsg("assistant", "Hi!"),
	}

	compressed, summary := cc.Compress(msgs)
	if summary != "" {
		t.Error("small history should not generate summary")
	}
	if len(compressed) != len(msgs) {
		t.Errorf("compressed = %d, want %d (no change)", len(compressed), len(msgs))
	}
}

func TestCompressLargeHistory(t *testing.T) {
	cc := NewContextCompressor(100000, WithProtectFirstN(2), WithProtectLastN(3))

	var msgs []providers.Message
	msgs = append(msgs, makeMsg("system", "System prompt"))
	msgs = append(msgs, makeMsg("user", "Setup message"))
	// Add 20 middle messages
	for i := 0; i < 20; i++ {
		msgs = append(msgs, makeMsg("user", strings.Repeat("question ", 100)))
		msgs = append(msgs, makeMsg("assistant", strings.Repeat("answer ", 100)))
	}
	// Last 3
	msgs = append(msgs, makeMsg("user", "Recent question"))
	msgs = append(msgs, makeMsg("assistant", "Recent answer"))
	msgs = append(msgs, makeMsg("user", "Latest"))

	compressed, summary := cc.Compress(msgs)

	if summary == "" {
		t.Error("large history should generate summary input")
	}
	if len(compressed) >= len(msgs) {
		t.Errorf("compressed (%d) should be smaller than original (%d)", len(compressed), len(msgs))
	}
	// Head preserved
	if compressed[0].Content != "System prompt" {
		t.Error("head not preserved")
	}
	// Tail preserved
	if compressed[len(compressed)-1].Content != "Latest" {
		t.Error("tail not preserved")
	}
	// Compression notice present
	found := false
	for _, m := range compressed {
		if strings.Contains(m.Content, "Context compressed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compression notice not found")
	}
}

func TestPruneOldToolResults(t *testing.T) {
	cc := NewContextCompressor(100000)

	longResult := strings.Repeat("x", 500) // > maxPrunedContentLen
	msgs := []providers.Message{
		makeMsg("user", "question"),
		makeAssistantWithTool("calling tool", "tc1", "exec"),
		makeTool("tc1", longResult),
		makeMsg("user", "recent question"),
		makeAssistantWithTool("calling again", "tc2", "exec"),
		makeTool("tc2", longResult),
	}

	pruned, count := cc.pruneOldToolResults(msgs, 3)

	// First tool result should be pruned (outside tail protection)
	if count != 1 {
		t.Errorf("pruned count = %d, want 1", count)
	}
	if !strings.Contains(pruned[2].Content, "truncated") {
		t.Error("old tool result should be truncated")
	}
	// Last tool result should be preserved (within tail protection)
	if strings.Contains(pruned[5].Content, "truncated") {
		t.Error("recent tool result should NOT be truncated")
	}
}

func TestSanitizeToolPairsOrphanResult(t *testing.T) {
	cc := NewContextCompressor(100000)

	msgs := []providers.Message{
		makeMsg("user", "question"),
		// Tool result without matching assistant call (orphan)
		makeTool("tc-gone", "some result"),
		makeMsg("assistant", "answer"),
	}

	sanitized := cc.sanitizeToolPairs(msgs)

	for _, m := range sanitized {
		if m.Role == "tool" && m.ToolCallID == "tc-gone" {
			t.Error("orphan tool result should be removed")
		}
	}
}

func TestSanitizeToolPairsOrphanCall(t *testing.T) {
	cc := NewContextCompressor(100000)

	msgs := []providers.Message{
		makeMsg("user", "question"),
		makeAssistantWithTool("calling", "tc1", "exec"),
		// No tool result for tc1
		makeMsg("user", "next question"),
	}

	sanitized := cc.sanitizeToolPairs(msgs)

	// Should add stub result
	foundStub := false
	for _, m := range sanitized {
		if m.Role == "tool" && m.ToolCallID == "tc1" {
			foundStub = true
			if !strings.Contains(m.Content, "earlier conversation") {
				t.Error("stub should reference earlier conversation")
			}
		}
	}
	if !foundStub {
		t.Error("expected stub for orphan tool call")
	}
}

func TestIterativeSummary(t *testing.T) {
	cc := NewContextCompressor(100000, WithProtectFirstN(1), WithProtectLastN(2))

	var msgs []providers.Message
	msgs = append(msgs, makeMsg("system", "System"))
	for i := 0; i < 30; i++ {
		msgs = append(msgs, makeMsg("user", strings.Repeat("q ", 50)))
		msgs = append(msgs, makeMsg("assistant", strings.Repeat("a ", 50)))
	}
	msgs = append(msgs, makeMsg("user", "last"))
	msgs = append(msgs, makeMsg("assistant", "final"))

	// First compression
	_, summary1 := cc.Compress(msgs)
	if !strings.Contains(summary1, "structured handoff summary") {
		t.Error("first compression should request new summary")
	}

	// Simulate LLM generating summary
	cc.SetPreviousSummary("## Goal\nHelp user\n## Progress\n### Done\n- Step 1")

	// Second compression
	_, summary2 := cc.Compress(msgs)
	if !strings.Contains(summary2, "UPDATE the previous summary") {
		t.Error("second compression should request update, not new summary")
	}
	if !strings.Contains(summary2, "Help user") {
		t.Error("second compression should include previous summary")
	}
}

func TestSerializeForSummary(t *testing.T) {
	cc := NewContextCompressor(100000)

	msgs := []providers.Message{
		makeMsg("user", "Hello"),
		makeAssistantWithTool("Let me check", "tc1", "exec"),
		makeTool("tc1", "result"),
	}

	result := cc.serializeForSummary(msgs)
	if !strings.Contains(result, "[USER]: Hello") {
		t.Error("should contain user message")
	}
	if !strings.Contains(result, "→ tool: exec") {
		t.Error("should contain tool call name")
	}
}

func TestEstimateTokens(t *testing.T) {
	msg := makeMsg("user", strings.Repeat("a", 400))
	tokens := estimateTokens(msg)
	if tokens != 100 { // 400 chars / 4 chars per token
		t.Errorf("tokens = %d, want 100", tokens)
	}
}

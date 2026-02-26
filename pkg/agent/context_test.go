package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func msg(role, content string) providers.Message {
	return providers.Message{Role: role, Content: content}
}

func assistantWithTools(toolIDs ...string) providers.Message {
	calls := make([]providers.ToolCall, len(toolIDs))
	for i, id := range toolIDs {
		calls[i] = providers.ToolCall{ID: id, Type: "function"}
	}
	return providers.Message{Role: "assistant", ToolCalls: calls}
}

func toolResult(id string) providers.Message {
	return providers.Message{Role: "tool", Content: "result", ToolCallID: id}
}

func TestSanitizeHistoryForProvider_EmptyHistory(t *testing.T) {
	result := sanitizeHistoryForProvider(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}

	result = sanitizeHistoryForProvider([]providers.Message{})
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}
}

func TestSanitizeHistoryForProvider_SingleToolCall(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		assistantWithTools("A"),
		toolResult("A"),
		msg("assistant", "done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_MultiToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "both done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 5 {
		t.Fatalf("expected 5 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAfterPlainAssistant(t *testing.T) {
	history := []providers.Message{
		msg("user", "hi"),
		msg("assistant", "thinking"),
		assistantWithTools("A"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_OrphanedLeadingTool(t *testing.T) {
	history := []providers.Message{
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterUserDropped(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterAssistantNoToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAtStart(t *testing.T) {
	history := []providers.Message{
		assistantWithTools("A"),
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_MultiToolCallsThenNewRound(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "done"),
		msg("user", "hi"),
		assistantWithTools("C"),
		toolResult("C"),
		msg("assistant", "done again"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 9 {
		t.Fatalf("expected 9 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_ConsecutiveMultiToolRounds(t *testing.T) {
	history := []providers.Message{
		msg("user", "start"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		assistantWithTools("C", "D"),
		toolResult("C"),
		toolResult("D"),
		msg("assistant", "all done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 8 {
		t.Fatalf("expected 8 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_PlainConversation(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		msg("user", "how are you"),
		msg("assistant", "fine"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "user", "assistant")
}

func roles(msgs []providers.Message) []string {
	r := make([]string, len(msgs))
	for i, m := range msgs {
		r[i] = m.Role
	}
	return r
}

func assertRoles(t *testing.T, msgs []providers.Message, expected ...string) {
	t.Helper()
	if len(msgs) != len(expected) {
		t.Fatalf("role count mismatch: got %v, want %v", roles(msgs), expected)
	}
	for i, exp := range expected {
		if msgs[i].Role != exp {
			t.Errorf("message[%d]: got role %q, want %q", i, msgs[i].Role, exp)
		}
	}
}

func TestContextBuilder_BuildMessagesWithOptions_FullStrategy(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyFull,
		IncludeMemory:  true,
		IncludeRuntime: true,
	}

	messages := cb.BuildMessagesWithOptions(history, "", "test message", nil, "telegram", "chat123", opts)

	// Should have system message + history + current message
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	// First message should be system
	if messages[0].Role != "system" {
		t.Errorf("expected first message to be system, got %s", messages[0].Role)
	}

	// System message should contain identity (picoclaw header)
	if !containsString(messages[0].Content, "# picoclaw") {
		t.Error("expected system message to contain picoclaw identity")
	}

	// Check history is preserved
	assertRoles(t, messages[1:], "user", "assistant", "user")
}

func TestContextBuilder_BuildMessagesWithOptions_LiteStrategy(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "quick question"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyLite,
		IncludeMemory:  false,
		IncludeRuntime: true,
	}

	messages := cb.BuildMessagesWithOptions(history, "", "what time is it?", nil, "telegram", "chat123", opts)

	// Should have system message + history + current message
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// System message should be minimal (lite)
	systemContent := messages[0].Content
	if containsString(systemContent, "<skills>") {
		t.Error("lite strategy should not include skills")
	}

	// But should still have identity
	if !containsString(systemContent, "# picoclaw") {
		t.Error("lite strategy should still include identity")
	}
}

func TestContextBuilder_BuildMessagesWithOptions_CustomStrategy(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "help me with code"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyCustom,
		IncludeSkills:  []string{"test-skill"},
		ExcludeSkills:  []string{},
		IncludeTools:   []string{},
		ExcludeTools:   []string{},
		IncludeMemory:  true,
		IncludeRuntime: true,
	}

	messages := cb.BuildMessagesWithOptions(history, "", "show me the code", nil, "slack", "channel456", opts)

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// System message should use custom strategy
	systemContent := messages[0].Content
	if !containsString(systemContent, "# picoclaw") {
		t.Error("custom strategy should include identity")
	}
}

func TestContextBuilder_BuildMessagesWithOptions_EmptyHistory(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	opts := ContextBuildOptions{
		Strategy: ContextStrategyFull,
	}

	messages := cb.BuildMessagesWithOptions([]providers.Message{}, "", "first message", nil, "discord", "server789", opts)

	// Should have system message + current message only
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	assertRoles(t, messages, "system", "user")
}

func TestContextBuilder_BuildMessagesWithOptions_Summary(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "earlier we discussed"},
	}

	summary := "User asked about Go concurrency patterns and error handling."

	opts := ContextBuildOptions{
		Strategy: ContextStrategyFull,
	}

	messages := cb.BuildMessagesWithOptions(history, summary, "can you elaborate?", nil, "telegram", "chat123", opts)

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// System message should contain summary
	if !containsString(messages[0].Content, "CONTEXT_SUMMARY") {
		t.Error("expected system message to contain CONTEXT_SUMMARY")
	}

	if !containsString(messages[0].Content, summary) {
		t.Error("expected system message to contain the summary text")
	}
}

func TestContextBuilder_BuildMessagesWithOptions_Media(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	opts := ContextBuildOptions{
		Strategy: ContextStrategyFull,
	}

	media := []string{"image.jpg", "document.pdf"}

	messages := cb.BuildMessagesWithOptions([]providers.Message{}, "", "check these files", media, "wecom", "user999", opts)

	// Media handling might be implemented in future
	// For now, just ensure it doesn't crash
	if len(messages) == 0 {
		t.Fatal("expected at least one message")
	}
}

func TestContextBuilder_BuildMessagesWithOptions_DefaultValues(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	// Test with zero-value options - should use defaults
	opts := ContextBuildOptions{}

	messages := cb.BuildMessagesWithOptions(
		[]providers.Message{{Role: "user", Content: "test"}},
		"",
		"message",
		nil,
		"telegram",
		"chat123",
		opts,
	)

	// Should default to Full strategy with memory and runtime enabled
	if len(messages) == 0 {
		t.Fatal("expected messages to be returned")
	}

	if messages[0].Role != "system" {
		t.Error("expected first message to be system")
	}
}

func TestContextBuilder_BuildMessagesWithOptions_ChannelSpecific(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	channels := []string{"telegram", "discord", "slack", "wecom", "feishu"}

	for _, channel := range channels {
		t.Run(channel, func(t *testing.T) {
			opts := ContextBuildOptions{
				Strategy: ContextStrategyFull,
			}

			messages := cb.BuildMessagesWithOptions(
				[]providers.Message{},
				"",
				"test",
				nil,
				channel,
				"chat-"+channel,
				opts,
			)

			if len(messages) != 2 {
				t.Errorf("expected 2 messages for %s, got %d", channel, len(messages))
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

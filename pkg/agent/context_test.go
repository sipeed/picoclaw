package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func msg(role, content string) providers.Message {
	return providers.Message{Role: role, Content: content}
}

func TestBuildMessages_DynamicWorkspace(t *testing.T) {
	root := t.TempDir()
	cb := NewContextBuilder(root)

	// Test host path
	msgs := cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", root, SandboxInfo{IsHost: true})
	if len(msgs) == 0 || msgs[0].Role != "system" {
		t.Fatal("expected system message")
	}
	if !strings.Contains(msgs[0].Content, "Your workspace is at: "+root) {
		t.Errorf("system prompt missing host workspace path: %s", msgs[0].Content)
	}

	// Test container path
	containerPath := "/workspace"
	msgs = cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", containerPath, SandboxInfo{IsHost: false})
	if !strings.Contains(msgs[0].Content, "Your workspace is at: "+containerPath) {
		t.Errorf("system prompt missing container workspace path: %s", msgs[0].Content)
	}
}

func TestBuildMessages_SandboxInfo(t *testing.T) {
	root := t.TempDir()
	cb := NewContextBuilder(root)

	// Test with sandbox enabled (container)
	sb := SandboxInfo{
		IsHost: false,
	}
	msgs := cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", "/workspace", sb)
	content := msgs[0].Content

	if !strings.Contains(content, "## Sandbox") {
		t.Error("expected ## Sandbox section in prompt")
	}
	if !strings.Contains(content, "Docker container") {
		t.Error("expected 'Docker container' description in prompt")
	}
	if !strings.Contains(content, "ALWAYS prefer relative paths") {
		t.Error("expected relative path guidance in prompt")
	}

	// Test with sandbox disabled (host)
	sb = SandboxInfo{
		IsHost: true,
	}
	msgs = cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", root, sb)
	content = msgs[0].Content
	if strings.Contains(content, "## Sandbox") {
		t.Error("did not expect ## Sandbox section in prompt when disabled")
	}
}

func TestBuildMessages_CacheIntegrity_SandboxToggle(t *testing.T) {
	root := t.TempDir()
	cb := NewContextBuilder(root)

	// 1. Call with Host - should populate cache without sandbox guidance
	msgs1 := cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", root, SandboxInfo{IsHost: true})
	if strings.Contains(msgs1[0].Content, "## Sandbox") {
		t.Fatal("First call (Host) should not have sandbox guidance")
	}

	// 2. Call with Container - should use SAME cache but inject guidance
	msgs2 := cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", "/workspace", SandboxInfo{IsHost: false})
	if !strings.Contains(msgs2[0].Content, "## Sandbox") {
		t.Fatal("Second call (Container) MUST have sandbox guidance via dynamic injection")
	}

	// 3. Call with Host again - should still be correct (no guidance)
	msgs3 := cb.BuildMessages(nil, "", "hi", nil, "cli", "chat1", root, SandboxInfo{IsHost: true})
	if strings.Contains(msgs3[0].Content, "## Sandbox") {
		t.Fatal("Third call (Host) should not have sandbox guidance (cache must remain clean)")
	}
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

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeHistoryForProvider_KeepMultipleToolOutputsFromOneAssistantTurn(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "check two files"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Name: "read_file"},
				{ID: "call_2", Name: "read_file"},
			},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "file a"},
		{Role: "tool", ToolCallID: "call_2", Content: "file b"},
	}

	got := sanitizeHistoryForProvider(history)

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4; got=%#v", len(got), got)
	}
	if got[2].Role != "tool" || got[2].ToolCallID != "call_1" {
		t.Fatalf("got[2] = %#v, want tool output for call_1", got[2])
	}
	if got[3].Role != "tool" || got[3].ToolCallID != "call_2" {
		t.Fatalf("got[3] = %#v, want tool output for call_2", got[3])
	}
}

func TestSanitizeHistoryForProvider_DropToolOutputWithUnknownCallID(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "check file"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Name: "read_file"},
			},
		},
		{Role: "tool", ToolCallID: "call_999", Content: "orphan"},
	}

	got := sanitizeHistoryForProvider(history)

	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3; got=%#v", len(got), got)
	}
	if got[2].Role != "tool" || got[2].ToolCallID != "call_1" {
		t.Fatalf("expected synthesized placeholder for call_1, got=%#v", got[2])
	}
}

func TestPruneHistoryForContext_ToolResultCondensed(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	cb.SetRuntimeSettings(ContextRuntimeSettings{
		ContextWindowTokens:      100,
		PruningMode:              "tools_only",
		SoftToolResultChars:      80,
		HardToolResultChars:      30,
		TriggerRatio:             0.1,
		BootstrapSnapshotEnabled: false,
	})

	history := []providers.Message{
		{Role: "user", Content: "run command"},
		{Role: "tool", Content: strings.Repeat("x", 600)},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: "next"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "continue"},
		{Role: "assistant", Content: "ready"},
		{Role: "user", Content: "go"},
		{Role: "assistant", Content: "working"},
		{Role: "user", Content: "status"},
	}

	pruned := cb.pruneHistoryForContext(history, strings.Repeat("S", 500))
	if len(pruned) != len(history) {
		t.Fatalf("len(pruned) = %d, want %d", len(pruned), len(history))
	}
	if !strings.Contains(pruned[1].Content, "tool result") {
		t.Fatalf("expected tool result to be condensed/omitted, got: %q", pruned[1].Content)
	}
}

func TestCompactOldChitChat_CondensesRun(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "ok"},
		{Role: "assistant", Content: "thanks"},
		{Role: "user", Content: "received"},
		{Role: "assistant", Content: "actual content"},
		{Role: "user", Content: "keep recent"},
	}

	got := compactOldChitChat(history, 4)
	if len(got) >= len(history) {
		t.Fatalf("expected condensed history, got len=%d", len(got))
	}
	if !strings.Contains(strings.ToLower(got[0].Content), "condensed") {
		t.Fatalf("expected condensed marker, got: %q", got[0].Content)
	}
}

func TestBuildMessagesForSession_IncludesRetrievedMemory(t *testing.T) {
	workspace := t.TempDir()
	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}

	memoryContent := `# MEMORY

## Long-term Facts
- Preferred editor is Neovim
`
	if err := os.WriteFile(filepath.Join(memoryDir, "MEMORY.md"), []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("write memory file: %v", err)
	}

	cb := NewContextBuilder(workspace)
	cb.SetRuntimeSettings(ContextRuntimeSettings{
		ContextWindowTokens:      4096,
		PruningMode:              "off",
		BootstrapSnapshotEnabled: false,
		MemoryVectorEnabled:      true,
		MemoryVectorDimensions:   128,
		MemoryVectorTopK:         3,
		MemoryVectorMinScore:     0.01,
		MemoryVectorMaxChars:     800,
		MemoryVectorRecentDays:   7,
	})

	messages := cb.BuildMessagesForSession(
		"sess-1",
		nil,
		"",
		"Which editor do I usually prefer?",
		nil,
		"",
		"",
	)
	if len(messages) == 0 {
		t.Fatalf("expected at least one message")
	}

	system := strings.ToLower(messages[0].Content)
	if !strings.Contains(system, "retrieved memory") {
		t.Fatalf("expected retrieved memory section in system prompt, got:\n%s", messages[0].Content)
	}
	if !strings.Contains(system, "neovim") {
		t.Fatalf("expected retrieved semantic hit to mention neovim, got:\n%s", messages[0].Content)
	}
}

func TestSanitizeHistoryForProvider_SynthesizesMissingToolOutputs(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "check file"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Name: "read_file"},
				{ID: "call_2", Name: "list_dir"},
			},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "ok"},
		{Role: "assistant", Content: "continuing"},
	}

	got := sanitizeHistoryForProvider(history)
	if len(got) != 5 {
		t.Fatalf("len(got) = %d, want 5; got=%#v", len(got), got)
	}
	if got[3].Role != "tool" || got[3].ToolCallID != "call_2" {
		t.Fatalf("expected synthesized tool output for call_2 at index 3, got %#v", got[3])
	}
	if !strings.Contains(strings.ToLower(got[3].Content), "synthesized") {
		t.Fatalf("expected synthesized marker, got %q", got[3].Content)
	}
}

func TestPruneHistoryForContext_DoesNotPanicAfterChitChatCompaction(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	cb.SetRuntimeSettings(ContextRuntimeSettings{
		ContextWindowTokens:      100,
		PruningMode:              "tools_only",
		IncludeOldChitChat:       true,
		SoftToolResultChars:      80,
		HardToolResultChars:      30,
		TriggerRatio:             0.2,
		BootstrapSnapshotEnabled: false,
	})

	history := []providers.Message{
		{Role: "user", Content: "ok"},
		{Role: "assistant", Content: "thanks"},
		{Role: "user", Content: "ok"},
		{Role: "assistant", Content: "thanks"},
		{Role: "user", Content: "ok"},
		{Role: "assistant", Content: "thanks"},
		{Role: "tool", Content: strings.Repeat("x", 600)},
		{Role: "assistant", Content: "ready"},
		{Role: "user", Content: "go"},
	}

	_ = cb.pruneHistoryForContext(history, strings.Repeat("S", 500))
}

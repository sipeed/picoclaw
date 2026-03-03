package agent

import (
	"os"
	"strings"
	"testing"
)

// --- Slash command tests ----------------------------------------------------

func TestRuntime_MemoryCommand(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()
	r := NewReflector(nil, "")

	// /memory with no args → help.
	resp, ok := r.HandleCommand("/memory", ms)
	if !ok {
		t.Fatal("expected /memory to be handled")
	}
	if !strings.Contains(resp, "Usage") {
		t.Error("expected usage text")
	}

	// /memory list → empty.
	resp, ok = r.HandleCommand("/memory list", ms)
	if !ok {
		t.Fatal("expected /memory list to be handled")
	}
	if !strings.Contains(resp, "No memories") {
		t.Errorf("expected empty list, got %q", resp)
	}

	// /memory add.
	resp, ok = r.HandleCommand("/memory add Go is great for concurrency #golang #concurrency", ms)
	if !ok {
		t.Fatal("expected /memory add to be handled")
	}
	if !strings.Contains(resp, "✅") {
		t.Errorf("expected success, got %q", resp)
	}
	if !strings.Contains(resp, "golang") {
		t.Errorf("should show tags, got %q", resp)
	}

	// /memory list → should have 1 entry.
	resp, _ = r.HandleCommand("/memory list", ms)
	if !strings.Contains(resp, "Go is great") {
		t.Errorf("should show entry, got %q", resp)
	}

	// /memory search.
	resp, _ = r.HandleCommand("/memory search golang", ms)
	if !strings.Contains(resp, "Found 1") {
		t.Errorf("expected 1 result, got %q", resp)
	}

	resp, _ = r.HandleCommand("/memory search nonexistent", ms)
	if !strings.Contains(resp, "No memories found") {
		t.Errorf("expected no results, got %q", resp)
	}

	// /memory stats — should show entry count.
	resp, _ = r.HandleCommand("/memory stats", ms)
	if !strings.Contains(resp, "Stats") {
		t.Errorf("expected stats, got %q", resp)
	}
	if !strings.Contains(resp, "Total entries: 1") {
		t.Errorf("expected 1 entry in stats, got %q", resp)
	}

	// /memory edit.
	resp, _ = r.HandleCommand("/memory edit 1 Updated content #go", ms)
	if !strings.Contains(resp, "✅") {
		t.Errorf("expected success, got %q", resp)
	}
	resp, _ = r.HandleCommand("/memory list", ms)
	if !strings.Contains(resp, "Updated content") {
		t.Errorf("edit should be reflected, got %q", resp)
	}

	// /memory delete.
	resp, _ = r.HandleCommand("/memory delete 1", ms)
	if !strings.Contains(resp, "✅") {
		t.Errorf("expected success, got %q", resp)
	}
	resp, _ = r.HandleCommand("/memory list", ms)
	if !strings.Contains(resp, "No memories") {
		t.Errorf("expected empty after delete, got %q", resp)
	}
}

func TestRuntime_HelpCommand(t *testing.T) {
	r := NewReflector(nil, "")
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	resp, ok := r.HandleCommand("/help", ms)
	if !ok {
		t.Fatal("expected /help to be handled")
	}
	if !strings.Contains(resp, "/memory") {
		t.Error("help should list /memory")
	}
	if !strings.Contains(resp, "/cot") {
		t.Error("help should list /cot")
	}
	if !strings.Contains(resp, "/show") {
		t.Error("help should list /show (now a runtime command)")
	}
	if !strings.Contains(resp, "/shell") {
		t.Error("help should list /shell")
	}
}

func TestRuntime_ShellSecurity(t *testing.T) {
	r := NewReflector(nil, "")
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// Unknown command (not builtin or dev tool).
	resp, _ := r.HandleCommand("/shell rm -rf /", ms)
	if !strings.Contains(resp, "Unknown command") {
		t.Errorf("rm should be unknown, got %q", resp)
	}

	// Unknown: sudo
	resp, _ = r.HandleCommand("/shell sudo ls", ms)
	if !strings.Contains(resp, "Unknown command") {
		t.Errorf("sudo should be unknown, got %q", resp)
	}

	// Injection via passthrough: git | bash
	resp, _ = r.HandleCommand("/shell git log | bash", ms)
	if !strings.Contains(resp, "blocked") {
		t.Errorf("injection should be blocked, got %q", resp)
	}

	// Builtin echo works (cross-platform).
	resp, _ = r.HandleCommand("/shell echo hello world", ms)
	if !strings.Contains(resp, "hello world") {
		t.Errorf("echo should work, got %q", resp)
	}
}

func TestRuntime_CotCommand(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()
	r := NewReflector(nil, "")

	// /cot with no args → help.
	resp, ok := r.HandleCommand("/cot", ms)
	if !ok {
		t.Fatal("expected /cot to be handled")
	}
	if !strings.Contains(resp, "Usage") {
		t.Error("expected usage text")
	}

	// /cot stats → empty.
	resp, _ = r.HandleCommand("/cot stats", ms)
	if !strings.Contains(resp, "No CoT usage") {
		t.Errorf("expected empty, got %q", resp)
	}

	// Add some usage first.
	ms.RecordCotUsage("code", []string{"golang"}, "1. Think\n2. Code", "write code")

	// /cot history.
	resp, _ = r.HandleCommand("/cot history", ms)
	if !strings.Contains(resp, "code") {
		t.Errorf("expected history entry, got %q", resp)
	}

	// /cot feedback.
	resp, _ = r.HandleCommand("/cot feedback 1", ms)
	if !strings.Contains(resp, "✅") {
		t.Errorf("expected success, got %q", resp)
	}

	// /cot feedback bad input.
	resp, _ = r.HandleCommand("/cot feedback 99", ms)
	if !strings.Contains(resp, "❌") {
		t.Errorf("expected error, got %q", resp)
	}
}

func TestRuntime_RuntimeCommand(t *testing.T) {
	r := NewReflector(nil, "")
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	resp, ok := r.HandleCommand("/runtime status", ms)
	if !ok {
		t.Fatal("expected /runtime to be handled")
	}
	if !strings.Contains(resp, "Processors") {
		t.Errorf("expected status, got %q", resp)
	}

	resp, _ = r.HandleCommand("/runtime processors", ms)
	if !strings.Contains(resp, "error_tracker") {
		t.Errorf("expected error_tracker processor, got %q", resp)
	}
}

func TestRuntime_UnknownCommand(t *testing.T) {
	r := NewReflector(nil, "")

	// Unknown /cmd → not handled (returns false).
	_, ok := r.HandleCommand("/unknown_cmd", nil)
	if ok {
		t.Error("expected unknown command to not be handled")
	}

	// Not a command at all.
	_, ok = r.HandleCommand("hello world", nil)
	if ok {
		t.Error("expected non-command to not be handled")
	}
}

func TestRuntime_ShellCommand(t *testing.T) {
	r := NewReflector(nil, "")
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// /shell with no args → help.
	resp, ok := r.HandleCommand("/shell", ms)
	if !ok {
		t.Fatal("expected /shell to be handled")
	}
	if !strings.Contains(resp, "Usage") {
		t.Errorf("expected usage, got %q", resp)
	}

	// /shell pwd → returns cwd (builtin, no tool registry needed).
	resp, _ = r.HandleCommand("/shell pwd", ms)
	if !strings.Contains(resp, string(os.PathSeparator)) {
		t.Errorf("expected directory path, got %q", resp)
	}

	// /shell dev tool without registry → warning.
	resp, _ = r.HandleCommand("/shell git status", ms)
	if !strings.Contains(resp, "not available") {
		t.Errorf("expected warning about no registry, got %q", resp)
	}
}

// --- Post-LLM processor tests -----------------------------------------------

func TestRuntime_ErrorTracker(t *testing.T) {
	tracker := &ErrorTracker{}
	input := RuntimeInput{
		ToolCalls: []ToolCallRecord{
			{Name: "exec", Error: "command not found"},
			{Name: "read_file", Error: ""},
		},
	}

	// Should not error.
	err := tracker.Process(nil, input, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRuntime_CotEvaluator_NoCot(t *testing.T) {
	eval := &CotEvaluator{}
	input := RuntimeInput{CotPrompt: ""} // No CoT → skip.

	err := eval.Process(nil, input, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRuntime_MemoryExtractor_SkipChat(t *testing.T) {
	extractor := &MemoryExtractor{}
	input := RuntimeInput{
		UserMessage: "hello",
		Intent:      "chat",
	}

	err := extractor.Process(nil, input, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRuntime_PostLLM_NilSafety(t *testing.T) {
	// Nil runtime should not panic.
	var r *Reflector
	r.RunPostLLM(RuntimeInput{}, nil) // Should be no-op.

	// Runtime with no processors.
	r = &Reflector{commands: map[string]CommandDef{}}
	r.RunPostLLM(RuntimeInput{}, nil) // Should be no-op.
}

func TestRuntime_ListCommands(t *testing.T) {
	r := NewReflector(nil, "")
	text := r.ListCommands()
	if !strings.Contains(text, "/memory") {
		t.Error("should list /memory command")
	}
	if !strings.Contains(text, "/cot") {
		t.Error("should list /cot command")
	}
	if !strings.Contains(text, "/runtime") {
		t.Error("should list /runtime command")
	}
}

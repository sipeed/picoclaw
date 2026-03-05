// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMatchesToolName_ExactMatch(t *testing.T) {
	if !matchesToolName("exec", "exec") {
		t.Error("expected exact match")
	}
	if matchesToolName("exec", "read_file") {
		t.Error("expected no match for different name")
	}
}

func TestMatchesToolName_Wildcard(t *testing.T) {
	if !matchesToolName("*", "exec") {
		t.Error("expected wildcard to match anything")
	}
	if !matchesToolName("*", "") {
		t.Error("expected wildcard to match empty")
	}
}

func TestMatchesToolName_PrefixGlob(t *testing.T) {
	if !matchesToolName("mcp_*", "mcp_github") {
		t.Error("expected prefix glob to match mcp_github")
	}
	if matchesToolName("mcp_*", "exec") {
		t.Error("expected prefix glob to not match exec")
	}
}

func TestMatchesToolName_EmptyMatcher(t *testing.T) {
	// Empty matcher is handled by the caller (Trigger), not matchesToolName.
	// matchesToolName("", "exec") does exact match against "", which is false.
	if matchesToolName("", "exec") {
		t.Error("empty matcher should not match via matchesToolName")
	}
}

func TestBuildEnvVars_Complete(t *testing.T) {
	payload := HookPayload{
		Event:      PostToolUse,
		ToolName:   "exec",
		ToolOutput: "hello",
		ToolError:  true,
		Channel:    "telegram",
		ChatID:     "chat-42",
	}

	env := buildEnvVars(payload)
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["PICOCLAW_HOOK_EVENT"] != "PostToolUse" {
		t.Errorf("expected event PostToolUse, got %q", envMap["PICOCLAW_HOOK_EVENT"])
	}
	if envMap["PICOCLAW_TOOL_NAME"] != "exec" {
		t.Errorf("expected tool_name exec, got %q", envMap["PICOCLAW_TOOL_NAME"])
	}
	if envMap["PICOCLAW_TOOL_OUTPUT"] != "hello" {
		t.Errorf("expected tool_output hello, got %q", envMap["PICOCLAW_TOOL_OUTPUT"])
	}
	if envMap["PICOCLAW_TOOL_ERROR"] != "true" {
		t.Errorf("expected tool_error true, got %q", envMap["PICOCLAW_TOOL_ERROR"])
	}
	if envMap["PICOCLAW_CHANNEL"] != "telegram" {
		t.Errorf("expected channel telegram, got %q", envMap["PICOCLAW_CHANNEL"])
	}
	if envMap["PICOCLAW_CHAT_ID"] != "chat-42" {
		t.Errorf("expected chat_id chat-42, got %q", envMap["PICOCLAW_CHAT_ID"])
	}
}

func TestBuildEnvVars_TruncatesLargeOutput(t *testing.T) {
	bigOutput := strings.Repeat("x", 10000)
	payload := HookPayload{
		ToolOutput: bigOutput,
	}
	env := buildEnvVars(payload)
	for _, e := range env {
		if strings.HasPrefix(e, "PICOCLAW_TOOL_OUTPUT=") {
			val := strings.TrimPrefix(e, "PICOCLAW_TOOL_OUTPUT=")
			if len(val) > 8192 {
				t.Errorf("expected output truncated to 8192, got %d", len(val))
			}
		}
	}
}

func TestBuildEnvVars_ErrorFalse(t *testing.T) {
	payload := HookPayload{
		ToolError: false,
	}
	env := buildEnvVars(payload)
	for _, e := range env {
		if strings.HasPrefix(e, "PICOCLAW_TOOL_ERROR=") {
			val := strings.TrimPrefix(e, "PICOCLAW_TOOL_ERROR=")
			if val != "false" {
				t.Errorf("expected tool_error false, got %q", val)
			}
		}
	}
}

func TestNewHookManager_NilRules(t *testing.T) {
	hm := NewHookManager(nil)
	if hm.HasHooks(PreMessage) {
		t.Error("expected no hooks for nil rules")
	}
}

func TestNewHookManager_EmptyRules(t *testing.T) {
	hm := NewHookManager(map[Event][]HookRule{})
	if hm.HasHooks(PostToolUse) {
		t.Error("expected no hooks for empty rules")
	}
}

func TestHookManager_HasHooks(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "exec", Command: "echo test"},
		},
	}
	hm := NewHookManager(rules)

	if !hm.HasHooks(PostToolUse) {
		t.Error("expected hooks for PostToolUse")
	}
	if hm.HasHooks(PreMessage) {
		t.Error("expected no hooks for PreMessage")
	}
}

func TestHookManager_Trigger_EchoCommand(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "", Command: "echo hook-output"},
		},
	}
	hm := NewHookManager(rules)

	results := hm.Trigger(context.Background(), PostToolUse, HookPayload{
		ToolName: "exec",
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Fatalf("expected no error, got %v", results[0].Err)
	}
	if strings.TrimSpace(results[0].Output) != "hook-output" {
		t.Errorf("expected output 'hook-output', got %q", results[0].Output)
	}
}

func TestHookManager_Trigger_MatcherFilters(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "read_file", Command: "echo should-not-run"},
		},
	}
	hm := NewHookManager(rules)

	results := hm.Trigger(context.Background(), PostToolUse, HookPayload{
		ToolName: "exec",
	})

	if len(results) != 0 {
		t.Errorf("expected 0 results (matcher filtered), got %d", len(results))
	}
}

func TestHookManager_Trigger_Timeout(t *testing.T) {
	rules := map[Event][]HookRule{
		PreMessage: {
			{Command: "sleep 10"},
		},
	}
	hm := NewHookManager(rules)
	hm.executor.Timeout = 100 * time.Millisecond

	results := hm.Trigger(context.Background(), PreMessage, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(results[0].Err.Error(), "timed out") {
		t.Errorf("expected timeout error message, got %q", results[0].Err.Error())
	}
}

func TestHookManager_CollectInjectedOutput(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "exec", Command: "echo injected-context", InjectOutput: true},
			{Matcher: "exec", Command: "echo not-injected", InjectOutput: false},
		},
	}
	hm := NewHookManager(rules)

	output := hm.CollectInjectedOutput(context.Background(), PostToolUse, HookPayload{
		ToolName: "exec",
	})

	trimmed := strings.TrimSpace(output)
	if trimmed != "injected-context" {
		t.Errorf("expected only inject_output=true content, got %q", trimmed)
	}
}

func TestHookManager_Trigger_StdinPayload(t *testing.T) {
	// Verify that the hook receives JSON payload via stdin
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "", Command: "cat", InjectOutput: true},
		},
	}
	hm := NewHookManager(rules)

	output := hm.CollectInjectedOutput(context.Background(), PostToolUse, HookPayload{
		ToolName:   "exec",
		ToolOutput: "test-output",
		Channel:    "telegram",
	})

	if !strings.Contains(output, `"tool_name":"exec"`) {
		t.Errorf("expected stdin to contain tool_name, got %q", output)
	}
	if !strings.Contains(output, `"channel":"telegram"`) {
		t.Errorf("expected stdin to contain channel, got %q", output)
	}
}

func TestHookManager_Trigger_FailedCommand(t *testing.T) {
	rules := map[Event][]HookRule{
		PreMessage: {
			{Command: "exit 1"},
		},
	}
	hm := NewHookManager(rules)

	results := hm.Trigger(context.Background(), PreMessage, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected error for exit 1")
	}
}

func TestHookManager_Trigger_NoRulesForEvent(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {{Command: "echo test"}},
	}
	hm := NewHookManager(rules)

	results := hm.Trigger(context.Background(), PreMessage, HookPayload{})
	if results != nil {
		t.Errorf("expected nil results for unregistered event, got %d", len(results))
	}
}

func TestHookManager_Trigger_MultipleRules(t *testing.T) {
	rules := map[Event][]HookRule{
		PostToolUse: {
			{Matcher: "", Command: "echo first"},
			{Matcher: "", Command: "echo second"},
		},
	}
	hm := NewHookManager(rules)

	results := hm.Trigger(context.Background(), PostToolUse, HookPayload{
		ToolName: "exec",
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if strings.TrimSpace(results[0].Output) != "first" {
		t.Errorf("expected first result 'first', got %q", results[0].Output)
	}
	if strings.TrimSpace(results[1].Output) != "second" {
		t.Errorf("expected second result 'second', got %q", results[1].Output)
	}
}

func TestHookManager_CollectInjectedOutput_MultipleInject(t *testing.T) {
	rules := map[Event][]HookRule{
		PreMessage: {
			{Command: "echo line1", InjectOutput: true},
			{Command: "echo line2", InjectOutput: true},
		},
	}
	hm := NewHookManager(rules)

	output := hm.CollectInjectedOutput(context.Background(), PreMessage, HookPayload{})

	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Errorf("expected both lines in output, got %q", output)
	}
}

func TestExpandHome(t *testing.T) {
	result := expandHome("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("expected absolute path unchanged, got %q", result)
	}

	result = expandHome("")
	if result != "" {
		t.Errorf("expected empty string unchanged, got %q", result)
	}

	// Verify ~ expansion doesn't panic
	result = expandHome("~/test")
	if strings.HasPrefix(result, "~") {
		t.Log("Home dir expansion may not work in test env, skipping")
	}
}

func TestExecutor_MaxOutputTruncation(t *testing.T) {
	e := NewExecutor()
	e.MaxOutputBytes = 10

	result := e.Run(context.Background(), "echo 'this is a long output that should be truncated'", nil, nil)
	if result.Err != nil {
		t.Fatalf("expected no error, got %v", result.Err)
	}
	if len(result.Output) > 10 {
		t.Errorf("expected output truncated to 10 bytes, got %d", len(result.Output))
	}
}

func TestExecutor_EmptyCommand(t *testing.T) {
	e := NewExecutor()
	result := e.Run(context.Background(), "", nil, nil)
	if result.Err == nil {
		t.Error("expected error for empty command")
	}
	if !strings.Contains(result.Err.Error(), "empty hook command") {
		t.Errorf("expected empty command error, got %q", result.Err.Error())
	}
}

func TestExecutor_ExtraEnvVars(t *testing.T) {
	e := NewExecutor()
	result := e.Run(context.Background(), "echo $TEST_HOOK_VAR", nil, []string{"TEST_HOOK_VAR=hello-hook"})
	if result.Err != nil {
		t.Fatalf("expected no error, got %v", result.Err)
	}
	if strings.TrimSpace(result.Output) != "hello-hook" {
		t.Errorf("expected 'hello-hook', got %q", result.Output)
	}
}

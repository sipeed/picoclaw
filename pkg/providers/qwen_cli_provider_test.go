// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// --- Compile-time interface check ---

var _ LLMProvider = (*QwenCliProvider)(nil)

// --- Helper: create mock CLI scripts ---

// createMockQwenCLI creates a temporary script that simulates the qwen CLI.
func createMockQwenCLI(t *testing.T, stdout, stderr string, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()

	if stdout != "" {
		if err := os.WriteFile(filepath.Join(dir, "stdout.txt"), []byte(stdout), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if stderr != "" {
		if err := os.WriteFile(filepath.Join(dir, "stderr.txt"), []byte(stderr), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	if stderr != "" {
		sb.WriteString(fmt.Sprintf("cat '%s/stderr.txt' >&2\n", dir))
	}
	if stdout != "" {
		sb.WriteString(fmt.Sprintf("cat '%s/stdout.txt'\n", dir))
	}
	sb.WriteString(fmt.Sprintf("exit %d\n", exitCode))

	script := filepath.Join(dir, "qwen")
	if err := os.WriteFile(script, []byte(sb.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createSlowMockQwenCLI creates a script that sleeps before responding.
func createSlowMockQwenCLI(t *testing.T, sleepSeconds int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "qwen")
	content := fmt.Sprintf(`#!/bin/sh
sleep %d
cat <<'EOFMOCK'
[{"type":"result","subtype":"success","is_error":false,"result":"late response"}]
EOFMOCK
`, sleepSeconds)
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createArgCaptureQwenCLI creates a script that captures CLI args to a file.
func createArgCaptureQwenCLI(t *testing.T, argsFile string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "qwen")
	content := fmt.Sprintf(`#!/bin/sh
echo "$@" > '%s'
cat <<'EOFMOCK'
[{"type":"result","subtype":"success","is_error":false,"result":"ok","session_id":"test"}]
EOFMOCK
`, argsFile)
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// --- Constructor tests ---

func TestNewQwenCliProvider(t *testing.T) {
	p := NewQwenCliProvider("/test/workspace")
	if p == nil {
		t.Fatal("NewQwenCliProvider returned nil")
	}
	if p.workspace != "/test/workspace" {
		t.Errorf("workspace = %q, want %q", p.workspace, "/test/workspace")
	}
	if p.command != "qwen" {
		t.Errorf("command = %q, want %q", p.command, "qwen")
	}
}

func TestNewQwenCliProvider_EmptyWorkspace(t *testing.T) {
	p := NewQwenCliProvider("")
	if p.workspace != "" {
		t.Errorf("workspace = %q, want empty", p.workspace)
	}
}

// --- GetDefaultModel tests ---

func TestQwenCliProvider_GetDefaultModel(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	if got := p.GetDefaultModel(); got != "qwen-cli" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "qwen-cli")
	}
}

// --- Chat() tests ---

func TestQwenChat_Success(t *testing.T) {
	mockJSON := `[{"type":"system","subtype":"init"},{"type":"assistant","message":{"content":[{"type":"text","text":"Hello from mock!"}]}},{"type":"result","subtype":"success","is_error":false,"result":"Hello from mock!","usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}]`
	script := createMockQwenCLI(t, mockJSON, "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "Hello from mock!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from mock!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestQwenChat_ResultOnly(t *testing.T) {
	// Test when only result event is present (no assistant event)
	mockJSON := `[{"type":"result","subtype":"success","is_error":false,"result":"Result only response","usage":{"input_tokens":5,"output_tokens":3}}]`
	script := createMockQwenCLI(t, mockJSON, "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "Result only response" {
		t.Errorf("Content = %q, want %q", resp.Content, "Result only response")
	}
}

func TestQwenChat_IsErrorResponse(t *testing.T) {
	mockJSON := `[{"type":"result","subtype":"error","is_error":true,"result":"API key invalid"}]`
	script := createMockQwenCLI(t, mockJSON, "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error when is_error=true")
	}
	if !strings.Contains(err.Error(), "API key invalid") {
		t.Errorf("error = %q, want to contain 'API key invalid'", err.Error())
	}
}

func TestQwenChat_WithToolCallsInResponse(t *testing.T) {
	result := `Let me check the weather.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"NYC\"}"}}]}`
	mockJSON := fmt.Sprintf(
		`[{"type":"result","subtype":"success","is_error":false,"result":%q,"usage":{"input_tokens":5,"output_tokens":20}}]`,
		result,
	)
	script := createMockQwenCLI(t, mockJSON, "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "What's the weather?"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
	if resp.ToolCalls[0].Arguments["location"] != "NYC" {
		t.Errorf("ToolCalls[0].Arguments[location] = %v, want NYC", resp.ToolCalls[0].Arguments["location"])
	}
}

func TestQwenChat_StderrError(t *testing.T) {
	script := createMockQwenCLI(t, "", "Error: connection failed", 1)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error")
	}
	if !strings.Contains(err.Error(), "connection failed") {
		t.Errorf("error = %q, want to contain 'connection failed'", err.Error())
	}
}

func TestQwenChat_NonZeroExitNoStderr(t *testing.T) {
	script := createMockQwenCLI(t, "", "", 1)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "qwen cli error") {
		t.Errorf("error = %q, want to contain 'qwen cli error'", err.Error())
	}
}

func TestQwenChat_CommandNotFound(t *testing.T) {
	p := NewQwenCliProvider(t.TempDir())
	p.command = "/nonexistent/qwen-binary-that-does-not-exist"

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for missing command")
	}
}

func TestQwenChat_InvalidResponseJSON(t *testing.T) {
	script := createMockQwenCLI(t, "not valid json", "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse qwen cli response") {
		t.Errorf("error = %q, want to contain 'failed to parse qwen cli response'", err.Error())
	}
}

func TestQwenChat_EmptyResponse(t *testing.T) {
	script := createMockQwenCLI(t, "", "", 0)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for empty response")
	}
}

func TestQwenChat_ContextCancellation(t *testing.T) {
	script := createSlowMockQwenCLI(t, 2) // sleep 2s

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := p.Chat(ctx, []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Chat() expected error on context cancellation")
	}
	// Should fail well before the full 2s sleep completes
	if elapsed > 3*time.Second {
		t.Errorf("Chat() took %v, expected to fail faster via context cancellation", elapsed)
	}
}

func TestQwenChat_PassesModelFlag(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureQwenCLI(t, argsFile)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "qwen-max", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if !strings.Contains(args, "-m") {
		t.Errorf("CLI args missing -m, got: %s", args)
	}
	if !strings.Contains(args, "qwen-max") {
		t.Errorf("CLI args missing model name, got: %s", args)
	}
}

func TestQwenChat_SkipsModelFlagForQwenCli(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureQwenCLI(t, argsFile)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "qwen-cli", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if strings.Contains(args, "-m") {
		t.Errorf("CLI args should NOT contain -m for qwen-cli, got: %s", args)
	}
}

func TestQwenChat_SkipsModelFlagForEmptyModel(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureQwenCLI(t, argsFile)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if strings.Contains(args, "-m") {
		t.Errorf("CLI args should NOT contain -m for empty model, got: %s", args)
	}
}

func TestQwenChat_EmptyWorkspaceDoesNotSetDir(t *testing.T) {
	mockJSON := `[{"type":"result","result":"ok","session_id":"s"}]`
	script := createMockQwenCLI(t, mockJSON, "", 0)

	p := NewQwenCliProvider("")
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() with empty workspace error = %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want %q", resp.Content, "ok")
	}
}

// --- buildPrompt tests ---

func TestQwenBuildPrompt_SingleUser(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	got := p.buildPrompt(messages, nil)
	want := "Hello"
	if got != want {
		t.Errorf("buildPrompt() = %q, want %q", got, want)
	}
}

func TestQwenBuildPrompt_Conversation(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
		{Role: "user", Content: "How are you?"},
	}
	got := p.buildPrompt(messages, nil)
	want := "Hi\nAssistant: Hello!\nHow are you?"
	if got != want {
		t.Errorf("buildPrompt() = %q, want %q", got, want)
	}
}

func TestQwenBuildPrompt_WithSystemMessage(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
	}
	got := p.buildPrompt(messages, nil)
	if !strings.Contains(got, "## System Instructions") {
		t.Error("missing system instructions header")
	}
	if !strings.Contains(got, "You are helpful.") {
		t.Error("missing system message")
	}
	if !strings.Contains(got, "## Task") {
		t.Error("missing task header")
	}
}

func TestQwenBuildPrompt_WithToolResults(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "What's the weather?"},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_123"},
	}
	got := p.buildPrompt(messages, nil)
	if !strings.Contains(got, "[Tool Result for call_123]") {
		t.Errorf("buildPrompt() missing tool result marker, got %q", got)
	}
	if !strings.Contains(got, `{"temp": 72}`) {
		t.Errorf("buildPrompt() missing tool result content, got %q", got)
	}
}

func TestQwenBuildPrompt_WithTools(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	got := p.buildPrompt(messages, tools)
	if !strings.Contains(got, "get_weather") {
		t.Error("buildPrompt() missing tool definition")
	}
	if !strings.Contains(got, "Available Tools") {
		t.Error("buildPrompt() missing tools header")
	}
}

func TestQwenBuildPrompt_EmptyMessages(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	got := p.buildPrompt(nil, nil)
	if got != "" {
		t.Errorf("buildPrompt(nil) = %q, want empty", got)
	}
}

// --- parseJSONEvents tests ---

func TestQwenParseJSONEvents_Success(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	output := `[{"type":"result","subtype":"success","is_error":false,"result":"Hello, world!","usage":{"input_tokens":10,"output_tokens":20}}]`

	resp, err := p.parseJSONEvents(output)
	if err != nil {
		t.Fatalf("parseJSONEvents() error = %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", resp.Usage.CompletionTokens)
	}
}

func TestQwenParseJSONEvents_WithAssistantEvent(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	output := `[{"type":"assistant","message":{"content":[{"type":"text","text":"Assistant says hi"}]}},{"type":"result","subtype":"success","is_error":false,"result":"Result text","usage":{"input_tokens":5,"output_tokens":3}}]`

	resp, err := p.parseJSONEvents(output)
	if err != nil {
		t.Fatalf("parseJSONEvents() error = %v", err)
	}
	// Should prefer assistant event content over result
	if !strings.Contains(resp.Content, "Assistant says hi") {
		t.Errorf("Content should contain assistant text, got %q", resp.Content)
	}
}

func TestQwenParseJSONEvents_Error(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	output := `[{"type":"result","subtype":"error","is_error":true,"result":"Something went wrong"}]`

	_, err := p.parseJSONEvents(output)
	if err == nil {
		t.Fatal("expected error when is_error=true")
	}
	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("error = %q, want to contain 'Something went wrong'", err.Error())
	}
}

func TestQwenParseJSONEvents_NoResultEvent(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	output := `[{"type":"system","subtype":"init"}]`

	resp, err := p.parseJSONEvents(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
}

func TestQwenParseJSONEvents_EmptyOutput(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	_, err := p.parseJSONEvents("")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestQwenParseJSONEvents_InvalidJSON(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	_, err := p.parseJSONEvents("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestQwenParseJSONEvents_NoUsage(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	output := `[{"type":"result","subtype":"success","is_error":false,"result":"hi"}]`

	resp, err := p.parseJSONEvents(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Usage != nil {
		t.Errorf("Usage should be nil when no tokens, got %+v", resp.Usage)
	}
}

func TestQwenParseJSONEvents_WithToolCalls(t *testing.T) {
	p := NewQwenCliProvider("/workspace")
	result := `Let me check.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}}]}`
	output := fmt.Sprintf(`[{"type":"result","subtype":"success","is_error":false,"result":%q}]`, result)

	resp, err := p.parseJSONEvents(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
}

// --- Factory integration tests ---

func TestCreateProviderFromConfig_QwenCli(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "qwen-code",
		Model:     "qwen-cli/qwen-code",
		Workspace: "/test/ws",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig(qwen-cli) error = %v", err)
	}

	if modelID != "qwen-code" {
		t.Errorf("modelID = %q, want %q", modelID, "qwen-code")
	}

	qwenProvider, ok := provider.(*QwenCliProvider)
	if !ok {
		t.Fatalf("CreateProviderFromConfig(qwen-cli) returned %T, want *QwenCliProvider", provider)
	}
	if qwenProvider.workspace != "/test/ws" {
		t.Errorf("workspace = %q, want %q", qwenProvider.workspace, "/test/ws")
	}
}

func TestCreateProviderFromConfig_QwenCliDefaultWorkspace(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "qwen-code",
		Model:     "qwen-cli/qwen-code",
		Workspace: "",
	}

	provider, _, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig error = %v", err)
	}

	qwenProvider, ok := provider.(*QwenCliProvider)
	if !ok {
		t.Fatalf("returned %T, want *QwenCliProvider", provider)
	}
	if qwenProvider.workspace != "." {
		t.Errorf("workspace = %q, want %q (default)", qwenProvider.workspace, ".")
	}
}

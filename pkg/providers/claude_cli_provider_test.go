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

var _ LLMProvider = (*ClaudeCliProvider)(nil)

// --- Helper: create mock CLI scripts ---

// createMockCLI creates a temporary script that simulates the claude CLI.
// Uses files for stdout/stderr to avoid shell quoting issues with JSON.
func createMockCLI(t *testing.T, stdout, stderr string, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()

	if stdout != "" {
		if err := os.WriteFile(filepath.Join(dir, "stdout.txt"), []byte(stdout), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if stderr != "" {
		if err := os.WriteFile(filepath.Join(dir, "stderr.txt"), []byte(stderr), 0644); err != nil {
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

	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte(sb.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createSlowMockCLI creates a script that sleeps before responding (for context cancellation tests).
func createSlowMockCLI(t *testing.T, sleepSeconds int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	content := fmt.Sprintf("#!/bin/sh\nsleep %d\necho '{\"type\":\"result\",\"result\":\"late\"}'\n", sleepSeconds)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createArgCaptureCLI creates a script that captures CLI args to a file, then outputs JSON.
func createArgCaptureCLI(t *testing.T, argsFile string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	content := fmt.Sprintf(`#!/bin/sh
echo "$@" > '%s'
cat <<'EOFMOCK'
{"type":"result","result":"ok","session_id":"test"}
EOFMOCK
`, argsFile)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// --- Constructor tests ---

func TestNewClaudeCliProvider(t *testing.T) {
	p := NewClaudeCliProvider("/test/workspace")
	if p == nil {
		t.Fatal("NewClaudeCliProvider returned nil")
	}
	if p.workspace != "/test/workspace" {
		t.Errorf("workspace = %q, want %q", p.workspace, "/test/workspace")
	}
	if p.command != "claude" {
		t.Errorf("command = %q, want %q", p.command, "claude")
	}
}

func TestNewClaudeCliProvider_EmptyWorkspace(t *testing.T) {
	p := NewClaudeCliProvider("")
	if p.workspace != "" {
		t.Errorf("workspace = %q, want empty", p.workspace)
	}
}

// --- GetDefaultModel tests ---

func TestClaudeCliProvider_GetDefaultModel(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	if got := p.GetDefaultModel(); got != "claude-code" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "claude-code")
	}
}

// --- Chat() tests ---

func TestChat_Success(t *testing.T) {
	mockJSON := `{"type":"result","subtype":"success","is_error":false,"result":"Hello from mock!","session_id":"sess_123","total_cost_usd":0.005,"duration_ms":200,"duration_api_ms":150,"num_turns":1,"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":100,"cache_read_input_tokens":0}}`
	script := createMockCLI(t, mockJSON, "", 0)

	p := NewClaudeCliProvider(t.TempDir())
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
	if resp.Usage.PromptTokens != 110 { // 10 + 100 + 0
		t.Errorf("PromptTokens = %d, want 110", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 115 { // 110 + 5
		t.Errorf("TotalTokens = %d, want 115", resp.Usage.TotalTokens)
	}
}

func TestChat_IsErrorResponse(t *testing.T) {
	mockJSON := `{"type":"result","subtype":"error","is_error":true,"result":"Rate limit exceeded","session_id":"s1","total_cost_usd":0}`
	script := createMockCLI(t, mockJSON, "", 0)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error when is_error=true")
	}
	if !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Errorf("error = %q, want to contain 'Rate limit exceeded'", err.Error())
	}
}

func TestChat_WithToolCallsInResponse(t *testing.T) {
	mockJSON := `{"type":"result","subtype":"success","is_error":false,"result":"Checking weather.\n{\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"location\\\":\\\"NYC\\\"}\"}}]}","session_id":"s1","total_cost_usd":0.01,"usage":{"input_tokens":5,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`
	script := createMockCLI(t, mockJSON, "", 0)

	p := NewClaudeCliProvider(t.TempDir())
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

func TestChat_StderrError(t *testing.T) {
	script := createMockCLI(t, "", "Error: rate limited", 1)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error = %q, want to contain 'rate limited'", err.Error())
	}
}

func TestChat_NonZeroExitNoStderr(t *testing.T) {
	script := createMockCLI(t, "", "", 1)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "claude cli error") {
		t.Errorf("error = %q, want to contain 'claude cli error'", err.Error())
	}
}

func TestChat_CommandNotFound(t *testing.T) {
	p := NewClaudeCliProvider(t.TempDir())
	p.command = "/nonexistent/claude-binary-that-does-not-exist"

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for missing command")
	}
}

func TestChat_InvalidResponseJSON(t *testing.T) {
	script := createMockCLI(t, "not valid json at all", "", 0)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse claude cli response") {
		t.Errorf("error = %q, want to contain 'failed to parse claude cli response'", err.Error())
	}
}

func TestChat_ContextCancellation(t *testing.T) {
	script := createSlowMockCLI(t, 2) // sleep 2s

	p := NewClaudeCliProvider(t.TempDir())
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

func TestChat_PassesSystemPromptFlag(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureCLI(t, argsFile)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "system", Content: "Be helpful."},
		{Role: "user", Content: "Hi"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read args file: %v", err)
	}
	args := string(argsBytes)
	if !strings.Contains(args, "--system-prompt") {
		t.Errorf("CLI args missing --system-prompt, got: %s", args)
	}
}

func TestChat_PassesModelFlag(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureCLI(t, argsFile)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "claude-sonnet-4-5-20250929", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if !strings.Contains(args, "--model") {
		t.Errorf("CLI args missing --model, got: %s", args)
	}
	if !strings.Contains(args, "claude-sonnet-4-5-20250929") {
		t.Errorf("CLI args missing model name, got: %s", args)
	}
}

func TestChat_SkipsModelFlagForClaudeCode(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureCLI(t, argsFile)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "claude-code", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if strings.Contains(args, "--model") {
		t.Errorf("CLI args should NOT contain --model for claude-code, got: %s", args)
	}
}

func TestChat_SkipsModelFlagForEmptyModel(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := createArgCaptureCLI(t, argsFile)

	p := NewClaudeCliProvider(t.TempDir())
	p.command = script

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	argsBytes, _ := os.ReadFile(argsFile)
	args := string(argsBytes)
	if strings.Contains(args, "--model") {
		t.Errorf("CLI args should NOT contain --model for empty model, got: %s", args)
	}
}

func TestChat_EmptyWorkspaceDoesNotSetDir(t *testing.T) {
	mockJSON := `{"type":"result","result":"ok","session_id":"s"}`
	script := createMockCLI(t, mockJSON, "", 0)

	p := NewClaudeCliProvider("")
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

// --- CreateProvider factory tests ---

func TestCreateProvider_ClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "claude-cli"
	cfg.Agents.Defaults.Workspace = "/test/ws"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claude-cli) error = %v", err)
	}

	cliProvider, ok := provider.(*ClaudeCliProvider)
	if !ok {
		t.Fatalf("CreateProvider(claude-cli) returned %T, want *ClaudeCliProvider", provider)
	}
	if cliProvider.workspace != "/test/ws" {
		t.Errorf("workspace = %q, want %q", cliProvider.workspace, "/test/ws")
	}
}

func TestCreateProvider_ClaudeCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "claude-code"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claude-code) error = %v", err)
	}
	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("CreateProvider(claude-code) returned %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProvider_ClaudeCodec(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "claudecode"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claudecode) error = %v", err)
	}
	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("CreateProvider(claudecode) returned %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProvider_ClaudeCliDefaultWorkspace(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "claude-cli"
	cfg.Agents.Defaults.Workspace = ""

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider error = %v", err)
	}

	cliProvider, ok := provider.(*ClaudeCliProvider)
	if !ok {
		t.Fatalf("returned %T, want *ClaudeCliProvider", provider)
	}
	if cliProvider.workspace != "." {
		t.Errorf("workspace = %q, want %q (default)", cliProvider.workspace, ".")
	}
}

// --- messagesToPrompt tests ---

func TestMessagesToPrompt_SingleUser(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	got := p.messagesToPrompt(messages)
	want := "Hello"
	if got != want {
		t.Errorf("messagesToPrompt() = %q, want %q", got, want)
	}
}

func TestMessagesToPrompt_Conversation(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
		{Role: "user", Content: "How are you?"},
	}
	got := p.messagesToPrompt(messages)
	want := "User: Hi\nAssistant: Hello!\nUser: How are you?"
	if got != want {
		t.Errorf("messagesToPrompt() = %q, want %q", got, want)
	}
}

func TestMessagesToPrompt_WithSystemMessage(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
	}
	got := p.messagesToPrompt(messages)
	want := "Hello"
	if got != want {
		t.Errorf("messagesToPrompt() = %q, want %q", got, want)
	}
}

func TestMessagesToPrompt_WithToolResults(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "What's the weather?"},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_123"},
	}
	got := p.messagesToPrompt(messages)
	if !strings.Contains(got, "[Tool Result for call_123]") {
		t.Errorf("messagesToPrompt() missing tool result marker, got %q", got)
	}
	if !strings.Contains(got, `{"temp": 72}`) {
		t.Errorf("messagesToPrompt() missing tool result content, got %q", got)
	}
}

func TestMessagesToPrompt_EmptyMessages(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	got := p.messagesToPrompt(nil)
	if got != "" {
		t.Errorf("messagesToPrompt(nil) = %q, want empty", got)
	}
}

func TestMessagesToPrompt_OnlySystemMessages(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "System 1"},
		{Role: "system", Content: "System 2"},
	}
	got := p.messagesToPrompt(messages)
	if got != "" {
		t.Errorf("messagesToPrompt() with only system msgs = %q, want empty", got)
	}
}

// --- buildSystemPrompt tests ---

func TestBuildSystemPrompt_NoSystemNoTools(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hi"},
	}
	got := p.buildSystemPrompt(messages, nil)
	if got != "" {
		t.Errorf("buildSystemPrompt() = %q, want empty", got)
	}
}

func TestBuildSystemPrompt_SystemOnly(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hi"},
	}
	got := p.buildSystemPrompt(messages, nil)
	if got != "You are helpful." {
		t.Errorf("buildSystemPrompt() = %q, want %q", got, "You are helpful.")
	}
}

func TestBuildSystemPrompt_MultipleSystemMessages(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "system", Content: "Be concise."},
		{Role: "user", Content: "Hi"},
	}
	got := p.buildSystemPrompt(messages, nil)
	if !strings.Contains(got, "You are helpful.") {
		t.Error("missing first system message")
	}
	if !strings.Contains(got, "Be concise.") {
		t.Error("missing second system message")
	}
	// Should be joined with double newline
	want := "You are helpful.\n\nBe concise."
	if got != want {
		t.Errorf("buildSystemPrompt() = %q, want %q", got, want)
	}
}

func TestBuildSystemPrompt_WithTools(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
	got := p.buildSystemPrompt(messages, tools)
	if !strings.Contains(got, "You are helpful.") {
		t.Error("buildSystemPrompt() missing system message")
	}
	if !strings.Contains(got, "get_weather") {
		t.Error("buildSystemPrompt() missing tool definition")
	}
	if !strings.Contains(got, "Available Tools") {
		t.Error("buildSystemPrompt() missing tools header")
	}
}

func TestBuildSystemPrompt_ToolsOnlyNoSystem(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
			},
		},
	}
	got := p.buildSystemPrompt(nil, tools)
	if !strings.Contains(got, "test_tool") {
		t.Error("should include tool definitions even without system messages")
	}
}

// --- buildToolsPrompt tests ---

func TestBuildToolsPrompt_SkipsNonFunction(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	tools := []ToolDefinition{
		{Type: "other", Function: ToolFunctionDefinition{Name: "skip_me"}},
		{Type: "function", Function: ToolFunctionDefinition{Name: "include_me", Description: "Included"}},
	}
	got := p.buildToolsPrompt(tools)
	if strings.Contains(got, "skip_me") {
		t.Error("buildToolsPrompt() should skip non-function tools")
	}
	if !strings.Contains(got, "include_me") {
		t.Error("buildToolsPrompt() should include function tools")
	}
}

func TestBuildToolsPrompt_NoDescription(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "bare_tool"}},
	}
	got := p.buildToolsPrompt(tools)
	if !strings.Contains(got, "bare_tool") {
		t.Error("should include tool name")
	}
	if strings.Contains(got, "Description:") {
		t.Error("should not include Description: line when empty")
	}
}

func TestBuildToolsPrompt_NoParameters(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{
			Name:        "no_params_tool",
			Description: "A tool with no parameters",
		}},
	}
	got := p.buildToolsPrompt(tools)
	if strings.Contains(got, "Parameters:") {
		t.Error("should not include Parameters: section when nil")
	}
}

// --- parseClaudeCliResponse tests ---

func TestParseClaudeCliResponse_TextOnly(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"success","is_error":false,"result":"Hello, world!","session_id":"abc123","total_cost_usd":0.01,"duration_ms":500,"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`

	resp, err := p.parseClaudeCliResponse(output)
	if err != nil {
		t.Fatalf("parseClaudeCliResponse() error = %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls = %d, want 0", len(resp.ToolCalls))
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

func TestParseClaudeCliResponse_EmptyResult(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"success","is_error":false,"result":"","session_id":"abc"}`

	resp, err := p.parseClaudeCliResponse(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestParseClaudeCliResponse_IsError(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"error","is_error":true,"result":"Something went wrong","session_id":"abc"}`

	_, err := p.parseClaudeCliResponse(output)
	if err == nil {
		t.Fatal("expected error when is_error=true")
	}
	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("error = %q, want to contain 'Something went wrong'", err.Error())
	}
}

func TestParseClaudeCliResponse_NoUsage(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"success","is_error":false,"result":"hi","session_id":"s"}`

	resp, err := p.parseClaudeCliResponse(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Usage != nil {
		t.Errorf("Usage should be nil when no tokens, got %+v", resp.Usage)
	}
}

func TestParseClaudeCliResponse_InvalidJSON(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	_, err := p.parseClaudeCliResponse("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse claude cli response") {
		t.Errorf("error = %q, want to contain 'failed to parse claude cli response'", err.Error())
	}
}

func TestParseClaudeCliResponse_WithToolCalls(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"success","is_error":false,"result":"Let me check.\n{\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"location\\\":\\\"Tokyo\\\"}\"}}]}","session_id":"abc123","total_cost_usd":0.01}`

	resp, err := p.parseClaudeCliResponse(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Function == nil {
		t.Fatal("Function is nil")
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Function.Name = %q, want %q", tc.Function.Name, "get_weather")
	}
	if tc.Arguments["location"] != "Tokyo" {
		t.Errorf("Arguments[location] = %v, want Tokyo", tc.Arguments["location"])
	}
	if strings.Contains(resp.Content, "tool_calls") {
		t.Errorf("Content should not contain tool_calls JSON, got %q", resp.Content)
	}
	if resp.Content != "Let me check." {
		t.Errorf("Content = %q, want %q", resp.Content, "Let me check.")
	}
}

func TestParseClaudeCliResponse_WhitespaceResult(t *testing.T) {
	p := NewClaudeCliProvider("/workspace")
	output := `{"type":"result","subtype":"success","is_error":false,"result":"  hello  \n  ","session_id":"s"}`

	resp, err := p.parseClaudeCliResponse(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q (should be trimmed)", resp.Content, "hello")
	}
}

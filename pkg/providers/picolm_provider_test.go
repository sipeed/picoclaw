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

var _ LLMProvider = (*PicoLMProvider)(nil)

// --- Helper: create mock picolm binary ---

// createMockPicoLM creates a temporary script that simulates the picolm binary.
// It reads stdin (the prompt) and writes stdout/stderr, exiting with the given code.
func createMockPicoLM(t *testing.T, stdout, stderr string, exitCode int) string {
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
	// Consume stdin to avoid broken pipe
	sb.WriteString("cat > /dev/null\n")
	if stderr != "" {
		sb.WriteString(fmt.Sprintf("cat '%s/stderr.txt' >&2\n", dir))
	}
	if stdout != "" {
		sb.WriteString(fmt.Sprintf("cat '%s/stdout.txt'\n", dir))
	}
	sb.WriteString(fmt.Sprintf("exit %d\n", exitCode))

	script := filepath.Join(dir, "picolm")
	if err := os.WriteFile(script, []byte(sb.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createStdinCaptureMockPicoLM creates a mock that captures stdin to a file.
func createStdinCaptureMockPicoLM(t *testing.T, captureFile string, stdout string) string {
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

	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	sb.WriteString(fmt.Sprintf("cat > '%s'\n", captureFile))
	if stdout != "" {
		sb.WriteString(fmt.Sprintf("cat '%s/stdout.txt'\n", dir))
	}
	sb.WriteString("exit 0\n")

	script := filepath.Join(dir, "picolm")
	if err := os.WriteFile(script, []byte(sb.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// createSlowMockPicoLM creates a script that sleeps before responding.
func createSlowMockPicoLM(t *testing.T, sleepSeconds int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "picolm")
	content := fmt.Sprintf("#!/bin/sh\ncat > /dev/null\nsleep %d\necho 'late response'\n", sleepSeconds)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// --- Constructor tests ---

func TestNewPicoLMProvider(t *testing.T) {
	binary := createMockPicoLM(t, "", "", 0)
	p, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Binary:    binary,
		Model:     "/tmp/model.gguf",
		MaxTokens: 128,
		Threads:   2,
	})
	if err != nil {
		t.Fatalf("NewPicoLMProvider() error = %v", err)
	}
	if p.binary != binary {
		t.Errorf("binary = %q, want %q", p.binary, binary)
	}
	if p.maxTokens != 128 {
		t.Errorf("maxTokens = %d, want 128", p.maxTokens)
	}
	if p.threads != 2 {
		t.Errorf("threads = %d, want 2", p.threads)
	}
}

func TestNewPicoLMProvider_Defaults(t *testing.T) {
	binary := createMockPicoLM(t, "", "", 0)
	p, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Binary: binary,
		Model:  "/tmp/model.gguf",
	})
	if err != nil {
		t.Fatalf("NewPicoLMProvider() error = %v", err)
	}
	if p.maxTokens != 256 {
		t.Errorf("maxTokens = %d, want 256 (default)", p.maxTokens)
	}
	if p.threads != 4 {
		t.Errorf("threads = %d, want 4 (default)", p.threads)
	}
}

func TestNewPicoLMProvider_BinaryNotFound(t *testing.T) {
	_, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Binary: "/nonexistent/path/picolm",
		Model:  "/tmp/model.gguf",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Errorf("error = %q, want to contain 'binary not found'", err.Error())
	}
}

func TestNewPicoLMProvider_BinaryIsDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Binary: dir,
		Model:  "/tmp/model.gguf",
	})
	if err == nil {
		t.Fatal("expected error when binary is a directory")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("error = %q, want to contain 'is a directory'", err.Error())
	}
}

func TestNewPicoLMProvider_BinaryNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit not meaningful on Windows")
	}
	dir := t.TempDir()
	notExec := filepath.Join(dir, "picolm")
	if err := os.WriteFile(notExec, []byte("not executable"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Binary: notExec,
		Model:  "/tmp/model.gguf",
	})
	if err == nil {
		t.Fatal("expected error for non-executable binary")
	}
	if !strings.Contains(err.Error(), "not executable") {
		t.Errorf("error = %q, want to contain 'not executable'", err.Error())
	}
}

func TestNewPicoLMProvider_EmptyBinaryAllowed(t *testing.T) {
	// Empty binary is allowed at construction time (caught at Chat time)
	p, err := NewPicoLMProvider(config.PicoLMProviderConfig{
		Model: "/tmp/model.gguf",
	})
	if err != nil {
		t.Fatalf("NewPicoLMProvider() error = %v", err)
	}
	if p.binary != "" {
		t.Errorf("binary = %q, want empty", p.binary)
	}
}

// --- GetDefaultModel tests ---

func TestPicoLMProvider_GetDefaultModel(t *testing.T) {
	binary := createMockPicoLM(t, "", "", 0)
	p, _ := NewPicoLMProvider(config.PicoLMProviderConfig{Binary: binary})
	if got := p.GetDefaultModel(); got != "picolm-local" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "picolm-local")
	}
}

// --- Chat() tests ---

func TestPicoLMChat_Success(t *testing.T) {
	script := createMockPicoLM(t, "Photosynthesis is the process by which plants convert sunlight.", "", 0)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "What is photosynthesis?"},
	}, nil, "", nil)

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "Photosynthesis is the process by which plants convert sunlight." {
		t.Errorf("Content = %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
}

func TestPicoLMChat_WithToolCallsInResponse(t *testing.T) {
	mockOutput := `Checking weather.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"NYC\"}"}}]}`
	script := createMockPicoLM(t, mockOutput, "", 0)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "What's the weather?"},
	}, []ToolDefinition{{
		Type: "function",
		Function: ToolFunctionDefinition{
			Name:        "get_weather",
			Description: "Get weather",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}}, "", nil)

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

func TestPicoLMChat_EmptyOutput(t *testing.T) {
	script := createMockPicoLM(t, "", "", 0)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestPicoLMChat_StderrError(t *testing.T) {
	script := createMockPicoLM(t, "", "model load failed: out of memory", 1)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error")
	}
	if !strings.Contains(err.Error(), "out of memory") {
		t.Errorf("error = %q, want to contain 'out of memory'", err.Error())
	}
}

func TestPicoLMChat_NonZeroExitNoStderr(t *testing.T) {
	script := createMockPicoLM(t, "", "", 1)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "picolm error") {
		t.Errorf("error = %q, want to contain 'picolm error'", err.Error())
	}
}

func TestPicoLMChat_NonZeroExitWithStdout(t *testing.T) {
	// When the process fails with non-zero exit, stdout should NOT be treated as valid output.
	script := createMockPicoLM(t, "partial garbage output", "segfault", 139)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error when process exits non-zero, even with stdout")
	}
}

func TestPicoLMChat_ContextCancellation(t *testing.T) {
	script := createSlowMockPicoLM(t, 30)

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := p.Chat(ctx, []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)

	if err == nil {
		t.Fatal("Chat() expected error on context cancellation")
	}
}

func TestPicoLMChat_NoBinary(t *testing.T) {
	p := &PicoLMProvider{model: "/tmp/model.gguf"}
	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	if err == nil || !strings.Contains(err.Error(), "binary path not configured") {
		t.Errorf("expected 'binary path not configured' error, got: %v", err)
	}
}

func TestPicoLMChat_NoModel(t *testing.T) {
	p := &PicoLMProvider{binary: "/usr/bin/echo"}
	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, nil, "", nil)
	if err == nil || !strings.Contains(err.Error(), "model path not configured") {
		t.Errorf("expected 'model path not configured' error, got: %v", err)
	}
}

// --- buildPrompt tests ---

func TestPicoLMBuildPrompt_SimpleUserMessage(t *testing.T) {
	p := &PicoLMProvider{}
	prompt := p.buildPrompt([]Message{
		{Role: "user", Content: "Hello"},
	}, nil)

	if !strings.Contains(prompt, "<|user|>\nHello</s>") {
		t.Errorf("prompt missing user message, got:\n%s", prompt)
	}
	if !strings.HasSuffix(prompt, "<|assistant|>\n") {
		t.Errorf("prompt should end with assistant turn, got:\n%s", prompt)
	}
}

func TestPicoLMBuildPrompt_SystemAndUser(t *testing.T) {
	p := &PicoLMProvider{}
	prompt := p.buildPrompt([]Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
	}, nil)

	if !strings.Contains(prompt, "<|system|>\nYou are helpful.</s>") {
		t.Errorf("prompt missing system message, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "<|user|>\nHello</s>") {
		t.Errorf("prompt missing user message, got:\n%s", prompt)
	}
}

func TestPicoLMBuildPrompt_MultipleSystemMessages(t *testing.T) {
	p := &PicoLMProvider{}
	prompt := p.buildPrompt([]Message{
		{Role: "system", Content: "Part one."},
		{Role: "system", Content: "Part two."},
		{Role: "user", Content: "Hello"},
	}, nil)

	if !strings.Contains(prompt, "Part one.\n\nPart two.") {
		t.Errorf("system parts should be joined, got:\n%s", prompt)
	}
	// Should have exactly one <|system|> block
	if strings.Count(prompt, "<|system|>") != 1 {
		t.Errorf("expected exactly 1 system block, got %d", strings.Count(prompt, "<|system|>"))
	}
}

func TestPicoLMBuildPrompt_AssistantMessage(t *testing.T) {
	p := &PicoLMProvider{}
	prompt := p.buildPrompt([]Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
		{Role: "user", Content: "How are you?"},
	}, nil)

	if !strings.Contains(prompt, "<|assistant|>\nHello!</s>") {
		t.Errorf("prompt missing assistant message, got:\n%s", prompt)
	}
}

func TestPicoLMBuildPrompt_ToolResult(t *testing.T) {
	p := &PicoLMProvider{}
	prompt := p.buildPrompt([]Message{
		{Role: "user", Content: "What's the weather?"},
		{Role: "tool", ToolCallID: "call_1", Content: `{"temp": 72}`},
	}, nil)

	if !strings.Contains(prompt, "[Tool Result for call_1]: {\"temp\": 72}") {
		t.Errorf("prompt missing tool result, got:\n%s", prompt)
	}
	// Tool results are wrapped in user tags
	if !strings.Contains(prompt, "<|user|>\n[Tool Result for call_1]") {
		t.Errorf("tool result should be in user block, got:\n%s", prompt)
	}
}

func TestPicoLMBuildPrompt_WithTools(t *testing.T) {
	p := &PicoLMProvider{}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a city",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"city"},
				},
			},
		},
	}

	prompt := p.buildPrompt([]Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "What's the weather in NYC?"},
	}, tools)

	// Tool definitions should be in the system block
	if !strings.Contains(prompt, "## Available Tools") {
		t.Errorf("prompt missing tool definitions header, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "#### get_weather") {
		t.Errorf("prompt missing tool name, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Get weather for a city") {
		t.Errorf("prompt missing tool description, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, `"tool_calls"`) {
		t.Errorf("prompt missing tool call format example, got:\n%s", prompt)
	}
	// Tool definitions should be part of the system block
	systemIdx := strings.Index(prompt, "<|system|>")
	systemEnd := strings.Index(prompt, "</s>")
	toolsIdx := strings.Index(prompt, "## Available Tools")
	if toolsIdx < systemIdx || toolsIdx > systemEnd {
		t.Errorf("tool definitions should be inside system block")
	}
}

func TestPicoLMBuildPrompt_WithTools_NoSystem(t *testing.T) {
	p := &PicoLMProvider{}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "search",
				Description: "Search the web",
			},
		},
	}

	prompt := p.buildPrompt([]Message{
		{Role: "user", Content: "Search for cats"},
	}, tools)

	// Should still create a system block for tools even without system messages
	if !strings.Contains(prompt, "<|system|>") {
		t.Errorf("expected system block for tools, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "#### search") {
		t.Errorf("prompt missing tool name, got:\n%s", prompt)
	}
}

func TestPicoLMBuildPrompt_ToolsFilterNonFunction(t *testing.T) {
	p := &PicoLMProvider{}
	tools := []ToolDefinition{
		{
			Type: "not_a_function",
			Function: ToolFunctionDefinition{
				Name: "should_be_skipped",
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "included",
				Description: "This one is included",
			},
		},
	}

	prompt := p.buildPrompt([]Message{
		{Role: "user", Content: "Hello"},
	}, tools)

	if strings.Contains(prompt, "should_be_skipped") {
		t.Errorf("non-function tools should be filtered out")
	}
	if !strings.Contains(prompt, "#### included") {
		t.Errorf("function tools should be included")
	}
}

// --- Stdin prompt verification ---

func TestPicoLMChat_PromptSentViaStdin(t *testing.T) {
	captureFile := filepath.Join(t.TempDir(), "captured_stdin.txt")
	script := createStdinCaptureMockPicoLM(t, captureFile, "Response from model")

	p := &PicoLMProvider{
		binary:    script,
		model:     "/tmp/model.gguf",
		maxTokens: 256,
		threads:   4,
	}

	_, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "What is 2+2?"},
	}, nil, "", nil)

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	captured, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("failed to read captured stdin: %v", err)
	}

	stdinContent := string(captured)
	if !strings.Contains(stdinContent, "What is 2+2?") {
		t.Errorf("stdin should contain user message, got:\n%s", stdinContent)
	}
	if !strings.Contains(stdinContent, "<|user|>") {
		t.Errorf("stdin should contain ChatML tags, got:\n%s", stdinContent)
	}
}

// --- expandHome tests ---

func TestExpandHome_Empty(t *testing.T) {
	result, err := expandHome("")
	if err != nil {
		t.Fatalf("expandHome(\"\") error = %v", err)
	}
	if result != "" {
		t.Errorf("expandHome(\"\") = %q, want empty", result)
	}
}

func TestExpandHome_NoTilde(t *testing.T) {
	result, err := expandHome("/usr/bin/picolm")
	if err != nil {
		t.Fatalf("expandHome error = %v", err)
	}
	if result != "/usr/bin/picolm" {
		t.Errorf("expandHome = %q, want %q", result, "/usr/bin/picolm")
	}
}

func TestExpandHome_WithTilde(t *testing.T) {
	result, err := expandHome("~/bin/picolm")
	if err != nil {
		t.Fatalf("expandHome error = %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := home + "/bin/picolm"
	if result != expected {
		t.Errorf("expandHome = %q, want %q", result, expected)
	}
}

func TestExpandHome_TildeOnly(t *testing.T) {
	result, err := expandHome("~")
	if err != nil {
		t.Fatalf("expandHome error = %v", err)
	}
	home, _ := os.UserHomeDir()
	if result != home {
		t.Errorf("expandHome = %q, want %q", result, home)
	}
}

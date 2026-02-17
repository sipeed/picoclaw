package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- JSONL Event Parsing Tests ---

func TestParseJSONLEvents_AgentMessage(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"thread.started","thread_id":"abc-123"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Hello from Codex!"}}
{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20}}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	if resp.Content != "Hello from Codex!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from Codex!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if resp.Usage.PromptTokens != 150 {
		t.Errorf("PromptTokens = %d, want 150", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 170 {
		t.Errorf("TotalTokens = %d, want 170", resp.Usage.TotalTokens)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty, got %d", len(resp.ToolCalls))
	}
}

func TestParseJSONLEvents_ToolCallExtraction(t *testing.T) {
	p := &CodexCliProvider{}
	toolCallText := `Let me read that file.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"/tmp/test.txt\"}"}}]}`
	// Build valid JSONL by marshaling the event
	item := codexEvent{
		Type: "item.completed",
		Item: &codexEventItem{ID: "item_1", Type: "agent_message", Text: toolCallText},
	}
	itemJSON, _ := json.Marshal(item)
	usageEvt := `{"type":"turn.completed","usage":{"input_tokens":50,"cached_input_tokens":0,"output_tokens":20}}`
	events := `{"type":"turn.started"}` + "\n" + string(itemJSON) + "\n" + usageEvt

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls count = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "read_file")
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", resp.ToolCalls[0].ID, "call_1")
	}
	if resp.ToolCalls[0].Function.Arguments != `{"path":"/tmp/test.txt"}` {
		t.Errorf("ToolCalls[0].Function.Arguments = %q", resp.ToolCalls[0].Function.Arguments)
	}
	// Content should have the tool call JSON stripped
	if strings.Contains(resp.Content, "tool_calls") {
		t.Errorf("Content should not contain tool_calls JSON, got: %q", resp.Content)
	}
}

func TestParseJSONLEvents_MultipleToolCalls(t *testing.T) {
	p := &CodexCliProvider{}
	toolCallText := `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}},{"id":"call_2","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"b.txt\",\"content\":\"hello\"}"}}]}`
	item := codexEvent{
		Type: "item.completed",
		Item: &codexEventItem{ID: "item_1", Type: "agent_message", Text: toolCallText},
	}
	itemJSON, _ := json.Marshal(item)
	events := `{"type":"turn.started"}` + "\n" + string(itemJSON) + "\n" + `{"type":"turn.completed"}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("ToolCalls count = %d, want 2", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "read_file")
	}
	if resp.ToolCalls[1].Name != "write_file" {
		t.Errorf("ToolCalls[1].Name = %q, want %q", resp.ToolCalls[1].Name, "write_file")
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
}

func TestParseJSONLEvents_MultipleMessages(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"First part."}}
{"type":"item.completed","item":{"id":"item_2","type":"command_execution","command":"ls","status":"completed"}}
{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Second part."}}
{"type":"turn.completed"}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	if resp.Content != "First part.\nSecond part." {
		t.Errorf("Content = %q, want %q", resp.Content, "First part.\nSecond part.")
	}
}

func TestParseJSONLEvents_ErrorEvent(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"thread.started","thread_id":"abc"}
{"type":"turn.started"}
{"type":"error","message":"token expired"}
{"type":"turn.failed","error":{"message":"token expired"}}`

	_, err := p.parseJSONLEvents(events)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token expired") {
		t.Errorf("error = %q, want to contain 'token expired'", err.Error())
	}
}

func TestParseJSONLEvents_TurnFailed(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"turn.started"}
{"type":"turn.failed","error":{"message":"rate limit exceeded"}}`

	_, err := p.parseJSONLEvents(events)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("error = %q, want to contain 'rate limit exceeded'", err.Error())
	}
}

func TestParseJSONLEvents_ErrorWithContent(t *testing.T) {
	p := &CodexCliProvider{}
	// If there's an error but also content, return the content (partial success)
	events := `{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Partial result."}}
{"type":"error","message":"connection reset"}
{"type":"turn.failed","error":{"message":"connection reset"}}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("should not error when content exists: %v", err)
	}
	if resp.Content != "Partial result." {
		t.Errorf("Content = %q, want %q", resp.Content, "Partial result.")
	}
}

func TestParseJSONLEvents_EmptyOutput(t *testing.T) {
	p := &CodexCliProvider{}
	resp, err := p.parseJSONLEvents("")
	if err != nil {
		t.Fatalf("empty output should not error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
}

func TestParseJSONLEvents_MalformedLines(t *testing.T) {
	p := &CodexCliProvider{}
	events := `not json at all
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Good line."}}
another bad line
{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":5}}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("should skip malformed lines: %v", err)
	}
	if resp.Content != "Good line." {
		t.Errorf("Content = %q, want %q", resp.Content, "Good line.")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 15 {
		t.Errorf("Usage.TotalTokens = %v, want 15", resp.Usage)
	}
}

func TestParseJSONLEvents_CommandExecution(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"turn.started"}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","status":"completed","exit_code":0,"output":"file1.go\nfile2.go"}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Found 2 files."}}
{"type":"turn.completed"}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	// command_execution items should be skipped; only agent_message text is returned
	if resp.Content != "Found 2 files." {
		t.Errorf("Content = %q, want %q", resp.Content, "Found 2 files.")
	}
}

func TestParseJSONLEvents_NoUsage(t *testing.T) {
	p := &CodexCliProvider{}
	events := `{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"No usage info."}}
{"type":"turn.completed"}`

	resp, err := p.parseJSONLEvents(events)
	if err != nil {
		t.Fatalf("parseJSONLEvents() error: %v", err)
	}
	if resp.Usage != nil {
		t.Errorf("Usage should be nil when turn.completed has no usage, got %+v", resp.Usage)
	}
}

// --- Prompt Building Tests ---

func TestBuildPrompt_SystemAsInstructions(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hi there"},
	}

	prompt := p.buildPrompt(messages, nil)

	if !strings.Contains(prompt, "## System Instructions") {
		t.Error("prompt should contain '## System Instructions'")
	}
	if !strings.Contains(prompt, "You are helpful.") {
		t.Error("prompt should contain system content")
	}
	if !strings.Contains(prompt, "## Task") {
		t.Error("prompt should contain '## Task'")
	}
	if !strings.Contains(prompt, "Hi there") {
		t.Error("prompt should contain user message")
	}
}

func TestBuildPrompt_NoSystem(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "user", Content: "Just a question"},
	}

	prompt := p.buildPrompt(messages, nil)

	if strings.Contains(prompt, "## System Instructions") {
		t.Error("prompt should not contain system instructions header")
	}
	if prompt != "Just a question" {
		t.Errorf("prompt = %q, want %q", prompt, "Just a question")
	}
}

func TestBuildPrompt_WithTools(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "user", Content: "Get weather"},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get current weather",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}

	prompt := p.buildPrompt(messages, tools)

	if !strings.Contains(prompt, "## Available Tools") {
		t.Error("prompt should contain tools section")
	}
	if !strings.Contains(prompt, "get_weather") {
		t.Error("prompt should contain tool name")
	}
	if !strings.Contains(prompt, "Get current weather") {
		t.Error("prompt should contain tool description")
	}
}

func TestBuildPrompt_MultipleMessages(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi! How can I help?"},
		{Role: "user", Content: "Tell me about Go"},
	}

	prompt := p.buildPrompt(messages, nil)

	if !strings.Contains(prompt, "Hello") {
		t.Error("prompt should contain first user message")
	}
	if !strings.Contains(prompt, "Assistant: Hi! How can I help?") {
		t.Error("prompt should contain assistant message with prefix")
	}
	if !strings.Contains(prompt, "Tell me about Go") {
		t.Error("prompt should contain second user message")
	}
}

func TestBuildPrompt_ToolResults(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "user", Content: "Weather?"},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
	}

	prompt := p.buildPrompt(messages, nil)

	if !strings.Contains(prompt, "[Tool Result for call_1]") {
		t.Error("prompt should contain tool result")
	}
	if !strings.Contains(prompt, `{"temp": 72}`) {
		t.Error("prompt should contain tool result content")
	}
}

func TestBuildPrompt_SystemAndTools(t *testing.T) {
	p := &CodexCliProvider{}
	messages := []Message{
		{Role: "system", Content: "Be concise."},
		{Role: "user", Content: "Do something"},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "my_tool",
				Description: "A tool",
			},
		},
	}

	prompt := p.buildPrompt(messages, tools)

	// System instructions should come first
	sysIdx := strings.Index(prompt, "## System Instructions")
	toolIdx := strings.Index(prompt, "## Available Tools")
	taskIdx := strings.Index(prompt, "## Task")

	if sysIdx == -1 || toolIdx == -1 || taskIdx == -1 {
		t.Fatal("prompt should contain all sections")
	}
	if sysIdx >= taskIdx {
		t.Error("system instructions should come before task")
	}
	if taskIdx >= toolIdx {
		t.Error("task section should come before tools in the output")
	}
}

// --- CLI Argument Tests ---

func TestCodexCliProvider_GetDefaultModel(t *testing.T) {
	p := NewCodexCliProvider("")
	if got := p.GetDefaultModel(); got != "codex-cli" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "codex-cli")
	}
}

// --- Mock CLI Integration Test ---

func createMockCodexCLI(t *testing.T, events []string) string {
	t.Helper()
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "codex")

	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	for _, event := range events {
		sb.WriteString(fmt.Sprintf("echo '%s'\n", event))
	}

	if err := os.WriteFile(scriptPath, []byte(sb.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return scriptPath
}

func TestCodexCliProvider_MockCLI_Success(t *testing.T) {
	scriptPath := createMockCodexCLI(t, []string{
		`{"type":"thread.started","thread_id":"test-123"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Mock response from Codex CLI"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":50,"cached_input_tokens":10,"output_tokens":15}}`,
	})

	p := &CodexCliProvider{
		command:   scriptPath,
		workspace: "",
	}

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := p.Chat(context.Background(), messages, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Mock response from Codex CLI" {
		t.Errorf("Content = %q, want %q", resp.Content, "Mock response from Codex CLI")
	}
	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if resp.Usage.PromptTokens != 60 {
		t.Errorf("PromptTokens = %d, want 60", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 15 {
		t.Errorf("CompletionTokens = %d, want 15", resp.Usage.CompletionTokens)
	}
}

func TestCodexCliProvider_MockCLI_Error(t *testing.T) {
	scriptPath := createMockCodexCLI(t, []string{
		`{"type":"thread.started","thread_id":"test-err"}`,
		`{"type":"turn.started"}`,
		`{"type":"error","message":"auth token expired"}`,
		`{"type":"turn.failed","error":{"message":"auth token expired"}}`,
	})

	p := &CodexCliProvider{
		command:   scriptPath,
		workspace: "",
	}

	messages := []Message{{Role: "user", Content: "Hello"}}
	_, err := p.Chat(context.Background(), messages, nil, "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth token expired") {
		t.Errorf("error = %q, want to contain 'auth token expired'", err.Error())
	}
}

func TestCodexCliProvider_MockCLI_WithModel(t *testing.T) {
	// Mock script that captures args to verify model flag is passed
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "codex")
	script := `#!/bin/bash
# Write args to a file for verification
echo "$@" > "` + filepath.Join(tmpDir, "args.txt") + `"
echo '{"type":"item.completed","item":{"id":"1","type":"agent_message","text":"ok"}}'
echo '{"type":"turn.completed"}'`

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	p := &CodexCliProvider{
		command:   scriptPath,
		workspace: "/tmp/test-workspace",
	}

	messages := []Message{{Role: "user", Content: "test"}}
	_, err := p.Chat(context.Background(), messages, nil, "gpt-5.2-codex", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// Verify the args
	argsData, err := os.ReadFile(filepath.Join(tmpDir, "args.txt"))
	if err != nil {
		t.Fatalf("reading args: %v", err)
	}
	args := string(argsData)

	if !strings.Contains(args, "-m gpt-5.2-codex") {
		t.Errorf("args should contain model flag, got: %s", args)
	}
	if !strings.Contains(args, "-C /tmp/test-workspace") {
		t.Errorf("args should contain workspace flag, got: %s", args)
	}
	if !strings.Contains(args, "--json") {
		t.Errorf("args should contain --json, got: %s", args)
	}
	if !strings.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
		t.Errorf("args should contain bypass flag, got: %s", args)
	}
}

func TestCodexCliProvider_MockCLI_ContextCancel(t *testing.T) {
	// Script that sleeps forever
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "codex")
	script := "#!/bin/bash\nsleep 60"

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	p := &CodexCliProvider{
		command:   scriptPath,
		workspace: "",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	messages := []Message{{Role: "user", Content: "test"}}
	_, err := p.Chat(ctx, messages, nil, "", nil)
	if err == nil {
		t.Fatal("expected error on canceled context")
	}
}

func TestCodexCliProvider_EmptyCommand(t *testing.T) {
	p := &CodexCliProvider{command: ""}

	messages := []Message{{Role: "user", Content: "test"}}
	_, err := p.Chat(context.Background(), messages, nil, "", nil)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

// --- Integration Test (requires real codex CLI with valid auth) ---

func TestCodexCliProvider_Integration(t *testing.T) {
	if os.Getenv("PICOCLAW_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test (set PICOCLAW_INTEGRATION_TESTS=1 to enable)")
	}

	// Verify codex is available
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex CLI not found in PATH")
	}

	p := &CodexCliProvider{
		command:   codexPath,
		workspace: "",
	}

	messages := []Message{
		{Role: "user", Content: "Respond with just the word 'hello' and nothing else."},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	lower := strings.ToLower(strings.TrimSpace(resp.Content))
	if !strings.Contains(lower, "hello") {
		t.Errorf("Content = %q, expected to contain 'hello'", resp.Content)
	}
}

package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// MockLLMProvider is a test implementation of LLMProvider
type MockLLMProvider struct {
	lastOptions map[string]any
}

func (m *MockLLMProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	m.lastOptions = options
	// Find the last user message to generate a response
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return &providers.LLMResponse{
				Content: "Task completed: " + messages[i].Content,
			}, nil
		}
	}
	return &providers.LLMResponse{Content: "No task provided"}, nil
}

func (m *MockLLMProvider) GetDefaultModel() string {
	return "test-model"
}

func (m *MockLLMProvider) SupportsTools() bool {
	return false
}

func (m *MockLLMProvider) GetContextWindow() int {
	return 4096
}

func TestSubagentManager_SetLLMOptions_AppliesToRunToolLoop(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	manager.SetLLMOptions(2048, 0.6)
	tool := NewSubagentTool(manager)
	tool.SetContext("cli", "direct")

	ctx := context.Background()
	args := map[string]any{"task": "Do something"}
	result := tool.Execute(ctx, args)

	if result == nil || result.IsError {
		t.Fatalf("Expected successful result, got: %+v", result)
	}

	if provider.lastOptions == nil {
		t.Fatal("Expected LLM options to be passed, got nil")
	}
	if provider.lastOptions["max_tokens"] != 2048 {
		t.Fatalf("max_tokens = %v, want %d", provider.lastOptions["max_tokens"], 2048)
	}
	if provider.lastOptions["temperature"] != 0.6 {
		t.Fatalf("temperature = %v, want %v", provider.lastOptions["temperature"], 0.6)
	}
}

// TestSubagentTool_Name verifies tool name
func TestSubagentTool_Name(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	if tool.Name() != "subagent" {
		t.Errorf("Expected name 'subagent', got '%s'", tool.Name())
	}
}

// TestSubagentTool_Description verifies tool description
func TestSubagentTool_Description(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if !strings.Contains(desc, "BLOCK") {
		t.Errorf("Description should mention 'BLOCK', got: %s", desc)
	}
	if !strings.Contains(desc, "spawn") {
		t.Errorf("Description should contrast with spawn, got: %s", desc)
	}
}

// TestSubagentTool_Parameters verifies tool parameters schema
func TestSubagentTool_Parameters(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters should not be nil")
	}

	// Check type
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got: %v", params["type"])
	}

	// Check properties
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Verify task parameter
	task, ok := props["task"].(map[string]any)
	if !ok {
		t.Fatal("Task parameter should exist")
	}
	if task["type"] != "string" {
		t.Errorf("Task type should be 'string', got: %v", task["type"])
	}

	// Verify label parameter
	label, ok := props["label"].(map[string]any)
	if !ok {
		t.Fatal("Label parameter should exist")
	}
	if label["type"] != "string" {
		t.Errorf("Label type should be 'string', got: %v", label["type"])
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a string array")
	}
	if len(required) != 1 || required[0] != "task" {
		t.Errorf("Required should be ['task'], got: %v", required)
	}
}

// TestSubagentTool_SetContext verifies context setting
func TestSubagentTool_SetContext(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	tool.SetContext("test-channel", "test-chat")

	// Verify context is set (we can't directly access private fields,
	// but we can verify it doesn't crash)
	// The actual context usage is tested in Execute tests
}

// TestSubagentTool_Execute_Success tests successful execution
func TestSubagentTool_Execute_Success(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)
	tool.SetContext("telegram", "chat-123")

	ctx := context.Background()
	args := map[string]any{
		"task":  "Write a haiku about coding",
		"label": "haiku-task",
	}

	result := tool.Execute(ctx, args)

	// Verify basic ToolResult structure
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Verify no error
	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.ForLLM)
	}

	// Verify not async
	if result.Async {
		t.Error("SubagentTool should be synchronous, not async")
	}

	// Verify not silent
	if result.Silent {
		t.Error("SubagentTool should not be silent")
	}

	// Verify ForUser contains brief summary (not empty)
	if result.ForUser == "" {
		t.Error("ForUser should contain result summary")
	}
	if !strings.Contains(result.ForUser, "Task completed") {
		t.Errorf("ForUser should contain task completion, got: %s", result.ForUser)
	}

	// Verify ForLLM contains full details
	if result.ForLLM == "" {
		t.Error("ForLLM should contain full details")
	}
	if !strings.Contains(result.ForLLM, "haiku-task") {
		t.Errorf("ForLLM should contain label 'haiku-task', got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Task completed:") {
		t.Errorf("ForLLM should contain task result, got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_NoLabel tests execution without label
func TestSubagentTool_Execute_NoLabel(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]any{
		"task": "Test task without label",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success without label, got error: %s", result.ForLLM)
	}

	// ForLLM should show (unnamed) for missing label
	if !strings.Contains(result.ForLLM, "(unnamed)") {
		t.Errorf("ForLLM should show '(unnamed)' for missing label, got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_MissingTask tests error handling for missing task
func TestSubagentTool_Execute_MissingTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]any{
		"label": "test",
	}

	result := tool.Execute(ctx, args)

	// Should return error
	if !result.IsError {
		t.Error("Expected error for missing task parameter")
	}

	// ForLLM should contain helpful error with example
	if !strings.Contains(result.ForLLM, `"task"`) {
		t.Errorf("Error message should mention '\"task\"', got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Example") {
		t.Errorf("Error message should include usage example, got: %s", result.ForLLM)
	}

	// Err should be set
	if result.Err == nil {
		t.Error("Err should be set for validation failure")
	}
}

// TestSubagentTool_Execute_NilManager tests error handling for nil manager
func TestSubagentTool_Execute_NilManager(t *testing.T) {
	tool := NewSubagentTool(nil)

	ctx := context.Background()
	args := map[string]any{
		"task": "test task",
	}

	result := tool.Execute(ctx, args)

	// Should return error
	if !result.IsError {
		t.Error("Expected error for nil manager")
	}

	if !strings.Contains(result.ForLLM, "not available in this session") {
		t.Errorf("Error message should mention 'not available in this session', got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_ContextPassing verifies context is properly used
func TestSubagentTool_Execute_ContextPassing(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	// Set context
	channel := "test-channel"
	chatID := "test-chat"
	tool.SetContext(channel, chatID)

	ctx := context.Background()
	args := map[string]any{
		"task": "Test context passing",
	}

	result := tool.Execute(ctx, args)

	// Should succeed
	if result.IsError {
		t.Errorf("Expected success with context, got error: %s", result.ForLLM)
	}

	// The context is used internally; we can't directly test it
	// but execution success indicates context was handled properly
}

// TestSubagentTool_ForUserTruncation verifies long content is truncated for user
func TestSubagentTool_ForUserTruncation(t *testing.T) {
	// Create a mock provider that returns very long content
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})
	tool := NewSubagentTool(manager)

	ctx := context.Background()

	// Create a task that will generate long response
	longTask := strings.Repeat("This is a very long task description. ", 100)
	args := map[string]any{
		"task":  longTask,
		"label": "long-test",
	}

	result := tool.Execute(ctx, args)

	// ForUser should be truncated to 500 chars + "..."
	maxUserLen := 500
	if len(result.ForUser) > maxUserLen+3 { // +3 for "..."
		t.Errorf("ForUser should be truncated to ~%d chars, got: %d", maxUserLen, len(result.ForUser))
	}

	// ForLLM should have full content
	if !strings.Contains(result.ForLLM, longTask[:50]) {
		t.Error("ForLLM should contain reference to original task")
	}
}

func TestFormatToolStats(t *testing.T) {
	tests := []struct {
		name  string
		stats map[string]int
		want  string
	}{
		{"empty", map[string]int{}, ""},
		{"single", map[string]int{"exec": 3}, "exec:3"},
		{"multiple sorted", map[string]int{"read_file": 5, "exec": 3, "write_file": 1}, "exec:3,read_file:5,write_file:1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolStats(tt.stats)
			if got != tt.want {
				t.Errorf("formatToolStats(%v) = %q, want %q", tt.stats, got, tt.want)
			}
		})
	}
}

// TestSubagentManager_Spawn_SetsMetadata verifies that the bus message from a
// completed spawn includes execution statistics in Metadata.
func TestSubagentManager_Spawn_SetsMetadata(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	mgr := NewSubagentManager(provider, "test-model", "/tmp/test", msgBus, orch.Noop, WebSearchToolOptions{})

	_, err := mgr.Spawn(
		context.Background(),
		"say hello", "meta-test", "", "cli", "direct", "",
		nil,
	)
	if err != nil {
		t.Fatalf("Spawn() error: %v", err)
	}

	// Consume the inbound message from the bus
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	received, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("timed out waiting for bus message")
	}

	if received.Channel != "system" {
		t.Fatalf("expected channel 'system', got %q", received.Channel)
	}
	if received.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if received.Metadata["iterations"] != "1" {
		t.Errorf("iterations = %q, want %q", received.Metadata["iterations"], "1")
	}
	if received.Metadata["tool_calls"] != "0" {
		t.Errorf("tool_calls = %q, want %q", received.Metadata["tool_calls"], "0")
	}
	// duration_ms should be a non-negative number
	if received.Metadata["duration_ms"] == "" {
		t.Error("duration_ms should be present")
	}
}

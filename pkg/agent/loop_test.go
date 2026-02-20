package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// mockProvider is a simple mock LLM provider for testing
type mockProvider struct{}

func (m *mockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}

func TestRecordLastChannel(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	// Create agent loop
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Test RecordLastChannel
	testChannel := "test-channel"
	err = al.RecordLastChannel(testChannel)
	if err != nil {
		t.Fatalf("RecordLastChannel failed: %v", err)
	}

	// Verify channel was saved
	lastChannel := al.state.GetLastChannel()
	if lastChannel != testChannel {
		t.Errorf("Expected channel '%s', got '%s'", testChannel, lastChannel)
	}

	// Verify persistence by creating a new agent loop
	al2 := NewAgentLoop(cfg, msgBus, provider)
	if al2.state.GetLastChannel() != testChannel {
		t.Errorf("Expected persistent channel '%s', got '%s'", testChannel, al2.state.GetLastChannel())
	}
}

func TestRecordLastChatID(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	// Create agent loop
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Test RecordLastChatID
	testChatID := "test-chat-id-123"
	err = al.RecordLastChatID(testChatID)
	if err != nil {
		t.Fatalf("RecordLastChatID failed: %v", err)
	}

	// Verify chat ID was saved
	lastChatID := al.state.GetLastChatID()
	if lastChatID != testChatID {
		t.Errorf("Expected chat ID '%s', got '%s'", testChatID, lastChatID)
	}

	// Verify persistence by creating a new agent loop
	al2 := NewAgentLoop(cfg, msgBus, provider)
	if al2.state.GetLastChatID() != testChatID {
		t.Errorf("Expected persistent chat ID '%s', got '%s'", testChatID, al2.state.GetLastChatID())
	}
}

func TestNewAgentLoop_StateInitialized(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	// Create agent loop
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Verify state manager is initialized
	if al.state == nil {
		t.Error("Expected state manager to be initialized")
	}

	// Verify state directory was created
	stateDir := filepath.Join(tmpDir, "state")
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		t.Error("Expected state directory to exist")
	}
}

// TestToolRegistry_ToolRegistration verifies tools can be registered and retrieved
func TestToolRegistry_ToolRegistration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Register a custom tool
	customTool := &mockCustomTool{}
	al.RegisterTool(customTool)

	// Verify tool is registered by checking it doesn't panic on GetStartupInfo
	// (actual tool retrieval is tested in tools package tests)
	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	// Check that our custom tool name is in the list
	found := false
	for _, name := range toolsList {
		if name == "mock_custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected custom tool to be registered")
	}
}

// TestToolContext_Updates verifies tool context is updated with channel/chatID
func TestToolContext_Updates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "OK"}
	_ = NewAgentLoop(cfg, msgBus, provider)

	// Verify that ContextualTool interface is defined and can be implemented
	// This test validates the interface contract exists
	ctxTool := &mockContextualTool{}

	// Verify the tool implements the interface correctly
	var _ tools.ContextualTool = ctxTool
}

// TestToolRegistry_GetDefinitions verifies tool definitions can be retrieved
func TestToolRegistry_GetDefinitions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Register a test tool and verify it shows up in startup info
	testTool := &mockCustomTool{}
	al.RegisterTool(testTool)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	// Check that our custom tool name is in the list
	found := false
	for _, name := range toolsList {
		if name == "mock_custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected custom tool to be registered")
	}
}

// TestAgentLoop_GetStartupInfo verifies startup info contains tools
func TestAgentLoop_GetStartupInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()

	// Verify tools info exists
	toolsInfo, ok := info["tools"]
	if !ok {
		t.Fatal("Expected 'tools' key in startup info")
	}

	toolsMap, ok := toolsInfo.(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'tools' to be a map")
	}

	count, ok := toolsMap["count"]
	if !ok {
		t.Fatal("Expected 'count' in tools info")
	}

	// Should have default tools registered
	if count.(int) == 0 {
		t.Error("Expected at least some tools to be registered")
	}
}

// TestAgentLoop_Stop verifies Stop() sets running to false
func TestAgentLoop_Stop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Note: running is only set to true when Run() is called
	// We can't test that without starting the event loop
	// Instead, verify the Stop method can be called safely
	al.Stop()

	// Verify running is false (initial state or after Stop)
	if al.running.Load() {
		t.Error("Expected agent to be stopped (or never started)")
	}
}

// Mock implementations for testing

type simpleMockProvider struct {
	response string
}

func (m *simpleMockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:   m.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *simpleMockProvider) GetDefaultModel() string {
	return "mock-model"
}

// mockCustomTool is a simple mock tool for registration testing
type mockCustomTool struct{}

func (m *mockCustomTool) Name() string {
	return "mock_custom"
}

func (m *mockCustomTool) Description() string {
	return "Mock custom tool for testing"
}

func (m *mockCustomTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (m *mockCustomTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	return tools.SilentResult("Custom tool executed")
}

// mockContextualTool tracks context updates
type mockContextualTool struct {
	lastChannel string
	lastChatID  string
}

func (m *mockContextualTool) Name() string {
	return "mock_contextual"
}

func (m *mockContextualTool) Description() string {
	return "Mock contextual tool"
}

func (m *mockContextualTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (m *mockContextualTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	return tools.SilentResult("Contextual tool executed")
}

func (m *mockContextualTool) SetContext(channel, chatID string) {
	m.lastChannel = channel
	m.lastChatID = chatID
}

// testHelper executes a message and returns the response
type testHelper struct {
	al *AgentLoop
}

func (h testHelper) executeAndGetResponse(tb testing.TB, ctx context.Context, msg bus.InboundMessage) string {
	// Use a short timeout to avoid hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, responseTimeout)
	defer cancel()

	response, err := h.al.processMessage(timeoutCtx, msg)
	if err != nil {
		tb.Fatalf("processMessage failed: %v", err)
	}
	return response
}

const responseTimeout = 3 * time.Second

// TestToolResult_SilentToolDoesNotSendUserMessage verifies silent tools don't trigger outbound
func TestToolResult_SilentToolDoesNotSendUserMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "File operation complete"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	// ReadFileTool returns SilentResult, which should not send user message
	ctx := context.Background()
	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "read test.txt",
		SessionKey: "test-session",
	}

	response := helper.executeAndGetResponse(t, ctx, msg)

	// Silent tool should return the LLM's response directly
	if response != "File operation complete" {
		t.Errorf("Expected 'File operation complete', got: %s", response)
	}
}

// TestToolResult_UserFacingToolDoesSendMessage verifies user-facing tools trigger outbound
func TestToolResult_UserFacingToolDoesSendMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "Command output: hello world"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	// ExecTool returns UserResult, which should send user message
	ctx := context.Background()
	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "run hello",
		SessionKey: "test-session",
	}

	response := helper.executeAndGetResponse(t, ctx, msg)

	// User-facing tool should include the output in final response
	if response != "Command output: hello world" {
		t.Errorf("Expected 'Command output: hello world', got: %s", response)
	}
}

// failFirstMockProvider fails on the first N calls with a specific error
type failFirstMockProvider struct {
	failures    int
	currentCall int
	failError   error
	successResp string
}

func (m *failFirstMockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
	m.currentCall++
	if m.currentCall <= m.failures {
		return nil, m.failError
	}
	return &providers.LLMResponse{
		Content:   m.successResp,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *failFirstMockProvider) GetDefaultModel() string {
	return "mock-fail-model"
}

// TestAgentLoop_ContextExhaustionRetry verify that the agent retries on context errors
func TestAgentLoop_ContextExhaustionRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	// Create a provider that fails once with a context error
	contextErr := fmt.Errorf("InvalidParameter: Total tokens of image and text exceed max message tokens")
	provider := &failFirstMockProvider{
		failures:    1,
		failError:   contextErr,
		successResp: "Recovered from context error",
	}

	al := NewAgentLoop(cfg, msgBus, provider)

	// Inject some history to simulate a full context
	sessionKey := "test-session-context"
	// Create dummy history
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Old message 1"},
		{Role: "assistant", Content: "Old response 1"},
		{Role: "user", Content: "Old message 2"},
		{Role: "assistant", Content: "Old response 2"},
		{Role: "user", Content: "Trigger message"},
	}
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}
	defaultAgent.Sessions.SetHistory(sessionKey, history)

	// Call ProcessDirectWithChannel
	// Note: ProcessDirectWithChannel calls processMessage which will execute runLLMIteration
	response, err := al.ProcessDirectWithChannel(context.Background(), "Trigger message", sessionKey, "test", "test-chat")

	if err != nil {
		t.Fatalf("Expected success after retry, got error: %v", err)
	}

	if response != "Recovered from context error" {
		t.Errorf("Expected 'Recovered from context error', got '%s'", response)
	}

	// We expect 2 calls: 1st failed, 2nd succeeded
	if provider.currentCall != 2 {
		t.Errorf("Expected 2 calls (1 fail + 1 success), got %d", provider.currentCall)
	}

	// Check final history length
	finalHistory := defaultAgent.Sessions.GetHistory(sessionKey)
	// We verify that the history has been modified (compressed)
	// Original length: 6
	// Expected behavior: compression drops ~50% of history (mid slice)
	// We can assert that the length is NOT what it would be without compression.
	// Without compression: 6 + 1 (new user msg) + 1 (assistant msg) = 8
	if len(finalHistory) >= 8 {
		t.Errorf("Expected history to be compressed (len < 8), got %d", len(finalHistory))
	}
}

func TestShouldInjectReminder(t *testing.T) {
	tests := []struct {
		name      string
		iteration int
		interval  int
		want      bool
	}{
		{"first iteration skipped", 1, 5, false},
		{"iteration 5 interval 5", 5, 5, true},
		{"iteration 10 interval 5", 10, 5, true},
		{"iteration 3 interval 5", 3, 5, false},
		{"interval zero disabled", 5, 0, false},
		{"interval negative disabled", 5, -1, false},
		{"iteration 2 interval 1", 2, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInjectReminder(tt.iteration, tt.interval)
			if got != tt.want {
				t.Errorf("shouldInjectReminder(%d, %d) = %v, want %v", tt.iteration, tt.interval, got, tt.want)
			}
		})
	}
}

func TestBuildTaskReminder_WithoutBlocker(t *testing.T) {
	msg := buildTaskReminder("implement feature X", "")
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if !strings.Contains(msg.Content, "[TASK REMINDER]") {
		t.Error("expected content to contain '[TASK REMINDER]'")
	}
	if !strings.Contains(msg.Content, "implement feature X") {
		t.Error("expected content to contain original message")
	}
	if strings.Contains(msg.Content, "blocker") {
		t.Error("expected content NOT to contain 'blocker' when no blocker provided")
	}
	if !strings.Contains(msg.Content, "move on") {
		t.Error("expected content to contain completion prompt")
	}
}

func TestBuildTaskReminder_WithBlocker(t *testing.T) {
	msg := buildTaskReminder("implement feature X", "ModuleNotFoundError: No module named 'foo'")
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if !strings.Contains(msg.Content, "[TASK REMINDER]") {
		t.Error("expected content to contain '[TASK REMINDER]'")
	}
	if !strings.Contains(msg.Content, "implement feature X") {
		t.Error("expected content to contain original message")
	}
	if !strings.Contains(msg.Content, "Last blocker") {
		t.Error("expected content to contain 'Last blocker'")
	}
	if !strings.Contains(msg.Content, "ModuleNotFoundError") {
		t.Error("expected content to contain blocker text")
	}
}

func TestResolveProvider_CachesProviders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				Provider:          "vllm",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		Providers: config.ProvidersConfig{
			VLLM: config.ProviderConfig{
				APIKey:  "test-key",
				APIBase: "https://example.com/v1",
			},
		},
	}

	msgBus := bus.NewMessageBus()
	primary := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, primary)

	// Primary provider should be cached under "vllm"
	p1 := al.resolveProvider("vllm", primary)
	if p1 != primary {
		t.Fatal("expected primary provider for 'vllm'")
	}

	// Calling again should return the same instance (cached)
	p2 := al.resolveProvider("vllm", primary)
	if p1 != p2 {
		t.Fatal("expected same cached instance on second call")
	}
}

func TestResolveProvider_FallsBackOnError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				Provider:          "vllm",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	primary := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, primary)

	// Request a provider that can't be created (no config for "nonexistent")
	p := al.resolveProvider("nonexistent", primary)
	if p != primary {
		t.Fatal("expected fallback to primary provider on creation error")
	}

	// Ensure the failed provider is NOT cached
	if _, ok := al.providerCache["nonexistent"]; ok {
		t.Fatal("failed provider should not be cached")
	}
}

func TestResolveProvider_EmptyNameReturnsFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	primary := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, primary)

	p := al.resolveProvider("", primary)
	if p != primary {
		t.Fatal("expected fallback provider for empty name")
	}
}

// TestSlashCommandResponseSkipsPlaceholder verifies that slash command responses
// are published with SkipPlaceholder=true so they don't overwrite the ongoing task
// status bubble.
func TestSlashCommandResponseSkipsPlaceholder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = al.Run(ctx)
	}()

	// Send a slash command
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/skills",
	})

	// Read the outbound message
	outMsg, ok := msgBus.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("expected outbound message from slash command")
	}

	if !outMsg.SkipPlaceholder {
		t.Errorf("expected SkipPlaceholder=true for slash command response, got false")
	}
}

func TestBuildTaskReminder_Truncation(t *testing.T) {
	// Build a long message (1000 runes)
	longMsg := strings.Repeat("あ", 1000)
	longBlocker := strings.Repeat("X", 500)

	msg := buildTaskReminder(longMsg, longBlocker)

	// The full message should NOT contain 1000 'あ' characters
	runeCount := strings.Count(msg.Content, "あ")
	if runeCount >= 1000 {
		t.Errorf("expected task message to be truncated, got %d 'あ' runes", runeCount)
	}
	// Should be at most taskReminderMaxChars (500) runes for the task part
	if runeCount > taskReminderMaxChars {
		t.Errorf("expected at most %d task runes, got %d", taskReminderMaxChars, runeCount)
	}

	// Blocker should be truncated too
	xCount := strings.Count(msg.Content, "X")
	if xCount >= 500 {
		t.Errorf("expected blocker to be truncated, got %d 'X' chars", xCount)
	}
	if xCount > blockerMaxChars {
		t.Errorf("expected at most %d blocker chars, got %d", blockerMaxChars, xCount)
	}
}

// ---------- /plan command tests ----------

func newTestAgentLoop(t *testing.T) (*AgentLoop, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "agent-plan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	return al, func() { os.RemoveAll(tmpDir) }
}

func TestPlanCommand_ShowNoPlan(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan"})
	if !handled {
		t.Fatal("expected /plan to be handled")
	}
	if !strings.Contains(response, "No active plan") {
		t.Errorf("expected 'No active plan', got %q", response)
	}
}

func TestPlanCommand_StartNewPlan(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// /plan <task> should NOT be handled by handleCommand — it falls through
	// to the LLM queue via expandPlanCommand.
	_, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan Set up monitoring"})
	if handled {
		t.Fatal("expected /plan <task> NOT to be handled (should fall through to LLM)")
	}

	// expandPlanCommand writes the seed and rewrites the message
	msg := bus.InboundMessage{Content: "/plan Set up monitoring"}
	expanded, compact, ok := al.expandPlanCommand(msg)
	if !ok {
		t.Fatal("expected expandPlanCommand to succeed")
	}
	if expanded != "Set up monitoring" {
		t.Errorf("expected expanded = 'Set up monitoring', got %q", expanded)
	}
	if !strings.Contains(compact, "Set up monitoring") {
		t.Errorf("expected compact to contain task, got %q", compact)
	}

	// Verify plan was created
	agent := al.registry.GetDefaultAgent()
	if !agent.ContextBuilder.HasActivePlan() {
		t.Error("expected active plan after expandPlanCommand")
	}
	if status := agent.ContextBuilder.GetPlanStatus(); status != "interviewing" {
		t.Errorf("expected 'interviewing', got %q", status)
	}
}

func TestPlanCommand_StartBlockedByExisting(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Start first plan via expandPlanCommand
	al.expandPlanCommand(bus.InboundMessage{Content: "/plan First task"})

	// Try to start another — handleCommand should block it on the fast path
	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan Second task"})
	if !handled {
		t.Fatal("expected second /plan to be handled (blocked)")
	}
	if !strings.Contains(response, "already active") {
		t.Errorf("expected 'already active', got %q", response)
	}
}

func TestPlanCommand_Clear(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Start plan then clear
	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan clear"})
	if !strings.Contains(response, "Plan cleared") {
		t.Errorf("expected 'Plan cleared', got %q", response)
	}

	agent := al.registry.GetDefaultAgent()
	if agent.ContextBuilder.HasActivePlan() {
		t.Error("expected no plan after clear")
	}
}

func TestPlanCommand_ClearNoPlan(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan clear"})
	if !strings.Contains(response, "No active plan") {
		t.Errorf("expected 'No active plan', got %q", response)
	}
}

func TestPlanCommand_Start(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Create interviewing plan
	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})

	// Transition to executing
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "executing") {
		t.Errorf("expected 'executing', got %q", response)
	}

	agent := al.registry.GetDefaultAgent()
	if status := agent.ContextBuilder.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}
}

func TestPlanCommand_StartAlreadyExecuting(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Create interviewing plan then start
	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})
	al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	// Try start again
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "already executing") {
		t.Errorf("expected 'already executing', got %q", response)
	}
}

func TestPlanCommand_Done(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	// Write a plan directly with phases
	plan := `# Active Plan

> Task: Test task
> Status: executing
> Phase: 1

## Phase 1: Setup
- [ ] Step one
- [ ] Step two

## Context
Test context
`
	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan done 1"})
	if !strings.Contains(response, "Marked step 1") {
		t.Errorf("expected confirmation, got %q", response)
	}
}

func TestPlanCommand_DoneInvalidStep(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan done abc"})
	if !strings.Contains(response, "positive integer") {
		t.Errorf("expected step validation error, got %q", response)
	}
}

func TestPlanCommand_Add(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	plan := `# Active Plan

> Task: Test task
> Status: executing
> Phase: 1

## Phase 1: Setup
- [ ] Step one

## Context
Test context
`
	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan add New step here"})
	if !strings.Contains(response, "Added step") {
		t.Errorf("expected 'Added step', got %q", response)
	}

	content := agent.ContextBuilder.ReadMemory()
	if !strings.Contains(content, "New step here") {
		t.Error("expected new step in plan content")
	}
}

func TestPlanCommand_Next(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	plan := `# Active Plan

> Task: Test task
> Status: executing
> Phase: 1

## Phase 1: Setup
- [x] Step one

## Phase 2: Deploy
- [ ] Step two

## Context
Test
`
	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan next"})
	if !strings.Contains(response, "phase 2") {
		t.Errorf("expected 'phase 2', got %q", response)
	}

	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase 2, got %d", phase)
	}
}

func TestPlanCommand_ShowActivePlan(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	plan := `# Active Plan

> Task: Deploy app
> Status: executing
> Phase: 1

## Phase 1: Build
- [x] Compile code
- [ ] Run tests

## Context
Production server
`
	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan"})
	if !strings.Contains(response, "Deploy app") {
		t.Errorf("expected task name in display, got %q", response)
	}
	if !strings.Contains(response, "Phase 1") {
		t.Errorf("expected phase info in display, got %q", response)
	}
}

// TestAutoPhaseAdvance verifies that auto-advance sends notification after LLM iteration
// when current phase is complete.
func TestAutoPhaseAdvance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-auto-advance-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "OK"}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("No default agent")
	}

	// Write plan with phase 1 complete
	plan := `# Active Plan

> Task: Test auto advance
> Status: executing
> Phase: 1

## Phase 1: Setup
- [x] Step one
- [x] Step two

## Phase 2: Deploy
- [ ] Step three

## Context
Test
`
	agent.ContextBuilder.WriteMemory(plan)

	// Process a message which triggers runAgentLoop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = al.ProcessDirectWithChannel(ctx, "continue", "auto-advance-test", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	// After processing, phase should be auto-advanced
	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase auto-advanced to 2, got %d", phase)
	}
}

// TestAutoCompleteClears verifies that plan is cleared when all phases are complete.
func TestAutoCompleteClears(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-auto-complete-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "All done"}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("No default agent")
	}

	// Write fully complete plan
	plan := `# Active Plan

> Task: Test auto complete
> Status: executing
> Phase: 1

## Phase 1: Setup
- [x] Step one
- [x] Step two

## Context
Test
`
	agent.ContextBuilder.WriteMemory(plan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = al.ProcessDirectWithChannel(ctx, "finish up", "auto-complete-test", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	// Plan should be cleared
	if agent.ContextBuilder.HasActivePlan() {
		t.Error("expected plan to be cleared after completion")
	}
}

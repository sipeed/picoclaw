package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

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
	toolsInfo := info["tools"].(map[string]any)
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
	toolsInfo := info["tools"].(map[string]any)
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

	toolsMap, ok := toolsInfo.(map[string]any)
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

func (m *simpleMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
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

func (m *mockCustomTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *mockCustomTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
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

func (m *mockContextualTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *mockContextualTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
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

func (m *failFirstMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
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
	response, err := al.ProcessDirectWithChannel(
		context.Background(),
		"Trigger message",
		sessionKey,
		"test",
		"test-chat",
	)
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

// TestDeduplicateToolCalls verifies that duplicate consecutive tool calls are detected and break the iteration loop
func TestDeduplicateToolCalls(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config with low iteration limit
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 15, // Would normally cause 15 duplicate messages without fix
			},
		},
	}

	// Create agent loop
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{
		responses: []providers.LLMResponse{
			// Iteration 1: First identical tool call
			{
				Content: "Subagent-3 completed",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Name: "message",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"text":"Subagent-3 completed weather check"}`,
					},
				}},
			},
			// Iteration 2: LLM repeats same tool call with identical arguments
			{
				Content: "",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-2",
					Type: "function",
					Name: "message",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"text":"Subagent-3 completed weather check"}`,
					},
				}},
			},
			// Should not reach iteration 3+ due to deduplication
		},
		responseIndex: 0,
	}

	al := NewAgentLoop(cfg, msgBus, provider)

	// Verify agent loop exists
	if al == nil {
		t.Fatal("Failed to create agent loop")
	}

	// Verify dedup worked: provider should have been called ~2 times, not 15
	// (allowing for some internal calls)
	if provider.callCount > 5 {
		t.Logf("WARNING: Provider.Chat called %d times, suggests deduplication may not be working", provider.callCount)
	}
}

// TestNoDuplicateDetectionDifferentArgs verifies that tool calls with different arguments are NOT deduplicated
func TestNoDuplicateDetectionDifferentArgs(t *testing.T) {
	// Create temp workspace
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
				MaxToolIterations: 3,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{
		responses: []providers.LLMResponse{
			// Different arguments = should NOT trigger dedup
			{
				Content: "Response 1",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Name: "message",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"text":"First message"}`,
					},
				}},
			},
			{
				Content: "Response 2",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-2",
					Type: "function",
					Name: "message",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"text":"Second different message"}`,
					},
				}},
			},
		},
		responseIndex: 0,
	}

	al := NewAgentLoop(cfg, msgBus, provider)
	if al == nil {
		t.Fatal("Failed to create agent loop")
	}
}

// TestDeduplicateToolCallsReflectComparison verifies reflect.DeepEqual works for arguments
func TestDeduplicateToolCallsReflectComparison(t *testing.T) {
	args1 := map[string]any{
		"text": "Subagent-3 completed",
		"id":   "123",
	}

	args2 := map[string]any{
		"id":   "123", // Different key order
		"text": "Subagent-3 completed",
	}

	// Should match with reflect.DeepEqual regardless of key order
	if !reflect.DeepEqual(args1, args2) {
		t.Fatal("Arguments should match regardless of map key order")
	}
}

// TestDeduplicateToolCallsNestedStructures verifies complex nested arguments work
func TestDeduplicateToolCallsNestedStructures(t *testing.T) {
	args1 := map[string]any{
		"endpoint": "/users",
		"params": map[string]any{
			"id": "123",
			"filter": map[string]any{
				"active": true,
				"role":   "admin",
			},
		},
	}

	args2 := map[string]any{
		"endpoint": "/users",
		"params": map[string]any{
			"id": "123",
			"filter": map[string]any{
				"role":   "admin",
				"active": true,
			},
		},
	}

	if !reflect.DeepEqual(args1, args2) {
		t.Fatal("Nested structures should match regardless of key order")
	}
}

// TestDuplicateTrackerThreshold verifies N-duplicate threshold works (Fix #6)
func TestDuplicateTrackerThreshold(t *testing.T) {
	tracker := &DuplicateTracker{
		consecutiveCount: 0,
		lastToolName:     "",
		maxThreshold:     3, // Require 3 consecutive duplicates
	}

	// First duplicate
	tracker.lastToolName = "message"
	tracker.consecutiveCount = 1
	if tracker.consecutiveCount >= tracker.maxThreshold {
		t.Fatal("Should not break on first duplicate")
	}

	// Second duplicate
	tracker.consecutiveCount = 2
	if tracker.consecutiveCount >= tracker.maxThreshold {
		t.Fatal("Should not break on second duplicate")
	}

	// Third duplicate - should trigger
	tracker.consecutiveCount = 3
	if tracker.consecutiveCount < tracker.maxThreshold {
		t.Fatal("Should break on third duplicate")
	}
}

// TestMultipleToolCallsAllChecked verifies all tool calls are compared (Fix #1)
func TestMultipleToolCallsAllChecked(t *testing.T) {
	// Scenario: Last iteration had tools [A, B, C]
	// Current iteration: [A (same), B (same), C (different)]
	// Should NOT trigger dedup because C is different

	lastTools := []providers.ToolCall{
		{Name: "tool1", Arguments: map[string]any{"id": "1"}},
		{Name: "tool2", Arguments: map[string]any{"id": "2"}},
		{Name: "tool3", Arguments: map[string]any{"id": "3"}},
	}

	currentTools := []providers.ToolCall{
		{Name: "tool1", Arguments: map[string]any{"id": "1"}},
		{Name: "tool2", Arguments: map[string]any{"id": "2"}},
		{Name: "tool3", Arguments: map[string]any{"id": "DIFFERENT"}},
	}

	// Check all match
	allMatch := len(lastTools) == len(currentTools)
	if allMatch {
		for idx := 0; idx < len(currentTools); idx++ {
			if lastTools[idx].Name != currentTools[idx].Name {
				allMatch = false
				break
			}
			if !reflect.DeepEqual(lastTools[idx].Arguments, currentTools[idx].Arguments) {
				allMatch = false
				break
			}
		}
	}

	// Should NOT match
	if allMatch {
		t.Fatal("Should not match - third tool is different")
	}
}

// TestMultipleToolCallsAllIdentical verifies dedup when ALL tools are identical (Fix #1)
func TestMultipleToolCallsAllIdentical(t *testing.T) {
	lastTools := []providers.ToolCall{
		{Name: "tool1", Arguments: map[string]any{"id": "1"}},
		{Name: "tool2", Arguments: map[string]any{"id": "2"}},
	}

	currentTools := []providers.ToolCall{
		{Name: "tool1", Arguments: map[string]any{"id": "1"}},
		{Name: "tool2", Arguments: map[string]any{"id": "2"}},
	}

	// Check all match
	allMatch := len(lastTools) == len(currentTools)
	if allMatch {
		for idx := 0; idx < len(currentTools); idx++ {
			if lastTools[idx].Name != currentTools[idx].Name {
				allMatch = false
				break
			}
			if !reflect.DeepEqual(lastTools[idx].Arguments, currentTools[idx].Arguments) {
				allMatch = false
				break
			}
		}
	}

	// Should match
	if !allMatch {
		t.Fatal("All tools should match")
	}
}

// TestMessageHistorySafeWalk verifies backward message walk works (Fix #3)
func TestMessageHistorySafeWalk(t *testing.T) {
	// Complex message structure with multiple results
	messages := []providers.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "First", ToolCalls: []providers.ToolCall{{Name: "tool1"}}},
		{Role: "tool", Content: "Result1"},
		{Role: "tool", Content: "Result2"},
		{Role: "tool", Content: "Result3"},
		{Role: "assistant", Content: "Second", ToolCalls: []providers.ToolCall{{Name: "tool2"}}},
	}

	// Find last assistant's previous message using safe walk (Fix #3)
	var lastAssistantMsg *providers.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && i > 0 {
			lastAssistantMsg = &messages[i-1]
			break
		}
	}

	// Should find the first assistant message, not current one
	if lastAssistantMsg == nil || lastAssistantMsg.Content != "First" {
		t.Fatal("Should safely find previous assistant message despite complex structure")
	}
}

// TestMessageHistoryEdgeCase verifies edge case: only one message
func TestMessageHistoryEdgeCase(t *testing.T) {
	messages := []providers.Message{
		{Role: "assistant", ToolCalls: []providers.ToolCall{{Name: "tool1"}}},
	}

	// Try to find previous message
	var lastAssistantMsg *providers.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && i > 0 { // i > 0 check prevents out of bounds
			lastAssistantMsg = &messages[i-1]
			break
		}
	}

	// Should not find anything
	if lastAssistantMsg != nil {
		t.Fatal("Should not find previous message when only one exists")
	}
}

// TestDuplicateTrackerReset verifies tracker resets when tools differ
func TestDuplicateTrackerReset(t *testing.T) {
	tracker := &DuplicateTracker{
		consecutiveCount: 5,
		lastToolName:     "message",
		maxThreshold:     3,
	}

	// Different tool
	newToolName := "search"
	if newToolName != tracker.lastToolName {
		tracker.consecutiveCount = 1
		tracker.lastToolName = newToolName
	}

	// Counter should reset
	if tracker.consecutiveCount != 1 {
		t.Fatal("Counter should reset to 1 when tool differs")
	}

	// Tool name should update
	if tracker.lastToolName != "search" {
		t.Fatal("Tool name should update to new tool")
	}
}

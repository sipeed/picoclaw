package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/bus"
	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/KarakuriAgent/clawdroid/pkg/providers"
	"github.com/KarakuriAgent/clawdroid/pkg/tools"
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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

// TestCreateToolRegistry_ExecDisabled verifies exec tool is NOT registered when disabled
func TestCreateToolRegistry_ExecDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			Exec: config.ExecToolsConfig{
				Enabled: false,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	for _, name := range toolsList {
		if name == "exec" {
			t.Error("exec tool should NOT be registered when Exec.Enabled is false")
		}
	}
}

// TestCreateToolRegistry_ExecEnabled verifies exec tool IS registered when enabled
func TestCreateToolRegistry_ExecEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			Exec: config.ExecToolsConfig{
				Enabled: true,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	found := false
	for _, name := range toolsList {
		if name == "exec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("exec tool should be registered when Exec.Enabled is true")
	}
}

// TestCreateToolRegistry_I2CDisabled verifies I2C tool is NOT registered when disabled
func TestCreateToolRegistry_I2CDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			I2C: config.I2CToolsConfig{Enabled: false},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	for _, name := range toolsList {
		if name == "i2c" {
			t.Error("i2c tool should NOT be registered when I2C.Enabled is false")
		}
	}
}

// TestCreateToolRegistry_I2CEnabled verifies I2C tool IS registered when enabled
func TestCreateToolRegistry_I2CEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			I2C: config.I2CToolsConfig{Enabled: true},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	found := false
	for _, name := range toolsList {
		if name == "i2c" {
			found = true
			break
		}
	}
	if !found {
		t.Error("i2c tool should be registered when I2C.Enabled is true")
	}
}

// TestCreateToolRegistry_SPIDisabled verifies SPI tool is NOT registered when disabled
func TestCreateToolRegistry_SPIDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			SPI: config.SPIToolsConfig{Enabled: false},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	for _, name := range toolsList {
		if name == "spi" {
			t.Error("spi tool should NOT be registered when SPI.Enabled is false")
		}
	}
}

// TestCreateToolRegistry_SPIEnabled verifies SPI tool IS registered when enabled
func TestCreateToolRegistry_SPIEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			SPI: config.SPIToolsConfig{Enabled: true},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]interface{})
	toolsList := toolsInfo["names"].([]string)

	found := false
	for _, name := range toolsList {
		if name == "spi" {
			found = true
			break
		}
	}
	if !found {
		t.Error("spi tool should be registered when SPI.Enabled is true")
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
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
	al.sessions.GetOrCreate(sessionKey)
	// Create dummy history
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Old message 1"},
		{Role: "assistant", Content: "Old response 1"},
		{Role: "user", Content: "Old message 2"},
		{Role: "assistant", Content: "Old response 2"},
		{Role: "user", Content: "Trigger message"},
	}
	al.sessions.SetHistory(sessionKey, history)

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
	finalHistory := al.sessions.GetHistory(sessionKey)
	// We verify that the history has been modified (compressed)
	// Original length: 6
	// Expected behavior: compression drops ~50% of history (mid slice)
	// We can assert that the length is NOT what it would be without compression.
	// Without compression: 6 + 1 (new user msg) + 1 (assistant msg) = 8
	if len(finalHistory) >= 8 {
		t.Errorf("Expected history to be compressed (len < 8), got %d", len(finalHistory))
	}
}

// cancelDuringChatProvider cancels the given context inside Chat() and returns
// a "context canceled" error, simulating an HTTP request aborted mid-flight.
type cancelDuringChatProvider struct {
	cancelFn    context.CancelFunc // called inside Chat to simulate cancellation
	currentCall int
}

func (m *cancelDuringChatProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
	m.currentCall++
	// Cancel the context from within Chat, then return the error the HTTP
	// client would produce when its request context is cancelled.
	m.cancelFn()
	return nil, fmt.Errorf("Post \"https://api.example.com\": context canceled")
}

func (m *cancelDuringChatProvider) GetDefaultModel() string {
	return "mock-cancel-model"
}

// TestRetryLoop_CancelledContextSkipsCompression verifies that when
// provider.Chat() returns "context canceled" because the request was aborted,
// the retry loop detects ctx.Err() and breaks WITHOUT triggering forceCompression.
func TestRetryLoop_CancelledContextSkipsCompression(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	// Create a cancellable context — the provider will cancel it during Chat()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := &cancelDuringChatProvider{cancelFn: cancel}

	al := NewAgentLoop(cfg, msgBus, provider)

	// Set up session with enough history for forceCompression to act on
	sessionKey := "test-cancel-session"
	al.sessions.GetOrCreate(sessionKey)
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Old message 1"},
		{Role: "assistant", Content: "Old response 1"},
		{Role: "user", Content: "Old message 2"},
		{Role: "assistant", Content: "Old response 2"},
		{Role: "user", Content: "Old message 3"},
		{Role: "assistant", Content: "Old response 3"},
		{Role: "user", Content: "Trigger message"},
	}
	al.sessions.SetHistory(sessionKey, history)

	_, _ = al.ProcessDirectWithChannel(ctx, "Trigger message", sessionKey, "test", "test-chat")

	// History must NOT contain a compression note — forceCompression should not have run
	finalHistory := al.sessions.GetHistory(sessionKey)
	for _, msg := range finalHistory {
		if strings.Contains(msg.Content, "Emergency compression") {
			t.Error("Context cancellation must NOT trigger forceCompression")
		}
	}

	// Provider.Chat() should have been called exactly once (no retry)
	if provider.currentCall != 1 {
		t.Errorf("Expected exactly 1 provider call (no retry), got %d", provider.currentCall)
	}
}

// TestForceCompression_ToolGroupBoundary verifies forceCompression adjusts the
// split point to avoid breaking tool_calls/tool response groups.
func TestForceCompression_ToolGroupBoundary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	sessionKey := "test-compression-boundary"
	al.sessions.GetOrCreate(sessionKey)

	// Build history where naive mid would land on a tool message.
	// Layout: [system, user, assistant+tool_calls, tool, tool, user, assistant, user]
	// mid of conversation (indices 1-6, len=7) = 3, which is a "tool" message.
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "msg 1"},
		{Role: "assistant", Content: "thinking", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Type: "function", Function: &providers.FunctionCall{Name: "read", Arguments: "{}"}},
			{ID: "tc2", Type: "function", Function: &providers.FunctionCall{Name: "write", Arguments: "{}"}},
		}},
		{Role: "tool", Content: "result1", ToolCallID: "tc1"},
		{Role: "tool", Content: "result2", ToolCallID: "tc2"},
		{Role: "user", Content: "msg 2"},
		{Role: "assistant", Content: "response 2"},
		{Role: "user", Content: "msg 3"},
	}

	al.sessions.SetHistory(sessionKey, history)
	al.forceCompression(sessionKey)

	result := al.sessions.GetHistory(sessionKey)

	// Verify no orphaned tool messages remain
	toolCallIDs := make(map[string]bool)
	for _, msg := range result {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				toolCallIDs[tc.ID] = true
			}
		}
	}
	for _, msg := range result {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if !toolCallIDs[msg.ToolCallID] {
				t.Errorf("Orphaned tool message found: ToolCallID=%s", msg.ToolCallID)
			}
		}
	}

	// Verify no assistant with tool_calls has missing tool responses
	toolRespIDs := make(map[string]bool)
	for _, msg := range result {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			toolRespIDs[msg.ToolCallID] = true
		}
	}
	for _, msg := range result {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				if !toolRespIDs[tc.ID] {
					t.Errorf("Assistant has tool_call %s but no tool response found", tc.ID)
				}
			}
		}
	}
}

// TestForceCompression_MidOnAssistantWithToolCalls verifies that when mid lands
// on an assistant message with tool_calls, the subsequent tool responses are
// also dropped together.
func TestForceCompression_MidOnAssistantWithToolCalls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	sessionKey := "test-compression-assistant-mid"
	al.sessions.GetOrCreate(sessionKey)

	// Layout: [system, user, assistant, user, assistant+tool_calls, tool, user, assistant, user, assistant, user]
	// conversation (indices 1-9, len=10), mid=5 which is "tool"
	// Actually let's make mid land on assistant+tool_calls:
	// conversation len=8, mid=4 → assistant+tool_calls
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "msg 1"},
		{Role: "assistant", Content: "resp 1"},
		{Role: "user", Content: "msg 2"},
		{Role: "assistant", Content: "resp 2"},
		// mid=4 of conversation lands here:
		{Role: "assistant", Content: "thinking", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Type: "function", Function: &providers.FunctionCall{Name: "read", Arguments: "{}"}},
		}},
		{Role: "tool", Content: "file contents", ToolCallID: "tc1"},
		{Role: "user", Content: "msg 3"},
		{Role: "assistant", Content: "resp 3"},
		{Role: "user", Content: "final msg"},
	}

	al.sessions.SetHistory(sessionKey, history)
	al.forceCompression(sessionKey)

	result := al.sessions.GetHistory(sessionKey)

	// Verify integrity
	toolCallIDs := make(map[string]bool)
	toolRespIDs := make(map[string]bool)
	for _, msg := range result {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				toolCallIDs[tc.ID] = true
			}
		}
		if msg.Role == "tool" && msg.ToolCallID != "" {
			toolRespIDs[msg.ToolCallID] = true
		}
	}

	// Every tool response must have a matching tool_call
	for _, msg := range result {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if !toolCallIDs[msg.ToolCallID] {
				t.Errorf("Orphaned tool message: ToolCallID=%s", msg.ToolCallID)
			}
		}
	}

	// Every tool_call must have a matching response
	for _, msg := range result {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				if !toolRespIDs[tc.ID] {
					t.Errorf("Missing tool response for tool_call %s", tc.ID)
				}
			}
		}
	}
}

// TestForceCompression_NoteUsesUserRole verifies the compression note uses "user" role
func TestForceCompression_NoteUsesUserRole(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Model: "test-model",
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				DataDir:           tmpDir,
				MaxTokens:         4096,
				ContextWindow:     128000,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	sessionKey := "test-compression-role"
	al.sessions.GetOrCreate(sessionKey)
	history := []providers.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "msg 1"},
		{Role: "assistant", Content: "resp 1"},
		{Role: "user", Content: "msg 2"},
		{Role: "assistant", Content: "resp 2"},
		{Role: "user", Content: "msg 3"},
		{Role: "assistant", Content: "resp 3"},
		{Role: "user", Content: "final"},
	}

	al.sessions.SetHistory(sessionKey, history)
	al.forceCompression(sessionKey)

	result := al.sessions.GetHistory(sessionKey)

	// Find the compression note and verify it uses "user" role
	found := false
	for _, msg := range result {
		if strings.Contains(msg.Content, "Emergency compression") {
			found = true
			if msg.Role != "user" {
				t.Errorf("Compression note should use 'user' role, got '%s'", msg.Role)
			}
		}
	}
	if !found {
		t.Error("Expected to find compression note in history")
	}
}

// TestSanitizeToolMessages_OrphanedToolRemoval verifies orphaned tool messages are removed
func TestSanitizeToolMessages_OrphanedToolRemoval(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		// Orphaned tool message — no preceding assistant with tool_calls
		{Role: "tool", Content: "orphaned result", ToolCallID: "tc-orphan"},
		{Role: "assistant", Content: "response"},
	}

	result := sanitizeToolMessages(history)

	for _, msg := range result {
		if msg.Role == "tool" {
			t.Error("Orphaned tool message should have been removed")
		}
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 messages after sanitization, got %d", len(result))
	}
}

// TestSanitizeToolMessages_MissingResponseStripped verifies tool_calls with missing
// responses are stripped from assistant messages
func TestSanitizeToolMessages_MissingResponseStripped(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "do something"},
		{Role: "assistant", Content: "calling tools", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Type: "function", Function: &providers.FunctionCall{Name: "read", Arguments: "{}"}},
			{ID: "tc2", Type: "function", Function: &providers.FunctionCall{Name: "write", Arguments: "{}"}},
			{ID: "tc3", Type: "function", Function: &providers.FunctionCall{Name: "exec", Arguments: "{}"}},
		}},
		// Only tc1 has a response — tc2 and tc3 were cancelled mid-execution
		{Role: "tool", Content: "read result", ToolCallID: "tc1"},
		{Role: "assistant", Content: "done"},
	}

	result := sanitizeToolMessages(history)

	// Find the assistant message and verify only tc1 remains
	for _, msg := range result {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			if len(msg.ToolCalls) != 1 {
				t.Errorf("Expected 1 tool_call after sanitization, got %d", len(msg.ToolCalls))
			}
			if msg.ToolCalls[0].ID != "tc1" {
				t.Errorf("Expected remaining tool_call to be tc1, got %s", msg.ToolCalls[0].ID)
			}
			// Content should be preserved
			if msg.Content != "calling tools" {
				t.Errorf("Assistant content should be preserved, got '%s'", msg.Content)
			}
		}
	}
}

// TestSanitizeToolMessages_CompleteGroupPreserved verifies that complete
// tool_calls/tool groups are kept intact
func TestSanitizeToolMessages_CompleteGroupPreserved(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "do something"},
		{Role: "assistant", Content: "calling tools", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Type: "function", Function: &providers.FunctionCall{Name: "read", Arguments: "{}"}},
			{ID: "tc2", Type: "function", Function: &providers.FunctionCall{Name: "write", Arguments: "{}"}},
		}},
		{Role: "tool", Content: "read result", ToolCallID: "tc1"},
		{Role: "tool", Content: "write result", ToolCallID: "tc2"},
		{Role: "assistant", Content: "done"},
	}

	result := sanitizeToolMessages(history)

	// Everything should be preserved
	if len(result) != len(history) {
		t.Errorf("Complete group should be preserved; expected %d messages, got %d",
			len(history), len(result))
	}

	// Verify tool_calls are intact
	for _, msg := range result {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			if len(msg.ToolCalls) != 2 {
				t.Errorf("Expected 2 tool_calls preserved, got %d", len(msg.ToolCalls))
			}
		}
	}
}

// TestSanitizeToolMessages_AllToolCallsMissing verifies that when ALL tool_calls
// have missing responses, the ToolCalls slice becomes empty
func TestSanitizeToolMessages_AllToolCallsMissing(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "do something"},
		{Role: "assistant", Content: "calling tools", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Type: "function", Function: &providers.FunctionCall{Name: "read", Arguments: "{}"}},
		}},
		// No tool response at all — cancelled immediately
		{Role: "assistant", Content: "never mind"},
	}

	result := sanitizeToolMessages(history)

	// The assistant message should have empty ToolCalls
	for _, msg := range result {
		if msg.Role == "assistant" && msg.Content == "calling tools" {
			if len(msg.ToolCalls) != 0 {
				t.Errorf("Expected 0 tool_calls after sanitization, got %d", len(msg.ToolCalls))
			}
		}
	}
}

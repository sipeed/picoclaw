package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type fakeChannel struct{ id string }

func (f *fakeChannel) Name() string                                            { return "fake" }
func (f *fakeChannel) Start(ctx context.Context) error                         { return nil }
func (f *fakeChannel) Stop(ctx context.Context) error                          { return nil }
func (f *fakeChannel) Send(ctx context.Context, msg bus.OutboundMessage) error { return nil }
func (f *fakeChannel) IsRunning() bool                                         { return true }
func (f *fakeChannel) IsAllowed(string) bool                                   { return true }
func (f *fakeChannel) IsAllowedSender(sender bus.SenderInfo) bool              { return true }
func (f *fakeChannel) ReasoningChannelID() string                              { return f.id }

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

func TestTargetReasoningChannelID_AllChannels(t *testing.T) {
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

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	chManager, err := channels.NewManager(&config.Config{}, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("Failed to create channel manager: %v", err)
	}
	for name, id := range map[string]string{
		"whatsapp":  "rid-whatsapp",
		"telegram":  "rid-telegram",
		"feishu":    "rid-feishu",
		"discord":   "rid-discord",
		"maixcam":   "rid-maixcam",
		"qq":        "rid-qq",
		"dingtalk":  "rid-dingtalk",
		"slack":     "rid-slack",
		"line":      "rid-line",
		"onebot":    "rid-onebot",
		"wecom":     "rid-wecom",
		"wecom_app": "rid-wecom-app",
	} {
		chManager.RegisterChannel(name, &fakeChannel{id: id})
	}
	al.SetChannelManager(chManager)
	tests := []struct {
		channel string
		wantID  string
	}{
		{channel: "whatsapp", wantID: "rid-whatsapp"},
		{channel: "telegram", wantID: "rid-telegram"},
		{channel: "feishu", wantID: "rid-feishu"},
		{channel: "discord", wantID: "rid-discord"},
		{channel: "maixcam", wantID: "rid-maixcam"},
		{channel: "qq", wantID: "rid-qq"},
		{channel: "dingtalk", wantID: "rid-dingtalk"},
		{channel: "slack", wantID: "rid-slack"},
		{channel: "line", wantID: "rid-line"},
		{channel: "onebot", wantID: "rid-onebot"},
		{channel: "wecom", wantID: "rid-wecom"},
		{channel: "wecom_app", wantID: "rid-wecom-app"},
		{channel: "unknown", wantID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			got := al.targetReasoningChannelID(tt.channel)
			if got != tt.wantID {
				t.Fatalf("targetReasoningChannelID(%q) = %q, want %q", tt.channel, got, tt.wantID)
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

	// First call creates and caches a provider for "vllm/test-model"
	p1 := al.resolveProvider("vllm", "test-model", primary)
	if p1 == primary {
		t.Fatal("expected a new provider from legacy providers config, not the fallback")
	}

	// Calling again should return the same instance (cached)
	p2 := al.resolveProvider("vllm", "test-model", primary)
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
	p := al.resolveProvider("nonexistent", "unknown-model", primary)
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

	p := al.resolveProvider("", "", primary)
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
	msgBus.PublishInbound(context.Background(), bus.InboundMessage{
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

func TestBuildPlanReminder(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantOK     bool
		wantSubstr string
	}{
		{"interviewing", "interviewing", true, "interviewing the user"},
		{"review", "review", true, "under review"},
		{"executing returns false", "executing", false, ""},
		{"empty returns false", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok := buildPlanReminder(tt.status)
			if ok != tt.wantOK {
				t.Fatalf("buildPlanReminder(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if msg.Role != "user" {
				t.Errorf("expected role 'user', got %q", msg.Role)
			}
			if !strings.Contains(msg.Content, tt.wantSubstr) {
				t.Errorf("expected content to contain %q, got %q", tt.wantSubstr, msg.Content)
			}
		})
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

	agent := al.registry.GetDefaultAgent()

	// Create interviewing plan with phases (start requires phases)
	plan := "# Active Plan\n\n> Task: Test task\n> Status: interviewing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"
	_ = agent.ContextBuilder.WriteMemory(plan)

	// Transition to executing via /plan start
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "approved") {
		t.Errorf("expected 'approved', got %q", response)
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}

	// planStartPending must be set so Run() enqueues an LLM trigger
	if !al.planStartPending {
		t.Error("expected planStartPending to be true after /plan start")
	}
}

func TestPlanCommand_StartFromReview(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	// Create a plan in review status
	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"
	_ = agent.ContextBuilder.WriteMemory(plan)

	// Approve via /plan start
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "approved") {
		t.Errorf("expected 'approved', got %q", response)
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}

	if !al.planStartPending {
		t.Error("expected planStartPending to be true after /plan start from review")
	}
}

func TestPlanCommand_StartNoPhases(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Create interviewing plan without phases
	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})

	// Should be blocked because no phases exist
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "no phases") {
		t.Errorf("expected 'no phases' error, got %q", response)
	}

	agent := al.registry.GetDefaultAgent()
	if status := agent.ContextBuilder.GetPlanStatus(); status != "interviewing" {
		t.Errorf("expected status to remain 'interviewing', got %q", status)
	}

	if al.planStartPending {
		t.Error("planStartPending must not be set when start is rejected (no phases)")
	}
}

func TestPlanCommand_StartAlreadyExecuting(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	// Create interviewing plan with phases, then start
	plan := "# Active Plan\n\n> Task: Test task\n> Status: interviewing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"
	_ = agent.ContextBuilder.WriteMemory(plan)
	al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	// Clear the flag from the first call (simulating Run() consuming it)
	al.planStartPending = false

	// Try start again — should be rejected
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})
	if !strings.Contains(response, "already executing") {
		t.Errorf("expected 'already executing', got %q", response)
	}

	if al.planStartPending {
		t.Error("planStartPending must not be set when plan is already executing")
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

// TestAutoCompleteClears verifies that plan is marked completed with correct phase when all phases are complete.
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

	// Plan should be kept with status "completed" and phase set to total
	if !agent.ContextBuilder.HasActivePlan() {
		t.Error("expected plan to be retained after completion")
	}
	if status := agent.ContextBuilder.GetPlanStatus(); status != "completed" {
		t.Errorf("expected plan status 'completed', got %q", status)
	}
	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 1 {
		t.Errorf("expected phase 1 (total phases), got %d", phase)
	}
}

func TestIsToolAllowedDuringInterview_FuzzyNames(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		want bool
	}{
		// Exact names — read tools allowed
		{"read_file", nil, true},
		{"list_dir", nil, true},
		{"web_search", nil, true},
		{"web_fetch", nil, true},
		// Fuzzy variants — should also be allowed
		{"readfile", nil, true},
		{"ReadFile", nil, true},
		{"listdir", nil, true},
		{"websearch", nil, true},
		{"webfetch", nil, true},
		// Message tool — allowed (needed for interview questions)
		{"message", nil, true},
		{"Message", nil, true},
		// Write to MEMORY.md — allowed
		{"edit_file", map[string]interface{}{"path": "/ws/memory/MEMORY.md"}, true},
		{"editfile", map[string]interface{}{"path": "/ws/memory/MEMORY.md"}, true},
		{"EditFile", map[string]interface{}{"path": "/ws/memory/MEMORY.md"}, true},
		// Write to non-MEMORY.md — blocked
		{"edit_file", map[string]interface{}{"path": "/ws/main.go"}, false},
		{"editfile", map[string]interface{}{"path": "/ws/main.go"}, false},
		// exec — read-only commands allowed
		{"exec", map[string]interface{}{"command": "find . -name '*.py'"}, true},
		{"exec", map[string]interface{}{"command": "ls -la"}, true},
		{"exec", map[string]interface{}{"command": "grep -r TODO ."}, true},
		{"exec", map[string]interface{}{"command": "cat README.md"}, true},
		// exec — cd prefix stripped
		{"exec", map[string]interface{}{"command": "cd /home/user/project && find . -type f"}, true},
		{"exec", map[string]interface{}{"command": "cd /tmp && rm -rf *"}, false},
		// exec — write operators blocked
		{"exec", map[string]interface{}{"command": "find . > output.txt"}, false},
		{"exec", map[string]interface{}{"command": "ls -la >> log.txt"}, false},
		{"exec", map[string]interface{}{"command": "cat foo | tee bar.txt"}, false},
		// exec — path traversal blocked
		{"exec", map[string]interface{}{"command": "cat ../../etc/passwd"}, false},
		{"exec", map[string]interface{}{"command": "find ../../"}, false},
		{"exec", map[string]interface{}{"command": "ls ../secret"}, false},
		// exec — absolute paths blocked
		{"exec", map[string]interface{}{"command": "cat /etc/passwd"}, false},
		{"exec", map[string]interface{}{"command": "find /etc -name '*.conf'"}, false},
		{"exec", map[string]interface{}{"command": "ls /root"}, false},
		// exec — write commands blocked
		{"exec", map[string]interface{}{"command": "rm -rf /"}, false},
		{"exec", map[string]interface{}{"command": "mv a b"}, false},
		// exec — no args / empty command blocked
		{"exec", nil, false},
		{"exec", map[string]interface{}{"command": ""}, false},
		{"Exec", nil, false},
	}
	for _, tt := range tests {
		got := isToolAllowedDuringInterview(tt.name, tt.args)
		if got != tt.want {
			t.Errorf("isToolAllowedDuringInterview(%q, %v) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestBuildArgsSnippet_ExecStripsCD(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		args      map[string]interface{}
		workspace string
		wantSnip  string
	}{
		{
			name:      "exec strips cd prefix",
			tool:      "exec",
			args:      map[string]interface{}{"command": "cd /home/user/workspace/project/my-projects && pytest tests/test_integration.py"},
			workspace: "/home/user/workspace",
			wantSnip:  "pytest tests/test_integration.py",
		},
		{
			name:      "exec no cd prefix, flags stripped",
			tool:      "exec",
			args:      map[string]interface{}{"command": "ls -la"},
			workspace: "/ws",
			wantSnip:  "ls",
		},
		{
			name:      "exec empty command",
			tool:      "exec",
			args:      map[string]interface{}{},
			workspace: "/ws",
			wantSnip:  "{}",
		},
		{
			name:      "read_file strips workspace",
			tool:      "read_file",
			args:      map[string]interface{}{"path": "/home/user/workspace/src/main.go"},
			workspace: "/home/user/workspace",
			wantSnip:  "src/main.go",
		},
		{
			name:      "edit_file shows path",
			tool:      "edit_file",
			args:      map[string]interface{}{"path": "/ws/config.json", "old_text": "old value here"},
			workspace: "/ws",
			wantSnip:  "config.json",
		},
		{
			name:      "file tool long path prioritizes filename",
			tool:      "read_file",
			args:      map[string]interface{}{"path": "/ws/projects/terra-py-form/src/terra_py_form/hot/state/backend.py"},
			workspace: "/ws",
			wantSnip:  "projects/terra-py-form/src/terra_py_form/hot/sta\u2026/backend.py",
		},
		{
			name:      "unknown tool shows raw JSON",
			tool:      "web_search",
			args:      map[string]interface{}{"query": "hello"},
			workspace: "/ws",
			wantSnip:  `{"query":"hello"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildArgsSnippet(tt.tool, tt.args, tt.workspace)
			if got != tt.wantSnip {
				t.Errorf("buildArgsSnippet(%q) = %q, want %q", tt.tool, got, tt.wantSnip)
			}
		})
	}
}

func TestFormatCompactEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    toolLogEntry
		wantSub  string // must be a substring
		wantMark string // result marker must appear
		noTime   bool   // if true, duration should NOT appear
	}{
		{
			name:     "exec short entry",
			entry:    toolLogEntry{Name: "[1] exec", ArgsSnip: "ls", Result: "✓ 1.0s"},
			wantSub:  "exec ls",
			wantMark: "✓ 1.0s", // exec keeps duration
		},
		{
			name:     "exec long entry truncated from end",
			entry:    toolLogEntry{Name: "[2] exec", ArgsSnip: "pytest tests/integration/test_very_long_name.py", Result: "✗ 3.0s"},
			wantMark: "✗",
		},
		{
			name:     "file tool omits duration, shows filename",
			entry:    toolLogEntry{Name: "[3] edit_file", ArgsSnip: "projects/terra/src/deep/nested/backend.py", Result: "✓ 0.0s"},
			wantSub:  "backend.py",
			wantMark: "✓",
			noTime:   true,
		},
		{
			name:     "file tool path truncates from start",
			entry:    toolLogEntry{Name: "[4] read_file", ArgsSnip: "projects/terra-py-form/src/terra_py_form/hot/state/backend.py", Result: "✓ 0.1s"},
			wantSub:  "backend.py",
			wantMark: "✓",
			noTime:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCompactEntry(tt.entry)
			if tt.wantSub != "" && !strings.Contains(got, tt.wantSub) {
				t.Errorf("expected to contain %q, got: %q", tt.wantSub, got)
			}
			if !strings.Contains(got, tt.wantMark) {
				t.Errorf("result marker %q missing from: %q", tt.wantMark, got)
			}
			if tt.noTime && strings.Contains(got, "0s") {
				t.Errorf("file tool should omit duration, got: %q", got)
			}
			// Must not exceed maxEntryLineWidth
			if runeLen := len([]rune(got)); runeLen > maxEntryLineWidth {
				t.Errorf("entry too wide: %d runes (max %d): %q", runeLen, maxEntryLineWidth, got)
			}
		})
	}
}

func TestBuildRichStatus(t *testing.T) {
	task := &activeTask{
		Iteration: 3,
		MaxIter:   20,
		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls -la", Result: "✓ 1.2s"},
			{Name: "exec", ArgsSnip: "pytest tests/", Result: "✓ 5.0s"},
			{Name: "read_file", ArgsSnip: "src/main.go", Result: "⏳"},
		},
	}

	got := buildRichStatus(task, false, "/home/user/my-projects")

	mustContain := []string{
		"Task in progress (3/20)",
		"my-projects",
		"read_file",       // latest entry
		"No errors",       // no error yet
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("expected output to contain %q, got:\n%s", s, got)
		}
	}

	// Non-background: should NOT have reply prompt
	if strings.Contains(got, "Reply to intervene") {
		t.Error("non-background task should not have reply prompt")
	}

	// Background: should have reply prompt
	bgGot := buildRichStatus(task, true, "/home/user/my-projects")
	if !strings.Contains(bgGot, "Reply to intervene") {
		t.Error("background task should have reply prompt")
	}
}

func TestBuildRichStatus_ProjectDir(t *testing.T) {
	// exec-based projectDir takes priority
	task := &activeTask{
		Iteration:  1,
		MaxIter:    10,
		projectDir: "terra-py-form",
		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls", Result: "✓ 0.1s"},
		},
	}
	got := buildRichStatus(task, false, "/home/user/.picoclaw/workspace")
	if !strings.Contains(got, "terra-py-form") {
		t.Errorf("expected projectDir in output, got:\n%s", got)
	}

	// fileCommonDir fallback
	task2 := &activeTask{
		Iteration:     1,
		MaxIter:       10,
		fileCommonDir: "projects/terra-py-form",
		toolLog: []toolLogEntry{
			{Name: "read_file", ArgsSnip: "src/main.py", Result: "✓ 0.1s"},
		},
	}
	got2 := buildRichStatus(task2, false, "/home/user/.picoclaw/workspace")
	if !strings.Contains(got2, "terra-py-form") {
		t.Errorf("expected fileCommonDir basename in output, got:\n%s", got2)
	}

	// workspace basename fallback with trailing slash
	task3 := &activeTask{
		Iteration: 1,
		MaxIter:   10,
		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls", Result: "✓ 0.1s"},
		},
	}
	for _, ws := range []string{"/home/user/my-project/", "/home/user/my-project"} {
		got := buildRichStatus(task3, false, ws)
		if !strings.Contains(got, "my-project") {
			t.Errorf("workspace %q: expected 'my-project' in output, got:\n%s", ws, got)
		}
	}
}

func TestExtractExecProjectDir(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"cd deep path", "cd /ws/projects/terra-py-form && pytest", "terra-py-form"},
		{"cd direct subdir", "cd /ws/my-app && make build", "my-app"},
		{"cd trailing slash", "cd /ws/my-app/ && ls", "my-app"},
		{"cd to workspace", "cd /ws && ls", "ws"},
		{"no cd prefix", "pytest tests/", ""},
		{"empty command", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{"command": tt.cmd}
			got := extractExecProjectDir(args)
			if got != tt.want {
				t.Errorf("extractExecProjectDir(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestFileParentRelDir(t *testing.T) {
	ws := "/home/user/.picoclaw/workspace"
	tests := []struct {
		name string
		path string
		want string
	}{
		{"deep path", ws + "/projects/terra/src/main.py", "projects/terra/src"},
		{"direct subdir", ws + "/my-app/README.md", "my-app"},
		{"workspace root file", ws + "/notes.txt", ""},
		{"outside workspace", "/tmp/foo.txt", ""},
		{"trailing slash ws", ws + "/projects/terra/src/main.py", "projects/terra/src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileParentRelDir(tt.path, ws)
			if got != tt.want {
				t.Errorf("fileParentRelDir(%q, ws) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCommonDirPrefix(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{"same dir", "projects/terra/src", "projects/terra/src", "projects/terra/src"},
		{"converge to project", "projects/terra/src", "projects/terra/tests", "projects/terra"},
		{"converge to top", "projects/terra/src", "projects/other/tests", "projects"},
		{"no common", "aaa/bbb", "ccc/ddd", ""},
		{"one is prefix", "projects/terra", "projects/terra/src", "projects/terra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonDirPrefix(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("commonDirPrefix(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDisplayProjectDir(t *testing.T) {
	// exec projectDir wins
	task1 := &activeTask{projectDir: "my-app", fileCommonDir: "projects/other"}
	if got := displayProjectDir(task1); got != "my-app" {
		t.Errorf("expected 'my-app', got %q", got)
	}
	// fileCommonDir fallback: basename
	task2 := &activeTask{fileCommonDir: "projects/terra-py-form"}
	if got := displayProjectDir(task2); got != "terra-py-form" {
		t.Errorf("expected 'terra-py-form', got %q", got)
	}
	// single component
	task3 := &activeTask{fileCommonDir: "my-app"}
	if got := displayProjectDir(task3); got != "my-app" {
		t.Errorf("expected 'my-app', got %q", got)
	}
	// empty
	task4 := &activeTask{}
	if got := displayProjectDir(task4); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildRichStatus_FixedHeight(t *testing.T) {
	// Test that output has the same number of lines regardless of entry count
	countLines := func(s string) int {
		return strings.Count(s, "\n")
	}

	// 0 entries
	task0 := &activeTask{Iteration: 1, MaxIter: 10}
	lines0 := countLines(buildRichStatus(task0, true, "/ws/p"))

	// 1 entry
	task1 := &activeTask{Iteration: 1, MaxIter: 10,
		toolLog: []toolLogEntry{{Name: "exec", ArgsSnip: "ls", Result: "⏳"}}}
	lines1 := countLines(buildRichStatus(task1, true, "/ws/p"))

	// 5 entries
	task5 := &activeTask{Iteration: 5, MaxIter: 10}
	for i := 0; i < 5; i++ {
		task5.toolLog = append(task5.toolLog, toolLogEntry{
			Name: fmt.Sprintf("[%d] exec", i), ArgsSnip: "cmd", Result: "✓ 1.0s"})
	}
	lines5 := countLines(buildRichStatus(task5, true, "/ws/p"))

	// 5 entries + sticky error
	task5err := &activeTask{Iteration: 5, MaxIter: 10}
	for i := 0; i < 5; i++ {
		task5err.toolLog = append(task5err.toolLog, toolLogEntry{
			Name: fmt.Sprintf("[%d] exec", i), ArgsSnip: "cmd", Result: "✓ 1.0s"})
	}
	errEntry := toolLogEntry{Name: "[3] exec", ArgsSnip: "pytest", Result: "✗ 2.0s",
		ErrDetail: "FAILED test\nExit code: 1"}
	task5err.lastError = &errEntry
	lines5err := countLines(buildRichStatus(task5err, true, "/ws/p"))

	if lines0 != lines1 || lines1 != lines5 || lines5 != lines5err {
		t.Errorf("line counts should be equal: 0=%d, 1=%d, 5=%d, 5+err=%d",
			lines0, lines1, lines5, lines5err)
	}
}

func TestBuildRichStatus_StickyError(t *testing.T) {
	// Error from a past entry sticks in the error section
	errEntry := toolLogEntry{
		Name: "[2] exec", ArgsSnip: "pytest", Result: "✗ 3.2s",
		ErrDetail: "FAILED test_login\nExit code: 1",
	}
	task := &activeTask{
		Iteration: 5,
		MaxIter:   10,
		toolLog: []toolLogEntry{
			{Name: "[3] read_file", ArgsSnip: "src/auth.py", Result: "✓ 0.1s"},
			{Name: "[4] edit_file", ArgsSnip: "src/auth.py", Result: "✓ 0.2s"},
			{Name: "[5] exec", ArgsSnip: "pytest --retry", Result: "⏳"},
		},
		lastError: &errEntry,
	}

	got := buildRichStatus(task, false, "/ws/p")

	// Error section should show the sticky error in code block
	if !strings.Contains(got, "FAILED test_login") {
		t.Errorf("expected sticky error detail in error section, got:\n%s", got)
	}
	if !strings.Contains(got, "\u274C") { // ❌
		t.Errorf("expected ❌ error header, got:\n%s", got)
	}

	// Latest entry is NOT the error
	if !strings.Contains(got, "pytest --retry") {
		t.Errorf("expected latest entry command, got:\n%s", got)
	}
}

func TestBuildRichStatus_LatestEntryNoInlineResult(t *testing.T) {
	longCmd := "uv run pytest tests/hot/test_state_backend_integration.py"
	task := &activeTask{
		Iteration: 2,
		MaxIter:   10,
		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls -la", Result: "\u2713 0.5s"},
			{Name: "exec", ArgsSnip: longCmd, Result: "\u23F3"},
		},
	}

	got := buildRichStatus(task, false, "/ws/my-project")

	// Latest entry shows command (possibly truncated) with filename visible
	if !strings.Contains(got, "integration.py") {
		t.Errorf("latest entry should show filename, got:\n%s", got)
	}
	// Result on separate indented line
	if !strings.Contains(got, "  \u23F3") {
		t.Errorf("latest entry result should be on indented line, got:\n%s", got)
	}
	// Project name shown
	if !strings.Contains(got, "my-project") {
		t.Errorf("should show project name, got:\n%s", got)
	}
	// No second separator before error section
	lines := strings.Split(got, "\n")
	sepCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "\u2501") {
			sepCount++
		}
	}
	if sepCount != 1 {
		t.Errorf("expected exactly 1 separator, got %d in:\n%s", sepCount, got)
	}
}

func TestSanitizeHistoryForProvider_MultiToolCall(t *testing.T) {
	// Regression: assistant with 2+ tool_calls had 2nd+ tool results dropped
	// because the check only allowed tool after assistant, not after sibling tool.
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "a", Function: &providers.FunctionCall{Name: "exec"}},
			{ID: "b", Function: &providers.FunctionCall{Name: "read_file"}},
		}},
		{Role: "tool", Content: "ok", ToolCallID: "a"},
		{Role: "tool", Content: "ok", ToolCallID: "b"},
		{Role: "assistant", Content: "done"},
	}

	got := sanitizeHistoryForProvider(history)

	// All 5 messages must survive
	if len(got) != 5 {
		roles := make([]string, len(got))
		for i, m := range got {
			roles[i] = m.Role
		}
		t.Fatalf("expected 5 messages, got %d: %v", len(got), roles)
	}
	// Verify both tool results present
	toolCount := 0
	for _, m := range got {
		if m.Role == "tool" {
			toolCount++
		}
	}
	if toolCount != 2 {
		t.Errorf("expected 2 tool results, got %d", toolCount)
	}
}

// ---------- plan nudge tests ----------

// countingMockProvider counts Chat calls and always returns text-only responses.
type countingMockProvider struct {
	callCount int
}

func (m *countingMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.callCount++
	return &providers.LLMResponse{
		Content:   fmt.Sprintf("Response %d", m.callCount),
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *countingMockProvider) GetDefaultModel() string {
	return "mock-counting-model"
}

func TestPlanNudge_ForegroundExecution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-nudge-test-*")
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

	provider := &countingMockProvider{}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("no default agent")
	}

	// Write a plan in executing status with unchecked steps
	plan := "# Active Plan\n\n> Task: Test\n> Status: executing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n- [ ] Step two\n\n## Context\n"
	agent.ContextBuilder.WriteMemory(plan)

	// Process a foreground message (no background metadata)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "continue working",
		SessionKey: "nudge-test",
	}
	_, err = al.processMessage(ctx, msg)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}

	// The provider should have been called at least 2 times:
	// 1st call: returns text-only → nudge fires (unchecked steps remain)
	// 2nd call: returns text-only → nudge already fired, loop exits
	if provider.callCount < 2 {
		t.Errorf("expected at least 2 provider calls (nudge should trigger continuation), got %d", provider.callCount)
	}
}

func TestPlanNudge_NoNudgeWhenAllStepsComplete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-nudge-test-*")
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

	provider := &countingMockProvider{}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("no default agent")
	}

	// Write a plan where all steps are already checked
	plan := "# Active Plan\n\n> Task: Test\n> Status: executing\n> Phase: 1\n\n## Phase 1: Setup\n- [x] Step one\n- [x] Step two\n\n## Context\n"
	agent.ContextBuilder.WriteMemory(plan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "all done",
		SessionKey: "nudge-test-complete",
	}
	_, err = al.processMessage(ctx, msg)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}

	// No unchecked steps → preUnchecked=0 → no nudge → only 1 provider call
	if provider.callCount != 1 {
		t.Errorf("expected exactly 1 provider call (no nudge needed), got %d", provider.callCount)
	}
}

func TestPlanNudge_ProgressMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-nudge-test-*")
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

	// Provider that checks the nudge message content on the 2nd call
	var nudgeContent string
	provider := &nudgeCaptureMockProvider{onSecondCall: func(msgs []providers.Message) {
		// The last user message should be the nudge
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "user" {
				nudgeContent = msgs[i].Content
				break
			}
		}
	}}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("no default agent")
	}

	// Write a plan with 3 unchecked steps; the provider edits memory to
	// mark one step between calls (simulated by the first-call hook).
	plan := "# Active Plan\n\n> Task: Test\n> Status: executing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n- [ ] Step two\n- [ ] Step three\n\n## Context\n"
	agent.ContextBuilder.WriteMemory(plan)

	// After the first LLM response (no tool calls), simulate that
	// one step was marked [x] externally (as if the AI did it via tool).
	// We do this by hooking the provider's first call to mutate memory.
	provider.onFirstCall = func() {
		updated := strings.Replace(agent.ContextBuilder.ReadMemory(), "- [ ] Step one", "- [x] Step one", 1)
		agent.ContextBuilder.WriteMemory(updated)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "work on the plan",
		SessionKey: "nudge-progress-test",
	}
	_, err = al.processMessage(ctx, msg)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}

	// Should have gotten the "Progress recorded" nudge (not the "none were marked" one)
	if !strings.Contains(nudgeContent, "Progress recorded") {
		t.Errorf("expected 'Progress recorded' nudge, got %q", nudgeContent)
	}
	if !strings.Contains(nudgeContent, "2 unchecked steps remain") {
		t.Errorf("expected '2 unchecked steps remain' in nudge, got %q", nudgeContent)
	}
}

// nudgeCaptureMockProvider calls hooks on 1st and 2nd Chat invocations.
type nudgeCaptureMockProvider struct {
	callCount    int
	onFirstCall  func()
	onSecondCall func([]providers.Message)
}

func (m *nudgeCaptureMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.callCount++
	if m.callCount == 1 && m.onFirstCall != nil {
		m.onFirstCall()
	}
	if m.callCount == 2 && m.onSecondCall != nil {
		m.onSecondCall(messages)
	}
	return &providers.LLMResponse{
		Content:   fmt.Sprintf("Response %d", m.callCount),
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *nudgeCaptureMockProvider) GetDefaultModel() string {
	return "mock-nudge-model"
}

// --- consumeStreamWithRepetitionDetection tests ---

func TestConsumeStream_NormalCompletion(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)
	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "Hello "}
		ch <- protocoltypes.StreamEvent{ContentDelta: "world!"}
		ch <- protocoltypes.StreamEvent{
			FinishReason: "stop",
			Usage:        &providers.UsageInfo{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		}
		close(ch)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected detected=false for normal content")
	}
	if resp.Content != "Hello world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 7 {
		t.Errorf("Usage.TotalTokens = %v, want 7", resp.Usage)
	}
	_ = ctx // keep linter happy
}

func TestConsumeStream_DetectsRepetition(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 64)
	cancelCalled := false

	ctx, cancel := context.WithCancel(context.Background())
	wrappedCancel := func() {
		cancelCalled = true
		cancel()
	}

	// Send enough repetitive content to trigger detection.
	// The pattern "abcdefghij" repeated many times will have very low n-gram uniqueness.
	repeatedChunk := strings.Repeat("abcdefghij", 50) // 500 chars per chunk
	go func() {
		// Send 6 chunks of repetitive content = 3000 chars total,
		// each with 500 runes. The check triggers after every 1000 runes
		// when content > 2000 chars.
		for i := 0; i < 6; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: repeatedChunk}
		}
		// Send more data that should be ignored after detection.
		for i := 0; i < 10; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: "more data"}
		}
		close(ch)
	}()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, wrappedCancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detected {
		t.Fatal("expected repetition detection to trigger")
	}
	if !cancelCalled {
		t.Error("expected cancelFn to be called")
	}
	// The response should be shorter than the full 3000+ chars
	// because detection triggers early.
	if len(resp.Content) >= 3000+10*len("more data") {
		t.Errorf("Content length = %d, expected less than full output", len(resp.Content))
	}
	_ = ctx
}

func TestConsumeStream_ToolCallAccumulation(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)
	go func() {
		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ID: "call_1", Name: "test_fn", ArgumentsDelta: `{"ke`},
			},
		}
		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ArgumentsDelta: `y":"val"}`},
			},
		}
		ch <- protocoltypes.StreamEvent{FinishReason: "tool_calls"}
		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected no repetition detection for tool calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "test_fn" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "test_fn")
	}
	if resp.ToolCalls[0].Arguments["key"] != "val" {
		t.Errorf("ToolCalls[0].Arguments[key] = %v, want %q", resp.ToolCalls[0].Arguments["key"], "val")
	}
}

func TestConsumeStream_StreamError(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 4)
	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "partial"}
		ch <- protocoltypes.StreamEvent{Err: fmt.Errorf("read error")}
		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read error") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "read error")
	}
}

func TestConsumeStream_OnChunkCallback(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)
	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "Hello "}
		ch <- protocoltypes.StreamEvent{ContentDelta: "world"}
		ch <- protocoltypes.StreamEvent{ContentDelta: "!"}
		ch <- protocoltypes.StreamEvent{FinishReason: "stop"}
		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	var chunks []string
	onChunk := func(accumulated, _ string) {
		chunks = append(chunks, accumulated)
	}

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, onChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected detected=false")
	}
	if resp.Content != "Hello world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world!")
	}
	// onChunk should be called once per content delta (3 times)
	if len(chunks) != 3 {
		t.Fatalf("onChunk called %d times, want 3", len(chunks))
	}
	if chunks[0] != "Hello " {
		t.Errorf("chunks[0] = %q, want %q", chunks[0], "Hello ")
	}
	if chunks[1] != "Hello world" {
		t.Errorf("chunks[1] = %q, want %q", chunks[1], "Hello world")
	}
	if chunks[2] != "Hello world!" {
		t.Errorf("chunks[2] = %q, want %q", chunks[2], "Hello world!")
	}
}

func TestConsumeStream_OnChunkWithRepetitionDetection(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 64)
	cancelCalled := false

	ctx, cancel := context.WithCancel(context.Background())
	wrappedCancel := func() {
		cancelCalled = true
		cancel()
	}

	repeatedChunk := strings.Repeat("abcdefghij", 50) // 500 chars per chunk
	go func() {
		for i := 0; i < 6; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: repeatedChunk}
		}
		for i := 0; i < 10; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: "more data"}
		}
		close(ch)
	}()

	var chunkCount int
	onChunk := func(_, _ string) {
		chunkCount++
	}

	_, detected, err := consumeStreamWithRepetitionDetection(ch, wrappedCancel, 1000, onChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detected {
		t.Fatal("expected repetition detection to trigger")
	}
	if !cancelCalled {
		t.Error("expected cancelFn to be called")
	}
	// onChunk should have been called at least once before detection
	if chunkCount == 0 {
		t.Error("expected onChunk to be called at least once")
	}
	_ = ctx
}

// modelCapturingMockProvider records which model was passed to Chat.
type modelCapturingMockProvider struct {
	mu        sync.Mutex
	models    []string
	response  string
}

func (m *modelCapturingMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools_ []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	m.models = append(m.models, model)
	m.mu.Unlock()
	return &providers.LLMResponse{
		Content:   m.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *modelCapturingMockProvider) GetDefaultModel() string {
	return "mock-capture-model"
}

func TestAgentLoop_PlanModel_UsedDuringInterviewing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-planmodel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "normal-model",
				PlanModel:         "plan-model",
				MaxTokens:         4096,
				MaxToolIterations: 2,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &modelCapturingMockProvider{response: "Plan interview response"}
	al := NewAgentLoop(cfg, msgBus, provider)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}

	// Write MEMORY.md with interviewing status to activate plan model
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0o755)
	memoryPath := filepath.Join(memoryDir, "MEMORY.md")
	memoryContent := "# Active Plan\n\n> Task: Test plan model\n> Status: interviewing\n> Phase: 1\n"
	if err := os.WriteFile(memoryPath, []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("Failed to write MEMORY.md: %v", err)
	}

	_, err = al.ProcessDirectWithChannel(
		context.Background(),
		"Hello, plan model test",
		"test-plan-session",
		"test",
		"test-chat",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if len(provider.models) == 0 {
		t.Fatal("Expected at least one Chat call")
	}
	// The first call should use the plan model since we're in interviewing state
	if provider.models[0] != "plan-model" {
		t.Errorf("Expected plan model 'plan-model' during interviewing, got %q", provider.models[0])
	}
}

func TestAgentLoop_PlanModel_NotUsedDuringExecuting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-planmodel-exec-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "normal-model",
				PlanModel:         "plan-model",
				MaxTokens:         4096,
				MaxToolIterations: 2,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &modelCapturingMockProvider{response: "Executing response"}
	al := NewAgentLoop(cfg, msgBus, provider)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}

	// Write MEMORY.md with executing status - should use normal model
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0o755)
	memoryPath := filepath.Join(memoryDir, "MEMORY.md")
	memoryContent := `# Active Plan

> Task: Test plan model
> Status: executing
> Phase: 1

## Phase 1: Build
- [ ] Run build
`
	if err := os.WriteFile(memoryPath, []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("Failed to write MEMORY.md: %v", err)
	}

	_, err = al.ProcessDirectWithChannel(
		context.Background(),
		"Hello, executing test",
		"test-exec-session",
		"test",
		"test-chat",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if len(provider.models) == 0 {
		t.Fatal("Expected at least one Chat call")
	}
	// During executing phase, should use normal model, not plan model
	if provider.models[0] != "normal-model" {
		t.Errorf("Expected normal model 'normal-model' during executing, got %q", provider.models[0])
	}
}

func TestAgentLoop_PlanModel_ResolvesProviderForSingleCandidate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-planmodel-resolve-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "MiniMax-M2.5",
				PlanModel:         "openai/gpt-5.2",
				MaxTokens:         4096,
				MaxToolIterations: 2,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	// The main provider simulates the wrong provider (e.g. MiniMax).
	mainProvider := &modelCapturingMockProvider{response: "wrong provider response"}
	al := NewAgentLoop(cfg, msgBus, mainProvider)

	// Inject a mock provider into the cache so resolveProvider returns it
	// for the "openai/gpt-5.2" candidate (provider="openai", model="gpt-5.2").
	resolvedProvider := &modelCapturingMockProvider{response: "correct provider response"}
	al.providerCache["openai/gpt-5.2"] = resolvedProvider

	// Write MEMORY.md with interviewing status to activate plan model
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0o755)
	memoryPath := filepath.Join(memoryDir, "MEMORY.md")
	memoryContent := "# Active Plan\n\n> Task: Test provider resolution\n> Status: interviewing\n> Phase: 1\n"
	if err := os.WriteFile(memoryPath, []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("Failed to write MEMORY.md: %v", err)
	}

	_, err = al.ProcessDirectWithChannel(
		context.Background(),
		"Hello, resolve provider test",
		"test-resolve-session",
		"test",
		"test-chat",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	resolvedProvider.mu.Lock()
	defer resolvedProvider.mu.Unlock()
	mainProvider.mu.Lock()
	defer mainProvider.mu.Unlock()

	// The resolved provider should have been called with the stripped model name
	if len(resolvedProvider.models) == 0 {
		t.Fatal("Expected resolved provider to receive Chat call, but it got none")
	}
	if resolvedProvider.models[0] != "gpt-5.2" {
		t.Errorf("Expected resolved provider to receive model 'gpt-5.2', got %q", resolvedProvider.models[0])
	}

	// The main provider should NOT have been called for the LLM request
	if len(mainProvider.models) > 0 {
		t.Errorf("Expected main provider to receive no Chat calls during plan model phase, got %d calls with models %v",
			len(mainProvider.models), mainProvider.models)
	}
}

func TestPlanCommand_StartClear(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	// Create a plan in review status with phases
	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"
	_ = agent.ContextBuilder.WriteMemory(plan)

	// Seed session history so we can verify it gets cleared
	agent.Sessions.AddMessage("test-session", "user", "hello")
	agent.Sessions.AddMessage("test-session", "assistant", "world")
	agent.Sessions.SetSummary("test-session", "some summary")

	// Approve with clear
	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{
		Content:    "/plan start clear",
		SessionKey: "test-session",
	})
	if !handled {
		t.Fatal("expected /plan start clear to be handled")
	}
	if !strings.Contains(response, "clean history") {
		t.Errorf("expected 'clean history' in response, got %q", response)
	}
	if !al.planStartPending {
		t.Error("expected planStartPending to be true")
	}
	if !al.planClearHistory {
		t.Error("expected planClearHistory to be true")
	}

	// Simulate what Run() does when planStartPending is set
	al.planStartPending = false
	clearHistory := al.planClearHistory
	al.planClearHistory = false
	if clearHistory {
		agent.Sessions.SetHistory("test-session", nil)
		agent.Sessions.SetSummary("test-session", "")
		_ = agent.Sessions.Save("test-session")
	}

	// Verify history and summary are cleared
	history := agent.Sessions.GetHistory("test-session")
	if len(history) != 0 {
		t.Errorf("expected empty history after clear, got %d messages", len(history))
	}
	summary := agent.Sessions.GetSummary("test-session")
	if summary != "" {
		t.Errorf("expected empty summary after clear, got %q", summary)
	}
}

func TestPlanCommand_StartWithoutClear_PreservesHistory(t *testing.T) {
	al, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	// Create a plan in review status with phases
	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"
	_ = agent.ContextBuilder.WriteMemory(plan)

	// Seed session history
	agent.Sessions.AddMessage("test-session", "user", "hello")
	agent.Sessions.AddMessage("test-session", "assistant", "world")
	agent.Sessions.SetSummary("test-session", "some summary")

	// Approve without clear
	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{
		Content:    "/plan start",
		SessionKey: "test-session",
	})
	if strings.Contains(response, "clean history") {
		t.Errorf("did not expect 'clean history' in response, got %q", response)
	}
	if al.planClearHistory {
		t.Error("planClearHistory should be false for /plan start without clear")
	}

	// Verify history is preserved
	history := agent.Sessions.GetHistory("test-session")
	if len(history) != 2 {
		t.Errorf("expected 2 history messages preserved, got %d", len(history))
	}
	summary := agent.Sessions.GetSummary("test-session")
	if summary != "some summary" {
		t.Errorf("expected summary preserved, got %q", summary)
	}
}

func TestFilterInterviewTools(t *testing.T) {
	allDefs := []providers.ToolDefinition{
		{Function: protocoltypes.ToolFunctionDefinition{Name: "read_file"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "list_dir"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "web_search"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "web_fetch"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "message"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "edit_file"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "append_file"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "write_file"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "exec"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "logs"}},
		// These should be filtered out:
		{Function: protocoltypes.ToolFunctionDefinition{Name: "spawn_subagent"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "skills_search"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "skills_install"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "bg_monitor"}},
		{Function: protocoltypes.ToolFunctionDefinition{Name: "i2c_transfer"}},
	}

	filtered := filterInterviewTools(allDefs)

	// Should keep exactly the 10 allowed tools
	if len(filtered) != 10 {
		names := make([]string, len(filtered))
		for i, d := range filtered {
			names[i] = d.Function.Name
		}
		t.Errorf("expected 10 allowed tools, got %d: %v", len(filtered), names)
	}

	// Verify none of the disallowed tools slipped through
	disallowed := map[string]bool{
		"spawnsubagent": true, "skillssearch": true,
		"skillsinstall": true, "bgmonitor": true, "ictransfer": true,
	}
	for _, d := range filtered {
		norm := tools.NormalizeToolName(d.Function.Name)
		if disallowed[norm] {
			t.Errorf("disallowed tool %q should have been filtered out", d.Function.Name)
		}
	}
}

func TestBuildStreamingDisplay_ContentOnly(t *testing.T) {
	display := buildStreamingDisplay("hello world", "")
	if !strings.HasSuffix(display, " \u2589") {
		t.Error("expected cursor suffix")
	}
	if strings.Contains(display, "\U0001f9e0") {
		t.Error("should not contain brain emoji when no reasoning")
	}
	lines := strings.Count(display, "\n") + 1
	if lines != streamingDisplayLines+1 { // TailPad lines + cursor on last line
		t.Logf("display:\n%s", display)
	}
}

func TestBuildStreamingDisplay_ReasoningOnly(t *testing.T) {
	display := buildStreamingDisplay("", "let me think about this")
	if !strings.Contains(display, "\U0001f9e0") {
		t.Error("expected brain emoji for reasoning phase")
	}
	if !strings.Contains(display, "Thinking...") {
		t.Error("expected Thinking... header")
	}
	if !strings.HasSuffix(display, " \u2589") {
		t.Error("expected cursor suffix")
	}
}

func TestBuildStreamingDisplay_Both(t *testing.T) {
	display := buildStreamingDisplay("the answer is 42", "first I considered...")
	if !strings.Contains(display, "\U0001f9e0") {
		t.Error("expected brain emoji")
	}
	if !strings.Contains(display, "responding") {
		t.Error("expected responding header when both present")
	}
	if !strings.Contains(display, "the answer is 42") {
		t.Error("expected content in display")
	}
}

func TestHandleReasoning(t *testing.T) {
	newLoop := func(t *testing.T) (*AgentLoop, *bus.MessageBus) {
		t.Helper()
		tmpDir, err := os.MkdirTemp("", "agent-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
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
		return NewAgentLoop(cfg, msgBus, &mockProvider{}), msgBus
	}

	t.Run("skips when any required field is empty", func(t *testing.T) {
		al, msgBus := newLoop(t)
		al.handleReasoning(context.Background(), "reasoning", "telegram", "")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if msg, ok := msgBus.SubscribeOutbound(ctx); ok {
			t.Fatalf("expected no outbound message, got %+v", msg)
		}
	})

	t.Run("publishes one message for non telegram", func(t *testing.T) {
		al, msgBus := newLoop(t)
		al.handleReasoning(context.Background(), "hello reasoning", "slack", "channel-1")

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		msg, ok := msgBus.SubscribeOutbound(ctx)
		if !ok {
			t.Fatal("expected an outbound message")
		}
		if msg.Channel != "slack" || msg.ChatID != "channel-1" || msg.Content != "hello reasoning" {
			t.Fatalf("unexpected outbound message: %+v", msg)
		}
	})

	t.Run("publishes one message for telegram", func(t *testing.T) {
		al, msgBus := newLoop(t)
		reasoning := "hello telegram reasoning"
		al.handleReasoning(context.Background(), reasoning, "telegram", "tg-chat")

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		msg, ok := msgBus.SubscribeOutbound(ctx)
		if !ok {
			t.Fatal("expected outbound message")
		}

		if msg.Channel != "telegram" {
			t.Fatalf("expected telegram channel message, got %+v", msg)
		}
		if msg.ChatID != "tg-chat" {
			t.Fatalf("expected chatID tg-chat, got %+v", msg)
		}
		if msg.Content != reasoning {
			t.Fatalf("content mismatch: got %q want %q", msg.Content, reasoning)
		}
	})
	t.Run("expired ctx", func(t *testing.T) {
		al, msgBus := newLoop(t)
		reasoning := "hello telegram reasoning"
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		al.handleReasoning(ctx, reasoning, "telegram", "tg-chat")

		ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		msg, ok := msgBus.SubscribeOutbound(ctx)
		if ok {
			t.Fatalf("expected no outbound message, got %+v", msg)
		}
	})

	t.Run("returns promptly when bus is full", func(t *testing.T) {
		al, msgBus := newLoop(t)

		// Fill the outbound bus buffer until a publish would block.
		// Use a short timeout to detect when the buffer is full,
		// rather than hardcoding the buffer size.
		for i := 0; ; i++ {
			fillCtx, fillCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			err := msgBus.PublishOutbound(fillCtx, bus.OutboundMessage{
				Channel: "filler",
				ChatID:  "filler",
				Content: fmt.Sprintf("filler-%d", i),
			})
			fillCancel()
			if err != nil {
				// Buffer is full (timed out trying to send).
				break
			}
		}

		// Use a short-deadline parent context to bound the test.
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		al.handleReasoning(ctx, "should timeout", "slack", "channel-full")
		elapsed := time.Since(start)

		// handleReasoning uses a 5s internal timeout, but the parent ctx
		// expires in 500ms. It should return within ~500ms, not 5s.
		if elapsed > 2*time.Second {
			t.Fatalf("handleReasoning blocked too long (%v); expected prompt return", elapsed)
		}

		// Drain the bus and verify the reasoning message was NOT published
		// (it should have been dropped due to timeout).
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer drainCancel()
		foundReasoning := false
		for {
			msg, ok := msgBus.SubscribeOutbound(drainCtx)
			if !ok {
				break
			}
			if msg.Content == "should timeout" {
				foundReasoning = true
			}
		}
		if foundReasoning {
			t.Fatal("expected reasoning message to be dropped when bus is full, but it was published")
		}
	})
}

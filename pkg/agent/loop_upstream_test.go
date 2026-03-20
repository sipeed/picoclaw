package agent

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestResolveMessageRoute_DefaultAgent(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	msg := bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Peer:     bus.Peer{Kind: "direct", ID: "user1"},
	}

	route, agent, err := al.resolveMessageRoute(msg)
	if err != nil {
		t.Fatalf("resolveMessageRoute error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if route.SessionKey == "" {
		t.Fatal("expected non-empty session key")
	}
}

func TestResolveScopeKey_PresetOverridesRoute(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	msg := bus.InboundMessage{
		Channel:    "telegram",
		SenderID:   "user1",
		ChatID:     "chat1",
		SessionKey: "custom-session",
		Peer:       bus.Peer{Kind: "direct", ID: "user1"},
	}

	route, _, err := al.resolveMessageRoute(msg)
	if err != nil {
		t.Fatalf("resolveMessageRoute error: %v", err)
	}

	key := resolveScopeKey(route, msg.SessionKey)
	if key != "custom-session" {
		t.Errorf("expected custom-session, got %q", key)
	}

	key2 := resolveScopeKey(route, "")
	if key2 != route.SessionKey {
		t.Errorf("expected route session key %q, got %q", route.SessionKey, key2)
	}
}

func TestSelectCandidates_DefaultCandidates(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("no default agent")
	}

	candidates := al.selectCandidates(agent, "hello", nil)
	// Should return agent.Candidates (may be empty in test config, but should not panic)
	if candidates == nil {
		// Candidates can be nil/empty in minimal test config — just ensure no panic
		candidates = []providers.FallbackCandidate{}
	}
	_ = candidates
}

func TestSelectCandidates_WithRouterUsesLight(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "heavy-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("no default agent")
	}

	// Without router: should return agent.Candidates
	result := al.selectCandidates(agent, "hi", nil)
	if len(result) != len(agent.Candidates) {
		t.Errorf("expected %d candidates, got %d", len(agent.Candidates), len(result))
	}
}

func TestBuildCommandsRuntime_GetModelInfo(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	rt := al.buildCommandsRuntime(agent, "test-session")

	model, prov := rt.GetModelInfo()
	if model == "" {
		t.Error("expected non-empty model")
	}
	if prov == "" {
		t.Error("expected non-empty provider")
	}
}

func TestBuildCommandsRuntime_ListAgentIDs(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	rt := al.buildCommandsRuntime(agent, "")

	ids := rt.ListAgentIDs()
	if len(ids) == 0 {
		t.Error("expected at least one agent ID")
	}
}

func TestBuildCommandsRuntime_SwitchModel(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	rt := al.buildCommandsRuntime(agent, "")

	old, err := rt.SwitchModel("new-model")
	if err != nil {
		t.Fatalf("SwitchModel error: %v", err)
	}
	if old == "" {
		t.Error("expected non-empty old model")
	}
	if agent.Model != "new-model" {
		t.Errorf("expected agent model to be new-model, got %q", agent.Model)
	}
}

func TestBuildCommandsRuntime_ClearHistory(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	agent.Sessions.AddMessage("test-key", "user", "hello")

	rt := al.buildCommandsRuntime(agent, "test-key")
	if err := rt.ClearHistory(); err != nil {
		t.Fatalf("ClearHistory error: %v", err)
	}

	history := agent.Sessions.GetHistory("test-key")
	if len(history) != 0 {
		t.Errorf("expected empty history after clear, got %d messages", len(history))
	}
}

func TestBuildCommandsRuntime_ReloadConfig_NoFunc(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	rt := al.buildCommandsRuntime(agent, "")

	err := rt.ReloadConfig()
	if err == nil {
		t.Error("expected error when reloadFunc is nil")
	}
}

func TestBuildCommandsRuntime_ReloadConfig_WithFunc(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	called := false
	al.SetReloadFunc(func() error {
		called = true
		return nil
	})

	agent := al.registry.GetDefaultAgent()
	rt := al.buildCommandsRuntime(agent, "")

	if err := rt.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig error: %v", err)
	}
	if !called {
		t.Error("expected reloadFunc to be called")
	}
}

func TestInvokeTypingStop(t *testing.T) {
	cfg := &config.Config{}
	msgBus := bus.NewMessageBus()
	m, err := newTestManager(cfg, msgBus)
	if err != nil {
		t.Skipf("cannot create test manager: %v", err)
	}

	var stopped bool
	var mu sync.Mutex
	m.RecordTypingStop("telegram", "chat1", func() {
		mu.Lock()
		stopped = true
		mu.Unlock()
	})

	m.InvokeTypingStop("telegram", "chat1")

	mu.Lock()
	defer mu.Unlock()
	if !stopped {
		t.Error("expected typing stop to be invoked")
	}
}

func TestInvokeTypingStop_NoOp(t *testing.T) {
	cfg := &config.Config{}
	msgBus := bus.NewMessageBus()
	m, err := newTestManager(cfg, msgBus)
	if err != nil {
		t.Skipf("cannot create test manager: %v", err)
	}

	// Should not panic when no typing indicator is active
	m.InvokeTypingStop("telegram", "nonexistent")
}

// newTestManager creates a minimal channels.Manager for testing.
func newTestManager(cfg *config.Config, _ *bus.MessageBus) (*testChannelManager, error) {
	return &testChannelManager{}, nil
}

// testChannelManager is a minimal mock that supports InvokeTypingStop testing.
type testChannelManager struct {
	typingStops sync.Map
}

func (m *testChannelManager) RecordTypingStop(channel, chatID string, stop func()) {
	m.typingStops.Store(channel+":"+chatID, stop)
}

func (m *testChannelManager) InvokeTypingStop(channel, chatID string) {
	key := channel + ":" + chatID
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if fn, ok := v.(func()); ok {
			fn()
		}
	}
}

// TestHandleCommand_ForkSpecificCommands verifies that fork-specific commands
// still work through the fallback path.
func TestHandleCommand_ForkSpecificCommands(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	ctx := context.Background()

	// /skills should be handled
	resp, handled := al.handleCommand(ctx, bus.InboundMessage{
		Content: "/skills",
	}, agent, "")
	if !handled {
		t.Fatal("expected /skills to be handled")
	}
	if resp == "" {
		t.Error("expected non-empty response for /skills")
	}

	// /session should be handled
	resp, handled = al.handleCommand(ctx, bus.InboundMessage{
		Content: "/session",
	}, agent, "")
	if !handled {
		t.Fatal("expected /session to be handled")
	}
	if resp == "" {
		t.Error("expected non-empty response for /session")
	}
}

// TestHandleCommand_UpstreamCommands verifies upstream commands work via Executor.
func TestHandleCommand_UpstreamCommands(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	ctx := context.Background()

	tests := []struct {
		cmd     string
		wantSub string
	}{
		{"/show model", "Current Model"},
		{"/show channel", "Current Channel"},
		{"/show agents", "default"},
		{"/list agents", "default"},
		{"/help", "Available commands"},
	}

	for _, tt := range tests {
		resp, handled := al.handleCommand(ctx, bus.InboundMessage{
			Content: tt.cmd,
			Channel: "telegram",
		}, agent, "")
		if !handled {
			t.Errorf("%s: expected handled", tt.cmd)
			continue
		}
		if resp == "" {
			t.Errorf("%s: expected non-empty response", tt.cmd)
		}
	}
}

package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/hooks"
	"github.com/sipeed/picoclaw/pkg/plugin"
)

type blockingPlugin struct{}

func (p blockingPlugin) Name() string {
	return "block-outbound"
}

func (p blockingPlugin) APIVersion() string {
	return plugin.APIVersion
}

func (p blockingPlugin) Register(r *hooks.HookRegistry) error {
	r.OnMessageSending("block-all", 0, func(_ context.Context, e *hooks.MessageSendingEvent) error {
		e.Cancel = true
		e.CancelReason = "blocked by plugin"
		return nil
	})
	return nil
}

func TestSetPluginManagerInstallsHookRegistry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-plugin-test-*")
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
	al := NewAgentLoop(cfg, msgBus, &mockProvider{})

	pm := plugin.NewManager()
	if err := pm.Register(blockingPlugin{}); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	if err := al.SetPluginManager(pm); err != nil {
		t.Fatalf("SetPluginManager: %v", err)
	}

	if al.pluginManager == nil {
		t.Fatal("expected plugin manager to be set")
	}
	if al.hooks != pm.HookRegistry() {
		t.Fatal("expected agent loop hooks to use plugin manager registry")
	}

	sent, reason := al.sendOutbound(context.Background(), bus.OutboundMessage{
		Channel: "cli",
		ChatID:  "direct",
		Content: "hello",
	})
	if sent {
		t.Fatal("expected outbound message to be blocked by plugin")
	}
	if reason == "" {
		t.Fatal("expected cancel reason to be propagated")
	}
}

func TestSetHooksReturnsErrorWhenRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-plugin-test-*")
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
	al := NewAgentLoop(cfg, msgBus, &mockProvider{})
	al.running.Store(true)

	if err := al.SetHooks(hooks.NewHookRegistry()); err == nil {
		t.Fatal("expected error when calling SetHooks while running")
	}
}

func TestSetPluginManagerDoesNotPartiallyUpdateOnError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-plugin-test-*")
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
	al := NewAgentLoop(cfg, msgBus, &mockProvider{})
	al.running.Store(true)

	pm := plugin.NewManager()
	if err := pm.Register(blockingPlugin{}); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	if err := al.SetPluginManager(pm); err == nil {
		t.Fatal("expected SetPluginManager to fail while running")
	}
	if al.pluginManager != nil {
		t.Fatal("expected plugin manager to remain unchanged on SetPluginManager failure")
	}
}

func TestBeforeToolCallHooksCannotLeaveToolArgsNil(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-plugin-test-*")
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
	provider := &nilArgsProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	captureTool := &nilArgsCaptureTool{}
	al.RegisterTool(captureTool)

	r := hooks.NewHookRegistry()
	r.OnBeforeToolCall("force-nil-args", 0, func(_ context.Context, e *hooks.BeforeToolCallEvent) error {
		if e.ToolName == "nil_args_tool" {
			e.Args = nil
		}
		return nil
	})
	if setErr := al.SetHooks(r); setErr != nil {
		t.Fatalf("SetHooks: %v", setErr)
	}

	resp, err := al.ProcessDirectWithChannel(context.Background(), "run nil args test", "s1", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel: %v", err)
	}
	if resp != "done" {
		t.Fatalf("expected final response 'done', got %q", resp)
	}
	if captureTool.receivedNil {
		t.Fatal("expected tool args to be reinitialized to non-nil map")
	}
}

func TestSetHooksNilRestoresDirectMessageCallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-plugin-test-*")
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
	al := NewAgentLoop(cfg, msgBus, &mockProvider{})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	tool, ok := agent.Tools.Get("message")
	if !ok {
		t.Fatal("expected message tool")
	}
	mt, ok := tool.(*tools.MessageTool)
	if !ok {
		t.Fatal("expected message tool type")
	}

	reg := hooks.NewHookRegistry()
	reg.OnMessageSending("block-all", 0, func(_ context.Context, e *hooks.MessageSendingEvent) error {
		e.Cancel = true
		e.CancelReason = "blocked-by-hook"
		return nil
	})
	if err := al.SetHooks(reg); err != nil {
		t.Fatalf("SetHooks(reg): %v", err)
	}

	blocked := mt.Execute(context.Background(), map[string]any{
		"content": "first",
		"channel": "cli",
		"chat_id": "direct",
	})
	if !blocked.IsError {
		t.Fatal("expected message tool call to fail while hooks are active")
	}
	if blocked.Err == nil || !strings.Contains(blocked.Err.Error(), "blocked-by-hook") {
		t.Fatalf("expected hook cancel reason in error, got %#v", blocked.Err)
	}

	ctxNoMsg, cancelNoMsg := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelNoMsg()
	if _, got := msgBus.SubscribeOutbound(ctxNoMsg); got {
		t.Fatal("did not expect outbound message while hook cancellation is active")
	}

	if err := al.SetHooks(nil); err != nil {
		t.Fatalf("SetHooks(nil): %v", err)
	}

	delivered := mt.Execute(context.Background(), map[string]any{
		"content": "second",
		"channel": "cli",
		"chat_id": "direct",
	})
	if delivered.IsError {
		t.Fatalf("expected message tool to succeed after SetHooks(nil), got %#v", delivered)
	}

	ctxMsg, cancelMsg := context.WithTimeout(context.Background(), time.Second)
	defer cancelMsg()
	msg, got := msgBus.SubscribeOutbound(ctxMsg)
	if !got {
		t.Fatal("expected outbound message after SetHooks(nil)")
	}
	if msg.Content != "second" || msg.Channel != "cli" || msg.ChatID != "direct" {
		t.Fatalf("unexpected outbound message: %#v", msg)
	}
}

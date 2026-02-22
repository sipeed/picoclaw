package demoplugin

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/hooks"
	"github.com/sipeed/picoclaw/pkg/plugin"
)

func TestPolicyDemoPluginBlocksConfiguredTool(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{
		BlockedTools: []string{"shell"},
	})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	e := &hooks.BeforeToolCallEvent{ToolName: "shell", Args: map[string]any{}, Channel: "cli"}
	pm.HookRegistry().TriggerBeforeToolCall(context.Background(), e)

	if !e.Cancel {
		t.Fatal("expected tool call to be canceled")
	}
	if e.CancelReason == "" {
		t.Fatal("expected cancel reason")
	}

	stats := p.Snapshot()
	if stats.BeforeToolCalls != 1 || stats.BlockedToolCalls != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestPolicyDemoPluginRedactsOutboundContent(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{
		RedactPrefixes: []string{"sk-"},
	})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	e := &hooks.MessageSendingEvent{Content: "token=sk-abc123"}
	pm.HookRegistry().TriggerMessageSending(context.Background(), e)

	if e.Cancel {
		t.Fatal("did not expect cancellation")
	}
	if e.Content != "token=[redacted]-abc123" {
		t.Fatalf("unexpected redaction result: %q", e.Content)
	}

	stats := p.Snapshot()
	if stats.MessageSends != 1 || stats.RedactedMessages != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestPolicyDemoPluginChannelAllowlist(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{
		ChannelToolAllowlist: map[string][]string{
			"telegram": {"web_search"},
		},
	})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	blocked := &hooks.BeforeToolCallEvent{ToolName: "shell", Args: map[string]any{}, Channel: "telegram"}
	pm.HookRegistry().TriggerBeforeToolCall(context.Background(), blocked)
	if !blocked.Cancel {
		t.Fatal("expected tool to be blocked by channel allowlist")
	}

	allowed := &hooks.BeforeToolCallEvent{ToolName: "web_search", Args: map[string]any{}, Channel: "telegram"}
	pm.HookRegistry().TriggerBeforeToolCall(context.Background(), allowed)
	if allowed.Cancel {
		t.Fatalf("did not expect allowlisted tool to be blocked: %s", allowed.CancelReason)
	}
}

func TestPolicyDemoPluginOutboundGuard(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{
		DenyOutboundPatterns: []string{"4111-1111-1111-1111", "@corp.internal"},
	})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	e := &hooks.MessageSendingEvent{Content: "card=4111-1111-1111-1111"}
	pm.HookRegistry().TriggerMessageSending(context.Background(), e)
	if !e.Cancel {
		t.Fatal("expected outbound message to be blocked")
	}
	if e.CancelReason == "" {
		t.Fatal("expected block reason")
	}

	stats := p.Snapshot()
	if stats.BlockedMessages != 1 {
		t.Fatalf("expected blocked message count to be 1, got %+v", stats)
	}
}

func TestPolicyDemoPluginNormalizesTimeoutArg(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{MaxToolTimeoutSecond: 30})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	e := &hooks.BeforeToolCallEvent{
		ToolName: "web_fetch",
		Channel:  "cli",
		Args: map[string]any{
			"timeout":         120,
			"timeout_seconds": 90.0,
		},
	}
	pm.HookRegistry().TriggerBeforeToolCall(context.Background(), e)

	if got, ok := e.Args["timeout"].(int); !ok || got != 30 {
		t.Fatalf("expected timeout to be clamped to 30, got %#v", e.Args["timeout"])
	}
	if got, ok := e.Args["timeout_seconds"].(int); !ok || got != 30 {
		t.Fatalf("expected timeout_seconds to be clamped to 30, got %#v", e.Args["timeout_seconds"])
	}
}

func TestPolicyDemoPluginAuditHooks(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	pm.HookRegistry().TriggerSessionStart(context.Background(), &hooks.SessionEvent{AgentID: "a1", SessionKey: "s1"})
	pm.HookRegistry().TriggerAfterToolCall(context.Background(), &hooks.AfterToolCallEvent{ToolName: "web_search", Duration: 45 * time.Millisecond})
	pm.HookRegistry().TriggerSessionEnd(context.Background(), &hooks.SessionEvent{AgentID: "a1", SessionKey: "s1"})

	stats := p.Snapshot()
	if stats.SessionStarts != 1 || stats.SessionEnds != 1 {
		t.Fatalf("unexpected session stats: %+v", stats)
	}
	if stats.AfterToolCalls != 1 || stats.TotalToolDuration != 45*time.Millisecond {
		t.Fatalf("unexpected after_tool_call stats: %+v", stats)
	}
}

func TestPolicyDemoPluginNoConfigNoEffect(t *testing.T) {
	pm := plugin.NewManager()
	p := NewPolicyDemoPlugin(PolicyDemoConfig{})
	if err := pm.Register(p); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	toolEvent := &hooks.BeforeToolCallEvent{ToolName: "shell", Args: map[string]any{}, Channel: "telegram"}
	pm.HookRegistry().TriggerBeforeToolCall(context.Background(), toolEvent)
	if toolEvent.Cancel {
		t.Fatal("did not expect cancellation with empty config")
	}

	msgEvent := &hooks.MessageSendingEvent{Content: "token=sk-abc123"}
	pm.HookRegistry().TriggerMessageSending(context.Background(), msgEvent)
	if msgEvent.Content != "token=sk-abc123" {
		t.Fatalf("did not expect content rewrite, got %q", msgEvent.Content)
	}
}

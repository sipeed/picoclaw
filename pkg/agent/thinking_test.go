package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestNormalizeThinkingLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "off", want: "off", ok: true},
		{input: "none", want: "off", ok: true},
		{input: "enable", want: "low", ok: true},
		{input: "adaptive", want: "adaptive", ok: true},
		{input: "invalid", ok: false},
	}
	for _, tt := range tests {
		got, ok := normalizeThinkingLevel(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("normalizeThinkingLevel(%q) = (%q,%v), want (%q,%v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseSupportedThinkingLevels(t *testing.T) {
	msg := `bad request: supported values: ["none","low","medium","high"]`
	got := parseSupportedThinkingLevels(msg)
	want := []string{"off", "low", "medium", "high"}
	if len(got) != len(want) {
		t.Fatalf("parseSupportedThinkingLevels len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseSupportedThinkingLevels[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestProcessMessage_ThinkCommandSetsSessionLevel(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	msg := bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "u1",
		ChatID:   "c1",
		Content:  "/think high",
	}
	resp, err := al.processMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if !strings.Contains(resp, "high") {
		t.Fatalf("response = %q, want contains high", resp)
	}

	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel: msg.Channel,
	})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("default agent is nil")
	}
	if got := agent.Sessions.GetThinkingLevel(route.SessionKey); got != "high" {
		t.Fatalf("session thinking = %q, want %q", got, "high")
	}
}

func TestProcessMessage_ThinkXHighRejectedForNonWhitelistModel(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	msg := bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "u1",
		ChatID:   "c1",
		Content:  "/think xhigh",
	}
	resp, err := al.processMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if !strings.Contains(strings.ToLower(resp), "xhigh") {
		t.Fatalf("response = %q, want xhigh hint", resp)
	}
}

func TestResolveThinkingLevelFromConfig(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Thinking: "medium",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "test-model", ThinkingLevel: "high"},
		},
	}
	agent := &AgentInstance{
		Model:    "test-model",
		Sessions: sessionManagerWithLevel(t, "", ""),
	}
	if got := resolveThinkingLevel(cfg, agent, "s1"); got != "high" {
		t.Fatalf("resolveThinkingLevel() = %q, want %q", got, "high")
	}
}

func sessionManagerWithLevel(t *testing.T, key, level string) *session.SessionManager {
	t.Helper()
	sm := session.NewSessionManager(t.TempDir())
	if key != "" {
		sm.SetThinkingLevel(key, level)
	}
	return sm
}

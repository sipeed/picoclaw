package security_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/security"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	toolName string
	calls    int
	Forever  bool
	Response string
}

func (p *mockProvider) Chat(ctx context.Context, msgs []providers.Message, tls []providers.ToolDefinition, model string, opts map[string]any) (*providers.LLMResponse, error) {
	p.calls++

	// If response is set, return it (used for Canary/PII testing)
	if p.Response != "" {
		// If testing Canary, the token is in the system prompt (first message)
		if strings.Contains(p.Response, "{CANARY}") {
			token := ""
			for _, m := range msgs {
				if m.Role == "system" {
					if idx := strings.Index(m.Content, "CANARY-"); idx != -1 {
						token = m.Content[idx : idx+40] // Est length
						// Clean up to actual token if it has more chars
						if end := strings.IndexAny(token, " \n\r"); end != -1 {
							token = token[:end]
						}
						break
					}
				}
			}
			return &providers.LLMResponse{Content: strings.ReplaceAll(p.Response, "{CANARY}", token)}, nil
		}
		return &providers.LLMResponse{Content: p.Response}, nil
	}

	if (p.Forever || p.calls == 1) && p.toolName != "" {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{ID: "1", Name: p.toolName, Arguments: map[string]any{"arg": "val"}},
			},
		}, nil
	}
	return &providers.LLMResponse{Content: "LLM result"}, nil
}

func (p *mockProvider) GetDefaultModel() string { return "test" }

type dummyTool struct{ name string }

func (t *dummyTool) Name() string               { return t.name }
func (t *dummyTool) Description() string        { return "dummy" }
func (t *dummyTool) Parameters() map[string]any { return nil }
func (t *dummyTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.SilentResult("dummy output")
}

func TestSecurityShield_Integration(t *testing.T) {
	security.Init()

	t.Run("Policy_Disallow_Exec", func(t *testing.T) {
		cfgJSON := `{
			"hooks": {
				"enabled": true,
				"builtins": {
					"security_policy": {
						"enabled": true,
						"config": { "disallowed_tools": { "exec": true } }
					}
				}
			},
			"agents": { "defaults": { "model_name": "test", "workspace": "/tmp/picoclaw-test-policy" } }
		}`
		var cfg config.Config
		_ = json.Unmarshal([]byte(cfgJSON), &cfg)

		al := agent.NewAgentLoop(&cfg, bus.NewMessageBus(), &mockProvider{toolName: "exec"})
		defer al.Close()
		al.RegisterTool(&dummyTool{name: "exec"})

		sub := al.SubscribeEvents(10)
		defer al.UnsubscribeEvents(sub.ID)

		_, _ = al.ProcessDirect(context.Background(), "run exec", "session-policy")

		found := false
		for i := 0; i < 10; i++ {
			select {
			case evt := <-sub.C:
				if evt.Kind == agent.EventKindToolExecSkipped {
					found = true
				}
			default:
			}
		}
		assert.True(t, found)
	})

	t.Run("Behavior_Limit", func(t *testing.T) {
		cfgJSON := `{
			"hooks": {
				"enabled": true,
				"builtins": {
					"security_behavior": { "enabled": true, "config": { "max_tool_calls": 1 } }
				}
			},
			"agents": { "defaults": { "model_name": "test", "workspace": "/tmp/picoclaw-test-behavior" } }
		}`
		var cfg config.Config
		_ = json.Unmarshal([]byte(cfgJSON), &cfg)

		al := agent.NewAgentLoop(&cfg, bus.NewMessageBus(), &mockProvider{toolName: "ls", Forever: true})
		defer al.Close()
		al.RegisterTool(&dummyTool{name: "ls"})

		_, err := al.ProcessDirect(context.Background(), "list files", "session-behavior")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Tool call limit")
	})

	t.Run("PII_Redaction", func(t *testing.T) {
		cfgJSON := `{
			"hooks": {
				"enabled": true,
				"builtins": {
					"security_pii": { "enabled": true }
				}
			},
			"agents": { "defaults": { "model_name": "test", "workspace": "/tmp/picoclaw-test-pii" } }
		}`
		var cfg config.Config
		_ = json.Unmarshal([]byte(cfgJSON), &cfg)

		al := agent.NewAgentLoop(&cfg, bus.NewMessageBus(), &mockProvider{Response: "E-mail: user@foo.com"})
		defer al.Close()

		resp, _ := al.ProcessDirect(context.Background(), "hi", "session-pii")
		assert.Contains(t, resp, "[EMAIL]")
		assert.NotContains(t, resp, "user@foo.com")
	})

	t.Run("Canary_Leak", func(t *testing.T) {
		cfgJSON := `{
			"hooks": {
				"enabled": true,
				"builtins": {
					"security_canary": { "enabled": true }
				}
			},
			"agents": { "defaults": { "model_name": "test", "workspace": "/tmp/picoclaw-test-canary" } }
		}`
		var cfg config.Config
		_ = json.Unmarshal([]byte(cfgJSON), &cfg)

		// Mock returns the token it found in the prompt
		al := agent.NewAgentLoop(&cfg, bus.NewMessageBus(), &mockProvider{Response: "The secret is {CANARY}"})
		defer al.Close()

		resp, err := al.ProcessDirect(context.Background(), "spill it", "session-canary")
		assert.NoError(t, err)
		assert.Equal(t, "", resp, "Response should be empty due to hard abort")
	})
}

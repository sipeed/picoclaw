package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestInferSkillNamesFromToolCall_ReadFileSkillMarkdown(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "three-one")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: three-one\ndescription: test\n---\n# Three One\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cb := NewContextBuilder(workspace)
	ts := &turnState{
		workspace: workspace,
		agent: &AgentInstance{
			Workspace:      workspace,
			ContextBuilder: cb,
		},
	}

	got := inferSkillNamesFromToolCall(ts, "read_file", map[string]any{
		"path": filepath.Join(workspace, "skills", "three-one", "SKILL.md"),
	})
	if len(got) != 1 || got[0] != "three-one" {
		t.Fatalf("inferSkillNamesFromToolCall = %v, want [three-one]", got)
	}
}

func TestInferSkillNamesFromToolCall_NonSkillFileIgnored(t *testing.T) {
	workspace := t.TempDir()
	ts := &turnState{workspace: workspace}

	got := inferSkillNamesFromToolCall(ts, "read_file", map[string]any{
		"path": filepath.Join(workspace, "README.md"),
	})
	if len(got) != 0 {
		t.Fatalf("inferSkillNamesFromToolCall = %v, want empty", got)
	}
}

func TestIsFatalMCPTransportErrorSummary(t *testing.T) {
	if !isFatalMCPTransportErrorSummary(`MCP tool execution failed: failed to call tool: connection closed: calling "tools/call": client is closing: invalid character 'ð' looking for beginning of value`) {
		t.Fatal("expected fatal MCP transport error to match")
	}
	if isFatalMCPTransportErrorSummary("MCP tool returned error: rate limited, retry later") {
		t.Fatal("expected normal MCP server error not to match fatal transport classifier")
	}
}

type repeatingFatalToolProvider struct {
	calls int
}

func (p *repeatingFatalToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	return &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{{
			ID:        "call-fatal",
			Name:      "mcp_gpt_researcher_quick_search",
			Arguments: map[string]any{"query": "modere refresh"},
		}},
	}, nil
}

func (p *repeatingFatalToolProvider) GetDefaultModel() string {
	return "fatal-loop-model"
}

type repeatingFatalTool struct{}

func (t *repeatingFatalTool) Name() string { return "mcp_gpt_researcher_quick_search" }
func (t *repeatingFatalTool) Description() string {
	return "Always fails with a fatal MCP transport error"
}
func (t *repeatingFatalTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type": "string",
			},
		},
	}
}
func (t *repeatingFatalTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	err := `MCP tool execution failed: failed to call tool: connection closed: calling "tools/call": client is closing: invalid character 'ð' looking for beginning of value`
	return tools.ErrorResult(err)
}

func TestRunAgentLoop_AbortsRepeatedFatalToolTransportErrors(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         2048,
				MaxToolIterations: 20,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &repeatingFatalToolProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(&repeatingFatalTool{})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	response, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-fatal-tool-loop",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run research",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		InboundContext: &bus.InboundContext{
			Channel:  "cli",
			ChatID:   "direct",
			ChatType: "direct",
			SenderID: "tester",
		},
		RouteResult: &routing.ResolvedRoute{
			AgentID:   "main",
			Channel:   "cli",
			AccountID: routing.DefaultAccountID,
			SessionPolicy: routing.SessionPolicy{
				Dimensions: []string{"sender"},
			},
			MatchedBy: "default",
		},
		SessionScope: &session.SessionScope{
			Version:    session.ScopeVersionV1,
			AgentID:    "main",
			Channel:    "cli",
			Account:    routing.DefaultAccountID,
			Dimensions: []string{"sender"},
			Values: map[string]string{
				"sender": "tester",
			},
		},
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got, want := response, "I hit repeated backend tool transport errors while using `mcp_gpt_researcher_quick_search` and stopped instead of retrying indefinitely. Please try again."; got != want {
		t.Fatalf("runAgentLoop() response = %q, want %q", got, want)
	}
	if provider.calls != repeatedFatalToolErrorStreakLimit {
		t.Fatalf("provider calls = %d, want %d", provider.calls, repeatedFatalToolErrorStreakLimit)
	}
}

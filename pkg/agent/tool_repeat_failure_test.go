package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// alwaysFailingToolProvider simulates an LLM that keeps requesting the same
// tool call over and over and never produces a final answer. This mirrors a
// real-world failure where the model retries a broken tool (e.g. a git command
// that fails because no credentials are configured) indefinitely.
type alwaysFailingToolProvider struct {
	mu       sync.Mutex
	callCount int
}

func (p *alwaysFailingToolProvider) Chat(
	ctx context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	p.callCount++
	p.mu.Unlock()
	return &providers.LLMResponse{
		Content: "let me try that again",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Name: "mock_fail",
				Arguments: map[string]any{
					"action": "git",
				},
			},
		},
		FinishReason: "tool_calls",
	}, nil
}

func (p *alwaysFailingToolProvider) GetDefaultModel() string {
	return "always-failing-model"
}

func (p *alwaysFailingToolProvider) callCountSnapshot() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.callCount
}

// alwaysFailTool is a tool that always returns the same error, exactly like
// the shell safety guard returning "Command blocked by safety guard" or git
// failing with "could not read Username".
type alwaysFailTool struct {
	mu        sync.Mutex
	execCount int
}

func (m *alwaysFailTool) Name() string { return "mock_fail" }

func (m *alwaysFailTool) Description() string {
	return "Always fails for testing the repeated-failure loop"
}

func (m *alwaysFailTool) Parameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": true,
	}
}

func (m *alwaysFailTool) Execute(_ context.Context, _ map[string]any) *tools.ToolResult {
	m.mu.Lock()
	m.execCount++
	m.mu.Unlock()
	return tools.ErrorResult("Command blocked by safety guard (dangerous pattern detected)")
}

func (m *alwaysFailTool) execCountSnapshot() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.execCount
}

// TestRepeatedIdenticalToolFailure_LoopsToMaxWithoutUserFeedback reproduces a
// real bug observed in production (issue: telegram messages that never get an
// answer). When a tool fails with the same error on every call, the agent loop
// keeps calling the LLM and re-executing the same broken tool until
// max_tool_iterations, with zero feedback to the user in between.
//
// Desired behavior: detect repeated identical failures and stop the turn early
// with a user-visible explanation instead of spinning silently to the cap.
func TestRepeatedIdenticalToolFailure_LoopsToMaxWithoutUserFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 5,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &alwaysFailingToolProvider{}
	failingTool := &alwaysFailTool{}

	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(failingTool)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	response, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-repeat-fail",
		Channel:         "telegram",
		ChatID:          "direct",
		UserMessage:     "run a git command",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		InboundContext: &bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "direct",
			ChatType: "direct",
			SenderID: "tester",
		},
		RouteResult: &routing.ResolvedRoute{
			AgentID:   "main",
			Channel:   "telegram",
			AccountID: routing.DefaultAccountID,
			SessionPolicy: routing.SessionPolicy{
				Dimensions: []string{"sender"},
			},
			MatchedBy: "default",
		},
		SessionScope: &session.SessionScope{
			Version:    session.ScopeVersionV1,
			AgentID:    "main",
			Channel:    "telegram",
			Account:    routing.DefaultAccountID,
			Dimensions: []string{"sender"},
			Values: map[string]string{
				"sender": "tester",
			},
		},
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}

	llmCalls := provider.callCountSnapshot()
	toolRuns := failingTool.execCountSnapshot()

	// The loop should not spin all the way to max_tool_iterations when the
	// tool fails identically every time. Currently it does (bug).
	if toolRuns >= 5 {
		t.Errorf("BUG: repeated identical tool failure looped to max_tool_iterations: tool executed %d times (cap=5), no early stop", toolRuns)
	}

	// The user should receive an explanation mentioning the actual failure,
	// not the generic max-iteration message.
	if response == toolLimitResponse {
		t.Errorf("BUG: user got generic max_tool_iterations message instead of a message explaining the repeated failure: %q", response)
	}

	t.Logf("llm calls=%d, tool executions=%d, final response=%q", llmCalls, toolRuns, response)
}

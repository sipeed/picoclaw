package agent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type parallelLoopMockProvider struct {
	callCount int
}

func (m *parallelLoopMockProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	m.callCount++
	if m.callCount == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{ID: "tc-1", Name: "slow_parallel", Arguments: map[string]any{}},
				{ID: "tc-2", Name: "fast_parallel", Arguments: map[string]any{}},
			},
		}, nil
	}

	if m.callCount == 2 {
		toolMessages := make([]providers.Message, 0, 2)
		for _, msg := range messages {
			if msg.Role == "tool" {
				toolMessages = append(toolMessages, msg)
			}
		}
		if len(toolMessages) != 2 {
			return nil, fmt.Errorf("tool message count = %d, want 2", len(toolMessages))
		}
		if toolMessages[0].ToolCallID != "tc-1" || toolMessages[0].Content != "slow-ok" {
			return nil, fmt.Errorf("first tool message = %+v, want tc-1/slow-ok", toolMessages[0])
		}
		if toolMessages[1].ToolCallID != "tc-2" || toolMessages[1].Content != "fast-ok" {
			return nil, fmt.Errorf("second tool message = %+v, want tc-2/fast-ok", toolMessages[1])
		}
		return &providers.LLMResponse{Content: "final-from-provider"}, nil
	}

	return &providers.LLMResponse{Content: "unexpected-extra-call"}, nil
}

func (m *parallelLoopMockProvider) GetDefaultModel() string {
	return "parallel-loop-mock"
}

type parallelTestTool struct {
	name   string
	result string
	delay  time.Duration
}

func (t *parallelTestTool) Name() string {
	return t.name
}

func (t *parallelTestTool) Description() string {
	return "parallel test tool"
}

func (t *parallelTestTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (t *parallelTestTool) ParallelPolicy() tools.ToolParallelPolicy {
	return tools.ToolParallelReadOnly
}

func (t *parallelTestTool) Execute(_ context.Context, _ map[string]any) *tools.ToolResult {
	if t.delay > 0 {
		time.Sleep(t.delay)
	}
	return tools.SilentResult(t.result)
}

func TestAgentLoop_RunLLMIteration_ParallelToolCallsPreserveOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-loop-parallel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.Model = "test-model"
	cfg.Agents.Defaults.MaxToolIterations = 4
	cfg.Orchestration.ToolCallsParallelEnabled = true
	cfg.Orchestration.MaxToolCallConcurrency = 8
	cfg.Orchestration.ParallelToolsMode = tools.ParallelToolsModeReadOnlyOnly

	msgBus := bus.NewMessageBus()
	provider := &parallelLoopMockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(&parallelTestTool{
		name:   "slow_parallel",
		result: "slow-ok",
		delay:  50 * time.Millisecond,
	})
	al.RegisterTool(&parallelTestTool{
		name:   "fast_parallel",
		result: "fast-ok",
		delay:  5 * time.Millisecond,
	})

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("default agent not found")
	}

	finalContent, iterations, err := al.runLLMIteration(
		context.Background(),
		agent,
		[]providers.Message{
			{Role: "system", Content: "you are a test assistant"},
			{Role: "user", Content: "run parallel tools"},
		},
		processOptions{
			SessionKey:   "parallel-session",
			Channel:      "cli",
			ChatID:       "direct",
			SenderID:     "tester",
			SendResponse: false,
		},
	)
	if err != nil {
		t.Fatalf("runLLMIteration() error = %v", err)
	}
	if iterations != 2 {
		t.Fatalf("iterations = %d, want 2", iterations)
	}
	if finalContent != "final-from-provider" {
		t.Fatalf("finalContent = %q, want %q", finalContent, "final-from-provider")
	}
}

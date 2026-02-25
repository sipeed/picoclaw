package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type scriptedToolLoopProvider struct {
	callCount int
}

func (p *scriptedToolLoopProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.callCount++
	if p.callCount == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{ID: "tc-1", Name: "slow", Arguments: map[string]any{}},
				{ID: "tc-2", Name: "fast", Arguments: map[string]any{}},
			},
		}, nil
	}

	if p.callCount == 2 {
		toolMessages := make([]providers.Message, 0, 2)
		for _, m := range messages {
			if m.Role == "tool" {
				toolMessages = append(toolMessages, m)
			}
		}
		if len(toolMessages) != 2 {
			return nil, fmt.Errorf("tool message count = %d, want 2", len(toolMessages))
		}
		if toolMessages[0].ToolCallID != "tc-1" || toolMessages[0].Content != "slow-result" {
			return nil, fmt.Errorf("first tool message = %+v, want tc-1/slow-result", toolMessages[0])
		}
		if toolMessages[1].ToolCallID != "tc-2" || toolMessages[1].Content != "fast-result" {
			return nil, fmt.Errorf("second tool message = %+v, want tc-2/fast-result", toolMessages[1])
		}
		return &providers.LLMResponse{Content: "final-answer"}, nil
	}

	return &providers.LLMResponse{Content: "unexpected-extra-call"}, nil
}

func (p *scriptedToolLoopProvider) GetDefaultModel() string {
	return "toolloop-scripted"
}

func TestRunToolLoop_ParallelToolCallsPreserveOrder(t *testing.T) {
	provider := &scriptedToolLoopProvider{}
	registry := NewToolRegistry()
	registry.Register(&executorMockTool{
		name:   "slow",
		policy: ToolParallelReadOnly,
		delay:  50 * time.Millisecond,
		result: SilentResult("slow-result"),
	})
	registry.Register(&executorMockTool{
		name:   "fast",
		policy: ToolParallelReadOnly,
		delay:  5 * time.Millisecond,
		result: SilentResult("fast-result"),
	})

	result, err := RunToolLoop(
		context.Background(),
		ToolLoopConfig{
			Provider:                 provider,
			Model:                    "mock-model",
			Tools:                    registry,
			MaxIterations:            4,
			ToolCallsParallelEnabled: true,
			MaxToolCallConcurrency:   8,
			ParallelToolsMode:        ParallelToolsModeReadOnlyOnly,
		},
		[]providers.Message{
			{Role: "system", Content: "you are a test assistant"},
			{Role: "user", Content: "run tools"},
		},
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("RunToolLoop() error = %v", err)
	}
	if result == nil {
		t.Fatal("RunToolLoop() returned nil result")
	}
	if result.Content != "final-answer" {
		t.Fatalf("result.Content = %q, want %q", result.Content, "final-answer")
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if len(result.Trace) != 2 {
		t.Fatalf("len(result.Trace) = %d, want 2", len(result.Trace))
	}
	if result.Trace[0].ToolCallID != "tc-1" || result.Trace[0].Result != "slow-result" {
		t.Fatalf("trace[0] = %+v, want tc-1/slow-result", result.Trace[0])
	}
	if result.Trace[1].ToolCallID != "tc-2" || result.Trace[1].Result != "fast-result" {
		t.Fatalf("trace[1] = %+v, want tc-2/fast-result", result.Trace[1])
	}
}

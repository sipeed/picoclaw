package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

type scriptedToolLoopProvider struct {
	responses   []providers.LLMResponse
	calls       int
	toolOutputs []string
}

func (p *scriptedToolLoopProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	for _, msg := range messages {
		if msg.Role == "tool" {
			p.toolOutputs = append(p.toolOutputs, msg.Content)
		}
	}
	if p.calls >= len(p.responses) {
		return &providers.LLMResponse{Content: "done"}, nil
	}
	response := p.responses[p.calls]
	p.calls++
	return &response, nil
}

func (p *scriptedToolLoopProvider) GetDefaultModel() string {
	return "scripted-test-model"
}

func TestRunToolLoop_TaskBoardScenarioProducesReadyAndNextOutputs(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tools := NewToolRegistry()
	tools.Register(NewTaskBoardTool(registry))

	provider := &scriptedToolLoopProvider{
		responses: []providers.LLMResponse{
			toolCallResponse("call-create", "task_board", map[string]any{
				"action":   "create",
				"board_id": "workflow-1",
				"title":    "Instagram recipe workflow",
			}),
			toolCallResponse("call-add-media", "task_board", map[string]any{
				"action":     "add_step",
				"board_id":   "workflow-1",
				"step_id":    "media-extract",
				"step_title": "Extract media",
				"owner":      "media",
				"task":       "Download the reel and extract the caption.",
			}),
			toolCallResponse("call-add-polish", "task_board", map[string]any{
				"action":     "add_step",
				"board_id":   "workflow-1",
				"step_id":    "polish-translation",
				"step_title": "Polish translation",
				"owner":      "research",
				"task":       "Polish the translated recipe.",
				"depends_on": []any{"media-extract"},
			}),
			toolCallResponse("call-update-media", "task_board", map[string]any{
				"action":   "update",
				"board_id": "workflow-1",
				"step_id":  "media-extract",
				"status":   "succeeded",
				"summary":  "caption extracted",
			}),
			toolCallResponse("call-ready", "task_board", map[string]any{
				"action":   "ready",
				"board_id": "workflow-1",
			}),
			toolCallResponse("call-next", "task_board", map[string]any{
				"action":   "next",
				"board_id": "workflow-1",
			}),
			{Content: "workflow board inspected"},
		},
	}

	result, err := RunToolLoop(
		context.Background(),
		ToolLoopConfig{
			Provider:      provider,
			Model:         provider.GetDefaultModel(),
			Tools:         tools,
			MaxIterations: 8,
		},
		[]providers.Message{{Role: "user", Content: "Build a recipe workflow board."}},
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("RunToolLoop() error = %v", err)
	}
	if result.Content != "workflow board inspected" {
		t.Fatalf("content = %q", result.Content)
	}

	combinedOutputs := strings.Join(provider.toolOutputs, "\n")
	if !strings.Contains(combinedOutputs, `"ready_steps"`) ||
		!strings.Contains(combinedOutputs, `"step_id": "polish-translation"`) ||
		!strings.Contains(combinedOutputs, `"action": "next"`) ||
		!strings.Contains(combinedOutputs, `"recommended_tool": "delegate"`) ||
		!strings.Contains(combinedOutputs, `"agent_id": "research"`) {
		t.Fatalf("tool loop did not expose expected task_board ready/next outputs:\n%s", combinedOutputs)
	}
}

func toolCallResponse(id, name string, args map[string]any) providers.LLMResponse {
	return providers.LLMResponse{
		Content: "calling " + name,
		ToolCalls: []providers.ToolCall{{
			ID:        id,
			Type:      "function",
			Name:      name,
			Arguments: args,
		}},
	}
}

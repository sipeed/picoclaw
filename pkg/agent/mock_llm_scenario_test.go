package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/testharness/llmscenario"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestMockLLMScenario_ProcessDirectExecutesToolAndReturnsFinalAnswer(t *testing.T) {
	const toolName = "scenario_extract_recipe"

	provider := llmscenario.NewScriptedProvider(
		"scenario-model",
		llmscenario.ProviderStep{
			Name: "request recipe extraction tool",
			Assert: func(call llmscenario.ProviderCall) error {
				if err := llmscenario.RequireToolDefinition(toolName)(call); err != nil {
					return err
				}
				if len(call.Messages) == 0 ||
					!strings.Contains(call.Messages[len(call.Messages)-1].Content, "Instagram caption") {
					return fmt.Errorf("first model call did not receive user prompt: %#v", call.Messages)
				}
				return nil
			},
			Response: llmscenario.ToolCallResponse(
				"I will extract the recipe.",
				llmscenario.ToolCall("call_extract_1", toolName, map[string]any{
					"url": "https://example.test/reel",
				}),
			),
		},
		llmscenario.ProviderStep{
			Name: "final answer after tool result",
			Assert: func(call llmscenario.ProviderCall) error {
				if err := llmscenario.RequireLastMessage("tool", "raspberries + chocolate")(call); err != nil {
					return err
				}
				return nil
			},
			Response: llmscenario.TextResponse("Final recipe: raspberries with dark chocolate."),
		},
	)

	al, agent, cleanup := newTurnCoordTestLoop(t, provider)
	defer cleanup()

	stub := llmscenario.NewStubTool(
		toolName,
		tools.NewToolResult("recipe extracted: raspberries + chocolate"),
	)
	agent.Tools.Register(stub)

	response, err := al.ProcessDirectWithChannel(
		context.Background(),
		"Extract recipe from this Instagram caption",
		"scenario-session",
		"telegram",
		"chat-123",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if response != "Final recipe: raspberries with dark chocolate." {
		t.Fatalf("response = %q", response)
	}
	if err := provider.AssertExhausted(); err != nil {
		t.Fatal(err)
	}

	toolCalls := stub.Calls()
	if len(toolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Args["url"] != "https://example.test/reel" {
		t.Fatalf("tool args = %#v", toolCalls[0].Args)
	}
	if toolCalls[0].Channel != "telegram" || toolCalls[0].ChatID != "chat-123" {
		t.Fatalf("tool context = channel %q chat %q, want telegram/chat-123", toolCalls[0].Channel, toolCalls[0].ChatID)
	}
}

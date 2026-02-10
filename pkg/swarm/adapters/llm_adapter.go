package adapters

import (
	"context"
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
)

type LLMAdapter struct {
	provider providers.LLMProvider
}

func NewLLMAdapter(provider providers.LLMProvider) *LLMAdapter {
	return &LLMAdapter{provider: provider}
}

func (a *LLMAdapter) Chat(ctx context.Context, messages []core.Message, tools []core.ToolDef, model string) (*core.LLMResponse, error) {
	pMsgs := make([]providers.Message, len(messages))
	for i, m := range messages {
		pMsgs[i] = providers.Message{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
	}

	pTools := make([]providers.ToolDefinition, len(tools))
	for i, t := range tools {
		pTools[i] = providers.ToolDefinition{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name: t.Name, Description: t.Description, Parameters: t.Parameters.(map[string]any),
			},
		}
	}

	resp, err := a.provider.Chat(ctx, pMsgs, pTools, model, nil)
	if err != nil {
		return nil, err
	}

	res := &core.LLMResponse{
		Content: resp.Content,
		Usage:   core.TokenUsage{Input: resp.Usage.PromptTokens, Output: resp.Usage.CompletionTokens},
	}

	for _, tc := range resp.ToolCalls {
		args, _ := json.Marshal(tc.Arguments)
		res.ToolCalls = append(res.ToolCalls, core.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: args})
	}

	return res, nil
}

func (a *LLMAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	if e, ok := a.provider.(interface {
		Embed(context.Context, string) ([]float32, error)
	}); ok {
		return e.Embed(ctx, text)
	}
	return nil, nil
}

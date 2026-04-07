package agent

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type mockProvider struct {
	streamChunks []string
}

func (m *mockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}

func (m *mockProvider) ChatStream(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
	onChunk func(accumulated string),
) (*providers.LLMResponse, error) {
	accumulated := ""
	for _, chunk := range m.streamChunks {
		accumulated += chunk
		if onChunk != nil {
			onChunk(accumulated)
		}
	}
	if accumulated == "" {
		accumulated = "Mock response"
		if onChunk != nil {
			onChunk(accumulated)
		}
	}
	return &providers.LLMResponse{
		Content:   accumulated,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

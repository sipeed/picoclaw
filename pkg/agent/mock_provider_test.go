package agent

import (
	"context"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type mockProvider struct {
	mu        sync.Mutex
	lastModel string
}

func (m *mockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	m.lastModel = model
	m.mu.Unlock()
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}

func (m *mockProvider) LastModel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastModel
}

package agent

import (
	"context"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type mockProvider struct {
	mu            sync.Mutex
	callCount     int
	responses     []providers.LLMResponse
	responseIndex int
}

func (m *mockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// If responses are configured, return them in sequence
	if len(m.responses) > 0 {
		if m.responseIndex >= len(m.responses) {
			// Cycle back or repeat last response
			m.responseIndex = len(m.responses) - 1
		}
		resp := m.responses[m.responseIndex]
		m.responseIndex++
		return &resp, nil
	}

	// Default mock response
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}

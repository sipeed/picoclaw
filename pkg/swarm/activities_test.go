// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider is a mock implementation of LLMProvider for testing
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	args := m.Called(ctx, messages, tools, model, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*providers.LLMResponse), args.Error(1)
}

func (m *MockLLMProvider) GetDefaultModel() string {
	args := m.Called()
	return args.String(0)
}

// AgentLoopInterface defines the methods we need from AgentLoop for testing
type AgentLoopInterface interface {
	ProcessDirect(ctx context.Context, content, sessionKey string) (string, error)
}

// MockAgentLoop is a mock implementation of AgentLoop interface for testing
type MockAgentLoop struct {
	mock.Mock
}

func (m *MockAgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	args := m.Called(ctx, content, sessionKey)
	return args.String(0), args.Error(1)
}

// mockAgentLoopWrapper wraps MockAgentLoop to conform to the *agent.AgentLoop type expectation
// In practice, you would create a proper interface or use dependency injection
// For now, we'll modify the activities to use an interface
func mockAgentLoopAsPtr(mock *MockAgentLoop) *agent.AgentLoop {
	// This is a workaround for testing - in production, use proper interfaces
	// For now, we just pass nil and test without agentLoop
	return nil
}

func TestNewActivities(t *testing.T) {
	cfg := &config.SwarmConfig{}

	activities := NewActivities(nil, nil, cfg, nil)
	assert.NotNil(t, activities)
	assert.Equal(t, cfg, activities.cfg)
}

func TestActivities_DecomposeTaskActivity_SimpleTask(t *testing.T) {
	mockProvider := new(MockLLMProvider)
	cfg := &config.SwarmConfig{}

	// Create activities with nil agentLoop for this test
	activities := NewActivities(mockProvider, nil, cfg, nil)

	task := &SwarmTask{
		ID:        "test-task-1",
		Type:      TaskTypeWorkflow,
		Prompt:    "Simple task that doesn't need decomposition",
		Capability: "general",
	}

	// Mock LLM to return non-decompose response
	mockProvider.On("Chat", mock.Anything, mock.Anything, mock.Anything, "gpt-4", mock.Anything).
		Return(&providers.LLMResponse{
			Content: `{"decompose": false, "reason": "Task is simple enough to execute directly"}`,
		}, nil)

	ctx := context.Background()
	subtasks, err := activities.DecomposeTaskActivity(ctx, task)

	require.NoError(t, err)
	assert.Nil(t, subtasks, "Simple tasks should not be decomposed")
	mockProvider.AssertExpectations(t)
}

func TestActivities_ExecuteDirectActivity(t *testing.T) {
	mockProvider := new(MockLLMProvider)
	cfg := &config.SwarmConfig{}

	// Use nil for agentLoop in this test
	activities := NewActivities(mockProvider, nil, cfg, nil)

	task := &SwarmTask{
		ID:        "test-task-2",
		Prompt:    "Execute this directly",
		Capability: "general",
	}

	ctx := context.Background()
	_, err := activities.ExecuteDirectActivity(ctx, task)

	// Should fail because agentLoop is nil
	assert.Error(t, err, "Should error when agentLoop is nil")
}

func TestActivities_SynthesizeResultsActivity(t *testing.T) {
	mockProvider := new(MockLLMProvider)
	cfg := &config.SwarmConfig{}

	activities := NewActivities(mockProvider, nil, cfg, nil)

	task := &SwarmTask{
		ID:        "test-task-3",
		Prompt:    "Original task",
		Capability: "general",
	}

	results := []string{
		"Result 1: Data analysis complete",
		"Result 2: Report generated",
	}

	// Mock LLM to return synthesis result
	mockProvider.On("Chat", mock.Anything, mock.Anything, mock.Anything, "gpt-4", mock.Anything).
		Return(&providers.LLMResponse{
			Content: "Synthesized final result combining all findings",
		}, nil)

	ctx := context.Background()
	synthesized, err := activities.SynthesizeResultsActivity(ctx, task, results)

	require.NoError(t, err)
	assert.Contains(t, synthesized, "Synthesized final result")
	mockProvider.AssertExpectations(t)
}

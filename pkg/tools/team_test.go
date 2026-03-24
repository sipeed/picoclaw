package tools

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
)

func TestUpgradeRegistryForConcurrency(t *testing.T) {
	// Create a standard tool registry
	original := NewToolRegistry()

	// Register a mix of tools: some upgradeable, some not
	readTool := NewReadFileTool("", false, MaxReadFileSize)
	listTool := NewListDirTool("", false) // Not upgradeable
	writeTool := NewWriteFileTool("", false)

	original.Register(readTool)
	original.Register(listTool)
	original.Register(writeTool)

	// Perform the upgrade
	upgraded := upgradeRegistryForConcurrency(original)

	// Verify count matches
	assert.Equal(t, len(original.ListTools()), len(upgraded.ListTools()), "Upgraded registry should have same number of tools")

	// Verify ReadFileTool got upgraded
	actualReadTool, ok := upgraded.Get("read_file")
	assert.True(t, ok)
	if upgradedRead, isUpgraded := actualReadTool.(*ReadFileTool); isUpgraded {
		_, isConcurrent := upgradedRead.fs.(*ConcurrentFS)
		assert.True(t, isConcurrent, "read_file should have been upgraded to ConcurrentFS")
	}

	// Verify WriteFileTool got upgraded
	actualWriteTool, ok := upgraded.Get("write_file")
	assert.True(t, ok)
	if upgradedWrite, isUpgraded := actualWriteTool.(*WriteFileTool); isUpgraded {
		_, isConcurrent := upgradedWrite.fs.(*ConcurrentFS)
		assert.True(t, isConcurrent, "write_file should have been upgraded to ConcurrentFS")
	}

	// Verify ListDirTool remained the same
	actualListTool, ok := upgraded.Get("list_dir")
	assert.True(t, ok)
	_, isListDir := actualListTool.(*ListDirTool)
	assert.True(t, isListDir, "list_dir should still be ListDirTool")

	// Double check list_dir doesn't randomly have ConcurrentFS injected
	if listImpl, ok := actualListTool.(*ListDirTool); ok {
		_, isConcurrent := listImpl.fs.(*ConcurrentFS)
		assert.False(t, isConcurrent, "list_dir should NOT have ConcurrentFS because it's not upgradeable")
	}

	// Double check original registry was entirely unmodified
	origReadTool, _ := original.Get("read_file")
	if origRead, _ := origReadTool.(*ReadFileTool); origRead != nil {
		_, isConcurrent := origRead.fs.(*ConcurrentFS)
		assert.False(t, isConcurrent, "Original registry components MUST REMAIN completely lock-free")
	}
}

func TestBuildWorkerConfig(t *testing.T) {
	// 1. Setup global config with model aliases
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			func() *config.ModelConfig {
				m := &config.ModelConfig{
					ModelName: "strong-model",
					Model:     "openai/gpt-4o",
					APIBase:   "https://api.openai.com/v1",
				}
				m.SetAPIKey("sk-test")
				return m
			}(),
			func() *config.ModelConfig {
				m := &config.ModelConfig{
					ModelName: "fast-model",
					Model:     "anthropic/claude-3-haiku",
					APIBase:   "https://api.anthropic.com/v1",
				}
				m.SetAPIKey("sk-test")
				return m
			}(),
			func() *config.ModelConfig {
				m := &config.ModelConfig{
					ModelName: "direct-id",
					Model:     "openai/gpt-3.5-turbo",
					APIBase:   "https://api.openai.com/v1",
				}
				m.SetAPIKey("sk-test")
				return m
			}(),
		},
	}

	// 2. Setup SubagentManager (needed by TeamTool for IsModelAllowed check)
	manager := NewSubagentManager(nil, "default-model", nil, "", config.TeamToolsConfig{
		AllowedModels: []config.TeamModelConfig{
			{Name: "fast-model", Tags: []string{"vision"}},
			{Name: "strong-model", Tags: []string{"coding"}},
			{Name: "direct-id", Tags: []string{"coding"}},
		},
	}, nil)

	// 3. Create TeamTool
	tool := NewTeamTool(manager, cfg)

	baseConfig := ToolLoopConfig{
		Model: "base-model",
	}

	tests := []struct {
		name          string
		memberModel   string
		expectedModel string
		expectError   bool
	}{
		{
			name:          "Resolve alias to actual ID",
			memberModel:   "strong-model",
			expectedModel: "gpt-4o",
			expectError:   false,
		},
		{
			name:          "Resolve another alias",
			memberModel:   "fast-model",
			expectedModel: "claude-3-haiku",
			expectError:   false,
		},
		{
			name:          "Resolve direct name if it matches an alias",
			memberModel:   "direct-id",
			expectedModel: "gpt-3.5-turbo",
			expectError:   false,
		},
		{
			name:          "Inherit base model if member model is empty",
			memberModel:   "",
			expectedModel: "base-model",
			expectError:   false,
		},
		{
			name:          "Error if model is not allowed",
			memberModel:   "forbidden-model",
			expectedModel: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := TeamMember{
				Model: tt.memberModel,
			}
			res, err := tool.buildWorkerConfig(baseConfig, nil, m)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedModel, res.Model)
				if tt.memberModel != "" {
					assert.NotNil(t, res.Provider, "Provider should be set when model is specified")
				} else {
					assert.Nil(t, res.Provider, "Provider should be nil (inherited from baseConfig)")
				}
			}
		})
	}
}

type mockProvider struct {
	responses []string
	callCount int
}

func (m *mockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]any) (*providers.LLMResponse, error) {
	if m.callCount >= len(m.responses) {
		return &providers.LLMResponse{Content: "Default response"}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &providers.LLMResponse{Content: resp}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}

func TestExecuteSequential(t *testing.T) {
	// 1. Setup mock provider to return specific outputs for each agent
	mock := &mockProvider{
		responses: []string{
			"Result from Agent A",
			"Derived result from Agent B",
		},
	}

	// 2. Setup TeamTool
	manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
	tool := NewTeamTool(manager, &config.Config{})

	baseConfig := ToolLoopConfig{
		Provider:      mock,
		Model:         "mock-model",
		MaxIterations: 1,
	}

	members := []TeamMember{
		{ID: "worker-A", Role: "Researcher", Task: "Research topic X"},
		{ID: "worker-B", Role: "Writer", Task: "Write summary of researcher output"},
	}

	// 3. Run sequential execution
	result := tool.executeSequential(context.Background(), baseConfig, members, 1000)

	// 4. Verify results
	assert.False(t, result.IsError, "Should not return error")
	assert.Contains(t, result.ForLLM, "Result from Agent A")
	assert.Contains(t, result.ForLLM, "Derived result from Agent B")
	assert.Equal(t, 2, mock.callCount, "Should have called mock provider exactly twice")
}

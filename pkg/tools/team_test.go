package tools

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
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
	assert.Equal(
		t,
		len(original.ListTools()),
		len(upgraded.ListTools()),
		"Upgraded registry should have same number of tools",
	)

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

func (m *mockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
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

// mockProviderWithID returns responses based on the role in the system message
// This is needed for DAG tests where execution order is non-deterministic
type mockProviderWithID struct {
	responses map[string]string
	mu        sync.Mutex
	callCount int
}

func (m *mockProviderWithID) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	// Extract role from system message to determine which response to return
	for _, msg := range messages {
		if msg.Role == "system" {
			if resp, ok := m.responses[msg.Content]; ok {
				return &providers.LLMResponse{Content: resp}, nil
			}
		}
	}
	return &providers.LLMResponse{Content: "Default response"}, nil
}

func (m *mockProviderWithID) GetDefaultModel() string {
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

func TestExecuteDAG(t *testing.T) {
	t.Run("Simple DAG with dependencies", func(t *testing.T) {
		// Setup: A -> C, B -> C (C depends on both A and B)
		mock := &mockProviderWithID{
			responses: map[string]string{
				"Data Collector A": "Data from A",
				"Data Collector B": "Data from B",
				"Aggregator":       "Combined result from C using A and B",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(), // Empty registry for test
		}

		members := []TeamMember{
			{ID: "A", Role: "Data Collector A", Task: "Collect data A"},
			{ID: "B", Role: "Data Collector B", Task: "Collect data B"},
			{ID: "C", Role: "Aggregator", Task: "Combine data", DependsOn: []string{"A", "B"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tool.executeDAG(ctx, cancel, baseConfig, members, 1000)

		assert.False(t, result.IsError, "Should not return error")
		assert.Contains(t, result.ForLLM, "Data from A")
		assert.Contains(t, result.ForLLM, "Data from B")
		assert.Contains(t, result.ForLLM, "Combined result from C")
		assert.Equal(t, 3, mock.callCount, "Should have called mock provider 3 times")
	})

	t.Run("Linear chain DAG", func(t *testing.T) {
		// Setup: A -> B -> C (linear dependency chain)
		mock := &mockProviderWithID{
			responses: map[string]string{
				"Step 1": "Step 1 output",
				"Step 2": "Step 2 output",
				"Step 3": "Step 3 output",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "A", Role: "Step 1", Task: "Do step 1"},
			{ID: "B", Role: "Step 2", Task: "Do step 2", DependsOn: []string{"A"}},
			{ID: "C", Role: "Step 3", Task: "Do step 3", DependsOn: []string{"B"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tool.executeDAG(ctx, cancel, baseConfig, members, 1000)

		assert.False(t, result.IsError)
		assert.Contains(t, result.ForLLM, "Step 1 output")
		assert.Contains(t, result.ForLLM, "Step 2 output")
		assert.Contains(t, result.ForLLM, "Step 3 output")
	})

	t.Run("Detect circular dependency", func(t *testing.T) {
		// Setup: A -> B -> C -> A (circular)
		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      &mockProvider{},
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "A", Role: "Worker A", Task: "Task A", DependsOn: []string{"C"}},
			{ID: "B", Role: "Worker B", Task: "Task B", DependsOn: []string{"A"}},
			{ID: "C", Role: "Worker C", Task: "Task C", DependsOn: []string{"B"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tool.executeDAG(ctx, cancel, baseConfig, members, 1000)

		assert.True(t, result.IsError, "Should detect circular dependency")
		assert.Contains(t, result.ForLLM, "cycle")
	})

	t.Run("Detect undefined dependency", func(t *testing.T) {
		// Setup: A depends on non-existent "X"
		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      &mockProvider{},
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "A", Role: "Worker A", Task: "Task A", DependsOn: []string{"X"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tool.executeDAG(ctx, cancel, baseConfig, members, 1000)

		assert.True(t, result.IsError, "Should detect undefined dependency")
		assert.Contains(t, result.ForLLM, "undefined member")
	})

	t.Run("Complex DAG with multiple roots", func(t *testing.T) {
		// Setup: A, B (roots) -> C, D -> E
		mock := &mockProviderWithID{
			responses: map[string]string{
				"Root A":   "Root A output",
				"Root B":   "Root B output",
				"Worker C": "C output",
				"Worker D": "D output",
				"Final":    "Final E output",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "A", Role: "Root A", Task: "Root task A"},
			{ID: "B", Role: "Root B", Task: "Root task B"},
			{ID: "C", Role: "Worker C", Task: "Task C", DependsOn: []string{"A"}},
			{ID: "D", Role: "Worker D", Task: "Task D", DependsOn: []string{"B"}},
			{ID: "E", Role: "Final", Task: "Final task", DependsOn: []string{"C", "D"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tool.executeDAG(ctx, cancel, baseConfig, members, 1000)

		assert.False(t, result.IsError)
		assert.Contains(t, result.ForLLM, "Root A output")
		assert.Contains(t, result.ForLLM, "Root B output")
		assert.Contains(t, result.ForLLM, "Final E output")
		assert.Equal(t, 5, mock.callCount)
	})
}

func TestExecuteEvaluatorOptimizer(t *testing.T) {
	t.Run("Pass on first attempt", func(t *testing.T) {
		mock := &mockProvider{
			responses: []string{
				"Perfect code implementation",
				"[PASS] The code is correct",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{
			MaxEvaluatorLoops: 3,
		}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write a function"},
			{ID: "evaluator", Role: "Code Reviewer", Task: "Review the code"},
		}

		result := tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)

		assert.False(t, result.IsError)
		assert.Contains(t, result.ForLLM, "Perfect code implementation")
		assert.Contains(t, result.ForLLM, "[PASS]")
		assert.Contains(t, result.ForUser, "passed on attempt 1")
		assert.Equal(t, 2, mock.callCount, "Should call worker once and evaluator once")
	})

	t.Run("Pass on second attempt after feedback", func(t *testing.T) {
		mock := &mockProvider{
			responses: []string{
				"Initial code with bug",
				"Missing error handling",
				"Fixed code with error handling",
				"[PASS] Now it's correct",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{
			MaxEvaluatorLoops: 3,
		}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write a function"},
			{ID: "evaluator", Role: "Code Reviewer", Task: "Review the code"},
		}

		result := tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)

		assert.False(t, result.IsError)
		assert.Contains(t, result.ForLLM, "Initial code with bug")
		assert.Contains(t, result.ForLLM, "Missing error handling")
		assert.Contains(t, result.ForLLM, "Fixed code with error handling")
		assert.Contains(t, result.ForLLM, "[PASS]")
		assert.Contains(t, result.ForUser, "passed on attempt 2")
		assert.Equal(t, 4, mock.callCount, "Should call worker twice and evaluator twice")
	})

	t.Run("Exhaust max loops without pass", func(t *testing.T) {
		mock := &mockProvider{
			responses: []string{
				"Attempt 1",
				"Still has issues",
				"Attempt 2",
				"Still not good",
				"Attempt 3",
				"Still failing",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{
			MaxEvaluatorLoops: 3,
		}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mock,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write a function"},
			{ID: "evaluator", Role: "Code Reviewer", Task: "Review the code"},
		}

		result := tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)

		assert.False(t, result.IsError, "Should not error, just report exhaustion")
		assert.Contains(t, result.ForLLM, "Maximum evaluation loops reached")
		assert.Contains(t, result.ForUser, "exhausted 3 attempts")
		assert.Equal(t, 6, mock.callCount, "Should call worker 3 times and evaluator 3 times")
	})

	t.Run("Require exactly two members", func(t *testing.T) {
		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      &mockProvider{},
			Model:         "mock-model",
			MaxIterations: 1,
		}

		// Test with 1 member
		members := []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write a function"},
		}

		result := tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)
		assert.True(t, result.IsError)
		assert.Contains(t, result.ForLLM, "exactly two members")

		// Test with 3 members
		members = []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write a function"},
			{ID: "evaluator", Role: "Reviewer", Task: "Review"},
			{ID: "extra", Role: "Extra", Task: "Extra task"},
		}

		result = tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)
		assert.True(t, result.IsError)
		assert.Contains(t, result.ForLLM, "exactly two members")
	})

	t.Run("Stateful worker memory across iterations", func(t *testing.T) {
		// This test verifies that the worker's message history is preserved
		// across iterations, allowing it to "remember" previous feedback
		mockWithMemory := &mockProvider{
			responses: []string{
				"First attempt",
				"Needs improvement: add validation",
				"Second attempt with validation",
				"[PASS] Good now",
			},
		}

		manager := NewSubagentManager(nil, "mock-model", nil, "", config.TeamToolsConfig{
			MaxEvaluatorLoops: 3,
		}, nil)
		tool := NewTeamTool(manager, &config.Config{})

		baseConfig := ToolLoopConfig{
			Provider:      mockWithMemory,
			Model:         "mock-model",
			MaxIterations: 1,
			Tools:         NewToolRegistry(),
		}

		members := []TeamMember{
			{ID: "worker", Role: "Coder", Task: "Write validation logic"},
			{ID: "evaluator", Role: "Reviewer", Task: "Check validation"},
		}

		result := tool.executeEvaluatorOptimizer(context.Background(), baseConfig, members, 1000)

		assert.False(t, result.IsError)
		assert.Contains(t, result.ForLLM, "First attempt")
		assert.Contains(t, result.ForLLM, "Needs improvement")
		assert.Contains(t, result.ForLLM, "Second attempt with validation")
		assert.Contains(t, result.ForLLM, "[PASS]")

		// Verify the worker was called twice (once initially, once after feedback)
		// and evaluator was called twice
		assert.Equal(t, 4, mockWithMemory.callCount)
	})
}

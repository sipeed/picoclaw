// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Activities bridges Temporal workflows with LLM and agent functionality
type Activities struct {
	provider  providers.LLMProvider
	agentLoop *agent.AgentLoop
	cfg       *config.SwarmConfig
	nodeInfo  *NodeInfo
}

// NewActivities creates a new Activities instance
func NewActivities(provider providers.LLMProvider, agentLoop *agent.AgentLoop, cfg *config.SwarmConfig, nodeInfo *NodeInfo) *Activities {
	return &Activities{
		provider:  provider,
		agentLoop: agentLoop,
		cfg:       cfg,
		nodeInfo:  nodeInfo,
	}
}

// DecomposeTaskActivity breaks down a complex task into subtasks using LLM analysis
func (a *Activities) DecomposeTaskActivity(ctx context.Context, task *SwarmTask) ([]*SwarmTask, error) {
	logger.InfoCF("swarm", "Decomposing task", map[string]interface{}{
		"task_id": task.ID,
		"prompt":  truncateString(task.Prompt, 100),
	})

	// Test mode: force decomposition for test tasks
	if strings.HasPrefix(task.Prompt, "PARALLEL") || strings.HasPrefix(task.Prompt, "TEST PARALLEL") {
		logger.InfoCF("swarm", "Test mode: forcing decomposition", map[string]interface{}{
			"task_id": task.ID,
		})
		// Extract the actual task from PARALLEL: prefix
		actualTask := strings.TrimPrefix(task.Prompt, "PARALLEL:")
		actualTask = strings.TrimPrefix(actualTask, "TEST PARALLEL:")
		actualTask = strings.TrimSpace(actualTask)

		// If no specific task given, use a default one
		if actualTask == "" {
			actualTask = "list current directory structure"
		}

		// Create test subtasks - each worker will report its own directory
		return []*SwarmTask{
			{
				ID:         task.ID + "-sub-1",
				ParentID:   task.ID,
				Type:       TaskTypeDirect,
				Priority:   task.Priority,
				Capability: "general",
				Prompt:     fmt.Sprintf(`IMPORTANT: First identify yourself by calling swarm_info tool to find your node ID and role. Then:
1. Use 'pwd' to show your current working directory
2. Use 'ls' to list your current directory contents
3. Report in format: "I am [node-id] ([role]), current directory: [path], contents: [directory listing]"

Task: %s`, actualTask),
				Context:    task.Context,
				Status:     TaskPending,
				CreatedAt:  time.Now().UnixMilli(),
				Timeout:    task.Timeout,
			},
			{
				ID:         task.ID + "-sub-2",
				ParentID:   task.ID,
				Type:       TaskTypeDirect,
				Priority:   task.Priority,
				Capability: "general",
				Prompt:     fmt.Sprintf(`IMPORTANT: First identify yourself by calling swarm_info tool to find your node ID and role. Then:
1. Use 'pwd' to show your current working directory
2. Use 'ls' to list your current directory contents
3. Report in format: "I am [node-id] ([role]), current directory: [path], contents: [directory listing]"

Task: %s`, actualTask),
				Context:    task.Context,
				Status:     TaskPending,
				CreatedAt:  time.Now().UnixMilli(),
				Timeout:    task.Timeout,
			},
		}, nil
	}

	// Build decomposition prompt
	decomposePrompt := fmt.Sprintf(`You are a task decomposition expert. Analyze the following task and determine if it should be decomposed into parallel subtasks.

TASK: %s

CAPABILITY REQUIRED: %s

IMPORTANT DECOMPOSITION RULES:
1. If the task mentions multiple files, multiple operations, or explicitly asks for parallel execution - ALWAYS DECOMPOSE
2. If the task contains the words "parallel", "concurrent", "together", or "simultaneously" - ALWAYS DECOMPOSE
3. Simple single-file operations can be executed directly

Respond with a JSON object. If the task is simple and can be executed directly, return:
{"decompose": false, "reason": "explanation"}

If the task should be decomposed, return:
{
  "decompose": true,
  "reason": "explanation of why decomposition helps",
  "subtasks": [
    {"id": "subtask-1", "prompt": "specific instruction", "capability": "capability_needed"},
    {"id": "subtask-2", "prompt": "specific instruction", "capability": "capability_needed"}
  ]
}

Keep subtasks focused and independently executable. Each subtask should produce a partial result that can be synthesized later.`,
		task.Prompt, task.Capability)

	// Call LLM for decomposition decision
	messages := []providers.Message{
		{Role: "user", Content: decomposePrompt},
	}

	// Get model from config, provider default, or fallback
	model := a.getModel()

	response, err := a.provider.Chat(ctx, messages, nil, model, map[string]interface{}{
		"max_tokens":  2048,
		"temperature": 0.3,
	})
	if err != nil {
		logger.WarnCF("swarm", "LLM decomposition failed, executing directly", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, nil // Fall back to direct execution
	}

	// Parse LLM response
	var result struct {
		Decompose bool           `json:"decompose"`
		Reason    string         `json:"reason"`
		Subtasks []SubtaskSpec   `json:"subtasks"`
	}

	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		logger.WarnCF("swarm", "Failed to parse decomposition response, executing directly", map[string]interface{}{
			"error":  err.Error(),
			"response": response.Content,
		})
		return nil, nil
	}

	if !result.Decompose {
		logger.InfoCF("swarm", "Task marked for direct execution", map[string]interface{}{
			"reason": result.Reason,
		})
		return nil, nil
	}

	// Create subtask structures
	subtasks := make([]*SwarmTask, len(result.Subtasks))
	for i, spec := range result.Subtasks {
		subtasks[i] = &SwarmTask{
			ID:         fmt.Sprintf("%s-sub-%d", task.ID, i+1),
			ParentID:   task.ID,
			Type:       TaskTypeDirect,
			Priority:   task.Priority,
			Capability: spec.Capability,
			Prompt:     spec.Prompt,
			Context:    task.Context,
			Status:     TaskPending,
			CreatedAt:  time.Now().UnixMilli(),
			Timeout:    task.Timeout,
		}
	}

	logger.InfoCF("swarm", "Task decomposed into subtasks", map[string]interface{}{
		"task_id":    task.ID,
		"subtasks":   len(subtasks),
		"reason":     result.Reason,
	})

	return subtasks, nil
}

// ExecuteDirectActivity executes a task directly on the local agent
func (a *Activities) ExecuteDirectActivity(ctx context.Context, task *SwarmTask) (string, error) {
	logger.InfoCF("swarm", "Executing task directly", map[string]interface{}{
		"task_id": task.ID,
		"prompt":  truncateString(task.Prompt, 100),
	})

	// Check if agentLoop is available
	if a.agentLoop == nil {
		return "", fmt.Errorf("agentLoop is not configured")
	}

	// Use agent loop's ProcessDirect for execution
	result, err := a.agentLoop.ProcessDirect(ctx, task.Prompt, "swarm:"+task.ID)
	if err != nil {
		logger.WarnCF("swarm", "Direct execution failed", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return "", err
	}

	logger.InfoCF("swarm", "Direct execution completed", map[string]interface{}{
		"task_id":       task.ID,
		"result_length": len(result),
	})

	return result, nil
}

// ExecuteSubtaskActivity executes a subtask, potentially dispatching to a specialist worker
func (a *Activities) ExecuteSubtaskActivity(ctx context.Context, task *SwarmTask) (string, error) {
	logger.InfoCF("swarm", "Executing subtask", map[string]interface{}{
		"task_id":  task.ID,
		"parent":   task.ParentID,
		"prompt":   truncateString(task.Prompt, 100),
		"capability": task.Capability,
	})

	// For now, execute locally using the agent loop
	// In a full implementation, this would check for specialist workers
	// and dispatch to the appropriate node based on capability
	result, err := a.agentLoop.ProcessDirect(ctx, task.Prompt, "swarm:subtask:"+task.ID)
	if err != nil {
		logger.WarnCF("swarm", "Subtask execution failed", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return "", err
	}

	logger.InfoCF("swarm", "Subtask completed", map[string]interface{}{
		"task_id":       task.ID,
		"result_length": len(result),
	})

	// Prefix result with node identifier for synthesis
	nodeName := a.getNodeName()
	result = fmt.Sprintf("=== %s ===\n%s", nodeName, result)

	return result, nil
}

// getNodeName returns a human-readable name for this node
func (a *Activities) getNodeName() string {
	if a.nodeInfo == nil {
		return "unknown-node"
	}

	// Try to use SID (service ID) as the primary identifier
	if sid := a.nodeInfo.Metadata["sid"]; sid != "" {
		return sid
	}

	// Fall back to node ID (shortened)
	nodeID := a.nodeInfo.ID
	if len(nodeID) > 12 {
		nodeID = nodeID[:12]
	}
	return nodeID
}

// SynthesizeResultsActivity combines subtask results into a coherent final response
func (a *Activities) SynthesizeResultsActivity(ctx context.Context, task *SwarmTask, results []string) (string, error) {
	logger.InfoCF("swarm", "Synthesizing results", map[string]interface{}{
		"task_id":  task.ID,
		"results":  len(results),
	})

	// Build synthesis prompt
	var resultsBlock strings.Builder
	resultsBlock.WriteString(fmt.Sprintf("ORIGINAL TASK: %s\n\n", task.Prompt))
	resultsBlock.WriteString("SUBTASK RESULTS:\n\n")

	for i, result := range results {
		// Skip failed results
		if strings.Contains(result, "[FAILED]") {
			resultsBlock.WriteString(fmt.Sprintf("[Result %d - FAILED]\n%s\n\n", i+1, result))
			continue
		}
		// Truncate very long results for the synthesis prompt
		truncated := result
		if len(result) > 2000 {
			truncated = result[:2000] + "\n...[truncated]"
		}
		resultsBlock.WriteString(fmt.Sprintf("[Result %d]\n%s\n\n", i+1, truncated))
	}

	synthesisPrompt := fmt.Sprintf(`You are synthesizing results from parallel task execution.

%s

Your job:
1. Analyze all subtask results
2. Identify key findings and insights
3. Create a coherent, unified response that addresses the original task
4. If any results failed or contain errors, acknowledge them appropriately
5. Present the final answer in a clear, well-structured format

Provide a comprehensive synthesis that directly addresses the original task.`, resultsBlock.String())

	messages := []providers.Message{
		{Role: "user", Content: synthesisPrompt},
	}

	// Get model from config, provider default, or fallback
	model := a.getModel()

	response, err := a.provider.Chat(ctx, messages, nil, model, map[string]interface{}{
		"max_tokens":  4096,
		"temperature": 0.5,
	})
	if err != nil {
		logger.WarnCF("swarm", "LLM synthesis failed, returning error for Temporal retry", map[string]interface{}{
			"error": err.Error(),
		})
		// Return error to trigger Temporal retry
		// Temporal will retry up to MaximumAttempts (3) before giving up
		return "", fmt.Errorf("LLM synthesis failed: %w", err)
	}

	logger.InfoCF("swarm", "Synthesis completed", map[string]interface{}{
		"task_id":       task.ID,
		"final_length":  len(response.Content),
	})

	return response.Content, nil
}

// SubtaskSpec defines a subtask from decomposition
type SubtaskSpec struct {
	ID         string `json:"id"`
	Prompt     string `json:"prompt"`
	Capability string `json:"capability"`
}

// truncateString limits string length for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getModel returns the LLM model to use for swarm tasks
// Priority: config > provider default > fallback
func (a *Activities) getModel() string {
	// 1. Check if model is configured in swarm config
	if a.cfg != nil && a.cfg.Temporal.Model != "" {
		return a.cfg.Temporal.Model
	}

	// 2. Fall back to provider's default model
	if a.provider != nil {
		if model := a.provider.GetDefaultModel(); model != "" {
			return model
		}
	}

	// 3. Final fallback
	return "gpt-4"
}

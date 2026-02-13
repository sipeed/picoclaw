// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SwarmWorkflow is the main workflow for task orchestration.
// It decomposes a task, runs subtasks in parallel, and synthesizes results.
func SwarmWorkflow(ctx workflow.Context, task *SwarmTask) (string, error) {
	wfLogger := workflow.GetLogger(ctx)
	wfLogger.Info("Starting swarm workflow", "task_id", task.ID)

	// Step 1: Decompose task into subtasks
	ctx1 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})

	var subtasks []*SwarmTask
	err := workflow.ExecuteActivity(ctx1, DecomposeTaskActivity, task).Get(ctx, &subtasks)
	if err != nil {
		return "", fmt.Errorf("failed to decompose task: %w", err)
	}

	// If no subtasks, execute directly
	if len(subtasks) == 0 {
		var result string
		err := workflow.ExecuteActivity(ctx1, ExecuteDirectActivity, task).Get(ctx, &result)
		if err != nil {
			return "", err
		}
		return result, nil
	}

	// Step 2: Execute subtasks in parallel
	futures := make([]workflow.Future, len(subtasks))
	for i, sub := range subtasks {
		ctx2 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    5 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumAttempts:    3,
			},
		})
		futures[i] = workflow.ExecuteActivity(ctx2, ExecuteSubtaskActivity, sub)
	}

	// Collect results
	results := make([]string, len(futures))
	for i, f := range futures {
		var result string
		if err := f.Get(ctx, &result); err != nil {
			wfLogger.Warn("Subtask failed", "error", err, "index", i)
			results[i] = fmt.Sprintf("[FAILED] %v", err)
		} else {
			results[i] = result
		}
	}

	// Step 3: Synthesize final result
	ctx3 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	var finalResult string
	err = workflow.ExecuteActivity(ctx3, SynthesizeResultsActivity, task, results).Get(ctx, &finalResult)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize results: %w", err)
	}

	return finalResult, nil
}

// Activities - these are executed by workers

// DecomposeTaskActivity breaks down a complex task into subtasks.
// The actual LLM-based decomposition logic will be injected via the Activities struct.
func DecomposeTaskActivity(ctx context.Context, task *SwarmTask) ([]*SwarmTask, error) {
	activity.RecordHeartbeat(ctx, "decomposing task")
	// Default behavior: no decomposition, execute directly
	// In production, the coordinator will register a custom activity that uses
	// the LLM to analyze and decompose the task
	return nil, nil
}

// ExecuteDirectActivity executes a simple task directly on the local agent
func ExecuteDirectActivity(ctx context.Context, task *SwarmTask) (string, error) {
	activity.RecordHeartbeat(ctx, "executing task directly")
	// This will use the local agent loop to process
	// The actual implementation is injected when registering activities
	return fmt.Sprintf("Executed task: %s", task.Prompt), nil
}

// ExecuteSubtaskActivity executes a subtask on a worker node
func ExecuteSubtaskActivity(ctx context.Context, task *SwarmTask) (string, error) {
	activity.RecordHeartbeat(ctx, "processing subtask")
	// This will dispatch to appropriate worker via NATS
	// The actual implementation is injected when registering activities
	return fmt.Sprintf("Subtask result: %s", task.Prompt), nil
}

// SynthesizeResultsActivity combines subtask results into final output
func SynthesizeResultsActivity(ctx context.Context, task *SwarmTask, results []string) (string, error) {
	activity.RecordHeartbeat(ctx, "synthesizing results")
	// Default: concatenate results
	// In production, uses coordinator's LLM to create coherent response
	combined := ""
	for i, r := range results {
		combined += fmt.Sprintf("=== Result %d ===\n%s\n\n", i+1, r)
	}
	return combined, nil
}

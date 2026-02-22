// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"time"

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
	// Add retry policy for LLM synthesis failures (API errors, rate limits, etc.)
	ctx3 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	})

	var finalResult string
	err = workflow.ExecuteActivity(ctx3, SynthesizeResultsActivity, task, results).Get(ctx, &finalResult)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize results: %w", err)
	}

	return finalResult, nil
}

// Note: Activity implementations are in activities.go as methods on the Activities struct.
// These global functions are retained for Temporal registration but delegate to the Activities struct.

// Global activity wrappers for Temporal registration
// These are registered by the Temporal worker and dispatch to the Activities struct

// ActivitiesRegistry holds the global activities instance for Temporal workflow execution
// This is set before starting the Temporal worker and accessed by the activity functions
var ActivitiesRegistry *Activities

// RegisterActivities registers the activities instance for workflow use
func RegisterActivities(activities *Activities) {
	ActivitiesRegistry = activities
}

// DecomposeTaskActivity is a wrapper for the Activities method
func DecomposeTaskActivity(ctx context.Context, task *SwarmTask) ([]*SwarmTask, error) {
	if ActivitiesRegistry == nil {
		return nil, fmt.Errorf("activities not initialized")
	}
	return ActivitiesRegistry.DecomposeTaskActivity(ctx, task)
}

// ExecuteDirectActivity is a wrapper for the Activities method
func ExecuteDirectActivity(ctx context.Context, task *SwarmTask) (string, error) {
	if ActivitiesRegistry == nil {
		return "", fmt.Errorf("activities not initialized")
	}
	return ActivitiesRegistry.ExecuteDirectActivity(ctx, task)
}

// ExecuteSubtaskActivity is a wrapper for the Activities method
func ExecuteSubtaskActivity(ctx context.Context, task *SwarmTask) (string, error) {
	if ActivitiesRegistry == nil {
		return "", fmt.Errorf("activities not initialized")
	}
	return ActivitiesRegistry.ExecuteSubtaskActivity(ctx, task)
}

// SynthesizeResultsActivity is a wrapper for the Activities method
func SynthesizeResultsActivity(ctx context.Context, task *SwarmTask, results []string) (string, error) {
	if ActivitiesRegistry == nil {
		return "", fmt.Errorf("activities not initialized")
	}
	return ActivitiesRegistry.SynthesizeResultsActivity(ctx, task, results)
}

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// TemporalClient wraps the Temporal SDK client
type TemporalClient struct {
	client    client.Client
	worker    worker.Worker
	cfg       *config.TemporalConfig
	taskQueue string
	connected bool
}

// NewTemporalClient creates a new Temporal client
func NewTemporalClient(cfg *config.TemporalConfig) *TemporalClient {
	return &TemporalClient{
		cfg:       cfg,
		taskQueue: cfg.TaskQueue,
	}
}

// Connect establishes connection to Temporal server
func (tc *TemporalClient) Connect(ctx context.Context) error {
	c, err := client.Dial(client.Options{
		HostPort:  tc.cfg.Host,
		Namespace: tc.cfg.Namespace,
	})
	if err != nil {
		// Temporal is optional - log warning but don't fail
		logger.WarnCF("swarm", "Failed to connect to Temporal (workflows disabled)", map[string]interface{}{
			"host":  tc.cfg.Host,
			"error": err.Error(),
		})
		return nil
	}

	tc.client = c
	tc.connected = true
	logger.InfoCF("swarm", "Connected to Temporal", map[string]interface{}{
		"host":      tc.cfg.Host,
		"namespace": tc.cfg.Namespace,
	})
	return nil
}

// IsConnected returns true if connected to Temporal
func (tc *TemporalClient) IsConnected() bool {
	return tc.connected
}

// StartWorker starts a Temporal worker that processes workflows and activities
func (tc *TemporalClient) StartWorker(ctx context.Context, workflows []interface{}, activities *Activities) error {
	if !tc.connected {
		logger.WarnC("swarm", "Temporal not connected, skipping worker start")
		return nil
	}

	// Register activities globally for workflow access
	RegisterActivities(activities)

	w := worker.New(tc.client, tc.taskQueue, worker.Options{})

	// Register workflows
	for _, wf := range workflows {
		w.RegisterWorkflow(wf)
	}

	// Register activity functions
	w.RegisterActivity(DecomposeTaskActivity)
	w.RegisterActivity(ExecuteDirectActivity)
	w.RegisterActivity(ExecuteSubtaskActivity)
	w.RegisterActivity(SynthesizeResultsActivity)

	tc.worker = w

	// Start worker in background
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			logger.ErrorCF("swarm", "Temporal worker error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	logger.InfoCF("swarm", "Temporal worker started", map[string]interface{}{
		"task_queue": tc.taskQueue,
	})
	return nil
}

// StartWorkflow starts a new workflow execution
func (tc *TemporalClient) StartWorkflow(ctx context.Context, workflowType string, task *SwarmTask) (string, error) {
	if !tc.connected {
		return "", fmt.Errorf("temporal not connected")
	}

	workflowTimeout, _ := time.ParseDuration(tc.cfg.WorkflowTimeout)
	if workflowTimeout == 0 {
		workflowTimeout = 30 * time.Minute
	}

	options := client.StartWorkflowOptions{
		ID:                       task.ID,
		TaskQueue:                tc.taskQueue,
		WorkflowExecutionTimeout: workflowTimeout,
	}

	we, err := tc.client.ExecuteWorkflow(ctx, options, workflowType, task)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	logger.InfoCF("swarm", "Workflow started", map[string]interface{}{
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
		"task_id":     task.ID,
	})

	return we.GetID(), nil
}

// GetWorkflowResult waits for and returns workflow result
func (tc *TemporalClient) GetWorkflowResult(ctx context.Context, workflowID string) (string, error) {
	if !tc.connected {
		return "", fmt.Errorf("temporal not connected")
	}

	run := tc.client.GetWorkflow(ctx, workflowID, "")

	var result string
	if err := run.Get(ctx, &result); err != nil {
		return "", err
	}
	return result, nil
}

// Stop stops the Temporal client and worker
func (tc *TemporalClient) Stop() {
	if tc.worker != nil {
		tc.worker.Stop()
	}
	if tc.client != nil {
		tc.client.Close()
	}
	tc.connected = false
}

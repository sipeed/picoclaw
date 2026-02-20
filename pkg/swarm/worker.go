// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Worker executes tasks received from the swarm
type Worker struct {
	bridge       *NATSBridge
	temporal     *TemporalClient
	agentLoop    *agent.AgentLoop
	provider     providers.LLMProvider
	nodeInfo     *NodeInfo
	cfg          *config.SwarmConfig
	taskQueue    chan *SwarmTask
	activeTasks  sync.Map
	running      atomic.Bool
	tasksRunning atomic.Int32 // atomic counter for thread-safe tracking
}

// NewWorker creates a new worker
func NewWorker(
	cfg *config.SwarmConfig,
	bridge *NATSBridge,
	temporal *TemporalClient,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
	nodeInfo *NodeInfo,
) *Worker {
	return &Worker{
		bridge:    bridge,
		temporal:  temporal,
		agentLoop: agentLoop,
		provider:  provider,
		nodeInfo:  nodeInfo,
		cfg:       cfg,
		taskQueue: make(chan *SwarmTask, cfg.MaxConcurrent*2),
	}
}

// Start begins the worker
func (w *Worker) Start(ctx context.Context) error {
	w.running.Store(true)

	// Set up task handler
	w.bridge.SetOnTaskReceived(func(task *SwarmTask) {
		select {
		case w.taskQueue <- task:
		default:
			logger.WarnCF("swarm", "Task queue full, rejecting task", map[string]interface{}{
				"task_id": task.ID,
			})
		}
	})

	// Start task processors
	for i := 0; i < w.cfg.MaxConcurrent; i++ {
		go w.processTaskLoop(ctx)
	}

	logger.InfoCF("swarm", "Worker started", map[string]interface{}{
		"node_id":        w.nodeInfo.ID,
		"max_concurrent": w.cfg.MaxConcurrent,
		"capabilities":   w.nodeInfo.Capabilities,
	})

	return nil
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.running.Store(false)
	close(w.taskQueue)
}

func (w *Worker) processTaskLoop(ctx context.Context) {
	for {
		select {
		case task, ok := <-w.taskQueue:
			if !ok {
				return
			}
			w.executeTask(ctx, task)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) executeTask(ctx context.Context, task *SwarmTask) {
	logger.InfoCF("swarm", "Executing task", map[string]interface{}{
		"task_id":    task.ID,
		"capability": task.Capability,
	})

	// Track active task using atomic operations for thread safety
	w.activeTasks.Store(task.ID, task)
	current := w.tasksRunning.Add(1)
	w.nodeInfo.TasksRunning = int(current)
	w.updateLoad()

	defer func() {
		w.activeTasks.Delete(task.ID)
		current := w.tasksRunning.Add(-1)
		w.nodeInfo.TasksRunning = int(current)
		w.updateLoad()
	}()

	// Create timeout context
	timeout := time.Duration(task.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Send progress updates periodically
	progressDone := make(chan struct{})
	go w.sendProgressUpdates(taskCtx, task, progressDone)

	// Execute using local agent
	result, err := w.agentLoop.ProcessDirect(taskCtx, task.Prompt, "swarm:"+task.ID)

	close(progressDone)

	// Send result
	taskResult := &TaskResult{
		TaskID:      task.ID,
		NodeID:      w.nodeInfo.ID,
		CompletedAt: time.Now().UnixMilli(),
	}

	if err != nil {
		taskResult.Status = string(TaskFailed)
		taskResult.Error = err.Error()
		logger.WarnCF("swarm", "Task execution failed", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	} else {
		taskResult.Status = string(TaskDone)
		taskResult.Result = result
		logger.InfoCF("swarm", "Task completed", map[string]interface{}{
			"task_id":       task.ID,
			"result_length": len(result),
		})
	}

	if err := w.bridge.PublishTaskResult(taskResult); err != nil {
		logger.ErrorCF("swarm", "Failed to publish task result", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	}
}

// sendProgressUpdates sends periodic progress updates for a running task
// The progress is estimated based on elapsed time relative to the task timeout
func (w *Worker) sendProgressUpdates(ctx context.Context, task *SwarmTask, done chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	// Determine timeout for progress estimation
	timeout := time.Duration(task.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	for {
		select {
		case <-ticker.C:
			// Estimate progress based on elapsed time
			elapsed := time.Since(startTime)
			progress := float64(elapsed.Milliseconds()) / float64(timeout.Milliseconds())

			// Clamp progress between 0.1 and 0.9 (we don't know actual LLM progress)
			if progress < 0.1 {
				progress = 0.1
			} else if progress > 0.9 {
				progress = 0.9
			}

			// Generate contextual message based on progress
			message := "processing"
			if progress < 0.3 {
				message = "initializing"
			} else if progress < 0.6 {
				message = "processing"
			} else if progress < 0.9 {
				message = "finalizing"
			}

			progressUpdate := &TaskProgress{
				TaskID:   task.ID,
				NodeID:   w.nodeInfo.ID,
				Progress: progress,
				Message:  message,
			}
			if err := w.bridge.PublishTaskProgress(progressUpdate); err != nil {
				logger.DebugCF("swarm", "Failed to publish progress", map[string]interface{}{
					"error": err.Error(),
				})
			}
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) updateLoad() {
	tasksRunning := int(w.tasksRunning.Load())
	if w.cfg.MaxConcurrent > 0 {
		w.nodeInfo.Load = float64(tasksRunning) / float64(w.cfg.MaxConcurrent)
	}
	if tasksRunning >= w.cfg.MaxConcurrent {
		w.nodeInfo.Status = StatusBusy
	} else {
		w.nodeInfo.Status = StatusOnline
	}
	w.nodeInfo.TasksRunning = tasksRunning
}

// ActiveTaskCount returns the number of currently executing tasks
func (w *Worker) ActiveTaskCount() int {
	count := 0
	w.activeTasks.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// RecoverFromCheckpoint resumes task execution from a saved checkpoint
func (w *Worker) RecoverFromCheckpoint(ctx context.Context, task *SwarmTask, checkpoint *TaskCheckpoint) (string, error) {
	logger.InfoCF("swarm", "Recovering task from checkpoint", map[string]interface{}{
		"task_id":       task.ID,
		"checkpoint_id": checkpoint.CheckpointID,
		"progress":      checkpoint.Progress,
	})

	// Build recovery prompt with checkpoint context
	recoveryPrompt := fmt.Sprintf(`[TASK RECOVERY MODE]

You are resuming execution of a task that was interrupted.

Original Task: %s

Checkpoint Progress: %.0f%%

Partial Work Completed:
%s

Previous Context:
- Last checkpoint was taken by node: %s
- Checkpoint type: %s
- Timestamp: %s

Continue from where the previous execution left off. Use the partial result as context and complete the remaining work.`,
		task.Prompt,
		checkpoint.Progress*100,
		checkpoint.PartialResult,
		checkpoint.NodeID,
		string(checkpoint.Type),
		time.UnixMilli(checkpoint.Timestamp).Format(time.RFC3339),
	)

	// Execute with the recovery prompt
	result, err := w.agentLoop.ProcessDirect(ctx, recoveryPrompt, "swarm:recovery:"+task.ID)
	if err != nil {
		logger.WarnCF("swarm", "Checkpoint recovery failed", map[string]interface{}{
			"task_id":       task.ID,
			"checkpoint_id": checkpoint.CheckpointID,
			"error":         err.Error(),
		})
		return "", err
	}

	logger.InfoCF("swarm", "Task recovery completed", map[string]interface{}{
		"task_id":       task.ID,
		"checkpoint_id": checkpoint.CheckpointID,
		"result_length": len(result),
	})

	return result, nil
}

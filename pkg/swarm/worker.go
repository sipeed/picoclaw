// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
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
	bridge      *NATSBridge
	temporal    *TemporalClient
	agentLoop   *agent.AgentLoop
	provider    providers.LLMProvider
	nodeInfo    *NodeInfo
	cfg         *config.SwarmConfig
	taskQueue   chan *SwarmTask
	activeTasks sync.Map
	running     atomic.Bool
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

	// Track active task
	w.activeTasks.Store(task.ID, task)
	w.nodeInfo.TasksRunning++
	w.updateLoad()

	defer func() {
		w.activeTasks.Delete(task.ID)
		w.nodeInfo.TasksRunning--
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

func (w *Worker) sendProgressUpdates(ctx context.Context, task *SwarmTask, done chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progress := &TaskProgress{
				TaskID:   task.ID,
				NodeID:   w.nodeInfo.ID,
				Progress: 0.5, // TODO: track actual progress
				Message:  "processing",
			}
			if err := w.bridge.PublishTaskProgress(progress); err != nil {
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
	if w.cfg.MaxConcurrent > 0 {
		w.nodeInfo.Load = float64(w.nodeInfo.TasksRunning) / float64(w.cfg.MaxConcurrent)
	}
	if w.nodeInfo.TasksRunning >= w.cfg.MaxConcurrent {
		w.nodeInfo.Status = StatusBusy
	} else {
		w.nodeInfo.Status = StatusOnline
	}
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

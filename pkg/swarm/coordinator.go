// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Coordinator orchestrates task distribution across the swarm
type Coordinator struct {
	bridge       *NATSBridge
	temporal     *TemporalClient
	discovery    *Discovery
	agentLoop    *agent.AgentLoop
	provider     providers.LLMProvider
	cfg          *config.SwarmConfig
	localBus     *bus.MessageBus
	pendingTasks map[string]*SwarmTask
	taskResults  map[string]chan *TaskResult
	mu           sync.RWMutex
}

// NewCoordinator creates a new coordinator
func NewCoordinator(
	cfg *config.SwarmConfig,
	bridge *NATSBridge,
	temporal *TemporalClient,
	discovery *Discovery,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
	localBus *bus.MessageBus,
) *Coordinator {
	return &Coordinator{
		bridge:       bridge,
		temporal:     temporal,
		discovery:    discovery,
		agentLoop:    agentLoop,
		provider:     provider,
		cfg:          cfg,
		localBus:     localBus,
		pendingTasks: make(map[string]*SwarmTask),
		taskResults:  make(map[string]chan *TaskResult),
	}
}

// Start begins the coordinator
func (c *Coordinator) Start(ctx context.Context) error {
	logger.InfoC("swarm", "Coordinator starting")

	// Listen for inbound messages from local bus and dispatch to swarm
	go c.processInboundMessages(ctx)

	return nil
}

// Stop stops the coordinator
func (c *Coordinator) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close all pending result channels
	for _, ch := range c.taskResults {
		close(ch)
	}
	c.taskResults = make(map[string]chan *TaskResult)
	c.pendingTasks = make(map[string]*SwarmTask)
}

// DispatchTask sends a task to the swarm
func (c *Coordinator) DispatchTask(ctx context.Context, task *SwarmTask) (*TaskResult, error) {
	// Assign task ID if not set
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%s", uuid.New().String()[:8])
	}

	logger.InfoCF("swarm", "Dispatching task", map[string]interface{}{
		"task_id":    task.ID,
		"type":       string(task.Type),
		"capability": task.Capability,
	})

	switch task.Type {
	case TaskTypeWorkflow:
		return c.dispatchWorkflow(ctx, task)
	case TaskTypeDirect:
		return c.dispatchDirect(ctx, task)
	case TaskTypeBroadcast:
		return c.dispatchBroadcast(ctx, task)
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (c *Coordinator) dispatchWorkflow(ctx context.Context, task *SwarmTask) (*TaskResult, error) {
	if !c.temporal.IsConnected() {
		// Fall back to direct dispatch
		logger.WarnC("swarm", "Temporal not connected, falling back to direct dispatch")
		task.Type = TaskTypeDirect
		return c.dispatchDirect(ctx, task)
	}

	// Start Temporal workflow
	workflowID, err := c.temporal.StartWorkflow(ctx, "SwarmWorkflow", task)
	if err != nil {
		return nil, err
	}
	task.WorkflowID = workflowID

	// Wait for result
	result, err := c.temporal.GetWorkflowResult(ctx, workflowID)
	if err != nil {
		return &TaskResult{
			TaskID:      task.ID,
			Status:      string(TaskFailed),
			Error:       err.Error(),
			CompletedAt: time.Now().UnixMilli(),
		}, nil
	}

	return &TaskResult{
		TaskID:      task.ID,
		Status:      string(TaskDone),
		Result:      result,
		CompletedAt: time.Now().UnixMilli(),
	}, nil
}

func (c *Coordinator) dispatchDirect(ctx context.Context, task *SwarmTask) (*TaskResult, error) {
	// Find best worker
	if task.AssignedTo == "" {
		worker := c.discovery.SelectWorker(task.Capability)
		if worker == nil {
			// No remote worker available, execute locally
			logger.InfoC("swarm", "No remote workers, executing locally")
			return c.executeLocally(ctx, task)
		}
		task.AssignedTo = worker.ID
	}

	// Create result channel
	resultCh := make(chan *TaskResult, 1)
	c.mu.Lock()
	c.taskResults[task.ID] = resultCh
	c.pendingTasks[task.ID] = task
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.taskResults, task.ID)
		delete(c.pendingTasks, task.ID)
		c.mu.Unlock()
	}()

	// Subscribe to result
	sub, err := c.bridge.SubscribeTaskResult(task.ID, func(result *TaskResult) {
		select {
		case resultCh <- result:
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()

	// Publish task
	if err := c.bridge.PublishTask(task); err != nil {
		return nil, err
	}

	// Wait for result with timeout
	timeout := time.Duration(task.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	select {
	case result := <-resultCh:
		return result, nil
	case <-time.After(timeout):
		return &TaskResult{
			TaskID:      task.ID,
			Status:      string(TaskFailed),
			Error:       "task timeout",
			CompletedAt: time.Now().UnixMilli(),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Coordinator) dispatchBroadcast(ctx context.Context, task *SwarmTask) (*TaskResult, error) {
	// Same as direct but without specific assignment - NATS queue group handles distribution
	task.AssignedTo = "" // Clear any assignment
	return c.dispatchDirect(ctx, task)
}

func (c *Coordinator) executeLocally(ctx context.Context, task *SwarmTask) (*TaskResult, error) {
	result, err := c.agentLoop.ProcessDirect(ctx, task.Prompt, "swarm:"+task.ID)

	taskResult := &TaskResult{
		TaskID:      task.ID,
		CompletedAt: time.Now().UnixMilli(),
	}

	if err != nil {
		taskResult.Status = string(TaskFailed)
		taskResult.Error = err.Error()
	} else {
		taskResult.Status = string(TaskDone)
		taskResult.Result = result
	}

	return taskResult, nil
}

func (c *Coordinator) processInboundMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, ok := c.localBus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			// Analyze message complexity to decide routing
			task := c.analyzeAndCreateTask(ctx, msg)
			if task == nil {
				// Simple task - process locally
				continue
			}

			// Complex task - dispatch to swarm
			go func() {
				result, err := c.DispatchTask(ctx, task)
				if err != nil {
					logger.ErrorCF("swarm", "Task dispatch failed", map[string]interface{}{
						"error": err.Error(),
					})
					return
				}

				// Send result back to original channel
				c.localBus.PublishOutbound(bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: result.Result,
				})
			}()
		}
	}
}

// analyzeAndCreateTask uses heuristics to decide if a task should be distributed
func (c *Coordinator) analyzeAndCreateTask(ctx context.Context, msg bus.InboundMessage) *SwarmTask {
	// For now, return nil to process locally
	// In production, use LLM analysis to determine task complexity and routing
	// Tasks mentioning "parallel", "analyze", "compare", "comprehensive"
	// could be considered complex and suitable for distribution
	return nil
}

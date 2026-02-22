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
	// Find best worker with priority consideration
	if task.AssignedTo == "" {
		worker := c.discovery.SelectWorkerWithPriority(task.Capability, task.Priority)
		if worker == nil {
			// No remote worker available, execute locally
			logger.InfoC("swarm", "No remote workers, executing locally")
			return c.executeLocally(ctx, task)
		}
		task.AssignedTo = worker.ID

		logger.DebugCF("swarm", "Selected worker for task", map[string]interface{}{
			"task_id":   task.ID,
			"worker_id": worker.ID,
			"priority":  task.Priority,
			"load":      worker.Load,
		})
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
				// Simple task - process locally by agent
				go c.processLocally(ctx, msg)
				continue
			}

			// Complex task - dispatch to swarm
			go c.dispatchWorkflowTask(ctx, task, msg)
		}
	}
}

// processLocally handles simple tasks by forwarding them to the local agent
func (c *Coordinator) processLocally(ctx context.Context, msg bus.InboundMessage) {
	response, err := c.agentLoop.ProcessInboundMessage(ctx, msg)
	if err != nil {
		response = fmt.Sprintf("Error processing message: %v", err)
	}

	if response != "" {
		c.localBus.PublishOutbound(bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: response,
		})
	}
}

// dispatchWorkflowTask handles complex tasks by dispatching them to the swarm
func (c *Coordinator) dispatchWorkflowTask(ctx context.Context, task *SwarmTask, msg bus.InboundMessage) {
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
}

// analyzeAndCreateTask uses heuristics to decide if a task should be distributed
func (c *Coordinator) analyzeAndCreateTask(ctx context.Context, msg bus.InboundMessage) *SwarmTask {
	content := msg.Content

	// Check for keywords that indicate workflow/decomposition is needed
	workflowKeywords := []string{
		"PARALLEL:", "parallel", "concurrent",
		"同时", "分别", "一起",
		"analyze all", "compare", "summarize",
		"汇总", "分别", "列出",
	}

	shouldUseWorkflow := false
	for _, keyword := range workflowKeywords {
		if contains(content, keyword) {
			shouldUseWorkflow = true
			break
		}
	}

	if !shouldUseWorkflow {
		// Simple task - process locally
		return nil
	}

	// Create workflow task for decomposition
	task := &SwarmTask{
		ID:         fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Prompt:     content,
		Type:       TaskTypeWorkflow,
		Capability: "general",
		Priority:   5,
		Status:     TaskPending,
		CreatedAt:  time.Now().UnixMilli(),
		Timeout:    300000, // 5 minutes
	}

	logger.InfoCF("swarm", "Created workflow task from message", map[string]interface{}{
		"task_id": task.ID,
		"prompt":  truncateString(content, 50),
	})

	return task
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		// Simple case-insensitive contains
		toLower(s) == toLower(substr) ||
		findSubstring(toLower(s), toLower(substr))))
}

func toLower(s string) string {
	// Simple ASCII lowercase
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

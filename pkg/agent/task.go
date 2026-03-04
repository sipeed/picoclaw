// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// DEPRECATED: This file contains the legacy TaskManager implementation for Phase 2 concurrent task management.
// The new steering architecture (nanobot-inspired) uses per-session InterruptionChecker instead.
// This code is kept for backward compatibility but will be removed in a future version.
// See: pkg/agent/interruption_checker.go for the new implementation.

package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // Task is queued but not started
	TaskStatusRunning   TaskStatus = "running"   // Task is currently executing
	TaskStatusCompleted TaskStatus = "completed" // Task finished successfully
	TaskStatusCanceled  TaskStatus = "canceled"  // Task was canceled by user/system
	TaskStatusFailed    TaskStatus = "failed"    // Task failed with error
)

// Task represents a single message processing task
type Task struct {
	ID        string             // Unique task identifier
	Message   bus.InboundMessage // The message being processed
	Status    TaskStatus         // Current status
	Priority  int                // Task priority (from interrupt handler)
	CreatedAt time.Time          // When task was created
	StartedAt time.Time          // When task started executing
	EndedAt   time.Time          // When task completed/canceled/failed
	Error     error              // Error if task failed
	Metadata  map[string]any     // Task metadata for custom attributes

	// Context management
	ctx    context.Context    // Task execution context
	cancel context.CancelFunc // Function to cancel this task

	// Result channel
	done chan struct{} // Closed when task completes
}

// NewTask creates a new task from an inbound message
func NewTask(msg bus.InboundMessage, priority int) *Task {
	return &Task{
		ID:        fmt.Sprintf("%s:%s:%d", msg.Channel, msg.ChatID, time.Now().UnixNano()),
		Message:   msg,
		Status:    TaskStatusPending,
		Priority:  priority,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]any),
		done:      make(chan struct{}),
	}
}

// Start marks the task as running and sets up cancellation context
func (t *Task) Start(parentCtx context.Context) {
	t.ctx, t.cancel = context.WithCancel(parentCtx)
	t.Status = TaskStatusRunning
	t.StartedAt = time.Now()
}

// Cancel cancels the task execution
func (t *Task) Cancel() {
	// Check if already in a terminal state
	if t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed || t.Status == TaskStatusCanceled {
		return
	}

	if t.cancel != nil {
		t.cancel()
	}
	t.Status = TaskStatusCanceled
	t.EndedAt = time.Now()
	close(t.done)
}

// Complete marks the task as completed
func (t *Task) Complete() {
	// Check if already in a terminal state
	if t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed || t.Status == TaskStatusCanceled {
		return
	}

	t.Status = TaskStatusCompleted
	t.EndedAt = time.Now()
	close(t.done)
}

// Fail marks the task as failed with an error
func (t *Task) Fail(err error) {
	// Check if already in a terminal state
	if t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed || t.Status == TaskStatusCanceled {
		return
	}

	t.Status = TaskStatusFailed
	t.Error = err
	t.EndedAt = time.Now()
	close(t.done)
}

// Wait blocks until the task completes or is canceled
func (t *Task) Wait() {
	<-t.done
}

// Context returns the task's execution context
func (t *Task) Context() context.Context {
	return t.ctx
}

// TaskManager manages concurrent task execution and cancellation
type TaskManager struct {
	mu            sync.RWMutex
	tasks         map[string]*Task // All tasks by ID
	runningTasks  map[string]*Task // Currently running tasks
	maxConcurrent int              // Maximum concurrent tasks (0 = unlimited)
}

// NewTaskManager creates a new task manager
func NewTaskManager(maxConcurrent int) *TaskManager {
	return &TaskManager{
		tasks:         make(map[string]*Task),
		runningTasks:  make(map[string]*Task),
		maxConcurrent: maxConcurrent,
	}
}

// AddTask adds a new task to the manager
func (tm *TaskManager) AddTask(task *Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	tm.tasks[task.ID] = task
	logger.DebugCF("task", "Task added", map[string]any{
		"task_id":  task.ID,
		"priority": task.Priority,
		"channel":  task.Message.Channel,
		"chat_id":  task.Message.ChatID,
	})
	return nil
}

// StartTask marks a task as running
func (tm *TaskManager) StartTask(taskID string, parentCtx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Check concurrency limit
	if tm.maxConcurrent > 0 && len(tm.runningTasks) >= tm.maxConcurrent {
		return fmt.Errorf("max concurrent tasks (%d) reached", tm.maxConcurrent)
	}

	task.Start(parentCtx)
	tm.runningTasks[taskID] = task

	logger.InfoCF("task", "Task started", map[string]any{
		"task_id":        taskID,
		"running_tasks":  len(tm.runningTasks),
		"max_concurrent": tm.maxConcurrent,
	})
	return nil
}

// CompleteTask marks a task as completed and removes it from running tasks
func (tm *TaskManager) CompleteTask(taskID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.Complete()
		delete(tm.runningTasks, taskID)

		logger.InfoCF("task", "Task completed", map[string]any{
			"task_id":       taskID,
			"duration":      time.Since(task.StartedAt).String(),
			"running_tasks": len(tm.runningTasks),
		})
	}
}

// CancelTask cancels a running task
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusRunning {
		return fmt.Errorf("task %s is not running (status: %s)", taskID, task.Status)
	}

	task.Cancel()
	delete(tm.runningTasks, taskID)

	logger.InfoCF("task", "Task canceled", map[string]any{
		"task_id":       taskID,
		"duration":      time.Since(task.StartedAt).String(),
		"running_tasks": len(tm.runningTasks),
	})
	return nil
}

// FailTask marks a task as failed
func (tm *TaskManager) FailTask(taskID string, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.Fail(err)
		delete(tm.runningTasks, taskID)

		logger.ErrorCF("task", "Task failed", map[string]any{
			"task_id":       taskID,
			"error":         err.Error(),
			"duration":      time.Since(task.StartedAt).String(),
			"running_tasks": len(tm.runningTasks),
		})
	}
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(taskID string) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	return task, exists
}

// GetRunningTasks returns all currently running tasks
func (tm *TaskManager) GetRunningTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.runningTasks))
	for _, task := range tm.runningTasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetRunningTasksForSession returns running tasks for a specific session
func (tm *TaskManager) GetRunningTasksForSession(channel, chatID string) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0)
	for _, task := range tm.runningTasks {
		if task.Message.Channel == channel && task.Message.ChatID == chatID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// CancelAllTasksForSession cancels all running tasks for a specific session
func (tm *TaskManager) CancelAllTasksForSession(channel, chatID string) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	canceled := 0
	for taskID, task := range tm.runningTasks {
		if task.Message.Channel == channel && task.Message.ChatID == chatID {
			task.Cancel()
			delete(tm.runningTasks, taskID)
			canceled++
		}
	}

	if canceled > 0 {
		logger.InfoCF("task", "Canceled tasks for session", map[string]any{
			"channel":        channel,
			"chat_id":        chatID,
			"canceled_count": canceled,
			"running_tasks":  len(tm.runningTasks),
		})
	}

	return canceled
}

// Cleanup removes completed/failed/canceled tasks older than the specified duration
func (tm *TaskManager) Cleanup(olderThan time.Duration) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for taskID, task := range tm.tasks {
		if task.Status != TaskStatusRunning && task.EndedAt.Before(cutoff) {
			delete(tm.tasks, taskID)
			removed++
		}
	}

	if removed > 0 {
		logger.DebugCF("task", "Cleaned up old tasks", map[string]any{
			"removed":    removed,
			"total":      len(tm.tasks),
			"older_than": olderThan.String(),
		})
	}

	return removed
}

// Stats returns task manager statistics
func (tm *TaskManager) Stats() map[string]int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := map[string]int{
		"total":     len(tm.tasks),
		"running":   len(tm.runningTasks),
		"pending":   0,
		"completed": 0,
		"canceled":  0,
		"failed":    0,
	}

	for _, task := range tm.tasks {
		switch task.Status {
		case TaskStatusPending:
			stats["pending"]++
		case TaskStatusCompleted:
			stats["completed"]++
		case TaskStatusCanceled:
			stats["canceled"]++
		case TaskStatusFailed:
			stats["failed"]++
		}
	}

	return stats
}

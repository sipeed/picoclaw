// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestNewTask(t *testing.T) {
	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "test message",
	}

	task := NewTask(msg, 5)

	if task.ID == "" {
		t.Error("Expected task ID to be set")
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
	if task.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", task.Priority)
	}
	if task.Message.Channel != "telegram" {
		t.Errorf("Expected channel telegram, got %s", task.Message.Channel)
	}
}

func TestTask_StartAndComplete(t *testing.T) {
	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)

	ctx := context.Background()
	task.Start(ctx)

	if task.Status != TaskStatusRunning {
		t.Errorf("Expected status running, got %s", task.Status)
	}
	if task.ctx == nil {
		t.Error("Expected context to be set")
	}
	if task.cancel == nil {
		t.Error("Expected cancel function to be set")
	}

	task.Complete()

	if task.Status != TaskStatusCompleted {
		t.Errorf("Expected status completed, got %s", task.Status)
	}

	// done channel should be closed
	select {
	case <-task.done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected done channel to be closed")
	}
}

func TestTask_Cancel(t *testing.T) {
	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)

	ctx := context.Background()
	task.Start(ctx)
	task.Cancel()

	if task.Status != TaskStatusCanceled {
		t.Errorf("Expected status canceled, got %s", task.Status)
	}

	// Context should be canceled
	select {
	case <-task.ctx.Done():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be canceled")
	}
}

func TestTask_Fail(t *testing.T) {
	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)

	ctx := context.Background()
	task.Start(ctx)

	testErr := context.DeadlineExceeded
	task.Fail(testErr)

	if task.Status != TaskStatusFailed {
		t.Errorf("Expected status failed, got %s", task.Status)
	}
	if task.Error != testErr {
		t.Errorf("Expected error %v, got %v", testErr, task.Error)
	}
}

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager(5)

	if tm.maxConcurrent != 5 {
		t.Errorf("Expected max concurrent 5, got %d", tm.maxConcurrent)
	}
	if tm.tasks == nil {
		t.Error("Expected tasks map to be initialized")
	}
	if tm.runningTasks == nil {
		t.Error("Expected running tasks map to be initialized")
	}
}

func TestTaskManager_AddTask(t *testing.T) {
	tm := NewTaskManager(0)
	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)

	err := tm.AddTask(task)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Try adding same task again
	err = tm.AddTask(task)
	if err == nil {
		t.Error("Expected error when adding duplicate task")
	}

	stats := tm.Stats()
	if stats["total"] != 1 {
		t.Errorf("Expected 1 total task, got %d", stats["total"])
	}
	if stats["pending"] != 1 {
		t.Errorf("Expected 1 pending task, got %d", stats["pending"])
	}
}

func TestTaskManager_StartTask(t *testing.T) {
	tm := NewTaskManager(2)
	ctx := context.Background()

	msg1 := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test1"}
	task1 := NewTask(msg1, 5)
	tm.AddTask(task1)

	err := tm.StartTask(task1.ID, ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	stats := tm.Stats()
	if stats["running"] != 1 {
		t.Errorf("Expected 1 running task, got %d", stats["running"])
	}

	// Test concurrency limit
	msg2 := bus.InboundMessage{Channel: "test", ChatID: "2", Content: "test2"}
	task2 := NewTask(msg2, 5)
	tm.AddTask(task2)
	tm.StartTask(task2.ID, ctx)

	msg3 := bus.InboundMessage{Channel: "test", ChatID: "3", Content: "test3"}
	task3 := NewTask(msg3, 5)
	tm.AddTask(task3)

	err = tm.StartTask(task3.ID, ctx)
	if err == nil {
		t.Error("Expected error when exceeding concurrency limit")
	}
}

func TestTaskManager_CompleteTask(t *testing.T) {
	tm := NewTaskManager(0)
	ctx := context.Background()

	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)
	tm.AddTask(task)
	tm.StartTask(task.ID, ctx)

	tm.CompleteTask(task.ID)

	stats := tm.Stats()
	if stats["running"] != 0 {
		t.Errorf("Expected 0 running tasks, got %d", stats["running"])
	}
	if stats["completed"] != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats["completed"])
	}
}

func TestTaskManager_CancelTask(t *testing.T) {
	tm := NewTaskManager(0)
	ctx := context.Background()

	msg := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test"}
	task := NewTask(msg, 5)
	tm.AddTask(task)
	tm.StartTask(task.ID, ctx)

	err := tm.CancelTask(task.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	stats := tm.Stats()
	if stats["running"] != 0 {
		t.Errorf("Expected 0 running tasks, got %d", stats["running"])
	}
	if stats["canceled"] != 1 {
		t.Errorf("Expected 1 canceled task, got %d", stats["canceled"])
	}
}

func TestTaskManager_GetRunningTasksForSession(t *testing.T) {
	tm := NewTaskManager(0)
	ctx := context.Background()

	msg1 := bus.InboundMessage{Channel: "telegram", ChatID: "123", Content: "test1"}
	task1 := NewTask(msg1, 5)
	tm.AddTask(task1)
	tm.StartTask(task1.ID, ctx)

	msg2 := bus.InboundMessage{Channel: "telegram", ChatID: "123", Content: "test2"}
	task2 := NewTask(msg2, 5)
	tm.AddTask(task2)
	tm.StartTask(task2.ID, ctx)

	msg3 := bus.InboundMessage{Channel: "telegram", ChatID: "456", Content: "test3"}
	task3 := NewTask(msg3, 5)
	tm.AddTask(task3)
	tm.StartTask(task3.ID, ctx)

	tasks := tm.GetRunningTasksForSession("telegram", "123")
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks for session, got %d", len(tasks))
	}

	tasks = tm.GetRunningTasksForSession("telegram", "456")
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task for session, got %d", len(tasks))
	}
}

func TestTaskManager_CancelAllTasksForSession(t *testing.T) {
	tm := NewTaskManager(0)
	ctx := context.Background()

	msg1 := bus.InboundMessage{Channel: "telegram", ChatID: "123", Content: "test1"}
	task1 := NewTask(msg1, 5)
	tm.AddTask(task1)
	tm.StartTask(task1.ID, ctx)

	msg2 := bus.InboundMessage{Channel: "telegram", ChatID: "123", Content: "test2"}
	task2 := NewTask(msg2, 5)
	tm.AddTask(task2)
	tm.StartTask(task2.ID, ctx)

	msg3 := bus.InboundMessage{Channel: "telegram", ChatID: "456", Content: "test3"}
	task3 := NewTask(msg3, 5)
	tm.AddTask(task3)
	tm.StartTask(task3.ID, ctx)

	canceled := tm.CancelAllTasksForSession("telegram", "123")
	if canceled != 2 {
		t.Errorf("Expected 2 tasks canceled, got %d", canceled)
	}

	stats := tm.Stats()
	if stats["running"] != 1 {
		t.Errorf("Expected 1 running task, got %d", stats["running"])
	}
	if stats["canceled"] != 2 {
		t.Errorf("Expected 2 canceled tasks, got %d", stats["canceled"])
	}
}

func TestTaskManager_Cleanup(t *testing.T) {
	tm := NewTaskManager(0)
	ctx := context.Background()

	// Create and complete an old task
	msg1 := bus.InboundMessage{Channel: "test", ChatID: "1", Content: "test1"}
	task1 := NewTask(msg1, 5)
	tm.AddTask(task1)
	tm.StartTask(task1.ID, ctx)
	tm.CompleteTask(task1.ID)

	// Manually set EndedAt to simulate old task
	task1.EndedAt = time.Now().Add(-2 * time.Hour)

	// Create a recent completed task
	msg2 := bus.InboundMessage{Channel: "test", ChatID: "2", Content: "test2"}
	task2 := NewTask(msg2, 5)
	tm.AddTask(task2)
	tm.StartTask(task2.ID, ctx)
	tm.CompleteTask(task2.ID)

	// Cleanup tasks older than 1 hour
	removed := tm.Cleanup(1 * time.Hour)
	if removed != 1 {
		t.Errorf("Expected 1 task removed, got %d", removed)
	}

	stats := tm.Stats()
	if stats["total"] != 1 {
		t.Errorf("Expected 1 remaining task, got %d", stats["total"])
	}
}

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskLifecycleStore(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		// Verify stream was created
		stream := GetStreamInfo(t, tn, TaskStreamName)
		assert.NotNil(t, stream)
		assert.Equal(t, TaskStreamName, stream.Config.Name)
	})
}

func TestTaskLifecycleStore_SaveAndGetTaskHistory(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		// Create a test task
		task := &SwarmTask{
			ID:         "test-task-1",
			Type:       TaskTypeDirect,
			Prompt:     "Test prompt",
			Capability: "test",
			Status:     TaskPending,
		}

		// Save task status
		err = store.SaveTaskStatus(task, TaskEventCreated, "Task created")
		require.NoError(t, err)

		// Get task history
		history, err := store.GetTaskHistory(ctx, task.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, history)

		// Verify the event
		event := history[0]
		assert.Equal(t, task.ID, event.TaskID)
		assert.Equal(t, TaskEventCreated, event.EventType)
		assert.Equal(t, "Task created", event.Message)
	})
}

func TestTaskLifecycleStore_TaskTransitions(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		// Purge the stream to ensure clean state
		PurgeTestStream(t, tn, TaskStreamName)

		task := CreateTestTask("task-transitions", "direct", "Test task", "test")

		// Simulate task lifecycle
		transitions := []struct {
			event    TaskEventType
			status   SwarmTaskStatus
			message  string
		}{
			{TaskEventCreated, TaskPending, "Task created"},
			{TaskEventAssigned, TaskAssigned, "Assigned to node-1"},
			{TaskEventStarted, TaskRunning, "Task started"},
			{TaskEventProgress, TaskRunning, "50% complete"},
			{TaskEventCompleted, TaskDone, "Task completed"},
		}

		for _, tt := range transitions {
			task.Status = tt.status
			err = store.SaveTaskStatus(task, tt.event, tt.message)
			require.NoError(t, err)
		}

		// Get full history
		history, err := store.GetTaskHistory(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, history, 5)

		// Verify order
		assert.Equal(t, TaskEventCreated, history[0].EventType)
		assert.Equal(t, TaskEventCompleted, history[4].EventType)
	})
}

func TestTaskLifecycleStore_GetActiveTasks(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		// Create tasks with different statuses
		tasks := []*SwarmTask{
			CreateTestTask("task-active-1", "direct", "Active task 1", "test"),
			CreateTestTask("task-active-2", "direct", "Active task 2", "test"),
			CreateTestTask("task-completed", "direct", "Completed task", "test"),
		}

		// Save statuses
		tasks[0].Status = TaskRunning
		err = store.SaveTaskStatus(tasks[0], TaskEventStarted, "Started")
		require.NoError(t, err)

		tasks[1].Status = TaskAssigned
		err = store.SaveTaskStatus(tasks[1], TaskEventAssigned, "Assigned")
		require.NoError(t, err)

		tasks[2].Status = TaskDone
		err = store.SaveTaskStatus(tasks[2], TaskEventCompleted, "Completed")
		require.NoError(t, err)

		// Get active tasks
		active, err := store.GetActiveTasks(ctx)
		require.NoError(t, err)

		// Should have 2 active tasks
		assert.GreaterOrEqual(t, len(active), 2)
	})
}

func TestTaskLifecycleStore_GetTasksByNode(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		// Create tasks for different nodes
		node1Tasks := []string{"task-node1-1", "task-node1-2"}

		for _, taskID := range node1Tasks {
			task := CreateTestTask(taskID, "direct", "Task for node-1", "test")
			task.Status = TaskRunning
			task.AssignedTo = "node-1"
			err = store.SaveTaskStatus(task, TaskEventStarted, "Started on node-1")
			require.NoError(t, err)
		}

		// Create task for node-2
		task := CreateTestTask("task-node2-1", "direct", "Task for node-2", "test")
		task.Status = TaskRunning
		task.AssignedTo = "node-2"
		err = store.SaveTaskStatus(task, TaskEventStarted, "Started on node-2")
		require.NoError(t, err)

		// Get tasks for node-1
		tasks, err := store.GetTasksByNode(ctx, "node-1")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 2)
	})
}

func TestTaskLifecycleStore_DeleteTaskHistory(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		task := CreateTestTask("task-delete-history", "direct", "Task to delete", "test")

		// Save some history
		err = store.SaveTaskStatus(task, TaskEventCreated, "Created")
		require.NoError(t, err)

		task.Status = TaskRunning
		err = store.SaveTaskStatus(task, TaskEventStarted, "Started")
		require.NoError(t, err)

		// Verify history exists
		history, err := store.GetTaskHistory(ctx, task.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, history)

		// Delete history
		err = store.DeleteTaskHistory(ctx, task.ID)
		require.NoError(t, err)

		// History should be cleared or stream should be empty for this task
		history, err = store.GetTaskHistory(ctx, task.ID)
		require.NoError(t, err)
		// After delete, history should be empty or not exist
		assert.Empty(t, history)
	})
}

func TestTaskLifecycleStore_GetLatestTaskState(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store := NewTaskLifecycleStore(tn.JS())

		err := store.Initialize(ctx)
		require.NoError(t, err)

		task := CreateTestTask("task-latest", "direct", "Get latest state", "test")

		// Save multiple status updates
		transitions := []struct {
			event TaskEventType
			status SwarmTaskStatus
		}{
			{TaskEventCreated, TaskPending},
			{TaskEventAssigned, TaskAssigned},
			{TaskEventStarted, TaskRunning},
			{TaskEventCompleted, TaskDone},
		}
		for _, tt := range transitions {
			task.Status = tt.status
			err = store.SaveTaskStatus(task, tt.event, fmt.Sprintf("State %s", tt.status))
			require.NoError(t, err)
		}

		// Get latest state
		latest, err := store.GetLatestTaskState(ctx, task.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, TaskDone, latest.Status)
	})
}

func TestTaskEventTypes(t *testing.T) {
	events := []TaskEventType{
		TaskEventCreated,
		TaskEventAssigned,
		TaskEventStarted,
		TaskEventProgress,
		TaskEventCompleted,
		TaskEventFailed,
		TaskEventRetry,
		TaskEventCheckpoint,
	}

	for _, event := range events {
		assert.NotEmpty(t, string(event), "Event type should not be empty")
	}
}

func TestTaskEvent(t *testing.T) {
	event := &TaskEvent{
		EventID:   "test-event-1",
		TaskID:    "test-task-1",
		EventType: TaskEventCreated,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    "node-1",
		Status:    TaskPending,
		Message:   "Test message",
		Progress:  0.0,
	}

	assert.Equal(t, "test-event-1", event.EventID)
	assert.Equal(t, "test-task-1", event.TaskID)
	assert.Equal(t, TaskEventCreated, event.EventType)
	assert.Equal(t, "node-1", event.NodeID)
	assert.Equal(t, TaskPending, event.Status)
	assert.Equal(t, "Test message", event.Message)
	assert.Equal(t, 0.0, event.Progress)
}

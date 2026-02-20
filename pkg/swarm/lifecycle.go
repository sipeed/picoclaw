// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// TaskStreamName is the JetStream stream for task events
	TaskStreamName = "PICOCLAW_TASKS"
	// TaskStatusSubject is the subject pattern for task status updates
	// Note: NATS doesn't allow '.' in durable stream names, so we use underscore
	TaskStatusSubject = "picoclaw_tasks_status"
)

// TaskLifecycleStore manages task state persistence using JetStream
type TaskLifecycleStore struct {
	js   nats.JetStreamContext
	mu   sync.RWMutex
	cfg  *lifecycleConfig
}

// lifecycleConfig holds configuration for the lifecycle store
type lifecycleConfig struct {
	streamMaxAge   time.Duration
	streamMaxBytes int64
}

// NewTaskLifecycleStore creates a new task lifecycle store
func NewTaskLifecycleStore(js nats.JetStreamContext) *TaskLifecycleStore {
	return &TaskLifecycleStore{
		js: js,
		cfg: &lifecycleConfig{
			streamMaxAge:   24 * time.Hour * 7, // Keep task history for 7 days
			streamMaxBytes: 1024 * 1024 * 100, // 100MB per stream
		},
	}
}

// Initialize creates the JetStream stream if it doesn't exist
func (s *TaskLifecycleStore) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create stream for task events
	stream, err := s.js.StreamInfo(TaskStreamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = s.js.AddStream(&nats.StreamConfig{
			Name:     TaskStreamName,
			Subjects: []string{TaskStatusSubject + ".>"},
			MaxAge:   s.cfg.streamMaxAge,
			MaxBytes: s.cfg.streamMaxBytes,
			Storage:  nats.FileStorage,
			Discard:  nats.DiscardOld,
			Replicas: 1,
		})
		if err != nil {
			return fmt.Errorf("failed to create task stream: %w", err)
		}
		logger.InfoC("swarm", fmt.Sprintf("Created task stream: %s", TaskStreamName))
	} else {
		logger.DebugC("swarm", fmt.Sprintf("Task stream exists: %s", stream.Config.Name))
	}

	return nil
}

// SaveTaskStatus persists a task status event to JetStream
func (s *TaskLifecycleStore) SaveTaskStatus(task *SwarmTask, eventType TaskEventType, message string) error {
	return s.SaveTaskStatusWithMetadata(task, eventType, message, nil)
}

// SaveTaskStatusWithMetadata persists a task status event with additional metadata
func (s *TaskLifecycleStore) SaveTaskStatusWithMetadata(task *SwarmTask, eventType TaskEventType, message string, metadata map[string]interface{}) error {
	event := &TaskEvent{
		EventID:   fmt.Sprintf("evt-%d-%s", time.Now().UnixNano(), task.ID),
		TaskID:    task.ID,
		EventType: eventType,
		Timestamp: time.Now().UnixMilli(),
		Status:    task.Status,
		Message:   message,
		Metadata:  metadata,
	}

	if task.AssignedTo != "" {
		event.NodeID = task.AssignedTo
	}

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal task event: %w", err)
	}

	// Publish to JetStream
	subject := fmt.Sprintf("%s.%s", TaskStatusSubject, task.ID)
	_, err = s.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish task event: %w", err)
	}

	logger.DebugCF("swarm", "Saved task status event", map[string]interface{}{
		"task_id":    task.ID,
		"event_type": string(eventType),
		"status":     string(task.Status),
	})

	return nil
}

// GetTaskHistory retrieves the complete event history for a task
func (s *TaskLifecycleStore) GetTaskHistory(ctx context.Context, taskID string) ([]TaskEvent, error) {
	subject := fmt.Sprintf("%s.%s", TaskStatusSubject, taskID)

	// Create ephemeral subscription without durable consumer
	queueName := fmt.Sprintf("task-history-%s", taskID)
	sub, err := s.js.PullSubscribe(subject, queueName, nats.AckExplicit())
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	// Fetch messages with timeout
	events := []TaskEvent{}
	fetchDeadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(fetchDeadline) {
		msgs, err := sub.Fetch(100)
		if err != nil {
			if err == nats.ErrTimeout {
				break
			}
			continue
		}

		for _, msg := range msgs {
			var event TaskEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				msg.Ack()
				continue
			}
			events = append(events, event)
			msg.Ack()
		}

		if len(msgs) < 100 {
			// No more messages
			break
		}
	}

	return events, nil
}

// GetActiveTasks retrieves all currently active (running/pending/assigned) tasks
// from the recent event history
func (s *TaskLifecycleStore) GetActiveTasks(ctx context.Context) ([]*SwarmTask, error) {
	// Get stream info to check if stream exists
	_, err := s.js.StreamInfo(TaskStreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	// Use stream's state to understand active tasks
	// Since we're using subject-based filtering, we need to scan recent messages
	activeTasks := make(map[string]*SwarmTask)

	// Create a durable consumer for scanning all task events
	consumerName := "active-tasks-scan"
	_, err = s.js.ConsumerInfo(TaskStreamName, consumerName)
	if err != nil {
		// Create consumer for scanning all task status events
		// FilterSubject must match the stream's subject pattern
		_, err = s.js.AddConsumer(TaskStreamName, &nats.ConsumerConfig{
			Durable:       consumerName,
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			FilterSubject: TaskStatusSubject + ".>",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create scan consumer: %w", err)
		}
	}

	// When binding to an existing consumer, subject must match the filter subject
	sub, err := s.js.PullSubscribe(TaskStatusSubject+".>", "", nats.AckExplicit(), nats.Bind(TaskStreamName, consumerName))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe for active tasks: %w", err)
	}
	defer sub.Unsubscribe()

	// Fetch recent messages
	msgs, err := sub.Fetch(1000, nats.MaxWait(2*time.Second))
	if err != nil && err != nats.ErrTimeout {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	for _, msg := range msgs {
		var event TaskEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			msg.Ack()
			continue
		}

		// Check if task is still active
		if event.Status == TaskPending || event.Status == TaskAssigned || event.Status == TaskRunning {
			// Create a minimal SwarmTask from the event
			task := &SwarmTask{
				ID:        event.TaskID,
				Status:    event.Status,
				AssignedTo: event.NodeID,
			}
			activeTasks[event.TaskID] = task
		}
		msg.Ack()
	}

	// Convert map to slice
	result := make([]*SwarmTask, 0, len(activeTasks))
	for _, task := range activeTasks {
		result = append(result, task)
	}

	return result, nil
}

// GetTasksByNode retrieves all tasks assigned to a specific node
func (s *TaskLifecycleStore) GetTasksByNode(ctx context.Context, nodeID string) ([]*SwarmTask, error) {
	// Get stream info to check if stream exists
	_, err := s.js.StreamInfo(TaskStreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	tasks := make(map[string]*SwarmTask)

	// Create a consumer for scanning all tasks
	consumerName := "node-tasks-scan"
	_, err = s.js.ConsumerInfo(TaskStreamName, consumerName)
	if err != nil {
		// Create consumer for scanning
		_, err = s.js.AddConsumer(TaskStreamName, &nats.ConsumerConfig{
			Durable:       consumerName,
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			FilterSubject: TaskStatusSubject + ".>",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create scan consumer: %w", err)
		}
	}

	// When binding to an existing consumer, subject must match the filter subject
	sub, err := s.js.PullSubscribe(TaskStatusSubject+".>", "", nats.AckExplicit(), nats.Bind(TaskStreamName, consumerName))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe for node tasks: %w", err)
	}
	defer sub.Unsubscribe()

	// Fetch recent messages
	msgs, err := sub.Fetch(1000, nats.MaxWait(2*time.Second))
	if err != nil && err != nats.ErrTimeout {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	for _, msg := range msgs {
		var event TaskEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			msg.Ack()
			continue
		}

		// Only process events for this node
		if event.NodeID == nodeID {
			if task, exists := tasks[event.TaskID]; exists {
				// Update existing task with latest status
				task.Status = event.Status
				task.AssignedTo = event.NodeID
			} else {
				// Create new task entry
				task := &SwarmTask{
					ID:        event.TaskID,
					Status:    event.Status,
					AssignedTo: event.NodeID,
				}
				tasks[event.TaskID] = task
			}
		}
		msg.Ack()
	}

	result := make([]*SwarmTask, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, task)
	}

	return result, nil
}

// DeleteTaskHistory removes event history for a specific task
func (s *TaskLifecycleStore) DeleteTaskHistory(ctx context.Context, taskID string) error {
	subject := fmt.Sprintf("%s.%s", TaskStatusSubject, taskID)

	// Create an ephemeral pull consumer to get messages for deletion
	sub, err := s.js.PullSubscribe(subject, "", nats.AckExplicit())
	if err != nil {
		return fmt.Errorf("failed to create delete consumer: %w", err)
	}
	defer sub.Unsubscribe()

	// Fetch and delete messages for this task
	deletedCount := 0
	for {
		msgs, err := sub.Fetch(100, nats.MaxWait(1*time.Second))
		if err == nats.ErrTimeout {
			break
		}
		if err != nil || len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			// Get message metadata to find sequence number
			meta, err := msg.Metadata()
			if err != nil {
				msg.Ack()
				continue
			}

			// Delete the message using the JetStream API
			// The API uses stream name and sequence number
			err = s.js.DeleteMsg(TaskStreamName, meta.Sequence.Stream)
			if err != nil {
				// Message might have been deleted already
				msg.Ack()
				continue
			}
			deletedCount++
		}
	}

	logger.DebugCF("swarm", "Deleted task history", map[string]interface{}{
		"task_id":      taskID,
		"deleted_msgs": deletedCount,
	})

	return nil
}

// GetLatestTaskState retrieves the latest state of a task from its history
func (s *TaskLifecycleStore) GetLatestTaskState(ctx context.Context, taskID string) (*SwarmTask, error) {
	events, err := s.GetTaskHistory(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no history found for task %s", taskID)
	}

	// Get the most recent event
	latestEvent := events[len(events)-1]

	task := &SwarmTask{
		ID:        latestEvent.TaskID,
		Status:    latestEvent.Status,
		AssignedTo: latestEvent.NodeID,
	}

	return task, nil
}

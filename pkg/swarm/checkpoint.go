// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// CheckpointBucketName is the KV bucket for task checkpoints
	CheckpointBucketName = "PICOCLAW_CHECKPOINTS"
)

// CheckpointStore manages task checkpoints using JetStream KV
type CheckpointStore struct {
	bucket nats.KeyValue
}

// NewCheckpointStore creates a new checkpoint store
func NewCheckpointStore(js nats.JetStreamContext) (*CheckpointStore, error) {
	// Create or get KV bucket for checkpoints
	bucket, err := js.KeyValue(CheckpointBucketName)
	if err != nil {
		// Bucket doesn't exist, create it
		bucket, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:      CheckpointBucketName,
			Description: "Task checkpoints for PicoClaw swarm failover",
			MaxBytes:    1024 * 1024 * 100, // 100MB
			TTL:         24 * time.Hour * 7, // 7 day retention
			Storage:     nats.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create checkpoint bucket: %w", err)
		}
		logger.InfoC("swarm", fmt.Sprintf("Created checkpoint bucket: %s", CheckpointBucketName))
	}

	return &CheckpointStore{bucket: bucket}, nil
}

// SaveCheckpoint persists a checkpoint to the KV store
func (s *CheckpointStore) SaveCheckpoint(ctx context.Context, cp *TaskCheckpoint) error {
	if cp.CheckpointID == "" {
		cp.CheckpointID = fmt.Sprintf("cp-%d", time.Now().UnixNano())
	}
	cp.Timestamp = time.Now().UnixMilli()

	// Marshal checkpoint to JSON
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Key format: {task_id}_{checkpoint_id}
	key := fmt.Sprintf("%s_%s", cp.TaskID, cp.CheckpointID)

	// Use Update for atomic operation with version checking
	_, err = s.bucket.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	logger.DebugCF("swarm", "Saved checkpoint", map[string]interface{}{
		"task_id":      cp.TaskID,
		"checkpoint_id": cp.CheckpointID,
		"type":         string(cp.Type),
		"progress":     cp.Progress,
	})

	return nil
}

// SaveCheckpointAtomic saves a checkpoint with atomic compare-and-swap semantics
// This prevents race conditions during concurrent checkpoint saves
func (s *CheckpointStore) SaveCheckpointAtomic(ctx context.Context, cp *TaskCheckpoint, lastRevision uint64) error {
	if cp.CheckpointID == "" {
		cp.CheckpointID = fmt.Sprintf("cp-%d", time.Now().UnixNano())
	}
	cp.Timestamp = time.Now().UnixMilli()

	// Marshal checkpoint to JSON
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Key format: {task_id}_{checkpoint_id}
	key := fmt.Sprintf("%s_%s", cp.TaskID, cp.CheckpointID)

	// Atomic update with version check
	_, err = s.bucket.Update(key, data, lastRevision)
	if err != nil {
		return fmt.Errorf("atomic checkpoint save failed: %w", err)
	}

	logger.DebugCF("swarm", "Saved checkpoint atomically", map[string]interface{}{
		"task_id":      cp.TaskID,
		"checkpoint_id": cp.CheckpointID,
		"revision":     lastRevision,
	})

	return nil
}

// LoadCheckpoint retrieves the latest checkpoint for a task
func (s *CheckpointStore) LoadCheckpoint(ctx context.Context, taskID string) (*TaskCheckpoint, error) {
	// List all checkpoints for this task
	checkpoints, err := s.ListCheckpoints(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for task %s", taskID)
	}

	// Return the most recent checkpoint
	return checkpoints[0], nil
}

// LoadCheckpointByID retrieves a specific checkpoint by ID
func (s *CheckpointStore) LoadCheckpointByID(ctx context.Context, taskID, checkpointID string) (*TaskCheckpoint, error) {
	key := fmt.Sprintf("%s_%s", taskID, checkpointID)

	entry, err := s.bucket.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	var cp TaskCheckpoint
	if err := json.Unmarshal(entry.Value(), &cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &cp, nil
}

// ListCheckpoints retrieves all checkpoints for a task, ordered by timestamp (newest first)
func (s *CheckpointStore) ListCheckpoints(ctx context.Context, taskID string) ([]*TaskCheckpoint, error) {
	// List all keys with the task prefix
	watcher, err := s.bucket.WatchAll(nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to watch checkpoints: %w", err)
	}
	defer watcher.Stop()

	checkpoints := make([]*TaskCheckpoint, 0)
	prefix := taskID + "_"

	for {
		select {
		case entry := <-watcher.Updates():
			if entry == nil {
				// No more entries
				goto done
			}

			if len(entry.Key()) <= len(prefix) {
				continue
			}

			// Check if key matches our task prefix
			if entry.Key()[:len(prefix)] == prefix {
				var cp TaskCheckpoint
				if err := json.Unmarshal(entry.Value(), &cp); err != nil {
					logger.WarnCF("swarm", "Failed to unmarshal checkpoint", map[string]interface{}{
						"key":   entry.Key(),
						"error": err.Error(),
					})
					continue
				}
				checkpoints = append(checkpoints, &cp)
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

done:
	// Sort by timestamp descending (newest first)
	sortCheckpoints(checkpoints)

	return checkpoints, nil
}

// DeleteCheckpoint removes a specific checkpoint
func (s *CheckpointStore) DeleteCheckpoint(ctx context.Context, taskID, checkpointID string) error {
	key := fmt.Sprintf("%s_%s", taskID, checkpointID)

	err := s.bucket.Delete(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	logger.DebugCF("swarm", "Deleted checkpoint", map[string]interface{}{
		"task_id":      taskID,
		"checkpoint_id": checkpointID,
	})

	return nil
}

// DeleteAllCheckpoints removes all checkpoints for a task
func (s *CheckpointStore) DeleteAllCheckpoints(ctx context.Context, taskID string) error {
	checkpoints, err := s.ListCheckpoints(ctx, taskID)
	if err != nil {
		return err
	}

	for _, cp := range checkpoints {
		if err := s.DeleteCheckpoint(ctx, taskID, cp.CheckpointID); err != nil {
			logger.WarnCF("swarm", "Failed to delete checkpoint", map[string]interface{}{
				"task_id":      taskID,
				"checkpoint_id": cp.CheckpointID,
				"error":        err.Error(),
			})
		}
	}

	return nil
}

// GetCheckpointRevision returns the current revision of a checkpoint for CAS operations
func (s *CheckpointStore) GetCheckpointRevision(ctx context.Context, taskID, checkpointID string) (uint64, error) {
	key := fmt.Sprintf("%s_%s", taskID, checkpointID)

	entry, err := s.bucket.Get(key)
	if err != nil {
		return 0, fmt.Errorf("failed to get checkpoint revision: %w", err)
	}

	return entry.Revision(), nil
}

// PruneOldCheckpoints removes old checkpoints keeping only the most recent N
func (s *CheckpointStore) PruneOldCheckpoints(ctx context.Context, taskID string, keep int) error {
	checkpoints, err := s.ListCheckpoints(ctx, taskID)
	if err != nil {
		return err
	}

	if len(checkpoints) <= keep {
		return nil
	}

	// Delete oldest checkpoints (starting from index 'keep')
	for i := keep; i < len(checkpoints); i++ {
		cp := checkpoints[i]
		if err := s.DeleteCheckpoint(ctx, taskID, cp.CheckpointID); err != nil {
			logger.WarnCF("swarm", "Failed to prune old checkpoint", map[string]interface{}{
				"task_id":      taskID,
				"checkpoint_id": cp.CheckpointID,
				"error":        err.Error(),
			})
		}
	}

	return nil
}

// sortCheckpoints sorts checkpoints by timestamp descending (newest first)
func sortCheckpoints(checkpoints []*TaskCheckpoint) {
	// Simple bubble sort for small lists
	n := len(checkpoints)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if checkpoints[j].Timestamp < checkpoints[j+1].Timestamp {
				checkpoints[j], checkpoints[j+1] = checkpoints[j+1], checkpoints[j]
			}
		}
	}
}

// CreateProgressCheckpoint creates a progress checkpoint from current execution state
func CreateProgressCheckpoint(taskID, nodeID string, progress float64, partialResult string, state map[string]interface{}) *TaskCheckpoint {
	return &TaskCheckpoint{
		CheckpointID:  fmt.Sprintf("prog-%d", time.Now().UnixNano()),
		TaskID:        taskID,
		Type:          CheckpointTypeProgress,
		NodeID:        nodeID,
		Progress:      progress,
		PartialResult: partialResult,
		State:         state,
		Timestamp:     time.Now().UnixMilli(),
	}
}

// CreateMilestoneCheckpoint creates a milestone checkpoint
func CreateMilestoneCheckpoint(taskID, nodeID string, progress float64, result string, metadata map[string]string) *TaskCheckpoint {
	return &TaskCheckpoint{
		CheckpointID: fmt.Sprintf("milestone-%d", time.Now().UnixNano()),
		TaskID:       taskID,
		Type:         CheckpointTypeMilestone,
		NodeID:       nodeID,
		Progress:     progress,
		PartialResult: result,
		Metadata:     metadata,
		Timestamp:    time.Now().UnixMilli(),
	}
}

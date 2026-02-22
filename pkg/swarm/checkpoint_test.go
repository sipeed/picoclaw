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

func TestNewCheckpointStore(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)
		assert.NotNil(t, store)
	})
}

func TestCheckpointStore_SaveAndLoad(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Create a test checkpoint
		cp := &TaskCheckpoint{
			CheckpointID:  "cp-test-1",
			TaskID:        "task-1",
			Type:          CheckpointTypeProgress,
			Timestamp:     time.Now().UnixMilli(),
			NodeID:        "node-1",
			Progress:      0.5,
			State:         map[string]interface{}{"step": 2},
			PartialResult: "Partial result",
		}

		// Save checkpoint
		err = store.SaveCheckpoint(ctx, cp)
		require.NoError(t, err)

		// Load checkpoint
		loaded, err := store.LoadCheckpoint(ctx, "task-1")
		require.NoError(t, err)
		assert.Equal(t, cp.CheckpointID, loaded.CheckpointID)
		assert.Equal(t, cp.TaskID, loaded.TaskID)
		assert.Equal(t, cp.Type, loaded.Type)
		assert.Equal(t, cp.NodeID, loaded.NodeID)
		assert.Equal(t, cp.Progress, loaded.Progress)
		assert.Equal(t, cp.PartialResult, loaded.PartialResult)
	})
}

func TestCheckpointStore_LoadByID(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		cp := &TaskCheckpoint{
			CheckpointID:  "cp-test-2",
			TaskID:        "task-2",
			Type:          CheckpointTypeMilestone,
			Timestamp:     time.Now().UnixMilli(),
			NodeID:        "node-2",
			Progress:      1.0,
			PartialResult: "Complete",
		}

		err = store.SaveCheckpoint(ctx, cp)
		require.NoError(t, err)

		// Load by checkpoint ID
		loaded, err := store.LoadCheckpointByID(ctx, "task-2", cp.CheckpointID)
		require.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, cp.CheckpointID, loaded.CheckpointID)
		assert.Equal(t, cp.TaskID, loaded.TaskID)
	})
}

func TestCheckpointStore_ListCheckpoints(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Create multiple checkpoints for the same task
		for i := 0; i < 3; i++ {
			cp := &TaskCheckpoint{
				CheckpointID:  fmt.Sprintf("cp-list-%d", i),
				TaskID:        "task-list",
				Type:          CheckpointTypeProgress,
				Timestamp:     time.Now().Add(time.Duration(i) * time.Second).UnixMilli(),
				NodeID:        "node-1",
				Progress:      float64(i) * 0.3,
			}
			err = store.SaveCheckpoint(ctx, cp)
			require.NoError(t, err)
		}

		// List checkpoints
		checkpoints, err := store.ListCheckpoints(ctx, "task-list")
		require.NoError(t, err)
		assert.Len(t, checkpoints, 3)
	})
}

func TestCheckpointStore_PruneOldCheckpoints(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Create multiple checkpoints
		for i := 0; i < 3; i++ {
			cp := &TaskCheckpoint{
				CheckpointID:  fmt.Sprintf("cp-prune-%d", i),
				TaskID:        "task-prune",
				Type:          CheckpointTypeProgress,
				Timestamp:     time.Now().Add(time.Duration(i) * time.Second).UnixMilli(),
				NodeID:        "node-1",
				Progress:      float64(i) * 0.3,
			}
			err = store.SaveCheckpoint(ctx, cp)
			require.NoError(t, err)
		}

		// Prune to keep only 1 most recent checkpoint
		err = store.PruneOldCheckpoints(ctx, "task-prune", 1)
		require.NoError(t, err)

		// Verify only one checkpoint remains
		checkpoints, err := store.ListCheckpoints(ctx, "task-prune")
		require.NoError(t, err)
		assert.Len(t, checkpoints, 1)
	})
}

func TestCheckpointStore_DeleteCheckpoint(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()
		store, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		cp := &TaskCheckpoint{
			CheckpointID:  "cp-delete",
			TaskID:        "task-delete",
			Type:          CheckpointTypeProgress,
			Timestamp:     time.Now().UnixMilli(),
			NodeID:        "node-1",
			Progress:      0.5,
		}

		err = store.SaveCheckpoint(ctx, cp)
		require.NoError(t, err)

		// Verify it exists
		loaded, err := store.LoadCheckpoint(ctx, "task-delete")
		require.NoError(t, err)
		assert.NotNil(t, loaded)

		// Delete checkpoint
		err = store.DeleteCheckpoint(ctx, "task-delete", cp.CheckpointID)
		require.NoError(t, err)

		// Verify it's gone - LoadCheckpoint should return error or empty result
		_, err = store.LoadCheckpoint(ctx, "task-delete")
		// After deleting the only checkpoint, the result should be empty or error
		assert.Error(t, err)
	})
}

func TestCreateProgressCheckpoint(t *testing.T) {
	taskID := "test-task-1"
	nodeID := "node-1"
	progress := 0.5
	partialResult := "Partial work done"
	state := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	cp := CreateProgressCheckpoint(taskID, nodeID, progress, partialResult, state)

	assert.NotEmpty(t, cp.CheckpointID)
	assert.Equal(t, taskID, cp.TaskID)
	assert.Equal(t, CheckpointTypeProgress, cp.Type)
	assert.Equal(t, nodeID, cp.NodeID)
	assert.Equal(t, progress, cp.Progress)
	assert.Equal(t, partialResult, cp.PartialResult)
	assert.Equal(t, state, cp.State)
	assert.NotZero(t, cp.Timestamp)
}

func TestCreateMilestoneCheckpoint(t *testing.T) {
	taskID := "test-task-2"
	nodeID := "node-2"
	progress := 1.0
	result := "Work completed"
	metadata := map[string]string{
		"milestone": "first",
		"category":  "test",
	}

	cp := CreateMilestoneCheckpoint(taskID, nodeID, progress, result, metadata)

	assert.NotEmpty(t, cp.CheckpointID)
	assert.Contains(t, cp.CheckpointID, "milestone-")
	assert.Equal(t, taskID, cp.TaskID)
	assert.Equal(t, CheckpointTypeMilestone, cp.Type)
	assert.Equal(t, nodeID, cp.NodeID)
	assert.Equal(t, progress, cp.Progress)
	assert.Equal(t, result, cp.PartialResult)
	assert.Equal(t, metadata, cp.Metadata)
	assert.NotZero(t, cp.Timestamp)
}

func TestCheckpointTypes(t *testing.T) {
	types := []CheckpointType{
		CheckpointTypeProgress,
		CheckpointTypeMilestone,
		CheckpointTypePreFailover,
		CheckpointTypeUserCheckpointType,
	}

	for _, ct := range types {
		assert.NotEmpty(t, string(ct), "Checkpoint type should not be empty")
	}
}

func TestTaskCheckpoint(t *testing.T) {
	cp := &TaskCheckpoint{
		CheckpointID:  "cp-test-1",
		TaskID:        "task-1",
		Type:          CheckpointTypeProgress,
		Timestamp:     time.Now().UnixMilli(),
		NodeID:        "node-1",
		Progress:      0.75,
		State:         map[string]interface{}{"step": 3},
		PartialResult: "Partial result",
		Context:       map[string]interface{}{"messages": []string{"msg1", "msg2"}},
		Metadata:      map[string]string{"key": "value"},
	}

	assert.Equal(t, "cp-test-1", cp.CheckpointID)
	assert.Equal(t, "task-1", cp.TaskID)
	assert.Equal(t, CheckpointTypeProgress, cp.Type)
	assert.Equal(t, "node-1", cp.NodeID)
	assert.Equal(t, 0.75, cp.Progress)
	assert.Equal(t, "Partial result", cp.PartialResult)
	assert.NotNil(t, cp.State)
	assert.NotNil(t, cp.Context)
	assert.NotNil(t, cp.Metadata)
}

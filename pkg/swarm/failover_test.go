// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCfg provides a default test configuration
var testCfg = &config.Config{
	Swarm: config.SwarmConfig{
		NATS: config.NATSConfig{
			HeartbeatInterval: "10s",
			NodeTimeout:       "60s",
		},
		Temporal: config.TemporalConfig{
			Host:      "localhost:7233",
			Namespace: "default",
			TaskQueue: "picoclaw-test",
		},
		MaxConcurrent: 5,
	},
}

func TestNewFailoverManager(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()

		// Create test config
		swarmCfg := &testCfg.Swarm
		swarmCfg.NATS.URLs = []string{tn.URL()}

		// Create discovery
		bridge := NewNATSBridge(swarmCfg, nil, CreateTestNodeInfo("test-node", "coordinator", []string{}))
		err := bridge.Connect(ctx)
		require.NoError(t, err)

		discovery := NewDiscovery(bridge, CreateTestNodeInfo("test-node", "coordinator", []string{}), swarmCfg)

		// Create lifecycle store
		lifecycle := NewTaskLifecycleStore(tn.JS())
		err = lifecycle.Initialize(ctx)
		require.NoError(t, err)

		// Create checkpoint store
		checkpointStore, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Create failover manager
		fm := NewFailoverManager(discovery, lifecycle, checkpointStore, bridge,
			CreateTestNodeInfo("test-node", "coordinator", []string{}), tn.JS())

		assert.NotNil(t, fm)
	})
}

func TestFailoverManager_ClaimTask(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()

		// Setup
		swarmCfg := &testCfg.Swarm
		swarmCfg.NATS.URLs = []string{tn.URL()}

		bridge := NewNATSBridge(swarmCfg, nil, CreateTestNodeInfo("node-1", "coordinator", []string{}))
		err := bridge.Connect(ctx)
		require.NoError(t, err)

		discovery := NewDiscovery(bridge, CreateTestNodeInfo("node-1", "coordinator", []string{}), swarmCfg)

		lifecycle := NewTaskLifecycleStore(tn.JS())
		err = lifecycle.Initialize(ctx)
		require.NoError(t, err)

		checkpointStore, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Clean up KV buckets from previous tests
		_ = tn.JS().DeleteKeyValue("PICOCLAW_CLAIMS")
		_ = tn.JS().DeleteStream("PICOCLAW_TASKS")

		fm := NewFailoverManager(discovery, lifecycle, checkpointStore, bridge,
			CreateTestNodeInfo("node-1", "coordinator", []string{}), tn.JS())

		err = fm.Start(ctx)
		require.NoError(t, err)
		defer fm.Stop()

		// Create a task
		task := CreateTestTask("task-claim", "direct", "Test claim", "test")

		// Claim the task
		claimed, checkpoint, err := fm.ClaimTask(ctx, task.ID)
		require.NoError(t, err)
		assert.True(t, claimed)
		// checkpoint may be nil if task hasn't saved one yet
		_ = checkpoint // We got a claim, that's what matters

		// Try to claim again - should fail (key already exists)
		// Note: This will return an error since Create fails on existing key
		_, _, err = fm.ClaimTask(ctx, task.ID)
		assert.Error(t, err, "Should fail when trying to claim an already claimed task")
	})
}

func TestFailoverManager_ReleaseClaim(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()

		// Setup
		swarmCfg := &testCfg.Swarm
		swarmCfg.NATS.URLs = []string{tn.URL()}

		bridge := NewNATSBridge(swarmCfg, nil, CreateTestNodeInfo("node-1", "coordinator", []string{}))
		err := bridge.Connect(ctx)
		require.NoError(t, err)

		discovery := NewDiscovery(bridge, CreateTestNodeInfo("node-1", "coordinator", []string{}), swarmCfg)

		lifecycle := NewTaskLifecycleStore(tn.JS())
		err = lifecycle.Initialize(ctx)
		require.NoError(t, err)

		checkpointStore, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Clean up KV buckets from previous tests
		_ = tn.JS().DeleteKeyValue("PICOCLAW_CLAIMS")
		_ = tn.JS().DeleteStream("PICOCLAW_TASKS")

		fm := NewFailoverManager(discovery, lifecycle, checkpointStore, bridge,
			CreateTestNodeInfo("node-1", "coordinator", []string{}), tn.JS())

		err = fm.Start(ctx)
		require.NoError(t, err)
		defer fm.Stop()

		task := CreateTestTask("task-release", "direct", "Test release", "test")

		// Claim the task
		claimed, _, err := fm.ClaimTask(ctx, task.ID)
		require.NoError(t, err)
		assert.True(t, claimed)

		// Release the claim
		err = fm.ReleaseClaim(ctx, task.ID)
		require.NoError(t, err)

		// Now should be claimable again
		claimed2, _, err := fm.ClaimTask(ctx, task.ID)
		require.NoError(t, err)
		assert.True(t, claimed2, "Task should be claimable after release")
	})
}

func TestFailoverManager_RenewClaim(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx := context.Background()

		// Setup
		swarmCfg := &testCfg.Swarm
		swarmCfg.NATS.URLs = []string{tn.URL()}

		bridge := NewNATSBridge(swarmCfg, nil, CreateTestNodeInfo("node-1", "coordinator", []string{}))
		err := bridge.Connect(ctx)
		require.NoError(t, err)

		discovery := NewDiscovery(bridge, CreateTestNodeInfo("node-1", "coordinator", []string{}), swarmCfg)

		lifecycle := NewTaskLifecycleStore(tn.JS())
		err = lifecycle.Initialize(ctx)
		require.NoError(t, err)

		checkpointStore, err := NewCheckpointStore(tn.JS())
		require.NoError(t, err)

		// Clean up KV buckets from previous tests
		_ = tn.JS().DeleteKeyValue("PICOCLAW_CLAIMS")
		_ = tn.JS().DeleteStream("PICOCLAW_TASKS")

		fm := NewFailoverManager(discovery, lifecycle, checkpointStore, bridge,
			CreateTestNodeInfo("node-1", "coordinator", []string{}), tn.JS())

		err = fm.Start(ctx)
		require.NoError(t, err)
		defer fm.Stop()

		task := CreateTestTask("task-renew", "direct", "Test renew", "test")

		// Initial claim
		claimed, _, err := fm.ClaimTask(ctx, task.ID)
		require.NoError(t, err)
		assert.True(t, claimed)

		// Renew claim
		err = fm.RenewClaim(ctx, task.ID)
		require.NoError(t, err)

		// Verify claim is still held by trying to claim again (should fail)
		_, _, err = fm.ClaimTask(ctx, task.ID)
		assert.Error(t, err, "Task should still be claimed")
	})
}

func TestClaimInfo(t *testing.T) {
	info := &ClaimInfo{
		TaskID:    "task-1",
		ClaimedBy: "node-1",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Second),
	}

	assert.Equal(t, "task-1", info.TaskID)
	assert.Equal(t, "node-1", info.ClaimedBy)
	assert.True(t, time.Until(info.ExpiresAt) < 30*time.Second, "Claim should expire within 30 seconds")
}

func TestDefaultTimeouts(t *testing.T) {
	assert.Equal(t, 60*time.Second, DefaultHeartbeatTimeout)
	assert.Equal(t, 2*time.Minute, DefaultProgressStallTimeout)
	assert.Equal(t, 10*time.Second, FailoverCheckInterval)
	assert.Equal(t, 30*time.Second, ClaimLockTTL)
}

func TestClaimInfo_IsExpired(t *testing.T) {
	expiredInfo := &ClaimInfo{
		TaskID:    "task-1",
		ClaimedBy: "node-1",
		ClaimedAt: time.Now().Add(-1 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	assert.True(t, expiredInfo.ExpiresAt.Before(time.Now()), "Old claim should be expired")

	validInfo := &ClaimInfo{
		TaskID:    "task-2",
		ClaimedBy: "node-1",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Second),
	}

	assert.True(t, validInfo.ExpiresAt.After(time.Now()), "Recent claim should not be expired")
}

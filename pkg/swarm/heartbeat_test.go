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

func TestDefaultHeartbeatConfig(t *testing.T) {
	cfg := DefaultHeartbeatConfig()

	assert.Equal(t, HeartbeatInterval, cfg.Interval)
	assert.Equal(t, HeartbeatSuspiciousThreshold, cfg.SuspiciousTimeout)
	assert.Equal(t, HeartbeatOfflineThreshold, cfg.OfflineTimeout)
}

func TestHeartbeatPublisher_StartStop(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		nodeInfo := CreateTestNodeInfo("hb-pub-test", string(RoleWorker), []string{"test"})

		// Use connectTestBridge which properly configures the URL
		bridge := connectTestBridge(t, tn.url, nodeInfo)
		defer bridge.Stop()

		pub := NewHeartbeatPublisher(bridge, nodeInfo, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		require.NoError(t, pub.Start(ctx))
		assert.True(t, pub.IsRunning())

		pub.Stop()
		// Give it a moment to stop
		time.Sleep(10 * time.Millisecond)
	})
}

func TestHeartbeatPublisher_SendHeartbeat(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		nodeInfo := CreateTestNodeInfo("hb-send-test", string(RoleWorker), []string{"test"})

		// Use connectTestBridge which properly configures the URL
		bridge := connectTestBridge(t, tn.url, nodeInfo)
		defer bridge.Stop()

		pub := NewHeartbeatPublisher(bridge, nodeInfo, nil)

		// Subscribe to heartbeats
		received := make(chan *Heartbeat, 1)
		sub, err := bridge.SubscribeAllHeartbeats(func(hb *Heartbeat) {
			select {
			case received <- hb:
			default:
			}
		})
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Start publisher
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		require.NoError(t, pub.Start(ctx))
		defer pub.Stop()

		// Wait for heartbeat
		select {
		case hb := <-received:
			assert.Equal(t, "hb-send-test", hb.NodeID)
			assert.NotZero(t, hb.Timestamp)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Did not receive heartbeat in time")
		case <-ctx.Done():
			t.Fatal("Context canceled while waiting for heartbeat")
		}
	})
}

func TestHeartbeatMonitor_TrackHeartbeats(t *testing.T) {
	// Create a simple discovery for the monitor
	nodeInfo := CreateTestNodeInfo("monitor-test", string(RoleCoordinator), nil)
	swarmCfg := &config.SwarmConfig{
		Enabled:       true,
		MaxConcurrent: 2,
		NATS:          config.NATSConfig{HeartbeatInterval: "50ms", NodeTimeout: "200ms"},
	}

	// Create a mock discovery - we need a real one but can use nil bridge
	bridge := NewNATSBridge(swarmCfg, nil, nodeInfo)
	discovery := NewDiscovery(bridge, nodeInfo, swarmCfg)

	// Add the test node to discovery
	testNode := CreateTestNodeInfo("test-node", string(RoleWorker), []string{"test"})
	discovery.handleNodeJoin(testNode)

	monitor := NewHeartbeatMonitor(discovery, nil)

	hb := &Heartbeat{
		NodeID:    "test-node",
		Timestamp: time.Now().UnixMilli(),
		Status:    StatusOnline,
	}

	// Before any heartbeat
	assert.Zero(t, monitor.GetLastHeartbeat("test-node"))

	// Record heartbeat
	monitor.UpdateHeartbeat(hb)

	// After heartbeat
	lastHB := monitor.GetLastHeartbeat("test-node")
	assert.False(t, lastHB.IsZero())
	assert.True(t, time.Since(lastHB) < time.Second)
}

func TestHeartbeatMonitor_OfflineDetection(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		nodeInfo := CreateTestNodeInfo("coord-main", string(RoleCoordinator), nil)

		// Use connectTestBridge which properly configures the URL
		bridge := connectTestBridge(t, tn.url, nodeInfo)
		defer bridge.Stop()

		// Create a minimal swarm config for discovery
		swarmCfg := &config.SwarmConfig{
			Enabled:           true,
			MaxConcurrent:     2,
			NATS:              config.NATSConfig{HeartbeatInterval: "50ms", NodeTimeout: "200ms"},
		}
		discovery := NewDiscovery(bridge, nodeInfo, swarmCfg)

		// Short timeout for testing
		cfg := &HeartbeatConfig{
			Interval:           10 * time.Millisecond,
			SuspiciousTimeout:  25 * time.Millisecond,
			OfflineTimeout:     50 * time.Millisecond,
		}

		monitor := NewHeartbeatMonitor(discovery, cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		require.NoError(t, monitor.Start(ctx))
		defer monitor.Stop()

		// Add a node to discovery
		testNode := CreateTestNodeInfo("test-offline-node", string(RoleWorker), []string{"test"})
		discovery.handleNodeJoin(testNode)

		// Send initial heartbeat
		hb := &Heartbeat{
			NodeID:    "test-offline-node",
			Timestamp: time.Now().UnixMilli(),
			Status:    StatusOnline,
		}
		monitor.UpdateHeartbeat(hb)

		// Node should be online
		node, ok := discovery.GetNode("test-offline-node")
		require.True(t, ok)
		assert.Equal(t, StatusOnline, node.Status)

		// Wait for suspicious threshold
		time.Sleep(30 * time.Millisecond)

		// Check heartbeats manually (since checker runs every 5s in tests, we'll trigger it)
		monitor.checkHeartbeats()

		node, ok = discovery.GetNode("test-offline-node")
		require.True(t, ok)
		assert.Equal(t, StatusSuspicious, node.Status, "Node should be marked suspicious")

		// Wait for offline threshold
		time.Sleep(30 * time.Millisecond)
		monitor.checkHeartbeats()

		node, ok = discovery.GetNode("test-offline-node")
		require.True(t, ok)
		assert.Equal(t, StatusOffline, node.Status, "Node should be marked offline")
	})
}

func TestHeartbeat_MessageFields(t *testing.T) {
	hb := &Heartbeat{
		NodeID:       "test-node",
		Timestamp:    1234567890,
		Load:         0.75,
		TasksRunning: 3,
		Status:       StatusBusy,
		Capabilities: []string{"code", "write"},
	}

	assert.Equal(t, "test-node", hb.NodeID)
	assert.Equal(t, int64(1234567890), hb.Timestamp)
	assert.Equal(t, 0.75, hb.Load)
	assert.Equal(t, 3, hb.TasksRunning)
	assert.Equal(t, StatusBusy, hb.Status)
	assert.Equal(t, []string{"code", "write"}, hb.Capabilities)
}

func TestHeartbeatPublisher_IsRunning(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		nodeInfo := CreateTestNodeInfo("hb-running-test", string(RoleWorker), []string{"test"})

		// Use connectTestBridge which properly configures the URL
		bridge := connectTestBridge(t, tn.url, nodeInfo)
		defer bridge.Stop()

		pub := NewHeartbeatPublisher(bridge, nodeInfo, nil)

		// Not running initially
		assert.False(t, pub.isRunning())

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		pub.Start(ctx)
		assert.True(t, pub.isRunning())

		pub.Stop()
		assert.False(t, pub.isRunning())
	})
}

// Helper method for testing
func (hp *HeartbeatPublisher) IsRunning() bool {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	return hp.running
}

func (hp *HeartbeatPublisher) isRunning() bool {
	return hp.IsRunning()
}

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

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// Discovery manages node discovery and heartbeats
type Discovery struct {
	bridge        *NATSBridge
	nodeInfo      *NodeInfo
	cfg           *config.SwarmConfig
	registry      map[string]*NodeInfo
	mu            sync.RWMutex
	heartbeatStop chan struct{}
	cleanupStop   chan struct{}
}

// NewDiscovery creates a new discovery service
func NewDiscovery(bridge *NATSBridge, nodeInfo *NodeInfo, cfg *config.SwarmConfig) *Discovery {
	return &Discovery{
		bridge:        bridge,
		nodeInfo:      nodeInfo,
		cfg:           cfg,
		registry:      make(map[string]*NodeInfo),
		heartbeatStop: make(chan struct{}),
		cleanupStop:   make(chan struct{}),
	}
}

// Start begins heartbeat publishing and node cleanup
func (d *Discovery) Start(ctx context.Context) error {
	// Register callbacks
	d.bridge.SetOnNodeJoin(d.handleNodeJoin)
	d.bridge.SetOnNodeLeave(d.handleNodeLeave)

	// Subscribe to all heartbeats
	if _, err := d.bridge.SubscribeAllHeartbeats(d.handleHeartbeat); err != nil {
		return fmt.Errorf("failed to subscribe to heartbeats: %w", err)
	}

	// Subscribe to shutdown notices
	if _, err := d.bridge.SubscribeShutdown(d.handleNodeLeave); err != nil {
		return fmt.Errorf("failed to subscribe to shutdown notices: %w", err)
	}

	// Start heartbeat goroutine
	go d.heartbeatLoop(ctx)

	// Start cleanup goroutine
	go d.cleanupLoop(ctx)

	// Query for existing nodes
	d.queryNodes()

	logger.InfoC("swarm", "Discovery service started")
	return nil
}

// Stop stops the discovery service
func (d *Discovery) Stop() {
	close(d.heartbeatStop)
	close(d.cleanupStop)
}

// GetNodes returns all known nodes (optionally filtered)
func (d *Discovery) GetNodes(role NodeRole, capability string) []*NodeInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nodes := make([]*NodeInfo, 0)
	for _, node := range d.registry {
		// Skip offline nodes
		if node.Status == StatusOffline {
			continue
		}

		// Filter by role if specified
		if role != "" && node.Role != role {
			continue
		}

		// Filter by capability if specified
		if capability != "" && !containsCapability(node.Capabilities, capability) {
			continue
		}

		nodes = append(nodes, node)
	}
	return nodes
}

// GetNode returns a specific node by ID
func (d *Discovery) GetNode(nodeID string) (*NodeInfo, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	node, ok := d.registry[nodeID]
	return node, ok
}

// NodeCount returns the total number of known online nodes
func (d *Discovery) NodeCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	count := 0
	for _, node := range d.registry {
		if node.Status != StatusOffline {
			count++
		}
	}
	return count
}

// SelectWorker selects the best worker for a capability using load balancing
func (d *Discovery) SelectWorker(capability string) *NodeInfo {
	workers := d.GetNodes(RoleWorker, capability)
	if len(workers) == 0 {
		// Try specialists
		workers = d.GetNodes(RoleSpecialist, capability)
	}
	if len(workers) == 0 {
		return nil
	}

	// Simple load-based selection: pick the one with lowest load
	var best *NodeInfo
	var bestLoad float64 = 2.0 // > 1.0 so any node is better
	for _, w := range workers {
		if w.Load < bestLoad && w.TasksRunning < w.MaxTasks {
			best = w
			bestLoad = w.Load
		}
	}
	return best
}

func (d *Discovery) heartbeatLoop(ctx context.Context) {
	interval := d.cfg.GetHeartbeatInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.publishHeartbeat()
		case <-d.heartbeatStop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (d *Discovery) publishHeartbeat() {
	hb := &Heartbeat{
		NodeID:       d.nodeInfo.ID,
		Status:       d.nodeInfo.Status,
		Load:         d.nodeInfo.Load,
		TasksRunning: d.nodeInfo.TasksRunning,
		Timestamp:    time.Now().UnixMilli(),
	}
	if err := d.bridge.PublishHeartbeat(hb); err != nil {
		logger.DebugCF("swarm", "Failed to publish heartbeat", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (d *Discovery) handleHeartbeat(hb *Heartbeat) {
	// Skip our own heartbeats
	if hb.NodeID == d.nodeInfo.ID {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if node, ok := d.registry[hb.NodeID]; ok {
		node.Status = hb.Status
		node.Load = hb.Load
		node.TasksRunning = hb.TasksRunning
		node.LastSeen = hb.Timestamp
	}
}

func (d *Discovery) cleanupLoop(ctx context.Context) {
	timeout := d.cfg.GetNodeTimeout()
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.cleanupStaleNodes()
		case <-d.cleanupStop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (d *Discovery) cleanupStaleNodes() {
	d.mu.Lock()
	defer d.mu.Unlock()

	timeout := d.cfg.GetNodeTimeout()
	now := time.Now().UnixMilli()
	staleThreshold := now - int64(timeout.Milliseconds())

	for id, node := range d.registry {
		if node.LastSeen < staleThreshold && node.Status != StatusOffline {
			node.Status = StatusOffline
			logger.WarnCF("swarm", "Node marked offline (heartbeat timeout)", map[string]interface{}{
				"node_id":   id,
				"last_seen": time.UnixMilli(node.LastSeen).Format(time.RFC3339),
			})
		}
	}
}

func (d *Discovery) queryNodes() {
	query := &DiscoveryQuery{
		RequesterID: d.nodeInfo.ID,
	}

	nodes, err := d.bridge.RequestDiscovery(query, 2*time.Second)
	if err != nil {
		logger.WarnCF("swarm", "Failed to query existing nodes", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for _, node := range nodes {
		d.handleNodeJoin(node)
	}

	logger.InfoCF("swarm", "Discovery query completed", map[string]interface{}{
		"nodes_found": len(nodes),
	})
}

func (d *Discovery) handleNodeJoin(node *NodeInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Don't register ourselves
	if node.ID == d.nodeInfo.ID {
		return
	}

	node.LastSeen = time.Now().UnixMilli()
	d.registry[node.ID] = node

	logger.InfoCF("swarm", "Node registered", map[string]interface{}{
		"node_id":      node.ID,
		"role":         string(node.Role),
		"capabilities": fmt.Sprintf("%v", node.Capabilities),
	})
}

func (d *Discovery) handleNodeLeave(nodeID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if node, ok := d.registry[nodeID]; ok {
		node.Status = StatusOffline
		logger.InfoCF("swarm", "Node left swarm", map[string]interface{}{
			"node_id": nodeID,
		})
	}
}

// MarshalRegistryJSON returns the current registry as JSON (for debugging)
func (d *Discovery) MarshalRegistryJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return json.Marshal(d.registry)
}

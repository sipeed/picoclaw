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

// GetAllNodes returns all known nodes including offline ones
func (d *Discovery) GetAllNodes() []*NodeInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	nodes := make([]*NodeInfo, 0, len(d.registry))
	for _, node := range d.registry {
		nodes = append(nodes, node)
	}
	return nodes
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
	return d.SelectWorkerWithPriority(capability, 1) // Default to normal priority
}

// SelectWorkerWithPriority selects the best worker considering task priority
// Priority levels: 0=low, 1=normal, 2=high, 3=critical
// Higher priority tasks prefer nodes with lower current load and more available capacity
func (d *Discovery) SelectWorkerWithPriority(capability string, priority int) *NodeInfo {
	workers := d.GetNodes(RoleWorker, capability)
	if len(workers) == 0 {
		// Try specialists
		workers = d.GetNodes(RoleSpecialist, capability)
	}
	if len(workers) == 0 {
		return nil
	}

	// Calculate selection score based on priority
	// For high priority tasks, prefer nodes with:
	// 1. Lower current load
	// 2. More available capacity (maxTasks - tasksRunning)
	// 3. Online status
	var best *NodeInfo
	var bestScore float64 = -1

	for _, w := range workers {
		if w.Status == StatusOffline {
			continue
		}
		if w.TasksRunning >= w.MaxTasks {
			continue // Skip full nodes
		}

		score := d.calculateNodeScore(w, priority)
		if score > bestScore {
			best = w
			bestScore = score
		}
	}

	return best
}

// calculateNodeScore calculates a node's suitability score for a given priority
// Higher score = better candidate
func (d *Discovery) calculateNodeScore(node *NodeInfo, priority int) float64 {
	// Base score: inverse of load (0-1 range, where 1 = idle)
	loadScore := 1.0 - node.Load

	// Capacity score: ratio of available tasks
	capacityScore := float64(node.MaxTasks-node.TasksRunning) / float64(node.MaxTasks)

	// Priority multiplier:
	// - Low priority (0): Prefer busy nodes (0.5x), spread load
	// - Normal priority (1): Standard selection (1.0x)
	// - High priority (2): Prefer idle nodes (1.5x)
	// - Critical priority (3): Strongly prefer idle nodes (2.0x)
	var priorityMult float64
	switch priority {
	case 0:
		priorityMult = 0.5
	case 1:
		priorityMult = 1.0
	case 2:
		priorityMult = 1.5
	case 3:
		priorityMult = 2.0
	default:
		priorityMult = 1.0
	}

	// Final score combines load and capacity, weighted by priority
	// For high priority, idle nodes get much higher scores
	score := (loadScore*0.6 + capacityScore*0.4) * priorityMult

	// Bonus for completely idle nodes for high+ priority
	if node.Load == 0 && priority >= 2 {
		score *= 1.5
	}

	return score
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
	hid, _ := d.nodeInfo.Metadata["hid"]
	sid, _ := d.nodeInfo.Metadata["sid"]
	hb := &Heartbeat{
		NodeID:       d.nodeInfo.ID,
		Role:         d.nodeInfo.Role,
		Status:       d.nodeInfo.Status,
		Load:         d.nodeInfo.Load,
		TasksRunning: d.nodeInfo.TasksRunning,
		Timestamp:    time.Now().UnixMilli(),
		Capabilities: d.nodeInfo.Capabilities,
		HID:          hid,
		SID:          sid,
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
	// GC threshold: 10x timeout to remove long-dead nodes and prevent memory leak
	gcThreshold := now - int64(timeout.Milliseconds()*10)

	for id, node := range d.registry {
		if node.LastSeen < staleThreshold && node.Status != StatusOffline {
			node.Status = StatusOffline
			logger.WarnCF("swarm", "Node marked offline (heartbeat timeout)", map[string]interface{}{
				"node_id":   id,
				"last_seen": time.UnixMilli(node.LastSeen).Format(time.RFC3339),
			})
		}
		// GC long-dead nodes to prevent memory leak
		if node.LastSeen < gcThreshold {
			delete(d.registry, id)
			logger.InfoCF("swarm", "Node removed from registry (GC)", map[string]interface{}{
				"node_id": id,
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
	if node == nil {
		logger.WarnC("swarm", "Attempted to register nil node")
		return
	}
	if node.ID == "" {
		logger.WarnC("swarm", "Attempted to register node with empty ID")
		return
	}
	if node.ID == d.nodeInfo.ID {
		return // Skip self
	}

	// Validate role
	validRoles := map[NodeRole]bool{RoleCoordinator: true, RoleWorker: true, RoleSpecialist: true}
	if !validRoles[node.Role] {
		logger.WarnCF("swarm", "Invalid node role", map[string]interface{}{
			"node_id": node.ID,
			"role":    string(node.Role),
		})
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

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

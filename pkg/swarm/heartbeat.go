// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// HeartbeatInterval is how often nodes send heartbeat messages
	HeartbeatInterval = 10 * time.Second
	// HeartbeatSuspiciousThreshold is how long before a node is marked suspicious
	HeartbeatSuspiciousThreshold = 30 * time.Second
	// HeartbeatOfflineThreshold is how long before a node is marked offline
	HeartbeatOfflineThreshold = 60 * time.Second
)

// HeartbeatConfig configures heartbeat behavior
type HeartbeatConfig struct {
	Interval           time.Duration // Heartbeat send interval
	SuspiciousTimeout  time.Duration // Time before marking suspicious
	OfflineTimeout     time.Duration // Time before marking offline
}

// DefaultHeartbeatConfig returns the default heartbeat configuration
func DefaultHeartbeatConfig() *HeartbeatConfig {
	return &HeartbeatConfig{
		Interval:           HeartbeatInterval,
		SuspiciousTimeout:  HeartbeatSuspiciousThreshold,
		OfflineTimeout:     HeartbeatOfflineThreshold,
	}
}

// HeartbeatPublisher sends periodic heartbeat messages for a node
type HeartbeatPublisher struct {
	bridge    *NATSBridge
	nodeInfo  *NodeInfo
	cfg       *HeartbeatConfig
	ticker    *time.Ticker
	stopChan  chan struct{}
	running   bool
	mu        sync.RWMutex
}

// NewHeartbeatPublisher creates a new heartbeat publisher
func NewHeartbeatPublisher(bridge *NATSBridge, nodeInfo *NodeInfo, cfg *HeartbeatConfig) *HeartbeatPublisher {
	if cfg == nil {
		cfg = DefaultHeartbeatConfig()
	}
	return &HeartbeatPublisher{
		bridge:   bridge,
		nodeInfo: nodeInfo,
		cfg:      cfg,
		stopChan: make(chan struct{}),
	}
}

// Start begins sending heartbeat messages
func (hp *HeartbeatPublisher) Start(ctx context.Context) error {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	if hp.running {
		return nil
	}

	hp.ticker = time.NewTicker(hp.cfg.Interval)
	hp.running = true

	go hp.run(ctx)

	logger.InfoCF("swarm", "Heartbeat publisher started", map[string]interface{}{
		"node_id":  hp.nodeInfo.ID,
		"interval": hp.cfg.Interval.String(),
	})

	return nil
}

// Stop stops sending heartbeat messages
func (hp *HeartbeatPublisher) Stop() {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	if !hp.running {
		return
	}

	close(hp.stopChan)
	if hp.ticker != nil {
		hp.ticker.Stop()
	}
	hp.running = false

	logger.InfoC("swarm", "Heartbeat publisher stopped")
}

func (hp *HeartbeatPublisher) run(ctx context.Context) {
	// Send first heartbeat immediately
	hp.sendHeartbeat()

	for {
		select {
		case <-hp.ticker.C:
			hp.sendHeartbeat()
		case <-hp.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (hp *HeartbeatPublisher) sendHeartbeat() {
	hb := &Heartbeat{
		NodeID:       hp.nodeInfo.ID,
		Timestamp:    time.Now().UnixMilli(),
		Load:         hp.nodeInfo.Load,
		TasksRunning: hp.nodeInfo.TasksRunning,
		Status:       hp.nodeInfo.Status,
		Capabilities: hp.nodeInfo.Capabilities,
	}

	if err := hp.bridge.PublishHeartbeat(hb); err != nil {
		logger.DebugCF("swarm", "Failed to publish heartbeat", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// HeartbeatMonitor tracks heartbeats from other nodes
type HeartbeatMonitor struct {
	cfg        *HeartbeatConfig
	discovery  *Discovery
	heartbeats map[string]int64 // node_id -> last heartbeat timestamp
	mu         sync.RWMutex
	stopChan   chan struct{}
	running    bool
}

// NewHeartbeatMonitor creates a new heartbeat monitor
func NewHeartbeatMonitor(discovery *Discovery, cfg *HeartbeatConfig) *HeartbeatMonitor {
	if cfg == nil {
		cfg = DefaultHeartbeatConfig()
	}
	return &HeartbeatMonitor{
		cfg:        cfg,
		discovery:  discovery,
		heartbeats: make(map[string]int64),
		stopChan:   make(chan struct{}),
	}
}

// Start begins monitoring heartbeats
func (hm *HeartbeatMonitor) Start(ctx context.Context) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return nil
	}

	hm.running = true

	// Start checker goroutine
	go hm.runChecker(ctx)

	logger.InfoC("swarm", "Heartbeat monitor started")
	return nil
}

// Stop stops monitoring
func (hm *HeartbeatMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.running {
		return
	}

	close(hm.stopChan)
	hm.running = false

	logger.InfoC("swarm", "Heartbeat monitor stopped")
}

// UpdateHeartbeat records a heartbeat from a node
func (hm *HeartbeatMonitor) UpdateHeartbeat(hb *Heartbeat) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now().UnixMilli()
	hm.heartbeats[hb.NodeID] = now

	// Update node status in discovery based on heartbeat
	node, ok := hm.discovery.GetNode(hb.NodeID)
	if ok {
		node.Load = hb.Load
		node.TasksRunning = hb.TasksRunning
		node.LastSeen = now
		// If node was offline/suspicious, mark it back online
		if node.Status == StatusOffline || node.Status == StatusSuspicious {
			node.Status = hb.Status
		}
	}

	logger.DebugCF("swarm", "Heartbeat received", map[string]interface{}{
		"node_id":  hb.NodeID,
		"status":   string(hb.Status),
		"load":     hb.Load,
		"tasks":    hb.TasksRunning,
	})
}

// runChecker periodically checks for missed heartbeats
func (hm *HeartbeatMonitor) runChecker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.checkHeartbeats()
		case <-hm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkHeartbeats checks all tracked nodes for missed heartbeats
func (hm *HeartbeatMonitor) checkHeartbeats() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now().UnixMilli()
	suspiciousThreshold := now - hm.cfg.SuspiciousTimeout.Milliseconds()
	offlineThreshold := now - hm.cfg.OfflineTimeout.Milliseconds()

	for nodeID, lastHB := range hm.heartbeats {
		node, ok := hm.discovery.GetNode(nodeID)
		if !ok {
			continue
		}

		if lastHB < offlineThreshold {
			// Node is offline
			if node.Status != StatusOffline {
				logger.WarnCF("swarm", "Node marked offline", map[string]interface{}{
					"node_id":      nodeID,
					"last_heartbeat": time.UnixMilli(lastHB).Format(time.RFC3339),
				})
				node.Status = StatusOffline
				hm.discovery.handleNodeLeave(nodeID)
			}
		} else if lastHB < suspiciousThreshold {
			// Node is suspicious
			if node.Status != StatusSuspicious && node.Status != StatusOffline {
				logger.WarnCF("swarm", "Node marked suspicious", map[string]interface{}{
					"node_id": nodeID,
				})
				node.Status = StatusSuspicious
			}
		}
	}
}

// RemoveNode stops monitoring a node
func (hm *HeartbeatMonitor) RemoveNode(nodeID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.heartbeats, nodeID)
}

// GetLastHeartbeat returns the last heartbeat time for a node
func (hm *HeartbeatMonitor) GetLastHeartbeat(nodeID string) time.Time {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	ts, ok := hm.heartbeats[nodeID]
	if !ok {
		return time.Time{}
	}
	return time.UnixMilli(ts)
}

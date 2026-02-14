// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Manager orchestrates all swarm components
type Manager struct {
	cfg          *config.Config
	embeddedNATS *EmbeddedNATS
	bridge       *NATSBridge
	temporal     *TemporalClient
	discovery    *Discovery
	coordinator  *Coordinator
	worker       *Worker
	nodeInfo     *NodeInfo
	agentLoop    *agent.AgentLoop
	localBus     *bus.MessageBus
}

// NewManager creates a new swarm manager
func NewManager(cfg *config.Config, agentLoop *agent.AgentLoop, provider providers.LLMProvider, localBus *bus.MessageBus) *Manager {
	swarmCfg := &cfg.Swarm

	// Validate configuration
	if err := swarmCfg.Validate(); err != nil {
		logger.ErrorCF("swarm", "Invalid configuration", map[string]interface{}{"error": err.Error()})
		return nil
	}

	// Generate node ID if not set
	nodeID := swarmCfg.NodeID
	if nodeID == "" {
		nodeID = fmt.Sprintf("claw-%s", uuid.New().String()[:8])
	}

	// Create node info
	nodeInfo := &NodeInfo{
		ID:           nodeID,
		Role:         NodeRole(swarmCfg.Role),
		Capabilities: swarmCfg.Capabilities,
		Model:        cfg.Agents.Defaults.Model,
		Status:       StatusOnline,
		MaxTasks:     swarmCfg.MaxConcurrent,
		StartedAt:    time.Now().UnixMilli(),
		Metadata:     make(map[string]string),
	}

	m := &Manager{
		cfg:       cfg,
		nodeInfo:  nodeInfo,
		agentLoop: agentLoop,
		localBus:  localBus,
	}

	// Create components
	m.bridge = NewNATSBridge(swarmCfg, localBus, nodeInfo)
	m.temporal = NewTemporalClient(&swarmCfg.Temporal)
	m.discovery = NewDiscovery(m.bridge, nodeInfo, swarmCfg)

	// Create role-specific components
	if nodeInfo.Role == RoleCoordinator {
		m.coordinator = NewCoordinator(swarmCfg, m.bridge, m.temporal, m.discovery, agentLoop, provider, localBus)
	}
	if nodeInfo.Role == RoleWorker || nodeInfo.Role == RoleSpecialist {
		m.worker = NewWorker(swarmCfg, m.bridge, m.temporal, agentLoop, provider, nodeInfo)
	}

	return m
}

// Start initializes and starts all swarm components
func (m *Manager) Start(ctx context.Context) error {
	swarmCfg := &m.cfg.Swarm

	// Start embedded NATS if configured
	if swarmCfg.NATS.Embedded {
		m.embeddedNATS = NewEmbeddedNATS(&swarmCfg.NATS)
		if err := m.embeddedNATS.Start(); err != nil {
			return fmt.Errorf("failed to start embedded NATS: %w", err)
		}
		// Override URLs to connect to embedded server
		swarmCfg.NATS.URLs = []string{m.embeddedNATS.ClientURL()}
	}

	// Connect to NATS
	if err := m.bridge.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Start NATS bridge
	if err := m.bridge.Start(ctx); err != nil {
		return fmt.Errorf("failed to start NATS bridge: %w", err)
	}

	// Connect to Temporal (non-fatal if unavailable)
	if err := m.temporal.Connect(ctx); err != nil {
		logger.WarnCF("swarm", "Temporal connection failed (workflows disabled)", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start Temporal worker with workflow registrations if connected
	if m.temporal.IsConnected() {
		wfs := []interface{}{SwarmWorkflow}
		acts := []interface{}{
			DecomposeTaskActivity,
			ExecuteDirectActivity,
			ExecuteSubtaskActivity,
			SynthesizeResultsActivity,
		}
		if err := m.temporal.StartWorker(ctx, wfs, acts); err != nil {
			logger.WarnCF("swarm", "Failed to start Temporal worker", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Start discovery
	if err := m.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}

	// Start role-specific components
	if m.coordinator != nil {
		if err := m.coordinator.Start(ctx); err != nil {
			return fmt.Errorf("failed to start coordinator: %w", err)
		}
	}

	if m.worker != nil {
		if err := m.worker.Start(ctx); err != nil {
			return fmt.Errorf("failed to start worker: %w", err)
		}
	}

	logger.InfoCF("swarm", "Swarm manager started", map[string]interface{}{
		"node_id":      m.nodeInfo.ID,
		"role":         string(m.nodeInfo.Role),
		"capabilities": fmt.Sprintf("%v", m.nodeInfo.Capabilities),
		"nats":         m.bridge.IsConnected(),
		"temporal":     m.temporal.IsConnected(),
	})

	return nil
}

// Stop gracefully stops all swarm components
func (m *Manager) Stop() {
	logger.InfoC("swarm", "Stopping swarm manager")

	if m.worker != nil {
		m.worker.Stop()
	}

	if m.coordinator != nil {
		m.coordinator.Stop()
	}

	m.discovery.Stop()
	m.temporal.Stop()
	if err := m.bridge.Stop(); err != nil {
		logger.WarnCF("swarm", "Error stopping NATS bridge", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if m.embeddedNATS != nil {
		m.embeddedNATS.Stop()
	}

	logger.InfoC("swarm", "Swarm manager stopped")
}

// GetNodeInfo returns this node's information
func (m *Manager) GetNodeInfo() *NodeInfo {
	return m.nodeInfo
}

// GetDiscoveredNodes returns all discovered nodes
func (m *Manager) GetDiscoveredNodes() []*NodeInfo {
	return m.discovery.GetNodes("", "")
}

// IsNATSConnected returns true if connected to NATS
func (m *Manager) IsNATSConnected() bool {
	return m.bridge.IsConnected()
}

// IsTemporalConnected returns true if connected to Temporal
func (m *Manager) IsTemporalConnected() bool {
	return m.temporal.IsConnected()
}

// DiscoveredNodeCount returns the count of discovered nodes
func (m *Manager) DiscoveredNodeCount() int {
	return m.discovery.NodeCount()
}

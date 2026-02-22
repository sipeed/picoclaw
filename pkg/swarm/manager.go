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
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Manager orchestrates all swarm components
type Manager struct {
	cfg             *config.Config
	provider        providers.LLMProvider
	embeddedNATS    *EmbeddedNATS
	bridge          *NATSBridge
	temporal        *TemporalClient
	discovery       *Discovery
	coordinator     *Coordinator
	worker          *Worker
	specialist      *SpecialistNode
	activities      *Activities
	lifecycle       *TaskLifecycleStore
	checkpointStore *CheckpointStore
	failoverManager *FailoverManager
	contextPool     *ContextPool
	electionMgr     *ElectionManager
	roleSwitcher    *RoleSwitcher
	identity        *identity.LoadedIdentity
	nodeInfo        *NodeInfo
	agentLoop       *agent.AgentLoop
	localBus        *bus.MessageBus
	enableElection  bool // Enable leader election for dynamic role switching
}

// NewManager creates a new swarm manager
func NewManager(cfg *config.Config, agentLoop *agent.AgentLoop, provider providers.LLMProvider, localBus *bus.MessageBus) *Manager {
	swarmCfg := &cfg.Swarm

	// Validate configuration
	if err := swarmCfg.Validate(); err != nil {
		logger.ErrorCF("swarm", "Invalid configuration", map[string]interface{}{"error": err.Error()})
		return nil
	}

	// Load or generate identity
	identityLoader := identity.NewLoader()
	identityLoader.SetConfig(swarmCfg.HID, swarmCfg.SID)
	loadedIdentity := identityLoader.LoadOrGenerate()
	hid := loadedIdentity.HID
	sid := loadedIdentity.SID

	// Set identity on agent loop for cross-instance communication
	agentLoop.SetIdentity(hid, sid)

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
	// Store identity in node metadata for discovery
	nodeInfo.Metadata["hid"] = hid
	nodeInfo.Metadata["sid"] = sid

	m := &Manager{
		cfg:      cfg,
		provider: provider,
		identity: loadedIdentity,
		nodeInfo: nodeInfo,
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
	if nodeInfo.Role == RoleSpecialist {
		// Create specialist node for capability-based routing
		m.specialist = NewSpecialistNode(swarmCfg, m.bridge, m.temporal, agentLoop, provider, nodeInfo, m.bridge.js, m.bridge.nc, "")
	}

	logger.InfoCF("swarm", "Swarm manager initialized with identity", map[string]interface{}{
		"hid":       hid,
		"sid":       sid,
		"source":    loadedIdentity.Source.String(),
		"node_id":   nodeID,
		"role":      string(nodeInfo.Role),
	})

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

	// Create activities instance for LLM-driven task operations
	m.activities = NewActivities(m.provider, m.agentLoop, &m.cfg.Swarm, m.nodeInfo)

	// Start Temporal worker with workflow registrations if connected
	if m.temporal.IsConnected() {
		wfs := []interface{}{SwarmWorkflow}
		if err := m.temporal.StartWorker(ctx, wfs, m.activities); err != nil {
			logger.WarnCF("swarm", "Failed to start Temporal worker", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Start discovery
	if err := m.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}

	// Initialize lifecycle store
	m.lifecycle = NewTaskLifecycleStore(m.bridge.js)
	if err := m.lifecycle.Initialize(ctx); err != nil {
		logger.WarnCF("swarm", "Failed to initialize lifecycle store", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize checkpoint store
	var err error
	m.checkpointStore, err = NewCheckpointStore(m.bridge.js)
	if err != nil {
		logger.WarnCF("swarm", "Failed to initialize checkpoint store", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize failover manager
	if m.lifecycle != nil && m.checkpointStore != nil {
		m.failoverManager = NewFailoverManager(m.discovery, m.lifecycle, m.checkpointStore, m.bridge, m.nodeInfo, m.bridge.js)
		if err := m.failoverManager.Start(ctx); err != nil {
			logger.WarnCF("swarm", "Failed to start failover manager", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Initialize shared context pool
	m.contextPool = NewContextPool(m.bridge.js, m.nodeInfo.ID, m.identity.HID, m.identity.SID)
	if err := m.contextPool.Start(ctx); err != nil {
		logger.WarnCF("swarm", "Failed to start context pool", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		logger.InfoCF("swarm", "Shared context pool started", map[string]interface{}{
			"hid":      m.identity.HID,
			"sid":      m.identity.SID,
			"node_id":  m.nodeInfo.ID,
		})
	}

	// Initialize leader election if enabled
	if m.enableElection {
		m.electionMgr = NewElectionManager(m.bridge.nc, m.bridge.js, m.nodeInfo.ID, m.identity.HID, m.identity.SID)
		m.roleSwitcher = NewRoleSwitcher(m.electionMgr, m.nodeInfo, m)

		electionCfg := &ElectionConfig{
			ElectionSubject:   fmt.Sprintf("picoclaw.election.%s", m.identity.HID),
			LeaseDuration:     10 * time.Second,
			ElectionInterval: 3 * time.Second,
			PreVoteDelay:      time.Duration(0),
		}

		if err := m.electionMgr.Start(ctx, electionCfg); err != nil {
			logger.WarnCF("swarm", "Failed to start election manager", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			logger.InfoCF("swarm", "Leader election enabled", map[string]interface{}{
				"node_id": m.nodeInfo.ID,
				"hid":     m.identity.HID,
			})
		}
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

	if m.specialist != nil {
		if err := m.specialist.Start(ctx); err != nil {
			return fmt.Errorf("failed to start specialist: %w", err)
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

	if m.specialist != nil {
		m.specialist.Stop()
	}

	if m.worker != nil {
		m.worker.Stop()
	}

	if m.coordinator != nil {
		m.coordinator.Stop()
	}

	if m.electionMgr != nil {
		m.electionMgr.Stop()
	}

	if m.failoverManager != nil {
		m.failoverManager.Stop()
	}

	if m.contextPool != nil {
		m.contextPool.Stop()
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

// GetContextPool returns the shared context pool
func (m *Manager) GetContextPool() *ContextPool {
	return m.contextPool
}

// GetIdentity returns the node's identity
func (m *Manager) GetIdentity() *identity.LoadedIdentity {
	return m.identity
}

// IsLeader returns true if this node is the leader (via election)
func (m *Manager) IsLeader() bool {
	if m.electionMgr != nil {
		return m.electionMgr.IsLeader()
	}
	return false
}

// GetLeaderID returns the current leader's node ID
func (m *Manager) GetLeaderID() string {
	if m.electionMgr != nil {
		return m.electionMgr.GetLeaderID()
	}
	return ""
}

// SetElectionEnabled enables or disables leader election
func (m *Manager) SetElectionEnabled(enabled bool) {
	m.enableElection = enabled
}

// handleRoleChange handles dynamic role changes
func (m *Manager) handleRoleChange(newRole NodeRole) {
	logger.InfoCF("swarm", "Handling role change", map[string]interface{}{
		"node_id":    m.nodeInfo.ID,
		"new_role":   string(newRole),
		"old_role":   string(m.nodeInfo.Role),
	})

	// Stop old role components
	switch m.nodeInfo.Role {
	case RoleCoordinator:
		if m.coordinator != nil {
			m.coordinator.Stop()
			m.coordinator = nil
		}
	case RoleWorker:
		if m.worker != nil {
			m.worker.Stop()
			m.worker = nil
		}
	case RoleSpecialist:
		if m.specialist != nil {
			m.specialist.Stop()
			m.specialist = nil
		}
	}

	// Update node role
	m.nodeInfo.Role = newRole

	// Start new role components
	swarmCfg := &m.cfg.Swarm
	switch newRole {
	case RoleCoordinator:
		m.coordinator = NewCoordinator(swarmCfg, m.bridge, m.temporal, m.discovery, m.agentLoop, m.provider, m.localBus)
		if m.coordinator != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.coordinator.Start(ctx); err != nil {
				logger.ErrorCF("swarm", "Failed to start coordinator after role change", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	case RoleWorker:
		m.worker = NewWorker(swarmCfg, m.bridge, m.temporal, m.agentLoop, m.provider, m.nodeInfo)
		if m.worker != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.worker.Start(ctx); err != nil {
				logger.ErrorCF("swarm", "Failed to start worker after role change", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	case RoleSpecialist:
		m.worker = NewWorker(swarmCfg, m.bridge, m.temporal, m.agentLoop, m.provider, m.nodeInfo)
		m.specialist = NewSpecialistNode(swarmCfg, m.bridge, m.temporal, m.agentLoop, m.provider, m.nodeInfo, m.bridge.js, m.bridge.nc, "")
		if m.worker != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.worker.Start(ctx); err != nil {
				logger.ErrorCF("swarm", "Failed to start worker after role change", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
		if m.specialist != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.specialist.Start(ctx); err != nil {
				logger.ErrorCF("swarm", "Failed to start specialist after role change", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}

	logger.InfoCF("swarm", "Role change completed", map[string]interface{}{
		"node_id": m.nodeInfo.ID,
		"role":    string(newRole),
	})
}

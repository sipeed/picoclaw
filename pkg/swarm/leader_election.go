// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
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

// LeaderElection handles leader election using a simple bully algorithm variant.
type LeaderElection struct {
	localNodeID string
	membership  *MembershipManager
	config      LeaderElectionConfig

	mu                 sync.RWMutex
	currentLeader      string
	isLeader           bool
	electionInProgress bool
	leaderChangeCh     chan string
	stopCh             chan struct{}
}

// NewLeaderElection creates a new leader election instance.
func NewLeaderElection(nodeID string, membership *MembershipManager, config LeaderElectionConfig) *LeaderElection {
	return &LeaderElection{
		localNodeID:    nodeID,
		membership:     membership,
		config:         config,
		leaderChangeCh: make(chan string, 10),
		stopCh:         make(chan struct{}),
	}
}

// Start starts the leader election process.
func (le *LeaderElection) Start() {
	// Start election checker
	go le.electionChecker()

	// Start leader heartbeat monitor
	go le.leaderMonitor()
}

// Stop stops the leader election process.
func (le *LeaderElection) Stop() {
	close(le.stopCh)
}

// IsLeader returns true if this node is the current leader.
func (le *LeaderElection) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.isLeader
}

// GetLeader returns the current leader ID.
func (le *LeaderElection) GetLeader() string {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.currentLeader
}

// LeaderChanges returns a channel that receives leader ID changes.
func (le *LeaderElection) LeaderChanges() <-chan string {
	return le.leaderChangeCh
}

// electionChecker periodically checks if we should become leader.
func (le *LeaderElection) electionChecker() {
	interval := le.config.ElectionInterval.Duration
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-le.stopCh:
			return
		case <-ticker.C:
			le.checkElection()
		}
	}
}

// checkElection runs the leader election algorithm.
func (le *LeaderElection) checkElection() {
	le.mu.Lock()
	defer le.mu.Unlock()

	members := le.membership.GetMembers()

	// Filter to only alive nodes for leader candidacy
	aliveMembers := make([]*NodeWithState, 0, len(members))
	for _, m := range members {
		if m.State != nil && m.State.Status == NodeStatusAlive {
			aliveMembers = append(aliveMembers, m)
		}
	}

	if len(aliveMembers) == 0 {
		// No alive members in view (including self not yet registered), become leader as fallback
		le.becomeLeader()
		return
	}

	// Find the node with the lowest ID (simple deterministic leader selection)
	candidateID := le.localNodeID
	for _, m := range aliveMembers {
		if m.Node.ID < candidateID {
			candidateID = m.Node.ID
		}
	}

	// Update current leader
	if le.currentLeader != candidateID {
		le.currentLeader = candidateID

		if candidateID == le.localNodeID {
			le.becomeLeader()
		} else {
			le.becomeFollower()
		}
	}
}

// becomeLeader marks this node as the leader.
func (le *LeaderElection) becomeLeader() {
	if !le.isLeader {
		le.isLeader = true
		logger.InfoCF("swarm", "This node is now the leader", map[string]any{"node_id": le.localNodeID})

		// Notify listeners (non-blocking)
		select {
		case le.leaderChangeCh <- le.localNodeID:
		default:
			logger.WarnC("swarm", "Leader change notification dropped, channel full")
		}
	}
}

// becomeFollower marks this node as a follower.
func (le *LeaderElection) becomeFollower() {
	wasLeader := le.isLeader
	le.isLeader = false

	if wasLeader {
		logger.InfoCF("swarm", "This node is now a follower", map[string]any{
			"node_id":    le.localNodeID,
			"new_leader": le.currentLeader,
		})

		// Notify listeners of leader change (non-blocking)
		select {
		case le.leaderChangeCh <- le.currentLeader:
		default:
			logger.WarnC("swarm", "Leader change notification dropped, channel full")
		}
	}
}

// leaderMonitor monitors if the current leader is still alive.
func (le *LeaderElection) leaderMonitor() {
	interval := le.config.LeaderHeartbeatTimeout.Duration
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-le.stopCh:
			return
		case <-ticker.C:
			le.monitorLeader()
		}
	}
}

// monitorLeader checks if the current leader is still alive.
func (le *LeaderElection) monitorLeader() {
	le.mu.RLock()
	leaderID := le.currentLeader
	amLeader := le.isLeader
	le.mu.RUnlock()

	if amLeader || leaderID == "" {
		return
	}

	// Check if leader is still in the membership and healthy
	needReelection := false
	node, exists := le.membership.GetNode(leaderID)
	if !exists {
		logger.WarnCF("swarm", "Leader no longer in membership, triggering reelection",
			map[string]any{"leader_id": leaderID})
		needReelection = true
	} else if node.State != nil && node.State.Status != NodeStatusAlive {
		logger.WarnCF("swarm", "Leader is no longer alive, triggering reelection",
			map[string]any{"leader_id": leaderID, "status": node.State.Status})
		needReelection = true
	}

	if needReelection {
		le.mu.Lock()
		le.currentLeader = ""
		le.mu.Unlock()
		le.checkElection()
	}
}

// ElectLeader triggers a new leader election.
func (le *LeaderElection) ElectLeader(ctx context.Context) (string, error) {
	le.mu.Lock()
	le.currentLeader = "" // Clear current leader to trigger reelection
	le.mu.Unlock()

	// Run election immediately
	le.checkElection()

	// Wait for new leader
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			le.mu.RLock()
			leader := le.currentLeader
			le.mu.RUnlock()

			if leader != "" {
				return leader, nil
			}
		}
	}
}

// LeadershipState represents the current leadership state.
type LeadershipState struct {
	LeaderID    string    `json:"leader_id"`
	IsLeader    bool      `json:"is_leader"`
	LastChange  time.Time `json:"last_change"`
	MemberCount int       `json:"member_count"`
}

// GetState returns the current leadership state.
func (le *LeaderElection) GetState() LeadershipState {
	le.mu.RLock()
	defer le.mu.RUnlock()

	return LeadershipState{
		LeaderID:    le.currentLeader,
		IsLeader:    le.isLeader,
		MemberCount: len(le.membership.GetMembers()),
	}
}

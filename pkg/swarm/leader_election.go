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

	mu                 sync.RWMutex
	currentLeader      string
	isLeader           bool
	electionInProgress bool
	leaderChangeCh     chan string
	stopCh             chan struct{}
}

// NewLeaderElection creates a new leader election instance.
func NewLeaderElection(nodeID string, membership *MembershipManager) *LeaderElection {
	return &LeaderElection{
		localNodeID:    nodeID,
		membership:     membership,
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
	ticker := time.NewTicker(time.Second * 5)
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
	if len(members) == 0 {
		// No other members, we become leader
		le.becomeLeader()
		return
	}

	// Find the node with the lowest ID (simple deterministic leader selection)
	var candidateID string
	candidateID = le.localNodeID

	for _, m := range members {
		if m.Node.ID < candidateID {
			candidateID = m.Node.ID
		}
	}

	// Update current leader
	if le.currentLeader != candidateID {
		oldLeader := le.currentLeader
		le.currentLeader = candidateID

		if candidateID == le.localNodeID {
			le.becomeLeader()
		} else {
			le.becomeFollower()
		}

		logger.InfoCF("swarm", "Leader changed", map[string]any{"old_leader": oldLeader, "new_leader": candidateID})

		// Notify followers of leader change
		select {
		case le.leaderChangeCh <- candidateID:
		default:
		}
	}
}

// becomeLeader marks this node as the leader.
func (le *LeaderElection) becomeLeader() {
	if !le.isLeader {
		le.isLeader = true
		logger.InfoCF("swarm", "This node is now the leader", map[string]any{"node_id": le.localNodeID})

		// Notify listeners
		select {
		case le.leaderChangeCh <- le.localNodeID:
		default:
		}
	}
}

// becomeFollower marks this node as a follower.
func (le *LeaderElection) becomeFollower() {
	if le.isLeader {
		le.isLeader = false
		logger.InfoCF("swarm", "This node is now a follower", map[string]any{"node_id": le.localNodeID})
	}
}

// leaderMonitor monitors if the current leader is still alive.
func (le *LeaderElection) leaderMonitor() {
	ticker := time.NewTicker(time.Second * 10)
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

	// Check if leader is still in the membership
	if _, exists := le.membership.GetNode(leaderID); !exists {
		logger.WarnCF("swarm", "Leader no longer in membership, triggering reelection", map[string]any{"leader_id": leaderID})
		// Trigger reelection by clearing current leader
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

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ElectionManager manages leader election using NATS JetStream KV store
type ElectionManager struct {
	// NATS connection
	nc *nats.Conn
	js nats.JetStreamContext

	// Identity
	nodeID  string
	hid     string
	sid     string

	// Election configuration
	electionSubject string // Subject for election coordination
	leaseDuration   time.Duration
	leaderKey       string // KV store key for leader lease

	// State
	mu              sync.RWMutex
	isLeader        bool
	isParticipant   bool
	currentLeaderID string
	leaseExpiry     int64
	lastRevision    uint64 // Last known revision for optimistic updates

	// Channels
	leaderChan    chan bool // true when becomes leader, false when loses leadership
	stopChan      chan struct{}
	electionTimer  *time.Timer

	// NATS subscription
	leaseSub      *nats.Subscription

	// Callbacks
	onBecameLeader   func()
	onLostLeadership func()
	onNewLeader      func(leaderID string)
}

// ElectionConfig holds configuration for leader election
type ElectionConfig struct {
	// ElectionSubject is the NATS subject for election messages
	ElectionSubject string

	// LeaseDuration is how long a leader lease is valid
	LeaseDuration time.Duration

	// ElectionInterval is how often to check/renew leadership
	ElectionInterval time.Duration

	// PreVoteDelay is delay before attempting election (for staggered starts)
	PreVoteDelay time.Duration
}

// DefaultElectionConfig returns default election configuration
func DefaultElectionConfig() *ElectionConfig {
	return &ElectionConfig{
		ElectionSubject:   "picoclaw.election",
		LeaseDuration:     10 * time.Second,
		ElectionInterval:  3 * time.Second,
		PreVoteDelay:      time.Duration(0),
	}
}

// NewElectionManager creates a new election manager
func NewElectionManager(nc *nats.Conn, js nats.JetStreamContext, nodeID, hid, sid string) *ElectionManager {
	return &ElectionManager{
		nc:           nc,
		js:           js,
		nodeID:       nodeID,
		hid:          hid,
		sid:          sid,
		leaderKey:    fmt.Sprintf("picoclaw.election.%s.leader", hid),
		leaderChan:   make(chan bool, 1),
		stopChan:     make(chan struct{}),
	}
}

// Start begins participating in leader election
func (em *ElectionManager) Start(ctx context.Context, cfg *ElectionConfig) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.isParticipant {
		return nil
	}

	if cfg == nil {
		cfg = DefaultElectionConfig()
	}

	em.electionSubject = cfg.ElectionSubject
	em.leaseDuration = cfg.LeaseDuration

	// Create KV store for leader lease (or get if exists)
	bucketName := fmt.Sprintf("PICOCLAW_ELECTION_%s", em.hid)
	_, err := em.js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: bucketName,
		TTL:    cfg.LeaseDuration * 2,
	})
	if err != nil {
		// Check if it already exists - try to bind to it
		_, bindErr := em.js.KeyValue(bucketName)
		if bindErr != nil {
			return fmt.Errorf("failed to create or bind to election KV store: %w", err)
		}
	}

	// Store bucket name for later use
	em.leaderKey = fmt.Sprintf("leader.%s", em.hid)

	// Subscribe to leadership change notifications
	subject := fmt.Sprintf("$KV.%s.>", bucketName)
	sub, err := em.nc.Subscribe(subject, func(msg *nats.Msg) {
		em.handleLeaseUpdate(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to lease updates: %w", err)
	}
	em.leaseSub = sub

	em.isParticipant = true

	// Start election goroutine
	go em.electionLoop(ctx, cfg)

	logger.InfoCF("swarm", "Election manager started", map[string]interface{}{
		"node_id":   em.nodeID,
		"hid":       em.hid,
		"lease_ttl": cfg.LeaseDuration.String(),
	})

	return nil
}

// Stop stops participating in leader election
func (em *ElectionManager) Stop() {
	em.mu.Lock()
	if !em.isParticipant {
		em.mu.Unlock()
		return
	}

	wasLeader := em.isLeader
	em.isParticipant = false
	em.mu.Unlock()

	// Close NATS subscription first to prevent new callbacks
	if em.leaseSub != nil {
		em.leaseSub.Unsubscribe()
		em.leaseSub = nil
	}

	// Stop the election loop
	close(em.stopChan)

	if em.electionTimer != nil {
		em.electionTimer.Stop()
	}

	// Step down from leadership without holding the lock
	if wasLeader {
		em.stepDown()
	}

	logger.InfoCF("swarm", "Election manager stopped", map[string]interface{}{
		"node_id": em.nodeID,
	})
}

// electionLoop runs the election logic
func (em *ElectionManager) electionLoop(ctx context.Context, cfg *ElectionConfig) {
	// Initial delay for staggered starts across nodes
	if cfg.PreVoteDelay > 0 {
		time.Sleep(cfg.PreVoteDelay)
	}

	ticker := time.NewTicker(cfg.ElectionInterval)
	defer ticker.Stop()

	// Try to become leader immediately
	em.attemptLeadership()

	for {
		select {
		case <-ctx.Done():
			return
		case <-em.stopChan:
			return
		case isLeader := <-em.leaderChan:
			em.mu.Lock()
			em.isLeader = isLeader
			em.mu.Unlock()

			if isLeader {
				logger.InfoCF("swarm", "Became leader", map[string]interface{}{
					"node_id": em.nodeID,
				})
				if em.onBecameLeader != nil {
					em.onBecameLeader()
				}
			} else {
				logger.WarnCF("swarm", "Lost leadership", map[string]interface{}{
					"node_id": em.nodeID,
				})
				if em.onLostLeadership != nil {
					em.onLostLeadership()
				}
				// Try to regain leadership
				em.attemptLeadership()
			}

		case <-ticker.C:
			// Renew lease if we're leader
			em.mu.RLock()
			isLeader := em.isLeader
			em.mu.RUnlock()

			if isLeader {
				em.renewLease()
			} else {
				// Periodically attempt to acquire leadership
				em.attemptLeadership()
			}
		}
	}
}

// attemptLeadership tries to acquire leadership lease
func (em *ElectionManager) attemptLeadership() {
	kv, err := em.js.KeyValue(fmt.Sprintf("PICOCLAW_ELECTION_%s", em.hid))
	if err != nil {
		logger.DebugCF("swarm", "Failed to get election KV", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Try to get current entry first
	entry, err := kv.Get(em.leaderKey)

	now := time.Now().UnixMilli()
	leaseExpiry := now + em.leaseDuration.Milliseconds()
	infoBytes := []byte(fmt.Sprintf("%s|%s|%d", em.nodeID, em.sid, leaseExpiry))

	if err != nil {
		// No entry exists, try to create
		revision, err := kv.Create(em.leaderKey, infoBytes)
		if err != nil {
			// Someone else created it first, or other error
			return
		}

		// Verify we're actually the leader by reading back
		entry, err := kv.Get(em.leaderKey)
		if err != nil {
			return
		}

		entryLeader, _, entryExpiry, ok := parseLeaderInfo(string(entry.Value()))
		if !ok || entryLeader != em.nodeID {
			// Not our entry, someone else won the race
			em.mu.Lock()
			em.currentLeaderID = entryLeader
			em.leaseExpiry = entryExpiry
			em.mu.Unlock()
			return
		}

		// Successfully became leader
		em.mu.Lock()
		em.isLeader = true
		em.currentLeaderID = em.nodeID
		em.leaseExpiry = entryExpiry
		em.lastRevision = revision
		em.mu.Unlock()

		select {
		case em.leaderChan <- true:
		default:
		}
		return
	}

	// Entry exists, parse it
	currentLeader, _, expiry, ok := parseLeaderInfo(string(entry.Value()))
	if !ok {
		// Corrupted entry, try to take over
		em.tryUpdateLeader(kv, infoBytes, entry.Revision())
		return
	}

	// Update our view of current leader
	em.mu.Lock()
	em.currentLeaderID = currentLeader
	em.leaseExpiry = expiry
	wasLeader := em.isLeader
	em.mu.Unlock()

	// Check if we are already the leader
	if currentLeader == em.nodeID {
		// Try to renew our lease if it's expiring soon
		if expiry < now+em.leaseDuration.Milliseconds()/2 {
			em.renewLease()
		}
		return
	}

	// Someone else is the leader
	if wasLeader {
		// We lost leadership
		em.mu.Lock()
		em.isLeader = false
		em.mu.Unlock()

		select {
		case em.leaderChan <- false:
		default:
		}
	}

	// Check if current lease is expired
	if expiry < now {
		// Lease expired, try to take over
		em.tryUpdateLeader(kv, infoBytes, entry.Revision())
	}
}

// tryUpdateLeader attempts to update the leader entry
func (em *ElectionManager) tryUpdateLeader(kv nats.KeyValue, infoBytes []byte, lastRevision uint64) {
	revision, err := kv.Update(em.leaderKey, infoBytes, lastRevision)
	if err != nil {
		// Failed to update (race condition)
		return
	}

	// Verify by reading back
	entry, err := kv.Get(em.leaderKey)
	if err != nil {
		return
	}

	entryLeader, _, entryExpiry, ok := parseLeaderInfo(string(entry.Value()))
	if !ok || entryLeader != em.nodeID {
		// Not our entry after update
		em.mu.Lock()
		em.currentLeaderID = entryLeader
		em.leaseExpiry = entryExpiry
		em.mu.Unlock()
		return
	}

	// Successfully became leader
	em.mu.Lock()
	wasNotLeader := !em.isLeader
	em.isLeader = true
	em.currentLeaderID = em.nodeID
	em.leaseExpiry = entryExpiry
	em.lastRevision = revision
	em.mu.Unlock()

	if em.onNewLeader != nil {
		em.onNewLeader(em.nodeID)
	}

	if wasNotLeader {
		select {
		case em.leaderChan <- true:
		default:
		}
	}
}

// renewLease renews the leadership lease
func (em *ElectionManager) renewLease() {
	kv, err := em.js.KeyValue(fmt.Sprintf("PICOCLAW_ELECTION_%s", em.hid))
	if err != nil {
		return
	}

	em.mu.RLock()
	lastRevision := em.lastRevision
	em.mu.RUnlock()

	now := time.Now().UnixMilli()
	leaseExpiry := now + em.leaseDuration.Milliseconds()

	infoBytes := []byte(fmt.Sprintf("%s|%s|%d", em.nodeID, em.sid, leaseExpiry))

	// Use the last revision for optimistic update
	revision, err := kv.Update(em.leaderKey, infoBytes, lastRevision)
	if err != nil {
		// Lost leadership - revision mismatch or other error
		em.mu.Lock()
		em.isLeader = false
		em.lastRevision = 0
		em.mu.Unlock()

		select {
		case em.leaderChan <- false:
		default:
		}

		logger.WarnCF("swarm", "Failed to renew leadership lease", map[string]interface{}{
			"node_id": em.nodeID,
			"error":   err.Error(),
		})
		return
	}

	em.mu.Lock()
	em.leaseExpiry = leaseExpiry
	em.lastRevision = revision
	em.mu.Unlock()

	logger.DebugCF("swarm", "Renewed leadership lease", map[string]interface{}{
		"node_id": em.nodeID,
		"expires": time.UnixMilli(leaseExpiry).Format(time.RFC3339),
	})
}

// stepDown voluntarily gives up leadership
func (em *ElectionManager) stepDown() {
	kv, err := em.js.KeyValue(fmt.Sprintf("PICOCLAW_ELECTION_%s", em.hid))
	if err != nil {
		return
	}

	kv.Delete(em.leaderKey)

	em.mu.Lock()
	em.isLeader = false
	em.currentLeaderID = ""
	em.lastRevision = 0
	em.mu.Unlock()

	logger.InfoCF("swarm", "Stepped down from leadership", map[string]interface{}{
		"node_id": em.nodeID,
	})
}

// handleLeaseUpdate handles KV updates for leadership changes
func (em *ElectionManager) handleLeaseUpdate(msg *nats.Msg) {
	em.mu.RLock()
	isParticipant := em.isParticipant
	wasLeader := em.isLeader
	em.mu.RUnlock()

	if !isParticipant {
		return
	}

	// Check if this is a deletion or update
	if len(msg.Data) == 0 || msg.Header.Get("Operation") == "DEL" {
		em.mu.Lock()
		em.currentLeaderID = ""
		em.leaseExpiry = 0
		if !wasLeader {
			em.mu.Unlock()
			// Leader stepped down, try to acquire
			go em.attemptLeadership()
			return
		}
		em.mu.Unlock()
		return
	}

	// Parse the new leader info
	newLeader, _, expiry, ok := parseLeaderInfo(string(msg.Data))
	if !ok {
		return
	}

	// Update our view of current leader
	em.mu.Lock()
	oldLeaderID := em.currentLeaderID
	em.currentLeaderID = newLeader
	em.leaseExpiry = expiry

	// Check if we were leader but are no longer
	if wasLeader && newLeader != em.nodeID {
		em.isLeader = false
		em.mu.Unlock()

		select {
		case em.leaderChan <- false:
		default:
		}
		return
	}

	// Check if we thought we were leader but someone else is
	if em.isLeader && newLeader != em.nodeID {
		em.isLeader = false
		em.mu.Unlock()

		select {
		case em.leaderChan <- false:
		default:
		}
		return
	}

	em.mu.Unlock()

	// Notify about new leader if it changed
	if em.onNewLeader != nil && newLeader != em.nodeID && oldLeaderID != newLeader {
		em.onNewLeader(newLeader)
	}
}

// IsLeader returns true if this node is currently the leader
func (em *ElectionManager) IsLeader() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// If we are leader, return our ID
	if em.isLeader && time.Now().UnixMilli() <= em.leaseExpiry {
		return true
	}

	return false
}

// GetLeaderID returns the current leader's node ID
func (em *ElectionManager) GetLeaderID() string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// If we are leader, return our ID
	if em.isLeader && time.Now().UnixMilli() <= em.leaseExpiry {
		return em.nodeID
	}

	// Return the known leader ID if lease is valid
	if em.leaseExpiry > 0 && time.Now().UnixMilli() <= em.leaseExpiry {
		return em.currentLeaderID
	}

	return ""
}

// OnBecameLeader sets callback for when this node becomes leader
func (em *ElectionManager) OnBecameLeader(fn func()) {
	em.onBecameLeader = fn
}

// OnLostLeadership sets callback for when this node loses leadership
func (em *ElectionManager) OnLostLeadership(fn func()) {
	em.onLostLeadership = fn
}

// OnNewLeader sets callback for when any node becomes leader
func (em *ElectionManager) OnNewLeader(fn func(leaderID string)) {
	em.onNewLeader = fn
}

// parseLeaderInfo parses the leader info string
func parseLeaderInfo(s string) (nodeID, sid string, expiry int64, ok bool) {
	parts := splitN(s, 3, "|")
	if len(parts) != 3 {
		return "", "", 0, false
	}

	var exp int64
	_, err := fmt.Sscanf(parts[2], "%d", &exp)
	if err != nil {
		return "", "", 0, false
	}

	return parts[0], parts[1], exp, true
}

// splitN splits a string into at most n parts
func splitN(s string, n int, sep string) []string {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []string{s}
	}

	parts := make([]string, 0, n)
	start := 0
	sepLen := len(sep)

	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep, start)
		if idx == -1 {
			// Fewer than n parts, return what we have
			parts = append(parts, s[start:])
			return parts
		}
		parts = append(parts, s[start:idx])
		start = idx + sepLen
	}

	// Add the rest
	parts = append(parts, s[start:])
	return parts
}

// indexOf finds the index of sep in s starting from start
func indexOf(s, sep string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// RoleSwitcher handles dynamic role switching based on election results
type RoleSwitcher struct {
	electionMgr  *ElectionManager
	nodeInfo     *NodeInfo
	manager      *Manager
	originalRole NodeRole
	mu           sync.RWMutex
}

// NewRoleSwitcher creates a new role switcher
func NewRoleSwitcher(em *ElectionManager, nodeInfo *NodeInfo, manager *Manager) *RoleSwitcher {
	return &RoleSwitcher{
		electionMgr:  em,
		nodeInfo:     nodeInfo,
		manager:      manager,
		originalRole: nodeInfo.Role,
	}
}

// GetCurrentRole returns the current role
func (rs *RoleSwitcher) GetCurrentRole() NodeRole {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.nodeInfo.Role
}

// Start begins monitoring election results
func (rs *RoleSwitcher) Start() {
	rs.electionMgr.OnBecameLeader(rs.onBecameLeader)
	rs.electionMgr.OnLostLeadership(rs.onLostLeadership)
}

func (rs *RoleSwitcher) onBecameLeader() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Promote to coordinator if not already
	if rs.nodeInfo.Role != RoleCoordinator {
		logger.InfoCF("swarm", "Promoting to coordinator", map[string]interface{}{
			"node_id": rs.nodeInfo.ID,
			"from":    string(rs.nodeInfo.Role),
		})
		rs.nodeInfo.Metadata = map[string]string{
			"original_role": string(rs.nodeInfo.Role),
		}
		rs.nodeInfo.Role = RoleCoordinator
	}
}

func (rs *RoleSwitcher) onLostLeadership() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Demote back to original role
	originalRole := rs.nodeInfo.Metadata["original_role"]
	if originalRole != "" && originalRole != string(rs.nodeInfo.Role) {
		logger.InfoCF("swarm", "Demoting from coordinator", map[string]interface{}{
			"node_id": rs.nodeInfo.ID,
			"to":      originalRole,
		})
		rs.nodeInfo.Role = NodeRole(originalRole)
	}
}

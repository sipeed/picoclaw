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

const (
	// DefaultHeartbeatTimeout is how long before a node is considered failed
	DefaultHeartbeatTimeout = 60 * time.Second
	// DefaultProgressStallTimeout is how long without progress before a task is failed
	DefaultProgressStallTimeout = 2 * time.Minute
	// FailoverCheckInterval is how often to check for failures
	FailoverCheckInterval = 10 * time.Second
	// ClaimLockTTL is how long a claim lock is valid
	ClaimLockTTL = 30 * time.Second
)

// FailoverManager manages task failure detection and recovery
type FailoverManager struct {
	discovery       *Discovery
	lifecycle       *TaskLifecycleStore
	checkpointStore *CheckpointStore
	bridge          *NATSBridge
	nodeInfo        *NodeInfo
	js              nats.JetStreamContext

	// Configuration
	heartbeatTimeout    time.Duration
	progressStallTimeout time.Duration

	// State
	running      bool
	mu           sync.RWMutex
	claimedTasks map[string]*ClaimInfo
}

// ClaimInfo tracks claimed task information
type ClaimInfo struct {
	TaskID      string
	ClaimedBy   string
	ClaimedAt   time.Time
	ExpiresAt   time.Time
	Checkpoint  *TaskCheckpoint
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(
	discovery *Discovery,
	lifecycle *TaskLifecycleStore,
	checkpointStore *CheckpointStore,
	bridge *NATSBridge,
	nodeInfo *NodeInfo,
	js nats.JetStreamContext,
) *FailoverManager {
	return &FailoverManager{
		discovery:           discovery,
		lifecycle:           lifecycle,
		checkpointStore:     checkpointStore,
		bridge:              bridge,
		nodeInfo:            nodeInfo,
		js:                  js,
		heartbeatTimeout:    DefaultHeartbeatTimeout,
		progressStallTimeout: DefaultProgressStallTimeout,
		claimedTasks:        make(map[string]*ClaimInfo),
	}
}

// Start begins failure detection and recovery
func (fm *FailoverManager) Start(ctx context.Context) error {
	fm.mu.Lock()
	fm.running = true
	fm.mu.Unlock()

	logger.InfoC("swarm", "Failover manager starting")

	// Start failure detection loop
	go fm.detectFailuresLoop(ctx)

	// Start claim expiration cleanup
	go fm.cleanupExpiredClaims(ctx)

	return nil
}

// Stop stops the failover manager
func (fm *FailoverManager) Stop() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.running = false
}

// detectFailuresLoop continuously checks for node and task failures
func (fm *FailoverManager) detectFailuresLoop(ctx context.Context) {
	ticker := time.NewTicker(FailoverCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fm.DetectFailures(ctx)
		}
	}
}

// DetectFailures checks for node failures and stalled tasks
func (fm *FailoverManager) DetectFailures(ctx context.Context) {
	// Get all discovered nodes
	nodes := fm.discovery.GetAllNodes()

	now := time.Now()

	for _, node := range nodes {
		// Skip self and offline nodes
		if node.ID == fm.nodeInfo.ID || node.Status == StatusOffline {
			continue
		}

		// Check heartbeat timeout
		lastSeen := time.UnixMilli(node.LastSeen)
		if now.Sub(lastSeen) > fm.heartbeatTimeout {
			logger.WarnCF("swarm", "Node heartbeat timeout detected", map[string]interface{}{
				"node_id":    node.ID,
				"last_seen":  lastSeen.Format(time.RFC3339),
				"timeout":    fm.heartbeatTimeout,
			})
			fm.handleNodeFailure(ctx, node)
		}
	}

	// Check for stalled tasks
	fm.detectStalledTasks(ctx)
}

// detectStalledTasks looks for tasks that haven't made progress
func (fm *FailoverManager) detectStalledTasks(ctx context.Context) {
	// Get active tasks
	activeTasks, err := fm.lifecycle.GetActiveTasks(ctx)
	if err != nil {
		logger.WarnCF("swarm", "Failed to get active tasks for stall detection", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	now := time.Now()

	for _, task := range activeTasks {
		// Get task history to check last progress update
		history, err := fm.lifecycle.GetTaskHistory(ctx, task.ID)
		if err != nil {
			continue
		}

		// Find the most recent event
		if len(history) == 0 {
			continue
		}

		latestEvent := history[len(history)-1]
		eventTime := time.UnixMilli(latestEvent.Timestamp)

		// Check if task is running but stalled
		if task.Status == TaskRunning && now.Sub(eventTime) > fm.progressStallTimeout {
			logger.WarnCF("swarm", "Stalled task detected", map[string]interface{}{
				"task_id":     task.ID,
				"assigned_to": task.AssignedTo,
				"last_update": eventTime.Format(time.RFC3339),
			})

			// Attempt to claim and recover the task
			go fm.attemptTaskRecovery(ctx, task)
		}
	}
}

// handleNodeFailure processes a detected node failure
func (fm *FailoverManager) handleNodeFailure(ctx context.Context, failedNode *NodeInfo) {
	// Get tasks assigned to this node
	tasks, err := fm.lifecycle.GetTasksByNode(ctx, failedNode.ID)
	if err != nil {
		logger.WarnCF("swarm", "Failed to get tasks for failed node", map[string]interface{}{
			"node_id": failedNode.ID,
			"error":   err.Error(),
		})
		return
	}

	for _, task := range tasks {
		if task.Status == TaskRunning || task.Status == TaskAssigned {
			logger.WarnCF("swarm", "Attempting recovery of task from failed node", map[string]interface{}{
				"task_id":     task.ID,
				"failed_node": failedNode.ID,
			})
			go fm.attemptTaskRecovery(ctx, task)
		}
	}
}

// attemptTaskRecovery tries to claim and recover a failed task
func (fm *FailoverManager) attemptTaskRecovery(ctx context.Context, task *SwarmTask) {
	// Try to claim the task
	claimed, checkpoint, err := fm.ClaimTask(ctx, task.ID)
	if err != nil {
		logger.WarnCF("swarm", "Failed to claim task for recovery", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return
	}

	if !claimed {
		// Another node claimed it
		return
	}

	logger.InfoCF("swarm", "Successfully claimed task for recovery", map[string]interface{}{
		"task_id": task.ID,
	})

	// Emit recovered event
	if err := fm.lifecycle.SaveTaskStatus(task, TaskEventRetry, "Task claimed for failover recovery"); err != nil {
		logger.WarnCF("swarm", "Failed to save task status during recovery", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	}

	// Dispatch to worker for recovery
	// In a full implementation, this would dispatch to the worker with checkpoint data
	logger.InfoCF("swarm", "Task ready for recovery execution", map[string]interface{}{
		"task_id":      task.ID,
		"has_checkpoint": checkpoint != nil,
	})
}

// ClaimTask attempts to claim a failed task using distributed locking
func (fm *FailoverManager) ClaimTask(ctx context.Context, taskID string) (bool, *TaskCheckpoint, error) {
	// Try to acquire distributed lock using NATS KV
	lockKey := fmt.Sprintf("claim_%s", taskID)

	// Create KV bucket for claims if it doesn't exist
	bucket, err := fm.js.KeyValue("PICOCLAW_CLAIMS")
	if err != nil {
		bucket, err = fm.js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: "PICOCLAW_CLAIMS",
		})
		if err != nil {
			return false, nil, fmt.Errorf("failed to create claims bucket: %w", err)
		}
	}

	// Try to create the lock entry (atomic create)
	claimData := []byte(fmt.Sprintf(`{"claimed_by":"%s","claimed_at":%d,"expires_at":%d}`,
		fm.nodeInfo.ID,
		time.Now().UnixMilli(),
		time.Now().Add(ClaimLockTTL).UnixMilli(),
	))

	// Use Create to ensure we're the first to claim
	_, err = bucket.Create(lockKey, claimData)
	if err != nil {
		if err == nats.ErrKeyExists {
			// Already claimed by someone else
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to create claim lock: %w", err)
	}

	// Successfully claimed! Now get the checkpoint
	checkpoint, err := fm.checkpointStore.LoadCheckpoint(ctx, taskID)
	if err != nil {
		logger.WarnCF("swarm", "No checkpoint found for task", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		// Continue without checkpoint - will restart from beginning
	}

	// Track claim locally
	fm.mu.Lock()
	fm.claimedTasks[taskID] = &ClaimInfo{
		TaskID:     taskID,
		ClaimedBy:  fm.nodeInfo.ID,
		ClaimedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ClaimLockTTL),
		Checkpoint: checkpoint,
	}
	fm.mu.Unlock()

	return true, checkpoint, nil
}

// ReleaseClaim releases a claim on a task
func (fm *FailoverManager) ReleaseClaim(ctx context.Context, taskID string) error {
	lockKey := fmt.Sprintf("claim_%s", taskID)

	bucket, err := fm.js.KeyValue("PICOCLAW_CLAIMS")
	if err != nil {
		return fmt.Errorf("failed to get claims bucket: %w", err)
	}

	err = bucket.Delete(lockKey)
	if err != nil && err != nats.ErrKeyNotFound {
		return fmt.Errorf("failed to release claim: %w", err)
	}

	fm.mu.Lock()
	delete(fm.claimedTasks, taskID)
	fm.mu.Unlock()

	logger.DebugCF("swarm", "Released task claim", map[string]interface{}{
		"task_id": taskID,
	})

	return nil
}

// cleanupExpiredClaims removes expired claims and retries them
func (fm *FailoverManager) cleanupExpiredClaims(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fm.cleanupExpiredClaimsOnce(ctx)
		}
	}
}

func (fm *FailoverManager) cleanupExpiredClaimsOnce(ctx context.Context) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	now := time.Now()
	for taskID, claim := range fm.claimedTasks {
		if now.After(claim.ExpiresAt) {
			logger.WarnCF("swarm", "Claim expired, releasing", map[string]interface{}{
				"task_id": taskID,
			})
			delete(fm.claimedTasks, taskID)

			// Also remove from KV
			lockKey := fmt.Sprintf("claim_%s", taskID)
			if bucket, err := fm.js.KeyValue("PICOCLAW_CLAIMS"); err == nil {
				_ = bucket.Delete(lockKey)
			}
		}
	}
}

// RenewClaim renews a claim lock before it expires
func (fm *FailoverManager) RenewClaim(ctx context.Context, taskID string) error {
	lockKey := fmt.Sprintf("claim_%s", taskID)

	bucket, err := fm.js.KeyValue("PICOCLAW_CLAIMS")
	if err != nil {
		return fmt.Errorf("failed to get claims bucket: %w", err)
	}

	// Update the claim with new expiration
	claimData := []byte(fmt.Sprintf(`{"claimed_by":"%s","claimed_at":%d,"expires_at":%d}`,
		fm.nodeInfo.ID,
		time.Now().UnixMilli(),
		time.Now().Add(ClaimLockTTL).UnixMilli(),
	))

	_, err = bucket.Put(lockKey, claimData)
	if err != nil {
		return fmt.Errorf("failed to renew claim: %w", err)
	}

	// Update local tracking
	fm.mu.Lock()
	if claim, exists := fm.claimedTasks[taskID]; exists {
		claim.ExpiresAt = time.Now().Add(ClaimLockTTL)
	}
	fm.mu.Unlock()

	return nil
}

// GetClaimedTasks returns the list of tasks claimed by this node
func (fm *FailoverManager) GetClaimedTasks() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	tasks := make([]string, 0, len(fm.claimedTasks))
	for taskID := range fm.claimedTasks {
		tasks = append(tasks, taskID)
	}
	return tasks
}

// IsClaimedByThisNode checks if a task is claimed by this node
func (fm *FailoverManager) IsClaimedByThisNode(taskID string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	claim, exists := fm.claimedTasks[taskID]
	if !exists {
		return false
	}
	return claim.ClaimedBy == fm.nodeInfo.ID
}

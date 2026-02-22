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

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ContextPool manages shared context for swarm tasks
// It uses NATS JetStream KV store for distributed context sharing
type ContextPool struct {
	js      nats.JetStreamContext
	kv      nats.KeyValue
	bucket  string
	nodeID  string
	hid     string // Human ID (tenant/cluster identity)
	sid     string // Service ID (instance identity)
	mu      sync.RWMutex
	running bool
}

// ContextEntry represents a single context entry
type ContextEntry struct {
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Type      string                 `json:"type"` // "string", "number", "boolean", "object", "array"
	Timestamp int64                  `json:"timestamp"`
	NodeID    string                 `json:"node_id"`
	ExpiresAt int64                  `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskContext represents all context for a specific task
type TaskContext struct {
	TaskID      string                     `json:"task_id"`
	WorkflowID  string                     `json:"workflow_id,omitempty"`
	ParentTaskID string                    `json:"parent_task_id,omitempty"`
	Entries     map[string]*ContextEntry   `json:"entries"`
	CreatedAt   int64                      `json:"created_at"`
	UpdatedAt   int64                      `json:"updated_at"`
	CreatedBy   string                     `json:"created_by"`
	Permissions map[string]ContextPermission `json:"permissions,omitempty"` // H-id -> permission level
}

// ContextPermission defines access level for context
type ContextPermission string

const (
	PermRead  ContextPermission = "read"
	PermWrite ContextPermission = "write"
	PermAdmin ContextPermission = "admin"
)

const (
	// Default context bucket name
	contextBucketName = "PICOCLAW_CONTEXT"

	// Default TTL for context entries (24 hours)
	defaultContextTTL = 24 * time.Hour

	// Key prefix for task context
	taskContextPrefix = "task:"
)

// NewContextPool creates a new shared context pool
func NewContextPool(js nats.JetStreamContext, nodeID, hid, sid string) *ContextPool {
	return &ContextPool{
		js:     js,
		nodeID: nodeID,
		hid:    hid,
		sid:    sid,
		bucket: contextBucketName,
	}
}

// Start initializes the context pool KV store
func (cp *ContextPool) Start(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.running {
		return nil
	}

	// Create or get KV bucket for context storage
	kv, err := cp.js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:      cp.bucket,
		Description: "PicoClaw swarm shared context storage",
		TTL:         defaultContextTTL,
		MaxBytes:    100 * 1024 * 1024, // 100MB default
		Storage:     nats.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		// Try to get existing bucket
		kv, err = cp.js.KeyValue(cp.bucket)
		if err != nil {
			return fmt.Errorf("failed to create/get context KV store: %w", err)
		}
	}

	cp.kv = kv
	cp.running = true

	logger.InfoCF("swarm", "Context pool started", map[string]interface{}{
		"bucket":  cp.bucket,
		"node_id": cp.nodeID,
	})

	return nil
}

// Stop gracefully stops the context pool
func (cp *ContextPool) Stop() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.running = false
	return nil
}

// CreateTaskContext creates a new context for a task
func (cp *ContextPool) CreateTaskContext(taskID, workflowID, parentTaskID string) (*TaskContext, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.running {
		return nil, fmt.Errorf("context pool not running")
	}

	now := time.Now().UnixMilli()

	taskCtx := &TaskContext{
		TaskID:      taskID,
		WorkflowID:  workflowID,
		ParentTaskID: parentTaskID,
		Entries:     make(map[string]*ContextEntry),
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   cp.nodeID,
		Permissions: make(map[string]ContextPermission),
	}

	// Grant creator admin permissions
	taskCtx.Permissions[cp.hid] = PermAdmin

	// Save to KV store
	if err := cp.saveTaskContext(taskCtx); err != nil {
		return nil, err
	}

	logger.InfoCF("swarm", "Created task context", map[string]interface{}{
		"task_id":     taskID,
		"workflow_id": workflowID,
		"created_by":  cp.nodeID,
	})

	return taskCtx, nil
}

// GetTaskContext retrieves context for a task
func (cp *ContextPool) GetTaskContext(taskID string) (*TaskContext, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.running {
		return nil, fmt.Errorf("context pool not running")
	}

	key := cp.taskContextKey(taskID)
	entry, err := cp.kv.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			// Return empty context instead of error
			return &TaskContext{
				TaskID:      taskID,
				Entries:     make(map[string]*ContextEntry),
				CreatedAt:   time.Now().UnixMilli(),
				UpdatedAt:   time.Now().UnixMilli(),
				CreatedBy:   cp.nodeID,
				Permissions: make(map[string]ContextPermission),
			}, nil
		}
		return nil, fmt.Errorf("failed to get task context: %w", err)
	}

	var taskCtx TaskContext
	if err := json.Unmarshal(entry.Value(), &taskCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task context: %w", err)
	}

	return &taskCtx, nil
}

// SetEntry sets a context entry for a task
func (cp *ContextPool) SetEntry(taskID, key string, value interface{}) error {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.running {
		return fmt.Errorf("context pool not running")
	}

	// Get existing context
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Check write permission
	if perm, ok := taskCtx.Permissions[cp.hid]; !ok || perm == PermRead {
		return fmt.Errorf("no write permission for context %s", taskID)
	}

	// Determine value type
	var valueType string
	switch value.(type) {
	case string:
		valueType = "string"
	case int, int32, int64, float32, float64:
		valueType = "number"
	case bool:
		valueType = "boolean"
	case map[string]interface{}, []byte:
		valueType = "object"
	case []interface{}:
		valueType = "array"
	default:
		valueType = "unknown"
	}

	// Create/update entry
	taskCtx.Entries[key] = &ContextEntry{
		Key:       key,
		Value:     value,
		Type:      valueType,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    cp.nodeID,
	}
	taskCtx.UpdatedAt = time.Now().UnixMilli()

	// Save to KV store
	return cp.saveTaskContext(taskCtx)
}

// GetEntry gets a specific context entry for a task
func (cp *ContextPool) GetEntry(taskID, key string) (*ContextEntry, error) {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return nil, err
	}

	entry, ok := taskCtx.Entries[key]
	if !ok {
		return nil, fmt.Errorf("entry not found: %s", key)
	}

	return entry, nil
}

// GetAllEntries retrieves all entries for a task
func (cp *ContextPool) GetAllEntries(taskID string) (map[string]*ContextEntry, error) {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return nil, err
	}

	return taskCtx.Entries, nil
}

// DeleteEntry deletes a context entry
func (cp *ContextPool) DeleteEntry(taskID, key string) error {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Check write permission
	if perm, ok := taskCtx.Permissions[cp.hid]; !ok || perm == PermRead {
		return fmt.Errorf("no write permission for context %s", taskID)
	}

	delete(taskCtx.Entries, key)
	taskCtx.UpdatedAt = time.Now().UnixMilli()

	return cp.saveTaskContext(taskCtx)
}

// GrantPermission grants permission to another H-id
func (cp *ContextPool) GrantPermission(taskID, targetHID string, perm ContextPermission) error {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Only admin can grant permissions
	if existingPerm, ok := taskCtx.Permissions[cp.hid]; !ok || existingPerm != PermAdmin {
		return fmt.Errorf("no admin permission for context %s", taskID)
	}

	taskCtx.Permissions[targetHID] = perm
	taskCtx.UpdatedAt = time.Now().UnixMilli()

	return cp.saveTaskContext(taskCtx)
}

// RevokePermission revokes permission from an H-id
func (cp *ContextPool) RevokePermission(taskID, targetHID string) error {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Only admin can revoke permissions
	if existingPerm, ok := taskCtx.Permissions[cp.hid]; !ok || existingPerm != PermAdmin {
		return fmt.Errorf("no admin permission for context %s", taskID)
	}

	delete(taskCtx.Permissions, targetHID)
	taskCtx.UpdatedAt = time.Now().UnixMilli()

	return cp.saveTaskContext(taskCtx)
}

// MergeContext merges entries from parent task context
func (cp *ContextPool) MergeContext(taskID, parentTaskID string) error {
	parentCtx, err := cp.GetTaskContext(parentTaskID)
	if err != nil {
		return err
	}

	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Merge entries from parent
	for key, entry := range parentCtx.Entries {
		// Only add if not already present
		if _, exists := taskCtx.Entries[key]; !exists {
			// Copy entry
			copiedEntry := *entry
			taskCtx.Entries[key] = &copiedEntry
		}
	}

	// Inherit permissions from parent
	for hid, perm := range parentCtx.Permissions {
		if _, exists := taskCtx.Permissions[hid]; !exists {
			taskCtx.Permissions[hid] = perm
		}
	}

	taskCtx.ParentTaskID = parentTaskID
	taskCtx.UpdatedAt = time.Now().UnixMilli()

	return cp.saveTaskContext(taskCtx)
}

// DeleteTaskContext removes context for a task
func (cp *ContextPool) DeleteTaskContext(taskID string) error {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.running {
		return fmt.Errorf("context pool not running")
	}

	key := cp.taskContextKey(taskID)
	return cp.kv.Delete(key)
}

// ListTaskContexts lists all task contexts (with optional filtering)
func (cp *ContextPool) ListTaskContexts(filter string) ([]*TaskContext, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.running {
		return nil, fmt.Errorf("context pool not running")
	}

	watcher, err := cp.kv.WatchAll()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()

	var contexts []*TaskContext

	for entry := range watcher.Updates() {
		if entry == nil {
			break
		}

		// Filter by task prefix
		if len(entry.Key()) <= len(taskContextPrefix) || entry.Key()[:len(taskContextPrefix)+1] != taskContextPrefix {
			continue
		}

		// Apply additional filter if provided
		if filter != "" && entry.Key() != cp.taskContextKey(filter) && entry.Key() != taskContextPrefix+filter {
			continue
		}

		var taskCtx TaskContext
		if err := json.Unmarshal(entry.Value(), &taskCtx); err != nil {
			continue
		}

		contexts = append(contexts, &taskCtx)
	}

	return contexts, nil
}

// SetEntryWithTTL sets a context entry with TTL
func (cp *ContextPool) SetEntryWithTTL(taskID, key string, value interface{}, ttl time.Duration) error {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return err
	}

	// Check write permission
	if perm, ok := taskCtx.Permissions[cp.hid]; !ok || perm == PermRead {
		return fmt.Errorf("no write permission for context %s", taskID)
	}

	// Determine value type
	var valueType string
	switch value.(type) {
	case string:
		valueType = "string"
	case int, int32, int64, float32, float64:
		valueType = "number"
	case bool:
		valueType = "boolean"
	case map[string]interface{}, []byte:
		valueType = "object"
	case []interface{}:
		valueType = "array"
	default:
		valueType = "unknown"
	}

	now := time.Now().UnixMilli()

	// Create/update entry with TTL
	taskCtx.Entries[key] = &ContextEntry{
		Key:       key,
		Value:     value,
		Type:      valueType,
		Timestamp: now,
		NodeID:    cp.nodeID,
		ExpiresAt: now + ttl.Milliseconds(),
	}
	taskCtx.UpdatedAt = now

	return cp.saveTaskContext(taskCtx)
}

// CleanExpiredEntries removes expired entries from a task context
func (cp *ContextPool) CleanExpiredEntries(taskID string) (int, error) {
	taskCtx, err := cp.GetTaskContext(taskID)
	if err != nil {
		return 0, err
	}

	now := time.Now().UnixMilli()
	removed := 0

	for key, entry := range taskCtx.Entries {
		if entry.ExpiresAt > 0 && entry.ExpiresAt < now {
			delete(taskCtx.Entries, key)
			removed++
		}
	}

	if removed > 0 {
		taskCtx.UpdatedAt = now
		if err := cp.saveTaskContext(taskCtx); err != nil {
			return 0, err
		}
	}

	return removed, nil
}

// saveTaskContext saves task context to KV store
func (cp *ContextPool) saveTaskContext(taskCtx *TaskContext) error {
	data, err := json.Marshal(taskCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal task context: %w", err)
	}

	key := cp.taskContextKey(taskCtx.TaskID)

	// Use Update with CreateIfMissing for safety
	_, err = cp.kv.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	return nil
}

// taskContextKey returns the KV store key for a task context
func (cp *ContextPool) taskContextKey(taskID string) string {
	return taskContextPrefix + taskID
}

// GetContextForPrompt returns a formatted string of context entries for use in prompts
func (cp *ContextPool) GetContextForPrompt(taskID string) (string, error) {
	entries, err := cp.GetAllEntries(taskID)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", nil
	}

	var result string
	result = fmt.Sprintf("[Shared Context for task %s]\n", taskID)

	for key, entry := range entries {
		// Skip expired entries
		if entry.ExpiresAt > 0 && entry.ExpiresAt < time.Now().UnixMilli() {
			continue
		}

		result += fmt.Sprintf("- %s: ", key)
		switch v := entry.Value.(type) {
		case string:
			result += fmt.Sprintf("%q", v)
		case map[string]interface{}:
			jsonBytes, _ := json.Marshal(v)
			result += string(jsonBytes)
		case []interface{}:
			jsonBytes, _ := json.Marshal(v)
			result += string(jsonBytes)
		default:
			result += fmt.Sprintf("%v", v)
		}
		result += "\n"
	}

	return result, nil
}

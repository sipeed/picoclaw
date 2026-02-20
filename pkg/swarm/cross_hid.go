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

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/relation"
)

// CrossHIDBridge manages cross H-id communication with authorization
type CrossHIDBridge struct {
	bridge    *NATSBridge
	localHID  string
	authorizer *relation.Authorizer
	mu        sync.RWMutex

	// Exported H-ids (H-ids we allow to communicate with us)
	exported map[string]bool

	// Imported H-ids (H-ids we allow ourselves to communicate with)
	imported map[string]bool

	// Subscription handlers
	handlers map[string]func(*CrossHIDMessage)
}

// CrossHIDMessage represents a message sent between H-ids
type CrossHIDMessage struct {
	// FromHID is the sender's H-id
	FromHID string `json:"from_hid"`

	// FromSID is the sender's S-id
	FromSID string `json:"from_sid"`

	// ToHID is the recipient's H-id
	ToHID string `json:"to_hid"`

	// Type is the message type
	Type string `json:"type"`

	// Payload is the message payload
	Payload map[string]interface{} `json:"payload"`

	// Timestamp is when the message was sent
	Timestamp int64 `json:"timestamp"`

	// ID is a unique message identifier
	ID string `json:"id"`
}

// NewCrossHIDBridge creates a new cross H-id communication bridge
func NewCrossHIDBridge(bridge *NATSBridge, localHID string, authorizer *relation.Authorizer) *CrossHIDBridge {
	return &CrossHIDBridge{
		bridge:     bridge,
		localHID:   localHID,
		authorizer: authorizer,
		exported:   make(map[string]bool),
		imported:   make(map[string]bool),
		handlers:   make(map[string]func(*CrossHIDMessage)),
	}
}

// Start initializes the cross H-id bridge
func (b *CrossHIDBridge) Start(ctx context.Context) error {
	// Subscribe to cross-domain messages for our H-id
	subject := fmt.Sprintf("picoclaw.x.*.%s.>", b.localHID)

	sub, err := b.bridge.conn.Subscribe(subject, b.handleIncomingMessage)
	if err != nil {
		return fmt.Errorf("failed to subscribe to cross H-id messages: %w", err)
	}

	logger.InfoCF("swarm", "Cross H-id bridge started", map[string]interface{}{
		"subject": subject,
	})

	// Keep subscription alive
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	return nil
}

// Export allows another H-id to send messages to us
func (b *CrossHIDBridge) Export(hid string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.exported[hid] = true

	logger.InfoCF("swarm", "Exported H-id", map[string]interface{}{
		"hid": hid,
	})

	return nil
}

// Import allows us to send messages to another H-id
func (b *CrossHIDBridge) Import(hid string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.imported[hid] = true

	logger.InfoCF("swarm", "Imported H-id", map[string]interface{}{
		"hid": hid,
	})

	return nil
}

// Revoke removes an export
func (b *CrossHIDBridge) RevokeExport(hid string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.exported, hid)
}

// RevokeImport removes an import
func (b *CrossHIDBridge) RevokeImport(hid string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.imported, hid)
}

// Send sends a message to another H-id
func (b *CrossHIDBridge) Send(ctx context.Context, toHID, messageType string, payload map[string]interface{}) error {
	b.mu.RLock()
	_, isImported := b.imported[toHID]
	b.mu.RUnlock()

	// Check if we're allowed to send to this H-id
	if !isImported {
		return fmt.Errorf("H-id %s is not imported", toHID)
	}

	msg := &CrossHIDMessage{
		FromHID:  b.localHID,
		ToHID:    toHID,
		Type:     messageType,
		Payload:  payload,
		Timestamp: currentTimeMillis(),
		ID:       generateMessageID(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	subject := fmt.Sprintf("picoclaw.x.%s.%s.%s", b.localHID, toHID, messageType)

	if err := b.bridge.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logger.DebugCF("swarm", "Sent cross H-id message", map[string]interface{}{
		"to_hid": toHID,
		"type":   messageType,
		"msg_id": msg.ID,
	})

	return nil
}

// SendWithAuth sends a message with authorization check
func (b *CrossHIDBridge) SendWithAuth(ctx context.Context, fromSID, toHID, messageType string, payload map[string]interface{}) error {
	// Check authorization using relation system
	resource := relation.NewResourceID(relation.ResourceNode, toHID)

	authzReq := &relation.AuthzRequest{
		SubjectHID: b.localHID,
		SubjectSID: fromSID,
		Action:     relation.ActionRead, // Use read as "communicate with"
		Resource:   resource,
	}

	result := b.authorizer.Authorize(authzReq)
	if !result.Allowed {
		return fmt.Errorf("authorization denied: %s", result.Reason)
	}

	return b.Send(ctx, toHID, messageType, payload)
}

// RegisterHandler registers a handler for a specific message type
func (b *CrossHIDBridge) RegisterHandler(messageType string, handler func(*CrossHIDMessage)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[messageType] = handler
}

// UnregisterHandler removes a handler
func (b *CrossHIDBridge) UnregisterHandler(messageType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.handlers, messageType)
}

// handleIncomingMessage handles incoming cross H-id messages
func (b *CrossHIDBridge) handleIncomingMessage(msg *nats.Msg) {
	var message CrossHIDMessage
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.WarnCF("swarm", "Failed to unmarshal cross H-id message", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Verify the message is for us
	if message.ToHID != b.localHID {
		logger.DebugCF("swarm", "Ignoring cross H-id message for different H-id", map[string]interface{}{
			"to_hid":    message.ToHID,
			"local_hid": b.localHID,
		})
		return
	}

	b.mu.RLock()
	isExported := b.exported[message.FromHID]
	handler, hasHandler := b.handlers[message.Type]
	b.mu.RUnlock()

	// Check if the sender is exported (allowed to send to us)
	if !isExported {
		logger.WarnCF("swarm", "Rejected cross H-id message from non-exported H-id", map[string]interface{}{
			"from_hid": message.FromHID,
		})
		return
	}

	logger.DebugCF("swarm", "Received cross H-id message", map[string]interface{}{
		"from_hid": message.FromHID,
		"type":     message.Type,
		"msg_id":   message.ID,
	})

	// Call handler if registered
	if hasHandler && handler != nil {
		handler(&message)
	}
}

// GetExported returns all exported H-ids
func (b *CrossHIDBridge) GetExported() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hids := make([]string, 0, len(b.exported))
	for hid := range b.exported {
		hids = append(hids, hid)
	}
	return hids
}

// GetImported returns all imported H-ids
func (b *CrossHIDBridge) GetImported() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hids := make([]string, 0, len(b.imported))
	for hid := range b.imported {
		hids = append(hids, hid)
	}
	return hids
}

// IsExported checks if an H-id is exported
func (b *CrossHIDBridge) IsExported(hid string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.exported[hid]
}

// IsImported checks if an H-id is imported
func (b *CrossHIDBridge) IsImported(hid string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.imported[hid]
}

// currentTimeMillis returns the current time in milliseconds
func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("xmsg-%s", uuid.New().String()[:8])
}

// CrossHIDConfig contains configuration for cross H-id communication
type CrossHIDConfig struct {
	// DefaultExportPolicy determines the default export policy
	DefaultExportPolicy string // "allow", "deny", "auth"

	// DefaultImportPolicy determines the default import policy
	DefaultImportPolicy string // "allow", "deny", "auth"

	// ExportedHIDs is a list of H-ids to export to
	ExportedHIDs []string

	// ImportedHIDs is a list of H-ids to import from
	ImportedHIDs []string
}

// ApplyConfig applies a configuration to the bridge
func (b *CrossHIDBridge) ApplyConfig(cfg *CrossHIDConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Clear existing
	b.exported = make(map[string]bool)
	b.imported = make(map[string]bool)

	// Apply default policy
	if cfg.DefaultExportPolicy == "allow" {
		// Wildcard - all H-ids allowed (use with caution)
		b.exported["*"] = true
	}

	if cfg.DefaultImportPolicy == "allow" {
		b.imported["*"] = true
	}

	// Apply explicit exports
	for _, hid := range cfg.ExportedHIDs {
		b.exported[hid] = true
	}

	// Apply explicit imports
	for _, hid := range cfg.ImportedHIDs {
		b.imported[hid] = true
	}

	return nil
}

// Message types for cross H-id communication
const (
	// MessageTypeTaskRequest is for requesting a task across H-ids
	MessageTypeTaskRequest = "task.request"

	// MessageTypeTaskResponse is for responding to a task
	MessageTypeTaskResponse = "task.response"

	// MessageTypeMemoryQuery is for querying memory across H-ids
	MessageTypeMemoryQuery = "memory.query"

	// MessageTypeMemoryResponse is for responding to memory queries
	MessageTypeMemoryResponse = "memory.response"

	// MessageTypeDiscovery is for discovering nodes across H-ids
	MessageTypeDiscovery = "discovery"

	// MessageTypeHeartbeat is for heartbeat across H-ids
	MessageTypeHeartbeat = "heartbeat"
)

// TaskRequestPayload is the payload for a task request
type TaskRequestPayload struct {
	TaskID    string                 `json:"task_id"`
	Prompt    string                 `json:"prompt"`
	Context   map[string]interface{} `json:"context"`
	Timeout   int64                  `json:"timeout"`
}

// TaskResponsePayload is the payload for a task response
type TaskResponsePayload struct {
	TaskID    string `json:"task_id"`
	Result    string `json:"result"`
	Error     string `json:"error,omitempty"`
	Completed bool   `json:"completed"`
}

// SendTaskRequest sends a task request to another H-id
func (b *CrossHIDBridge) SendTaskRequest(ctx context.Context, toHID string, payload *TaskRequestPayload) error {
	msgPayload := map[string]interface{}{
		"task_id": payload.TaskID,
		"prompt":   payload.Prompt,
		"context":  payload.Context,
		"timeout":  payload.Timeout,
	}
	return b.Send(ctx, toHID, MessageTypeTaskRequest, msgPayload)
}

// SendTaskResponse sends a task response to another H-id
func (b *CrossHIDBridge) SendTaskResponse(ctx context.Context, toHID string, payload *TaskResponsePayload) error {
	msgPayload := map[string]interface{}{
		"task_id":   payload.TaskID,
		"result":    payload.Result,
		"error":     payload.Error,
		"completed": payload.Completed,
	}
	return b.Send(ctx, toHID, MessageTypeTaskResponse, msgPayload)
}

// NewTaskRequest creates a new task request payload
func NewTaskRequest(taskID, prompt string) *TaskRequestPayload {
	return &TaskRequestPayload{
		TaskID:  taskID,
		Prompt:  prompt,
		Context: make(map[string]interface{}),
		Timeout: 600000, // 10 minutes default
	}
}

// NewTaskResponse creates a new task response payload
func NewTaskResponse(taskID, result string, completed bool) *TaskResponsePayload {
	return &TaskResponsePayload{
		TaskID:    taskID,
		Result:    result,
		Completed: completed,
	}
}

// NewTaskResponseError creates a task response with an error
func NewTaskResponseError(taskID, errMsg string) *TaskResponsePayload {
	return &TaskResponsePayload{
		TaskID:    taskID,
		Error:     errMsg,
		Completed: false,
	}
}

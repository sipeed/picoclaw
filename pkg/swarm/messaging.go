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

// MessagingAPI provides a high-level API for inter-shrimp communication
// It handles both same-H-id (within tenant) and cross-H-id (cross-tenant) messaging
type MessagingAPI struct {
	bridge    *NATSBridge
	localHID  string
	localSID  string
	nodeID    string
	mu        sync.RWMutex

	// Message handlers by type
	handlers map[string][]func(*InterShrimpMessage)
}

// InterShrimpMessage represents a message between shrimp instances
type InterShrimpMessage struct {
	// From identifies the sender
	FromHID string `json:"from_hid"`
	FromSID string `json:"from_sid"`
	FromNodeID string `json:"from_node_id"`

	// To identifies the recipient (empty for broadcast within same H-id)
	ToHID    string `json:"to_hid,omitempty"`
	ToSID    string `json:"to_sid,omitempty"`
	ToNodeID string `json:"to_node_id,omitempty"`

	// Type is the message type for routing
	Type string `json:"type"`

	// Payload contains the message data
	Payload map[string]interface{} `json:"payload"`

	// Timestamp when message was sent
	Timestamp int64 `json:"timestamp"`

	// ID uniquely identifies this message
	ID string `json:"id"`

	// InResponseTo links this message to a previous message
	InResponseTo string `json:"in_response_to,omitempty"`
}

// MessagingConfig configures the messaging API
type MessagingConfig struct {
	// AllowCrossHID enables cross-H-id communication
	AllowCrossHID bool

	// RequireAuth requires authorization for cross-H-id messages
	RequireAuth bool

	// AllowedHIDs lists H-ids allowed to communicate with us
	AllowedHIDs []string
}

// NewMessagingAPI creates a new messaging API
func NewMessagingAPI(bridge *NATSBridge, hid, sid, nodeID string) *MessagingAPI {
	api := &MessagingAPI{
		bridge:   bridge,
		localHID: hid,
		localSID: sid,
		nodeID:   nodeID,
		handlers: make(map[string][]func(*InterShrimpMessage)),
	}

	// Subscribe to messages for this node
	go api.subscribeToMessages()

	return api
}

// Subscribe registers a handler for a specific message type
func (m *MessagingAPI) Subscribe(messageType string, handler func(*InterShrimpMessage)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers[messageType] == nil {
		m.handlers[messageType] = make([]func(*InterShrimpMessage), 0)
	}
	m.handlers[messageType] = append(m.handlers[messageType], handler)

	logger.DebugCF("swarm", "Registered message handler", map[string]interface{}{
		"type": messageType,
	})
}

// Unsubscribe removes a handler (not implemented - handlers persist for session)
func (m *MessagingAPI) Unsubscribe(messageType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.handlers, messageType)
}

// SendBroadcast sends a message to all nodes in the same H-id
func (m *MessagingAPI) SendBroadcast(ctx context.Context, messageType string, payload map[string]interface{}) error {
	msg := &InterShrimpMessage{
		FromHID:    m.localHID,
		FromSID:    m.localSID,
		FromNodeID: m.nodeID,
		Type:       messageType,
		Payload:    payload,
		Timestamp:  time.Now().UnixMilli(),
		ID:         generateMessageID(),
	}

	// For same-H-id broadcast, use a special subject
	subject := fmt.Sprintf("picoclaw.msg.%s.*.%s", m.localHID, messageType)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := m.bridge.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish broadcast: %w", err)
	}

	logger.DebugCF("swarm", "Sent broadcast message", map[string]interface{}{
		"type":   messageType,
		"msg_id": msg.ID,
	})

	return nil
}

// SendToNode sends a message to a specific node (same or different H-id)
func (m *MessagingAPI) SendToNode(ctx context.Context, targetHID, targetSID, targetNodeID, messageType string, payload map[string]interface{}) error {
	msg := &InterShrimpMessage{
		FromHID:    m.localHID,
		FromSID:    m.localSID,
		FromNodeID: m.nodeID,
		ToHID:      targetHID,
		ToSID:      targetSID,
		ToNodeID:   targetNodeID,
		Type:       messageType,
		Payload:    payload,
		Timestamp:  time.Now().UnixMilli(),
		ID:         generateMessageID(),
	}

	var subject string
	if targetNodeID != "" {
		// Direct to node
		subject = fmt.Sprintf("picoclaw.msg.%s.%s.node.%s.%s", targetHID, targetSID, targetNodeID, messageType)
	} else if targetSID != "" {
		// To any node with this S-id
		subject = fmt.Sprintf("picoclaw.msg.%s.%s.*.%s", targetHID, targetSID, messageType)
	} else {
		// To any node in this H-id
		subject = fmt.Sprintf("picoclaw.msg.%s.*.%s", targetHID, messageType)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := m.bridge.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logger.DebugCF("swarm", "Sent direct message", map[string]interface{}{
		"target":  fmt.Sprintf("%s/%s/%s", targetHID, targetSID, targetNodeID),
		"type":    messageType,
		"msg_id":  msg.ID,
	})

	return nil
}

// SendReply sends a reply to a previous message
func (m *MessagingAPI) SendReply(ctx context.Context, originalMsg *InterShrimpMessage, payload map[string]interface{}) error {
	msg := &InterShrimpMessage{
		FromHID:     m.localHID,
		FromSID:     m.localSID,
		FromNodeID:  m.nodeID,
		ToHID:       originalMsg.FromHID,
		ToSID:       originalMsg.FromSID,
		ToNodeID:    originalMsg.FromNodeID,
		Type:        originalMsg.Type + ".reply",
		Payload:     payload,
		Timestamp:   time.Now().UnixMilli(),
		ID:          generateMessageID(),
		InResponseTo: originalMsg.ID,
	}

	subject := fmt.Sprintf("picoclaw.msg.%s.%s.node.%s.%s",
		msg.ToHID, msg.ToSID, msg.ToNodeID, msg.Type)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := m.bridge.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish reply: %w", err)
	}

	return nil
}

// Request sends a message and waits for a response
func (m *MessagingAPI) Request(ctx context.Context, targetHID, targetSID, targetNodeID, messageType string, payload map[string]interface{}, timeout time.Duration) (*InterShrimpMessage, error) {
	// Create inbox for response
	inbox := m.bridge.conn.NewRespInbox()
	responseCh := make(chan *InterShrimpMessage, 1)

	// Subscribe to responses
	sub, err := m.bridge.conn.Subscribe(inbox, func(msg *nats.Msg) {
		var response InterShrimpMessage
		if err := json.Unmarshal(msg.Data, &response); err != nil {
			return
		}
		select {
		case responseCh <- &response:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %w", err)
	}
	defer sub.Unsubscribe()

	// Send request with reply-to inbox
	requestMsg := &InterShrimpMessage{
		FromHID:    m.localHID,
		FromSID:    m.localSID,
		FromNodeID: m.nodeID,
		ToHID:      targetHID,
		ToSID:      targetSID,
		ToNodeID:   targetNodeID,
		Type:       messageType,
		Payload:    payload,
		Timestamp:  time.Now().UnixMilli(),
		ID:         generateMessageID(),
	}

	subject := fmt.Sprintf("picoclaw.msg.%s.%s.node.%s.%s", targetHID, targetSID, targetNodeID, messageType)

	data, err := json.Marshal(requestMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := m.bridge.conn.PublishRequest(subject, inbox, data); err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	// Wait for response
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case response := <-responseCh:
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("request timeout")
	}
}

// subscribeToMessages subscribes to messages for this node
func (m *MessagingAPI) subscribeToMessages() {
	// Subscribe to messages for this specific node
	nodeSubject := fmt.Sprintf("picoclaw.msg.%s.%s.node.%s.*", m.localHID, m.localSID, m.nodeID)
	m.bridge.conn.Subscribe(nodeSubject, func(msg *nats.Msg) {
		var message InterShrimpMessage
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			logger.WarnCF("swarm", "Failed to unmarshal message", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		m.dispatchMessage(&message)
	})

	// Subscribe to broadcast messages for our H-id
	broadcastSubject := fmt.Sprintf("picoclaw.msg.%s.*.*", m.localHID)
	m.bridge.conn.Subscribe(broadcastSubject, func(msg *nats.Msg) {
		var message InterShrimpMessage
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			return
		}
		// Skip messages from ourselves
		if message.FromNodeID == m.nodeID {
			return
		}
		m.dispatchMessage(&message)
	})

	logger.InfoCF("swarm", "Messaging API subscribed", map[string]interface{}{
		"hid":     m.localHID,
		"sid":     m.localSID,
		"node_id": m.nodeID,
	})
}

// dispatchMessage dispatches a message to registered handlers
func (m *MessagingAPI) dispatchMessage(msg *InterShrimpMessage) {
	m.mu.RLock()
	handlers := m.handlers[msg.Type]
	m.mu.RUnlock()

	logger.DebugCF("swarm", "Received inter-shrimp message", map[string]interface{}{
		"from":     fmt.Sprintf("%s/%s", msg.FromHID, msg.FromSID),
		"type":     msg.Type,
		"handlers": len(handlers),
	})

	for _, handler := range handlers {
		go func(h func(*InterShrimpMessage)) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorCF("swarm", "Message handler panic", map[string]interface{}{
						"error": fmt.Sprintf("%v", r),
					})
				}
			}()
			h(msg)
		}(handler)
	}
}

// Standard message types for MessagingAPI
const (
	// MessageTypeTask is for task-related messages
	MessageTypeTask = "task"

	// MessageTypeStatus is for status updates
	MessageTypeStatus = "status"

	// MessageTypeContext is for shared context updates
	MessageTypeContext = "context"

	// MessageTypeSync is for synchronization
	MessageTypeSync = "sync"
)

// TaskMessage is a task-related message payload
type TaskMessage struct {
	TaskID   string                 `json:"task_id"`
	Status   string                 `json:"status"`
	Result   string                 `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StatusMessage is a status update payload
type StatusMessage struct {
	Load         float64 `json:"load"`
	TasksRunning int     `json:"tasks_running"`
	Status       string  `json:"status"`
}

// ContextMessage is a context update payload
type ContextMessage struct {
	TaskID  string `json:"task_id"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Action  string `json:"action"` // set, delete, merge
}

// BroadcastTaskStatus broadcasts a task status update
func (m *MessagingAPI) BroadcastTaskStatus(ctx context.Context, taskID, status string, result string, resultErr error) error {
	payload := map[string]interface{}{
		"task_id": taskID,
		"status":  status,
	}
	if result != "" {
		payload["result"] = result
	}
	if resultErr != nil {
		payload["error"] = resultErr.Error()
	}
	return m.SendBroadcast(ctx, MessageTypeTask, payload)
}

// BroadcastStatus broadcasts this node's status
func (m *MessagingAPI) BroadcastStatus(ctx context.Context, load float64, tasksRunning int, status string) error {
	payload := map[string]interface{}{
		"load":          load,
		"tasks_running": tasksRunning,
		"status":        status,
	}
	return m.SendBroadcast(ctx, MessageTypeStatus, payload)
}

// PublishContextUpdate publishes a context update to the swarm
func (m *MessagingAPI) PublishContextUpdate(ctx context.Context, taskID, key, value, action string) error {
	payload := map[string]interface{}{
		"task_id": taskID,
		"key":     key,
		"value":   value,
		"action":  action,
	}
	return m.SendBroadcast(ctx, MessageTypeContext, payload)
}

// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

// SessionTransfer handles session migration between nodes.
type SessionTransfer struct {
	config     RPCConfig
	localNode  *NodeInfo
	transfers  map[string]*TransferOperation // session_key -> operation
	mu         sync.RWMutex
	conn       *net.UDPConn
	onReceive  func(*TransferPayload)
}

// TransferOperation represents an ongoing transfer operation.
type TransferOperation struct {
	SessionKey   string
	SourceNodeID string
	TargetNodeID string
	State        TransferState
	StartTime    int64
	LastUpdate   int64
	Payload      *TransferPayload
}

// TransferState represents the state of a transfer.
type TransferState string

const (
	TransferStatePending   TransferState = "pending"
	TransferStateSending   TransferState = "sending"
	TransferStateReceived  TransferState = "received"
	TransferStateCompleted TransferState = "completed"
	TransferStateFailed    TransferState = "failed"
)

// TransferPayload represents the session data being transferred.
type TransferPayload struct {
	SessionKey      string            `json:"session_key"`
	SourceNodeID    string            `json:"source_node_id"`
	TargetNodeID    string            `json:"target_node_id"`
	Messages        []SessionMessage  `json:"messages"`
	Summary         string            `json:"summary,omitempty"`
	Context         map[string]any    `json:"context,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Timestamp       int64             `json:"timestamp"`
	TransferID      string            `json:"transfer_id"`
}

// NewSessionTransfer creates a new session transfer handler.
func NewSessionTransfer(localNode *NodeInfo, config RPCConfig) (*SessionTransfer, error) {
	st := &SessionTransfer{
		config:    config,
		localNode: localNode,
		transfers: make(map[string]*TransferOperation),
	}

	// Bind UDP socket
	addr := fmt.Sprintf("%s:%d", localNode.Addr, config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RPC address: %w", err)
	}

	st.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for session transfer: %w", err)
	}

	// Start message handler
	go st.messageHandler()

	return st, nil
}

// Close closes the session transfer handler.
func (st *SessionTransfer) Close() error {
	if st.conn != nil {
		return st.conn.Close()
	}
	return nil
}

// TransferSession transfers a session to another node.
func (st *SessionTransfer) TransferSession(ctx context.Context, targetNode *NodeInfo, payload *TransferPayload) error {
	if payload.TransferID == "" {
		payload.TransferID = fmt.Sprintf("%s-%d", payload.SessionKey, payload.Timestamp)
	}

	payload.SourceNodeID = st.localNode.ID
	payload.TargetNodeID = targetNode.ID
	payload.Timestamp = payload.Timestamp

	// Create transfer operation
	op := &TransferOperation{
		SessionKey:   payload.SessionKey,
		SourceNodeID: st.localNode.ID,
		TargetNodeID: targetNode.ID,
		State:        TransferStateSending,
		StartTime:    payload.Timestamp,
		LastUpdate:   payload.Timestamp,
		Payload:      payload,
	}

	st.mu.Lock()
	st.transfers[payload.SessionKey] = op
	st.mu.Unlock()

	// Send transfer message
	if err := st.sendTransfer(targetNode, payload); err != nil {
		st.mu.Lock()
		op.State = TransferStateFailed
		delete(st.transfers, payload.SessionKey)
		st.mu.Unlock()
		return err
	}

	return nil
}

// sendTransfer sends a transfer message to a target node.
func (st *SessionTransfer) sendTransfer(targetNode *NodeInfo, payload *TransferPayload) error {
	msg := map[string]any{
		"type":    "session_transfer",
		"payload": payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", targetNode.Addr, st.config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	// st.conn.SetWriteDeadline(nil) // Not setting deadline
	_, err = st.conn.WriteToUDP(data, udpAddr)
	return err
}

// SendAck sends an acknowledgment for a received transfer.
func (st *SessionTransfer) SendAck(targetNode *NodeInfo, transferID string, accepted bool) error {
	msg := map[string]any{
		"type": "session_transfer_ack",
		"payload": map[string]any{
			"transfer_id": transferID,
			"accepted":    accepted,
			"node_id":     st.localNode.ID,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", targetNode.Addr, st.config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	// st.conn.SetWriteDeadline(nil) // Not setting deadline
	_, err = st.conn.WriteToUDP(data, udpAddr)
	return err
}

// messageHandler handles incoming transfer messages.
func (st *SessionTransfer) messageHandler() {
	buf := make([]byte, MaxSessionMessageSize)

	for {
		n, addr, err := st.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n > 0 {
			go st.handleMessage(buf[:n], addr)
		}
	}
}

// handleMessage handles an incoming message.
func (st *SessionTransfer) handleMessage(data []byte, addr *net.UDPAddr) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	msgType, _ := msg["type"].(string)

	switch msgType {
	case "session_transfer":
		st.handleTransfer(data, addr)
	case "session_transfer_ack":
		st.handleTransferAck(data)
	}
}

// handleTransfer handles a session transfer message.
func (st *SessionTransfer) handleTransfer(data []byte, addr *net.UDPAddr) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	payloadData, _ := json.Marshal(msg["payload"])
	var payload TransferPayload
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return
	}

	// Check if this transfer is for us
	if payload.TargetNodeID != st.localNode.ID {
		return
	}

	// Create transfer operation
	op := &TransferOperation{
		SessionKey:   payload.SessionKey,
		SourceNodeID: payload.SourceNodeID,
		TargetNodeID: st.localNode.ID,
		State:        TransferStateReceived,
		StartTime:    payload.Timestamp,
		LastUpdate:   payload.Timestamp,
		Payload:      &payload,
	}

	st.mu.Lock()
	st.transfers[payload.SessionKey] = op
	st.mu.Unlock()

	// Send acknowledgment
	// Find source node from membership (simplified - in real implementation would look up node address)
	st.SendAck(&NodeInfo{
		ID: payload.SourceNodeID,
		// Need to look up actual address from membership
	}, payload.TransferID, true)

	// Call receive callback if set
	if st.onReceive != nil {
		go st.onReceive(&payload)
	}
}

// handleTransferAck handles a transfer acknowledgment.
func (st *SessionTransfer) handleTransferAck(data []byte) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	payload, _ := msg["payload"].(map[string]any)
	transferID, _ := payload["transfer_id"].(string)
	accepted, _ := payload["accepted"].(bool)

	st.mu.Lock()
	defer st.mu.Unlock()

	// Find and update transfer operation
	for sessionKey, op := range st.transfers {
		if op.Payload != nil && op.Payload.TransferID == transferID {
			if accepted {
				op.State = TransferStateCompleted
			} else {
				op.State = TransferStateFailed
			}
			op.LastUpdate = 0 // Use zero value

			// Clean up completed transfers after a delay
			if op.State == TransferStateCompleted {
				delete(st.transfers, sessionKey)
			}
			break
		}
	}
}

// SetReceiveCallback sets a callback for receiving session transfers.
func (st *SessionTransfer) SetReceiveCallback(callback func(*TransferPayload)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.onReceive = callback
}

// GetTransfer retrieves a transfer operation by session key.
func (st *SessionTransfer) GetTransfer(sessionKey string) (*TransferOperation, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	op, ok := st.transfers[sessionKey]
	return op, ok
}

// ListTransfers returns all active transfers.
func (st *SessionTransfer) ListTransfers() []*TransferOperation {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make([]*TransferOperation, 0, len(st.transfers))
	for _, op := range st.transfers {
		result = append(result, op)
	}
	return result
}

// RemoveTransfer removes a transfer operation.
func (st *SessionTransfer) RemoveTransfer(sessionKey string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.transfers, sessionKey)
}

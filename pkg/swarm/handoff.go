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
	"time"

	"github.com/google/uuid"
)

// HandoffReason represents the reason for a handoff.
type HandoffReason string

const (
	ReasonOverloaded   HandoffReason = "overloaded"    // Load is too high
	ReasonNoCapability HandoffReason = "no_capability" // Missing capability
	ReasonUserRequest  HandoffReason = "user_request"  // User explicitly requested
	ReasonNodeLeave    HandoffReason = "node_leave"    // Node is leaving
	ReasonShutdown     HandoffReason = "shutdown"      // Graceful shutdown
)

// HandoffState represents the state of a handoff operation.
type HandoffState string

const (
	HandoffStatePending   HandoffState = "pending"
	HandoffStateAccepted  HandoffState = "accepted"
	HandoffStateRejected  HandoffState = "rejected"
	HandoffStateCompleted HandoffState = "completed"
	HandoffStateFailed    HandoffState = "failed"
	HandoffStateTimeout   HandoffState = "timeout"
)

// HandoffRequest represents a request to hand off a session.
type HandoffRequest struct {
	RequestID       string            `json:"request_id"`
	Reason          HandoffReason     `json:"reason"`
	SessionKey      string            `json:"session_key"`
	SessionMessages []SessionMessage  `json:"session_messages,omitempty"`
	Context         map[string]any    `json:"context,omitempty"`
	RequiredCap     string            `json:"required_cap,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	FromNodeID      string            `json:"from_node_id"`
	FromNodeAddr    string            `json:"from_node_addr"`
	Timestamp       int64             `json:"timestamp"`
}

// HandoffResponse represents the response to a handoff request.
type HandoffResponse struct {
	RequestID  string       `json:"request_id"`
	Accepted   bool         `json:"accepted"`
	NodeID     string       `json:"node_id"`
	Reason     string       `json:"reason,omitempty"`
	SessionKey string       `json:"session_key,omitempty"` // New session key on target
	Timestamp  int64        `json:"timestamp"`
	State      HandoffState `json:"state"`
}

// HandoffCoordinator coordinates handoff operations between nodes.
type HandoffCoordinator struct {
	discovery  *DiscoveryService
	membership *MembershipManager
	config     HandoffConfig

	pending map[string]*HandoffOperation // request_id -> operation
	mu      sync.RWMutex
	conn    *net.UDPConn

	// Accept/reject callbacks
	onHandoffRequest  func(*HandoffRequest) *HandoffResponse
	onHandoffComplete func(*HandoffRequest, *HandoffResponse)
}

// HandoffOperation represents an ongoing handoff operation.
type HandoffOperation struct {
	Request    *HandoffRequest
	Response   *HandoffResponse
	State      HandoffState
	StartTime  time.Time
	LastUpdate time.Time
	RetryCount int
	TargetNode *NodeWithState
}

// NewHandoffCoordinator creates a new handoff coordinator.
func NewHandoffCoordinator(ds *DiscoveryService, config HandoffConfig) (*HandoffCoordinator, error) {
	hc := &HandoffCoordinator{
		discovery:  ds,
		membership: ds.membership,
		config:     config,
		pending:    make(map[string]*HandoffOperation),
	}

	// Bind UDP socket for handoff messages
	addr := fmt.Sprintf("%s:%d", ds.config.BindAddr, ds.config.RPC.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RPC address: %w", err)
	}

	hc.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for RPC: %w", err)
	}

	// Start message handler
	go hc.messageHandler()

	return hc, nil
}

// Close closes the handoff coordinator.
func (hc *HandoffCoordinator) Close() error {
	if hc.conn != nil {
		return hc.conn.Close()
	}
	return nil
}

// CanHandle checks if the local node can handle a request.
func (hc *HandoffCoordinator) CanHandle(requiredCap string) bool {
	if !hc.config.Enabled {
		return false
	}

	// Check load
	loadScore := hc.discovery.localNode.LoadScore
	if loadScore > hc.config.LoadThreshold {
		return false
	}

	// Check capability
	if requiredCap != "" {
		hasCap := false
		for _, cap := range hc.discovery.localNode.AgentCaps {
			if cap == requiredCap {
				hasCap = true
				break
			}
		}
		if !hasCap {
			return false
		}
	}

	return true
}

// InitiateHandoff initiates a handoff to another node.
func (hc *HandoffCoordinator) InitiateHandoff(ctx context.Context, req *HandoffRequest) (*HandoffResponse, error) {
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	req.FromNodeID = hc.discovery.localNode.ID
	req.FromNodeAddr = fmt.Sprintf("%s:%d", hc.discovery.localNode.Addr, hc.discovery.config.RPC.Port)
	req.Timestamp = time.Now().UnixNano()

	// Find target node
	targetNode, err := hc.findTargetNode(req)
	if err != nil {
		return &HandoffResponse{
			RequestID: req.RequestID,
			Accepted:  false,
			Reason:    err.Error(),
			State:     HandoffStateFailed,
		}, nil
	}

	// Create operation
	op := &HandoffOperation{
		Request:    req,
		State:      HandoffStatePending,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		TargetNode: targetNode,
	}

	hc.mu.Lock()
	hc.pending[req.RequestID] = op
	hc.mu.Unlock()

	// Send request
	err = hc.sendHandoffRequest(req, targetNode)
	if err != nil {
		hc.mu.Lock()
		op.State = HandoffStateFailed
		delete(hc.pending, req.RequestID)
		hc.mu.Unlock()

		return &HandoffResponse{
			RequestID: req.RequestID,
			Accepted:  false,
			Reason:    err.Error(),
			State:     HandoffStateFailed,
		}, nil
	}

	// Wait for response with timeout
	timeout := hc.config.Timeout.Duration
	if timeout == 0 {
		timeout = DefaultHandoffTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp := hc.waitForResponse(ctx, req.RequestID)

	// Retry if needed (op.RetryCount is 0 at this point, representing the first attempt)
	for !resp.Accepted && op.RetryCount < hc.config.MaxRetries {
		op.RetryCount++

		// Find new target
		newTarget, err := hc.findTargetNode(req)
		if err != nil {
			continue
		}
		op.TargetNode = newTarget

		// Delay before retry
		time.Sleep(hc.config.RetryDelay.Duration)

		// Send request
		err = hc.sendHandoffRequest(req, newTarget)
		if err != nil {
			continue
		}

		// Wait for response with timeout, preserving parent context
		retryCtx, retryCancel := context.WithTimeout(ctx, timeout)
		resp = hc.waitForResponse(retryCtx, req.RequestID)
		retryCancel()

		// If accepted, break out of retry loop
		if resp.Accepted {
			break
		}
	}

	// Clean up
	hc.mu.Lock()
	delete(hc.pending, req.RequestID)
	hc.mu.Unlock()

	// Notify callback
	if hc.onHandoffComplete != nil {
		go hc.onHandoffComplete(req, resp)
	}

	return resp, nil
}

// findTargetNode finds a suitable target node for handoff.
func (hc *HandoffCoordinator) findTargetNode(req *HandoffRequest) (*NodeWithState, error) {
	var candidates []*NodeWithState

	if req.RequiredCap != "" {
		// Find nodes with required capability
		candidates = hc.membership.SelectByCapability([]string{req.RequiredCap})
	} else {
		// Find all available nodes
		candidates = hc.membership.GetAvailableMembers()
	}

	if len(candidates) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// Select least loaded node
	target := candidates[0]
	for _, c := range candidates[1:] {
		if c.Node.LoadScore < target.Node.LoadScore {
			target = c
		}
	}

	return target, nil
}

// sendHandoffRequest sends a handoff request to a target node.
func (hc *HandoffCoordinator) sendHandoffRequest(req *HandoffRequest, target *NodeWithState) error {
	// Handoff message type
	msg := map[string]any{
		MsgFieldType:    RPCTypeHandoffRequest,
		MsgFieldPayload: req,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", target.Node.Addr, hc.discovery.config.RPC.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	hc.conn.SetWriteDeadline(time.Now().Add(DefaultUDPWriteDeadline))
	_, err = hc.conn.WriteToUDP(data, udpAddr)
	return err
}

// waitForResponse waits for a handoff response.
func (hc *HandoffCoordinator) waitForResponse(ctx context.Context, requestID string) *HandoffResponse {
	ticker := time.NewTicker(HandoffResponsePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.mu.Lock()
			if op, ok := hc.pending[requestID]; ok {
				op.State = HandoffStateTimeout
			}
			hc.mu.Unlock()

			return &HandoffResponse{
				RequestID: requestID,
				Accepted:  false,
				Reason:    "timeout",
				State:     HandoffStateTimeout,
			}
		case <-ticker.C:
			hc.mu.RLock()
			op, ok := hc.pending[requestID]
			hc.mu.RUnlock()

			if ok && op.Response != nil {
				return op.Response
			}
		}
	}
}

// messageHandler handles incoming handoff messages.
func (hc *HandoffCoordinator) messageHandler() {
	buf := make([]byte, MaxGossipMessageSize)

	for {
		n, addr, err := hc.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n > 0 {
			go hc.handleMessage(buf[:n], addr)
		}
	}
}

// handleMessage handles an incoming message.
func (hc *HandoffCoordinator) handleMessage(data []byte, addr *net.UDPAddr) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	msgType, ok := msg[MsgFieldType].(string)
	if !ok {
		return
	}

	switch RPCMessageType(msgType) {
	case RPCTypeHandoffRequest:
		hc.handleHandoffRequest(data, addr)
	case RPCTypeHandoffResponse:
		hc.handleHandoffResponse(data)
	}
}

// handleHandoffRequest handles a handoff request from another node.
func (hc *HandoffCoordinator) handleHandoffRequest(data []byte, addr *net.UDPAddr) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	payloadData, _ := json.Marshal(msg[MsgFieldPayload])
	var req HandoffRequest
	if err := json.Unmarshal(payloadData, &req); err != nil {
		return
	}

	// Check if we can handle it
	accepted := hc.CanHandle(req.RequiredCap)
	response := &HandoffResponse{
		RequestID: req.RequestID,
		Accepted:  accepted,
		NodeID:    hc.discovery.localNode.ID,
		State:     HandoffStateAccepted,
		Timestamp: time.Now().UnixNano(),
	}

	if !accepted {
		response.Reason = "cannot handle (overloaded or missing capability)"
		response.State = HandoffStateRejected
	}

	// Call custom handler if set
	if hc.onHandoffRequest != nil {
		response = hc.onHandoffRequest(&req)
	}

	// Send response
	respMsg := map[string]any{
		MsgFieldType:    RPCTypeHandoffResponse,
		MsgFieldPayload: response,
	}

	respData, _ := json.Marshal(respMsg)
	hc.conn.WriteToUDP(respData, addr)

	// Update operation if we accepted
	if accepted {
		op := &HandoffOperation{
			Request:  &req,
			Response: response,
			State:    HandoffStateAccepted,
		}

		hc.mu.Lock()
		hc.pending[req.RequestID] = op
		hc.mu.Unlock()
	}
}

// handleHandoffResponse handles a handoff response.
func (hc *HandoffCoordinator) handleHandoffResponse(data []byte) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	payloadData, _ := json.Marshal(msg[MsgFieldPayload])
	var resp HandoffResponse
	if err := json.Unmarshal(payloadData, &resp); err != nil {
		return
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	if op, ok := hc.pending[resp.RequestID]; ok {
		op.Response = &resp
		op.LastUpdate = time.Now()

		if resp.Accepted {
			op.State = HandoffStateAccepted
		} else {
			op.State = HandoffStateRejected
		}
	}
}

// SetRequestHandler sets a custom handler for handoff requests.
func (hc *HandoffCoordinator) SetRequestHandler(handler func(*HandoffRequest) *HandoffResponse) {
	hc.onHandoffRequest = handler
}

// SetCompleteHandler sets a callback for handoff completion.
func (hc *HandoffCoordinator) SetCompleteHandler(handler func(*HandoffRequest, *HandoffResponse)) {
	hc.onHandoffComplete = handler
}

// GetPending returns all pending handoff operations.
func (hc *HandoffCoordinator) GetPending() []*HandoffOperation {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make([]*HandoffOperation, 0, len(hc.pending))
	for _, op := range hc.pending {
		result = append(result, op)
	}
	return result
}

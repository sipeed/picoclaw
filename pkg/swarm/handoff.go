// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	picolib "github.com/sipeed/picoclaw/pkg/pico"
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
	RequestID       string                   `json:"request_id"`
	Reason          HandoffReason            `json:"reason"`
	SessionKey      string                   `json:"session_key"`
	SessionMessages []picolib.SessionMessage `json:"session_messages,omitempty"`
	Context         map[string]any           `json:"context,omitempty"`
	RequiredCap     string                   `json:"required_cap,omitempty"`
	Metadata        map[string]string        `json:"metadata,omitempty"`
	FromNodeID      string                   `json:"from_node_id"`
	FromNodeAddr    string                   `json:"from_node_addr"`
	Timestamp       int64                    `json:"timestamp"`
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

// HandoffSendFunc is the function signature for sending a handoff request to a
// target node via Pico channel. The caller provides the target node address and
// the request; the function returns the response synchronously.
type HandoffSendFunc func(ctx context.Context, targetAddr string, req *HandoffRequest) (*HandoffResponse, error)

// HandoffCoordinator coordinates handoff operations between nodes.
// Communication is handled via an injected send function (typically backed by PicoNodeClient).
type HandoffCoordinator struct {
	discovery  *DiscoveryService
	membership *MembershipManager
	config     HandoffConfig

	pending map[string]*HandoffOperation // request_id -> operation
	mu      sync.RWMutex

	// Injected communication function (set via SetSendFunc)
	sendRequestFn HandoffSendFunc

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
// Unlike the previous version, this does NOT open a UDP socket.
// Communication is injected via SetSendFunc.
func NewHandoffCoordinator(ds *DiscoveryService, config HandoffConfig) *HandoffCoordinator {
	return &HandoffCoordinator{
		discovery:  ds,
		membership: ds.membership,
		config:     config,
		pending:    make(map[string]*HandoffOperation),
	}
}

// Close cleans up the handoff coordinator.
func (hc *HandoffCoordinator) Close() error {
	return nil
}

// SetSendFunc injects the function used to send handoff requests to remote nodes.
func (hc *HandoffCoordinator) SetSendFunc(fn HandoffSendFunc) {
	hc.sendRequestFn = fn
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
	if hc.sendRequestFn == nil {
		return nil, fmt.Errorf("handoff send function not configured")
	}

	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	req.FromNodeID = hc.discovery.localNode.ID
	req.FromNodeAddr = hc.discovery.localNode.Addr
	req.Timestamp = time.Now().UnixNano()

	// Find target node
	targetNode, findErr := hc.findTargetNode(req)
	if findErr != nil {
		// Intentionally return nil error: convert internal error to a rejected response.
		reason := findErr.Error()
		return &HandoffResponse{ //nolint:nilerr // intentional: wraps error as rejected response
			RequestID: req.RequestID,
			Accepted:  false,
			Reason:    reason,
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

	// Build target address (use HTTPPort for Pico channel)
	targetAddr := picolib.BuildNodeAddr(targetNode.Node.Addr, targetNode.Node.HTTPPort)

	// Send request synchronously via Pico
	resp, err := hc.sendRequestFn(ctx, targetAddr, req)
	if err != nil {
		// First attempt failed, try retries
		resp = &HandoffResponse{
			RequestID: req.RequestID,
			Accepted:  false,
			Reason:    err.Error(),
			State:     HandoffStateFailed,
		}
	}

	// Retry if needed
	for !resp.Accepted && op.RetryCount < hc.config.MaxRetries {
		op.RetryCount++

		// Find new target
		newTarget, findErr := hc.findTargetNode(req)
		if findErr != nil {
			continue
		}
		op.TargetNode = newTarget

		// Delay before retry
		time.Sleep(hc.config.RetryDelay.Duration)

		newAddr := picolib.BuildNodeAddr(newTarget.Node.Addr, newTarget.Node.HTTPPort)
		retryResp, retryErr := hc.sendRequestFn(ctx, newAddr, req)
		if retryErr != nil {
			resp = &HandoffResponse{
				RequestID: req.RequestID,
				Accepted:  false,
				Reason:    retryErr.Error(),
				State:     HandoffStateFailed,
			}
			continue
		}

		resp = retryResp
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

// HandleIncomingHandoff handles a handoff request received from another node
// (via the Pico channel node request handler). Returns a HandoffResponse.
func (hc *HandoffCoordinator) HandleIncomingHandoff(req *HandoffRequest) *HandoffResponse {
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
		response = hc.onHandoffRequest(req)
	}

	return response
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

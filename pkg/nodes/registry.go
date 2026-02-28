package nodes

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// NodeSession holds the state of a connected node.
type NodeSession struct {
	// NodeID is the stable identifier for this node (device.id or client.id).
	NodeID string

	// ConnID is the unique connection identifier for this session.
	ConnID string

	// DisplayName is the human-readable name of the node.
	DisplayName string

	// Platform is the OS platform (linux, darwin, ios, android, etc.).
	Platform string

	// DeviceFamily is the device category (phone, tablet, desktop, server, etc.).
	DeviceFamily string

	// ModelIdentifier is the hardware model identifier.
	ModelIdentifier string

	// Version is the client version string.
	Version string

	// Commands lists the commands this node has declared it can handle.
	Commands []string

	// Caps lists the capability groups this node exposes.
	Caps []string

	// RemoteIP is the remote IP address of the connection (may be empty).
	RemoteIP string

	// ConnectedAtMs is the unix millisecond timestamp when the node connected.
	ConnectedAtMs int64

	// conn is the underlying WebSocket connection.
	conn *websocket.Conn
	// mu protects conn sends.
	mu sync.Mutex
}

// SendEvent sends an event frame to this node.
// Returns an error if the send fails.
func (s *NodeSession) SendEvent(event string, payload any) error {
	frame := EventFrame{
		Type:    FrameTypeEvent,
		Event:   event,
		Payload: payload,
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(websocket.TextMessage, data)
}

// HasCommand returns true if this node declared the given command.
func (s *NodeSession) HasCommand(cmd string) bool {
	for _, c := range s.Commands {
		if c == cmd {
			return true
		}
	}
	return false
}

// pendingInvoke tracks an in-flight node invoke waiting for a result.
type pendingInvoke struct {
	nodeID  string
	command string
	resolve chan InvokeResult
}

// InvokeResult holds the result of a node invoke operation.
type InvokeResult struct {
	Ok          bool
	Payload     any
	PayloadJSON *string
	Error       *InvokeError
}

// PairingRequest represents a node waiting for manual approval.
// When the server is configured with PairingMode="manual", new node connections
// are held here until an operator approves or rejects them.
type PairingRequest struct {
	// RequestID is a unique ID for this pairing request.
	RequestID string

	// NodeID is the connecting node's ID.
	NodeID string

	// DisplayName is the connecting node's human-readable name.
	DisplayName string

	// Platform is the OS of the connecting node.
	Platform string

	// DeviceFamily is the device family of the connecting node.
	DeviceFamily string

	// RemoteIP is the remote address of the connecting node.
	RemoteIP string

	// RequestedAtMs is when the pairing request was created.
	RequestedAtMs int64

	// resolved is closed when the request is approved or rejected.
	resolved chan bool
}

// Registry tracks all connected nodes and manages in-flight invocations.
type Registry struct {
	mu          sync.RWMutex
	nodesByID   map[string]*NodeSession
	nodesByConn map[string]string // connID -> nodeID
	pendingByID map[string]*pendingInvoke

	pairingMu       sync.RWMutex
	pairingByID     map[string]*PairingRequest // requestID -> PairingRequest
	pairingByNodeID map[string]string          // nodeID -> requestID
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		nodesByID:       make(map[string]*NodeSession),
		nodesByConn:     make(map[string]string),
		pendingByID:     make(map[string]*pendingInvoke),
		pairingByID:     make(map[string]*PairingRequest),
		pairingByNodeID: make(map[string]string),
	}
}

// Register adds a new node session to the registry.
func (r *Registry) Register(connID string, conn *websocket.Conn, params *ConnectParams, remoteIP string) *NodeSession {
	// Determine the stable node ID: prefer device.id, fall back to client.instanceId then client.id.
	nodeID := ""
	if params.Device != nil && params.Device.ID != "" {
		nodeID = params.Device.ID
	} else if params.Client.InstanceID != "" {
		nodeID = params.Client.InstanceID
	} else {
		nodeID = params.Client.ID
	}

	session := &NodeSession{
		NodeID:          nodeID,
		ConnID:          connID,
		DisplayName:     params.Client.DisplayName,
		Platform:        params.Client.Platform,
		DeviceFamily:    params.Client.DeviceFamily,
		ModelIdentifier: params.Client.ModelIdentifier,
		Version:         params.Client.Version,
		Commands:        params.Commands,
		Caps:            params.Caps,
		RemoteIP:        remoteIP,
		ConnectedAtMs:   time.Now().UnixMilli(),
		conn:            conn,
	}

	r.mu.Lock()
	r.nodesByID[nodeID] = session
	r.nodesByConn[connID] = nodeID
	r.mu.Unlock()

	return session
}

// Unregister removes a node by connection ID. Returns the nodeID if found.
func (r *Registry) Unregister(connID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeID, ok := r.nodesByConn[connID]
	if !ok {
		return ""
	}
	delete(r.nodesByConn, connID)
	delete(r.nodesByID, nodeID)

	// Fail any pending invokes for this node.
	for id, pending := range r.pendingByID {
		if pending.nodeID != nodeID {
			continue
		}
		pending.resolve <- InvokeResult{
			Ok:    false,
			Error: &InvokeError{Code: ErrCodeUnavailable, Message: "node disconnected"},
		}
		delete(r.pendingByID, id)
	}

	return nodeID
}

// Get returns the NodeSession for the given nodeID, or nil if not found.
func (r *Registry) Get(nodeID string) *NodeSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodesByID[nodeID]
}

// GetByConn returns the NodeSession for the given connection ID, or nil.
func (r *Registry) GetByConn(connID string) *NodeSession {
	r.mu.RLock()
	nodeID := r.nodesByConn[connID]
	r.mu.RUnlock()
	if nodeID == "" {
		return nil
	}
	return r.Get(nodeID)
}

// ListConnected returns a snapshot of all currently connected node sessions.
func (r *Registry) ListConnected() []*NodeSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*NodeSession, 0, len(r.nodesByID))
	for _, s := range r.nodesByID {
		result = append(result, s)
	}
	return result
}

// Invoke sends a command to a node and waits for the result.
// The server sends a node.invoke.request event to the node; the node
// executes the command and replies with a node.invoke.result request.
func (r *Registry) Invoke(nodeID, command string, params any, timeoutMs int) InvokeResult {
	node := r.Get(nodeID)
	if node == nil {
		return InvokeResult{
			Ok:    false,
			Error: &InvokeError{Code: ErrCodeUnavailable, Message: "node not connected"},
		}
	}

	requestID := uuid.New().String()
	ch := make(chan InvokeResult, 1)

	pending := &pendingInvoke{
		nodeID:  nodeID,
		command: command,
		resolve: ch,
	}

	r.mu.Lock()
	r.pendingByID[requestID] = pending
	r.mu.Unlock()

	// Build the invoke request payload.
	var paramsJSONPtr *string
	if params != nil {
		data, err := json.Marshal(params)
		if err == nil {
			s := string(data)
			paramsJSONPtr = &s
		}
	}
	if timeoutMs <= 0 {
		timeoutMs = 30_000
	}
	payload := NodeInvokeRequestPayload{
		ID:         requestID,
		NodeID:     nodeID,
		Command:    command,
		ParamsJSON: paramsJSONPtr,
		TimeoutMs:  &timeoutMs,
	}

	// Send the event to the node.
	if err := node.SendEvent(EventNodeInvokeRequest, payload); err != nil {
		r.mu.Lock()
		delete(r.pendingByID, requestID)
		r.mu.Unlock()
		return InvokeResult{
			Ok:    false,
			Error: &InvokeError{Code: ErrCodeUnavailable, Message: "failed to send invoke to node"},
		}
	}

	// Wait for the result with timeout.
	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case result := <-ch:
		return result
	case <-timer.C:
		r.mu.Lock()
		delete(r.pendingByID, requestID)
		r.mu.Unlock()
		return InvokeResult{
			Ok:    false,
			Error: &InvokeError{Code: ErrCodeTimeout, Message: "node invoke timed out"},
		}
	}
}

// HandleInvokeResult processes a node.invoke.result response from a node.
// Returns true if the result was matched to a pending invoke.
func (r *Registry) HandleInvokeResult(p *NodeInvokeResultParams) bool {
	r.mu.Lock()
	pending, ok := r.pendingByID[p.ID]
	if !ok {
		r.mu.Unlock()
		return false
	}
	if pending.nodeID != p.NodeID {
		r.mu.Unlock()
		return false
	}
	delete(r.pendingByID, p.ID)
	r.mu.Unlock()

	result := InvokeResult{
		Ok:          p.Ok,
		Payload:     p.Payload,
		PayloadJSON: p.PayloadJSON,
		Error:       p.Error,
	}
	pending.resolve <- result
	return true
}

// SendEvent sends an event to the node with the given nodeID.
func (r *Registry) SendEvent(nodeID, event string, payload any) bool {
	node := r.Get(nodeID)
	if node == nil {
		return false
	}
	return node.SendEvent(event, payload) == nil
}

// Disconnect forcibly closes the WebSocket connection for the given nodeID.
// This is used to reject a connected node.
func (r *Registry) Disconnect(nodeID string) bool {
	node := r.Get(nodeID)
	if node == nil {
		return false
	}
	node.mu.Lock()
	defer node.mu.Unlock()
	_ = node.conn.Close()
	return true
}

// --- Pairing queue (for manual pairing mode) ---

// AddPairingRequest adds a node to the pending pairing queue and blocks until
// approved or rejected (or the channel is closed). Returns true if approved.
func (r *Registry) AddPairingRequest(nodeID, displayName, platform, deviceFamily, remoteIP string) *PairingRequest {
	req := &PairingRequest{
		RequestID:     uuid.New().String(),
		NodeID:        nodeID,
		DisplayName:   displayName,
		Platform:      platform,
		DeviceFamily:  deviceFamily,
		RemoteIP:      remoteIP,
		RequestedAtMs: time.Now().UnixMilli(),
		resolved:      make(chan bool, 1),
	}
	r.pairingMu.Lock()
	r.pairingByID[req.RequestID] = req
	r.pairingByNodeID[nodeID] = req.RequestID
	r.pairingMu.Unlock()
	return req
}

// WaitPairingDecision blocks until the pairing request is resolved.
// Returns true if approved, false if rejected.
func (r *Registry) WaitPairingDecision(req *PairingRequest) bool {
	return <-req.resolved
}

// ListPairingRequests returns all pending pairing requests.
func (r *Registry) ListPairingRequests() []*PairingRequest {
	r.pairingMu.RLock()
	defer r.pairingMu.RUnlock()
	result := make([]*PairingRequest, 0, len(r.pairingByID))
	for _, req := range r.pairingByID {
		result = append(result, req)
	}
	return result
}

// ApprovePairing approves the pairing request with the given requestID.
// Returns false if the request is not found.
func (r *Registry) ApprovePairing(requestID string) bool {
	r.pairingMu.Lock()
	req, ok := r.pairingByID[requestID]
	if ok {
		delete(r.pairingByID, requestID)
		delete(r.pairingByNodeID, req.NodeID)
	}
	r.pairingMu.Unlock()
	if !ok {
		return false
	}
	req.resolved <- true
	return true
}

// RejectPairing rejects the pairing request with the given requestID.
// Returns false if the request is not found.
func (r *Registry) RejectPairing(requestID string) bool {
	r.pairingMu.Lock()
	req, ok := r.pairingByID[requestID]
	if ok {
		delete(r.pairingByID, requestID)
		delete(r.pairingByNodeID, req.NodeID)
	}
	r.pairingMu.Unlock()
	if !ok {
		return false
	}
	req.resolved <- false
	return true
}

// RemovePairingRequest removes a pending request without notifying (used on disconnect).
func (r *Registry) RemovePairingRequest(nodeID string) {
	r.pairingMu.Lock()
	reqID, ok := r.pairingByNodeID[nodeID]
	if ok {
		delete(r.pairingByNodeID, nodeID)
		if req, exists := r.pairingByID[reqID]; exists {
			delete(r.pairingByID, reqID)
			// Non-blocking: resolve as rejected so the goroutine unblocks.
			select {
			case req.resolved <- false:
			default:
			}
		}
	}
	r.pairingMu.Unlock()
}

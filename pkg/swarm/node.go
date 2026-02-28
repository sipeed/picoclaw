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
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// NodeInfo represents a node in the swarm cluster.
type NodeInfo struct {
	ID        string            `json:"id"`         // Unique node identifier
	Addr      string            `json:"addr"`       // Listening address
	Port      int               `json:"port"`       // RPC port
	AgentCaps map[string]string `json:"agent_caps"` // Agent capabilities {agent_id: capability}
	LoadScore float64           `json:"load_score"` // Load score 0-1
	Labels    map[string]string `json:"labels"`     // Custom labels
	HTTPPort  int               `json:"http_port"`  // HTTP/gateway port (for Pico channel)
	Timestamp int64             `json:"timestamp"`  // Last update time (Unix nano)
	Version   string            `json:"version"`    // PicoClaw version
}

// IsAlive checks if the node is considered alive based on timestamp.
func (n *NodeInfo) IsAlive(timeout time.Duration) bool {
	if n.Timestamp == 0 {
		return false
	}
	age := time.Since(time.Unix(0, n.Timestamp))
	return age < timeout
}

// String returns a JSON representation of the node.
func (n *NodeInfo) String() string {
	data, _ := json.Marshal(n)
	return string(data)
}

// GetAddress returns the full address (host:port) for RPC communication.
func (n *NodeInfo) GetAddress() string {
	if n.Port > 0 {
		return fmt.Sprintf("%s:%d", n.Addr, n.Port)
	}
	return n.Addr
}

// NodeStatus represents the current status of a node.
type NodeStatus string

const (
	NodeStatusAlive   NodeStatus = "alive"
	NodeStatusSuspect NodeStatus = "suspect"
	NodeStatusDead    NodeStatus = "dead"
	NodeStatusLeft    NodeStatus = "left"
)

// NodeState represents the state of a node in the membership view.
type NodeState struct {
	Node        *NodeInfo  `json:"node"`
	Status      NodeStatus `json:"status"`
	StatusSince int64      `json:"status_since"` // Unix nano when status was set
	LastSeen    int64      `json:"last_seen"`    // Unix nano of last sighting
	LastPing    int64      `json:"last_ping"`    // Unix nano of last successful ping
	PingSuccess int        `json:"ping_success"` // Consecutive successful pings
	PingFailure int        `json:"ping_failure"` // Consecutive failed pings
}

// IsAvailable returns true if the node is available for handoff.
func (ns *NodeState) IsAvailable() bool {
	return ns.Status == NodeStatusAlive && ns.Node.LoadScore < DefaultAvailableLoadThreshold
}

// UpdateStatus updates the node status with timestamp.
func (ns *NodeState) UpdateStatus(status NodeStatus) {
	ns.Status = status
	ns.StatusSince = time.Now().UnixNano()
}

// NodeEvent represents a node state change event.
type NodeEvent struct {
	Node  *NodeInfo `json:"node"`
	Event EventType `json:"event"`
	Time  int64     `json:"time"`
}

// EventType represents the type of node event.
type EventType string

const (
	EventJoin   EventType = "join"
	EventLeave  EventType = "leave"
	EventUpdate EventType = "update"
)

// EventHandler is a callback function for node events.
type EventHandler func(*NodeEvent)

// EventHandlerID is a unique identifier for a subscribed handler.
type EventHandlerID int

// EventDispatcher manages event handlers.
type EventDispatcher struct {
	handlers []EventHandler
	mu       sync.RWMutex
	nextID   EventHandlerID
	ids      map[EventHandlerID]int // handler ID -> index in handlers slice
}

// NewEventDispatcher creates a new event dispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make([]EventHandler, 0),
		ids:      make(map[EventHandlerID]int),
		nextID:   1,
	}
}

// Subscribe adds a new event handler and returns its ID.
func (ed *EventDispatcher) Subscribe(handler EventHandler) EventHandlerID {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	id := ed.nextID
	ed.nextID++

	ed.handlers = append(ed.handlers, handler)
	ed.ids[id] = len(ed.handlers) - 1
	return id
}

// Unsubscribe removes an event handler by ID.
func (ed *EventDispatcher) Unsubscribe(id EventHandlerID) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	idx, ok := ed.ids[id]
	if !ok {
		return
	}

	// Remove handler
	ed.handlers = append(ed.handlers[:idx], ed.handlers[idx+1:]...)

	// Update indices
	delete(ed.ids, id)
	for handlerID, handlerIdx := range ed.ids {
		if handlerIdx > idx {
			ed.ids[handlerID] = handlerIdx - 1
		}
	}
}

// Dispatch sends an event to all registered handlers.
func (ed *EventDispatcher) Dispatch(event *NodeEvent) {
	ed.DispatchContext(event, nil)
}

// DispatchContext sends an event to all registered handlers with context cancellation support.
func (ed *EventDispatcher) DispatchContext(event *NodeEvent, ctx context.Context) {
	ed.mu.RLock()
	handlers := make([]EventHandler, len(ed.handlers))
	copy(handlers, ed.handlers)
	ed.mu.RUnlock()

	for _, handler := range handlers {
		// Run handlers in goroutines to avoid blocking
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorCF("swarm", "handler panic recovered", map[string]any{"panic": r})
				}
			}()

			// Check if context is canceled
			if ctx != nil {
				select {
				case <-ctx.Done():
					logger.DebugC("swarm", "handler skipped due to context cancellation")
					return
				default:
				}
			}

			h(event)
		}(handler)
	}
}

// NodeStats tracks statistics about a node.
type NodeStats struct {
	MessagesSent      int64     `json:"messages_sent"`
	MessagesReceived  int64     `json:"messages_received"`
	HandoffsAccepted  int       `json:"handoffs_accepted"`
	HandoffsInitiated int       `json:"handoffs_initiated"`
	LastError         string    `json:"last_error,omitempty"`
	LastErrorTime     time.Time `json:"last_error_time,omitempty"`
	UptimeStart       time.Time `json:"uptime_start"`
}

// NodeWithState combines a node with its state and stats.
type NodeWithState struct {
	Node  *NodeInfo  `json:"node"`
	State *NodeState `json:"state"`
	Stats *NodeStats `json:"stats,omitempty"`
}

// IsAvailable returns true if the node is available for handoff.
func (nws *NodeWithState) IsAvailable() bool {
	if nws.State == nil || nws.Node == nil {
		return false
	}
	return nws.State.Status == NodeStatusAlive && nws.Node.LoadScore < DefaultAvailableLoadThreshold
}

// ClusterView represents the current view of the cluster.
type ClusterView struct {
	Nodes       map[string]*NodeWithState `json:"nodes"`
	LocalNodeID string                    `json:"local_node_id"`
	Size        int                       `json:"size"`
	Version     int64                     `json:"version"` // View version for conflict detection
	mu          sync.RWMutex
}

// NewClusterView creates a new cluster view.
func NewClusterView(localNodeID string) *ClusterView {
	return &ClusterView{
		Nodes:       make(map[string]*NodeWithState),
		LocalNodeID: localNodeID,
		Version:     time.Now().UnixNano(),
	}
}

// AddOrUpdate adds or updates a node in the view.
func (cv *ClusterView) AddOrUpdate(node *NodeInfo) *NodeWithState {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	cv.Version++

	existing, ok := cv.Nodes[node.ID]
	if ok {
		// Update existing node
		existing.Node = node
		return existing
	}

	// Add new node
	nws := &NodeWithState{
		Node: node,
		State: &NodeState{
			Node:        node,
			Status:      NodeStatusAlive,
			StatusSince: time.Now().UnixNano(),
			LastSeen:    time.Now().UnixNano(),
		},
		Stats: &NodeStats{
			UptimeStart: time.Now(),
		},
	}
	cv.Nodes[node.ID] = nws
	cv.Size = len(cv.Nodes)
	return nws
}

// Remove removes a node from the view.
func (cv *ClusterView) Remove(nodeID string) {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	cv.Version++
	delete(cv.Nodes, nodeID)
	cv.Size = len(cv.Nodes)
}

// Get retrieves a node from the view.
func (cv *ClusterView) Get(nodeID string) (*NodeWithState, bool) {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	nws, ok := cv.Nodes[nodeID]
	return nws, ok
}

// List returns all nodes in the view.
func (cv *ClusterView) List() []*NodeWithState {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	result := make([]*NodeWithState, 0, len(cv.Nodes))
	for _, nws := range cv.Nodes {
		result = append(result, nws)
	}
	return result
}

// GetAliveNodes returns all alive nodes.
func (cv *ClusterView) GetAliveNodes() []*NodeWithState {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	result := make([]*NodeWithState, 0)
	for _, nws := range cv.Nodes {
		if nws.State.Status == NodeStatusAlive {
			result = append(result, nws)
		}
	}
	return result
}

// GetAvailableNodes returns all available nodes (alive and not overloaded).
func (cv *ClusterView) GetAvailableNodes() []*NodeWithState {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	result := make([]*NodeWithState, 0)
	for _, nws := range cv.Nodes {
		if nws.IsAvailable() {
			result = append(result, nws)
		}
	}
	return result
}

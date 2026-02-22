// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NodeRole defines the role of a swarm node
type NodeRole string

const (
	RoleCoordinator NodeRole = "coordinator"
	RoleWorker      NodeRole = "worker"
	RoleSpecialist  NodeRole = "specialist"
)

// NodeStatus defines the status of a swarm node
type NodeStatus string

const (
	StatusOnline    NodeStatus = "online"
	StatusBusy      NodeStatus = "busy"
	StatusOffline   NodeStatus = "offline"
	StatusSuspicious NodeStatus = "suspicious"
	StatusDraining  NodeStatus = "draining"
)

// NodeInfo represents a node in the swarm
type NodeInfo struct {
	ID           string            `json:"id"`
	Role         NodeRole          `json:"role"`
	Capabilities []string          `json:"capabilities"`
	Model        string            `json:"model"`
	Status       NodeStatus        `json:"status"`
	Load         float64           `json:"load"`
	TasksRunning int               `json:"tasks_running"`
	MaxTasks     int               `json:"max_tasks"`
	Metadata     map[string]string `json:"metadata"`
	LastSeen     int64             `json:"last_seen"`
	StartedAt    int64             `json:"started_at"`
	Address      string            `json:"address"` // NATS address for direct messaging
}

// SwarmTaskType defines how a task is routed
type SwarmTaskType string

const (
	TaskTypeDirect    SwarmTaskType = "direct"    // Assigned to specific node
	TaskTypeWorkflow  SwarmTaskType = "workflow"  // Temporal workflow
	TaskTypeBroadcast SwarmTaskType = "broadcast" // Broadcast by capability
)

// SwarmTaskStatus defines the status of a task
type SwarmTaskStatus string

const (
	TaskPending  SwarmTaskStatus = "pending"
	TaskAssigned SwarmTaskStatus = "assigned"
	TaskRunning  SwarmTaskStatus = "running"
	TaskDone     SwarmTaskStatus = "done"
	TaskFailed   SwarmTaskStatus = "failed"
)

// SwarmTask represents a task in the swarm
type SwarmTask struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Type        SwarmTaskType          `json:"type"`
	Priority    int                    `json:"priority"` // 0=low, 1=normal, 2=high, 3=critical
	Capability  string                 `json:"capability"`
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context"`
	AssignedTo  string                 `json:"assigned_to"`
	Status      SwarmTaskStatus        `json:"status"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	CompletedAt int64                  `json:"completed_at,omitempty"`
	Timeout     int64                  `json:"timeout"` // Timeout in milliseconds
}

// NewSwarmTask creates a new task with default values
func NewSwarmTask(taskType SwarmTaskType, capability, prompt string) *SwarmTask {
	return &SwarmTask{
		ID:         generateTaskID(),
		Type:       taskType,
		Priority:   1, // normal
		Capability: capability,
		Prompt:     prompt,
		Context:    make(map[string]interface{}),
		Status:     TaskPending,
		CreatedAt:  time.Now().UnixMilli(),
		Timeout:    10 * 60 * 1000, // 10 minutes default
	}
}

// TaskResult is sent when a task completes
type TaskResult struct {
	TaskID      string `json:"task_id"`
	NodeID      string `json:"node_id"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
	CompletedAt int64  `json:"completed_at"`
}

// TaskProgress is sent periodically during task execution
type TaskProgress struct {
	TaskID   string  `json:"task_id"`
	NodeID   string  `json:"node_id"`
	Progress float64 `json:"progress"` // 0.0 to 1.0
	Message  string  `json:"message"`
}

// DiscoveryAnnounce is published when a node joins
type DiscoveryAnnounce struct {
	Node      NodeInfo `json:"node"`
	Timestamp int64    `json:"timestamp"`
}

// DiscoveryQuery is published to discover nodes
type DiscoveryQuery struct {
	RequesterID string   `json:"requester_id"`
	Capability  string   `json:"capability,omitempty"` // Filter by capability
	Role        NodeRole `json:"role,omitempty"`       // Filter by role
}

// Heartbeat is published periodically by each node
type Heartbeat struct {
	NodeID       string     `json:"node_id"`
	Role         NodeRole   `json:"role,omitempty"`
	Status       NodeStatus `json:"status"`
	Load         float64    `json:"load"`
	TasksRunning int        `json:"tasks_running"`
	Timestamp    int64      `json:"timestamp"`
	Capabilities []string   `json:"capabilities,omitempty"`
	HID          string     `json:"hid,omitempty"`
	SID          string     `json:"sid,omitempty"`
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task-%s", uuid.New().String()[:8])
}

// TaskEventType represents the type of task event
type TaskEventType string

const (
	TaskEventCreated   TaskEventType = "created"
	TaskEventAssigned  TaskEventType = "assigned"
	TaskEventStarted   TaskEventType = "started"
	TaskEventProgress  TaskEventType = "progress"
	TaskEventCompleted TaskEventType = "completed"
	TaskEventFailed    TaskEventType = "failed"
	TaskEventRetry     TaskEventType = "retry"
	TaskEventCheckpoint TaskEventType = "checkpoint"
)

// TaskEvent represents a single event in task lifecycle
type TaskEvent struct {
	EventID     string                 `json:"event_id"`
	TaskID      string                 `json:"task_id"`
	EventType   TaskEventType          `json:"event_type"`
	Timestamp   int64                  `json:"timestamp"`
	NodeID      string                 `json:"node_id,omitempty"`
	Status      SwarmTaskStatus        `json:"status,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Progress    float64                `json:"progress,omitempty"`
}

// CheckpointType defines the type of checkpoint
type CheckpointType string

const (
	CheckpointTypeProgress CheckpointType = "progress" // Periodic progress checkpoint
	CheckpointTypeMilestone CheckpointType = "milestone" // Significant milestone reached
	CheckpointTypePreFailover CheckpointType = "pre_failover" // Before potential failover
	CheckpointTypeUserCheckpointType CheckpointType = "user" // User-requested checkpoint
)

// TaskCheckpoint represents a saved state for task recovery
type TaskCheckpoint struct {
	CheckpointID  string                 `json:"checkpoint_id"`
	TaskID        string                 `json:"task_id"`
	Type          CheckpointType         `json:"type"`
	Timestamp     int64                  `json:"timestamp"`
	NodeID        string                 `json:"node_id"`
	Progress      float64                `json:"progress"`       // 0.0 to 1.0
	State         map[string]interface{} `json:"state"`          // Arbitrary state data
	PartialResult string                 `json:"partial_result"` // Partial output so far
	Context       map[string]interface{} `json:"context"`        // LLM context/messages
	Metadata      map[string]string      `json:"metadata"`       // Additional metadata
}

// DAGNode represents a single node in a DAG workflow
type DAGNode struct {
	ID          string                 `json:"id"`
	Task        *SwarmTask              `json:"task"`
	Dependencies []string               `json:"dependencies"` // IDs of nodes this depends on
	Status      DAGNodeStatus           `json:"status"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   int64                  `json:"started_at,omitempty"`
	CompletedAt int64                  `json:"completed_at,omitempty"`
}

// DAGNodeStatus represents the status of a DAG node
type DAGNodeStatus string

const (
	DAGNodePending   DAGNodeStatus = "pending"
	DAGNodeReady     DAGNodeStatus = "ready"
	DAGNodeRunning   DAGNodeStatus = "running"
	DAGNodeCompleted DAGNodeStatus = "completed"
	DAGNodeFailed    DAGNodeStatus = "failed"
	DAGNodeSkipped   DAGNodeStatus = "skipped"
)

// Capability represents a specialized capability that a node can provide
type Capability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Metadata    map[string]interface{} `json:"metadata"`
	NodeID      string                 `json:"node_id"`
	RegisteredAt int64                 `json:"registered_at"`
}

// CapabilityRequest is used to discover capabilities across the swarm
type CapabilityRequest struct {
	RequesterID string   `json:"requester_id"`
	Capability  string   `json:"capability,omitempty"` // Optional: filter by specific capability
	Version     string   `json:"version,omitempty"`    // Optional: filter by version
}

// CapabilityResponse is the response to a capability discovery request
type CapabilityResponse struct {
	Capabilities []Capability `json:"capabilities"`
	RequestID    string       `json:"request_id"`
	Timestamp    int64        `json:"timestamp"`
}

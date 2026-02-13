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
	StatusOnline  NodeStatus = "online"
	StatusBusy    NodeStatus = "busy"
	StatusOffline NodeStatus = "offline"
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
	Status       NodeStatus `json:"status"`
	Load         float64    `json:"load"`
	TasksRunning int        `json:"tasks_running"`
	Timestamp    int64      `json:"timestamp"`
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task-%s", uuid.New().String()[:8])
}

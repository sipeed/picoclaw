package core

import (
	"encoding/json"
	"errors"
	"time"
)

// --- Common Errors ---
var (
	ErrSwarmNotFound = errors.New("swarm not found")
	ErrNodeNotFound  = errors.New("node not found")
	ErrTaskTimeout   = errors.New("task timeout")
	ErrMailboxFull   = errors.New("mailbox full")
)

// --- ID Types ---
type SwarmID string
type NodeID string

// --- Enums ---
type SwarmStatus string
const (
	SwarmStatusActive    SwarmStatus = "active"
	SwarmStatusPaused    SwarmStatus = "paused"
	SwarmStatusCompleted SwarmStatus = "completed"
)

type NodeStatus string
const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
)

// --- Core Structs ---
type Swarm struct {
	ID            SwarmID   `json:"id"`
	Goal          string    `json:"goal"`
	Status        SwarmStatus `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	OriginChannel string    `json:"origin_channel"`
	OriginChatID  string    `json:"origin_chat_id"`
}

type Node struct {
	ID        NodeID     `json:"id"`
	SwarmID   SwarmID    `json:"swarm_id"`
	ParentID  NodeID     `json:"parent_id"`
	Role      Role       `json:"role"`
	Task      string     `json:"task"`
	Status    NodeStatus `json:"status"`
	Output    string     `json:"output"`
	Stats     NodeStats  `json:"stats"`
}

type Role struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Tools        []string `json:"tools"`
	Model        string   `json:"model"`
}

type NodeStats struct {
	Iterations   int `json:"iterations"`
	TokensInput  int `json:"tokens_input"`
	TokensOutput int `json:"tokens_output"`
}

// --- Event Protocol ---
type EventType string
const (
	EventNodeThinking  EventType = "node.thinking"
	EventNodeCompleted EventType = "node.completed"
	EventNodeFailed    EventType = "node.failed"
	EventSwarmSpawned  EventType = "swarm.spawned"
)

type Event struct {
	Type      EventType      `json:"type"`
	SwarmID   SwarmID        `json:"swarm_id"`
	NodeID    NodeID         `json:"node_id"`
	Payload   map[string]any `json:"payload"`
}

// --- LLM Types ---
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type LLMResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Usage     TokenUsage `json:"usage"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// --- Memory Types ---
type Fact struct {
	SwarmID    SwarmID        `json:"swarm_id"`
	Content    string         `json:"content"`
	Confidence float64        `json:"confidence"`
	Source     string         `json:"source"`
	Metadata   map[string]any `json:"metadata"`
}

type FactResult struct {
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

// --- Relay Types ---
type RelayMessage struct {
	Channel string
	ChatID  string
	Content string
}

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package pico

// SessionMessage represents a message in a session.
// This is shared across handoff, session transfer, and inter-node communication.
type SessionMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Timestamp int64          `json:"timestamp,omitempty"`
	ToolCalls []ToolCallData `json:"tool_calls,omitempty"`
}

// ToolCallData represents tool call information in a message.
type ToolCallData struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// DirectMessage represents a direct message sent to another node.
type DirectMessage struct {
	MessageID    string            `json:"message_id"`
	SourceNodeID string            `json:"source_node_id"`
	TargetNodeID string            `json:"target_node_id"`
	Content      string            `json:"content"`
	Channel      string            `json:"channel"`
	ChatID       string            `json:"chat_id"`
	SenderID     string            `json:"sender_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    int64             `json:"timestamp"`
}

// DirectMessageResponse represents a response to a direct message.
type DirectMessageResponse struct {
	MessageID string `json:"message_id"`
	Response  string `json:"response"`
	Error     string `json:"error,omitempty"`
}

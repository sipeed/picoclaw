// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

// SessionMessage represents a message in a session.
// This is shared across handoff and session transfer.
type SessionMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	Timestamp int64         `json:"timestamp,omitempty"`
	ToolCalls []ToolCallData `json:"tool_calls,omitempty"`
}

// ToolCallData represents tool call information in a message.
type ToolCallData struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Arguments map[string]any        `json:"arguments"`
	Result   string                 `json:"result,omitempty"`
	Extra    map[string]any         `json:"extra,omitempty"`
}

// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// MessageReceivedEvent is fired when an inbound message is consumed from the bus.
type MessageReceivedEvent struct {
	Channel  string
	SenderID string
	ChatID   string
	Content  string
	Media    []string
	Metadata map[string]string
}

// MessageSendingEvent is fired before an outbound message is published.
// Handlers can modify Content or set Cancel to block delivery.
type MessageSendingEvent struct {
	Channel      string
	ChatID       string
	Content      string // Modifiable
	Cancel       bool
	CancelReason string
}

// BeforeToolCallEvent is fired before a tool is executed.
// Handlers can modify Args, or set Cancel to block execution.
type BeforeToolCallEvent struct {
	ToolName     string
	Args         map[string]any // Modifiable
	Channel      string
	ChatID       string
	Cancel       bool
	CancelReason string // Message returned to LLM when canceled
}

// AfterToolCallEvent is fired after a tool completes execution.
type AfterToolCallEvent struct {
	ToolName string
	Args     map[string]any
	Channel  string
	ChatID   string
	Duration time.Duration
	Result   *tools.ToolResult
}

// LLMInputEvent is fired before the LLM provider is called.
type LLMInputEvent struct {
	AgentID   string
	Model     string
	Messages  []providers.Message
	Tools     []providers.ToolDefinition
	Iteration int
}

// LLMOutputEvent is fired after the LLM provider responds.
type LLMOutputEvent struct {
	AgentID   string
	Model     string
	Content   string
	ToolCalls []providers.ToolCall
	Iteration int
	Duration  time.Duration
}

// SessionEvent is fired at session start and end.
type SessionEvent struct {
	AgentID    string
	SessionKey string
	Channel    string
	ChatID     string
}

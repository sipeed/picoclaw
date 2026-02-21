// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package tui

import "github.com/sipeed/picoclaw/pkg/agent"

// ThinkingStartedMsg indicates the LLM is processing.
type ThinkingStartedMsg struct{}

// ToolCallStartedMsg indicates a tool execution has begun.
type ToolCallStartedMsg struct {
	ID   string
	Name string
	Args string // JSON string
}

// ToolCallCompletedMsg indicates a tool execution has finished.
type ToolCallCompletedMsg struct {
	ID      string
	Name    string
	Result  string
	IsError bool
}

// ResponseMsg carries the final LLM response text.
type ResponseMsg struct {
	Content string
}

// ErrorMsg carries an error from the agent loop.
type ErrorMsg struct {
	Err error
}

// SlashCommandResultMsg carries the result of a slash command.
type SlashCommandResultMsg struct {
	Result string
}

// eventBridge adapts AgentEventListener to send tea.Msg via a channel.
type eventBridge struct {
	events chan agent.AgentEvent
}

func newEventBridge() *eventBridge {
	return &eventBridge{
		events: make(chan agent.AgentEvent, 50),
	}
}

func (eb *eventBridge) OnEvent(event agent.AgentEvent) {
	select {
	case eb.events <- event:
	default:
		// Drop event if channel is full (TUI not consuming fast enough)
	}
}

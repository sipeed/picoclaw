// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package agent

// AgentEventType identifies the kind of agent lifecycle event
type AgentEventType int

const (
	// EventThinkingStarted fires before calling the LLM
	EventThinkingStarted AgentEventType = iota
	// EventToolCallStarted fires before executing a tool
	EventToolCallStarted
	// EventToolCallCompleted fires after a tool execution finishes
	EventToolCallCompleted
	// EventResponseComplete fires when the agent produces a final text response
	EventResponseComplete
	// EventError fires when the agent loop encounters an error
	EventError
)

// AgentEvent represents a lifecycle event emitted by the agent loop
type AgentEvent struct {
	Type AgentEventType
	Data any
}

// ToolCallStartedData carries information about a tool call that is about to execute
type ToolCallStartedData struct {
	ID   string
	Name string
	Args string
}

// ToolCallCompletedData carries information about a completed tool call
type ToolCallCompletedData struct {
	ID      string
	Name    string
	Result  string
	IsError bool
}

// ResponseCompleteData carries the final response content from the agent
type ResponseCompleteData struct {
	Content string
}

// ErrorData carries error information from the agent loop
type ErrorData struct {
	Err error
}

// AgentEventListener receives lifecycle events from the agent loop.
// Implementations must be safe for concurrent use, as events fire from
// the agent loop goroutine while the caller may run on a different goroutine.
type AgentEventListener interface {
	OnEvent(event AgentEvent)
}

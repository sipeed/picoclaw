package audit

import (
	"context"
	"time"
)

// Entry represents a single audit log entry.
// All fields are JSON-serializable for structured logging.
type Entry struct {
	// Core metadata
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Component string    `json:"component"`
	EventType EventType `json:"event_type"`

	// Request context for correlation
	RequestID string `json:"request_id"`
	SessionID string `json:"session_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`

	// Channel context
	Channel string `json:"channel,omitempty"`
	ChatID  string `json:"chat_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`

	// Event-specific data
	ToolCall *ToolCallData `json:"tool_call,omitempty"`
	Message  *MessageData  `json:"message,omitempty"`
	Error    *ErrorData    `json:"error,omitempty"`
	System   *SystemData   `json:"system,omitempty"`

	// Performance metrics
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// Level represents the severity of an audit entry.
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// EventType categorizes audit entries for filtering and analysis.
type EventType string

const (
	EventToolCall EventType = "tool_call"
	EventMessage  EventType = "message"
	EventError    EventType = "error"
	EventSystem   EventType = "system"
)

// ToolCallData captures details of a tool execution.
type ToolCallData struct {
	ToolID    string                 `json:"tool_id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    string                 `json:"result,omitempty"`
	IsError   bool                   `json:"is_error"`
	IsAsync   bool                   `json:"is_async"`
}

// MessageData captures message flow details.
type MessageData struct {
	Direction   string `json:"direction"`    // "inbound" or "outbound"
	ContentType string `json:"content_type"` // "text", "media", "command"
	Content     string `json:"content,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
}

// ErrorData captures error details for debugging.
type ErrorData struct {
	ErrorType   string `json:"error_type"`
	Message     string `json:"message"`
	StackTrace  string `json:"stack_trace,omitempty"`
	Recoverable bool   `json:"recoverable"`
}

// SystemData captures system-level events.
type SystemData struct {
	Operation string                 `json:"operation"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey int

const (
	requestIDKey contextKey = iota
	sessionIDKey
	agentIDKey
)

// WithRequestID returns a context with the request ID set.
// The request ID is used to correlate all audit entries from a single request.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithSessionID returns a context with the session ID set.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// SessionIDFromContext extracts the session ID from the context.
func SessionIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(sessionIDKey).(string); ok {
		return id
	}
	return ""
}

// WithAgentID returns a context with the agent ID set.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

// AgentIDFromContext extracts the agent ID from the context.
func AgentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(agentIDKey).(string); ok {
		return id
	}
	return ""
}

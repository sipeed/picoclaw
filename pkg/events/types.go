package events

import "time"

// Kind identifies a runtime event category.
type Kind string

// String returns the string representation of the event kind.
func (k Kind) String() string {
	return string(k)
}

// Event is the runtime event envelope shared across PicoClaw components.
type Event struct {
	ID          string         `json:"id"`
	Kind        Kind           `json:"kind"`
	Time        time.Time      `json:"time"`
	Source      Source         `json:"source"`
	Scope       Scope          `json:"scope,omitempty"`
	Correlation Correlation    `json:"correlation,omitempty"`
	Severity    Severity       `json:"severity,omitempty"`
	Payload     any            `json:"payload,omitempty"`
	Attrs       map[string]any `json:"attrs,omitempty"`
}

// Source identifies the component that emitted an event.
type Source struct {
	Component string `json:"component"`
	Name      string `json:"name,omitempty"`
}

// Scope identifies the runtime ownership of an event.
//
// Scope is intentionally limited to agent, session, turn, channel, chat,
// message, and sender identity. Tool, provider, model, and MCP details belong
// in Source, Payload, or Attrs.
type Scope struct {
	RuntimeID string `json:"runtime_id,omitempty"`

	AgentID    string `json:"agent_id,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	TurnID     string `json:"turn_id,omitempty"`

	Channel string `json:"channel,omitempty"`
	Account string `json:"account,omitempty"`
	ChatID  string `json:"chat_id,omitempty"`
	TopicID string `json:"topic_id,omitempty"`

	SpaceID   string `json:"space_id,omitempty"`
	SpaceType string `json:"space_type,omitempty"`
	ChatType  string `json:"chat_type,omitempty"`

	SenderID  string `json:"sender_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

// Correlation carries cross-event tracing fields.
type Correlation struct {
	TraceID      string `json:"trace_id,omitempty"`
	ParentTurnID string `json:"parent_turn_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	ReplyToID    string `json:"reply_to_id,omitempty"`
}

// Severity describes the operational severity of an event.
type Severity string

const (
	// SeverityDebug is used for verbose diagnostic events.
	SeverityDebug Severity = "debug"
	// SeverityInfo is used for normal lifecycle and activity events.
	SeverityInfo Severity = "info"
	// SeverityWarn is used for recoverable abnormal events.
	SeverityWarn Severity = "warn"
	// SeverityError is used for failed operations and unrecoverable events.
	SeverityError Severity = "error"
)

package bus

type InboundMessage struct {
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	SessionKey string            `json:"session_key"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type OutboundMessage struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

type MessageHandler func(InboundMessage) error

// StreamEventType defines the type of streaming event
type StreamEventType string

const (
	StreamEventThinking   StreamEventType = "thinking"    // AI is thinking/reasoning
	StreamEventToolCall   StreamEventType = "tool_call"   // About to execute a tool
	StreamEventToolResult StreamEventType = "tool_result" // Tool execution completed
	StreamEventProgress   StreamEventType = "progress"    // Progress update
	StreamEventContent    StreamEventType = "content"     // Partial content streaming
	StreamEventComplete   StreamEventType = "complete"    // Processing complete
	StreamEventError      StreamEventType = "error"       // Error occurred
)

// StreamEvent represents a streaming update during agent processing
type StreamEvent struct {
	Type       StreamEventType        `json:"type"`
	Channel    string                 `json:"channel"`
	ChatID     string                 `json:"chat_id"`
	SessionKey string                 `json:"session_key"`
	Content    string                 `json:"content,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
}

// InterruptSignal represents a user interrupt request
type InterruptSignal struct {
	Channel    string `json:"channel"`
	ChatID     string `json:"chat_id"`
	SessionKey string `json:"session_key"`
}

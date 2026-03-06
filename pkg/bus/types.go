package bus

// Peer identifies the routing peer for a message (direct, group, channel, etc.)
type Peer struct {
	Kind string `json:"kind"` // "direct" | "group" | "channel" | ""
	ID   string `json:"id"`
}

// SenderInfo provides structured sender identity information.
type SenderInfo struct {
	Platform    string `json:"platform,omitempty"`     // "telegram", "discord", "slack", ...
	PlatformID  string `json:"platform_id,omitempty"`  // raw platform ID, e.g. "123456"
	CanonicalID string `json:"canonical_id,omitempty"` // "platform:id" format
	Username    string `json:"username,omitempty"`     // username (e.g. @alice)
	DisplayName string `json:"display_name,omitempty"` // display name
}

type InboundMessage struct {
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	Sender     SenderInfo        `json:"sender"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	Peer       Peer              `json:"peer"`                  // routing peer
	MessageID  string            `json:"message_id,omitempty"`  // platform message ID
	MediaScope string            `json:"media_scope,omitempty"` // media lifecycle scope
	SessionKey string            `json:"session_key"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Outbound message type constants identify the kind of outbound message.
// Channels that don't support progress/escalation treat all messages as content.
const (
	MessageTypeFinal      = ""           // default: final response
	MessageTypeProgress   = "progress"   // intermediate progress update
	MessageTypeEscalation = "escalation" // needs human intervention
)

type OutboundMessage struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`

	// Type distinguishes final responses from progress/escalation updates.
	// Empty string (default) means final response. Channels that don't support
	// progress callbacks ignore this field and treat all messages as content.
	Type string `json:"type,omitempty"`

	// Metrics is populated by the agent loop for the final response message.
	// Channels that support rich callbacks (e.g. MagicForm) use this to include
	// execution metrics in their response payloads. Nil for intermediate messages.
	Metrics *ResponseMetrics `json:"metrics,omitempty"`

	// Progress is populated for Type="progress" messages.
	Progress *OutboundProgress `json:"progress,omitempty"`

	// Escalation is populated for Type="escalation" messages.
	Escalation *OutboundEscalation `json:"escalation,omitempty"`
}

// OutboundProgress carries progress update details.
type OutboundProgress struct {
	Status     string `json:"status"`               // e.g. "thinking"
	ToolName   string `json:"toolName,omitempty"`
	StepNumber int    `json:"stepNumber,omitempty"`
	Message    string `json:"message,omitempty"`
}

// OutboundEscalation carries escalation details.
type OutboundEscalation struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes,omitempty"`
}

// TokenUsage tracks LLM token consumption across all iterations in a turn.
type TokenUsage struct {
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	TotalTokens      int    `json:"totalTokens"`
	Model            string `json:"model"`
	Provider         string `json:"provider,omitempty"`
}

// ResponseMetrics captures execution metrics for a single agent processing turn.
type ResponseMetrics struct {
	DurationMs int64       `json:"durationMs"`
	TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`
	ToolCalls  int         `json:"toolCalls"`
	Iterations int         `json:"iterations"`
	Model      string      `json:"model,omitempty"`
}

// MediaPart describes a single media attachment to send.
type MediaPart struct {
	Type        string `json:"type"`                   // "image" | "audio" | "video" | "file"
	Ref         string `json:"ref"`                    // media store ref, e.g. "media://abc123"
	Caption     string `json:"caption,omitempty"`      // optional caption text
	Filename    string `json:"filename,omitempty"`     // original filename hint
	ContentType string `json:"content_type,omitempty"` // MIME type hint
}

// OutboundMediaMessage carries media attachments from Agent to channels via the bus.
type OutboundMediaMessage struct {
	Channel string      `json:"channel"`
	ChatID  string      `json:"chat_id"`
	Parts   []MediaPart `json:"parts"`
}

package bus

// SenderInfo provides structured sender identity information.
type SenderInfo struct {
	Platform    string `json:"platform,omitempty"`     // "telegram", "discord", "slack", ...
	PlatformID  string `json:"platform_id,omitempty"`  // raw platform ID, e.g. "123456"
	CanonicalID string `json:"canonical_id,omitempty"` // "platform:id" format
	Username    string `json:"username,omitempty"`     // username (e.g. @alice)
	DisplayName string `json:"display_name,omitempty"` // display name
}

// InboundContext captures the normalized, platform-agnostic facts about an
// inbound message. This is the source of truth for routing and session
// allocation.
type InboundContext struct {
	Channel     string `json:"channel"`
	ChannelType string `json:"channel_type,omitempty"` // telegram, discord, slack, etc.
	Account     string `json:"account,omitempty"`

	ChatID   string `json:"chat_id"`
	ChatType string `json:"chat_type,omitempty"` // direct / group / channel
	TopicID  string `json:"topic_id,omitempty"`

	SpaceID   string `json:"space_id,omitempty"`
	SpaceType string `json:"space_type,omitempty"` // guild / team / workspace / tenant

	SenderID  string `json:"sender_id"`
	MessageID string `json:"message_id,omitempty"`

	Mentioned bool `json:"mentioned,omitempty"`

	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	ReplyToSenderID  string `json:"reply_to_sender_id,omitempty"`

	ReplyHandles map[string]string `json:"reply_handles,omitempty"`
	Raw          map[string]string `json:"raw,omitempty"`
}

type InboundMessage struct {
	Context    InboundContext `json:"context"`
	Sender     SenderInfo     `json:"sender"`
	Content    string         `json:"content"`
	Media      []string       `json:"media,omitempty"`
	MediaScope string         `json:"media_scope,omitempty"` // media lifecycle scope
	SessionKey string         `json:"session_key"`

	// Convenience mirrors derived from Context for runtime consumers.
	Channel   string `json:"channel"`
	SenderID  string `json:"sender_id"`
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id,omitempty"` // platform message ID
}

// OutboundScope captures the structured session scope associated with an
// outbound turn result without depending on the session package.
type OutboundScope struct {
	Version    int               `json:"version,omitempty"`
	AgentID    string            `json:"agent_id,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Account    string            `json:"account,omitempty"`
	Dimensions []string          `json:"dimensions,omitempty"`
	Values     map[string]string `json:"values,omitempty"`
}

// ContextUsage describes how much of the model's context window the current
// session consumes, and how far it is from triggering compression.
type ContextUsage struct {
	UsedTokens       int `json:"used_tokens"`
	TotalTokens      int `json:"total_tokens"`       // model context window
	CompressAtTokens int `json:"compress_at_tokens"` // threshold that triggers compression
	UsedPercent      int `json:"used_percent"`       // 0-100
}

type OutboundMessage struct {
	Channel          string         `json:"channel"`
	ChatID           string         `json:"chat_id"`
	Context          InboundContext `json:"context"`
	AgentID          string         `json:"agent_id,omitempty"`
	SessionKey       string         `json:"session_key,omitempty"`
	Scope            *OutboundScope `json:"scope,omitempty"`
	Content          string         `json:"content"`
	ReplyToMessageID string         `json:"reply_to_message_id,omitempty"`
	ContextUsage     *ContextUsage  `json:"context_usage,omitempty"`
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
	Channel    string         `json:"channel"`
	ChatID     string         `json:"chat_id"`
	Context    InboundContext `json:"context"`
	AgentID    string         `json:"agent_id,omitempty"`
	SessionKey string         `json:"session_key,omitempty"`
	Scope      *OutboundScope `json:"scope,omitempty"`
	Parts      []MediaPart    `json:"parts"`
}

// AudioChunk represents a chunk of streaming voice data.
type AudioChunk struct {
	SessionID  string `json:"session_id"`
	SpeakerID  string `json:"speaker_id"` // User ID or SSRC
	ChatID     string `json:"chat_id"`    // Where to respond
	Channel    string `json:"channel"`    // Source channel type (e.g. "discord")
	Sequence   uint64 `json:"sequence"`
	Timestamp  uint32 `json:"timestamp"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"` // "opus", "pcm", etc
	Data       []byte `json:"data"`
}

// VoiceControl represents state or commands for voice sessions.
type VoiceControl struct {
	SessionID string `json:"session_id"`
	ChatID    string `json:"chat_id"`
	Type      string `json:"type"`   // "state", "command"
	Action    string `json:"action"` // "idle", "listening", "start", "stop", "leave"
}

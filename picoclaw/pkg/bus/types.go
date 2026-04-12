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

type OutboundMessage struct {
	Channel          string            `json:"channel"`
	ChatID           string            `json:"chat_id"`
	Content          string            `json:"content"`
	ReplyToMessageID string            `json:"reply_to_message_id,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
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

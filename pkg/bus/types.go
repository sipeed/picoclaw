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

// OutboundMessage is sent from the agent loop to channels (e.g. web gateway).
// State and RunID support streaming: "streaming" for chunks, "final" for complete reply.
type OutboundMessage struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
	State   string `json:"state,omitempty"`   // e.g. "streaming", "final"
	RunID   string `json:"run_id,omitempty"`  // idempotency/run identifier for the turn
}

type MessageHandler func(InboundMessage) error

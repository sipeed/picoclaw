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

// InboundInterceptor inspects an inbound message before it reaches the main consumer.
// Returns true if the message was consumed and should not be enqueued.
type InboundInterceptor func(msg InboundMessage) bool

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

type Attachment struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

type OutboundMessage struct {
	Channel     string       `json:"channel"`
	ChatID      string       `json:"chat_id"`
	Content     string       `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type MessageHandler func(InboundMessage) error

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
	Type     string `json:"type"` // image | file | audio | video
	Path     string `json:"path,omitempty"`
	URL      string `json:"url,omitempty"`
	FileName string `json:"file_name,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
}

type OutboundMessage struct {
	Channel     string       `json:"channel"`
	ChatID      string       `json:"chat_id"`
	Content     string       `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type MessageHandler func(InboundMessage) error

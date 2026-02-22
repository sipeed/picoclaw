package pico

import "time"

// Protocol message types.
const (
	// Client → Server
	TypeMessageSend = "message.send"
	TypeMediaSend   = "media.send"
	TypePing        = "ping"

	// Server → Client
	TypeMessageCreate = "message.create"
	TypeMessageUpdate = "message.update"
	TypeMediaCreate   = "media.create"
	TypeTypingStart   = "typing.start"
	TypeTypingStop    = "typing.stop"
	TypeError         = "error"
	TypePong          = "pong"
)

// PicoMessage is the wire format for all Pico Protocol messages.
type PicoMessage struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// newMessage creates a PicoMessage with the given type and payload.
func newMessage(msgType string, payload map[string]any) PicoMessage {
	return PicoMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

// newError creates an error PicoMessage.
func newError(code, message string) PicoMessage {
	return newMessage(TypeError, map[string]any{
		"code":    code,
		"message": message,
	})
}

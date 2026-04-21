package pico

import (
	"time"

	"github.com/google/uuid"
)

// Protocol message types.
const (
	// TypeMessageSend is sent from client to server.
	TypeMessageSend = "message.send"
	TypeMediaSend   = "media.send"
	TypePing        = "ping"

	// TypeMessageCreate is sent from server to client.
	TypeMessageCreate = "message.create"
	TypeMessageUpdate = "message.update"
	TypeMediaCreate   = "media.create"
	TypeTypingStart   = "typing.start"
	TypeTypingStop    = "typing.stop"
	TypeError         = "error"
	TypePong          = "pong"

	PicoTokenPrefix = "pico-"

	PayloadKeyContent    = "content"
	PayloadKeyThought    = "thought"
	PayloadKeyStructured = "structured"
	PayloadKeyMode       = "mode"

	MessageKindThought = "thought"

	ChatModeAgent = "agent"
	ChatModeAsk   = "ask"
	ChatModePlan  = "plan"
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
	if msgType == TypeMessageCreate {
		if payload == nil {
			payload = make(map[string]any, 1)
		}
		if _, exists := payload["message_id"]; !exists {
			payload["message_id"] = uuid.NewString()
		}
	}

	return PicoMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

func isThoughtPayload(payload map[string]any) bool {
	thought, _ := payload[PayloadKeyThought].(bool)
	return thought
}

func newErrorWithPayload(code, message string, extra map[string]any) PicoMessage {
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	for key, value := range extra {
		payload[key] = value
	}
	return newMessage(TypeError, payload)
}

// newError creates an error PicoMessage.
func newError(code, message string) PicoMessage {
	return newErrorWithPayload(code, message, nil)
}

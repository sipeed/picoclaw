package pico

import (
	"strings"
	"time"
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
	TypeMessageDelete = "message.delete"
	TypeMediaCreate   = "media.create"
	TypeTypingStart   = "typing.start"
	TypeTypingStop    = "typing.stop"
	TypeError         = "error"
	TypePong          = "pong"

	PayloadKeyContent   = "content"
	PayloadKeyThought   = "thought"
	PayloadKeyKind      = "kind"
	PayloadKeyToolCalls = "tool_calls"

	MessageKindThought   = "thought"
	MessageKindToolCalls = "tool_calls"
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

func isThoughtPayload(payload map[string]any) bool {
	kind, _ := payload[PayloadKeyKind].(string)
	if strings.EqualFold(strings.TrimSpace(kind), MessageKindThought) {
		return true
	}

	// Keep pico_client inbound-compatible with legacy servers that still send
	// the pre-kind boolean thought marker.
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

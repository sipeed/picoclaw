// Package protocol defines the Pico Protocol wire format shared by
// the Pico channel (server) and the swarm PicoNodeClient (client).
// This package has zero internal dependencies to stay at the bottom
// of the dependency graph.
package protocol

import "time"

// Message type constants for the Pico Protocol.
const (
	// TypeMessageSend is sent from client to server.
	TypeMessageSend = "message.send"
	// TypeMediaSend is sent from client to server for media.
	TypeMediaSend = "media.send"
	// TypePing is a client ping.
	TypePing = "ping"

	// TypeMessageCreate is sent from server to client.
	TypeMessageCreate = "message.create"
	// TypeMessageUpdate is sent from server to client for message updates.
	TypeMessageUpdate = "message.update"
	// TypeMediaCreate is sent from server to client for media.
	TypeMediaCreate = "media.create"
	// TypeTypingStart indicates typing has started.
	TypeTypingStart = "typing.start"
	// TypeTypingStop indicates typing has stopped.
	TypeTypingStop = "typing.stop"
	// TypeError is an error message.
	TypeError = "error"
	// TypePong is a server pong reply.
	TypePong = "pong"

	// TypeNodeRequest is for inter-node swarm communication.
	TypeNodeRequest = "node.request"
	// TypeNodeReply is the reply for inter-node swarm communication.
	TypeNodeReply = "node.reply"
)

// Message is the wire format for all Pico Protocol messages.
type Message struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// NewMessage creates a Message with the given type, payload, and current timestamp.
func NewMessage(msgType string, payload map[string]any) Message {
	return Message{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

// NewError creates an error Message with code and human-readable message.
func NewError(code, message string) Message {
	return NewMessage(TypeError, map[string]any{
		"code":    code,
		"message": message,
	})
}

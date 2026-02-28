// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package pico

// NodePayload is the typed payload exchanged between nodes via the
// Pico channel node.request / node.reply protocol. It provides named
// constants for field keys and typed accessor methods so that callers
// never need to use raw string literals.
type NodePayload map[string]any

// Payload field key constants.
const (
	PayloadKeyAction       = "action"
	PayloadKeyRequestID    = "request_id"
	PayloadKeySourceNodeID = "source_node_id"
	PayloadKeyContent      = "content"
	PayloadKeyChannel      = "channel"
	PayloadKeyChatID       = "chat_id"
	PayloadKeySenderID     = "sender_id"
	PayloadKeyMetadata     = "metadata"
	PayloadKeyError        = "error"
	PayloadKeyResponse     = "response"
	PayloadKeyRequest      = "request"          // nested handoff request object
	PayloadKeyHandoffResp  = "handoff_response" // nested handoff response object
)

// Node action constants used for action-based routing over Pico.
const (
	NodeActionMessage        = "message"
	NodeActionHandoffRequest = "handoff_request"
)

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// str is a small helper that extracts a string value from the payload.
func (p NodePayload) str(key string) string {
	v, _ := p[key].(string)
	return v
}

// Action returns the action field (e.g. "message", "handoff_request").
func (p NodePayload) Action() string { return p.str(PayloadKeyAction) }

// RequestID returns the request_id field.
func (p NodePayload) RequestID() string { return p.str(PayloadKeyRequestID) }

// SourceNodeID returns the source_node_id field.
func (p NodePayload) SourceNodeID() string { return p.str(PayloadKeySourceNodeID) }

// Content returns the content field.
func (p NodePayload) Content() string { return p.str(PayloadKeyContent) }

// Channel returns the channel field.
func (p NodePayload) Channel() string { return p.str(PayloadKeyChannel) }

// ChatID returns the chat_id field.
func (p NodePayload) ChatID() string { return p.str(PayloadKeyChatID) }

// SenderID returns the sender_id field.
func (p NodePayload) SenderID() string { return p.str(PayloadKeySenderID) }

// ErrorMsg returns the error field.
func (p NodePayload) ErrorMsg() string { return p.str(PayloadKeyError) }

// Response returns the response field.
func (p NodePayload) Response() string { return p.str(PayloadKeyResponse) }

// Metadata extracts the metadata map, converting map[string]any to map[string]string.
func (p NodePayload) Metadata() map[string]string {
	raw, ok := p[PayloadKeyMetadata]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// RawValue returns the raw value for an arbitrary key.
func (p NodePayload) RawValue(key string) (any, bool) {
	v, ok := p[key]
	return v, ok
}

// ---------------------------------------------------------------------------
// Builder helpers
// ---------------------------------------------------------------------------

// NewNodePayload creates an empty NodePayload.
func NewNodePayload() NodePayload {
	return make(NodePayload)
}

// NewMessagePayload creates a NodePayload pre-filled for a "message" action.
func NewMessagePayload(sourceNodeID, content, channel, chatID, senderID string) NodePayload {
	return NodePayload{
		PayloadKeyAction:       NodeActionMessage,
		PayloadKeySourceNodeID: sourceNodeID,
		PayloadKeyContent:      content,
		PayloadKeyChannel:      channel,
		PayloadKeyChatID:       chatID,
		PayloadKeySenderID:     senderID,
	}
}

// ---------------------------------------------------------------------------
// Reply constructors
// ---------------------------------------------------------------------------

// ErrorReply creates a reply payload carrying an error message.
func ErrorReply(msg string) NodePayload {
	return NodePayload{PayloadKeyError: msg}
}

// ResponseReply creates a reply payload carrying a string response.
func ResponseReply(response string) NodePayload {
	return NodePayload{PayloadKeyResponse: response}
}

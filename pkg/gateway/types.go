// Package gateway implements the WebSocket Gateway protocol for WebClaw/OpenClaw compatibility.
package gateway

import "encoding/json"

// GatewayFrame is the JSON wire format: req (method+params), res (ok+payload/error), or event.
type GatewayFrame struct {
	Type         string          `json:"type"`
	ID           string          `json:"id,omitempty"`
	Method       string          `json:"method,omitempty"`
	Params       json.RawMessage `json:"params,omitempty"`
	Event        string          `json:"event,omitempty"`
	Payload      interface{}     `json:"payload,omitempty"`
	Seq          int             `json:"seq,omitempty"`
	StateVersion int             `json:"stateVersion,omitempty"`
	Ok           bool            `json:"ok,omitempty"`
	Error        *GatewayError   `json:"error,omitempty"`
}

// GatewayError is the error payload in a res frame.
type GatewayError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ConnectParams is the params for the connect method (WebClaw sends auth.token / auth.password).
type ConnectParams struct {
	Auth struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	} `json:"auth"`
}

// SessionsListParams for sessions.list.
type SessionsListParams struct {
	Limit                  int  `json:"limit"`
	IncludeLastMessage     bool `json:"includeLastMessage"`
	IncludeDerivedTitles   bool `json:"includeDerivedTitles"`
}

// SessionsPatchParams for sessions.patch (create or update).
type SessionsPatchParams struct {
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
}

// SessionsResolveParams for sessions.resolve.
type SessionsResolveParams struct {
	Key           string `json:"key"`
	IncludeUnknown bool  `json:"includeUnknown"`
	IncludeGlobal  bool  `json:"includeGlobal"`
}

// SessionsDeleteParams for sessions.delete.
type SessionsDeleteParams struct {
	Key string `json:"key"`
}

// ChatSendParams for chat.send.
type ChatSendParams struct {
	SessionKey     string   `json:"sessionKey"`
	Message        string   `json:"message"`
	Thinking       string   `json:"thinking,omitempty"`
	Attachments    []any    `json:"attachments,omitempty"`
	Deliver        bool     `json:"deliver"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	IdempotencyKey string   `json:"idempotencyKey,omitempty"`
}

// ChatHistoryParams for chat.history.
type ChatHistoryParams struct {
	SessionKey string `json:"sessionKey"`
	Limit      int    `json:"limit"`
}

// ChatSubscribeParams for chat.subscribe.
type ChatSubscribeParams struct {
	SessionKey string `json:"sessionKey,omitempty"`
	FriendlyId string `json:"friendlyId,omitempty"`
}

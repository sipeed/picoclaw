package gateway

import "encoding/json"

// Protocol version (see openclaw src/gateway/protocol/schema.ts)
const ProtocolVersion = 3

// Frame types
const (
	FrameTypeReq  = "req"
	FrameTypeRes  = "res"
	FrameTypeEvent = "event"
)

// Events
const (
	EventConnectChallenge = "connect.challenge"
	EventNodeInvokeReq    = "node.invoke.request"
)

// Methods
const (
	MethodConnect        = "connect"
	MethodNodeEvent      = "node.event"
	MethodNodeInvokeRes  = "node.invoke.result"
)

// ConnectParams for node role
type ConnectParams struct {
	MinProtocol int                 `json:"minProtocol"`
	MaxProtocol int                 `json:"maxProtocol"`
	Client      ClientInfo          `json:"client"`
	Role        string              `json:"role"`
	Scopes      []string            `json:"scopes,omitempty"`
	Caps        []string            `json:"caps,omitempty"`
	Commands    []string            `json:"commands,omitempty"`
	Permissions map[string]bool     `json:"permissions,omitempty"`
	Auth        *AuthParams         `json:"auth,omitempty"`
	Device      *DeviceAuth         `json:"device,omitempty"`
	Locale      string              `json:"locale"`
	UserAgent   string              `json:"userAgent,omitempty"`
}

type ClientInfo struct {
	ID               string  `json:"id"`
	DisplayName      *string `json:"displayName,omitempty"`
	Version          string  `json:"version"`
	Platform         string  `json:"platform"`
	Mode             string  `json:"mode"`
	InstanceID       *string `json:"instanceId,omitempty"`
	DeviceFamily     *string `json:"deviceFamily,omitempty"`
	ModelIdentifier  *string `json:"modelIdentifier,omitempty"`
}

type AuthParams struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
}

type DeviceAuth struct {
	ID        string `json:"id"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
	SignedAt  int64  `json:"signedAt"`
	Nonce     string `json:"nonce"`
}

type RequestFrame struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type ResponseFrame struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	OK      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *ErrorShape     `json:"error,omitempty"`
}

type EventFrame struct {
	Type    string          `json:"type"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Seq     *int64          `json:"seq,omitempty"`
}

type ErrorShape struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

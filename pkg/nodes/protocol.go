// Package nodes implements the openclaw gateway WebSocket protocol server,
// allowing openclaw nodes (headless Linux, iOS, Android, macOS) to connect
// directly to picoclaw and expose their capabilities.
package nodes

// ProtocolVersion is the openclaw gateway protocol version this server supports.
const ProtocolVersion = 3

// TickIntervalMs is how often (ms) the server sends a tick keepalive event.
// Must match openclaw's TICK_INTERVAL_MS = 30_000.
// The node client's watchdog closes the connection if no tick arrives within
// tickIntervalMs * 2 (i.e. 60 s), so we must send one every 30 s.
const TickIntervalMs = 30_000

// MaxPayloadBytes is the maximum allowed incoming frame size (512 KB).
const MaxPayloadBytes = 512 * 1024

// MaxBufferedBytes is the per-connection send buffer limit (1.5 MB).
const MaxBufferedBytes = 1536 * 1024

// Frame type constants matching the openclaw gateway protocol.
const (
	FrameTypeReq   = "req"
	FrameTypeRes   = "res"
	FrameTypeEvent = "event"
)

// Method constants used in the protocol.
const (
	MethodConnect          = "connect"
	MethodNodeInvokeResult = "node.invoke.result"
	MethodNodeEvent        = "node.event"
)

// Event constants sent from server to node.
const (
	EventConnectChallenge  = "connect.challenge"
	EventNodeInvokeRequest = "node.invoke.request"
)

// Error code constants matching openclaw's ErrorCodes.
const (
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeNotPaired      = "NOT_PAIRED"
	ErrCodeUnavailable    = "UNAVAILABLE"
	ErrCodeTimeout        = "TIMEOUT"
)

// ReqFrame is a JSON-RPC style request frame sent by clients.
// Example: {"type":"req","id":"<uuid>","method":"connect","params":{...}}
type ReqFrame struct {
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// ResFrame is a response frame sent by the server.
// Example: {"type":"res","id":"<uuid>","ok":true,"payload":{...}}
type ResFrame struct {
	Type    string   `json:"type"`
	ID      string   `json:"id"`
	Ok      bool     `json:"ok"`
	Payload any      `json:"payload,omitempty"`
	Error   *ErrBody `json:"error,omitempty"`
}

// EventFrame is an event frame sent by the server to nodes.
// Example: {"type":"event","event":"node.invoke.request","payload":{...}}
type EventFrame struct {
	Type    string `json:"type"`
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
}

// ErrBody is the error body in a response frame.
type ErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ConnectChallengePayload is the payload of the connect.challenge event.
// The server sends this immediately upon connection.
type ConnectChallengePayload struct {
	Nonce string `json:"nonce"`
	Ts    int64  `json:"ts"`
}

// ConnectParams is the parameters of the connect request sent by nodes.
// Matches the openclaw gateway protocol ConnectParams structure.
type ConnectParams struct {
	// Protocol version range supported by the client.
	MinProtocol int `json:"minProtocol"`
	MaxProtocol int `json:"maxProtocol"`

	// Role: "node" or "operator"
	Role string `json:"role"`

	// Scopes requested by the client.
	Scopes []string `json:"scopes,omitempty"`

	// Client identity information.
	Client ConnectClientInfo `json:"client"`

	// Optional auth credentials.
	Auth *ConnectAuth `json:"auth,omitempty"`

	// Device identity (for nodes with a persistent device ID / key pair).
	Device *ConnectDevice `json:"device,omitempty"`

	// Commands this node can handle (only valid for role=node).
	Commands []string `json:"commands,omitempty"`

	// Capabilities this node exposes (e.g. "system", "camera").
	Caps []string `json:"caps,omitempty"`

	// Path env from the node host (for headless Linux nodes).
	PathEnv string `json:"pathEnv,omitempty"`
}

// ConnectClientInfo holds identifying information about the connecting client.
type ConnectClientInfo struct {
	// Unique stable client identifier (e.g. "openclaw-node-host").
	ID string `json:"id"`

	// Human-readable display name (e.g. "My Raspberry Pi").
	DisplayName string `json:"displayName,omitempty"`

	// Platform: "linux", "darwin", "win32", "ios", "android".
	Platform string `json:"platform,omitempty"`

	// Device family: "phone", "tablet", "desktop", "server", etc.
	DeviceFamily string `json:"deviceFamily,omitempty"`

	// Hardware model identifier (e.g. "iPhone16,2").
	ModelIdentifier string `json:"modelIdentifier,omitempty"`

	// Client mode: "node", "operator", etc.
	Mode string `json:"mode,omitempty"`

	// Client version string.
	Version string `json:"version,omitempty"`

	// Instance ID: stable identifier for this running instance.
	InstanceID string `json:"instanceId,omitempty"`
}

// ConnectAuth holds authentication credentials.
type ConnectAuth struct {
	// Shared token (set in gateway / picoclaw config).
	Token string `json:"token,omitempty"`

	// Password auth (alternative to token).
	Password string `json:"password,omitempty"`
}

// ConnectDevice holds the cryptographic device identity of the connecting node.
// picoclaw uses this for pairing tracking; signature verification is optional.
type ConnectDevice struct {
	// Stable device ID derived from the public key.
	ID string `json:"id"`

	// Base64url-encoded Ed25519 public key.
	PublicKey string `json:"publicKey"`

	// Base64url-encoded signature over the auth payload.
	Signature string `json:"signature,omitempty"`

	// Timestamp (ms since epoch) when the signature was created.
	SignedAt int64 `json:"signedAt"`

	// Nonce from the connect.challenge event (for replay protection).
	Nonce string `json:"nonce,omitempty"`
}

// HelloOkPayload is the payload returned in the successful connect response.
type HelloOkPayload struct {
	Type     string          `json:"type"`
	Protocol int             `json:"protocol"`
	Server   HelloOkServer   `json:"server"`
	Features HelloOkFeatures `json:"features"`
	// Policy tells the client the tick interval so its watchdog is calibrated.
	Policy HelloOkPolicy `json:"policy"`
}

// HelloOkPolicy carries server-enforced connection policy sent in hello-ok.
type HelloOkPolicy struct {
	MaxPayload       int `json:"maxPayload"`
	MaxBufferedBytes int `json:"maxBufferedBytes"`
	// TickIntervalMs is how often the server will send tick events (ms).
	// The node watchdog disconnects if no tick arrives within 2× this value.
	TickIntervalMs int `json:"tickIntervalMs"`
}

// HelloOkServer holds server identity info in the hello-ok response.
type HelloOkServer struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	ConnID  string `json:"connId"`
}

// HelloOkFeatures lists supported methods and events in the hello-ok response.
type HelloOkFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

// NodeInvokeRequestPayload is the payload of the node.invoke.request event.
// Server sends this to the node to invoke a command.
type NodeInvokeRequestPayload struct {
	// Unique request ID for correlating the response.
	ID string `json:"id"`

	// The node ID the request is destined for.
	NodeID string `json:"nodeId"`

	// The command to invoke on the node (e.g. "system.run").
	Command string `json:"command"`

	// JSON-encoded parameters for the command (may be null).
	ParamsJSON *string `json:"paramsJSON,omitempty"`

	// Optional timeout for the command in milliseconds.
	TimeoutMs *int `json:"timeoutMs,omitempty"`

	// Optional idempotency key for deduplication.
	IdempotencyKey *string `json:"idempotencyKey,omitempty"`
}

// NodeInvokeResultParams is the parameters of the node.invoke.result request.
// The node sends this back to the server after executing a command.
type NodeInvokeResultParams struct {
	// Matches the ID from NodeInvokeRequestPayload.
	ID string `json:"id"`

	// The node ID (must match the invoking node's ID).
	NodeID string `json:"nodeId"`

	// Whether the invocation succeeded.
	Ok bool `json:"ok"`

	// Structured result payload (alternative to PayloadJSON).
	Payload any `json:"payload,omitempty"`

	// JSON-encoded result payload (preferred over Payload for large results).
	PayloadJSON *string `json:"payloadJSON,omitempty"`

	// Error info if Ok is false.
	Error *InvokeError `json:"error,omitempty"`
}

// InvokeError represents an error from a node invoke.
type InvokeError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// NodeEventParams is the parameters of the node.event request.
// Nodes send this to forward events to the server.
type NodeEventParams struct {
	Event       string  `json:"event"`
	PayloadJSON *string `json:"payloadJSON,omitempty"`
	Payload     any     `json:"payload,omitempty"`
}

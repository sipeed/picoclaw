package gateway

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openclaw/go_node/infra"
)

// InvokeRequest is the payload of node.invoke.request event
type InvokeRequest struct {
	ID         string `json:"id"`
	NodeID     string `json:"nodeId"`
	Command    string `json:"command"`
	ParamsJSON string `json:"paramsJSON,omitempty"`
	TimeoutMs  *int64 `json:"timeoutMs,omitempty"`
}

// InvokeResult is sent via node.invoke.result
type InvokeResult struct {
	OK          bool        `json:"ok"`
	Payload     interface{} `json:"payload,omitempty"`
	PayloadJSON string      `json:"payloadJSON,omitempty"`
	Error       *ErrorShape `json:"error,omitempty"`
}

// OnInvoke is called when node.invoke.request is received
type OnInvoke func(req InvokeRequest) InvokeResult

// Session connects to the gateway as a node role and handles node.invoke.request
type Session struct {
	url          string
	token        string
	password     string
	opts         ConnectOptions
	identity     *infra.DeviceIdentity
	onInvoke     OnInvoke
	conn         *websocket.Conn
	writeMu      sync.Mutex
	pending      map[string]chan json.RawMessage
	pendingMu    sync.Mutex
	connectNonce string
}

// ConnectOptions for node role
type ConnectOptions struct {
	Client    ClientInfo
	Role      string // "node"
	Scopes    []string
	Caps      []string
	Commands  []string
	Locale    string
	UserAgent string
}

// NewSession creates a new node session
func NewSession(url, token, password string, opts ConnectOptions, identity *infra.DeviceIdentity, onInvoke OnInvoke) *Session {
	return &Session{
		url:      url,
		token:    token,
		password: password,
		opts:     opts,
		identity: identity,
		onInvoke: onInvoke,
		pending:  make(map[string]chan json.RawMessage),
	}
}

// Run connects to the gateway and processes messages until ctx is done
func (s *Session) Run(ctx context.Context) error {
	headers := http.Header{}
	headers.Set("Origin", s.url)
	if s.url[:5] == "wss:" {
		headers.Set("Origin", "https"+s.url[3:])
	}
	conn, _, err := websocket.DefaultDialer.Dial(s.url, headers)
	if err != nil {
		return err
	}
	defer conn.Close()
	s.conn = conn

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		typ, _ := raw["type"]
		var typStr string
		_ = json.Unmarshal(typ, &typStr)

		switch typStr {
		case FrameTypeEvent:
			s.handleEvent(data)
		case FrameTypeRes:
			s.handleResponse(data)
		}
	}
}

func (s *Session) handleEvent(data []byte) {
	var evt EventFrame
	if err := json.Unmarshal(data, &evt); err != nil {
		return
	}
	if evt.Event == EventConnectChallenge {
		var payload struct {
			Nonce string `json:"nonce"`
		}
		_ = json.Unmarshal(evt.Payload, &payload)
		s.connectNonce = payload.Nonce
		if s.connectNonce != "" {
			s.sendConnect()
		}
		return
	}
	if evt.Event == EventNodeInvokeReq && s.onInvoke != nil {
		var req struct {
			ID         string          `json:"id"`
			NodeID     string          `json:"nodeId"`
			Command    string          `json:"command"`
			ParamsJSON string          `json:"paramsJSON"`
			Params     json.RawMessage `json:"params"`
			TimeoutMs  *int64          `json:"timeoutMs"`
		}
		_ = json.Unmarshal(evt.Payload, &req)
		paramsJSON := req.ParamsJSON
		if paramsJSON == "" && len(req.Params) > 0 {
			paramsJSON = string(req.Params)
		}
		invReq := InvokeRequest{
			ID:         req.ID,
			NodeID:     req.NodeID,
			Command:    req.Command,
			ParamsJSON: paramsJSON,
			TimeoutMs:  req.TimeoutMs,
		}
		result := s.onInvoke(invReq)
		s.sendInvokeResult(invReq.ID, invReq.NodeID, result)
	}
}

func (s *Session) handleResponse(data []byte) {
	var res struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(data, &res)
	var idStr string
	if err := json.Unmarshal(res.ID, &idStr); err != nil {
		return
	}
	s.pendingMu.Lock()
	ch := s.pending[idStr]
	delete(s.pending, idStr)
	s.pendingMu.Unlock()
	if ch != nil {
		select {
		case ch <- data:
		default:
		}
	}
}

func (s *Session) sendConnect() {
	auth := &AuthParams{}
	if s.token != "" {
		auth.Token = s.token
	} else if s.password != "" {
		auth.Password = s.password
	}

	device := (*DeviceAuth)(nil)
	if s.connectNonce != "" && s.identity != nil {
		signedAtMs := time.Now().UnixMilli()
		payload := infra.BuildDeviceAuthPayload(struct {
			DeviceID   string
			ClientID   string
			ClientMode string
			Role       string
			Scopes     []string
			SignedAtMs int64
			Token      string
			Nonce      string
		}{
			s.identity.DeviceID,
			s.opts.Client.ID,
			s.opts.Client.Mode,
			s.opts.Role,
			s.opts.Scopes,
			signedAtMs,
			s.token,
			s.connectNonce,
		})
		sig := infra.SignDevicePayload(s.identity, payload)
		device = &DeviceAuth{
			ID:        s.identity.DeviceID,
			PublicKey: infra.PublicKeyBase64URL(s.identity),
			Signature: sig,
			SignedAt:  signedAtMs,
			Nonce:     s.connectNonce,
		}
	}

	params := ConnectParams{
		MinProtocol: ProtocolVersion,
		MaxProtocol: ProtocolVersion,
		Client:      s.opts.Client,
		Role:        s.opts.Role,
		Scopes:      s.opts.Scopes,
		Caps:        s.opts.Caps,
		Commands:    s.opts.Commands,
		Auth:        auth,
		Device:      device,
		Locale:      s.opts.Locale,
		UserAgent:   s.opts.UserAgent,
	}
	paramsB, _ := json.Marshal(params)
	req := map[string]interface{}{
		"type":   FrameTypeReq,
		"id":     genID(),
		"method": MethodConnect,
		"params": json.RawMessage(paramsB),
	}
	_ = s.request(req, 15*time.Second)
	log.Printf("go_node: connect sent")
}

func (s *Session) sendInvokeResult(id, nodeID string, result InvokeResult) {
	params := map[string]interface{}{
		"id":     id,
		"nodeId": nodeID,
		"ok":     result.OK,
	}
	if result.Payload != nil {
		params["payload"] = result.Payload
	}
	if result.PayloadJSON != "" {
		params["payloadJSON"] = result.PayloadJSON
	}
	if result.Error != nil {
		params["error"] = result.Error
	}
	req := map[string]interface{}{
		"type":   FrameTypeReq,
		"id":     genID(),
		"method": MethodNodeInvokeRes,
		"params": params,
	}
	if err := s.request(req, 15*time.Second); err != nil {
		log.Printf("go_node: node.invoke.result failed: %v", err)
	}
}

func (s *Session) request(req map[string]interface{}, timeout time.Duration) error {
	id, _ := req["id"].(string)
	ch := make(chan json.RawMessage, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()

	s.writeMu.Lock()
	err := s.conn.WriteJSON(req)
	s.writeMu.Unlock()
	if err != nil {
		return err
	}
	select {
	case <-time.After(timeout):
		return nil
	case <-ch:
		return nil
	}
}

func genID() string {
	b := make([]byte, 8)
	crand.Read(b)
	return "go-node-" + hex.EncodeToString(b)
}

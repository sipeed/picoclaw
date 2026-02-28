package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ServerConfig configures the nodes WebSocket server.
type ServerConfig struct {
	// Enabled controls whether the node server starts at all.
	Enabled bool `json:"enabled"`

	// Host is the bind address (e.g. "0.0.0.0" or "127.0.0.1").
	Host string `json:"host"`

	// Port is the TCP port to listen on (default: 18789).
	Port int `json:"port"`

	// Token is an optional shared secret. If non-empty, connecting nodes must
	// provide it as auth.token. Leave empty to allow all connections.
	Token string `json:"token"`
}

// Server is a WebSocket server that accepts openclaw node connections.
// It speaks the openclaw gateway protocol (version 3), allowing headless
// Linux nodes, iOS, Android, and macOS apps to connect as "node" role clients.
type Server struct {
	cfg      ServerConfig
	registry *Registry
	upgrader websocket.Upgrader
	httpSrv  *http.Server
}

// NewServer creates a new Server with the given configuration.
// The returned server shares the provided Registry with other components.
func NewServer(cfg ServerConfig, registry *Registry) *Server {
	return &Server{
		cfg:      cfg,
		registry: registry,
		upgrader: websocket.Upgrader{
			// Allow all origins since nodes may connect from any host.
			CheckOrigin: func(r *http.Request) bool { return true },
			// Match the openclaw gateway max payload (10 MB).
			ReadBufferSize:  1024 * 1024,
			WriteBufferSize: 1024 * 1024,
		},
	}
}

// Start begins listening for WebSocket connections.
// It returns after the listener is bound, running the accept loop in background.
// Cancel ctx to gracefully shut down the server.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTP)

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("nodes server listen %s: %w", addr, err)
	}

	logger.InfoCF("nodes", "Node server started",
		map[string]any{"addr": addr, "token_required": s.cfg.Token != ""})

	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("nodes", "Node server error", map[string]any{"error": err.Error()})
		}
	}()

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutCtx); err != nil {
			logger.ErrorCF("nodes", "Node server shutdown error", map[string]any{"error": err.Error()})
		}
	}()

	return nil
}

// Handler returns the http.HandlerFunc that handles WebSocket node connections.
// Use this to mount the nodes endpoint onto an external HTTP server (e.g. the
// health/API server) instead of starting a dedicated listener with Start().
// The caller is responsible for server lifecycle; this method is safe to call
// before or after Start().
func (s *Server) Handler() http.HandlerFunc {
	return s.handleHTTP
}

// handleHTTP upgrades HTTP connections to WebSocket for node connections.
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade errors are already written as HTTP responses.
		return
	}

	remoteIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		remoteIP = strings.SplitN(forwarded, ",", 2)[0]
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteIP = realIP
	}
	// Strip port from remote addr.
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}

	go s.handleConnection(conn, remoteIP)
}

// handleConnection runs the full lifecycle of a single node WebSocket connection.
func (s *Server) handleConnection(conn *websocket.Conn, remoteIP string) {
	connID := uuid.New().String()
	defer conn.Close()

	log := func(level, msg string, fields map[string]any) {
		if fields == nil {
			fields = map[string]any{}
		}
		fields["conn"] = connID
		fields["remote"] = remoteIP
		switch level {
		case "info":
			logger.InfoCF("nodes", msg, fields)
		case "warn":
			logger.WarnCF("nodes", msg, fields)
		case "error":
			logger.ErrorCF("nodes", msg, fields)
		case "debug":
			logger.DebugCF("nodes", msg, fields)
		}
	}

	send := func(v any) {
		data, err := json.Marshal(v)
		if err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log("warn", "send failed", map[string]any{"error": err.Error()})
		}
	}

	sendRes := func(id string, ok bool, payload any, errBody *ErrBody) {
		frame := ResFrame{
			Type:    FrameTypeRes,
			ID:      id,
			Ok:      ok,
			Payload: payload,
			Error:   errBody,
		}
		send(frame)
	}

	// Step 1: Send connect.challenge immediately upon connection.
	nonce := uuid.New().String()
	send(EventFrame{
		Type:  FrameTypeEvent,
		Event: EventConnectChallenge,
		Payload: ConnectChallengePayload{
			Nonce: nonce,
			Ts:    time.Now().UnixMilli(),
		},
	})

	log("debug", "sent connect.challenge", nil)

	// Step 2: Wait for the connect handshake request with a timeout.
	handshakeDeadline := time.Now().Add(30 * time.Second)
	if err := conn.SetReadDeadline(handshakeDeadline); err != nil {
		log("warn", "set read deadline failed", map[string]any{"error": err.Error()})
		return
	}

	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		log("debug", "read connect failed", map[string]any{"error": err.Error()})
		return
	}

	// Remove deadline for subsequent messages.
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		log("warn", "clear read deadline failed", map[string]any{"error": err.Error()})
	}

	// Parse the frame.
	var frame map[string]any
	if err := json.Unmarshal(rawMsg, &frame); err != nil {
		log("warn", "invalid JSON in connect frame", nil)
		sendRes("", false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "invalid JSON"})
		return
	}

	frameType, _ := frame["type"].(string)
	frameMethod, _ := frame["method"].(string)
	frameID, _ := frame["id"].(string)

	if frameType != FrameTypeReq || frameMethod != MethodConnect {
		msg := fmt.Sprintf("expected connect request, got type=%s method=%s", frameType, frameMethod)
		log("warn", "invalid handshake", map[string]any{"msg": msg})
		sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: msg})
		return
	}

	// Parse ConnectParams from the "params" field.
	paramsRaw, _ := frame["params"].(map[string]any)
	paramsBytes, _ := json.Marshal(paramsRaw)
	var params ConnectParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "invalid connect params"})
		return
	}

	// Step 3: Validate protocol version.
	if params.MaxProtocol < ProtocolVersion || params.MinProtocol > ProtocolVersion {
		msg := fmt.Sprintf("protocol mismatch: client supports [%d..%d], server requires %d",
			params.MinProtocol, params.MaxProtocol, ProtocolVersion)
		log("warn", "protocol mismatch", map[string]any{
			"min": params.MinProtocol,
			"max": params.MaxProtocol,
		})
		sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: msg})
		return
	}

	// Step 4: Validate role (only "node" is accepted by this server).
	if params.Role != "node" {
		msg := fmt.Sprintf("unsupported role: %q (only 'node' is accepted)", params.Role)
		log("warn", "invalid role", map[string]any{"role": params.Role})
		sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: msg})
		return
	}

	// Step 5: Authenticate if a token is configured.
	if s.cfg.Token != "" {
		token := ""
		if params.Auth != nil {
			token = params.Auth.Token
		}
		if token != s.cfg.Token {
			log("warn", "unauthorized connection", nil)
			sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "unauthorized"})
			return
		}
	}

	// Step 6: Validate nonce to prevent replay attacks (only for remote connections).
	if params.Device != nil && params.Device.Nonce != "" {
		if params.Device.Nonce != nonce {
			log("warn", "nonce mismatch", map[string]any{
				"got":      params.Device.Nonce,
				"expected": nonce,
			})
			sendRes(frameID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "device nonce mismatch"})
			return
		}
	}

	// Step 7: Register the node.
	session := s.registry.Register(connID, conn, &params, remoteIP)

	// Build hello-ok response matching the openclaw gateway protocol.
	// policy.tickIntervalMs is critical: the node client watchdog closes the
	// connection if no tick event arrives within tickIntervalMs*2. We must
	// broadcast a tick every TickIntervalMs milliseconds to keep nodes alive.
	hostname, _ := os.Hostname()
	helloOk := HelloOkPayload{
		Type:     "hello-ok",
		Protocol: ProtocolVersion,
		Server: HelloOkServer{
			Version: "picoclaw",
			Host:    hostname,
			ConnID:  connID,
		},
		Features: HelloOkFeatures{
			Methods: []string{MethodNodeInvokeResult, MethodNodeEvent},
			Events:  []string{EventConnectChallenge, EventNodeInvokeRequest, "tick"},
		},
		Policy: HelloOkPolicy{
			MaxPayload:       MaxPayloadBytes,
			MaxBufferedBytes: MaxBufferedBytes,
			TickIntervalMs:   TickIntervalMs,
		},
	}

	sendRes(frameID, true, helloOk, nil)

	log("info", "node connected", map[string]any{
		"node_id":      session.NodeID,
		"display_name": session.DisplayName,
		"platform":     session.Platform,
		"commands":     session.Commands,
	})

	defer func() {
		nodeID := s.registry.Unregister(connID)
		if nodeID != "" {
			log("info", "node disconnected", map[string]any{"node_id": nodeID})
		}
	}()

	// Step 8a: Start tick keepalive goroutine.
	// The openclaw node client's watchdog disconnects if it doesn't receive a
	// tick event within tickIntervalMs*2. We send one every TickIntervalMs.
	tickDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(TickIntervalMs * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-tickDone:
				return
			case t := <-ticker.C:
				send(EventFrame{
					Type:    FrameTypeEvent,
					Event:   "tick",
					Payload: map[string]any{"ts": t.UnixMilli()},
				})
			}
		}
	}()
	defer close(tickDone)

	// Step 8b: Handle subsequent requests from the node.
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log("debug", "connection read error", map[string]any{"error": err.Error()})
			}
			return
		}

		var req map[string]any
		if err := json.Unmarshal(rawMsg, &req); err != nil {
			sendRes("", false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "invalid JSON"})
			continue
		}

		reqType, _ := req["type"].(string)
		reqMethod, _ := req["method"].(string)
		reqID, _ := req["id"].(string)

		if reqType != FrameTypeReq {
			// Ignore non-request frames (events, etc.).
			continue
		}

		switch reqMethod {
		case MethodNodeInvokeResult:
			s.handleNodeInvokeResult(req, reqID, session, sendRes)
		case MethodNodeEvent:
			// Node is sending an event upward (e.g., voice, sensor data).
			// For now, just acknowledge. Future: route to agent bus.
			sendRes(reqID, true, map[string]any{"ok": true}, nil)
		default:
			log("warn", "unknown method", map[string]any{"method": reqMethod})
			sendRes(reqID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest,
				Message: fmt.Sprintf("unknown method: %s", reqMethod)})
		}
	}
}

// handleNodeInvokeResult processes a node.invoke.result request from a node.
func (s *Server) handleNodeInvokeResult(
	req map[string]any,
	reqID string,
	session *NodeSession,
	sendRes func(id string, ok bool, payload any, errBody *ErrBody),
) {
	paramsRaw, _ := req["params"].(map[string]any)
	paramsBytes, _ := json.Marshal(paramsRaw)
	var p NodeInvokeResultParams
	if err := json.Unmarshal(paramsBytes, &p); err != nil {
		sendRes(reqID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "invalid params"})
		return
	}

	// Security: the nodeId in the result must match the caller's nodeId.
	if p.NodeID != session.NodeID {
		sendRes(reqID, false, nil, &ErrBody{Code: ErrCodeInvalidRequest, Message: "nodeId mismatch"})
		return
	}

	// Deliver the result to any waiting Invoke() call.
	matched := s.registry.HandleInvokeResult(&p)
	if !matched {
		// Late-arriving result (after timeout) - acknowledged but ignored.
		logger.DebugCF("nodes", "late invoke result ignored",
			map[string]any{"id": p.ID, "node_id": p.NodeID})
	}
	sendRes(reqID, true, map[string]any{"ok": true}, nil)
}

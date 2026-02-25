package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/routing"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Server is the WebSocket Gateway server (WebClaw/OpenClaw protocol).
type Server struct {
	cfg        *config.GatewayConfig
	registry   *agent.AgentRegistry
	bus        *bus.MessageBus
	server     *http.Server
	subs       map[string]map[*websocket.Conn]struct{}
	subsMu     sync.RWMutex
	connWriteMu sync.Map // *websocket.Conn -> *sync.Mutex, serializes writes per conn
	seq        int
	seqMu      sync.Mutex
}

// NewServer creates a new Gateway server. It serves /health, /ready, and / (WebSocket).
func NewServer(cfg *config.GatewayConfig, registry *agent.AgentRegistry, msgBus *bus.MessageBus) *Server {
	s := &Server{
		cfg:      cfg,
		registry: registry,
		bus:      msgBus,
		subs:     make(map[string]map[*websocket.Conn]struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.serveHealth)
	mux.HandleFunc("/ready", s.serveReady)
	mux.HandleFunc("/", s.serveWebSocket)
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) serveHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) serveReady(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("gateway", "WebSocket upgrade failed", map[string]any{"error": err.Error()})
		return
	}
	defer conn.Close()

	defer func() {
		s.subsMu.Lock()
		for _, m := range s.subs {
			delete(m, conn)
		}
		s.subsMu.Unlock()
		s.connWriteMu.Delete(conn)
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var frame GatewayFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			continue
		}
		if frame.Type != "req" {
			continue
		}
		s.handleRequest(conn, &frame)
	}
}

func (s *Server) handleRequest(conn *websocket.Conn, frame *GatewayFrame) {
	var payload interface{}
	var errMsg string
	var errCode string

	switch frame.Method {
	case "connect":
		payload, errCode, errMsg = s.handleConnect(frame.Params)
	case "sessions.list":
		payload, errCode, errMsg = s.handleSessionsList(frame.Params)
	case "sessions.patch":
		payload, errCode, errMsg = s.handleSessionsPatch(frame.Params)
	case "sessions.resolve":
		payload, errCode, errMsg = s.handleSessionsResolve(frame.Params)
	case "sessions.delete":
		payload, errCode, errMsg = s.handleSessionsDelete(frame.Params)
	case "chat.send":
		payload, errCode, errMsg = s.handleChatSend(frame.Params)
	case "chat.history":
		payload, errCode, errMsg = s.handleChatHistory(frame.Params)
	case "chat.subscribe":
		payload, errCode, errMsg = s.handleChatSubscribe(conn, frame.Params)
	default:
		errCode = "METHOD_NOT_FOUND"
		errMsg = "Method not implemented: " + frame.Method
	}

	if errCode != "" {
		s.sendError(conn, frame.ID, errCode, errMsg)
		return
	}
	s.sendResponse(conn, frame.ID, payload)
}

func (s *Server) handleConnect(params json.RawMessage) (interface{}, string, string) {
	var p ConnectParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "BAD_REQUEST", "Invalid connect params"
	}
	token := strings.TrimSpace(p.Auth.Token)
	password := strings.TrimSpace(p.Auth.Password)
	cfgToken := strings.TrimSpace(s.cfg.Token)
	cfgPassword := strings.TrimSpace(s.cfg.Password)
	if cfgToken != "" && token != cfgToken {
		return nil, "UNAUTHORIZED", "Invalid gateway token"
	}
	if cfgPassword != "" && password != cfgPassword {
		return nil, "UNAUTHORIZED", "Invalid gateway password"
	}
	if cfgToken == "" && cfgPassword == "" && (token == "" && password == "") {
		return nil, "UNAUTHORIZED", "Missing gateway auth"
	}
	return map[string]any{"protocol": 3, "server": "picoclaw"}, "", ""
}

// resolveSessionKey returns internal key and agentID (4.3.1).
func (s *Server) resolveSessionKey(key string) (internalKey string, agentID string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "agent:main:main", "main"
	}
	if parsed := routing.ParseAgentSessionKey(key); parsed != nil {
		return key, parsed.AgentID
	}
	// Longest prefix match on WebSessionAgentBindings
	bindings := make([]config.WebSessionAgentBinding, len(s.cfg.WebSessionAgentBindings))
	copy(bindings, s.cfg.WebSessionAgentBindings)
	sort.Slice(bindings, func(i, j int) bool {
		return len(bindings[i].SessionKeyPrefix) > len(bindings[j].SessionKeyPrefix)
	})
	for _, b := range bindings {
		prefix := strings.TrimSpace(b.SessionKeyPrefix)
		if prefix != "" && strings.HasPrefix(key, prefix) {
			aid := strings.TrimSpace(b.AgentID)
			if aid == "" {
				aid = "main"
			}
			return "agent:" + aid + ":" + key, routing.NormalizeAgentID(aid)
		}
	}
	return "agent:main:" + key, "main"
}

func (s *Server) getAgent(agentID string) *agent.AgentInstance {
	ag, _ := s.registry.GetAgent(agentID)
	if ag != nil {
		return ag
	}
	return s.registry.GetDefaultAgent()
}

func (s *Server) handleSessionsList(params json.RawMessage) (interface{}, string, string) {
	var p SessionsListParams
	_ = json.Unmarshal(params, &p)
	if p.Limit <= 0 {
		p.Limit = 50
	}
	// List from default agent only for now; multi-agent can merge later.
	ag := s.registry.GetDefaultAgent()
	if ag == nil {
		return map[string]any{"sessions": []any{}}, "", ""
	}
	meta := ag.Sessions.ListSessions()
	sessions := make([]map[string]any, 0, len(meta))
	for _, m := range meta {
		// Display key: strip "agent:main:" prefix for friendlyId
		displayKey := m.Key
		if strings.HasPrefix(displayKey, "agent:main:") {
			displayKey = strings.TrimPrefix(displayKey, "agent:main:")
		}
		// Internal heartbeat session is not exposed to WebClaw
		if displayKey == "heartbeat" {
			continue
		}
		ent := map[string]any{
			"key":        displayKey,
			"friendlyId": displayKey,
			"updatedAt":  m.UpdatedAt.UnixMilli(),
			"label":      m.Label,
		}
		if p.IncludeLastMessage {
			hist := ag.Sessions.GetHistory(m.Key)
			if len(hist) > 0 {
				last := hist[len(hist)-1]
				ent["lastMessage"] = messageToGateway(last, len(hist)-1, 0)
			}
		}
		if p.IncludeDerivedTitles {
			ent["derivedTitle"] = m.Label
		}
		sessions = append(sessions, ent)
	}
	return map[string]any{"sessions": sessions}, "", ""
}

func (s *Server) handleSessionsPatch(params json.RawMessage) (interface{}, string, string) {
	var p SessionsPatchParams
	if err := json.Unmarshal(params, &p); err != nil || strings.TrimSpace(p.Key) == "" {
		return nil, "BAD_REQUEST", "key required"
	}
	internalKey, agentID := s.resolveSessionKey(p.Key)
	ag := s.getAgent(agentID)
	if ag == nil {
		return nil, "INTERNAL", "no agent"
	}
	ag.Sessions.GetOrCreate(internalKey)
	if p.Label != "" {
		ag.Sessions.SetLabel(internalKey, strings.TrimSpace(p.Label))
	}
	if err := ag.Sessions.Save(internalKey); err != nil {
		return nil, "INTERNAL", err.Error()
	}
	displayKey := internalKey
	if strings.HasPrefix(internalKey, "agent:") {
		if idx := strings.Index(internalKey[6:], ":"); idx >= 0 {
			displayKey = internalKey[6+idx+1:]
		}
	}
	return map[string]any{"ok": true, "key": displayKey, "entry": map[string]any{"key": displayKey}}, "", ""
}

func (s *Server) handleSessionsResolve(params json.RawMessage) (interface{}, string, string) {
	var p SessionsResolveParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "BAD_REQUEST", "invalid params"
	}
	internalKey, _ := s.resolveSessionKey(strings.TrimSpace(p.Key))
	// Return key in display form for WebClaw (friendlyId); chat.send/history accept either and resolve again.
	displayKey := internalKey
	if strings.HasPrefix(internalKey, "agent:") {
		if idx := strings.Index(internalKey[6:], ":"); idx >= 0 {
			displayKey = internalKey[6+idx+1:]
		}
	}
	return map[string]any{"ok": true, "key": displayKey}, "", ""
}

func (s *Server) handleSessionsDelete(params json.RawMessage) (interface{}, string, string) {
	var p SessionsDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "BAD_REQUEST", "invalid params"
	}
	internalKey, agentID := s.resolveSessionKey(strings.TrimSpace(p.Key))
	mainKey := routing.BuildAgentMainSessionKey(agentID)
	if internalKey == mainKey {
		return nil, "INVALID_REQUEST", fmt.Sprintf("Cannot delete the main session (%s).", mainKey)
	}
	displayKey := internalKey
	if strings.HasPrefix(internalKey, "agent:") {
		if idx := strings.Index(internalKey[6:], ":"); idx >= 0 {
			displayKey = internalKey[6+idx+1:]
		}
	}
	if displayKey == "heartbeat" {
		return nil, "INVALID_REQUEST", "Cannot delete the heartbeat session."
	}
	ag := s.getAgent(agentID)
	if ag != nil {
		_ = ag.Sessions.Delete(internalKey)
	}
	return map[string]any{"ok": true}, "", ""
}

func (s *Server) handleChatSend(params json.RawMessage) (interface{}, string, string) {
	var p ChatSendParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "BAD_REQUEST", "invalid params"
	}
	sessionKey := strings.TrimSpace(p.SessionKey)
	if sessionKey == "" && strings.TrimSpace(p.IdempotencyKey) != "" {
		sessionKey = "main"
	}
	internalKey, agentID := s.resolveSessionKey(sessionKey)
	runID := strings.TrimSpace(p.IdempotencyKey)
	if runID == "" {
		runID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	chatID := internalKey + "|" + runID
	content := strings.TrimSpace(p.Message)
	if content == "" {
		return nil, "BAD_REQUEST", "message required"
	}
	ag := s.getAgent(agentID)
	if ag == nil {
		return nil, "INTERNAL", "no agent"
	}
	s.bus.PublishInbound(bus.InboundMessage{
		Channel:    "web",
		SenderID:   "webclaw",
		ChatID:     chatID,
		Content:    content,
		SessionKey: internalKey,
	})
	displayKey := internalKey
	if strings.HasPrefix(internalKey, "agent:") {
		if idx := strings.Index(internalKey[6:], ":"); idx >= 0 {
			displayKey = internalKey[6+idx+1:]
		}
	}
	return map[string]any{"ok": true, "runId": runID, "sessionKey": displayKey}, "", ""
}

func (s *Server) handleChatHistory(params json.RawMessage) (interface{}, string, string) {
	var p ChatHistoryParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "BAD_REQUEST", "invalid params"
	}
	internalKey, agentID := s.resolveSessionKey(strings.TrimSpace(p.SessionKey))
	if p.Limit <= 0 {
		p.Limit = 200
	}
	ag := s.getAgent(agentID)
	if ag == nil {
		return map[string]any{"sessionKey": p.SessionKey, "messages": []any{}}, "", ""
	}
	history := ag.Sessions.GetHistory(internalKey)
	if len(history) > p.Limit {
		history = history[len(history)-p.Limit:]
	}
	msgs := historyToGatewayMessages(history, 0)
	return map[string]any{"sessionKey": p.SessionKey, "messages": msgs}, "", ""
}

func (s *Server) handleChatSubscribe(conn *websocket.Conn, params json.RawMessage) (interface{}, string, string) {
	var p ChatSubscribeParams
	_ = json.Unmarshal(params, &p)
	key := strings.TrimSpace(p.SessionKey)
	if key == "" {
		key = strings.TrimSpace(p.FriendlyId)
	}
	if key == "" {
		return map[string]any{"ok": true}, "", ""
	}
	internalKey, _ := s.resolveSessionKey(key)
	s.subsMu.Lock()
	if s.subs[internalKey] == nil {
		s.subs[internalKey] = make(map[*websocket.Conn]struct{})
	}
	s.subs[internalKey][conn] = struct{}{}
	s.subsMu.Unlock()
	return map[string]any{"ok": true}, "", ""
}

// writeJSONLocked serializes writes per connection (gorilla/websocket forbids concurrent writes).
func (s *Server) writeJSONLocked(conn *websocket.Conn, v any) error {
	muI, _ := s.connWriteMu.LoadOrStore(conn, &sync.Mutex{})
	mu := muI.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteJSON(v)
}

func (s *Server) sendResponse(conn *websocket.Conn, id string, payload interface{}) {
	frame := GatewayFrame{Type: "res", ID: id, Ok: true, Payload: payload}
	_ = s.writeJSONLocked(conn, frame)
}

func (s *Server) sendError(conn *websocket.Conn, id, code, message string) {
	frame := GatewayFrame{
		Type:  "res",
		ID:    id,
		Ok:    false,
		Error: &GatewayError{Code: code, Message: message},
	}
	_ = s.writeJSONLocked(conn, frame)
}

// Start starts the HTTP server and the outbound consumer goroutine. Blocks until ctx is done or server errors.
func (s *Server) Start(ctx context.Context) error {
	go s.consumeOutbound(ctx)
	errCh := make(chan error, 1)
	go func() { errCh <- s.server.ListenAndServe() }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

// Serve runs the outbound consumer and serves HTTP on the given listener. Used by tests to bind to a random port.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	go s.consumeOutbound(ctx)
	errCh := make(chan error, 1)
	go func() { errCh <- s.server.Serve(listener) }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

// consumeOutbound reads from the bus and pushes events to subscribed WebSocket connections.
func (s *Server) consumeOutbound(ctx context.Context) {
	for {
		msg, ok := s.bus.SubscribeOutbound(ctx)
		if !ok {
			return
		}
		if msg.Channel != "web" {
			continue
		}
		// ChatID format: sessionKey|runId
		parts := strings.SplitN(msg.ChatID, "|", 2)
		sessionKey := msg.ChatID
		runID := ""
		if len(parts) == 2 {
			sessionKey = parts[0]
			runID = parts[1]
		}
		state := strings.TrimSpace(msg.State)
		if state == "" {
			state = "final"
		}
		payload := map[string]any{
			"runId":      runID,
			"sessionKey": sessionKey,
			"state":      state,
			"message": map[string]any{
				"role": "assistant",
				"content": []map[string]any{
					{"type": "text", "text": msg.Content},
				},
			},
		}
		s.seqMu.Lock()
		s.seq++
		seq := s.seq
		s.seqMu.Unlock()
		eventFrame := GatewayFrame{
			Type:    "event",
			Event:   "chat",
			Payload: payload,
			Seq:     seq,
		}
		s.subsMu.RLock()
		conns := s.subs[sessionKey]
		var snapshot []*websocket.Conn
		if len(conns) > 0 {
			snapshot = make([]*websocket.Conn, 0, len(conns))
			for c := range conns {
				snapshot = append(snapshot, c)
			}
		}
		s.subsMu.RUnlock()
		for _, c := range snapshot {
			conn := c
			go func() {
				_ = s.writeJSONLocked(conn, eventFrame)
			}()
		}
	}
}
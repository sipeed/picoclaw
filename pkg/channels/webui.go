package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type WebUIChannel struct {
	*BaseChannel
	cfg        config.GatewayConfig
	httpServer *http.Server
	mu         sync.RWMutex
	clients    map[*webUIClient]struct{}
	acceptingWS atomic.Bool
	drainExitFn func(timeout time.Duration) error
	configUpdateFn func(raw []byte) error
	configReadFn func() ([]byte, error)
}

type webUIClient struct {
	conn    *websocket.Conn
	chatID  string
	sender  string
	writeMu sync.Mutex
}

type webUIInboundMessage struct {
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Content  string `json:"content"`
}

type webUIOutboundMessage struct {
	Type    string `json:"type"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

func NewWebUIChannel(cfg config.GatewayConfig, messageBus *bus.MessageBus) (*WebUIChannel, error) {
	base := NewBaseChannel("webui", cfg, messageBus, nil)
	c := &WebUIChannel{
		BaseChannel: base,
		cfg:         cfg,
		clients:     make(map[*webUIClient]struct{}),
	}
	c.acceptingWS.Store(true)
	return c, nil
}

func (c *WebUIChannel) SetDrainExit(fn func(timeout time.Duration) error) {
	c.drainExitFn = fn
}

func (c *WebUIChannel) SetConfigUpdate(fn func(raw []byte) error) {
	c.configUpdateFn = fn
}

func (c *WebUIChannel) SetConfigRead(fn func() ([]byte, error)) {
	c.configReadFn = fn
}

func (c *WebUIChannel) Start(ctx context.Context) error {
	addr, err := c.cfg.ResolvedAddr()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", c.handleWS)
	mux.HandleFunc("/admin/config", c.handleAdminConfig)
	mux.HandleFunc("/admin/schema", c.handleAdminSchema)
	mux.HandleFunc("/admin/drain-exit", c.handleAdminDrainExit)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/", c.staticHandler())

	c.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.InfoCF("webui", "WebUI server listening", map[string]interface{}{
			"addr": addr,
			"bind": c.cfg.Bind,
		})
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("webui", "WebUI server error", map[string]interface{}{"error": err.Error()})
		}
	}()

	c.setRunning(true)
	return nil
}

func (c *WebUIChannel) Stop(ctx context.Context) error {
	c.setRunning(false)
	c.beginDrain()

	if c.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = c.httpServer.Shutdown(shutdownCtx)
	}
	return nil
}

func (c *WebUIChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("webui channel not running")
	}

	out := webUIOutboundMessage{Type: "message", ChatID: msg.ChatID, Content: msg.Content}
	payload, err := json.Marshal(out)
	if err != nil {
		return err
	}

	c.mu.RLock()
	clients := make([]*webUIClient, 0, len(c.clients))
	for cl := range c.clients {
		clients = append(clients, cl)
	}
	c.mu.RUnlock()

	var sendErr error
	for _, cl := range clients {
		if msg.ChatID != "" && cl.chatID != "" && cl.chatID != msg.ChatID {
			continue
		}
		cl.writeMu.Lock()
		err := cl.conn.WriteMessage(websocket.TextMessage, payload)
		cl.writeMu.Unlock()
		if err != nil {
			sendErr = err
			c.removeClient(cl)
		}
	}

	return sendErr
}

func (c *WebUIChannel) removeClient(cl *webUIClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.clients[cl]; ok {
		_ = cl.conn.Close()
		delete(c.clients, cl)
	}
}

func (c *WebUIChannel) isAuthorized(u *url.URL) bool {
	expected := strings.TrimSpace(c.cfg.Token)
	if expected == "" {
		return true
	}
	provided := strings.TrimSpace(u.Query().Get("token"))
	return provided != "" && provided == expected
}

func (c *WebUIChannel) handleWS(w http.ResponseWriter, r *http.Request) {
	if !c.acceptingWS.Load() {
		http.Error(w, "Gateway is draining", http.StatusServiceUnavailable)
		return
	}

	if !c.isAuthorized(r.URL) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// If token auth is enabled, only allow same-origin WS to reduce cross-site hijacking.
			// If token is not set, keep permissive behavior for local/LAN usage.
			if strings.TrimSpace(c.cfg.Token) == "" {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return strings.EqualFold(u.Host, r.Host)
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &webUIClient{
		conn:   conn,
		chatID: r.URL.Query().Get("chat_id"),
		sender: r.RemoteAddr,
	}
	if client.chatID == "" {
		client.chatID = "browser"
	}

	c.mu.Lock()
	c.clients[client] = struct{}{}
	c.mu.Unlock()

	defer c.removeClient(client)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if !c.acceptingWS.Load() {
			return
		}

		var in webUIInboundMessage
		if err := json.Unmarshal(data, &in); err != nil {
			continue
		}
		content := strings.TrimSpace(in.Content)
		if content == "" {
			continue
		}

		chatID := strings.TrimSpace(in.ChatID)
		if chatID == "" {
			chatID = client.chatID
		}
		senderID := strings.TrimSpace(in.SenderID)
		if senderID == "" {
			senderID = client.sender
		}

		client.chatID = chatID
		client.sender = senderID

		c.bus.PublishInbound(bus.InboundMessage{
			Channel:   "webui",
			ChatID:     chatID,
			SenderID:   senderID,
			Content:    content,
			SessionKey: chatID,
			Metadata:   map[string]string{"source": "webui"},
		})
		c.HandleMessage(senderID, chatID, content, nil, map[string]string{"source": "webui"})
	}
}

func (c *WebUIChannel) beginDrain() {
	c.acceptingWS.Store(false)

	c.mu.Lock()
	for cl := range c.clients {
		_ = cl.conn.Close()
		delete(c.clients, cl)
	}
	c.mu.Unlock()
}

func (c *WebUIChannel) isAdminAuthorized(r *http.Request) bool {
	expected := strings.TrimSpace(c.cfg.AdminToken)
	if expected == "" {
		return false
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	provided := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
	return provided != "" && provided == expected
}

func (c *WebUIChannel) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut}, ", "))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !c.isAdminAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		if c.configReadFn == nil {
			http.Error(w, "Config read not available", http.StatusNotImplemented)
			return
		}
		raw, err := c.configReadFn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		masked, err := maskConfigJSON(raw)
		if err != nil {
			http.Error(w, "Failed to mask config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(masked)
		return
	}

	if c.configUpdateFn == nil {
		http.Error(w, "Config update not available", http.StatusNotImplemented)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}

	mergedRaw := body
	if c.configReadFn != nil {
		oldRaw, rerr := c.configReadFn()
		if rerr == nil {
			if out, merr := mergePreserveMaskedSecrets(oldRaw, body); merr == nil {
				mergedRaw = out
			}
		}
	}

	if err := c.configUpdateFn(mergedRaw); err != nil {
		msg := err.Error()
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "config is not writable") {
			http.Error(w, msg, http.StatusConflict)
			return
		}
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

const maskedSentinel = "********"

func maskStringPrefix8(s string) string {
	if s == "" {
		return s
	}
	n := 8
	if len(s) < n {
		n = len(s)
	}
	return s[:n] + maskedSentinel
}

func isSensitiveKey(k string) bool {
	switch strings.ToLower(k) {
	case "api_key", "token", "admin_token", "app_secret", "client_secret", "channel_secret", "channel_access_token", "access_token", "bot_token", "app_token", "encrypt_key", "verification_token":
		return true
	default:
		return false
	}
}

func maskConfigJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	maskAny(v)
	return json.MarshalIndent(v, "", "  ")
}

func maskAny(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			if isSensitiveKey(k) {
				if s, ok := vv.(string); ok && s != "" {
					x[k] = maskStringPrefix8(s)
					continue
				}
			}
			maskAny(vv)
		}
	case []any:
		for i := range x {
			maskAny(x[i])
		}
	default:
		return
	}
}

func mergePreserveMaskedSecrets(oldRaw, newRaw []byte) ([]byte, error) {
	var oldV any
	var newV any
	if err := json.Unmarshal(oldRaw, &oldV); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(newRaw, &newV); err != nil {
		return nil, err
	}
	mergeAny(oldV, newV)
	return json.MarshalIndent(newV, "", "  ")
}

func mergeAny(oldV any, newV any) {
	oldMap, okOld := oldV.(map[string]any)
	newMap, okNew := newV.(map[string]any)
	if okOld && okNew {
		for k, nv := range newMap {
			ov, ok := oldMap[k]
			if !ok {
				continue
			}
			if isSensitiveKey(k) {
				if s, ok := nv.(string); ok && strings.Contains(s, maskedSentinel) {
					if os, ok := ov.(string); ok {
						newMap[k] = os
						continue
					}
				}
			}
			mergeAny(ov, nv)
		}
		return
	}

	oldArr, okOldArr := oldV.([]any)
	newArr, okNewArr := newV.([]any)
	if okOldArr && okNewArr {
		min := len(oldArr)
		if len(newArr) < min {
			min = len(newArr)
		}
		for i := 0; i < min; i++ {
			mergeAny(oldArr[i], newArr[i])
		}
	}
}

func (c *WebUIChannel) handleAdminSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !c.isAdminAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	path := c.findConfigSchemaPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Schema not available", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/schema+json")
	w.WriteHeader(http.StatusOK)
	w.Write(raw)
}

type drainExitRequest struct {
	TimeoutSeconds int `json:"timeout_seconds"`
}

func (c *WebUIChannel) handleAdminDrainExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !c.isAdminAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if c.drainExitFn == nil {
		http.Error(w, "Drain/exit not available", http.StatusNotImplemented)
		return
	}

	// Stop accepting new WS connections and close existing sessions immediately.
	c.beginDrain()

	timeout := 30 * time.Second
	if r.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
		if len(strings.TrimSpace(string(body))) > 0 {
			var req drainExitRequest
			if err := json.Unmarshal(body, &req); err == nil {
				if req.TimeoutSeconds > 0 {
					timeout = time.Duration(req.TimeoutSeconds) * time.Second
				}
			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("draining"))

	go func() {
		_ = c.drainExitFn(timeout)
	}()
}

func (c *WebUIChannel) staticHandler() http.Handler {
	root := c.findUIRoot()
	fs := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fs.ServeHTTP(w, r)
			return
		}

		reqPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		full := filepath.Join(root, reqPath)
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		index := filepath.Join(root, "index.html")
		if _, err := os.Stat(index); err == nil {
			http.ServeFile(w, r, index)
			return
		}

		http.NotFound(w, r)
	})
}

func (c *WebUIChannel) findUIRoot() string {
	candidates := []string{
		filepath.Join(string(os.PathSeparator), "usr", "local", "share", "picoclaw", "ui", "dist"),
		filepath.Join(string(os.PathSeparator), "usr", "share", "picoclaw", "ui", "dist"),
		filepath.Join("ui", "dist"),
		"ui",
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return "ui"
}

func (c *WebUIChannel) findConfigSchemaPath() string {
	candidates := []string{
		filepath.Join(string(os.PathSeparator), "usr", "local", "share", "picoclaw", "config", "config.schema.json"),
		filepath.Join(string(os.PathSeparator), "usr", "share", "picoclaw", "config", "config.schema.json"),
		filepath.Join("config", "config.schema.json"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return filepath.Join("config", "config.schema.json")
}

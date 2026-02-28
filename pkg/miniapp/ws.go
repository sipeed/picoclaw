package miniapp

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const maxWSClients = 4

const (
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 54 * time.Second // must be less than wsPongWait
)

type wsClient struct {
	conn *websocket.Conn
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser clients (e.g. curl)
		}
		// Allow same-origin requests (e.g. Tailscale direct access)
		if u, err := url.Parse(origin); err == nil && u.Host == r.Host {
			return true
		}
		// Allow Telegram WebApp origins and localhost for dev
		return strings.HasSuffix(origin, ".telegram.org") ||
			strings.HasSuffix(origin, ".t.me") ||
			strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "http://127.0.0.1")
	},
}

// NewHandler creates a new Mini App handler.

// wsLogs serves a WebSocket endpoint that streams log entries in real time.
func (h *Handler) wsLogs(w http.ResponseWriter, r *http.Request) {
	// Parse filter params
	component := r.URL.Query().Get("component")
	levelStr := r.URL.Query().Get("level")
	minLevel := logger.INFO
	if levelStr != "" {
		minLevel = logger.ParseLevel(levelStr)
	}

	// Clear HTTP server deadlines before WebSocket hijack
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &wsClient{conn: conn}

	// Enforce max WS clients: evict oldest if full
	h.wsClientsMu.Lock()
	if len(h.wsClients) >= maxWSClients {
		oldest := h.wsClients[0]
		h.wsClients = h.wsClients[1:]
		oldest.conn.Close()
	}
	h.wsClients = append(h.wsClients, client)
	h.wsClientsMu.Unlock()

	defer func() {
		h.wsClientsMu.Lock()
		for i, c := range h.wsClients {
			if c == client {
				h.wsClients = append(h.wsClients[:i], h.wsClients[i+1:]...)
				break
			}
		}
		h.wsClientsMu.Unlock()
		conn.Close()
	}()

	// Build filter function
	filter := func(e logger.LogEntry) bool {
		if lvl := logger.ParseLevel(e.Level); lvl < minLevel {
			return false
		}
		if component != "" && e.Component != component {
			return false
		}
		return true
	}

	sub := logger.Subscribe(filter)
	defer logger.Unsubscribe(sub)

	// Configure ping/pong to detect dead connections
	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	// Send initial data
	initial := logger.RecentLogs(minLevel, component, 50)
	if err := conn.WriteJSON(map[string]any{"type": "init", "entries": initial}); err != nil {
		return
	}

	// Close detection goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Stream loop with periodic pings
	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-sub.Ch:
			if !ok {
				return
			}
			entry.Caller = ""                                  // strip for security
			entry.Fields = logger.SanitizeFields(entry.Fields) // mask sensitive values
			if err := conn.WriteJSON(map[string]any{"type": "entry", "entry": entry}); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// apiLogsSnapshot creates a tar.gz snapshot of the current log buffer.

// wsOrchestration streams live orchestration events (agent spawn/state/gc and
// conductor<->agent conversations) to the canvas UI.
//
// Protocol:
//
//	{"type":"init","agents":[...orch.AgentInfo]}  -- sent once on connect
//	{"type":"event","event":{...orch.Event}}       -- pushed on each state change
func (h *Handler) wsOrchestration(w http.ResponseWriter, r *http.Request) {
	if h.orchBroadcaster == nil {
		http.Error(w, `{"error":"orchestration not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sub := h.orchBroadcaster.Subscribe()
	defer h.orchBroadcaster.Unsubscribe(sub)

	// Send current agent snapshot so the canvas can populate immediately
	snapshot := h.orchBroadcaster.Snapshot()
	if err := conn.WriteJSON(map[string]any{"type": "init", "agents": snapshot}); err != nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-sub.Ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(map[string]any{"type": "event", "event": ev}); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

package pico

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// picoSubscriber is a WebSocket or SSE client registered for outbound Pico messages.
type picoSubscriber interface {
	ID() string
	SessionID() string
	Deliver(msg PicoMessage) error
	Close()
}

// picoSSEConn streams outbound messages as Server-Sent Events.
type picoSSEConn struct {
	id        string
	sessionID string
	w         http.ResponseWriter
	rc        *http.ResponseController
	writeMu   sync.Mutex
	closed    atomic.Bool
	cancel    context.CancelFunc
}

func (s *picoSSEConn) ID() string { return s.id }

func (s *picoSSEConn) SessionID() string { return s.sessionID }

func (s *picoSSEConn) Deliver(msg PicoMessage) error {
	if s.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", b); err != nil {
		return err
	}
	return s.rc.Flush()
}

func (s *picoSSEConn) Close() {
	s.shutdown()
}

func (s *picoSSEConn) shutdown() {
	if s.closed.CompareAndSwap(false, true) {
		if s.cancel != nil {
			s.cancel()
		}
	}
}

func (c *PicoChannel) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}
	if !c.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	c.disconnectSubscribersForSession(sessionID)

	maxConns := c.config.MaxConnections
	if maxConns <= 0 {
		maxConns = 100
	}
	if int(c.connCount.Load()) >= maxConns {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)

	id := uuid.New().String()
	ctx, cancel := context.WithCancel(r.Context())
	sse := &picoSSEConn{
		id:        id,
		sessionID: sessionID,
		w:         w,
		rc:        rc,
		cancel:    cancel,
	}

	c.subscribers.Store(id, sse)
	c.connCount.Add(1)
	defer func() {
		sse.shutdown()
		if _, loaded := c.subscribers.LoadAndDelete(id); loaded {
			c.connCount.Add(-1)
		}
		logger.InfoCF("pico", "SSE client disconnected", map[string]any{
			"conn_id":    id,
			"session_id": sessionID,
		})
	}()

	logger.InfoCF("pico", "SSE client connected", map[string]any{
		"conn_id":    id,
		"session_id": sessionID,
	})

	ready, err := json.Marshal(map[string]string{"conn_id": id, "session_id": sessionID})
	if err != nil {
		return
	}
	sse.writeMu.Lock()
	_, werr := fmt.Fprintf(w, "event: ready\ndata: %s\n\n", ready)
	sse.writeMu.Unlock()
	if werr != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			sse.writeMu.Lock()
			if sse.closed.Load() {
				sse.writeMu.Unlock()
				return
			}
			_, werr := fmt.Fprintf(w, ": ping\n\n")
			sse.writeMu.Unlock()
			if werr != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

func (c *PicoChannel) handlePostSend(w http.ResponseWriter, r *http.Request) {
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}
	if !c.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var msg PicoMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		writePicoSendError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if msg.Type != TypeMessageSend {
		writePicoSendError(w, http.StatusBadRequest, "unsupported_type")
		return
	}
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		writePicoSendError(w, http.StatusBadRequest, "empty_content")
		return
	}
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		writePicoSendError(w, http.StatusBadRequest, "session_id_required")
		return
	}

	connID := "http-" + uuid.New().String()
	c.publishUserMessage(connID, sessionID, msg.ID, content)
	w.WriteHeader(http.StatusNoContent)
}

func writePicoSendError(w http.ResponseWriter, code int, errCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": errCode})
}

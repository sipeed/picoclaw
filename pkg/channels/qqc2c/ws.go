package qqc2c

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// intents we subscribe to (QQ Bot API v2):
//
//	1<<25 = GROUP_AND_C2C_EVENT  (C2C_MESSAGE_CREATE ...)
//	1<<26 = INTERACTION          (message button / quick-menu callbacks)
const defaultIntents = (1 << 25) | (1 << 26)

// wsEventFunc receives a single dispatched event (t = event name, d = raw data).
type wsEventFunc func(t string, d json.RawMessage)

// wsClient is a minimal, self-contained QQ Bot WebSocket client.
// It owns reconnect/backoff and heartbeat; event handling is delegated upstream.
type wsClient struct {
	api     *apiClient
	intents int
	onEvent wsEventFunc

	mu        sync.Mutex
	conn      *websocket.Conn
	sessionID string
	seq       int64

	// dialer is a field so tests can substitute it.
	dialer *websocket.Dialer
}

func newWSClient(api *apiClient, intents int, onEvent wsEventFunc) *wsClient {
	if intents == 0 {
		intents = defaultIntents
	}
	return &wsClient{
		api:     api,
		intents: intents,
		onEvent: onEvent,
		dialer:  websocket.DefaultDialer,
	}
}

// Run keeps the connection alive until ctx is cancelled.
func (w *wsClient) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := w.connectOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.WarnCF("qq_c2c", "websocket session ended, reconnecting", map[string]any{
			"error":   errString(err),
			"backoff": backoff.String(),
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (w *wsClient) connectOnce(ctx context.Context) error {
	u, err := w.api.Gateway(ctx)
	if err != nil {
		return fmt.Errorf("gateway lookup: %w", err)
	}
	hdr := http.Header{}
	conn, _, err := w.dialer.DialContext(ctx, u, hdr)
	if err != nil {
		return fmt.Errorf("dial gateway: %w", err)
	}
	defer conn.Close()

	w.mu.Lock()
	w.conn = conn
	sessionID, seq := w.sessionID, w.seq
	w.mu.Unlock()

	// First frame must be Hello (op 10).
	var hello wsPayload
	if err := conn.ReadJSON(&hello); err != nil {
		return fmt.Errorf("read hello: %w", err)
	}
	if hello.Op != 10 {
		return fmt.Errorf("unexpected first op=%d (want 10 hello)", hello.Op)
	}
	var hi wsHello
	_ = json.Unmarshal(hello.D, &hi)
	if hi.HeartbeatInterval <= 0 {
		hi.HeartbeatInterval = 40000
	}

	token, err := w.api.tm.Access(ctx)
	if err != nil {
		return err
	}

	// Resume when we have a previous session, else Identify.
	if sessionID != "" {
		if err := conn.WriteJSON(map[string]any{
			"op": 6,
			"d":  wsResume{Token: "QQBot " + token, SessionID: sessionID, Seq: seq},
		}); err != nil {
			return fmt.Errorf("send resume: %w", err)
		}
		logger.InfoC("qq_c2c", "websocket resuming previous session")
	} else {
		if err := conn.WriteJSON(map[string]any{
			"op": 2,
			"d":  wsIdentify{Token: "QQBot " + token, Intents: w.intents, Shard: [2]int{0, 1}},
		}); err != nil {
			return fmt.Errorf("send identify: %w", err)
		}
		logger.InfoC("qq_c2c", "websocket identifying")
	}

	// Heartbeat goroutine.
	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	go func() {
		t := time.NewTicker(time.Duration(hi.HeartbeatInterval) * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-t.C:
				w.mu.Lock()
				s := w.seq
				w.mu.Unlock()
				_ = conn.WriteJSON(map[string]any{"op": 1, "d": s})
			}
		}
	}()

	// Read loop.
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var p wsPayload
		if err := conn.ReadJSON(&p); err != nil {
			return fmt.Errorf("read: %w", err)
		}
		w.mu.Lock()
		if p.S > 0 {
			w.seq = p.S
		}
		w.mu.Unlock()
		switch p.Op {
		case 0: // dispatch
			if p.T == "READY" || p.T == "RESUMED" {
				var ready struct {
					SessionID string `json:"session_id"`
				}
				_ = json.Unmarshal(p.D, &ready)
				w.mu.Lock()
				if ready.SessionID != "" {
					w.sessionID = ready.SessionID
				}
				w.mu.Unlock()
				logger.InfoCF("qq_c2c", "websocket ready", map[string]any{"event": p.T})
			}
			if w.onEvent != nil && p.T != "" {
				w.onEvent(p.T, p.D)
			}
		case 7: // reconnect requested by platform
			return fmt.Errorf("platform requested reconnect")
		case 9: // invalid session
			w.mu.Lock()
			w.sessionID = ""
			w.seq = 0
			w.mu.Unlock()
			return fmt.Errorf("invalid session, will re-identify")
		case 11: // heartbeat ack
		default:
			// op 1/2/6/10 echoes: ignore
		}
	}
}

// SendClose closes the current connection (used by Stop).
func (w *wsClient) SendClose() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		_ = w.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second))
		_ = w.conn.Close()
	}
}

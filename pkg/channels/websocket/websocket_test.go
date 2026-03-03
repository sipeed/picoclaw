package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// newTestChannel creates a WebSocketChannel with the given config and a fresh bus.
func newTestChannel(t *testing.T, cfg config.WebSocketConfig) (*WebSocketChannel, *bus.MessageBus) {
	t.Helper()
	mb := bus.NewMessageBus()
	t.Cleanup(func() { mb.Close() })
	ch, err := NewWebSocketChannel(cfg, mb)
	if err != nil {
		t.Fatalf("NewWebSocketChannel: %v", err)
	}
	return ch, mb
}

// mockServer starts an httptest server that upgrades to WebSocket and calls handler
// with each connection. Returns the server and its ws:// URL.
func mockServer(t *testing.T, handler func(*websocket.Conn)) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

// readEnvelope reads one WSEnvelope from the connection with a timeout.
func readEnvelope(t *testing.T, conn *websocket.Conn, timeout time.Duration) WSEnvelope {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("readEnvelope: %v", err)
	}
	var env WSEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("readEnvelope unmarshal: %v", err)
	}
	return env
}

// writeEnvelope writes a WSEnvelope to the connection.
func writeEnvelope(t *testing.T, conn *websocket.Conn, env WSEnvelope) {
	t.Helper()
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("writeEnvelope marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("writeEnvelope write: %v", err)
	}
}

// makeMessageEnvelope builds a WSEnvelope with type "message" from the given inbound fields.
func makeMessageEnvelope(t *testing.T, msg WSInboundMessage) WSEnvelope {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("makeMessageEnvelope: %v", err)
	}
	return WSEnvelope{Type: "message", Data: json.RawMessage(data)}
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

func TestNewWebSocketChannel(t *testing.T) {
	cfg := config.WebSocketConfig{
		Enabled:            true,
		WSUrl:              "ws://localhost:9999",
		AccessToken:        "tok",
		AgentID:            "researcher",
		ReconnectInterval:  5,
		AllowFrom:          config.FlexibleStringSlice{"orchestrator"},
		ReasoningChannelID: "reason-chan",
	}
	ch, _ := newTestChannel(t, cfg)

	if ch.Name() != "websocket" {
		t.Errorf("Name() = %q, want %q", ch.Name(), "websocket")
	}
	if ch.ReasoningChannelID() != "reason-chan" {
		t.Errorf("ReasoningChannelID() = %q, want %q", ch.ReasoningChannelID(), "reason-chan")
	}
	if ch.IsRunning() {
		t.Error("expected IsRunning() == false before Start")
	}
	if ch.config.AgentID != "researcher" {
		t.Errorf("config.AgentID = %q, want %q", ch.config.AgentID, "researcher")
	}
}

func TestSend_NotRunning(t *testing.T) {
	cfg := config.WebSocketConfig{Enabled: true}
	ch, _ := newTestChannel(t, cfg)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "websocket",
		ChatID:  "chat-1",
		Content: "hello",
	})
	if err != channels.ErrNotRunning {
		t.Errorf("Send() error = %v, want %v", err, channels.ErrNotRunning)
	}
}

func TestSend_WritesToConn(t *testing.T) {
	received := make(chan WSEnvelope, 1)
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		// skip auth envelope
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage()
		// read the actual message
		env := readEnvelope(t, conn, 2*time.Second)
		received <- env
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "a1",
	}
	ch, _ := newTestChannel(t, cfg)
	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	err := ch.Send(ctx, bus.OutboundMessage{
		Channel: "websocket",
		ChatID:  "task-1",
		Content: `{"summary": "done"}`,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case env := <-received:
		if env.Type != "message" {
			t.Errorf("envelope type = %q, want %q", env.Type, "message")
		}
		var out WSOutboundMessage
		if err := json.Unmarshal(env.Data, &out); err != nil {
			t.Fatalf("unmarshal outbound: %v", err)
		}
		if out.ContentType != "json" {
			t.Errorf("content_type = %q, want %q", out.ContentType, "json")
		}
		if out.ChatID != "task-1" {
			t.Errorf("chat_id = %q, want %q", out.ChatID, "task-1")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message on mock server")
	}
}

func TestContentType_AutoDetect(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"plain text", "hello world", "text"},
		{"valid JSON object", `{"key": "value"}`, "json"},
		{"valid JSON array", `[1, 2, 3]`, "json"},
		{"empty string", "", "text"},
		{"invalid JSON", `{"broken":`, "text"},
		{"JSON string literal", `"just a string"`, "json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.content)
			if got != tt.want {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestSessionKey_Passthrough(t *testing.T) {
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		// skip auth
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage()

		msg := WSInboundMessage{
			SenderID:   "orchestrator",
			ChatID:     "task-1",
			Content:    "analyze this",
			SessionKey: "task-042",
			MessageID:  "msg-1",
		}
		data, _ := json.Marshal(msg)
		env := WSEnvelope{Type: "message", Data: json.RawMessage(data)}
		envData, _ := json.Marshal(env)
		conn.WriteMessage(websocket.TextMessage, envData)
		// keep connection open briefly
		time.Sleep(500 * time.Millisecond)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "a1",
	}
	ch, mb := newTestChannel(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	msg, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("ConsumeInbound returned false")
	}
	if msg.SessionKey != "task-042" {
		t.Errorf("SessionKey = %q, want %q", msg.SessionKey, "task-042")
	}
}

func TestSessionKey_Fallback(t *testing.T) {
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage() // skip auth

		msg := WSInboundMessage{
			SenderID: "orchestrator",
			ChatID:   "task-2",
			Content:  "do something",
			// no session_key
		}
		data, _ := json.Marshal(msg)
		env := WSEnvelope{Type: "message", Data: json.RawMessage(data)}
		envData, _ := json.Marshal(env)
		conn.WriteMessage(websocket.TextMessage, envData)
		time.Sleep(500 * time.Millisecond)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "a1",
	}
	ch, mb := newTestChannel(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	msg, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("ConsumeInbound returned false")
	}
	if msg.SessionKey != "" {
		t.Errorf("SessionKey = %q, want empty string", msg.SessionKey)
	}
}

func TestParseEnvelope_InvalidJSON(t *testing.T) {
	_, err := parseEnvelope([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseEnvelope_UnknownType(t *testing.T) {
	env, err := parseEnvelope([]byte(`{"type":"unknown","data":{}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Type != "unknown" {
		t.Errorf("Type = %q, want %q", env.Type, "unknown")
	}
	// Unknown types should be parseable — the channel is responsible for ignoring them.
}

// ---------------------------------------------------------------------------
// Integration Tests (mock WebSocket server via httptest)
// ---------------------------------------------------------------------------

func TestConnectAndReceive(t *testing.T) {
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage() // skip auth

		msg := makeMessageEnvelope(t, WSInboundMessage{
			SenderID:  "orchestrator",
			ChatID:    "task-1",
			Content:   "Analyze this document",
			MessageID: "msg-100",
		})
		writeEnvelope(t, conn, msg)
		time.Sleep(500 * time.Millisecond)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "researcher",
	}
	ch, mb := newTestChannel(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	inbound, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("ConsumeInbound returned false")
	}
	if inbound.Channel != "websocket" {
		t.Errorf("Channel = %q, want %q", inbound.Channel, "websocket")
	}
	if inbound.SenderID != "orchestrator" {
		t.Errorf("SenderID = %q, want %q", inbound.SenderID, "orchestrator")
	}
	if inbound.ChatID != "task-1" {
		t.Errorf("ChatID = %q, want %q", inbound.ChatID, "task-1")
	}
	if inbound.Content != "Analyze this document" {
		t.Errorf("Content = %q, want %q", inbound.Content, "Analyze this document")
	}
	if inbound.MessageID != "msg-100" {
		t.Errorf("MessageID = %q, want %q", inbound.MessageID, "msg-100")
	}
}

func TestConnectAndSend(t *testing.T) {
	received := make(chan WSEnvelope, 1)
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage() // skip auth
		env := readEnvelope(t, conn, 2*time.Second)
		received <- env
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "writer",
	}
	ch, _ := newTestChannel(t, cfg)
	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	err := ch.Send(ctx, bus.OutboundMessage{
		Channel: "websocket",
		ChatID:  "task-1",
		Content: "plain text response",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case env := <-received:
		if env.Type != "message" {
			t.Errorf("type = %q, want %q", env.Type, "message")
		}
		var out WSOutboundMessage
		if err := json.Unmarshal(env.Data, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.ContentType != "text" {
			t.Errorf("content_type = %q, want %q", out.ContentType, "text")
		}
		if out.Content != "plain text response" {
			t.Errorf("content = %q, want %q", out.Content, "plain text response")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message on mock server")
	}
}

func TestAuth_Handshake(t *testing.T) {
	authReceived := make(chan WSAuthData, 1)
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		env := readEnvelope(t, conn, 2*time.Second)
		if env.Type != "auth" {
			t.Errorf("first envelope type = %q, want %q", env.Type, "auth")
		}
		var auth WSAuthData
		if err := json.Unmarshal(env.Data, &auth); err != nil {
			t.Fatalf("unmarshal auth: %v", err)
		}
		authReceived <- auth
		time.Sleep(500 * time.Millisecond)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "secret-token",
		AgentID:     "researcher",
	}
	ch, _ := newTestChannel(t, cfg)
	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	select {
	case auth := <-authReceived:
		if auth.AgentID != "researcher" {
			t.Errorf("agent_id = %q, want %q", auth.AgentID, "researcher")
		}
		if auth.Token != "secret-token" {
			t.Errorf("token = %q, want %q", auth.Token, "secret-token")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for auth handshake")
	}
}

func TestAuth_Rejected(t *testing.T) {
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		// Read auth and immediately close with an error
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage()
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "bad token"))
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:           true,
		WSUrl:             wsURL,
		AccessToken:       "wrong-token",
		AgentID:           "a1",
		ReconnectInterval: 0, // no reconnect
	}
	ch, _ := newTestChannel(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start should either return an error or handle gracefully
	// (the channel should not panic or hang)
	_ = ch.Start(ctx)
	defer ch.Stop(ctx)

	// Give it a moment to process the close
	time.Sleep(200 * time.Millisecond)
	// Channel should not be running after auth rejection (or should have handled it)
}

func TestReconnect(t *testing.T) {
	connCount := make(chan int, 5)
	count := 0
	var countMu sync.Mutex

	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		countMu.Lock()
		count++
		current := count
		countMu.Unlock()
		connCount <- current

		// Read auth
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadMessage()

		if current == 1 {
			// First connection: close immediately to trigger reconnect
			conn.Close()
			return
		}
		// Second connection: stay open a bit
		time.Sleep(2 * time.Second)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:           true,
		WSUrl:             wsURL,
		AccessToken:       "tok",
		AgentID:           "a1",
		ReconnectInterval: 1, // 1 second
	}
	ch, _ := newTestChannel(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	// Wait for at least 2 connections (initial + reconnect)
	seen := 0
	for seen < 2 {
		select {
		case <-connCount:
			seen++
		case <-ctx.Done():
			t.Fatalf("timed out waiting for reconnect, saw %d connections", seen)
		}
	}
}

func TestStop_CleanShutdown(t *testing.T) {
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		conn.ReadMessage() // auth
		// Hold connection open
		time.Sleep(5 * time.Second)
		conn.Close()
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "a1",
	}
	ch, _ := newTestChannel(t, cfg)
	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Give it a moment to connect
	time.Sleep(200 * time.Millisecond)

	if err := ch.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if ch.IsRunning() {
		t.Error("expected IsRunning() == false after Stop")
	}
}

// ---------------------------------------------------------------------------
// Concurrency Tests
// ---------------------------------------------------------------------------

func TestConcurrentSends(t *testing.T) {
	received := make(chan struct{}, 100)
	_, wsURL := mockServer(t, func(conn *websocket.Conn) {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		conn.ReadMessage() // auth

		// Read all messages until error
		for {
			conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			received <- struct{}{}
		}
	})

	cfg := config.WebSocketConfig{
		Enabled:     true,
		WSUrl:       wsURL,
		AccessToken: "tok",
		AgentID:     "a1",
	}
	ch, _ := newTestChannel(t, cfg)
	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	// Give it a moment to connect
	time.Sleep(200 * time.Millisecond)

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	errs := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			err := ch.Send(ctx, bus.OutboundMessage{
				Channel: "websocket",
				ChatID:  "task-1",
				Content: "concurrent message",
			})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Send error: %v", err)
	}

	// Verify server received messages (allow some time for delivery)
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	count := 0
	for count < numGoroutines {
		select {
		case <-received:
			count++
		case <-timer.C:
			t.Errorf("received %d/%d messages", count, numGoroutines)
			return
		}
	}
}

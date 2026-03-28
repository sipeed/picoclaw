package pico

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

const testToken = "test-pico-token"

func newTestPicoChannel(t *testing.T) *PicoChannel {
	t.Helper()

	cfg := config.PicoConfig{}
	cfg.SetToken("test-token")
	ch, err := NewPicoChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewPicoChannel: %v", err)
	}

	ch.ctx = context.Background()
	return ch
}

func TestCreateAndAddConnection_RespectsMaxConnectionsConcurrently(t *testing.T) {
	ch := newTestPicoChannel(t)

	const (
		maxConns   = 5
		goroutines = 64
		sessionID  = "session-a"
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errCount := 0

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			pc, err := ch.createAndAddConnection(nil, sessionID, maxConns)
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successCount++
				if pc == nil {
					t.Errorf("pc is nil on success")
				}
				return
			}
			if !errors.Is(err, channels.ErrTemporary) {
				t.Errorf("unexpected error: %v", err)
				return
			}
			errCount++
		}()
	}
	wg.Wait()

	if successCount > maxConns {
		t.Fatalf("successCount=%d > maxConns=%d", successCount, maxConns)
	}
	if successCount+errCount != goroutines {
		t.Fatalf("success=%d err=%d total=%d want=%d", successCount, errCount, successCount+errCount, goroutines)
	}
	if got := ch.currentConnCount(); got != maxConns {
		t.Fatalf("currentConnCount=%d want=%d", got, maxConns)
	}
}

func TestRemoveConnection_CleansBothIndexes(t *testing.T) {
	ch := newTestPicoChannel(t)

	pc, err := ch.createAndAddConnection(nil, "session-cleanup", 10)
	if err != nil {
		t.Fatalf("createAndAddConnection: %v", err)
	}

	removed := ch.removeConnection(pc.id)
	if removed == nil {
		t.Fatal("removeConnection returned nil")
	}

	ch.connsMu.RLock()
	defer ch.connsMu.RUnlock()

	if _, ok := ch.connections[pc.id]; ok {
		t.Fatalf("connID %s still exists in connections", pc.id)
	}
	if _, ok := ch.sessionConnections[pc.sessionID]; ok {
		t.Fatalf("session %s still exists in sessionConnections", pc.sessionID)
	}
	if got := len(ch.connections); got != 0 {
		t.Fatalf("len(connections)=%d want=0", got)
	}
}

func TestBroadcastToSession_TargetsOnlyRequestedSession(t *testing.T) {
	ch := newTestPicoChannel(t)

	target := &picoConn{id: "target", sessionID: "s-target"}
	target.closed.Store(true)
	ch.addConnForTest(target)

	other := &picoConn{id: "other", sessionID: "s-other"}
	ch.addConnForTest(other)

	err := ch.broadcastToSession("pico:s-target", newMessage(TypeMessageCreate, map[string]any{"content": "hello"}))
	if err == nil {
		t.Fatal("expected send failure due to closed target connection")
	}
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("expected ErrSendFailed, got %v", err)
	}
}

func (c *PicoChannel) addConnForTest(pc *picoConn) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	if c.connections == nil {
		c.connections = make(map[string]*picoConn)
	}
	if c.sessionConnections == nil {
		c.sessionConnections = make(map[string]map[string]*picoConn)
	}
	if _, exists := c.connections[pc.id]; exists {
		panic(fmt.Sprintf("duplicate conn id in test: %s", pc.id))
	}
	c.connections[pc.id] = pc
	bySession, ok := c.sessionConnections[pc.sessionID]
	if !ok {
		bySession = make(map[string]*picoConn)
		c.sessionConnections[pc.sessionID] = bySession
	}
	bySession[pc.id] = pc
}

// --- HTTP message endpoint helpers ---

// newTestChannelWithConfig creates a PicoChannel and returns it alongside the bus.
// The channel is NOT started — call Start to mark it running.
func newTestChannelWithConfig(t *testing.T, token string, opts ...func(*config.PicoConfig)) (*PicoChannel, *bus.MessageBus) {
	t.Helper()
	b := bus.NewMessageBus()
	cfg := config.PicoConfig{Enabled: true}
	cfg.SetToken(token)
	for _, opt := range opts {
		opt(&cfg)
	}
	ch, err := NewPicoChannel(cfg, b)
	if err != nil {
		t.Fatalf("NewPicoChannel: %v", err)
	}
	return ch, b
}

// newTestHTTPChannel creates a PicoChannel with default config for HTTP tests.
func newTestHTTPChannel(t *testing.T) *PicoChannel {
	t.Helper()
	ch, _ := newTestChannelWithConfig(t, testToken)
	return ch
}

// startTestChannel creates, starts, and registers cleanup for a PicoChannel.
func startTestChannel(t *testing.T) *PicoChannel {
	t.Helper()
	ch := newTestHTTPChannel(t)
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { ch.Stop(context.Background()) })
	return ch
}

// startTestChannelWithBus creates and starts a PicoChannel, returning the bus
// for message verification.
func startTestChannelWithBus(t *testing.T) (*PicoChannel, *bus.MessageBus) {
	t.Helper()
	ch, b := newTestChannelWithConfig(t, testToken)
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { ch.Stop(context.Background()) })
	return ch, b
}

func authRequest(method, url, body, token string) *http.Request {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// consumeInbound reads one message from the bus with a short timeout.
func consumeInbound(t *testing.T, b *bus.MessageBus) bus.InboundMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case msg := <-b.InboundChan():
		return msg
	case <-ctx.Done():
		t.Fatal("expected inbound message on bus, got none")
		return bus.InboundMessage{}
	}
}

// --- HTTP message rejection tests ---

func TestHTTPMessage_MethodNotAllowed(t *testing.T) {
	ch := startTestChannel(t)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		req := authRequest(method, "/pico/message", `{"content":"hello"}`, testToken)
		ch.handleHTTPMessage(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected %d, got %d", method, http.StatusMethodNotAllowed, rec.Code)
		}
	}
}

func TestHTTPMessage_NotRunning(t *testing.T) {
	ch := newTestHTTPChannel(t) // not started

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":"hello"}`, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestHTTPMessage_MissingAuth(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pico/message", strings.NewReader(`{"content":"hello"}`))
	// No Authorization header.
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHTTPMessage_InvalidAuth(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":"hello"}`, "wrong-token")
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHTTPMessage_OversizedBody(t *testing.T) {
	ch := startTestChannel(t)

	oversized := bytes.Repeat([]byte("A"), maxHTTPBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/pico/message", bytes.NewReader(oversized))
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()

	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestHTTPMessage_AcceptsMaxBodySize(t *testing.T) {
	ch := startTestChannel(t)

	// Exactly at the limit — should not trigger 413.
	// Body is not valid JSON, so expect 400 (not 413).
	body := bytes.Repeat([]byte("A"), maxHTTPBodySize)
	req := httptest.NewRequest(http.MethodPost, "/pico/message", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()

	ch.handleHTTPMessage(rec, req)

	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Error("body at exactly max size should not be rejected as too large")
	}
}

func TestHTTPMessage_MalformedJSON(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", "not json at all", testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHTTPMessage_EmptyContent(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":""}`, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHTTPMessage_WhitespaceContent(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":"   "}`, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// --- HTTP message acceptance tests ---

func TestHTTPMessage_Accepted(t *testing.T) {
	ch := startTestChannel(t)

	body := `{"content":"hello","session_id":"test-session"}`
	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", body, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	var resp httpMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if resp.Status != "accepted" {
		t.Errorf("expected status=accepted, got %s", resp.Status)
	}
	if resp.SessionID != "test-session" {
		t.Errorf("expected session_id=test-session, got %s", resp.SessionID)
	}
}

func TestHTTPMessage_GeneratesSessionID(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":"hello"}`, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	var resp httpMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SessionID == "" {
		t.Error("expected non-empty generated session_id")
	}
}

// --- HTTP message bus semantics tests ---

func TestHTTPMessage_PublishesBusMessage(t *testing.T) {
	ch, b := startTestChannelWithBus(t)

	body := `{"content":"hello world","session_id":"sess-42"}`
	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", body, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	msg := consumeInbound(t, b)

	if msg.Channel != "pico" {
		t.Errorf("Channel: expected pico, got %s", msg.Channel)
	}
	if msg.Content != "hello world" {
		t.Errorf("Content: expected hello world, got %s", msg.Content)
	}
	if msg.ChatID != "pico:sess-42" {
		t.Errorf("ChatID: expected pico:sess-42, got %s", msg.ChatID)
	}
	if msg.Peer.Kind != "direct" {
		t.Errorf("Peer.Kind: expected direct, got %s", msg.Peer.Kind)
	}
	if msg.Peer.ID != "pico:sess-42" {
		t.Errorf("Peer.ID: expected pico:sess-42, got %s", msg.Peer.ID)
	}
	if msg.MessageID == "" {
		t.Error("MessageID: expected non-empty generated ID")
	}
	if msg.Metadata["platform"] != "pico" {
		t.Errorf("Metadata[platform]: expected pico, got %s", msg.Metadata["platform"])
	}
	if msg.Metadata["session_id"] != "sess-42" {
		t.Errorf("Metadata[session_id]: expected sess-42, got %s", msg.Metadata["session_id"])
	}
	if msg.Metadata["transport"] != "http" {
		t.Errorf("Metadata[transport]: expected http, got %s", msg.Metadata["transport"])
	}
	// SenderID is resolved to canonical form by BaseChannel.HandleMessage.
	if msg.SenderID != "pico:pico-user" {
		t.Errorf("SenderID: expected pico:pico-user, got %s", msg.SenderID)
	}
	if msg.Sender.Platform != "pico" {
		t.Errorf("Sender.Platform: expected pico, got %s", msg.Sender.Platform)
	}
}

func TestHTTPMessage_PreservesContentWhitespace(t *testing.T) {
	ch, b := startTestChannelWithBus(t)

	// Content has leading/trailing whitespace — should be preserved on the bus
	// (matching WebSocket message.send semantics).
	body := `{"content":"  hello world  ","session_id":"ws-compat"}`
	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", body, testToken)
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	msg := consumeInbound(t, b)
	if msg.Content != "  hello world  " {
		t.Errorf("Content: expected '  hello world  ', got %q", msg.Content)
	}
}

// --- HTTP message auth variant tests ---

func TestHTTPMessage_QueryTokenAccepted(t *testing.T) {
	ch, _ := newTestChannelWithConfig(t, testToken, func(cfg *config.PicoConfig) {
		cfg.AllowTokenQuery = true
	})
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { ch.Stop(context.Background()) })

	req := httptest.NewRequest(
		http.MethodPost,
		"/pico/message?token="+testToken,
		strings.NewReader(`{"content":"hello"}`),
	)
	rec := httptest.NewRecorder()
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestHTTPMessage_QueryTokenRejectedWhenDisabled(t *testing.T) {
	ch := startTestChannel(t) // AllowTokenQuery defaults to false

	req := httptest.NewRequest(
		http.MethodPost,
		"/pico/message?token="+testToken,
		strings.NewReader(`{"content":"hello"}`),
	)
	rec := httptest.NewRecorder()
	ch.handleHTTPMessage(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

// --- HTTP message routing tests ---

func TestServeHTTP_RoutesMessage(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := authRequest(http.MethodPost, "/pico/message", `{"content":"routed"}`, testToken)
	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestServeHTTP_UnknownPathReturns404(t *testing.T) {
	ch := startTestChannel(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pico/unknown", nil)
	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

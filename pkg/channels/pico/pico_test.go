package pico

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

const testToken = "test-pico-token"

// newTestChannelWithConfig creates a PicoChannel and returns it alongside the bus.
// The channel is NOT started — call Start to mark it running.
func newTestChannelWithConfig(t *testing.T, cfg config.PicoConfig) (*PicoChannel, *bus.MessageBus) {
	t.Helper()
	b := bus.NewMessageBus()
	ch, err := NewPicoChannel(cfg, b)
	if err != nil {
		t.Fatalf("NewPicoChannel: %v", err)
	}
	return ch, b
}

// newTestChannel creates a PicoChannel with default config.
func newTestChannel(t *testing.T) *PicoChannel {
	t.Helper()
	ch, _ := newTestChannelWithConfig(t, config.PicoConfig{
		Enabled: true,
		Token:   testToken,
	})
	return ch
}

// startTestChannel creates, starts, and registers cleanup for a PicoChannel.
func startTestChannel(t *testing.T) *PicoChannel {
	t.Helper()
	ch := newTestChannel(t)
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
	ch, b := newTestChannelWithConfig(t, config.PicoConfig{
		Enabled: true,
		Token:   testToken,
	})
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
	msg, ok := b.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message on bus, got none")
	}
	return msg
}

// --- Rejection tests ---

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
	ch := newTestChannel(t) // not started

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

// --- Acceptance tests ---

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

// --- Bus semantics tests ---

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

// --- Auth variant tests ---

func TestHTTPMessage_QueryTokenAccepted(t *testing.T) {
	ch, _ := newTestChannelWithConfig(t, config.PicoConfig{
		Enabled:         true,
		Token:           testToken,
		AllowTokenQuery: true,
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

// --- Routing tests ---

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

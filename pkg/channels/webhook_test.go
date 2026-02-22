package channels

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewWebhookChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, err := NewWebhookChannel(config.WebhookConfig{}, msgBus)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if ch.Name() != "webhook" {
		t.Fatalf("Name() = %q, want webhook", ch.Name())
	}
	if ch.IsRunning() {
		t.Fatal("new channel should not be running")
	}
}

func TestWebhookPayloadContent(t *testing.T) {
	t.Run("uses content field", func(t *testing.T) {
		content, _ := webhookPayloadContent(map[string]any{"content": "hello"})
		if content != "hello" {
			t.Fatalf("content = %q, want hello", content)
		}
	})

	t.Run("uses text fallback", func(t *testing.T) {
		content, _ := webhookPayloadContent(map[string]any{"text": "hello text"})
		if content != "hello text" {
			t.Fatalf("content = %q, want hello text", content)
		}
	})

	t.Run("uses message fallback", func(t *testing.T) {
		content, _ := webhookPayloadContent(map[string]any{"message": "hello message"})
		if content != "hello message" {
			t.Fatalf("content = %q, want hello message", content)
		}
	})

	t.Run("falls back to serialized payload", func(t *testing.T) {
		content, payloadJSON := webhookPayloadContent(map[string]any{"z": 1})
		if content != payloadJSON {
			t.Fatalf("content should equal payload JSON fallback, got %q vs %q", content, payloadJSON)
		}
		if !strings.Contains(content, `"z":1`) {
			t.Fatalf("fallback JSON missing expected key: %q", content)
		}
	})
}

func TestWebhookHandleInboundPublishesInbound(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WebhookConfig{}
	ch, err := NewWebhookChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-webhook-sender-id", "sender-1")
	req.Header.Set("x-webhook-chat-id", "chat-1")
	req.Header.Set("x-request-id", "req-123")

	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published")
	}
	if msg.Channel != "webhook" {
		t.Fatalf("channel = %q, want webhook", msg.Channel)
	}
	if msg.SenderID != "sender-1" {
		t.Fatalf("sender = %q", msg.SenderID)
	}
	if msg.ChatID != "chat-1" {
		t.Fatalf("chatID = %q", msg.ChatID)
	}
	if msg.Content != "hello" {
		t.Fatalf("content = %q", msg.Content)
	}
	if msg.Metadata["request_id"] != "req-123" {
		t.Fatalf("request_id metadata = %q", msg.Metadata["request_id"])
	}
	if msg.Metadata["platform"] != "webhook" {
		t.Fatalf("platform metadata = %q", msg.Metadata["platform"])
	}
}

func TestWebhookHandleInboundAcceptsOptionalClawdentityMetadata(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-clawdentity-agent-did", "did:example:sender")
	req.Header.Set("x-clawdentity-to-agent-did", "did:example:receiver")

	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published")
	}
	if msg.SenderID != "did:example:sender" {
		t.Fatalf("sender = %q", msg.SenderID)
	}
	if msg.ChatID != "did:example:receiver" {
		t.Fatalf("chatID = %q", msg.ChatID)
	}
	if msg.Metadata["clawdentity_agent_did"] != "did:example:sender" {
		t.Fatalf("clawdentity_agent_did metadata = %q", msg.Metadata["clawdentity_agent_did"])
	}
	if msg.Metadata["clawdentity_to_agent_did"] != "did:example:receiver" {
		t.Fatalf("clawdentity_to_agent_did metadata = %q", msg.Metadata["clawdentity_to_agent_did"])
	}
}

func TestWebhookHandleInboundUsesBodyUserIDAsSender(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"userId":"user-123","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published")
	}
	if msg.SenderID != "user-123" {
		t.Fatalf("sender = %q", msg.SenderID)
	}
	if msg.ChatID != "user-123" {
		t.Fatalf("chatID = %q", msg.ChatID)
	}
}

func TestWebhookHandleInboundUsesBodyUserIDAndChatID(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"userId":"user-123","chatId":"chat-456","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published")
	}
	if msg.SenderID != "user-123" {
		t.Fatalf("sender = %q", msg.SenderID)
	}
	if msg.ChatID != "chat-456" {
		t.Fatalf("chatID = %q", msg.ChatID)
	}
}

func TestWebhookHandleInboundHeadersTakePrecedenceOverBodyFields(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"userId":"user-body","chatId":"chat-body","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-webhook-sender-id", "sender-header")
	req.Header.Set("x-webhook-chat-id", "chat-header")

	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published")
	}
	if msg.SenderID != "sender-header" {
		t.Fatalf("sender = %q", msg.SenderID)
	}
	if msg.ChatID != "chat-header" {
		t.Fatalf("chatID = %q", msg.ChatID)
	}
}

func TestWebhookHandleInboundRejectsMethod(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodGet, "/v1/inbound", nil)
	rr := httptest.NewRecorder()
	ch.handleInbound(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestWebhookHandleInboundRejectsMissingJSONContentType(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":"hello"}`))
	rr := httptest.NewRecorder()

	ch.handleInbound(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestWebhookHandleInboundRejectsInvalidToken(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WebhookConfig{Token: "expected-token"}
	ch, _ := NewWebhookChannel(cfg, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-webhook-token", "wrong-token")
	rr := httptest.NewRecorder()

	ch.handleInbound(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestWebhookHandleInboundAcceptsBearerToken(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WebhookConfig{Token: "expected-token"}
	ch, _ := NewWebhookChannel(cfg, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expected-token")
	req.Header.Set("x-webhook-sender-id", "sender-1")
	rr := httptest.NewRecorder()

	ch.handleInbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
}

func TestWebhookHandleInboundRejectsMalformedJSON(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-webhook-sender-id", "sender-1")
	rr := httptest.NewRecorder()

	ch.handleInbound(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestWebhookHandleInboundRejectsUnauthorizedSender(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WebhookConfig{
		AllowFrom: []string{"sender-allowed"},
	}
	ch, _ := NewWebhookChannel(cfg, msgBus)

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-webhook-sender-id", "sender-blocked")
	rr := httptest.NewRecorder()

	ch.handleInbound(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, ok := msgBus.ConsumeInbound(ctx); ok {
		t.Fatal("unexpected inbound message for unauthorized sender")
	}
}

func TestWebhookHandleOutboundForwardsPayload(t *testing.T) {
	type receivedPayload struct {
		To      string `json:"to"`
		Content string `json:"content"`
		Peer    string `json:"peer"`
	}
	received := make(chan receivedPayload, 1)

	connector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload receivedPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		received <- payload
		w.WriteHeader(http.StatusAccepted)
	}))
	defer connector.Close()

	ch, _ := NewWebhookChannel(config.WebhookConfig{ConnectorURL: connector.URL}, bus.NewMessageBus())

	req := httptest.NewRequest(http.MethodPost, "/v1/outbound", strings.NewReader(`{"to":"did:example:peer","content":"hello","peer":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ch.handleOutbound(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	select {
	case payload := <-received:
		if payload.To != "did:example:peer" {
			t.Fatalf("to = %q", payload.To)
		}
		if payload.Content != "hello" {
			t.Fatalf("content = %q", payload.Content)
		}
		if payload.Peer != "alice" {
			t.Fatalf("peer = %q", payload.Peer)
		}
	case <-time.After(time.Second):
		t.Fatal("expected outbound payload to be forwarded")
	}
}

func TestWebhookHandleOutboundRejectsMissingFields(t *testing.T) {
	ch, _ := NewWebhookChannel(config.WebhookConfig{}, bus.NewMessageBus())

	req := httptest.NewRequest(http.MethodPost, "/v1/outbound", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ch.handleOutbound(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/outbound", strings.NewReader(`{"to":"did:example:peer"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	ch.handleOutbound(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestWebhookHandleOutboundRejectsConnectorFailure(t *testing.T) {
	connector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer connector.Close()

	ch, _ := NewWebhookChannel(config.WebhookConfig{ConnectorURL: connector.URL}, bus.NewMessageBus())

	req := httptest.NewRequest(http.MethodPost, "/v1/outbound", strings.NewReader(`{"to":"did:example:peer","content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ch.handleOutbound(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

func TestWebhookSendForwardsToConnector(t *testing.T) {
	type receivedPayload struct {
		To      string `json:"to"`
		Content string `json:"content"`
	}
	received := make(chan receivedPayload, 1)

	connector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload receivedPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		received <- payload
		w.WriteHeader(http.StatusAccepted)
	}))
	defer connector.Close()

	ch, _ := NewWebhookChannel(config.WebhookConfig{ConnectorURL: connector.URL}, bus.NewMessageBus())

	if err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "webhook",
		ChatID:  "did:example:peer",
		Content: "hello",
	}); err != nil {
		t.Fatalf("send error: %v", err)
	}

	select {
	case payload := <-received:
		if payload.To != "did:example:peer" {
			t.Fatalf("to = %q", payload.To)
		}
		if payload.Content != "hello" {
			t.Fatalf("content = %q", payload.Content)
		}
	case <-time.After(time.Second):
		t.Fatal("expected outbound payload to be forwarded")
	}
}

func TestWebhookStartRejectsDuplicatePaths(t *testing.T) {
	ch, err := NewWebhookChannel(config.WebhookConfig{
		WebhookHost: "127.0.0.1",
		WebhookPort: 0,
		WebhookPath: "/same-path",
		SendPath:    "/same-path",
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	err = ch.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start() error for duplicate paths")
	}
	if !strings.Contains(err.Error(), `both are "/same-path"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.IsRunning() {
		t.Fatal("channel should not be running after failed Start()")
	}
}

func TestWebhookStartRejectsPortInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen setup: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	ch, err := NewWebhookChannel(config.WebhookConfig{
		WebhookHost: "127.0.0.1",
		WebhookPort: port,
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	err = ch.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start() error when port is already in use")
	}
	if ch.IsRunning() {
		t.Fatal("channel should not be running after bind failure")
	}
}

func TestWebhookSendRespectsCanceledContext(t *testing.T) {
	connector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer connector.Close()

	ch, err := NewWebhookChannel(config.WebhookConfig{ConnectorURL: connector.URL}, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = ch.Send(ctx, bus.OutboundMessage{
		Channel: "webhook",
		ChatID:  "did:example:peer",
		Content: "hello",
	})
	if err == nil {
		t.Fatal("expected send error with canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

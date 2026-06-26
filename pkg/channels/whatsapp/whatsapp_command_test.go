package whatsapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestHandleIncomingMessage_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp", config.WhatsAppSettings{}, messageBus, nil),
		ctx:         context.Background(),
	}

	ch.handleIncomingMessage(map[string]any{
		"type":    "message",
		"id":      "mid1",
		"from":    "user1",
		"chat":    "chat1",
		"content": "/help",
	})

	inbound, ok := <-messageBus.InboundChan()
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "whatsapp" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/help" {
		t.Fatalf("content=%q", inbound.Content)
	}
}

func TestListenReconnectsAfterBridgeDisconnect(t *testing.T) {
	var connections atomic.Int32
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}

		if connections.Add(1) == 1 {
			_ = conn.Close()
			return
		}
		defer conn.Close()

		err = conn.WriteJSON(map[string]any{
			"type":    "message",
			"id":      "after-reconnect",
			"from":    "user1",
			"chat":    "chat1",
			"content": "still connected",
		})
		if err != nil {
			t.Logf("write message: %v", err)
			return
		}

		<-r.Context().Done()
	}))
	defer srv.Close()

	messageBus := bus.NewMessageBus()
	channelConfig := &config.Channel{Type: config.ChannelWhatsApp, Enabled: true}
	settings := &config.WhatsAppSettings{BridgeURL: "ws" + strings.TrimPrefix(srv.URL, "http")}
	ch, err := NewWhatsAppChannel(channelConfig, settings, messageBus)
	if err != nil {
		t.Fatalf("NewWhatsAppChannel: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(context.Background())

	select {
	case inbound := <-messageBus.InboundChan():
		if inbound.Content != "still connected" {
			t.Fatalf("content=%q", inbound.Content)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for message after reconnect")
	}

	if got := connections.Load(); got < 2 {
		t.Fatalf("connections=%d, want reconnect", got)
	}
}

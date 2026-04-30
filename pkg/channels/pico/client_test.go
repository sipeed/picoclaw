package pico

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewPicoClientChannel_MissingURL(t *testing.T) {
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	_, err := NewPicoClientChannel(bc, &config.PicoClientSettings{}, bus.NewMessageBus())
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPicoClientChannel_OK(t *testing.T) {
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL: "ws://localhost:9999/ws",
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Name() != "pico_client" {
		t.Fatalf("name = %q, want pico_client", ch.Name())
	}
}

func TestSend_NotRunning(t *testing.T) {
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL: "ws://localhost:9999/ws",
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ch.Send(context.Background(), bus.OutboundMessage{Content: "hi"})
	if !errors.Is(err, channels.ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning, got %v", err)
	}
}

// testServer starts a WS server that echoes message.send back as message.create.
func testServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg PicoMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			if msg.Type == TypeMessageSend {
				reply := newMessage(TypeMessageCreate, msg.Payload)
				reply.SessionID = msg.SessionID
				if err := conn.WriteJSON(reply); err != nil {
					return
				}
			}
		}
	}))
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func TestClientChannel_ConnectAndSend(t *testing.T) {
	srv := testServer(t, "test-token")
	defer srv.Close()

	mb := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL:          wsURL(srv.URL),
		Token:        *config.NewSecureString("test-token"),
		SessionID:    "sess-1",
		PingInterval: 60,
		ReadTimeout:  10,
	}, mb)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	// Send a message
	_, err = ch.Send(ctx, bus.OutboundMessage{
		ChatID:  "pico_client:sess-1",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestClientChannel_AuthFailure(t *testing.T) {
	srv := testServer(t, "correct-token")
	defer srv.Close()

	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL:   wsURL(srv.URL),
		Token: *config.NewSecureString("wrong-token"),
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = ch.Start(ctx)
	if err == nil {
		ch.Stop(ctx)
		t.Fatal("expected auth failure")
	}
}

func TestClientChannel_ReceivesServerMessage(t *testing.T) {
	srv := testServer(t, "")
	defer srv.Close()

	mb := bus.NewMessageBus()

	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL:         wsURL(srv.URL),
		SessionID:   "sess-echo",
		ReadTimeout: 10,
	}, mb)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	// Send a message; the echo server replies with message.create
	_, err = ch.Send(ctx, bus.OutboundMessage{
		ChatID:  "pico_client:sess-echo",
		Content: "ping",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	// The echoed message.create is processed by handleServerMessage which
	// calls HandleMessage → PublishInbound. Consume it from the bus.
	select {
	case msg := <-mb.InboundChan():
		if msg.Content != "ping" {
			t.Fatalf("received = %q, want %q", msg.Content, "ping")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for echoed message")
	}
}

func TestClientChannel_StartTyping(t *testing.T) {
	srv := testServer(t, "")
	defer srv.Close()

	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL:         wsURL(srv.URL),
		SessionID:   "sess-type",
		ReadTimeout: 10,
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	stop, err := ch.StartTyping(ctx, "pico_client:sess-type")
	if err != nil {
		t.Fatalf("StartTyping: %v", err)
	}
	stop() // should not panic
}

func TestSend_ClosedConnection(t *testing.T) {
	srv := testServer(t, "")
	defer srv.Close()

	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL:         wsURL(srv.URL),
		SessionID:   "sess-close",
		ReadTimeout: 10,
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Force close the underlying connection
	ch.mu.Lock()
	ch.conn.close()
	ch.mu.Unlock()

	_, err = ch.Send(ctx, bus.OutboundMessage{
		ChatID:  "pico_client:sess-close",
		Content: "should fail",
	})
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("expected ErrSendFailed, got %v", err)
	}

	ch.Stop(ctx)
}

func TestParseInlineImageMedia_Valid(t *testing.T) {
	media, err := parseInlineImageMedia(map[string]any{
		"media": []any{
			"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+X2ioAAAAASUVORK5CYII=",
		},
	})
	if err != nil {
		t.Fatalf("parseInlineImageMedia() error = %v", err)
	}
	if len(media) != 1 {
		t.Fatalf("len(media) = %d, want 1", len(media))
	}
}

func TestPicoChannel_HandleMessageSend_AllowsMediaOnly(t *testing.T) {
	mb := bus.NewMessageBus()
	bc := &config.Channel{Type: "pico", Enabled: true}
	ch, err := NewPicoChannel(bc, &config.PicoSettings{
		Token: *config.NewSecureString("test-token"),
	}, mb)
	if err != nil {
		t.Fatalf("NewPicoChannel() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(ctx)

	pc := &picoConn{id: "conn-1", sessionID: "sess-1"}
	ch.handleMessageSend(pc, PicoMessage{
		ID: "msg-1",
		Payload: map[string]any{
			"media": []any{
				"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+X2ioAAAAASUVORK5CYII=",
			},
		},
	})

	select {
	case msg := <-mb.InboundChan():
		if msg.Content != "" {
			t.Fatalf("msg.Content = %q, want empty", msg.Content)
		}
		if len(msg.Media) != 1 || !strings.HasPrefix(msg.Media[0], "data:image/png;base64,") {
			t.Fatalf("msg.Media = %#v, want inline image payload", msg.Media)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for inbound media message")
	}
}

func TestIsThoughtPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		want    bool
	}{
		{
			name:    "explicit thought kind",
			payload: map[string]any{PayloadKeyKind: MessageKindThought},
			want:    true,
		},
		{
			name:    "thought kind ignores case and whitespace",
			payload: map[string]any{PayloadKeyKind: "  ThOuGhT  "},
			want:    true,
		},
		{
			name:    "legacy thought bool remains supported for inbound compatibility",
			payload: map[string]any{PayloadKeyThought: true},
			want:    true,
		},
		{
			name:    "legacy thought false",
			payload: map[string]any{PayloadKeyThought: false},
			want:    false,
		},
		{
			name:    "tool calls kind",
			payload: map[string]any{PayloadKeyKind: MessageKindToolCalls},
			want:    false,
		},
		{
			name:    "non-string kind ignored",
			payload: map[string]any{PayloadKeyKind: true},
			want:    false,
		},
		{
			name:    "default normal",
			payload: map[string]any{PayloadKeyContent: "hello"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isThoughtPayload(tt.payload); got != tt.want {
				t.Fatalf("isThoughtPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPicoClientChannel_HandleServerMessage_IgnoresThought(t *testing.T) {
	mb := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL: "ws://localhost:8080/ws",
	}, mb)
	if err != nil {
		t.Fatalf("NewPicoClientChannel() error = %v", err)
	}

	ch.ctx = context.Background()
	pc := &picoConn{sessionID: "sess-thought"}

	ch.handleServerMessage(pc, PicoMessage{
		Type: TypeMessageCreate,
		Payload: map[string]any{
			PayloadKeyContent: "internal reasoning",
			PayloadKeyKind:    MessageKindThought,
		},
	})

	select {
	case msg := <-mb.InboundChan():
		t.Fatalf("expected no inbound publish for thought payload, got %+v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestPicoClientChannel_HandleServerMessage_IgnoresLegacyThoughtBool(t *testing.T) {
	mb := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelPicoClient, Enabled: true}
	ch, err := NewPicoClientChannel(bc, &config.PicoClientSettings{
		URL: "ws://localhost:8080/ws",
	}, mb)
	if err != nil {
		t.Fatalf("NewPicoClientChannel() error = %v", err)
	}

	ch.ctx = context.Background()
	pc := &picoConn{sessionID: "sess-thought-legacy"}

	ch.handleServerMessage(pc, PicoMessage{
		Type: TypeMessageCreate,
		Payload: map[string]any{
			PayloadKeyContent: "legacy internal reasoning",
			PayloadKeyThought: true,
		},
	})

	select {
	case msg := <-mb.InboundChan():
		t.Fatalf("expected no inbound publish for legacy thought payload, got %+v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

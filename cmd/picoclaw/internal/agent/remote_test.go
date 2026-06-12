package agent

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/channels/pico"
)

type testPrinter struct {
	bytes.Buffer
}

func (p *testPrinter) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(&p.Buffer, format, args...)
}

func TestRemoteURLWithSession(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		session string
		want    map[string]string
	}{
		{
			name:    "adds session query",
			rawURL:  "ws://localhost:18790/pico/ws",
			session: "abc",
			want:    map[string]string{"scheme": "ws", "session_id": "abc"},
		},
		{
			name:    "preserves query params",
			rawURL:  "wss://example.test/pico/ws?foo=bar",
			session: "abc",
			want:    map[string]string{"scheme": "wss", "session_id": "abc", "foo": "bar"},
		},
		{
			name:    "replaces stale session",
			rawURL:  "ws://example.test/pico/ws?session_id=old&foo=bar",
			session: "new",
			want:    map[string]string{"scheme": "ws", "session_id": "new", "foo": "bar"},
		},
		{
			name:    "converts http to ws",
			rawURL:  "http://example.test/pico/ws",
			session: "abc",
			want:    map[string]string{"scheme": "ws", "session_id": "abc"},
		},
		{
			name:    "adds missing scheme",
			rawURL:  "localhost:18790/pico/ws",
			session: "abc",
			want:    map[string]string{"scheme": "ws", "session_id": "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := remoteURLWithSession(tt.rawURL, tt.session)
			if err != nil {
				t.Fatalf("remoteURLWithSession() error = %v", err)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("url.Parse(%q) error = %v", got, err)
			}
			if gotScheme := u.Scheme; gotScheme != tt.want["scheme"] {
				t.Fatalf("scheme = %q, want %q", gotScheme, tt.want["scheme"])
			}
			for key, want := range tt.want {
				if key == "scheme" {
					continue
				}
				if got := u.Query().Get(key); got != want {
					t.Fatalf("query %s = %q, want %q in %q", key, got, want, u.String())
				}
			}
		})
	}
}

func TestRemoteAuthHeader(t *testing.T) {
	if got := remoteAuthHeader("").Get("Authorization"); got != "" {
		t.Fatalf("empty token Authorization = %q, want empty", got)
	}
	if got := remoteAuthHeader("secret").Get("Authorization"); got != "Bearer secret" {
		t.Fatalf("token Authorization = %q, want Bearer secret", got)
	}
}

func TestBuildRemoteMessageSend(t *testing.T) {
	msg := buildRemoteMessageSend("sess-1", "hello")
	if msg.Type != pico.TypeMessageSend {
		t.Fatalf("Type = %q, want %q", msg.Type, pico.TypeMessageSend)
	}
	if msg.SessionID != "sess-1" {
		t.Fatalf("SessionID = %q, want sess-1", msg.SessionID)
	}
	if got := msg.Payload[pico.PayloadKeyContent]; got != "hello" {
		t.Fatalf("content = %#v, want hello", got)
	}
	if got := msg.Payload[pico.PayloadKeyClientKind]; got != pico.ClientKindRemoteCLI {
		t.Fatalf("client_kind = %#v, want %s", got, pico.ClientKindRemoteCLI)
	}
	if got := msg.Payload[pico.PayloadKeyClientName]; got != "picoclaw agent --remote" {
		t.Fatalf("client_name = %#v, want picoclaw agent --remote", got)
	}
	if got := msg.Payload[pico.PayloadKeyTransport]; got != pico.TransportWebSocket {
		t.Fatalf("transport = %#v, want %s", got, pico.TransportWebSocket)
	}
	if msg.ID == "" {
		t.Fatal("ID is empty")
	}
	if msg.Timestamp == 0 {
		t.Fatal("Timestamp is empty")
	}
}

func TestRenderRemoteEvent(t *testing.T) {
	tests := []struct {
		name       string
		msg        pico.PicoMessage
		want       string
		displayed  bool
		allowEmpty bool
	}{
		{
			name: "message create",
			msg: pico.PicoMessage{
				Type:    pico.TypeMessageCreate,
				Payload: map[string]any{pico.PayloadKeyContent: "hello"},
			},
			want:      "hello",
			displayed: true,
		},
		{
			name: "message update",
			msg: pico.PicoMessage{
				Type:    pico.TypeMessageUpdate,
				Payload: map[string]any{pico.PayloadKeyContent: "updated"},
			},
			want:      "updated",
			displayed: true,
		},
		{
			name: "typing start",
			msg:  pico.PicoMessage{Type: pico.TypeTypingStart},
			want: "[typing]",
		},
		{
			name:       "typing stop",
			msg:        pico.PicoMessage{Type: pico.TypeTypingStop},
			allowEmpty: true,
		},
		{
			name: "message delete",
			msg: pico.PicoMessage{
				Type:    pico.TypeMessageDelete,
				Payload: map[string]any{"message_id": "msg-1"},
			},
			want:      "[message deleted: msg-1]",
			displayed: true,
		},
		{
			name: "media create",
			msg: pico.PicoMessage{
				Type: pico.TypeMediaCreate,
				Payload: map[string]any{
					pico.PayloadKeyContent: "media text",
					"media":                []any{"data:image/png;base64,abc"},
				},
			},
			want:      "[media] data:image/png;base64,abc",
			displayed: true,
		},
		{
			name: "error",
			msg: pico.PicoMessage{
				Type:    pico.TypeError,
				Payload: map[string]any{"code": "bad", "message": "failed"},
			},
			want:      "error[bad]: failed",
			displayed: true,
		},
		{
			name: "pong",
			msg:  pico.PicoMessage{Type: pico.TypePong},
			want: "[pong]",
		},
		{
			name: "unknown",
			msg: pico.PicoMessage{
				Type:    "custom.event",
				Payload: map[string]any{"value": "ok"},
			},
			want: "[event custom.event]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out testPrinter
			displayed := renderRemoteEvent(&out, tt.msg)
			if displayed != tt.displayed {
				t.Fatalf("displayed = %v, want %v", displayed, tt.displayed)
			}
			if tt.allowEmpty && out.String() == "" {
				return
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output = %q, want to contain %q", out.String(), tt.want)
			}
		})
	}
}

func TestRemoteTypingLineClears(t *testing.T) {
	var out bytes.Buffer
	writer := &lockedWriter{w: &out}
	refreshes := 0
	writer.SetRefreshPrompt(func() { refreshes++ })

	renderRemoteEvent(writer, pico.PicoMessage{Type: pico.TypeTypingStart})
	renderRemoteEvent(writer, pico.PicoMessage{Type: pico.TypeTypingStop})

	got := out.String()
	if !strings.Contains(got, "[typing]\n") {
		t.Fatalf("output = %q, want typing line", got)
	}
	if !strings.Contains(got, "\033[1A\r\033[2K\033[1M") {
		t.Fatalf("output = %q, want typing clear sequence", got)
	}
	if strings.Contains(got, "[typing stopped]") {
		t.Fatalf("output = %q, should not contain typing stopped line", got)
	}
	if refreshes != 2 {
		t.Fatalf("refreshes = %d, want 2", refreshes)
	}
}

func TestRemoteOneShotWebSocket(t *testing.T) {
	const sessionID = "test-session"
	const token = "test-token"

	gotSession := make(chan string, 1)
	gotMessage := make(chan pico.PicoMessage, 1)

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/ws" {
			t.Errorf("path = %q, want /pico/ws", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q, want Bearer %s", got, token)
		}
		gotSession <- r.URL.Query().Get("session_id")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		defer conn.Close()

		var msg pico.PicoMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Errorf("ReadJSON() error = %v", err)
			return
		}
		gotMessage <- msg

		reply := pico.PicoMessage{
			Type:      pico.TypeMessageCreate,
			SessionID: msg.SessionID,
			Timestamp: time.Now().UnixMilli(),
			Payload:   map[string]any{pico.PayloadKeyContent: "remote response"},
		}
		if err := conn.WriteJSON(reply); err != nil {
			t.Errorf("WriteJSON() error = %v", err)
			return
		}
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
	}))
	defer srv.Close()

	remoteURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/pico/ws?foo=bar"
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := remoteAgentCmd(ctx, remoteURL, token, "hello", sessionID, strings.NewReader(""), &out); err != nil {
		t.Fatalf("remoteAgentCmd() error = %v", err)
	}

	select {
	case got := <-gotSession:
		if got != sessionID {
			t.Fatalf("connected session_id = %q, want %q", got, sessionID)
		}
	default:
		t.Fatal("server did not record connected session_id")
	}

	select {
	case msg := <-gotMessage:
		if msg.Type != pico.TypeMessageSend {
			t.Fatalf("sent type = %q, want %q", msg.Type, pico.TypeMessageSend)
		}
		if msg.SessionID != sessionID {
			t.Fatalf("sent session_id = %q, want %q", msg.SessionID, sessionID)
		}
		if got := msg.Payload[pico.PayloadKeyContent]; got != "hello" {
			t.Fatalf("sent content = %#v, want hello", got)
		}
	default:
		t.Fatal("server did not record sent message")
	}

	if !strings.Contains(out.String(), "remote response") {
		t.Fatalf("output = %q, want remote response", out.String())
	}
}

func TestRemoteOneShotUsesPicoTokenEnvFallback(t *testing.T) {
	const sessionID = "env-session"
	const token = "env-token"
	t.Setenv("PICO_TOKEN", token)

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q, want Bearer %s", got, token)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		defer conn.Close()

		var msg pico.PicoMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Errorf("ReadJSON() error = %v", err)
			return
		}
		_ = conn.WriteJSON(pico.PicoMessage{
			Type:      pico.TypeMessageCreate,
			SessionID: msg.SessionID,
			Payload:   map[string]any{pico.PayloadKeyContent: "env ok"},
		})
	}))
	defer srv.Close()

	remoteURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/pico/ws"
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := remoteAgentCmd(ctx, remoteURL, "", "hello", sessionID, strings.NewReader(""), &out); err != nil {
		t.Fatalf("remoteAgentCmd() error = %v", err)
	}
	if !strings.Contains(out.String(), "env ok") {
		t.Fatalf("output = %q, want env ok", out.String())
	}
}

func TestRemoteOneShotReportsHandshakeStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	remoteURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/pico/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := remoteAgentCmd(ctx, remoteURL, "", "hello", "auth-session", strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("remoteAgentCmd() error = nil, want handshake error")
	}
	if got := err.Error(); !strings.Contains(got, "HTTP 401 Unauthorized") || !strings.Contains(got, "unauthorized") {
		t.Fatalf("error = %q, want HTTP status and body", got)
	}
}

func TestRemoteCommandExecutesRemoteMode(t *testing.T) {
	const sessionID = "cmd-session"
	gotMessage := make(chan pico.PicoMessage, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("session_id"); got != sessionID {
			t.Errorf("session_id = %q, want %q", got, sessionID)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		defer conn.Close()

		var msg pico.PicoMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Errorf("ReadJSON() error = %v", err)
			return
		}
		gotMessage <- msg
		_ = conn.WriteJSON(pico.PicoMessage{
			Type:      pico.TypeMessageCreate,
			SessionID: msg.SessionID,
			Payload:   map[string]any{pico.PayloadKeyContent: "ok"},
		})
	}))
	defer srv.Close()

	cmd := NewAgentCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{
		"--remote", "ws" + strings.TrimPrefix(srv.URL, "http") + "/pico/ws",
		"--session", sessionID,
		"--message", "hello",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	select {
	case msg := <-gotMessage:
		if msg.Type != pico.TypeMessageSend {
			t.Fatalf("type = %q, want %q", msg.Type, pico.TypeMessageSend)
		}
		if msg.SessionID != sessionID {
			t.Fatalf("session_id = %q, want %q", msg.SessionID, sessionID)
		}
	default:
		t.Fatal("server did not receive command message")
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("output = %q, want ok", out.String())
	}
}

func TestRemoteCommandGeneratesSessionWhenOmitted(t *testing.T) {
	gotSession := make(chan string, 1)
	gotMessage := make(chan pico.PicoMessage, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession <- r.URL.Query().Get("session_id")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		defer conn.Close()

		var msg pico.PicoMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Errorf("ReadJSON() error = %v", err)
			return
		}
		gotMessage <- msg
		_ = conn.WriteJSON(pico.PicoMessage{
			Type:      pico.TypeMessageCreate,
			SessionID: msg.SessionID,
			Payload:   map[string]any{pico.PayloadKeyContent: "ok"},
		})
	}))
	defer srv.Close()

	cmd := NewAgentCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{
		"--remote", "ws" + strings.TrimPrefix(srv.URL, "http") + "/pico/ws",
		"--message", "hello",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	var connectedSession string
	select {
	case connectedSession = <-gotSession:
	default:
		t.Fatal("server did not receive generated session_id")
	}
	if connectedSession == "" {
		t.Fatal("generated session_id is empty")
	}
	if connectedSession == "cli:default" {
		t.Fatal("remote command used local default session cli:default")
	}
	if !strings.HasPrefix(connectedSession, remoteGeneratedSessionPrefix) {
		t.Fatalf("generated session_id = %q, want %s prefix", connectedSession, remoteGeneratedSessionPrefix)
	}

	select {
	case msg := <-gotMessage:
		if msg.SessionID != connectedSession {
			t.Fatalf("message session_id = %q, want %q", msg.SessionID, connectedSession)
		}
	default:
		t.Fatal("server did not receive command message")
	}
}

package discord

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
)

// fakeGateway answers the Discord REST gateway lookup and plays the websocket
// side of a handshake: Hello, read the client's Identify/Resume, then reply
// with firstPacket instead of READY.
func fakeGateway(t *testing.T, firstPacket string) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gateway" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"url":"ws`+strings.TrimPrefix(server.URL, "http")+`/ws"}`)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		hello := []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`)
		if err := conn.WriteMessage(websocket.TextMessage, hello); err != nil {
			return
		}
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(firstPacket)); err != nil {
			return
		}
		// Hold the socket open until the client closes it.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	return server
}

// Discord answers a failed resume with Op 9 (Invalid Session), and can send
// Op 7 (Reconnect) at any time. When either arrives as the first packet of a
// handshake, Open must give the session lock back. A discordgo that closes the
// socket from inside Open (CloseWithCode takes the lock Open is holding) wedges
// the channel for good: no reconnect, no heartbeat, nothing in the logs.
func TestGatewayOpenReleasesSessionOnReconnectRequest(t *testing.T) {
	for name, packet := range map[string]string{
		"op9 invalid session": `{"op":9,"d":false}`,
		"op7 reconnect":       `{"op":7,"d":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := fakeGateway(t, packet)
			defer server.Close()

			origGateway := discordgo.EndpointGateway
			discordgo.EndpointGateway = server.URL + "/gateway"
			defer func() { discordgo.EndpointGateway = origGateway }()

			session, err := discordgo.New("Bot test-token")
			if err != nil {
				t.Fatalf("discordgo.New() error: %v", err)
			}
			session.Client = server.Client()
			// Not websocket.DefaultDialer: its Proxy is http.ProxyFromEnvironment, which
			// caches the environment on first use and would break TestApplyDiscordProxy_*.
			session.Dialer = &websocket.Dialer{}
			session.ShouldReconnectOnError = false

			opened := make(chan error, 1)
			go func() { opened <- session.Open() }()
			select {
			case <-opened:
				// Any result is fine; the gateway asked for a new connection.
			case <-time.After(5 * time.Second):
				t.Fatal("session.Open() never returned: deadlocked on the session lock")
			}

			closed := make(chan error, 1)
			go func() { closed <- session.Close() }()
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("session.Close() never returned: the session lock was not released")
			}
		})
	}
}

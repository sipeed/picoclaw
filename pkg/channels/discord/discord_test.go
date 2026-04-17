package discord

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

func TestApplyDiscordProxy_CustomProxy(t *testing.T) {
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("discordgo.New() error: %v", err)
	}

	if err = applyDiscordProxy(session, "http://127.0.0.1:7890"); err != nil {
		t.Fatalf("applyDiscordProxy() error: %v", err)
	}

	req, err := http.NewRequest("GET", "https://discord.com/api/v10/gateway", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}

	restProxy := session.Client.Transport.(*http.Transport).Proxy
	restProxyURL, err := restProxy(req)
	if err != nil {
		t.Fatalf("rest proxy func error: %v", err)
	}
	if got, want := restProxyURL.String(), "http://127.0.0.1:7890"; got != want {
		t.Fatalf("REST proxy = %q, want %q", got, want)
	}

	wsProxyURL, err := session.Dialer.Proxy(req)
	if err != nil {
		t.Fatalf("ws proxy func error: %v", err)
	}
	if got, want := wsProxyURL.String(), "http://127.0.0.1:7890"; got != want {
		t.Fatalf("WS proxy = %q, want %q", got, want)
	}
}

func TestApplyDiscordProxy_FromEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("http_proxy", "http://127.0.0.1:8888")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("https_proxy", "http://127.0.0.1:8888")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("discordgo.New() error: %v", err)
	}

	if err = applyDiscordProxy(session, ""); err != nil {
		t.Fatalf("applyDiscordProxy() error: %v", err)
	}

	req, err := http.NewRequest("GET", "https://discord.com/api/v10/gateway", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}

	gotURL, err := session.Dialer.Proxy(req)
	if err != nil {
		t.Fatalf("ws proxy func error: %v", err)
	}

	wantURL, err := url.Parse("http://127.0.0.1:8888")
	if err != nil {
		t.Fatalf("url.Parse() error: %v", err)
	}
	if gotURL.String() != wantURL.String() {
		t.Fatalf("WS proxy = %q, want %q", gotURL.String(), wantURL.String())
	}
}

func TestApplyDiscordProxy_InvalidProxyURL(t *testing.T) {
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("discordgo.New() error: %v", err)
	}

	if err = applyDiscordProxy(session, "://bad-proxy"); err == nil {
		t.Fatal("applyDiscordProxy() expected error for invalid proxy URL, got nil")
	}
}

func TestSend_NonToolFeedbackDeletesTrackedProgressMessage(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/channels/chat-1/messages/prog-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/channels/chat-1/messages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"final-1"}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	origChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	defer func() {
		discordgo.EndpointChannels = origChannels
	}()

	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("discordgo.New() error: %v", err)
	}
	session.Client = server.Client()

	ch := &DiscordChannel{
		BaseChannel: channels.NewBaseChannel("discord", nil, bus.NewMessageBus(), nil),
		session:     session,
		ctx:         context.Background(),
		typingStop:  make(map[string]chan struct{}),
		voiceSSRC:   make(map[string]map[uint32]string),
	}
	ch.progress = channels.NewToolFeedbackAnimator(ch.EditMessage)
	ch.SetRunning(true)
	ch.RecordToolFeedbackMessage("chat-1", "prog-1", "🔧 `read_file`")

	ids, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "chat-1",
		Content: "final reply",
		Context: bus.InboundContext{
			Channel: "discord",
			ChatID:  "chat-1",
		},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got, want := ids, []string{"final-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Send() ids = %v, want %v", got, want)
	}
	if _, ok := ch.currentToolFeedbackMessage("chat-1"); ok {
		t.Fatal("expected tracked tool feedback message to be cleared")
	}

	mu.Lock()
	defer mu.Unlock()
	wantRequests := []string{
		"DELETE /channels/chat-1/messages/prog-1",
		"POST /channels/chat-1/messages",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %v, want %v", requests, wantRequests)
	}
}

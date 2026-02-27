//go:build discord_proxy
// +build discord_proxy

package channels

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestApplyDiscordProxy_CustomProxy(t *testing.T) {
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("discordgo.New() error: %v", err)
	}

	if err := applyDiscordProxy(session, "http://127.0.0.1:7890"); err != nil {
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

	if err := applyDiscordProxy(session, ""); err != nil {
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

	if err := applyDiscordProxy(session, "://bad-proxy"); err == nil {
		t.Fatal("applyDiscordProxy() expected error for invalid proxy URL, got nil")
	}
}

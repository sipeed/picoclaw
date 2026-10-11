package telegram

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

// A long poll stuck on a dead connection used to hang the poller forever because
// the Telegram HTTP client had no timeouts at all.
func TestNewTelegramHTTPClientBoundsHungRequests(t *testing.T) {
	client := newTelegramHTTPClient(nil)

	if client.Timeout != 0 {
		t.Fatalf("client.Timeout = %v, want 0: an overall timeout would abort long polls", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("DialContext is not configured: a dead host would hang forever")
	}
	if transport.TLSHandshakeTimeout != telegramDialTimeout {
		t.Fatalf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, telegramDialTimeout)
	}
	if transport.ResponseHeaderTimeout != telegramResponseHeaderTimeout {
		t.Fatalf(
			"ResponseHeaderTimeout = %v, want %v",
			transport.ResponseHeaderTimeout,
			telegramResponseHeaderTimeout,
		)
	}
	if transport.ResponseHeaderTimeout <= telegramLongPollTimeoutSeconds*time.Second {
		t.Fatalf(
			"ResponseHeaderTimeout = %v must exceed the %ds long-poll timeout",
			transport.ResponseHeaderTimeout,
			telegramLongPollTimeoutSeconds,
		)
	}
	if transport.IdleConnTimeout != telegramIdleConnTimeout {
		t.Fatalf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, telegramIdleConnTimeout)
	}
}

func TestNewTelegramHTTPClientAppliesConfiguredProxy(t *testing.T) {
	proxyURL, err := url.Parse("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatalf("url.Parse() error: %v", err)
	}

	client := newTelegramHTTPClient(http.ProxyURL(proxyURL))
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want the configured proxy")
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.telegram.org/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	got, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy lookup error: %v", err)
	}
	if got == nil || got.String() != proxyURL.String() {
		t.Fatalf("proxy = %v, want %v", got, proxyURL)
	}
}

func TestNewTelegramHTTPClientWithoutProxy(t *testing.T) {
	client := newTelegramHTTPClient(nil)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("transport.Proxy is set, want nil when no proxy is configured")
	}
}

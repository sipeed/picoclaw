package auth

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// serveCallback starts the callback handler on every loopback listener,
// mirroring what LoginBrowserWithOptions does.
func serveCallback(t *testing.T, state string) (int, chan callbackResult) {
	t.Helper()

	listeners, port, err := listenOAuthCallback(0)
	if err != nil {
		t.Fatalf("listenOAuthCallback: %v", err)
	}

	resultCh := make(chan callbackResult, 4)
	server := &http.Server{Handler: oauthCallbackHandler(state, resultCh)}
	for _, l := range listeners {
		go func(l net.Listener) { _ = server.Serve(l) }(l)
	}
	t.Cleanup(func() { _ = server.Close() })

	return port, resultCh
}

// The callback must be reachable however the browser resolves "localhost".
// An IPv4-only listener made the redirect fail on hosts that prefer ::1.
func TestOAuthCallbackListensOnBothLoopbackFamilies(t *testing.T) {
	listeners, port, err := listenOAuthCallback(0)
	if err != nil {
		t.Fatalf("listenOAuthCallback: %v", err)
	}
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()

	if len(listeners) < 2 {
		t.Skip("host has no IPv6 loopback; IPv4-only listener is expected here")
	}

	for _, host := range []string{"127.0.0.1", "[::1]"} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
		if err != nil {
			t.Fatalf("dial %s:%d: %v", host, port, err)
		}
		_ = conn.Close()
	}
}

// A stale tab replaying an old callback used to push an error onto the result
// channel, aborting the login in flight. It must be rejected locally instead.
func TestOAuthCallbackIgnoresStateMismatch(t *testing.T) {
	port, resultCh := serveCallback(t, "good-state")
	base := fmt.Sprintf("http://127.0.0.1:%d/auth/callback", port)

	resp, err := http.Get(base + "?state=stale-state&code=stale-code")
	if err != nil {
		t.Fatalf("stale callback request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("stale callback status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	select {
	case r := <-resultCh:
		t.Fatalf("stale callback aborted the login: %+v", r)
	case <-time.After(200 * time.Millisecond):
	}

	// The real callback still completes afterwards.
	resp, err = http.Get(base + "?state=good-state&code=real-code")
	if err != nil {
		t.Fatalf("real callback request: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case r := <-resultCh:
		if r.err != nil || r.code != "real-code" {
			t.Fatalf("got %+v, want code real-code", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("real callback never delivered")
	}
}

// Without a TTY the manual-paste reader hits EOF immediately. That must not
// cancel the browser flow before the user has opened the URL.
func TestLoginBrowserEmptyStdinDoesNotCancel(t *testing.T) {
	origInput := browserLoginInput
	origOpen := openBrowserFunc
	t.Cleanup(func() {
		browserLoginInput = origInput
		openBrowserFunc = origOpen
	})

	browserLoginInput = strings.NewReader("") // EOF, as under systemd
	openBrowserFunc = func(string) error { return nil }

	t.Setenv("PICOCLAW_OAUTH_TIMEOUT", "1s")

	done := make(chan error, 1)
	go func() {
		_, err := LoginBrowserWithOptions(OAuthProviderConfig{
			Issuer:   "https://example.invalid/o/oauth2/v2",
			ClientID: "test-client",
			Port:     0,
		}, LoginBrowserOptions{})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("got %v, want a timeout (empty stdin must not cancel)", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("login did not return")
	}
}

func TestBrowserLoginTimeoutOverride(t *testing.T) {
	if got := browserLoginTimeout(); got != 15*time.Minute {
		t.Fatalf("default timeout = %s, want 15m", got)
	}

	t.Setenv("PICOCLAW_OAUTH_TIMEOUT", "30m")
	if got := browserLoginTimeout(); got != 30*time.Minute {
		t.Fatalf("override timeout = %s, want 30m", got)
	}

	t.Setenv("PICOCLAW_OAUTH_TIMEOUT", "not-a-duration")
	if got := browserLoginTimeout(); got != 15*time.Minute {
		t.Fatalf("invalid override timeout = %s, want fallback 15m", got)
	}
}

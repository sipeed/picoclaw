package gateway

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/netbind"
)

func TestOpenGatewayListeners_HonorsIPv6OnlyHost(t *testing.T) {
	hasIPv4, hasIPv6 := netbind.DetectIPFamilies()
	if !hasIPv6 {
		t.Skip("IPv6 is unavailable in this environment")
	}

	_, result, err := openGatewayListeners("::", 0)
	if err != nil {
		t.Fatalf("openGatewayListeners() error = %v", err)
	}
	startGatewayTestHTTPServer(t, result.Listeners)
	port := mustGatewayAtoi(t, result.Port)

	requireGatewayHTTPReachable(t, "::1", port)
	if hasIPv4 {
		requireGatewayHTTPUnreachable(t, "127.0.0.1", port)
	}
}

func TestOpenGatewayListeners_SupportsExplicitMultiHost(t *testing.T) {
	hasIPv4, hasIPv6 := netbind.DetectIPFamilies()
	if !hasIPv4 || !hasIPv6 {
		t.Skip("dual-stack loopback is unavailable in this environment")
	}

	_, result, err := openGatewayListeners("127.0.0.1,::1", 0)
	if err != nil {
		t.Fatalf("openGatewayListeners() error = %v", err)
	}
	startGatewayTestHTTPServer(t, result.Listeners)
	port := mustGatewayAtoi(t, result.Port)

	requireGatewayHTTPReachable(t, "127.0.0.1", port)
	requireGatewayHTTPReachable(t, "::1", port)
}

func startGatewayTestHTTPServer(t *testing.T, listeners []net.Listener) {
	t.Helper()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}),
	}

	errCh := make(chan error, len(listeners))
	for _, listener := range listeners {
		ln := listener
		go func() {
			errCh <- server.Serve(ln)
		}()
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		for range listeners {
			err := <-errCh
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Fatalf("server.Serve() error = %v", err)
			}
		}
	})
}

func requireGatewayHTTPReachable(t *testing.T, host string, port int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := gatewayHTTPGet(host, port)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %s:%d to be reachable: %v", host, port, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requireGatewayHTTPUnreachable(t *testing.T, host string, port int) {
	t.Helper()
	if err := gatewayHTTPGet(host, port); err == nil {
		t.Fatalf("expected %s:%d to be unreachable", host, port)
	}
}

func gatewayHTTPGet(host string, port int) error {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	resp, err := client.Get("http://" + net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

func mustGatewayAtoi(t *testing.T, value string) int {
	t.Helper()
	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", value, err)
	}
	return n
}

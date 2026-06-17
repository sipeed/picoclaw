package utils

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCreateSafeHTTPClient_AllowsLoopbackProxy(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "http://example.com/proxied" {
			t.Fatalf("proxy received URL %q, want %q", r.URL.String(), "http://example.com/proxied")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer proxy.Close()

	client, err := CreateSafeHTTPClient(SafeHTTPClientOptions{
		ProxyURL: proxy.URL,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateSafeHTTPClient() error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/proxied", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	AllowConfiguredProxyFirstHop(req, client.Transport)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error: %v", err)
	}
	defer resp.Body.Close()
}

func TestCreateSafeHTTPClient_BlocksPrivateRedirect(t *testing.T) {
	allowPrivateHosts := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/secret", http.StatusFound)
	}))
	defer server.Close()

	client, err := CreateSafeHTTPClient(SafeHTTPClientOptions{
		Timeout: 5 * time.Second,
		AllowPrivateHosts: func() bool {
			return allowPrivateHosts
		},
		MaxRedirects: 5,
	})
	if err != nil {
		t.Fatalf("CreateSafeHTTPClient() error: %v", err)
	}

	allowPrivateHosts = false
	resp, err := client.Get(server.URL)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected redirect to private host to fail")
	}
	if !strings.Contains(err.Error(), "private or local network host") &&
		!strings.Contains(err.Error(), "blocked private or local target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSafeHTTPURL_BlocksLoopback(t *testing.T) {
	err := ValidateSafeHTTPURL("http://127.0.0.1:8080/file", nil, nil)
	if err == nil {
		t.Fatal("expected loopback URL to be blocked")
	}
	if !strings.Contains(err.Error(), "private or local network hosts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSafeHTTPURL_AllowsWhitelistedPrivateHost(t *testing.T) {
	whitelist, err := NewPrivateHostWhitelist([]string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("NewPrivateHostWhitelist() error: %v", err)
	}

	err = ValidateSafeHTTPURL("http://127.0.0.1:8080/file", whitelist, nil)
	if err != nil {
		t.Fatalf("expected whitelisted private host to pass, got %v", err)
	}
}

func TestNewSafeDialContext_BlocksPrivateDNSResolutionWithoutWhitelist(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on loopback: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}

	dialContext := NewSafeDialContext(&net.Dialer{Timeout: time.Second}, nil, nil)
	_, err = dialContext(context.Background(), "tcp", net.JoinHostPort("localhost", port))
	if err == nil {
		t.Fatal("expected localhost DNS resolution to be blocked without whitelist")
	}
	if !strings.Contains(err.Error(), "private") && !strings.Contains(err.Error(), "whitelisted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSafeDialContext_AllowsWhitelistedPrivateDNSResolution(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on loopback: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{}, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		conn.Close()
		accepted <- struct{}{}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}

	whitelist, err := NewPrivateHostWhitelist([]string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("failed to parse whitelist: %v", err)
	}

	dialContext := NewSafeDialContext(&net.Dialer{Timeout: time.Second}, whitelist, nil)
	conn, err := dialContext(context.Background(), "tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("expected localhost DNS resolution to succeed with whitelist, got %v", err)
	}
	conn.Close()

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("expected localhost listener to accept a connection")
	}
}

func TestIsPrivateOrRestrictedIP_Table(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"100.64.0.1", true},
		{"198.18.0.1", true},
		{"198.20.0.1", false},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"::1", true},
		{"::ffff:127.0.0.1", true},
		{"fc00::1", true},
		{"2002:7f00:0001::1", true},
		{"2002:0801:0101::1", false},
		{"2001:0000:4136:e378:8000:63bf:f5ff:fffe", true},
		{"2607:f8b0:4004:800::200e", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := IsPrivateOrRestrictedIP(ip)
			if got != tt.blocked {
				t.Fatalf("IsPrivateOrRestrictedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestDownloadFile_DefaultAllowsLoopbackURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("local download"))
	}))
	defer server.Close()

	path := DownloadFile(server.URL, "file.txt", DownloadOptions{
		LoggerPrefix: "test",
		Timeout:      5 * time.Second,
	})
	if path == "" {
		t.Fatal("expected default DownloadFile to allow loopback URL")
	}
	defer os.Remove(path)
}

func TestDownloadFile_BlockPrivateTargetsBlocksRedirectToLoopback(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secret"))
	}))
	defer target.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer proxy.Close()

	path := DownloadFile("http://example.com/file.txt", "file.txt", DownloadOptions{
		LoggerPrefix:        "test",
		Timeout:             5 * time.Second,
		ProxyURL:            proxy.URL,
		BlockPrivateTargets: true,
	})
	if path != "" {
		t.Fatalf("expected safe DownloadFile to block redirect to loopback, got %q", path)
	}
}

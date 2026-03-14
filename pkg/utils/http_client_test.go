package utils

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCreateHTTPClient_ProxyConfigured(t *testing.T) {
	client, err := CreateHTTPClient("http://127.0.0.1:7890", 12*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}
	if client.Timeout != 12*time.Second {
		t.Fatalf("client.Timeout = %v, want %v", client.Timeout, 12*time.Second)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want non-nil")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "http://127.0.0.1:7890")
	}
}

func TestCreateHTTPClient_InvalidProxy(t *testing.T) {
	_, err := CreateHTTPClient("://bad-proxy", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for invalid proxy URL, got nil")
	}
}

func TestCreateHTTPClient_Socks5ProxyConfigured(t *testing.T) {
	client, err := CreateHTTPClient("socks5://127.0.0.1:1080", 8*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "socks5://127.0.0.1:1080")
	}
}

func TestCreateHTTPClient_UnsupportedProxyScheme(t *testing.T) {
	_, err := CreateHTTPClient("ftp://127.0.0.1:21", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for unsupported scheme, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "unsupported proxy scheme")
	}
}

func TestCreateHTTPClient_ProxyFromEnvironmentWhenConfigEmpty(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("http_proxy", "http://127.0.0.1:8888")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("https_proxy", "http://127.0.0.1:8888")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	client, err := CreateHTTPClient("", 10*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want proxy function from environment")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	if _, err := tr.Proxy(req); err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
}

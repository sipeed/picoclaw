package api

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

func TestGatewayHostOverrideUsesExplicitRuntimePublic(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	if err := launcherconfig.Save(launcherPath, launcherconfig.Config{
		Port:   18800,
		Public: false,
	}); err != nil {
		t.Fatalf("launcherconfig.Save() error = %v", err)
	}

	h := NewHandler(configPath)
	h.SetServerOptions(18800, true, true, nil)

	if got := h.gatewayHostOverride(); got != "0.0.0.0" {
		t.Fatalf("gatewayHostOverride() = %q, want %q", got, "0.0.0.0")
	}
}

func TestBuildWsURLUsesRequestHostWhenNoForwardedHost(t *testing.T) {
	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/token", nil)
	req.Host = "192.168.1.9:18800"

	if got := buildWsURL(req); got != "ws://192.168.1.9:18800/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://192.168.1.9:18800/pico/ws")
	}
}

func TestGatewayProbeHostUsesLoopbackForWildcardBind(t *testing.T) {
	if got := gatewayProbeHost("0.0.0.0"); got != "127.0.0.1" {
		t.Fatalf("gatewayProbeHost() = %q, want %q", got, "127.0.0.1")
	}
}

func TestGatewayProxyURLUsesConfiguredHost(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "192.168.1.10"
	cfg.Gateway.Port = 18791
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if got := h.gatewayProxyURL().String(); got != "http://192.168.1.10:18791" {
		t.Fatalf("gatewayProxyURL() = %q, want %q", got, "http://192.168.1.10:18791")
	}
}

func TestGetGatewayHealthUsesConfiguredHost(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "192.168.1.10"
	cfg.Gateway.Port = 18791

	originalHealthGet := gatewayHealthGet
	t.Cleanup(func() {
		gatewayHealthGet = originalHealthGet
	})

	var requestedURL string
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		requestedURL = url
		return nil, errors.New("probe failed")
	}

	_, statusCode, err := h.getGatewayHealth(cfg, time.Second)
	_ = statusCode
	_ = err

	if requestedURL != "http://192.168.1.10:18791/health" {
		t.Fatalf("health url = %q, want %q", requestedURL, "http://192.168.1.10:18791/health")
	}
}

func TestGetGatewayHealthUsesProbeHostForPublicLauncher(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	h.SetServerOptions(18800, true, true, nil)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18791

	originalHealthGet := gatewayHealthGet
	t.Cleanup(func() {
		gatewayHealthGet = originalHealthGet
	})

	var requestedURL string
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		requestedURL = url
		return nil, errors.New("probe failed")
	}

	_, statusCode, err := h.getGatewayHealth(cfg, time.Second)
	_ = statusCode
	_ = err

	if requestedURL != "http://127.0.0.1:18791/health" {
		t.Fatalf("health url = %q, want %q", requestedURL, "http://127.0.0.1:18791/health")
	}
}

func TestBuildWsURLUsesWSSWhenForwardedProtoIsHTTPS(t *testing.T) {
	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/token", nil)
	req.Host = "chat.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := buildWsURL(req); got != "wss://chat.example.com/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://chat.example.com/pico/ws")
	}
}

func TestBuildWsURLUsesWSSWhenRequestIsTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://launcher.local/api/pico/token", nil)
	req.Host = "secure.example.com"
	req.TLS = &tls.ConnectionState{}

	if got := buildWsURL(req); got != "wss://secure.example.com/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://secure.example.com/pico/ws")
	}
}

func TestBuildWsURLPrefersForwardedHTTPOverTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://launcher.local/api/pico/token", nil)
	req.Host = "chat.example.com"
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("X-Forwarded-Proto", "http")

	if got := buildWsURL(req); got != "ws://chat.example.com/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://chat.example.com/pico/ws")
	}
}

func TestBuildWsURLUsesForwardedHostWithoutDefaultHTTPSPort(t *testing.T) {
	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/token", nil)
	req.Host = "127.0.0.1:18800"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "pico.rex.cc")
	req.Header.Set("X-Forwarded-Port", "443")

	if got := buildWsURL(req); got != "wss://pico.rex.cc/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://pico.rex.cc/pico/ws")
	}
}

func TestBuildWsURLUsesForwardedHostWithNonDefaultPort(t *testing.T) {
	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/token", nil)
	req.Host = "127.0.0.1:18800"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "pico.rex.cc")
	req.Header.Set("X-Forwarded-Port", "8443")

	if got := buildWsURL(req); got != "wss://pico.rex.cc:8443/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://pico.rex.cc:8443/pico/ws")
	}
}

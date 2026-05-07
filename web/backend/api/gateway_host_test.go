package api

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/netbind"
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

	if got := h.gatewayHostOverride(); got != "*" {
		t.Fatalf("gatewayHostOverride() = %q, want %q", got, "*")
	}
}

func TestBuildWsURLUsesRequestHostWhenLauncherPublicSaved(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	if err := launcherconfig.Save(launcherPath, launcherconfig.Config{
		Port:   18800,
		Public: true,
	}); err != nil {
		t.Fatalf("launcherconfig.Save() error = %v", err)
	}

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, false, nil)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18790

	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/info", nil)
	req.Host = "192.168.1.9:18800"

	if got := h.buildWsURL(req); got != "ws://192.168.1.9:18800/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://192.168.1.9:18800/pico/ws")
	}

	if got := h.buildPicoEventsURL(req); got != "http://192.168.1.9:18800/pico/events" {
		t.Fatalf("buildPicoEventsURL() = %q, want %q", got, "http://192.168.1.9:18800/pico/events")
	}
	if got := h.buildPicoSendURL(req); got != "http://192.168.1.9:18800/pico/send" {
		t.Fatalf("buildPicoSendURL() = %q, want %q", got, "http://192.168.1.9:18800/pico/send")
	}
}

func TestGatewayProbeHostUsesLoopbackForWildcardBind(t *testing.T) {
	want := "127.0.0.1"
	if got := gatewayProbeHost("0.0.0.0"); got != want {
		t.Fatalf("gatewayProbeHost() = %q, want %q", got, want)
	}
}

func TestGatewayProbeHostUsesPreferredLoopbackForEmptyBind(t *testing.T) {
	want := netbind.ResolveAdaptiveLoopbackHost()
	if got := gatewayProbeHost(""); got != want {
		t.Fatalf("gatewayProbeHost(empty) = %q, want %q", got, want)
	}
}

func TestGatewayProbeHostUsesPreferredLoopbackForLocalhostBind(t *testing.T) {
	want := netbind.ResolveAdaptiveLoopbackHost()
	if got := gatewayProbeHost("localhost"); got != want {
		t.Fatalf("gatewayProbeHost(localhost) = %q, want %q", got, want)
	}
}

func TestGatewayProbeHostUsesLoopbackForIPv6WildcardBind(t *testing.T) {
	want := "::1"
	if got := gatewayProbeHost("::"); got != want {
		t.Fatalf("gatewayProbeHost(::) = %q, want %q", got, want)
	}
}

func TestGatewayProbeHostUsesFirstConcreteHostForMultiHostBind(t *testing.T) {
	if got := gatewayProbeHost("127.0.0.1,::1"); got != "127.0.0.1" {
		t.Fatalf("gatewayProbeHost(multi) = %q, want %q", got, "127.0.0.1")
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

	want := "http://" + net.JoinHostPort(netbind.ResolveAdaptiveLoopbackHost(), "18791") + "/health"
	if requestedURL != want {
		t.Fatalf("health url = %q, want %q", requestedURL, want)
	}
}

func TestBuildWsURLUsesWSSWhenForwardedProtoIsHTTPS(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.Port = 18790

	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/info", nil)
	req.Host = "chat.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := h.buildWsURL(req); got != "wss://chat.example.com:443/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://chat.example.com:443/pico/ws")
	}
}

func TestBuildWsURLUsesWSSWhenRequestIsTLS(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.Port = 18790

	req := httptest.NewRequest("GET", "https://launcher.local/api/pico/info", nil)
	req.Host = "secure.example.com"
	req.TLS = &tls.ConnectionState{}

	if got := h.buildWsURL(req); got != "wss://secure.example.com:443/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://secure.example.com:443/pico/ws")
	}
}

func TestBuildPicoURLsPreferXForwardedHost(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	if err := launcherconfig.Save(launcherPath, launcherconfig.Config{
		Port:   18800,
		Public: true,
	}); err != nil {
		t.Fatalf("launcherconfig.Save() error = %v", err)
	}

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, false, nil)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.Port = 18790

	req := httptest.NewRequest("GET", "http://127.0.0.1:18800/api/pico/info", nil)
	req.Host = "127.0.0.1:18800"
	req.Header.Set("X-Forwarded-Host", "vscode-tunnel.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Port", "443")

	if got := h.buildPicoEventsURL(req); got != "https://vscode-tunnel.example.com:443/pico/events" {
		t.Fatalf("buildPicoEventsURL() = %q, want %q", got, "https://vscode-tunnel.example.com:443/pico/events")
	}
	if got := h.buildPicoSendURL(req); got != "https://vscode-tunnel.example.com:443/pico/send" {
		t.Fatalf("buildPicoSendURL() = %q, want %q", got, "https://vscode-tunnel.example.com:443/pico/send")
	}
	if got := h.buildWsURL(req); got != "wss://vscode-tunnel.example.com:443/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "wss://vscode-tunnel.example.com:443/pico/ws")
	}
}

func TestBuildWsURLPrefersForwardedHTTPOverTLS(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.Port = 18790

	req := httptest.NewRequest("GET", "https://launcher.local/api/pico/info", nil)
	req.Host = "chat.example.com"
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("X-Forwarded-Proto", "http")

	if got := h.buildWsURL(req); got != "ws://chat.example.com:80/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://chat.example.com:80/pico/ws")
	}
}

func TestBuildWsURLDoesNotTrustOriginWhenProxyOmitsForwardedProto(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/info", nil)
	req.Host = "fs-952210-xwj.picoclaw.lan.sipeed.com"
	req.Header.Set("Origin", "https://fs-952210-xwj.picoclaw.lan.sipeed.com")

	if got := h.buildWsURL(req); got != "ws://fs-952210-xwj.picoclaw.lan.sipeed.com:80/pico/ws" {
		t.Fatalf(
			"buildWsURL() = %q, want %q",
			got,
			"ws://fs-952210-xwj.picoclaw.lan.sipeed.com:80/pico/ws",
		)
	}
}

func TestBuildWsURLUsesRequestHostNotGatewayBindLoopback(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, false, nil)

	req := httptest.NewRequest("GET", "http://localhost:18800/api/pico/info", nil)
	req.Host = "localhost:18800"

	if got := h.buildWsURL(req); got != "ws://localhost:18800/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://localhost:18800/pico/ws")
	}
}

func TestGatewayHostOverrideWithExplicitHostAndAlignedGatewayHost(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	h.SetServerOptions(18800, false, false, nil)
	h.SetServerBindHost("0.0.0.0", true)

	if got := h.gatewayHostOverride(); got != "0.0.0.0" {
		t.Fatalf("gatewayHostOverride() = %q, want %q", got, "0.0.0.0")
	}
}

func TestGatewayHostOverrideWithExplicitHostAndLocalhostGatewayHost(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	h.SetServerOptions(18800, false, false, nil)
	h.SetServerBindHost("::", true)

	if got := h.gatewayHostOverride(); got != "::" {
		t.Fatalf("gatewayHostOverride() = %q, want %q", got, "::")
	}
}

func TestGatewayHostOverrideWithExplicitMultiHost(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	h.SetServerOptions(18800, false, false, nil)
	h.SetServerBindHost("127.0.0.1,::1", true)

	if got := h.gatewayHostOverride(); got != "127.0.0.1,::1" {
		t.Fatalf("gatewayHostOverride() = %q, want %q", got, "127.0.0.1,::1")
	}
}

func TestGatewayHostExplicitIgnoresPublicFlag(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	h.SetServerOptions(18800, true, true, nil)
	h.SetServerBindHost("127.0.0.1", true)

	if got := h.effectiveLauncherPublic(); got {
		t.Fatalf("effectiveLauncherPublic() = %t, want false when explicit host is set", got)
	}
}

package api

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

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

	req := httptest.NewRequest("GET", "http://launcher.local/api/pico/token", nil)
	req.Host = "192.168.1.9:18800"

	if got := h.buildWsURL(req, cfg); got != "ws://192.168.1.9:18790/pico/ws" {
		t.Fatalf("buildWsURL() = %q, want %q", got, "ws://192.168.1.9:18790/pico/ws")
	}
}

func TestGatewayProbeHostUsesLoopbackForWildcardBind(t *testing.T) {
	if got := gatewayProbeHost("0.0.0.0"); got != "127.0.0.1" {
		t.Fatalf("gatewayProbeHost() = %q, want %q", got, "127.0.0.1")
	}
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeGatewayHostTestConfig(t *testing.T, host string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	raw := fmt.Sprintf(`{"version":2,"gateway":{"host":%q,"port":18790}}`, host)
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile(configPath): %v", err)
	}
	return configPath
}

func TestLoadConfig_GatewayHostEnvTrimmed(t *testing.T) {
	configPath := writeGatewayHostTestConfig(t, "127.0.0.1")
	t.Setenv(EnvGatewayHost, "  ::1  ")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Gateway.Host != "::1" {
		t.Fatalf("cfg.Gateway.Host = %q, want %q", cfg.Gateway.Host, "::1")
	}
}

func TestLoadConfig_GatewayHostBlankEnvFallsBackToConfigHost(t *testing.T) {
	configPath := writeGatewayHostTestConfig(t, "  localhost  ")
	t.Setenv(EnvGatewayHost, "   ")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	want, err := normalizeGatewayHostInput("localhost")
	if err != nil {
		t.Fatalf("normalizeGatewayHostInput() error: %v", err)
	}
	if cfg.Gateway.Host != want {
		t.Fatalf("cfg.Gateway.Host = %q, want %q", cfg.Gateway.Host, want)
	}
}

func TestLoadConfig_GatewayHostBlankEnvAndConfigFallsBackToDefault(t *testing.T) {
	configPath := writeGatewayHostTestConfig(t, "   ")
	t.Setenv(EnvGatewayHost, "   ")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	defaultHost, err := normalizeGatewayHostInput(DefaultConfig().Gateway.Host)
	if err != nil {
		t.Fatalf("normalizeGatewayHostInput() error: %v", err)
	}
	if cfg.Gateway.Host != defaultHost {
		t.Fatalf("cfg.Gateway.Host = %q, want %q", cfg.Gateway.Host, defaultHost)
	}
}

func TestLoadConfig_GatewayHostEnvPreservesExplicitWildcardHost(t *testing.T) {
	configPath := writeGatewayHostTestConfig(t, "localhost")
	t.Setenv(EnvGatewayHost, "  0.0.0.0  ")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	want, err := normalizeGatewayHostInput("0.0.0.0")
	if err != nil {
		t.Fatalf("normalizeGatewayHostInput() error: %v", err)
	}
	if cfg.Gateway.Host != want {
		t.Fatalf("cfg.Gateway.Host = %q, want %q", cfg.Gateway.Host, want)
	}
}

func TestLoadConfig_GatewayHostEnvNormalizesMultiHostInput(t *testing.T) {
	configPath := writeGatewayHostTestConfig(t, "localhost")
	t.Setenv(EnvGatewayHost, " [::1] , 127.0.0.1 , ::1 ")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Gateway.Host != "::1,127.0.0.1" {
		t.Fatalf("cfg.Gateway.Host = %q, want %q", cfg.Gateway.Host, "::1,127.0.0.1")
	}
}

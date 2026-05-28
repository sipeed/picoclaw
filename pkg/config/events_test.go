package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultEventLoggingConfig(t *testing.T) {
	cfg := DefaultConfig()
	logCfg := EffectiveEventLoggingConfig(cfg)

	if !logCfg.Enabled {
		t.Fatal("default event logging should be enabled")
	}
	if !reflect.DeepEqual(logCfg.Include, []string{"agent.*"}) {
		t.Fatalf("default include = %#v, want agent.*", logCfg.Include)
	}
	if logCfg.MinSeverity != "info" {
		t.Fatalf("default min severity = %q, want info", logCfg.MinSeverity)
	}
	if logCfg.IncludePayload {
		t.Fatal("default event logging should not include raw payloads")
	}
}

func TestLoadConfigEventLoggingOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
		"version": 3,
		"events": {
			"logging": {
				"enabled": false,
				"include": ["gateway.*"],
				"exclude": ["gateway.ready"],
				"min_severity": "warn",
				"include_payload": true
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	logCfg := EffectiveEventLoggingConfig(cfg)

	if logCfg.Enabled {
		t.Fatal("loaded event logging enabled = true, want false")
	}
	if !reflect.DeepEqual(logCfg.Include, []string{"gateway.*"}) {
		t.Fatalf("loaded include = %#v, want gateway.*", logCfg.Include)
	}
	if !reflect.DeepEqual(logCfg.Exclude, []string{"gateway.ready"}) {
		t.Fatalf("loaded exclude = %#v, want gateway.ready", logCfg.Exclude)
	}
	if logCfg.MinSeverity != "warn" {
		t.Fatalf("loaded min severity = %q, want warn", logCfg.MinSeverity)
	}
	if !logCfg.IncludePayload {
		t.Fatal("loaded include_payload = false, want true")
	}
}

func TestLoadConfigEventLoggingEnvOverrides(t *testing.T) {
	t.Setenv("PICOCLAW_EVENTS_LOGGING_ENABLED", "false")
	t.Setenv("PICOCLAW_EVENTS_LOGGING_INCLUDE", "gateway.*,channel.lifecycle.*")
	t.Setenv("PICOCLAW_EVENTS_LOGGING_EXCLUDE", "gateway.ready")
	t.Setenv("PICOCLAW_EVENTS_LOGGING_MIN_SEVERITY", "error")
	t.Setenv("PICOCLAW_EVENTS_LOGGING_INCLUDE_PAYLOAD", "true")

	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"version": 3}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	logCfg := EffectiveEventLoggingConfig(cfg)

	if logCfg.Enabled {
		t.Fatal("env enabled override = true, want false")
	}
	if !reflect.DeepEqual(logCfg.Include, []string{"gateway.*", "channel.lifecycle.*"}) {
		t.Fatalf("env include = %#v, want gateway/channel lifecycle", logCfg.Include)
	}
	if !reflect.DeepEqual(logCfg.Exclude, []string{"gateway.ready"}) {
		t.Fatalf("env exclude = %#v, want gateway.ready", logCfg.Exclude)
	}
	if logCfg.MinSeverity != "error" {
		t.Fatalf("env min severity = %q, want error", logCfg.MinSeverity)
	}
	if !logCfg.IncludePayload {
		t.Fatal("env include_payload = false, want true")
	}
}

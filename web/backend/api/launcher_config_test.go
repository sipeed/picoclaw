package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

func TestGetLauncherConfigUsesRuntimeFallback(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	h.SetServerOptions(19999, true, false, []string{"192.168.1.0/24"})

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/launcher-config", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got launcherConfigPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Port != 19999 || !got.Public {
		t.Fatalf("response = %+v, want port=19999 public=true", got)
	}
	if len(got.AllowedCIDRs) != 1 || got.AllowedCIDRs[0] != "192.168.1.0/24" {
		t.Fatalf("response allowed_cidrs = %v, want [192.168.1.0/24]", got.AllowedCIDRs)
	}
}

func TestPutLauncherConfigPersists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	path := launcherconfig.PathForAppConfig(configPath)
	if err := os.WriteFile(
		path,
		[]byte(`{"port":18800,"public":false,"dashboard_password_hash":"saved-hash","launcher_token":"legacy-token"}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	h := NewHandler(configPath)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/system/launcher-config",
		strings.NewReader(
			`{"port":18080,"public":true,"allowed_cidrs":["192.168.1.0/24"]}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := launcherconfig.Load(path, launcherconfig.Default())
	if err != nil {
		t.Fatalf("launcherconfig.Load() error = %v", err)
	}
	if cfg.Port != 18080 || !cfg.Public {
		t.Fatalf("saved config = %+v, want port=18080 public=true", cfg)
	}
	if cfg.DashboardPasswordHash != "saved-hash" {
		t.Fatalf("saved dashboard_password_hash = %q, want saved-hash", cfg.DashboardPasswordHash)
	}
	if cfg.LegacyLauncherToken != "" {
		t.Fatalf("saved legacy launcher_token = %q, want empty", cfg.LegacyLauncherToken)
	}
	if len(cfg.AllowedCIDRs) != 1 || cfg.AllowedCIDRs[0] != "192.168.1.0/24" {
		t.Fatalf("saved config allowed_cidrs = %v, want [192.168.1.0/24]", cfg.AllowedCIDRs)
	}
}

func TestPutLauncherConfigRejectsInvalidPort(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/system/launcher-config",
		strings.NewReader(`{"port":70000,"public":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestPutLauncherConfigRejectsInvalidCIDR(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/system/launcher-config",
		strings.NewReader(`{"port":18080,"public":false,"allowed_cidrs":["bad-cidr"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

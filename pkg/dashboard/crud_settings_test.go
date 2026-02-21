package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestPasswordChange(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dashboard.Password = "oldpassword"
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := passwordChangeHandler(cfg, configPath)

	form := url.Values{}
	form.Set("current_password", "oldpassword")
	form.Set("new_password", "newpassword123")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/password",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.Dashboard.Password != "newpassword123" {
		t.Errorf("expected password to be updated, got %q", cfg.Dashboard.Password)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Password changed successfully") {
		t.Error("response should contain success message")
	}

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == cookieName {
			found = true
			if c.Value == "" {
				t.Error("session cookie should not be empty")
			}
			break
		}
	}
	if !found {
		t.Error("response should set a new session cookie")
	}
}

func TestPasswordChangeWrongCurrent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dashboard.Password = "oldpassword"
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := passwordChangeHandler(cfg, configPath)

	form := url.Values{}
	form.Set("current_password", "wrongpassword")
	form.Set("new_password", "newpassword123")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/password",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "incorrect") {
		t.Error("response should mention incorrect password")
	}

	if cfg.Dashboard.Password != "oldpassword" {
		t.Error("password should not have changed")
	}
}

func TestPasswordChangeTooShort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dashboard.Password = "oldpassword"
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := passwordChangeHandler(cfg, configPath)

	form := url.Values{}
	form.Set("current_password", "oldpassword")
	form.Set("new_password", "short")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/password",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "at least 8") {
		t.Error("response should mention minimum length")
	}

	if cfg.Dashboard.Password != "oldpassword" {
		t.Error("password should not have changed")
	}
}

func TestGatewayUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := gatewayUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("host", "127.0.0.1")
	form.Set("port", "9090")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/gateway",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got %q", cfg.Gateway.Host)
	}
	if cfg.Gateway.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Gateway.Port)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Gateway settings saved") {
		t.Error("response should contain success message")
	}
}

func TestGatewayUpdateBadPort(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := gatewayUpdateHandler(cfg, configPath)

	tests := []struct {
		name string
		port string
	}{
		{"zero", "0"},
		{"too_high", "99999"},
		{"negative", "-1"},
		{"not_a_number", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("host", "localhost")
			form.Set("port", tt.port)

			req := httptest.NewRequest(
				http.MethodPost,
				"/dashboard/crud/settings/gateway",
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for port=%s, got %d: %s", tt.port, w.Code, w.Body.String())
			}
		})
	}
}

func TestHeartbeatUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := heartbeatUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("enabled", "on")
	form.Set("interval", "15")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/heartbeat",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !cfg.Heartbeat.Enabled {
		t.Error("expected heartbeat enabled")
	}
	if cfg.Heartbeat.Interval != 15 {
		t.Errorf("expected interval 15, got %d", cfg.Heartbeat.Interval)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Heartbeat settings saved") {
		t.Error("response should contain success message")
	}
}

func TestDevicesUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := devicesUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("enabled", "on")
	form.Set("monitor_usb", "on")

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/crud/settings/devices",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !cfg.Devices.Enabled {
		t.Error("expected devices enabled")
	}
	if !cfg.Devices.MonitorUSB {
		t.Error("expected monitor_usb enabled")
	}

	body := w.Body.String()
	if !strings.Contains(body, "Devices settings saved") {
		t.Error("response should contain success message")
	}
}

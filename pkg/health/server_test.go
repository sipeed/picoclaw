package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer() *Server {
	s := &Server{
		ready:     false,
		checks:    make(map[string]Check),
		startTime: time.Now(),
		authToken: "test",
	}
	return s
}

func TestHealthHandler_ReturnsOK(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestReadyHandler_NotReady(t *testing.T) {
	s := newTestServer()
	// s.ready defaults to false
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	s.readyHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "not ready" {
		t.Errorf("status = %q, want %q", resp.Status, "not ready")
	}
}

func TestReadyHandler_Ready(t *testing.T) {
	s := newTestServer()
	s.SetReady(true)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	s.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ready" {
		t.Errorf("status = %q, want %q", resp.Status, "ready")
	}
}

func TestReadyHandler_FailedCheck(t *testing.T) {
	s := newTestServer()
	s.SetReady(true)

	// Register a failing check
	s.RegisterCheck("database", func() (bool, string) {
		return false, "connection refused"
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	s.readyHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready with failed check = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "not ready" {
		t.Errorf("status = %q, want %q", resp.Status, "not ready")
	}
	check, ok := resp.Checks["database"]
	if !ok {
		t.Fatal("missing database check in response")
	}
	if check.Status != "fail" {
		t.Errorf("check status = %q, want %q", check.Status, "fail")
	}
	if check.Message != "connection refused" {
		t.Errorf("check message = %q, want %q", check.Message, "connection refused")
	}
}

func TestReadyHandler_PassingCheck(t *testing.T) {
	s := newTestServer()
	s.SetReady(true)

	s.RegisterCheck("redis", func() (bool, string) {
		return true, "connected"
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	s.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready with passing check = %d, want %d", w.Code, http.StatusOK)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Checks["redis"].Status != "ok" {
		t.Errorf("redis check status = %q, want %q", resp.Checks["redis"].Status, "ok")
	}
}

func TestReloadHandler_MethodNotAllowed(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/reload", nil)
	w := httptest.NewRecorder()

	s.reloadHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("reload GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestReloadHandler_NoReloadFunc(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	req.Header.Set("Authorization", "Bearer test")
	w := httptest.NewRecorder()

	s.reloadHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("reload without func = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestReloadHandler_Success(t *testing.T) {
	s := newTestServer()
	called := false
	s.SetReloadFunc(func() error {
		called = true
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	req.Header.Set("Authorization", "Bearer test")
	w := httptest.NewRecorder()

	s.reloadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("reload status = %d, want %d", w.Code, http.StatusOK)
	}
	if !called {
		t.Error("reload function was not called")
	}
}

func TestReloadHandler_Error(t *testing.T) {
	s := newTestServer()
	s.SetReloadFunc(func() error {
		return errors.New("config parse error")
	})

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	req.Header.Set("Authorization", "Bearer test")
	w := httptest.NewRecorder()

	s.reloadHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("reload error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestSetReady_Toggle(t *testing.T) {
	s := newTestServer()

	s.SetReady(true)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	s.readyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("after SetReady(true): status = %d, want %d", w.Code, http.StatusOK)
	}

	s.SetReady(false)
	w = httptest.NewRecorder()
	s.readyHandler(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("after SetReady(false): status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestRegisterCheck_MultipleChecks(t *testing.T) {
	s := newTestServer()
	s.SetReady(true)

	s.RegisterCheck("db", func() (bool, string) {
		return true, "ok"
	})
	s.RegisterCheck("cache", func() (bool, string) {
		return true, "ok"
	})
	s.RegisterCheck("queue", func() (bool, string) {
		return false, "timeout"
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	s.readyHandler(w, req)

	// Should be not ready because queue check fails
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (queue check failed)", w.Code, http.StatusServiceUnavailable)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Checks) != 3 {
		t.Errorf("checks count = %d, want 3", len(resp.Checks))
	}
}

func TestRegisterOnMux(t *testing.T) {
	s := newTestServer()
	s.SetReady(true)

	mux := http.NewServeMux()
	s.RegisterOnMux(mux)

	// Test /health on custom mux
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/health on custom mux = %d, want %d", w.Code, http.StatusOK)
	}

	// Test /ready on custom mux
	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/ready on custom mux = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewServer(t *testing.T) {
	s := NewServer("127.0.0.1", 0, "")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.ready {
		t.Error("new server should not be ready by default")
	}
	if s.checks == nil {
		t.Error("checks map should be initialized")
	}
}

func TestNewServer_IPv6ListenAddrFormatting(t *testing.T) {
	s := NewServer("::", 18790, "")
	if s.server == nil {
		t.Fatal("server should be initialized")
	}
	if s.server.Addr != "[::]:18790" {
		t.Fatalf("server.Addr = %q, want %q", s.server.Addr, "[::]:18790")
	}
}

func TestStartContext_Cancellation(t *testing.T) {
	s := NewServer("127.0.0.1", 0, "")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.StartContext(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context should trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("StartContext returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("StartContext did not return after context cancellation")
	}
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		input bool
		want  string
	}{
		{true, "ok"},
		{false, "fail"},
	}
	for _, tt := range tests {
		got := statusString(tt.input)
		if got != tt.want {
			t.Errorf("statusString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

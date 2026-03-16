package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/requestlog"
)

func TestHandleGetRequestLogs(t *testing.T) {
	tmpDir := t.TempDir()
	storage := requestlog.NewStorage(tmpDir, 1)
	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	storage.Close()

	reader := requestlog.NewReader(tmpDir, 1)
	handler := &Handler{
		requestLogReader: reader,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/logs/requests?limit=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := resp["records"]; !ok {
		t.Error("expected 'records' field in response")
	}
}

func TestHandleGetRequestStats(t *testing.T) {
	tmpDir := t.TempDir()
	reader := requestlog.NewReader(tmpDir, 1)
	handler := &Handler{
		requestLogReader: reader,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/stats/requests", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := resp["total"]; !ok {
		t.Error("expected 'total' field in response")
	}
}

func TestHandleExportRequestLogs_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	storage := requestlog.NewStorage(tmpDir, 1)
	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	storage.Close()

	reader := requestlog.NewReader(tmpDir, 1)
	handler := &Handler{
		requestLogReader: reader,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/logs/requests/export?format=json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestHandleExportRequestLogs_CSV(t *testing.T) {
	tmpDir := t.TempDir()
	storage := requestlog.NewStorage(tmpDir, 1)
	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	storage.Close()

	reader := requestlog.NewReader(tmpDir, 1)
	handler := &Handler{
		requestLogReader: reader,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/logs/requests/export?format=csv", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("expected Content-Type 'text/csv', got %q", contentType)
	}
}

func TestHandleGetRequestLogConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logger := requestlog.NewLogger(requestlog.DefaultConfig(), nil, tmpDir)

	handler := &Handler{
		requestLogger: logger,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/config/requestlog", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var cfg requestlog.Config
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestHandlePutRequestLogConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logger := requestlog.NewLogger(requestlog.DefaultConfig(), nil, tmpDir)

	handler := &Handler{
		requestLogger: logger,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	newConfig := requestlog.Config{
		Enabled:         true,
		LogDir:          "logs/requests",
		MaxFileSizeMB:   50,
		MaxFiles:        50,
		RetentionDays:   7,
		ArchiveInterval: "12h",
		CompressArchive: false,
	}

	body, _ := json.Marshal(newConfig)
	req := httptest.NewRequest("PUT", "/api/config/requestlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleArchiveNow(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs", "requests")
	os.MkdirAll(logDir, 0o755)

	logger := requestlog.NewLogger(requestlog.DefaultConfig(), nil, tmpDir)

	handler := &Handler{
		requestLogger: logger,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("POST", "/api/logs/requests/archive-now", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleGetRequestLogs_NotAvailable(t *testing.T) {
	handler := &Handler{
		requestLogReader: nil,
	}

	mux := http.NewServeMux()
	handler.registerRequestLogRoutes(mux)

	req := httptest.NewRequest("GET", "/api/logs/requests", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

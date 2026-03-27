package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGetSystemVersion(t *testing.T) {
	originalVersion := config.Version
	originalGitCommit := config.GitCommit
	originalBuildTime := config.BuildTime
	originalGoVersion := config.GoVersion
	t.Cleanup(func() {
		config.Version = originalVersion
		config.GitCommit = originalGitCommit
		config.BuildTime = originalBuildTime
		config.GoVersion = originalGoVersion
	})

	config.Version = "v1.2.3"
	config.GitCommit = "deadbeef"
	config.BuildTime = "2026-03-27T12:34:56Z"
	config.GoVersion = "go1.24.1"

	h := NewHandler("")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/version", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got systemVersionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.Version != config.Version {
		t.Fatalf("version = %q, want %q", got.Version, config.Version)
	}
	if got.GitCommit != config.GitCommit {
		t.Fatalf("git_commit = %q, want %q", got.GitCommit, config.GitCommit)
	}
	if got.BuildTime != config.BuildTime {
		t.Fatalf("build_time = %q, want %q", got.BuildTime, config.BuildTime)
	}
	if got.GoVersion != config.GoVersion {
		t.Fatalf("go_version = %q, want %q", got.GoVersion, config.GoVersion)
	}
}

func TestGetSystemVersionUsesRuntimeGoVersionFallback(t *testing.T) {
	originalVersion := config.Version
	originalGitCommit := config.GitCommit
	originalBuildTime := config.BuildTime
	originalGoVersion := config.GoVersion
	t.Cleanup(func() {
		config.Version = originalVersion
		config.GitCommit = originalGitCommit
		config.BuildTime = originalBuildTime
		config.GoVersion = originalGoVersion
	})

	config.Version = "dev"
	config.GitCommit = ""
	config.BuildTime = ""
	config.GoVersion = ""

	h := NewHandler("")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/version", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got systemVersionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.GoVersion != runtime.Version() {
		t.Fatalf("go_version = %q, want runtime version %q", got.GoVersion, runtime.Version())
	}
}

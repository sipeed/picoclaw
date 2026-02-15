package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	publicPaths := []string{"/api/health"}

	tests := []struct {
		name       string
		apiKey     string
		authHeader string
		path       string
		wantStatus int
	}{
		{
			name:       "valid bearer token",
			apiKey:     "secret123",
			authHeader: "Bearer secret123",
			path:       "/api/chat",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid bearer token",
			apiKey:     "secret123",
			authHeader: "Bearer wrong",
			path:       "/api/chat",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing auth header",
			apiKey:     "secret123",
			authHeader: "",
			path:       "/api/chat",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "public path skips auth",
			apiKey:     "secret123",
			authHeader: "",
			path:       "/api/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "no api key configured (open mode)",
			apiKey:     "",
			authHeader: "",
			path:       "/api/chat",
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong auth format",
			apiKey:     "secret123",
			authHeader: "Basic dXNlcjpwYXNz",
			path:       "/api/chat",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := AuthMiddleware(tt.apiKey, publicPaths, inner)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	SetVersion("1.2.3")
	s := &APIServer{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("status = %q, want %q", health.Status, "ok")
	}
	if health.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", health.Version, "1.2.3")
	}
}

func TestSpecEndpoint(t *testing.T) {
	SetOpenAPISpec([]byte("openapi: '3.1.0'\ninfo:\n  title: test"))
	s := &APIServer{}
	req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	w := httptest.NewRecorder()
	s.handleSpec(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/x-yaml" {
		t.Errorf("content-type = %q, want %q", ct, "application/x-yaml")
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "openapi") {
		t.Error("response body should contain 'openapi'")
	}
}

func TestChatEndpointRequiresMessage(t *testing.T) {
	s := &APIServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	s.handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error != "message is required" {
		t.Errorf("error = %q, want %q", errResp.Error, "message is required")
	}
}

func TestChatEndpointRejectsGet(t *testing.T) {
	s := &APIServer{}
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()
	s.handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestChatEndpointInvalidJSON(t *testing.T) {
	s := &APIServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	s.handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

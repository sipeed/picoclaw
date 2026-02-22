// ABOUTME: Tests for Ollama health check and connectivity.
// ABOUTME: Uses httptest mock server to verify model detection behavior.
package memory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOllamaAvailable_ModelFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := ollamaTagsResponse{
			Models: []ollamaModel{
				{Name: "llama3.2:latest"},
				{Name: "nomic-embed-text:latest"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	err := CheckOllamaAvailable(t.Context(), server.URL, "nomic-embed-text")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCheckOllamaAvailable_ModelFoundExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaTagsResponse{
			Models: []ollamaModel{
				{Name: "nomic-embed-text"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	err := CheckOllamaAvailable(t.Context(), server.URL, "nomic-embed-text")
	if err != nil {
		t.Fatalf("expected no error for exact match, got: %v", err)
	}
}

func TestCheckOllamaAvailable_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaTagsResponse{
			Models: []ollamaModel{
				{Name: "llama3.2:latest"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	err := CheckOllamaAvailable(t.Context(), server.URL, "nomic-embed-text")
	if err == nil {
		t.Fatal("expected error when model not found")
	}
	if got := err.Error(); !contains(got, "not found") {
		t.Errorf("error = %q, want it to contain 'not found'", got)
	}
}

func TestCheckOllamaAvailable_ServerUnreachable(t *testing.T) {
	err := CheckOllamaAvailable(t.Context(), "http://127.0.0.1:1", "nomic-embed-text")
	if err == nil {
		t.Fatal("expected error when server unreachable")
	}
	if got := err.Error(); !contains(got, "unreachable") {
		t.Errorf("error = %q, want it to contain 'unreachable'", got)
	}
}

func TestCheckOllamaAvailable_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := CheckOllamaAvailable(t.Context(), server.URL, "nomic-embed-text")
	if err == nil {
		t.Fatal("expected error on server 500")
	}
	if got := err.Error(); !contains(got, "status 500") {
		t.Errorf("error = %q, want it to contain 'status 500'", got)
	}
}

func TestCheckOllamaAvailable_DefaultValues(t *testing.T) {
	// When called with empty strings, it should use defaults and likely fail
	// (no local Ollama in test environment). Just verify it doesn't panic.
	_ = CheckOllamaAvailable(t.Context(), "", "")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

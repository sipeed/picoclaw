package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestProbeLocalModelAvailability_OpenAICompatibleIncludesAPIKey(t *testing.T) {
	const apiKey = "test-api-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/v1/models")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"custom-model"}]}`))
	}))
	defer srv.Close()

	model := &config.ModelConfig{
		Model:   "openai/custom-model",
		APIBase: srv.URL + "/v1",
	}
	model.SetAPIKey(apiKey)

	if !probeLocalModelAvailability(model) {
		t.Fatal("probeLocalModelAvailability() = false, want true when api_key is configured")
	}
}

func TestRequiresRuntimeProbe_LMStudio(t *testing.T) {
	if !requiresRuntimeProbe(&config.ModelConfig{
		Model: "lmstudio/openai/gpt-oss-20b",
	}) {
		t.Fatal("requiresRuntimeProbe(lmstudio with default base) = false, want true")
	}

	if requiresRuntimeProbe(&config.ModelConfig{
		Model:   "lmstudio/openai/gpt-oss-20b",
		APIBase: "https://api.example.com/v1",
	}) {
		t.Fatal("requiresRuntimeProbe(lmstudio with remote base) = true, want false")
	}
}

func TestModelProbeAPIBase_LMStudioDefault(t *testing.T) {
	got := modelProbeAPIBase(&config.ModelConfig{Model: "lmstudio/openai/gpt-oss-20b"})
	if got != "http://localhost:1234/v1" {
		t.Fatalf("modelProbeAPIBase(lmstudio) = %q, want %q", got, "http://localhost:1234/v1")
	}
}

func TestProbeLocalModelAvailability_LMStudioUsesOpenAICompatibleProbe(t *testing.T) {
	originalProbe := probeOpenAICompatibleModelFunc
	defer func() { probeOpenAICompatibleModelFunc = originalProbe }()

	called := false
	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		called = true
		if apiBase != "http://localhost:1234/v1" {
			t.Fatalf("apiBase = %q, want %q", apiBase, "http://localhost:1234/v1")
		}
		if modelID != "openai/gpt-oss-20b" {
			t.Fatalf("modelID = %q, want %q", modelID, "openai/gpt-oss-20b")
		}
		if apiKey != "" {
			t.Fatalf("apiKey = %q, want empty", apiKey)
		}
		return true
	}

	model := &config.ModelConfig{Model: "lmstudio/openai/gpt-oss-20b"}
	if !probeLocalModelAvailability(model) {
		t.Fatal("probeLocalModelAvailability(lmstudio) = false, want true")
	}
	if !called {
		t.Fatal("probeOpenAICompatibleModelFunc was not called for lmstudio")
	}
}

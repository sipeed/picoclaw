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

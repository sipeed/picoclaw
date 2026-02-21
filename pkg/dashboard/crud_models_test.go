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

func TestModelCreateHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")
	initialLen := len(cfg.ModelList)

	handler := modelCreateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("model_name", "test-model")
	form.Set("model", "openai/gpt-4o")
	form.Set("api_base", "https://api.openai.com/v1")
	form.Set("api_key", "sk-test-key")
	form.Set("proxy", "")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(cfg.ModelList) != initialLen+1 {
		t.Fatalf("expected ModelList to grow by 1, got %d (was %d)", len(cfg.ModelList), initialLen)
	}

	added := cfg.ModelList[len(cfg.ModelList)-1]
	if added.ModelName != "test-model" {
		t.Errorf("expected model_name 'test-model', got %q", added.ModelName)
	}
	if added.Model != "openai/gpt-4o" {
		t.Errorf("expected model 'openai/gpt-4o', got %q", added.Model)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Model added") {
		t.Error("response should contain success message")
	}
	if w.Header().Get("HX-Trigger") == "" {
		t.Error("response should have HX-Trigger header")
	}
}

func TestModelCreateHandlerMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := modelCreateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("model", "openai/gpt-4o")
	// model_name is missing

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "required") {
		t.Error("response should mention required fields")
	}
}

func TestModelUpdateHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	// Ensure at least one model exists
	if len(cfg.ModelList) == 0 {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: "original",
			Model:     "openai/gpt-3.5",
		})
	}

	handler := modelUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("idx", "0")
	form.Set("model_name", "updated-name")
	form.Set("model", "openai/gpt-4o")
	form.Set("api_base", "https://new-base.com/v1")
	form.Set("api_key", "sk-new-key")
	form.Set("proxy", "http://proxy:8080")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.ModelList[0].ModelName != "updated-name" {
		t.Errorf("expected model_name 'updated-name', got %q", cfg.ModelList[0].ModelName)
	}
	if cfg.ModelList[0].Model != "openai/gpt-4o" {
		t.Errorf("expected model 'openai/gpt-4o', got %q", cfg.ModelList[0].Model)
	}
	if cfg.ModelList[0].Proxy != "http://proxy:8080" {
		t.Errorf("expected proxy 'http://proxy:8080', got %q", cfg.ModelList[0].Proxy)
	}
}

func TestModelUpdateHandlerBadIdx(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := modelUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("idx", "999")
	form.Set("model_name", "test")
	form.Set("model", "test")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid model index") {
		t.Error("response should mention invalid index")
	}
}

func TestModelDeleteHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	// Ensure at least one model exists
	if len(cfg.ModelList) == 0 {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: "to-delete",
			Model:     "openai/gpt-3.5",
		})
	}
	initialLen := len(cfg.ModelList)

	handler := modelDeleteHandler(cfg, configPath)

	form := url.Values{}
	form.Set("idx", "0")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(cfg.ModelList) != initialLen-1 {
		t.Fatalf("expected ModelList to shrink by 1, got %d (was %d)", len(cfg.ModelList), initialLen)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Model deleted") {
		t.Error("response should contain success message")
	}
}

func TestModelDeleteHandlerBadIdx(t *testing.T) {
	cfg := config.DefaultConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")

	handler := modelDeleteHandler(cfg, configPath)

	form := url.Values{}
	form.Set("idx", "-1")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/models/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid model index") {
		t.Error("response should mention invalid index")
	}
}

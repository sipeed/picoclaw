package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestHandleSetUILanguage(t *testing.T) {
	tools.SetPreferredWebSearchLanguage("")
	t.Cleanup(func() {
		tools.SetPreferredWebSearchLanguage("")
	})

	h := NewHandler("")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ui/language", strings.NewReader(`{"language":"zh"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := tools.GetPreferredWebSearchLanguage(); got != "zh" {
		t.Fatalf("preferred web search language = %q, want zh", got)
	}
}

func TestHandleSetUILanguage_RejectsInvalidJSON(t *testing.T) {
	h := NewHandler("")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ui/language", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRF_AllowsGetWithoutHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET without X-Requested-With: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCSRF_BlocksPostWithoutHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST without X-Requested-With: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCSRF_BlocksPutWithoutHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("PUT without X-Requested-With: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCSRF_BlocksDeleteWithoutHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/skills/test", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("DELETE without X-Requested-With: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCSRF_AllowsPostWithHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST with X-Requested-With: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCSRF_AllowsDeleteWithHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRF(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/skills/test", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE with X-Requested-With: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSecurityHeaders_SetsNosniff(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := SecurityHeaders(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

func TestSecurityHeaders_SetsDenyFrameOptions(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := SecurityHeaders(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want %q", got, "DENY")
	}
}

func TestSecurityHeaders_DoesNotOverrideExistingHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
	h := SecurityHeaders(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

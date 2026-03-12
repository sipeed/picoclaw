package main

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnknownAPIPathStays404(t *testing.T) {
	mux := http.NewServeMux()
	registerEmbedRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/not-found", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMissingAssetStays404(t *testing.T) {
	mux := http.NewServeMux()
	registerEmbedRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/assets/not-found.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestRegisterEmbedRoutesRegistersSVGContentType(t *testing.T) {
	if err := mime.AddExtensionType(".svg", "image/svg"); err != nil {
		t.Fatalf("AddExtensionType() seed error: %v", err)
	}

	mux := http.NewServeMux()
	registerEmbedRoutes(mux)

	if got := mime.TypeByExtension(".svg"); got != "image/svg+xml" {
		t.Fatalf("TypeByExtension(.svg) = %q, want %q", got, "image/svg+xml")
	}
}

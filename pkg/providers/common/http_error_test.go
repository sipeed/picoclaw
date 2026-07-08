package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleErrorResponse_ReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()

	err = HandleErrorResponse(resp, server.URL)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("HandleErrorResponse() error = %T, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d, want %d", httpErr.StatusCode, http.StatusUnauthorized)
	}
	if httpErr.BodyPreview != `{"error":"unauthorized"}` {
		t.Fatalf("BodyPreview = %q", httpErr.BodyPreview)
	}
	if !strings.Contains(err.Error(), "Status: 401") {
		t.Fatalf("Error() should preserve status text, got %q", err.Error())
	}
}

func TestWrapHTMLResponseError_ReturnsHTTPError(t *testing.T) {
	err := WrapHTMLResponseError(
		http.StatusBadGateway,
		[]byte("<html>bad</html>"),
		"text/html",
		"https://api.example.com",
	)

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("WrapHTMLResponseError() error = %T, want *HTTPError", err)
	}
	if !httpErr.IsHTML {
		t.Fatal("expected IsHTML")
	}
	if !strings.Contains(err.Error(), "HTML instead of JSON") {
		t.Fatalf("Error() should preserve HTML message, got %q", err.Error())
	}
}

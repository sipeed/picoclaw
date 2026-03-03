package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchAnthropicUsage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("Anthropic-Beta"); got != anthropicBetaHeader {
			t.Errorf("Anthropic-Beta = %q, want %q", got, anthropicBetaHeader)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"five_hour":{"utilization":0.42},"seven_day":{"utilization":0.85}}`))
	}))
	defer srv.Close()

	// Temporarily override the URL by using the test server
	origURL := anthropicUsageURL
	defer func() { setAnthropicUsageURL(origURL) }()
	setAnthropicUsageURL(srv.URL)

	usage, err := FetchAnthropicUsage("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.FiveHourUtilization != 0.42 {
		t.Errorf("FiveHourUtilization = %v, want 0.42", usage.FiveHourUtilization)
	}
	if usage.SevenDayUtilization != 0.85 {
		t.Errorf("SevenDayUtilization = %v, want 0.85", usage.SevenDayUtilization)
	}
}

func TestFetchAnthropicUsage_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	origURL := anthropicUsageURL
	defer func() { setAnthropicUsageURL(origURL) }()
	setAnthropicUsageURL(srv.URL)

	_, err := FetchAnthropicUsage("test-token")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient scope") {
		t.Errorf("expected 'insufficient scope' error, got %q", err.Error())
	}
}

func TestFetchAnthropicUsage_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer srv.Close()

	origURL := anthropicUsageURL
	defer func() { setAnthropicUsageURL(origURL) }()
	setAnthropicUsageURL(srv.URL)

	_, err := FetchAnthropicUsage("test-token")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error containing '500', got %q", err.Error())
	}
}

func TestFetchAnthropicUsage_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	origURL := anthropicUsageURL
	defer func() { setAnthropicUsageURL(origURL) }()
	setAnthropicUsageURL(srv.URL)

	_, err := FetchAnthropicUsage("test-token")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing usage response") {
		t.Errorf("expected 'parsing usage response' error, got %q", err.Error())
	}
}

package qq

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQQTokenSource_CachesValidToken(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","access_token":"test-token","expires_in":"3600"}`))
	}))
	defer srv.Close()

	source := &qqTokenSource{
		appID:     "appid",
		appSecret: "secret",
		tokenURL:  srv.URL,
		client: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
	}

	first, err := source.Token()
	if err != nil {
		t.Fatalf("Token() first call error = %v", err)
	}
	second, err := source.Token()
	if err != nil {
		t.Fatalf("Token() second call error = %v", err)
	}

	if first.AccessToken != "test-token" || second.AccessToken != "test-token" {
		t.Fatalf("unexpected access token values: first=%q second=%q", first.AccessToken, second.AccessToken)
	}
	if calls.Load() != 1 {
		t.Fatalf("HTTP call count = %d, want 1", calls.Load())
	}
}

func TestQQTokenSource_ParsesNumericExpiresIn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","access_token":"num-exp","expires_in":3600}`))
	}))
	defer srv.Close()

	source := &qqTokenSource{
		appID:     "appid",
		appSecret: "secret",
		tokenURL:  srv.URL,
		client: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
	}

	tokenValue, err := source.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if tokenValue.AccessToken != "num-exp" {
		t.Fatalf("AccessToken = %q, want num-exp", tokenValue.AccessToken)
	}
	if tokenValue.TokenType != "QQBot" {
		t.Fatalf("TokenType = %q, want QQBot", tokenValue.TokenType)
	}
}

func TestQQTokenSource_ReturnsErrorForNonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":400,"message":"invalid app","access_token":"","expires_in":"0"}`))
	}))
	defer srv.Close()

	source := &qqTokenSource{
		appID:     "appid",
		appSecret: "bad-secret",
		tokenURL:  srv.URL,
		client: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
	}

	tokenValue, err := source.Token()
	if err == nil {
		t.Fatalf("Token() expected error, got token=%+v", tokenValue)
	}
	if !strings.Contains(err.Error(), "400.invalid app") {
		t.Fatalf("Token() error = %q, want contains %q", err.Error(), "400.invalid app")
	}
}

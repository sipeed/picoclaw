package providers

import (
	"context"
	"errors"
	"testing"
)

// Error texts measured live in gateway.log 2026-10-25 19:57/19:58 UTC: the local Strata
// app proxy answers these while its model is stopped or still loading. Both used to kill
// the turn on the first candidate (400 -> FailoverFormat, 409 -> unclassified), so the
// configured fallback candidate was never tried.
const (
	proxyModelNotReady  = `Post "http://192.168.0.137:47832/v1/chat/completions": API error: 400 {"error":{"message":"This model is not ready.","type":"invalid_request_error","param":"model","code":null}}`
	proxyModelNotLoaded = `POST request failed: Post "http://192.168.0.137:47832/v1/chat/completions": API error: 409 {"error":"That model ID is not loaded. Choose one from /v1/models."}`
	// Measured by probing the same proxy with a model ID it has not loaded.
	proxyChooseLoaded = `API error: 400 {"error":{"message":"Choose a loaded model ID.","type":"invalid_request_error","param":"model","code":null}}`
)

func TestClassifyError_ModelUnavailableIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{"proxy model not ready (400)", proxyModelNotReady},
		{"proxy model not loaded (409)", proxyModelNotLoaded},
		{"proxy unknown model id (400)", proxyChooseLoaded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(errors.New(tt.msg), "openai", "iq3_s")
			if result == nil {
				t.Fatalf("expected classification for %q", tt.msg)
			}
			if result.Reason != FailoverModelUnavailable {
				t.Errorf("reason = %q, want %q", result.Reason, FailoverModelUnavailable)
			}
			if !result.IsRetriable() {
				t.Errorf("model_unavailable must be retriable so the fallback chain continues")
			}
		})
	}
}

// A genuine 400 request-format error must stay non-retriable: the availability patterns
// must not mask real request bugs behind a retry/fallback loop.
func TestClassifyError_RealFormatErrorStaysNonRetriable(t *testing.T) {
	msg := `API request failed:
  Status: 400
  Body:   {"error":{"message":"messages.1.content.1.tool_use.id: Field required","type":"invalid_request_error"}}`

	result := ClassifyError(errors.New(msg), "anthropic", "claude-haiku-5-5")
	if result == nil {
		t.Fatal("expected classification")
	}
	if result.Reason != FailoverFormat {
		t.Errorf("reason = %q, want %q", result.Reason, FailoverFormat)
	}
	if result.IsRetriable() {
		t.Error("format errors must stay non-retriable")
	}
}

func TestFallbackChain_SwitchesOnModelUnavailable(t *testing.T) {
	ct := NewCooldownTracker()
	fc := NewFallbackChain(ct, nil)

	candidates := []FallbackCandidate{
		makeCandidate("openai", "iq3_s"),
		makeCandidate("anthropic", "claude-haiku-5-5"),
	}

	run := func(_ context.Context, provider, _ string) (*LLMResponse, error) {
		if provider == "openai" {
			return nil, errors.New(proxyModelNotReady)
		}
		return &LLMResponse{Content: "ja", FinishReason: "stop"}, nil
	}

	result, err := fc.Execute(context.Background(), candidates, run)
	if err != nil {
		t.Fatalf("fallback chain must recover on the second candidate: %v", err)
	}
	if result.Provider != "anthropic" || result.Model != "claude-haiku-5-5" {
		t.Errorf("served by %s/%s, want anthropic/claude-haiku-5-5", result.Provider, result.Model)
	}
	// Only failed attempts are recorded; the successful fallback candidate is not.
	if len(result.Attempts) != 1 {
		t.Fatalf("attempts = %d, want 1 failed primary attempt", len(result.Attempts))
	}
	if got := result.Attempts[0].Reason; got != FailoverModelUnavailable {
		t.Errorf("primary attempt reason = %q, want %q", got, FailoverModelUnavailable)
	}
	// The failing primary now sits in a standard cooldown (60s for the first failure),
	// so the next turn goes straight to the fallback while the model is still loading.
	if ct.IsAvailable(candidates[0].StableKey()) {
		t.Error("primary candidate should be in cooldown after model_unavailable")
	}
}

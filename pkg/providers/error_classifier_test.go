package providers

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestClassifyError_Nil(t *testing.T) {
	result := ClassifyError(nil, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for nil error, got %+v", result)
	}
}

func TestClassifyError_ContextCanceled(t *testing.T) {
	result := ClassifyError(context.Canceled, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for context.Canceled (user abort), got %+v", result)
	}
}

func TestClassifyError_ContextDeadlineExceeded(t *testing.T) {
	result := ClassifyError(context.DeadlineExceeded, "openai", "gpt-4")
	if result == nil {
		t.Fatal("expected non-nil for deadline exceeded")
	}
	if result.Reason != FailoverTimeout {
		t.Errorf("reason = %q, want timeout", result.Reason)
	}
}

func TestClassifyError_StatusCodes(t *testing.T) {
	tests := []struct {
		status int
		reason FailoverReason
	}{
		{401, FailoverAuth},
		{403, FailoverAuth},
		{402, FailoverBilling},
		{408, FailoverTimeout},
		{429, FailoverRateLimit},
		{400, FailoverModelInvalid},
		{500, FailoverTimeout},
		{502, FailoverTimeout},
		{503, FailoverTimeout},
		{521, FailoverTimeout},
		{522, FailoverTimeout},
		{523, FailoverTimeout},
		{524, FailoverTimeout},
		{529, FailoverTimeout},
	}

	for _, tt := range tests {
		err := fmt.Errorf("API error: status: %d something went wrong", tt.status)
		result := ClassifyError(err, "test", "model")
		if result == nil {
			t.Errorf("status %d: expected non-nil", tt.status)
			continue
		}
		if result.Reason != tt.reason {
			t.Errorf("status %d: reason = %q, want %q", tt.status, result.Reason, tt.reason)
		}
	}
}

func TestClassifyError_RateLimitPatterns(t *testing.T) {
	patterns := []string{
		"rate limit exceeded",
		"rate_limit reached",
		"too many requests",
		"exceeded your current quota",
		"resource has been exhausted",
		"resource_exhausted",
		"quota exceeded",
		"usage limit reached",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverRateLimit {
			t.Errorf("pattern %q: reason = %q, want rate_limit", msg, result.Reason)
		}
	}
}

func TestClassifyError_OverloadedPatterns(t *testing.T) {
	patterns := []string{
		"overloaded_error",
		`{"type": "overloaded_error"}`,
		"server is overloaded",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "anthropic", "claude")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		// Overloaded is treated as rate_limit
		if result.Reason != FailoverRateLimit {
			t.Errorf("pattern %q: reason = %q, want rate_limit", msg, result.Reason)
		}
	}
}

func TestClassifyError_BillingPatterns(t *testing.T) {
	patterns := []string{
		"payment required",
		"insufficient credits",
		"credit balance too low",
		"plans & billing page",
		"insufficient balance",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverBilling {
			t.Errorf("pattern %q: reason = %q, want billing", msg, result.Reason)
		}
	}
}

func TestClassifyError_TimeoutPatterns(t *testing.T) {
	patterns := []string{
		"request timeout",
		"connection timed out",
		"deadline exceeded",
		"context deadline exceeded",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverTimeout {
			t.Errorf("pattern %q: reason = %q, want timeout", msg, result.Reason)
		}
	}
}

func TestClassifyError_AuthPatterns(t *testing.T) {
	patterns := []string{
		"invalid api key",
		"invalid_api_key",
		"incorrect api key",
		"invalid token",
		"authentication failed",
		"re-authenticate",
		"oauth token refresh failed",
		"unauthorized access",
		"forbidden",
		"access denied",
		"expired",
		"token has expired",
		"no credentials found",
		"no api key found",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverAuth {
			t.Errorf("pattern %q: reason = %q, want auth", msg, result.Reason)
		}
	}
}

func TestClassifyError_FormatPatterns(t *testing.T) {
	patterns := []string{
		"string should match pattern",
		"tool_use.id is required",
		"invalid tool_use_id",
		"messages.1.content.1.tool_use.id must be valid",
		"invalid request format",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "anthropic", "claude")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverFormat {
			t.Errorf("pattern %q: reason = %q, want format", msg, result.Reason)
		}
	}
}

func TestClassifyError_ModelInvalidPatterns(t *testing.T) {
	patterns := []string{
		"nemotron-3-nano-30b-a3b:free is not a valid model ID",
		"model not found",
		"model_not_found: the requested model does not exist",
		"model not available in this region",
		"the model does not exist or you do not have access",
		"no such model: gpt-5-turbo",
		"invalid model specified",
		"model llama-3-8b is not supported",
		"model gpt-4o-mini is unavailable",
		"model codellama is deprecated",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "nvidia", "test-model")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverModelInvalid {
			t.Errorf("pattern %q: reason = %q, want model_invalid", msg, result.Reason)
		}
		if !result.IsRetriable() {
			t.Errorf("pattern %q: should be retriable to allow fallback", msg)
		}
		if !result.IsModelInvalid() {
			t.Errorf("pattern %q: should be classified as model invalid", msg)
		}
	}
}

func TestClassifyError_ModelInvalid_OverridesStatus400(t *testing.T) {
	// This is the exact production error: status 400 + "not a valid model".
	// Before the fix, status 400 was classified as FailoverFormat (non-retriable),
	// which prevented fallback to other models.
	err := fmt.Errorf("API request failed:\n  Status: 400\n  Body:   {\"error\":{\"message\":\"nemotron-3-nano-30b-a3b:free is not a valid model ID\",\"code\":400}}")
	result := ClassifyError(err, "nvidia", "nemotron-3-nano-30b-a3b:free")
	if result == nil {
		t.Fatal("expected non-nil for model-invalid 400 error")
	}
	if result.Reason != FailoverModelInvalid {
		t.Errorf("reason = %q, want model_invalid (should override status 400)", result.Reason)
	}
	if !result.IsRetriable() {
		t.Error("model-invalid error should be retriable to allow fallback to next model")
	}
	if !result.IsModelInvalid() {
		t.Error("should be classified as model invalid")
	}
}

func TestClassifyError_ImageDimensionError(t *testing.T) {
	err := errors.New("image dimensions exceed max allowed 2048x2048")
	result := ClassifyError(err, "openai", "gpt-4o")
	if result == nil {
		t.Fatal("expected non-nil for image dimension error")
	}
	if result.Reason != FailoverFormat {
		t.Errorf("reason = %q, want format", result.Reason)
	}
	if result.IsRetriable() {
		t.Error("image dimension error should not be retriable")
	}
}

func TestClassifyError_ImageSizeError(t *testing.T) {
	err := errors.New("image exceeds 20 mb limit")
	result := ClassifyError(err, "openai", "gpt-4o")
	if result == nil {
		t.Fatal("expected non-nil for image size error")
	}
	if result.Reason != FailoverFormat {
		t.Errorf("reason = %q, want format", result.Reason)
	}
}

func TestClassifyError_UnknownError(t *testing.T) {
	err := errors.New("some completely random error")
	result := ClassifyError(err, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for unknown error, got %+v", result)
	}
}

func TestClassifyError_ProviderModelPropagation(t *testing.T) {
	err := errors.New("rate limit exceeded")
	result := ClassifyError(err, "my-provider", "my-model")
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if result.Provider != "my-provider" {
		t.Errorf("provider = %q, want my-provider", result.Provider)
	}
	if result.Model != "my-model" {
		t.Errorf("model = %q, want my-model", result.Model)
	}
}

func TestFailoverError_IsRetriable(t *testing.T) {
	tests := []struct {
		reason    FailoverReason
		retriable bool
	}{
		{FailoverAuth, true},
		{FailoverRateLimit, true},
		{FailoverBilling, true},
		{FailoverTimeout, true},
		{FailoverOverloaded, true},
		{FailoverModelInvalid, true},
		{FailoverFormat, false},
		{FailoverUnknown, true},
	}

	for _, tt := range tests {
		fe := &FailoverError{Reason: tt.reason}
		if fe.IsRetriable() != tt.retriable {
			t.Errorf("IsRetriable(%q) = %v, want %v", tt.reason, fe.IsRetriable(), tt.retriable)
		}
	}
}

func TestFailoverError_ErrorString(t *testing.T) {
	fe := &FailoverError{
		Reason:   FailoverRateLimit,
		Provider: "openai",
		Model:    "gpt-4",
		Status:   429,
		Wrapped:  errors.New("too many requests"),
	}
	s := fe.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
}

func TestFailoverError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	fe := &FailoverError{Reason: FailoverTimeout, Wrapped: inner}
	if fe.Unwrap() != inner {
		t.Error("Unwrap should return wrapped error")
	}
}

func TestExtractHTTPStatus(t *testing.T) {
	tests := []struct {
		msg  string
		want int
	}{
		{"status: 429 rate limited", 429},
		{"status 401 unauthorized", 401},
		{"HTTP/1.1 502 Bad Gateway", 502},
		{"no status code here", 0},
		{"random number 12345", 0},
	}

	for _, tt := range tests {
		got := extractHTTPStatus(tt.msg)
		if got != tt.want {
			t.Errorf("extractHTTPStatus(%q) = %d, want %d", tt.msg, got, tt.want)
		}
	}
}

func TestIsImageDimensionError(t *testing.T) {
	if !IsImageDimensionError("image dimensions exceed max 4096x4096") {
		t.Error("should match image dimensions exceed max")
	}
	if IsImageDimensionError("normal error message") {
		t.Error("should not match normal error")
	}
}

func TestIsImageSizeError(t *testing.T) {
	if !IsImageSizeError("image exceeds 20 mb") {
		t.Error("should match image exceeds mb")
	}
	if IsImageSizeError("normal error message") {
		t.Error("should not match normal error")
	}
}

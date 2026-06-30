package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

func TestFormatProcessingError_InvalidAPIKey(t *testing.T) {
	err := errors.New(
		`LLM call failed after retries: API request failed: Status: 401 Body: {"error":{"message":"Incorrect API key provided"}}`,
	)

	got := formatProcessingError(err)
	if !strings.Contains(got, "API key appears to be invalid") {
		t.Fatalf("formatted error missing friendly API key hint: %q", got)
	}
	if !strings.Contains(got, "Original error:") {
		t.Fatalf("formatted error missing original error label: %q", got)
	}
	if !strings.Contains(got, err.Error()) {
		t.Fatalf("formatted error missing original error: %q", got)
	}
}

func TestFormatProcessingError_GenericAuthHTTPError(t *testing.T) {
	err := &common.HTTPError{
		StatusCode:  401,
		BodyPreview: `{"error":"unauthorized"}`,
		ContentType: "application/json",
		APIBase:     "https://api.example.com",
	}

	got := formatProcessingError(err)
	if !strings.Contains(got, "check the API key, token, OAuth login, or provider permissions") {
		t.Fatalf("formatted error missing generic auth hint: %q", got)
	}
	if !strings.Contains(got, "Original error:") {
		t.Fatalf("formatted error missing original error: %q", got)
	}
}

func TestFormatProcessingError_NonAuth(t *testing.T) {
	err := errors.New("connection reset by peer")
	got := formatProcessingError(err)
	want := "Error processing message: connection reset by peer"
	if got != want {
		t.Fatalf("formatted error = %q, want %q", got, want)
	}
}

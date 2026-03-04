package channels

import (
	"testing"
)

func TestWithVersionPrefix(t *testing.T) {
	tests := []struct {
		version  string
		endpoint string
		expected string
		name     string
	}{
		{"v1", "/webhook/telegram", "/v1/webhook/telegram", "normal path"},
		{"v2", "/health", "/v2/health", "different version"},
		{"v1", "", "/v1", "empty endpoint"},
		{"", "/webhook/telegram", "/webhook/telegram", "empty version"},
		{"v1", "webhook/telegram", "/v1/webhook/telegram", "endpoint without slash"},
		{"", "", "/", "both empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithVersionPrefix(tt.version, tt.endpoint)
			if result != tt.expected {
				t.Errorf("WithVersionPrefix(%q, %q) = %q, want %q", tt.version, tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestAPIVersionNegotiator(t *testing.T) {
	versions := []string{"v1", "v2", "v3"}
	negotiator := NewAPIVersionNegotiator(versions, "v1")

	if !negotiator.isValidVersion("v1") {
		t.Error("Expected v1 to be valid version")
	}
	
	if negotiator.isValidVersion("v999") {
		t.Error("Expected v999 to be invalid version")
	}

	// Test default version
	if negotiator.DetermineVersion(nil) != "v1" {
		t.Error("Expected default version to be v1")
	}
}

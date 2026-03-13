// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"testing"
)

func TestNewKimiCodeProvider(t *testing.T) {
	provider := NewKimiCodeProvider("test-api-key", "", "")
	if provider == nil {
		t.Fatal("NewKimiCodeProvider() returned nil")
	}

	// Check default model
	defaultModel := provider.GetDefaultModel()
	if defaultModel != "kimi-for-coding" {
		t.Errorf("GetDefaultModel() = %q, want %q", defaultModel, "kimi-for-coding")
	}
}

func TestNewKimiCodeProviderWithCustomBaseURL(t *testing.T) {
	customBase := "https://custom.kimi.com/coding/v1"
	provider := NewKimiCodeProvider("test-api-key", customBase, "")
	if provider == nil {
		t.Fatal("NewKimiCodeProvider() returned nil")
	}
}

func TestNewKimiCodeProviderWithTimeout(t *testing.T) {
	provider := NewKimiCodeProviderWithTimeout("test-api-key", "", "", 60)
	if provider == nil {
		t.Fatal("NewKimiCodeProviderWithTimeout() returned nil")
	}
}

func TestKimiCodeDefaultBaseURL(t *testing.T) {
	expected := "https://api.kimi.com/coding/v1"
	if KimiCodeDefaultBaseURL != expected {
		t.Errorf("KimiCodeDefaultBaseURL = %q, want %q", KimiCodeDefaultBaseURL, expected)
	}
}

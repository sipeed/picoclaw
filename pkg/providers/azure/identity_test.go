//go:build azidentity

package azure

import (
	"testing"
)

func TestNewProviderWithIdentity_Construction(t *testing.T) {
	// DefaultAzureCredential construction itself does not require any env vars;
	// failures surface only on the first GetToken call. Verify we get a
	// non-nil provider back with a token source wired in.
	p, err := NewProviderWithIdentity("https://example.openai.azure.com", "", "ua-test")
	if err != nil {
		t.Fatalf("NewProviderWithIdentity() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProviderWithIdentity() returned nil provider")
	}
	if p.tokenSource == nil {
		t.Fatal("provider.tokenSource should be set")
	}
	if p.apiKey != "" {
		t.Errorf("provider.apiKey = %q, want empty", p.apiKey)
	}
}

func TestNewProviderWithIdentityAndTimeout_Construction(t *testing.T) {
	p, err := NewProviderWithIdentityAndTimeout("https://example.openai.azure.com", "", "ua-test", 30)
	if err != nil {
		t.Fatalf("NewProviderWithIdentityAndTimeout() error = %v", err)
	}
	if p == nil {
		t.Fatal("returned nil provider")
	}
	if p.httpClient.Timeout.Seconds() != 30 {
		t.Errorf("timeout = %v, want 30s", p.httpClient.Timeout)
	}
}

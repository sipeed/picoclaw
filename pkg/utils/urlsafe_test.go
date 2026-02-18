package utils

import (
	"testing"
)

func TestValidateURL_AllowsPublicURLs(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"https://api.github.com/repos/test/test",
		"http://example.org/path?q=1",
	}

	for _, u := range validURLs {
		if err := ValidateURL(u); err != nil {
			t.Errorf("Expected URL %q to be allowed, got error: %v", u, err)
		}
	}
}

func TestValidateURL_BlocksPrivateIPs(t *testing.T) {
	blockedURLs := []string{
		"http://127.0.0.1",
		"http://127.0.0.1:8080/admin",
		"http://localhost",
		"http://localhost:3000",
		"http://10.0.0.1",
		"http://10.0.0.1:9090/secret",
		"http://172.16.0.1",
		"http://192.168.1.1",
		"http://169.254.169.254/latest/meta-data/",
		"http://0.0.0.0",
		"http://[::1]",
	}

	for _, u := range blockedURLs {
		if err := ValidateURL(u); err == nil {
			t.Errorf("Expected URL %q to be blocked, but it was allowed", u)
		}
	}
}

func TestValidateURL_BlocksInvalidSchemes(t *testing.T) {
	blockedURLs := []string{
		"ftp://example.com",
		"file:///etc/passwd",
		"gopher://example.com",
	}

	for _, u := range blockedURLs {
		if err := ValidateURL(u); err == nil {
			t.Errorf("Expected URL %q with non-http scheme to be blocked", u)
		}
	}
}

func TestValidateURL_BlocksMissingHost(t *testing.T) {
	if err := ValidateURL("http://"); err == nil {
		t.Error("Expected URL with missing host to be blocked")
	}
}

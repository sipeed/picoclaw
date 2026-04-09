//go:build cdp

package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestFindChromePath(t *testing.T) {
	// This test verifies the Chrome detection logic.
	// It may or may not find Chrome depending on the CI/dev environment.
	path, err := FindChromePath()
	if err != nil {
		t.Logf("Chrome not found (expected in some environments): %v", err)
		return
	}
	if path == "" {
		t.Error("FindChromePath returned empty path without error")
	}
	t.Logf("Chrome found at: %s", path)
}

func TestFindChromePath_EnvOverride(t *testing.T) {
	t.Setenv("CHROME_PATH", "/nonexistent/chrome")
	_, err := FindChromePath()
	// Should not use the invalid env path; should fall back or error
	if err == nil {
		// Chrome was found via some other path; that's fine
		return
	}
	t.Logf("Expected error with invalid CHROME_PATH: %v", err)
}

func TestValidateBrowserURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
		desc    string
	}{
		{"https://example.com", false, "valid HTTPS"},
		{"http://example.com/path?q=test", false, "valid HTTP with path"},
		{"file:///etc/passwd", true, "file protocol blocked"},
		{"ftp://example.com", true, "ftp protocol blocked"},
		{"javascript:alert(1)", true, "javascript protocol blocked"},
		{"http://localhost:8080", true, "localhost blocked"},
		{"http://127.0.0.1:9222", true, "loopback blocked"},
		{"http://169.254.169.254/metadata", true, "metadata endpoint blocked"},
		{"http://0.0.0.0", true, "0.0.0.0 blocked"},
		{"http://10.0.0.1", true, "private 10.x blocked"},
		{"http://192.168.1.1", true, "private 192.168.x blocked"},
		{"http://172.16.0.1", true, "private 172.16.x blocked"},
		{"http://[::1]", true, "IPv6 loopback blocked"},
		{"not-a-url", true, "invalid URL"},
		{"", true, "empty URL"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBrowserURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBrowserURL(%q) error = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestGetIntArg(t *testing.T) {
	tests := []struct {
		args map[string]any
		key  string
		want int
		ok   bool
	}{
		{map[string]any{"index": float64(5)}, "index", 5, true},
		{map[string]any{"index": 3}, "index", 3, true},
		{map[string]any{"index": int64(7)}, "index", 7, true},
		{map[string]any{"other": float64(1)}, "index", 0, false},
		{map[string]any{}, "index", 0, false},
		{map[string]any{"index": "not a number"}, "index", 0, false},
	}

	for _, tt := range tests {
		got, ok := getIntArg(tt.args, tt.key)
		if got != tt.want || ok != tt.ok {
			t.Errorf("getIntArg(%v, %q) = (%d, %v), want (%d, %v)",
				tt.args, tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

func TestBrowserToolName(t *testing.T) {
	cfg := config.BrowserToolConfig{
		Stealth: true,
	}
	// NewBrowserTool will fail without Chrome, but we can test
	// the stub or interface conformance
	tool, err := NewBrowserTool(cfg)
	if err != nil {
		// Expected when Chrome is not running
		t.Logf("NewBrowserTool failed (expected without Chrome): %v", err)
		return
	}
	if tool.Name() != "browser" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "browser")
	}
	if tool.Description() == "" {
		t.Error("Description() returned empty string")
	}
	if tool.Parameters() == nil {
		t.Error("Parameters() returned nil")
	}
}

func TestStealthJSGeneration(t *testing.T) {
	js := generateStealthJS()
	if js == "" {
		t.Fatal("generateStealthJS() returned empty string")
	}
	if len(js) < 1000 {
		t.Errorf("Stealth JS too short (%d chars), expected comprehensive patches", len(js))
	}
	// Verify key patches are present
	checks := []string{
		"navigator.webdriver",     // Patch 1
		"window.chrome",           // Patch 2
		"navigator.plugins",       // Patch 3
		"navigator.languages",     // Patch 4
		"Permissions.prototype",   // Patch 5
		"__playwright",            // Patch 6
		"Error.prototype",         // Patch 7
		"debugger",                // Patch 8
		"console",                 // Patch 9
		"outerWidth",              // Patch 10
		"Performance.prototype",   // Patch 11
		"$cdc_",                   // Patch 12
		"contentWindow",           // Patch 13
	}
	for _, check := range checks {
		if !strings.Contains(js, check) {
			t.Errorf("Stealth JS missing patch for %q", check)
		}
	}
}

func TestLaunchChromeArgs(t *testing.T) {
	args := LaunchChromeArgs(9222, true)
	found := false
	for _, a := range args {
		if a == "--remote-debugging-port=9222" {
			found = true
		}
	}
	if !found {
		t.Error("LaunchChromeArgs missing --remote-debugging-port=9222")
	}

	headlessFound := false
	for _, a := range args {
		if a == "--headless=new" {
			headlessFound = true
		}
	}
	if !headlessFound {
		t.Error("LaunchChromeArgs with headless=true missing --headless=new")
	}

	// Test without headless
	args2 := LaunchChromeArgs(9222, false)
	for _, a := range args2 {
		if a == "--headless=new" {
			t.Error("LaunchChromeArgs with headless=false should not have --headless=new")
		}
	}
}

func TestCDPMessageMarshal(t *testing.T) {
	msg := map[string]any{
		"id":     int64(1),
		"method": "Page.navigate",
		"params": map[string]any{
			"url": "https://example.com",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal CDP message: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal CDP message: %v", err)
	}

	if parsed["method"] != "Page.navigate" {
		t.Errorf("method = %v, want Page.navigate", parsed["method"])
	}
}

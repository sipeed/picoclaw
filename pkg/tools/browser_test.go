package tools

import (
	"context"
	"os"
	"testing"
)

// TestBrowserTool_Unit tests browser tool without a real browser connection.
func TestBrowserTool_Unit(t *testing.T) {
	tool := NewBrowserTool(BrowserToolOptions{
		CdpURL:        "ws://localhost:19999",
		Token:         "test",
		Stealth:       true,
		LaunchTimeout: 5000,
		ActionTimeout: 5000,
	})

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "browser" {
			t.Errorf("expected 'browser', got %q", tool.Name())
		}
	})

	t.Run("Parameters", func(t *testing.T) {
		params := tool.Parameters()
		props, ok := params["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties map")
		}
		for _, key := range []string{"action", "url", "selector", "text", "expression", "timeout_ms"} {
			if _, ok := props[key]; !ok {
				t.Errorf("missing parameter: %s", key)
			}
		}
		required, ok := params["required"].([]string)
		if !ok || len(required) != 1 || required[0] != "action" {
			t.Errorf("expected required=[action], got %v", required)
		}
	})

	t.Run("MissingAction", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]interface{}{})
		if !result.IsError {
			t.Error("expected error for missing action")
		}
	})

	t.Run("UnknownAction", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]interface{}{"action": "fly"})
		if !result.IsError {
			t.Error("expected error for unknown action")
		}
	})

	t.Run("CloseWithoutConnect", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]interface{}{"action": "close"})
		if result.IsError {
			t.Error("close on unconnected browser should not error")
		}
	})

	t.Run("NavigateMissingURL", func(t *testing.T) {
		// Will try to connect and fail (no server), but let's check arg validation
		// The tool tries to connect first, so it will fail at connection
		result := tool.Execute(context.Background(), map[string]interface{}{"action": "navigate"})
		if !result.IsError {
			t.Error("expected error")
		}
	})

	t.Run("BuildCdpURL", func(t *testing.T) {
		url := tool.buildCdpURL()
		if url == "" {
			t.Error("expected non-empty CDP URL")
		}
		// Check token is in URL
		if !contains(url, "token=test") {
			t.Errorf("expected token in URL, got: %s", url)
		}
		if !contains(url, "stealth=true") {
			t.Errorf("expected stealth in URL, got: %s", url)
		}
		if !contains(url, "launch=") {
			t.Errorf("expected launch timeout in URL, got: %s", url)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBrowserTool_Integration runs against a real Browserless instance.
// Set BROWSER_TEST_CDP_URL and BROWSER_TEST_TOKEN to enable.
func TestBrowserTool_Integration(t *testing.T) {
	cdpURL := os.Getenv("BROWSER_TEST_CDP_URL")
	token := os.Getenv("BROWSER_TEST_TOKEN")
	if cdpURL == "" {
		t.Skip("BROWSER_TEST_CDP_URL not set, skipping integration test")
	}

	tool := NewBrowserTool(BrowserToolOptions{
		CdpURL:        cdpURL,
		Token:         token,
		Stealth:       true,
		LaunchTimeout: 120000,
		ActionTimeout: 30000,
	})

	ctx := context.Background()

	t.Run("NavigateAndGetText", func(t *testing.T) {
		// Navigate
		result := tool.Execute(ctx, map[string]interface{}{
			"action": "navigate",
			"url":    "https://example.com",
		})
		if result.IsError {
			t.Fatalf("navigate failed: %s", result.ForLLM)
		}
		t.Logf("Navigate result: %s", result.ForLLM)

		// Get text
		result = tool.Execute(ctx, map[string]interface{}{
			"action": "get_text",
		})
		if result.IsError {
			t.Fatalf("get_text failed: %s", result.ForLLM)
		}
		t.Logf("Page text (first 200 chars): %.200s", result.ForLLM)

		if !contains(result.ForLLM, "Example Domain") {
			t.Error("expected 'Example Domain' in page text")
		}
	})

	t.Run("Screenshot", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action": "screenshot",
		})
		if result.IsError {
			t.Fatalf("screenshot failed: %s", result.ForLLM)
		}
		t.Logf("Screenshot result (LLM): %.100s", result.ForLLM)
		if result.ForUser == "" {
			t.Error("expected ForUser with base64 data")
		}
	})

	t.Run("Evaluate", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action":     "evaluate",
			"expression": "document.title",
		})
		if result.IsError {
			t.Fatalf("evaluate failed: %s", result.ForLLM)
		}
		t.Logf("Evaluate result: %s", result.ForLLM)
		if !contains(result.ForLLM, "Example Domain") {
			t.Error("expected 'Example Domain' in eval result")
		}
	})

	t.Run("Scroll", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action":    "scroll",
			"direction": "down",
			"distance":  float64(300),
		})
		if result.IsError {
			t.Fatalf("scroll failed: %s", result.ForLLM)
		}
		t.Logf("Scroll result: %s", result.ForLLM)
	})

	t.Run("Hover", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action":   "hover",
			"selector": "a",
		})
		if result.IsError {
			t.Fatalf("hover failed: %s", result.ForLLM)
		}
		t.Logf("Hover result: %s", result.ForLLM)
	})

	t.Run("Cookies", func(t *testing.T) {
		// Set cookie
		result := tool.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "set",
			"cookie_name":   "test_cookie",
			"cookie_value":  "hello123",
			"cookie_domain": "example.com",
		})
		if result.IsError {
			t.Fatalf("set cookie failed: %s", result.ForLLM)
		}
		t.Logf("Set cookie: %s", result.ForLLM)

		// Get cookies
		result = tool.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "get",
		})
		if result.IsError {
			t.Fatalf("get cookies failed: %s", result.ForLLM)
		}
		t.Logf("Cookies: %s", result.ForLLM)
		if !contains(result.ForLLM, "test_cookie") {
			t.Error("expected test_cookie in cookies list")
		}

		// Delete cookie
		result = tool.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "delete",
			"cookie_name":   "test_cookie",
			"cookie_domain": "example.com",
		})
		if result.IsError {
			t.Fatalf("delete cookie failed: %s", result.ForLLM)
		}
		t.Logf("Delete cookie: %s", result.ForLLM)
	})

	t.Run("PDF", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action": "pdf",
		})
		if result.IsError {
			t.Fatalf("pdf failed: %s", result.ForLLM)
		}
		t.Logf("PDF result: %s", result.ForLLM)
		if result.ForUser == "" || !contains(result.ForUser, "data:application/pdf") {
			t.Error("expected PDF data URI in ForUser")
		}
	})

	t.Run("Close", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]interface{}{
			"action": "close",
		})
		if result.IsError {
			t.Fatalf("close failed: %s", result.ForLLM)
		}
	})
}

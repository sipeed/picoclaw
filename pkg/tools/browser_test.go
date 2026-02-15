package tools

import (
	"context"
	"os"
	"testing"
)

func TestBrowserTool_Unit_Name(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	if bt.Name() != "browser" {
		t.Errorf("expected name 'browser', got %q", bt.Name())
	}
}

func TestBrowserTool_Unit_Parameters(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	params := bt.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if _, ok := props["action"]; !ok {
		t.Error("expected 'action' in parameters")
	}
	if _, ok := props["url"]; !ok {
		t.Error("expected 'url' in parameters")
	}
	if _, ok := props["selector"]; !ok {
		t.Error("expected 'selector' in parameters")
	}
}

func TestBrowserTool_Unit_MissingAction(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	result := bt.Execute(context.Background(), map[string]interface{}{})
	if result.ForLLM != "Error: 'action' parameter is required" {
		t.Errorf("unexpected result: %s", result.ForLLM)
	}
}

func TestBrowserTool_Unit_UnknownAction(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	// Unknown action should fail at connect step since no browser is available,
	// but we test the action routing by checking close (which doesn't need connection)
	result := bt.Execute(context.Background(), map[string]interface{}{
		"action": "close",
	})
	if result.ForLLM != "Browser is not connected" {
		t.Errorf("unexpected result for close without connect: %s", result.ForLLM)
	}
}

func TestBrowserTool_Unit_CloseWithoutConnect(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	result := bt.Execute(context.Background(), map[string]interface{}{
		"action": "close",
	})
	if result.ForLLM != "Browser is not connected" {
		t.Errorf("expected 'Browser is not connected', got %q", result.ForLLM)
	}
}

func TestBrowserTool_Unit_NavigateMissingURL(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	// This will fail at ensureConnected, but we verify the tool doesn't panic
	result := bt.Execute(context.Background(), map[string]interface{}{
		"action": "navigate",
	})
	if result.ForLLM == "" {
		t.Error("expected error message for navigate without URL")
	}
}

func TestBrowserTool_Unit_DefaultProtocol(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{WsURL: "ws://localhost:3000"})
	if bt.protocol != "cdp" {
		t.Errorf("expected default protocol 'cdp', got %q", bt.protocol)
	}
}

func TestBrowserTool_Unit_PlaywrightProtocol(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{
		WsURL:    "ws://localhost:3000",
		Protocol: "playwright",
	})
	if bt.protocol != "playwright" {
		t.Errorf("expected protocol 'playwright', got %q", bt.protocol)
	}
}

func TestBrowserTool_Unit_BuildCdpURL(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{
		WsURL:   "ws://localhost:3000",
		Token:   "test-token",
		Stealth: true,
	})
	url := bt.buildCdpURL()
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
	if !contains(url, "token=test-token") {
		t.Errorf("expected token in URL, got %q", url)
	}
	if !contains(url, "stealth=true") {
		t.Errorf("expected stealth in URL, got %q", url)
	}
}

func TestBrowserTool_Unit_PDFBlockedOnPlaywright(t *testing.T) {
	bt := NewBrowserTool(BrowserToolOptions{
		WsURL:    "ws://localhost:3000",
		Protocol: "playwright",
	})
	// Manually set connected to skip ensureConnected
	bt.connected = true
	bt.page = nil // Will be nil but doPDF checks protocol first

	result := bt.doPDF(map[string]interface{}{})
	if result.ForLLM == "" || !contains(result.ForLLM, "only supported with Chromium") {
		t.Errorf("expected PDF blocked message for playwright protocol, got %q", result.ForLLM)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Integration tests ---
// These require a running browser instance.
// Set BROWSER_TEST_CDP_URL + BROWSER_TEST_TOKEN for CDP (Browserless)
// Set BROWSER_TEST_PW_URL for Playwright Wire Protocol (Camoufox)

func TestBrowserTool_Integration_CDP(t *testing.T) {
	cdpURL := os.Getenv("BROWSER_TEST_CDP_URL")
	token := os.Getenv("BROWSER_TEST_TOKEN")
	if cdpURL == "" {
		t.Skip("BROWSER_TEST_CDP_URL not set, skipping CDP integration test")
	}

	bt := NewBrowserTool(BrowserToolOptions{
		Protocol: "cdp",
		WsURL:    cdpURL,
		Token:    token,
		Stealth:  true,
	})

	ctx := context.Background()
	runBrowserIntegrationTests(t, ctx, bt, true)
}

func TestBrowserTool_Integration_Playwright(t *testing.T) {
	pwURL := os.Getenv("BROWSER_TEST_PW_URL")
	if pwURL == "" {
		t.Skip("BROWSER_TEST_PW_URL not set, skipping Playwright integration test")
	}

	bt := NewBrowserTool(BrowserToolOptions{
		Protocol: "playwright",
		WsURL:    pwURL,
	})

	ctx := context.Background()
	runBrowserIntegrationTests(t, ctx, bt, false)
}

func runBrowserIntegrationTests(t *testing.T, ctx context.Context, bt *BrowserTool, supportsPDF bool) {
	t.Helper()

	// Navigate
	t.Run("Navigate", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action": "navigate",
			"url":    "https://example.com",
		})
		if result.ForLLM == "" {
			t.Fatal("expected non-empty result")
		}
		if contains(result.ForLLM, "Error") {
			t.Fatalf("navigate failed: %s", result.ForLLM)
		}
		t.Log(result.ForLLM)
	})

	// Screenshot
	t.Run("Screenshot", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action": "screenshot",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("screenshot failed: %s", result.ForLLM)
		}
		if result.ForUser == "" {
			t.Error("expected screenshot data in ForUser")
		}
		t.Log(result.ForLLM)
	})

	// GetText
	t.Run("GetText", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action":   "get_text",
			"selector": "h1",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("get_text failed: %s", result.ForLLM)
		}
		if !contains(result.ForLLM, "Example Domain") {
			t.Errorf("expected 'Example Domain' in text, got %q", result.ForLLM)
		}
		t.Log(result.ForLLM)
	})

	// Evaluate
	t.Run("Evaluate", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action":     "evaluate",
			"expression": "document.title",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("evaluate failed: %s", result.ForLLM)
		}
		t.Log(result.ForLLM)
	})

	// Scroll
	t.Run("Scroll", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action":    "scroll",
			"direction": "down",
			"amount":    float64(200),
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("scroll failed: %s", result.ForLLM)
		}
		t.Log(result.ForLLM)
	})

	// Cookies
	t.Run("Cookies", func(t *testing.T) {
		// Set cookie
		result := bt.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "set",
			"cookie_name":   "test_cookie",
			"cookie_value":  "test_value",
			"cookie_domain": "example.com",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("cookie set failed: %s", result.ForLLM)
		}

		// Get cookies
		result = bt.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "get",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("cookie get failed: %s", result.ForLLM)
		}
		if !contains(result.ForLLM, "test_cookie") {
			t.Error("expected test_cookie in cookie list")
		}

		// Delete cookie
		result = bt.Execute(ctx, map[string]interface{}{
			"action":        "cookies",
			"cookie_action": "delete",
			"cookie_name":   "test_cookie",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("cookie delete failed: %s", result.ForLLM)
		}
		t.Log("Cookies test passed")
	})

	// PDF (only for CDP)
	if supportsPDF {
		t.Run("PDF", func(t *testing.T) {
			result := bt.Execute(ctx, map[string]interface{}{
				"action": "pdf",
			})
			if contains(result.ForLLM, "Error") {
				t.Fatalf("pdf failed: %s", result.ForLLM)
			}
			t.Log(result.ForLLM)
		})
	}

	// Close
	t.Run("Close", func(t *testing.T) {
		result := bt.Execute(ctx, map[string]interface{}{
			"action": "close",
		})
		if contains(result.ForLLM, "Error") {
			t.Fatalf("close failed: %s", result.ForLLM)
		}
		t.Log(result.ForLLM)
	})
}

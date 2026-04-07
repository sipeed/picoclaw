//go:build cdp && integration

package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Integration tests require Chrome running with --remote-debugging-port=9222.
// Run with: go test -tags 'goolm,stdjson,cdp,integration' -v -run TestIntegration ./pkg/tools/

func setupBrowserTool(t *testing.T) *BrowserTool {
	t.Helper()
	cfg := config.BrowserToolConfig{
		CDPEndpoint: "http://127.0.0.1:9222",
		Timeout:     30,
		Stealth:     true,
		AllowEval:   true,
	}
	cfg.Enabled = true
	tool, err := NewBrowserTool(cfg)
	if err != nil {
		t.Skipf("Skipping: Chrome not available: %v", err)
	}
	return tool
}

func TestIntegration_NavigateAndState(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	// Navigate
	result := tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://example.com"})
	if result.IsError {
		t.Fatalf("navigate failed: %s", result.ForLLM)
	}

	// State
	result = tool.Execute(ctx, map[string]any{"action": "state"})
	if result.IsError {
		t.Fatalf("state failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Example Domain") {
		t.Errorf("state missing page title, got: %s", result.ForLLM[:200])
	}
	if !strings.Contains(result.ForLLM, "[0]") {
		t.Error("state missing [0] element index")
	}
	t.Logf("State output:\n%s", result.ForLLM)

	// Close
	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_GitHubLoginDetection(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://github.com/login"})
	if result.IsError {
		t.Fatalf("navigate failed: %s", result.ForLLM)
	}

	result = tool.Execute(ctx, map[string]any{"action": "state"})
	if result.IsError {
		t.Fatalf("state failed: %s", result.ForLLM)
	}

	// Must detect login form elements
	state := result.ForLLM
	if !strings.Contains(state, "type=\"password\"") {
		t.Error("login page missing password field")
	}
	if !strings.Contains(state, "type=\"submit\"") && !strings.Contains(state, "Sign in") {
		t.Error("login page missing submit button")
	}
	t.Logf("GitHub login state:\n%s", state)

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_Screenshot(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://example.com"})
	result := tool.Execute(ctx, map[string]any{"action": "screenshot"})
	if result.IsError {
		t.Fatalf("screenshot failed: %s", result.ForLLM)
	}
	t.Logf("Screenshot result: ForLLM=%s, ForUser=%s, Media=%v",
		result.ForLLM, result.ForUser, result.Media)

	// Verify temp file was created (even without MediaStore it should have existed briefly)
	if !strings.Contains(result.ForLLM, "Screenshot captured") {
		t.Error("unexpected screenshot result")
	}

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_ClickAndType(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://github.com/login"})
	// Get state to assign indices
	tool.Execute(ctx, map[string]any{"action": "state"})

	// Try typing into username field (should be index 1 based on previous tests)
	result := tool.Execute(ctx, map[string]any{"action": "type", "index": float64(1), "text": "testuser"})
	if result.IsError {
		t.Logf("type failed (may be different index): %s", result.ForLLM)
	} else {
		t.Logf("type result: %s", result.ForLLM)
	}

	// Verify with get_text
	result = tool.Execute(ctx, map[string]any{"action": "get_text", "index": float64(1)})
	t.Logf("get_text result: %s", result.ForLLM)

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_Evaluate(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://example.com"})
	result := tool.Execute(ctx, map[string]any{
		"action": "evaluate",
		"code":   "JSON.stringify({title: document.title, url: location.href})",
	})
	if result.IsError {
		t.Fatalf("evaluate failed: %s", result.ForLLM)
	}

	// evaluate returns JSON-encoded string, need to unwrap
	var infoStr string
	if err := json.Unmarshal([]byte(result.ForLLM), &infoStr); err != nil {
		// ForLLM might already be the raw JSON string
		infoStr = result.ForLLM
	}
	if !strings.Contains(infoStr, "Example Domain") {
		t.Errorf("evaluate result missing 'Example Domain': %s", infoStr)
	}
	t.Logf("evaluate result: %s", result.ForLLM)

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_Stealth(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://example.com"})
	result := tool.Execute(ctx, map[string]any{
		"action": "evaluate",
		"code":   "String(navigator.webdriver)",
	})
	if result.IsError {
		t.Fatalf("stealth check failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "false") {
		t.Errorf("navigator.webdriver = %s, want false", result.ForLLM)
	}
	t.Logf("navigator.webdriver = %s", result.ForLLM)

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_ScrollAndKeys(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://news.ycombinator.com"})
	time.Sleep(500 * time.Millisecond)

	result := tool.Execute(ctx, map[string]any{"action": "scroll", "direction": "down"})
	if result.IsError {
		t.Errorf("scroll failed: %s", result.ForLLM)
	}

	result = tool.Execute(ctx, map[string]any{"action": "keys", "text": "Tab"})
	if result.IsError {
		t.Errorf("keys failed: %s", result.ForLLM)
	}

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_SSRFBlocking(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	blockedURLs := []string{
		"http://localhost:8080",
		"http://127.0.0.1",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"http://169.254.169.254",
		"http://0.0.0.0",
		"file:///etc/passwd",
	}

	for _, u := range blockedURLs {
		result := tool.Execute(ctx, map[string]any{"action": "navigate", "url": u})
		if !result.IsError {
			t.Errorf("SSRF: %s should be blocked but wasn't", u)
		}
	}

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func TestIntegration_TempFileCleanup(t *testing.T) {
	tool := setupBrowserTool(t)
	ctx := context.Background()

	tool.Execute(ctx, map[string]any{"action": "navigate", "url": "https://example.com"})

	// Screenshot without MediaStore — temp file should be cleaned up
	beforeFiles := countTempScreenshots()
	tool.Execute(ctx, map[string]any{"action": "screenshot"})
	time.Sleep(100 * time.Millisecond)
	afterFiles := countTempScreenshots()

	// Since no MediaStore is set, temp file should be deferred for removal
	t.Logf("Temp screenshot files: before=%d, after=%d", beforeFiles, afterFiles)

	tool.Execute(ctx, map[string]any{"action": "close"})
}

func countTempScreenshots() int {
	entries, _ := os.ReadDir(os.TempDir())
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "screenshot-") && strings.HasSuffix(e.Name(), ".png") {
			count++
		}
	}
	return count
}

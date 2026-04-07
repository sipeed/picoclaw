//go:build cdp

package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

// pageVisit records a single page navigation for browsing history.
type pageVisit struct {
	URL       string
	Title     string
	Timestamp time.Time
}

const maxHistorySize = 50 // keep last 50 pages to bound memory

// BrowserTool provides browser automation capabilities via Chrome DevTools Protocol.
// It enables AI agents to navigate web pages, interact with elements, and extract data.
type BrowserTool struct {
	cfg        config.BrowserToolConfig
	cdp        *CDPClient
	cdpMu      sync.Mutex // guards lazy cdp connection
	chromePath string
	stealthJS  string
	mediaStore media.MediaStore
	history    []pageVisit // browsing history, most recent last
}

// NewBrowserTool creates a new BrowserTool. It verifies that Chrome is available
// and attempts to connect to the CDP endpoint. Returns an error if Chrome is
// not found or the connection fails.
func NewBrowserTool(cfg config.BrowserToolConfig) (*BrowserTool, error) {
	chromePath, err := FindChromePath()
	if err != nil {
		return nil, fmt.Errorf(
			"Chrome/Chromium required for browser tool but not found. "+
				"Install Chrome and restart, or set CHROME_PATH env var. "+
				"Error: %w", err)
	}

	logger.InfoCF("tool", "Chrome found for browser tool",
		map[string]any{"path": chromePath})

	var stealthJS string
	if cfg.Stealth {
		stealthJS = generateStealthJS()
	}

	return &BrowserTool{
		cfg:        cfg,
		chromePath: chromePath,
		stealthJS:  stealthJS,
	}, nil
}

// connectIfNeeded lazily connects to the CDP endpoint.
// It is safe for concurrent use.
func (t *BrowserTool) connectIfNeeded() error {
	t.cdpMu.Lock()
	defer t.cdpMu.Unlock()

	if t.cdp != nil {
		return nil
	}

	endpoint := t.cfg.CDPEndpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9222"
	}

	cdp, err := NewCDPClient(endpoint)
	if err != nil {
		return fmt.Errorf(
			"failed to connect to Chrome CDP at %s. "+
				"Make sure Chrome is running with: %s --remote-debugging-port=9222. "+
				"Error: %w", endpoint, t.chromePath, err)
	}

	// Enable required CDP domains
	if err := cdp.EnablePage(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	if err := cdp.EnableRuntime(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable Runtime domain: %w", err)
	}
	if err := cdp.EnableDOM(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable DOM domain: %w", err)
	}

	// Inject stealth JS if configured
	if t.stealthJS != "" {
		if err := cdp.InjectScript(t.stealthJS); err != nil {
			logger.WarnCF("tool", "Failed to inject stealth JS",
				map[string]any{"error": err.Error()})
		}
	}

	t.cdp = cdp
	return nil
}

func (t *BrowserTool) Name() string { return "browser" }

func (t *BrowserTool) Description() string {
	return `Browser automation via CDP. Actions: navigate, state, click, type, fill, select, screenshot, get_text, scroll, keys, evaluate, close.
Workflow: navigate url → state (get [N] indices) → click/type [N] → state (verify). Always run state after navigate or click.`
}

func (t *BrowserTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Browser action to perform",
				"enum":        []string{"navigate", "state", "click", "type", "fill", "select", "screenshot", "get_text", "scroll", "keys", "evaluate", "close"},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL for navigate action",
			},
			"index": map[string]any{
				"type":        "integer",
				"description": "Element index [N] from state output for click/type/fill/select/get_text",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text for type/fill actions, option value for select, key name for keys",
			},
			"direction": map[string]any{
				"type":        "string",
				"description": "Scroll direction: up or down",
				"enum":        []string{"up", "down"},
			},
			"code": map[string]any{
				"type":        "string",
				"description": "JavaScript code for evaluate action (requires allow_evaluate=true in config)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	// Connect lazily on first use
	if action != "close" {
		if err := t.connectIfNeeded(); err != nil {
			return ErrorResult(err.Error())
		}
	}

	timeout := time.Duration(t.cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch action {
	case "navigate":
		return t.executeNavigate(ctx, args)
	case "state":
		return t.executeState(ctx)
	case "click":
		return t.executeClick(ctx, args)
	case "type":
		return t.executeType(ctx, args)
	case "fill":
		return t.executeFill(ctx, args)
	case "select":
		return t.executeSelect(ctx, args)
	case "screenshot":
		return t.executeScreenshot(ctx)
	case "get_text":
		return t.executeGetText(ctx, args)
	case "scroll":
		return t.executeScroll(ctx, args)
	case "keys":
		return t.executeKeys(ctx, args)
	case "evaluate":
		return t.executeEvaluate(ctx, args)
	case "close":
		return t.executeClose()
	default:
		return ErrorResult(fmt.Sprintf("unknown browser action: %s", action))
	}
}

// --- Action implementations ---

func (t *BrowserTool) executeNavigate(ctx context.Context, args map[string]any) *ToolResult {
	urlStr, _ := args["url"].(string)
	if urlStr == "" {
		return ErrorResult("url is required for navigate action")
	}

	// Validate URL
	if err := validateBrowserURL(urlStr); err != nil {
		return ErrorResult(err.Error())
	}

	if err := t.cdp.Navigate(ctx, urlStr); err != nil {
		return ErrorResult(fmt.Sprintf("navigation failed: %v", err))
	}

	// Record URL in history (title will be updated when state is called)
	t.recordVisit(urlStr, "")

	return SilentResult(fmt.Sprintf("Navigated to %s. Run 'state' to inspect page elements.", urlStr))
}

func (t *BrowserTool) executeState(ctx context.Context) *ToolResult {
	// JavaScript that extracts interactive elements with [N] indices
	js := `(function() {
  var selectors = 'a, button, input, select, textarea, [role="button"], [role="link"], [role="tab"], [onclick], [tabindex]:not([tabindex="-1"])';
  var elements = document.querySelectorAll(selectors);
  var result = [];
  var idx = 0;
  for (var i = 0; i < elements.length; i++) {
    var el = elements[i];
    var rect = el.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) continue;
    if (getComputedStyle(el).visibility === 'hidden') continue;
    if (getComputedStyle(el).display === 'none') continue;
    var tag = el.tagName.toLowerCase();
    var info = {
      i: idx,
      tag: tag,
      text: (el.textContent || '').trim().slice(0, 80).replace(/\s+/g, ' ')
    };
    if (el.getAttribute('role')) info.role = el.getAttribute('role');
    if (el.getAttribute('type')) info.type = el.getAttribute('type');
    if (el.getAttribute('name')) info.name = el.getAttribute('name');
    if (el.getAttribute('placeholder')) info.placeholder = el.getAttribute('placeholder');
    if (el.value !== undefined && el.value !== '') info.value = String(el.value).slice(0, 80);
    if (tag === 'a' && el.getAttribute('href')) info.href = el.getAttribute('href').slice(0, 120);
    if (el.getAttribute('aria-label')) info.label = el.getAttribute('aria-label').slice(0, 80);
    if (el.disabled) info.disabled = true;
    // Store selector path for interaction
    el.setAttribute('data-pcw-idx', String(idx));
    result.push(info);
    idx++;
  }
  return JSON.stringify({
    title: document.title,
    url: location.href,
    elements: result,
    count: idx
  });
})();`

	raw, err := t.cdp.Evaluate(ctx, js)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to extract page state: %v", err))
	}

	// Parse the JSON string returned by JS
	var stateStr string
	if err := json.Unmarshal(raw, &stateStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse state JSON: %v", err))
	}

	var state struct {
		Title    string           `json:"title"`
		URL      string           `json:"url"`
		Elements []map[string]any `json:"elements"`
		Count    int              `json:"count"`
	}
	if err := json.Unmarshal([]byte(stateStr), &state); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse state: %v", err))
	}

	// Update title in history if it changed (SPA navigations)
	t.updateCurrentTitle(state.Title)

	// Format output for LLM consumption
	var sb strings.Builder

	// Include browsing history summary for context
	if summary := t.historySummary(); summary != "" {
		sb.WriteString(summary)
		sb.WriteByte('\n')
	}

	sb.WriteString(fmt.Sprintf("Current page: %s\nURL: %s\n\n", state.Title, state.URL))
	sb.WriteString(fmt.Sprintf("Interactive elements (%d):\n", state.Count))

	for _, el := range state.Elements {
		idxVal, ok := el["i"].(float64)
		if !ok {
			continue
		}
		idx := int(idxVal)
		tag, _ := el["tag"].(string)
		text, _ := el["text"].(string)

		sb.WriteString(fmt.Sprintf("[%d] %s", idx, tag))
		if role, ok := el["role"].(string); ok {
			sb.WriteString(fmt.Sprintf(" role=%q", role))
		}
		if typ, ok := el["type"].(string); ok {
			sb.WriteString(fmt.Sprintf(" type=%q", typ))
		}
		if name, ok := el["name"].(string); ok {
			sb.WriteString(fmt.Sprintf(" name=%q", name))
		}
		if placeholder, ok := el["placeholder"].(string); ok {
			sb.WriteString(fmt.Sprintf(" placeholder=%q", placeholder))
		}
		if value, ok := el["value"].(string); ok {
			sb.WriteString(fmt.Sprintf(" value=%q", value))
		}
		if href, ok := el["href"].(string); ok {
			sb.WriteString(fmt.Sprintf(" href=%q", href))
		}
		if label, ok := el["label"].(string); ok {
			sb.WriteString(fmt.Sprintf(" aria-label=%q", label))
		}
		if disabled, ok := el["disabled"].(bool); ok && disabled {
			sb.WriteString(" [disabled]")
		}
		if text != "" {
			sb.WriteString(fmt.Sprintf(" %q", text))
		}
		sb.WriteByte('\n')
	}

	return SilentResult(sb.String())
}

func (t *BrowserTool) executeClick(ctx context.Context, args map[string]any) *ToolResult {
	index, ok := getIntArg(args, "index")
	if !ok {
		return ErrorResult("index is required for click action (use [N] from state output)")
	}

	// Click using the data-pcw-idx attribute we set during state
	js := fmt.Sprintf(`(function() {
  var el = document.querySelector('[data-pcw-idx="%d"]');
  if (!el) return JSON.stringify({error: 'Element [%d] not found. Run state to refresh indices.'});
  el.scrollIntoView({block: 'center', behavior: 'instant'});
  el.click();
  return JSON.stringify({ok: true, tag: el.tagName.toLowerCase(), text: (el.textContent || '').trim().slice(0, 40)});
})();`, index, index)

	raw, err := t.cdp.Evaluate(ctx, js)
	if err != nil {
		return ErrorResult(fmt.Sprintf("click failed: %v", err))
	}

	var resultStr string
	if err := json.Unmarshal(raw, &resultStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse click result: %v", err))
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Tag   string `json:"tag"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal([]byte(resultStr), &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse click result: %v", err))
	}

	if result.Error != "" {
		return ErrorResult(result.Error)
	}

	return SilentResult(fmt.Sprintf("Clicked [%d] <%s> %q. Run 'state' to see updated page.", index, result.Tag, result.Text))
}

func (t *BrowserTool) executeType(ctx context.Context, args map[string]any) *ToolResult {
	index, ok := getIntArg(args, "index")
	if !ok {
		return ErrorResult("index is required for type action")
	}
	text, _ := args["text"].(string)
	if text == "" {
		return ErrorResult("text is required for type action")
	}

	// Focus the element
	focusJS := fmt.Sprintf(`(function() {
  var el = document.querySelector('[data-pcw-idx="%d"]');
  if (!el) return JSON.stringify({error: 'Element [%d] not found. Run state to refresh indices.'});
  el.scrollIntoView({block: 'center', behavior: 'instant'});
  el.focus();
  return JSON.stringify({ok: true});
})();`, index, index)

	raw, err := t.cdp.Evaluate(ctx, focusJS)
	if err != nil {
		return ErrorResult(fmt.Sprintf("focus failed: %v", err))
	}

	var focusStr string
	if err := json.Unmarshal(raw, &focusStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse focus result: %v", err))
	}
	if strings.Contains(focusStr, "error") {
		var result struct{ Error string `json:"error"` }
		json.Unmarshal([]byte(focusStr), &result)
		if result.Error != "" {
			return ErrorResult(result.Error)
		}
	}

	// Type using CDP Input.insertText
	if err := t.cdp.InsertText(text); err != nil {
		return ErrorResult(fmt.Sprintf("type failed: %v", err))
	}

	return SilentResult(fmt.Sprintf("Typed %q into [%d].", text, index))
}

func (t *BrowserTool) executeFill(ctx context.Context, args map[string]any) *ToolResult {
	index, ok := getIntArg(args, "index")
	if !ok {
		return ErrorResult("index is required for fill action")
	}
	text, _ := args["text"].(string)
	if text == "" {
		return ErrorResult("text is required for fill action")
	}

	// Focus, select all existing text, then type new text
	clearJS := fmt.Sprintf(`(function() {
  var el = document.querySelector('[data-pcw-idx="%d"]');
  if (!el) return JSON.stringify({error: 'Element [%d] not found.'});
  el.scrollIntoView({block: 'center', behavior: 'instant'});
  el.focus();
  el.value = '';
  el.dispatchEvent(new Event('input', {bubbles: true}));
  return JSON.stringify({ok: true});
})();`, index, index)

	raw, err := t.cdp.Evaluate(ctx, clearJS)
	if err != nil {
		return ErrorResult(fmt.Sprintf("fill failed (clear step): %v", err))
	}

	var clearStr string
	if err := json.Unmarshal(raw, &clearStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse fill result: %v", err))
	}
	if strings.Contains(clearStr, "error") {
		var result struct{ Error string `json:"error"` }
		json.Unmarshal([]byte(clearStr), &result)
		if result.Error != "" {
			return ErrorResult(result.Error)
		}
	}

	// Type new value
	if err := t.cdp.InsertText(text); err != nil {
		return ErrorResult(fmt.Sprintf("fill failed (type step): %v", err))
	}

	return SilentResult(fmt.Sprintf("Filled [%d] with %q.", index, text))
}

func (t *BrowserTool) executeSelect(ctx context.Context, args map[string]any) *ToolResult {
	index, ok := getIntArg(args, "index")
	if !ok {
		return ErrorResult("index is required for select action")
	}
	text, _ := args["text"].(string)
	if text == "" {
		return ErrorResult("text (option value) is required for select action")
	}

	textJSON, _ := json.Marshal(text)
	js := fmt.Sprintf(`(function() {
  var el = document.querySelector('[data-pcw-idx="%d"]');
  if (!el) return JSON.stringify({error: 'Element [%d] not found.'});
  if (el.tagName.toLowerCase() !== 'select') return JSON.stringify({error: 'Element [%d] is not a <select>'});
  var target = %s;
  var opts = el.options;
  for (var i = 0; i < opts.length; i++) {
    if (opts[i].value === target || opts[i].text.trim() === target) {
      el.value = opts[i].value;
      el.dispatchEvent(new Event('change', {bubbles: true}));
      return JSON.stringify({ok: true, selected: opts[i].text.trim()});
    }
  }
  var available = [];
  for (var j = 0; j < opts.length; j++) available.push(opts[j].text.trim());
  return JSON.stringify({error: 'Option not found. Available: ' + available.join(', ')});
})();`, index, index, index, string(textJSON))

	raw, err := t.cdp.Evaluate(ctx, js)
	if err != nil {
		return ErrorResult(fmt.Sprintf("select failed: %v", err))
	}

	var resultStr string
	if err := json.Unmarshal(raw, &resultStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse select result: %v", err))
	}
	var result struct {
		OK       bool   `json:"ok"`
		Error    string `json:"error"`
		Selected string `json:"selected"`
	}
	if err := json.Unmarshal([]byte(resultStr), &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse select result: %v", err))
	}

	if result.Error != "" {
		return ErrorResult(result.Error)
	}
	return SilentResult(fmt.Sprintf("Selected %q in [%d].", result.Selected, index))
}

func (t *BrowserTool) executeScreenshot(ctx context.Context) *ToolResult {
	data, err := t.cdp.CaptureScreenshot("png", 0)
	if err != nil {
		return ErrorResult(fmt.Sprintf("screenshot failed: %v", err))
	}

	// Decode base64 PNG data
	pngBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to decode screenshot: %v", err))
	}

	// Write to temporary file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("screenshot-%d.png", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, pngBytes, 0600); err != nil {
		return ErrorResult(fmt.Sprintf("failed to save screenshot: %v", err))
	}

	// Build scope from channel and chatID context
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	scope := fmt.Sprintf("tool:browser:screenshot:%s:%s", channel, chatID)

	// Store in media store if available; otherwise return inline info to LLM
	if t.mediaStore != nil {
		ref, err := t.mediaStore.Store(tmpFile, media.MediaMeta{
			Filename:    "screenshot.png",
			ContentType: "image/png",
			Source:      "tool:browser.screenshot",
		}, scope)
		if err != nil {
			logger.WarnCF("tool", "Failed to store screenshot",
				map[string]any{"error": err.Error()})
			// Fall through to inline path
		} else {
			os.Remove(tmpFile)
			return &ToolResult{
				ForLLM:          "Screenshot captured and sent to user (PNG).",
				ForUser:         "Screenshot captured",
				Media:           []string{ref},
				ResponseHandled: true,
			}
		}
	}

	// No MediaStore or store failed — save to disk and return path as artifact
	// so LLM can reference it and user can access it via send_file if needed
	return &ToolResult{
		ForLLM:       fmt.Sprintf("Screenshot saved to %s (%d KB). Use send_file to deliver to user if needed.", tmpFile, len(pngBytes)/1024),
		ArtifactTags: []string{fmt.Sprintf("[file:%s]", tmpFile)},
	}
}

// SetMediaStore implements the mediaStoreAware interface.
func (t *BrowserTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *BrowserTool) executeGetText(ctx context.Context, args map[string]any) *ToolResult {
	index, ok := getIntArg(args, "index")
	if !ok {
		return ErrorResult("index is required for get_text action")
	}

	js := fmt.Sprintf(`(function() {
  var el = document.querySelector('[data-pcw-idx="%d"]');
  if (!el) return JSON.stringify({error: 'Element [%d] not found.'});
  var text = el.value !== undefined && el.value !== '' ? el.value : el.textContent;
  return JSON.stringify({text: (text || '').trim()});
})();`, index, index)

	raw, err := t.cdp.Evaluate(ctx, js)
	if err != nil {
		return ErrorResult(fmt.Sprintf("get_text failed: %v", err))
	}

	var resultStr string
	if err := json.Unmarshal(raw, &resultStr); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse get_text result: %v", err))
	}
	var result struct {
		Text  string `json:"text"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(resultStr), &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse get_text result: %v", err))
	}

	if result.Error != "" {
		return ErrorResult(result.Error)
	}
	return SilentResult(fmt.Sprintf("[%d] text: %q", index, result.Text))
}

func (t *BrowserTool) executeScroll(ctx context.Context, args map[string]any) *ToolResult {
	direction, _ := args["direction"].(string)
	if direction == "" {
		// Also accept from "text" parameter for convenience
		direction, _ = args["text"].(string)
	}

	switch direction {
	case "up":
		// ok
	case "down":
		// ok
	default:
		return ErrorResult("direction must be 'up' or 'down'")
	}

	amount := 500 // pixels
	if direction == "up" {
		amount = -500
	}

	js := fmt.Sprintf(`window.scrollBy(0, %d); JSON.stringify({scrollY: window.scrollY});`, amount)
	_, err := t.cdp.Evaluate(ctx, js)
	if err != nil {
		return ErrorResult(fmt.Sprintf("scroll failed: %v", err))
	}

	return SilentResult(fmt.Sprintf("Scrolled %s. Run 'state' to see updated elements.", direction))
}

func (t *BrowserTool) executeKeys(_ context.Context, args map[string]any) *ToolResult {
	text, _ := args["text"].(string)
	if text == "" {
		return ErrorResult("text (key name) is required for keys action. Examples: Enter, Tab, Escape, ArrowDown")
	}

	// Map common key names
	key := text
	switch strings.ToLower(text) {
	case "enter", "return":
		key = "Enter"
	case "tab":
		key = "Tab"
	case "escape", "esc":
		key = "Escape"
	case "backspace":
		key = "Backspace"
	case "delete":
		key = "Delete"
	case "arrowup", "up":
		key = "ArrowUp"
	case "arrowdown", "down":
		key = "ArrowDown"
	case "arrowleft", "left":
		key = "ArrowLeft"
	case "arrowright", "right":
		key = "ArrowRight"
	}

	if err := t.cdp.DispatchKeyEvent("keyDown", key, 0); err != nil {
		return ErrorResult(fmt.Sprintf("keyDown failed: %v", err))
	}
	if err := t.cdp.DispatchKeyEvent("keyUp", key, 0); err != nil {
		return ErrorResult(fmt.Sprintf("keyUp failed: %v", err))
	}

	return SilentResult(fmt.Sprintf("Pressed key: %s", key))
}

func (t *BrowserTool) executeEvaluate(ctx context.Context, args map[string]any) *ToolResult {
	if !t.cfg.AllowEval {
		return ErrorResult("evaluate action is disabled. Set tools.browser.allow_evaluate=true in config to enable JavaScript execution.")
	}

	code, _ := args["code"].(string)
	if code == "" {
		return ErrorResult("code is required for evaluate action")
	}

	raw, err := t.cdp.Evaluate(ctx, code)
	if err != nil {
		errMsg := err.Error()
		// Provide actionable guidance based on error type
		var hint string
		if strings.Contains(errMsg, "not a function") || strings.Contains(errMsg, "is not iterable") {
			hint = " Hint: the variable type may differ from expected. Use typeof/Array.isArray to check, or try document.querySelectorAll instead."
		} else if strings.Contains(errMsg, "already been declared") {
			hint = " Hint: wrap your code in an IIFE: (function(){ ... })()"
		} else if strings.Contains(errMsg, "Failed to fetch") || strings.Contains(errMsg, "NetworkError") {
			hint = " Hint: cross-origin fetch may be blocked. Try extracting data from the DOM directly instead of calling APIs."
		} else if strings.Contains(errMsg, "not defined") {
			hint = " Hint: the variable/function does not exist on this page. Use 'state' to see available elements."
		}
		return ErrorResult(fmt.Sprintf("evaluate failed: %v.%s", err, hint))
	}

	if raw == nil {
		return SilentResult("null")
	}
	result, _ := json.MarshalIndent(json.RawMessage(raw), "", "  ")
	return SilentResult(string(result))
}

func (t *BrowserTool) executeClose() *ToolResult {
	if t.cdp != nil {
		t.cdp.Close()
		t.cdp = nil
	}
	t.history = nil
	return SilentResult("Browser session closed.")
}

// --- Browsing history helpers ---

// recordVisit adds a page to the browsing history.
func (t *BrowserTool) recordVisit(pageURL, title string) {
	// Avoid duplicate consecutive entries (e.g. navigate then state on same page)
	if len(t.history) > 0 {
		last := &t.history[len(t.history)-1]
		if last.URL == pageURL {
			if title != "" {
				last.Title = title
			}
			return
		}
	}

	t.history = append(t.history, pageVisit{
		URL:       pageURL,
		Title:     title,
		Timestamp: time.Now(),
	})

	// Trim to max size
	if len(t.history) > maxHistorySize {
		t.history = t.history[len(t.history)-maxHistorySize:]
	}
}

// updateCurrentTitle updates the title of the most recent history entry.
// Useful when a SPA changes title after initial navigation.
func (t *BrowserTool) updateCurrentTitle(title string) {
	if len(t.history) > 0 && title != "" {
		t.history[len(t.history)-1].Title = title
	}
}

// historySummary returns a compact browsing history for LLM context.
// Only shown when there are 2+ pages visited (no point showing history for the first page).
func (t *BrowserTool) historySummary() string {
	if len(t.history) < 2 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Browsing history (%d pages):\n", len(t.history)))

	// Show all entries compactly: just index, title, URL
	for i, v := range t.history {
		marker := "  "
		if i == len(t.history)-1 {
			marker = "> " // current page marker
		}
		title := v.Title
		if title == "" {
			title = "(untitled)"
		}
		sb.WriteString(fmt.Sprintf("%s%d. %s — %s\n", marker, i+1, title, v.URL))
	}

	return sb.String()
}

// --- Helpers ---

// validateBrowserURL checks that a URL is safe to navigate to.
// It blocks private networks, loopback, metadata endpoints, and non-HTTP schemes.
func validateBrowserURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed (got %s://)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("missing host in URL")
	}

	hostname := parsed.Hostname()

	// Block obvious private/local hosts (string check)
	if hostname == "localhost" || hostname == "0.0.0.0" ||
		hostname == "metadata.google.internal" {
		return fmt.Errorf("navigation to %s is not allowed", hostname)
	}

	// Parse as IP to catch all representations (hex, octal, IPv4-mapped IPv6, etc.)
	ip := net.ParseIP(hostname)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("navigation to private/local IP %s is not allowed", hostname)
		}
		// Block cloud metadata IPs
		if ip.Equal(net.ParseIP("169.254.169.254")) {
			return fmt.Errorf("navigation to cloud metadata endpoint is not allowed")
		}
	} else {
		// Hostname (not literal IP) — resolve DNS with short timeout to check for private IPs
		resolver := &net.Resolver{}
		dnsCtx, dnsCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer dnsCancel()
		addrs, err := resolver.LookupHost(dnsCtx, hostname)
		if err == nil {
			for _, addr := range addrs {
				resolved := net.ParseIP(addr)
				if resolved == nil {
					continue
				}
				if resolved.IsLoopback() || resolved.IsPrivate() || resolved.IsLinkLocalUnicast() ||
					resolved.IsLinkLocalMulticast() || resolved.IsUnspecified() {
					return fmt.Errorf("hostname %s resolves to private IP %s, navigation not allowed", hostname, addr)
				}
				if resolved.Equal(net.ParseIP("169.254.169.254")) {
					return fmt.Errorf("hostname %s resolves to metadata IP, navigation not allowed", hostname)
				}
			}
		}
	}

	return nil
}

// getIntArg extracts an integer from args, handling both float64 (JSON) and int types.
func getIntArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

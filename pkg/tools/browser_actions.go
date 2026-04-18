//go:build cdp

package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

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
  // Clear stale indices from previous state calls to prevent interaction
  // with elements that are no longer considered interactive/visible.
  var stale = document.querySelectorAll('[data-pcw-idx]');
  for (var s = 0; s < stale.length; s++) stale[s].removeAttribute('data-pcw-idx');

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
  el.scrollIntoView({block: 'center', behavior: 'auto'});
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
  el.scrollIntoView({block: 'center', behavior: 'auto'});
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
		if err := json.Unmarshal([]byte(focusStr), &result); err != nil {
			return ErrorResult(fmt.Sprintf("failed to parse focus result: %v", err))
		}
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
  el.scrollIntoView({block: 'center', behavior: 'auto'});
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
		if err := json.Unmarshal([]byte(clearStr), &result); err != nil {
			return ErrorResult(fmt.Sprintf("failed to parse fill result: %v", err))
		}
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
			// Note: do NOT remove tmpFile here. MediaStore.Store records a
			// path mapping without copying the file. Removing it would make
			// the media reference unresolvable. Let MediaStore manage the
			// file lifecycle via its CleanupPolicy.
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

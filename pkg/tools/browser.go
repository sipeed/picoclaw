package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// BrowserToolOptions configures the browser tool.
type BrowserToolOptions struct {
	Protocol      string // "cdp" (default) or "playwright"
	WsURL         string // WebSocket URL for the remote browser
	Token         string // Auth token (used for CDP/Browserless)
	Stealth       bool   // Request stealth mode via launch params
	LaunchTimeout int    // Connection timeout in ms (default 120000)
	ActionTimeout int    // Per-action timeout in ms (default 30000)
}

// BrowserTool provides browser automation via Playwright.
// Supports two connection protocols:
//   - "cdp": Chromium via ConnectOverCDP (e.g. Browserless)
//   - "playwright": Firefox via Connect (e.g. Camoufox)
type BrowserTool struct {
	wsURL         string
	token         string
	protocol      string
	stealth       bool
	launchTimeout time.Duration
	actionTimeout time.Duration

	pw        *playwright.Playwright
	browser   playwright.Browser
	page      playwright.Page
	connected bool
	mu        sync.Mutex
}

// NewBrowserTool creates a new browser tool with the given options.
func NewBrowserTool(opts BrowserToolOptions) *BrowserTool {
	protocol := opts.Protocol
	if protocol == "" {
		protocol = "cdp"
	}

	launchTimeout := 120 * time.Second
	if opts.LaunchTimeout > 0 {
		launchTimeout = time.Duration(opts.LaunchTimeout) * time.Millisecond
	}

	actionTimeout := 30 * time.Second
	if opts.ActionTimeout > 0 {
		actionTimeout = time.Duration(opts.ActionTimeout) * time.Millisecond
	}

	return &BrowserTool{
		wsURL:         opts.WsURL,
		token:         opts.Token,
		protocol:      protocol,
		stealth:       opts.Stealth,
		launchTimeout: launchTimeout,
		actionTimeout: actionTimeout,
	}
}

func (t *BrowserTool) Name() string { return "browser" }

func (t *BrowserTool) Description() string {
	return "Control a remote browser to navigate web pages, take screenshots, click elements, fill forms, and extract content. " +
		"Supports Chromium (CDP) and Firefox (Playwright Wire Protocol)."
}

func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The browser action to perform",
				"enum": []string{
					"navigate", "click", "type", "screenshot", "get_text",
					"evaluate", "wait", "scroll", "hover", "select",
					"pdf", "cookies", "close",
				},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to (for 'navigate' action)",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for the target element",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to type into the element (for 'type' action)",
			},
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript expression to evaluate (for 'evaluate' action)",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "Scroll direction: 'up' or 'down' (for 'scroll' action)",
				"enum":        []string{"up", "down"},
			},
			"amount": map[string]interface{}{
				"type":        "number",
				"description": "Scroll amount in pixels (default 500)",
			},
			"values": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Values to select (for 'select' action)",
			},
			"cookie_action": map[string]interface{}{
				"type":        "string",
				"description": "Cookie sub-action: 'get', 'set', 'delete', 'clear'",
				"enum":        []string{"get", "set", "delete", "clear"},
			},
			"cookie_name": map[string]interface{}{
				"type":        "string",
				"description": "Cookie name (for set/delete)",
			},
			"cookie_value": map[string]interface{}{
				"type":        "string",
				"description": "Cookie value (for set)",
			},
			"cookie_domain": map[string]interface{}{
				"type":        "string",
				"description": "Cookie domain (for set)",
			},
			"cookie_url": map[string]interface{}{
				"type":        "string",
				"description": "Cookie URL (for set, alternative to domain)",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Action timeout in milliseconds (overrides default)",
			},
			"full_page": map[string]interface{}{
				"type":        "boolean",
				"description": "Take full-page screenshot (default true)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return &ToolResult{ForLLM: "Error: 'action' parameter is required"}
	}

	// Ensure browser connection (except for close action)
	if action != "close" {
		if err := t.ensureConnected(); err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error connecting to browser: %v", err)}
		}
	}

	switch action {
	case "navigate":
		return t.doNavigate(args)
	case "click":
		return t.doClick(args)
	case "type":
		return t.doType(args)
	case "screenshot":
		return t.doScreenshot(args)
	case "get_text":
		return t.doGetText(args)
	case "evaluate":
		return t.doEvaluate(args)
	case "wait":
		return t.doWait(args)
	case "scroll":
		return t.doScroll(args)
	case "hover":
		return t.doHover(args)
	case "select":
		return t.doSelect(args)
	case "pdf":
		return t.doPDF(args)
	case "cookies":
		return t.doCookies(args)
	case "close":
		return t.doClose()
	default:
		return &ToolResult{ForLLM: fmt.Sprintf("Error: unknown action '%s'", action)}
	}
}

// ensureConnected establishes a browser connection if not already connected.
func (t *BrowserTool) ensureConnected() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected && t.page != nil {
		return nil
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start playwright driver: %w", err)
	}

	timeoutMs := float64(t.launchTimeout.Milliseconds())

	var browser playwright.Browser

	switch t.protocol {
	case "playwright":
		// Firefox via Playwright Wire Protocol (e.g. Camoufox)
		browser, err = pw.Firefox.Connect(t.wsURL, playwright.BrowserTypeConnectOptions{
			Timeout: &timeoutMs,
		})
		if err != nil {
			pw.Stop()
			return fmt.Errorf("failed to connect via Playwright protocol to %s: %w", t.wsURL, err)
		}
	default:
		// Chromium via CDP (e.g. Browserless)
		cdpURL := t.buildCdpURL()
		browser, err = pw.Chromium.ConnectOverCDP(cdpURL, playwright.BrowserTypeConnectOverCDPOptions{
			Timeout: &timeoutMs,
		})
		if err != nil {
			pw.Stop()
			return fmt.Errorf("failed to connect via CDP to %s: %w", cdpURL, err)
		}
	}

	// Reuse existing page from browser context if available (common for CDP)
	var page playwright.Page
	contexts := browser.Contexts()
	var pages []playwright.Page
	if len(contexts) > 0 {
		pages = contexts[0].Pages()
	}
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = browser.NewPage()
		if err != nil {
			browser.Close()
			pw.Stop()
			return fmt.Errorf("failed to create new page: %w", err)
		}
	}

	t.pw = pw
	t.browser = browser
	t.page = page
	t.connected = true

	return nil
}

// buildCdpURL constructs the CDP WebSocket URL with token and launch params.
func (t *BrowserTool) buildCdpURL() string {
	u := t.wsURL

	params := []string{}
	if t.token != "" {
		params = append(params, "token="+t.token)
	}
	if t.stealth {
		params = append(params, "stealth=true")
	}
	if t.launchTimeout > 0 {
		launchJSON := fmt.Sprintf(`{"timeout":%d}`, t.launchTimeout.Milliseconds())
		params = append(params, "launch="+launchJSON)
	}

	if len(params) > 0 {
		separator := "?"
		if strings.Contains(u, "?") {
			separator = "&"
		}
		u += separator + strings.Join(params, "&")
	}

	return u
}

// getTimeout returns the action timeout, optionally overridden by args.
func (t *BrowserTool) getTimeout(args map[string]interface{}) float64 {
	if v, ok := args["timeout"].(float64); ok && v > 0 {
		return v
	}
	return float64(t.actionTimeout.Milliseconds())
}

func (t *BrowserTool) doNavigate(args map[string]interface{}) *ToolResult {
	url, _ := args["url"].(string)
	if url == "" {
		return &ToolResult{ForLLM: "Error: 'url' parameter is required for navigate action"}
	}

	timeout := t.getTimeout(args)
	resp, err := t.page.Goto(url, playwright.PageGotoOptions{
		Timeout:   &timeout,
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error navigating to %s: %v", url, err)}
	}

	status := 0
	if resp != nil {
		status = resp.Status()
	}

	title, _ := t.page.Title()

	return &ToolResult{
		ForLLM: fmt.Sprintf("Navigated to %s (status: %d, title: %q)", url, status, title),
	}
}

func (t *BrowserTool) doClick(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		return &ToolResult{ForLLM: "Error: 'selector' parameter is required for click action"}
	}

	timeout := t.getTimeout(args)
	err := t.page.Click(selector, playwright.PageClickOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error clicking %q: %v", selector, err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Clicked element: %s", selector)}
}

func (t *BrowserTool) doType(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		return &ToolResult{ForLLM: "Error: 'selector' parameter is required for type action"}
	}
	text, _ := args["text"].(string)

	timeout := t.getTimeout(args)
	err := t.page.Fill(selector, text, playwright.PageFillOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error typing into %q: %v", selector, err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Typed %d characters into %s", len(text), selector)}
}

func (t *BrowserTool) doScreenshot(args map[string]interface{}) *ToolResult {
	fullPage := true
	if v, ok := args["full_page"].(bool); ok {
		fullPage = v
	}

	selector, _ := args["selector"].(string)

	var data []byte
	var err error

	if selector != "" {
		// Screenshot a specific element
		locator := t.page.Locator(selector)
		timeout := t.getTimeout(args)
		data, err = locator.Screenshot(playwright.LocatorScreenshotOptions{
			Timeout: &timeout,
		})
	} else {
		// Full page or viewport screenshot
		data, err = t.page.Screenshot(playwright.PageScreenshotOptions{
			FullPage: playwright.Bool(fullPage),
		})
	}

	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error taking screenshot: %v", err)}
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	title, _ := t.page.Title()
	url := t.page.URL()

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Screenshot taken (page: %q, url: %s, size: %d bytes). Image data returned as base64.", title, url, len(data)),
		ForUser: fmt.Sprintf("![screenshot](data:image/png;base64,%s)", encoded),
	}
}

func (t *BrowserTool) doGetText(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		selector = "body"
	}

	timeout := t.getTimeout(args)
	text, err := t.page.InnerText(selector, playwright.PageInnerTextOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error getting text from %q: %v", selector, err)}
	}

	// Truncate very long text
	const maxLen = 50000
	if len(text) > maxLen {
		text = text[:maxLen] + "\n... [truncated]"
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Text from %q (%d chars):\n%s", selector, len(text), text)}
}

func (t *BrowserTool) doEvaluate(args map[string]interface{}) *ToolResult {
	expression, _ := args["expression"].(string)
	if expression == "" {
		return &ToolResult{ForLLM: "Error: 'expression' parameter is required for evaluate action"}
	}

	result, err := t.page.Evaluate(expression)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error evaluating JS: %v", err)}
	}

	resultJSON, _ := json.Marshal(result)
	return &ToolResult{ForLLM: fmt.Sprintf("Evaluation result: %s", string(resultJSON))}
}

func (t *BrowserTool) doWait(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		return &ToolResult{ForLLM: "Error: 'selector' parameter is required for wait action"}
	}

	timeout := t.getTimeout(args)
	locator := t.page.Locator(selector)
	err := locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error waiting for %q: %v", selector, err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Element %q is now visible", selector)}
}

func (t *BrowserTool) doScroll(args map[string]interface{}) *ToolResult {
	direction, _ := args["direction"].(string)
	if direction == "" {
		direction = "down"
	}

	amount := 500.0
	if v, ok := args["amount"].(float64); ok && v > 0 {
		amount = v
	}

	if direction == "up" {
		amount = -amount
	}

	js := fmt.Sprintf("window.scrollBy(0, %f)", amount)
	_, err := t.page.Evaluate(js)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error scrolling: %v", err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Scrolled %s by %.0f pixels", direction, amount)}
}

func (t *BrowserTool) doHover(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		return &ToolResult{ForLLM: "Error: 'selector' parameter is required for hover action"}
	}

	timeout := t.getTimeout(args)
	err := t.page.Hover(selector, playwright.PageHoverOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error hovering on %q: %v", selector, err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Hovering on element: %s", selector)}
}

func (t *BrowserTool) doSelect(args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		return &ToolResult{ForLLM: "Error: 'selector' parameter is required for select action"}
	}

	rawValues, _ := args["values"].([]interface{})
	values := make([]string, 0, len(rawValues))
	for _, v := range rawValues {
		if s, ok := v.(string); ok {
			values = append(values, s)
		}
	}

	timeout := t.getTimeout(args)
	chosen, err := t.page.SelectOption(selector, playwright.SelectOptionValues{
		Values: &values,
	}, playwright.PageSelectOptionOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error selecting options in %q: %v", selector, err)}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Selected %d option(s) in %s: %v", len(chosen), selector, chosen)}
}

func (t *BrowserTool) doPDF(args map[string]interface{}) *ToolResult {
	if t.protocol == "playwright" {
		return &ToolResult{ForLLM: "Error: PDF generation is only supported with Chromium (CDP protocol). Current protocol is 'playwright' (Firefox)."}
	}

	data, err := t.page.PDF()
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error generating PDF: %v", err)}
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return &ToolResult{
		ForLLM: fmt.Sprintf("PDF generated (%d bytes). Data returned as base64.", len(data)),
		ForUser: encoded,
	}
}

func (t *BrowserTool) doCookies(args map[string]interface{}) *ToolResult {
	cookieAction, _ := args["cookie_action"].(string)
	if cookieAction == "" {
		return &ToolResult{ForLLM: "Error: 'cookie_action' parameter is required (get, set, delete, clear)"}
	}

	browserCtx := t.page.Context()

	switch cookieAction {
	case "get":
		cookies, err := browserCtx.Cookies()
		if err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error getting cookies: %v", err)}
		}
		data, _ := json.MarshalIndent(cookies, "", "  ")
		return &ToolResult{ForLLM: fmt.Sprintf("Cookies (%d):\n%s", len(cookies), string(data))}

	case "set":
		name, _ := args["cookie_name"].(string)
		value, _ := args["cookie_value"].(string)
		domain, _ := args["cookie_domain"].(string)
		cookieURL, _ := args["cookie_url"].(string)

		if name == "" {
			return &ToolResult{ForLLM: "Error: 'cookie_name' is required for set"}
		}

		cookie := playwright.OptionalCookie{
			Name:  name,
			Value: value,
		}

		if cookieURL != "" {
			cookie.URL = &cookieURL
		} else if domain != "" {
			cookie.Domain = &domain
			defaultPath := "/"
			cookie.Path = &defaultPath
		} else {
			// Use current page URL as fallback
			pageURL := t.page.URL()
			cookie.URL = &pageURL
		}

		err := browserCtx.AddCookies([]playwright.OptionalCookie{cookie})
		if err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error setting cookie: %v", err)}
		}
		return &ToolResult{ForLLM: fmt.Sprintf("Cookie %q set successfully", name)}

	case "delete":
		name, _ := args["cookie_name"].(string)
		if name == "" {
			return &ToolResult{ForLLM: "Error: 'cookie_name' is required for delete"}
		}

		// Get all cookies, clear, re-add all except target
		cookies, err := browserCtx.Cookies()
		if err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error getting cookies for delete: %v", err)}
		}

		err = browserCtx.ClearCookies()
		if err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error clearing cookies: %v", err)}
		}

		var toReAdd []playwright.OptionalCookie
		for _, c := range cookies {
			if c.Name == name {
				continue
			}
			oc := playwright.OptionalCookie{
				Name:  c.Name,
				Value: c.Value,
			}
			if c.Domain != "" {
				oc.Domain = &c.Domain
				oc.Path = &c.Path
			}
			toReAdd = append(toReAdd, oc)
		}

		if len(toReAdd) > 0 {
			if err := browserCtx.AddCookies(toReAdd); err != nil {
				return &ToolResult{ForLLM: fmt.Sprintf("Error re-adding cookies after delete: %v", err)}
			}
		}

		return &ToolResult{ForLLM: fmt.Sprintf("Cookie %q deleted", name)}

	case "clear":
		err := browserCtx.ClearCookies()
		if err != nil {
			return &ToolResult{ForLLM: fmt.Sprintf("Error clearing cookies: %v", err)}
		}
		return &ToolResult{ForLLM: "All cookies cleared"}

	default:
		return &ToolResult{ForLLM: fmt.Sprintf("Error: unknown cookie_action %q", cookieAction)}
	}
}

func (t *BrowserTool) doClose() *ToolResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return &ToolResult{ForLLM: "Browser is not connected"}
	}

	var errs []string

	if t.browser != nil {
		if err := t.browser.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("browser close: %v", err))
		}
	}
	if t.pw != nil {
		if err := t.pw.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("playwright stop: %v", err))
		}
	}

	t.page = nil
	t.browser = nil
	t.pw = nil
	t.connected = false

	if len(errs) > 0 {
		return &ToolResult{ForLLM: fmt.Sprintf("Browser closed with warnings: %s", strings.Join(errs, "; "))}
	}

	return &ToolResult{ForLLM: "Browser closed successfully"}
}

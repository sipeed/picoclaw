package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// BrowserToolOptions configures the browser tool.
type BrowserToolOptions struct {
	CdpURL        string
	Token         string
	Stealth       bool
	LaunchTimeout int // ms
	ActionTimeout int // ms
}

// BrowserTool provides browser automation via Chrome DevTools Protocol.
// It connects to a remote browser via CDP WebSocket and supports navigation,
// clicking, typing, screenshots, text extraction, JS evaluation, and waiting.
type BrowserTool struct {
	cdpURL        string
	token         string
	stealth       bool
	launchTimeout time.Duration
	actionTimeout time.Duration

	allocCtx      context.Context
	allocCancel   context.CancelFunc
	browserCtx    context.Context
	browserCancel context.CancelFunc
	connected     bool
	mu            sync.Mutex
}

// NewBrowserTool creates a new BrowserTool with the given options.
func NewBrowserTool(opts BrowserToolOptions) *BrowserTool {
	launchTimeout := 120 * time.Second
	if opts.LaunchTimeout > 0 {
		launchTimeout = time.Duration(opts.LaunchTimeout) * time.Millisecond
	}

	actionTimeout := 30 * time.Second
	if opts.ActionTimeout > 0 {
		actionTimeout = time.Duration(opts.ActionTimeout) * time.Millisecond
	}

	return &BrowserTool{
		cdpURL:        opts.CdpURL,
		token:         opts.Token,
		stealth:       opts.Stealth,
		launchTimeout: launchTimeout,
		actionTimeout: actionTimeout,
	}
}

func (t *BrowserTool) Name() string {
	return "browser"
}

func (t *BrowserTool) Description() string {
	return "Control a browser via CDP (Chrome DevTools Protocol). " +
		"Actions: navigate, click, type, screenshot, get_text, evaluate, wait, scroll, hover, select, pdf, cookies, close. " +
		"The browser session persists across calls until explicitly closed."
}

func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"description": "Action to perform: " +
					"navigate (open URL), click (click element), type (type text into input), " +
					"screenshot (capture page/element), get_text (extract text), evaluate (run JS), " +
					"wait (wait for element), scroll (scroll page/element), hover (mouse over element), " +
					"select (choose dropdown option), pdf (save page as PDF), " +
					"cookies (get/set/delete cookies), close (end session)",
				"enum": []string{
					"navigate", "click", "type", "screenshot", "get_text", "evaluate",
					"wait", "scroll", "hover", "select", "pdf", "cookies", "close",
				},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to (for 'navigate' action)",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for target element (for click, type, get_text, screenshot, wait, scroll, hover, select)",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to type (for 'type' action)",
			},
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript expression to evaluate (for 'evaluate' action)",
			},
			"timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in milliseconds (for 'wait' action, default: 30000)",
			},
			"value": map[string]interface{}{
				"type":        "string",
				"description": "Value to select (for 'select' action — option value attribute)",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "Scroll direction: 'up' or 'down' (for 'scroll' action, default: 'down')",
				"enum":        []string{"up", "down"},
			},
			"distance": map[string]interface{}{
				"type":        "integer",
				"description": "Scroll distance in pixels (for 'scroll' action, default: 500)",
			},
			"wait_for_navigation": map[string]interface{}{
				"type":        "boolean",
				"description": "Wait for page navigation after click (for 'click' action, default: false)",
			},
			"cookie_action": map[string]interface{}{
				"type":        "string",
				"description": "Cookie sub-action: 'get' (list all), 'set' (add cookie), 'delete' (remove cookie), 'clear' (remove all)",
				"enum":        []string{"get", "set", "delete", "clear"},
			},
			"cookie_name": map[string]interface{}{
				"type":        "string",
				"description": "Cookie name (for cookies set/delete)",
			},
			"cookie_value": map[string]interface{}{
				"type":        "string",
				"description": "Cookie value (for cookies set)",
			},
			"cookie_domain": map[string]interface{}{
				"type":        "string",
				"description": "Cookie domain (for cookies set/delete)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	if action == "close" {
		return t.doClose()
	}

	if err := t.ensureConnected(); err != nil {
		return ErrorResult(fmt.Sprintf("failed to connect to browser: %v", err))
	}

	switch action {
	case "navigate":
		return t.doNavigate(ctx, args)
	case "click":
		return t.doClick(ctx, args)
	case "type":
		return t.doType(ctx, args)
	case "screenshot":
		return t.doScreenshot(ctx, args)
	case "get_text":
		return t.doGetText(ctx, args)
	case "evaluate":
		return t.doEvaluate(ctx, args)
	case "wait":
		return t.doWait(ctx, args)
	case "scroll":
		return t.doScroll(ctx, args)
	case "hover":
		return t.doHover(ctx, args)
	case "select":
		return t.doSelect(ctx, args)
	case "pdf":
		return t.doPDF(ctx, args)
	case "cookies":
		return t.doCookies(ctx, args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

// buildCdpURL constructs the WebSocket URL with token, stealth, and launch timeout params.
// The cdp_url config should point to the CDP WebSocket endpoint as-is.
func (t *BrowserTool) buildCdpURL() string {
	u, err := url.Parse(t.cdpURL)
	if err != nil {
		return t.cdpURL
	}

	q := u.Query()
	if t.token != "" {
		q.Set("token", t.token)
	}
	if t.stealth {
		q.Set("stealth", "true")
	}
	q.Set("launch", fmt.Sprintf(`{"timeout":%d}`, int(t.launchTimeout.Milliseconds())))
	u.RawQuery = q.Encode()

	return u.String()
}

// ensureConnected lazily connects to the remote browser, reusing the connection.
func (t *BrowserTool) ensureConnected() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected && t.browserCtx != nil {
		if t.browserCtx.Err() == nil {
			return nil
		}
		logger.InfoCF("browser", "Browser context expired, reconnecting", nil)
		t.cleanupLocked()
	}

	wsURL := t.buildCdpURL()
	logger.InfoCF("browser", "Connecting to remote browser",
		map[string]interface{}{
			"url": t.cdpURL,
		})

	// NoModifyURL prevents chromedp from fetching /json/version which returns
	// internal Chrome URLs inaccessible outside the container.
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL, chromedp.NoModifyURL)

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	// Establish the connection with a timeout.
	// IMPORTANT: don't use context.WithTimeout(browserCtx, ...) because chromedp's
	// RemoteAllocator closes the WebSocket when the allocate context is done.
	// Instead, run in a goroutine with a channel-based timeout.
	done := make(chan error, 1)
	go func() {
		done <- chromedp.Run(browserCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			browserCancel()
			allocCancel()
			return fmt.Errorf("browser connection failed: %w", err)
		}
	case <-time.After(t.launchTimeout):
		browserCancel()
		allocCancel()
		return fmt.Errorf("browser connection timed out after %s", t.launchTimeout)
	}

	t.allocCtx = allocCtx
	t.allocCancel = allocCancel
	t.browserCtx = browserCtx
	t.browserCancel = browserCancel
	t.connected = true
	logger.InfoCF("browser", "Connected to remote browser", nil)
	return nil
}

// cleanupLocked releases browser resources. Must be called with mu held.
func (t *BrowserTool) cleanupLocked() {
	if t.browserCancel != nil {
		t.browserCancel()
		t.browserCancel = nil
	}
	if t.allocCancel != nil {
		t.allocCancel()
		t.allocCancel = nil
	}
	t.browserCtx = nil
	t.allocCtx = nil
	t.connected = false
}

// runAction executes chromedp actions with the configured action timeout.
func (t *BrowserTool) runAction(ctx context.Context, actions ...chromedp.Action) error {
	t.mu.Lock()
	browserCtx := t.browserCtx
	t.mu.Unlock()

	if browserCtx == nil {
		return fmt.Errorf("browser not connected")
	}

	actionCtx, cancel := context.WithTimeout(browserCtx, t.actionTimeout)
	defer cancel()

	return chromedp.Run(actionCtx, actions...)
}

// getBrowserCtx returns the current browser context safely.
func (t *BrowserTool) getBrowserCtx() context.Context {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.browserCtx
}

func (t *BrowserTool) doNavigate(ctx context.Context, args map[string]interface{}) *ToolResult {
	navURL, ok := args["url"].(string)
	if !ok || navURL == "" {
		return ErrorResult("url is required for navigate action")
	}

	var title string
	var location string

	err := t.runAction(ctx,
		chromedp.Navigate(navURL),
		chromedp.WaitReady("body"),
		chromedp.Title(&title),
		chromedp.Location(&location),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("navigate failed: %v", err))
	}

	result := fmt.Sprintf("Navigated to: %s\nTitle: %s\nURL: %s", navURL, title, location)
	return SilentResult(result)
}

func (t *BrowserTool) doClick(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return ErrorResult("selector is required for click action")
	}

	waitNav, _ := args["wait_for_navigation"].(bool)

	if waitNav {
		// Click and wait for navigation to complete
		browserCtx := t.getBrowserCtx()
		if browserCtx == nil {
			return ErrorResult("browser not connected")
		}

		actionCtx, cancel := context.WithTimeout(browserCtx, t.actionTimeout)
		defer cancel()

		err := chromedp.Run(actionCtx,
			chromedp.WaitVisible(selector),
			chromedp.Click(selector),
			chromedp.WaitReady("body"),
		)
		if err != nil {
			return ErrorResult(fmt.Sprintf("click failed on '%s': %v", selector, err))
		}

		var title, location string
		_ = chromedp.Run(actionCtx,
			chromedp.Title(&title),
			chromedp.Location(&location),
		)
		return SilentResult(fmt.Sprintf("Clicked element: %s\nNavigated to: %s\nTitle: %s", selector, location, title))
	}

	err := t.runAction(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Click(selector),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("click failed on '%s': %v", selector, err))
	}

	return SilentResult(fmt.Sprintf("Clicked element: %s", selector))
}

func (t *BrowserTool) doType(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return ErrorResult("selector is required for type action")
	}

	text, ok := args["text"].(string)
	if !ok {
		return ErrorResult("text is required for type action")
	}

	err := t.runAction(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Clear(selector),
		chromedp.SendKeys(selector, text),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("type failed on '%s': %v", selector, err))
	}

	return SilentResult(fmt.Sprintf("Typed %d chars into: %s", len(text), selector))
}

func (t *BrowserTool) doScreenshot(ctx context.Context, args map[string]interface{}) *ToolResult {
	var buf []byte
	var err error

	selector, _ := args["selector"].(string)

	if selector != "" {
		err = t.runAction(ctx,
			chromedp.WaitVisible(selector),
			chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
		)
	} else {
		err = t.runAction(ctx,
			chromedp.FullScreenshot(&buf, 90),
		)
	}

	if err != nil {
		return ErrorResult(fmt.Sprintf("screenshot failed: %v", err))
	}

	encoded := base64.StdEncoding.EncodeToString(buf)

	llmContent := fmt.Sprintf("Screenshot taken (%d bytes PNG, %d chars base64)", len(buf), len(encoded))
	if len(encoded) > 500 {
		llmContent += fmt.Sprintf("\nBase64 preview: %s...", encoded[:500])
	} else {
		llmContent += fmt.Sprintf("\nBase64: %s", encoded)
	}

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: fmt.Sprintf("data:image/png;base64,%s", encoded),
		Silent:  false,
	}
}

func (t *BrowserTool) doGetText(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, _ := args["selector"].(string)
	if selector == "" {
		selector = "body"
	}

	var text string
	err := t.runAction(ctx,
		chromedp.WaitReady(selector),
		chromedp.Text(selector, &text),
	)

	if err != nil {
		return ErrorResult(fmt.Sprintf("get_text failed: %v", err))
	}

	text = strings.TrimSpace(text)

	const maxTextLen = 10000
	truncated := false
	if len(text) > maxTextLen {
		text = text[:maxTextLen]
		truncated = true
	}

	if truncated {
		text += "\n\n[... truncated at 10000 chars]"
	}

	return SilentResult(text)
}

func (t *BrowserTool) doEvaluate(ctx context.Context, args map[string]interface{}) *ToolResult {
	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return ErrorResult("expression is required for evaluate action")
	}

	var result interface{}
	err := t.runAction(ctx,
		chromedp.Evaluate(expression, &result),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("evaluate failed: %v", err))
	}

	resultStr := fmt.Sprintf("%v", result)

	const maxResultLen = 10000
	if len(resultStr) > maxResultLen {
		resultStr = resultStr[:maxResultLen] + "\n[... truncated]"
	}

	return SilentResult(fmt.Sprintf("JS eval result: %s", resultStr))
}

func (t *BrowserTool) doWait(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return ErrorResult("selector is required for wait action")
	}

	timeout := t.actionTimeout
	if ms, ok := args["timeout_ms"].(float64); ok && ms > 0 {
		timeout = time.Duration(ms) * time.Millisecond
	}

	browserCtx := t.getBrowserCtx()
	if browserCtx == nil {
		return ErrorResult("browser not connected")
	}

	waitCtx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()

	err := chromedp.Run(waitCtx,
		chromedp.WaitVisible(selector),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("wait timed out for '%s': %v", selector, err))
	}

	return SilentResult(fmt.Sprintf("Element visible: %s", selector))
}

func (t *BrowserTool) doScroll(ctx context.Context, args map[string]interface{}) *ToolResult {
	direction := "down"
	if d, ok := args["direction"].(string); ok && d != "" {
		direction = d
	}

	distance := 500
	if d, ok := args["distance"].(float64); ok && d > 0 {
		distance = int(d)
	}

	if direction == "up" {
		distance = -distance
	}

	selector, _ := args["selector"].(string)

	var js string
	if selector != "" {
		js = fmt.Sprintf(`document.querySelector(%q).scrollBy(0, %d)`, selector, distance)
	} else {
		js = fmt.Sprintf(`window.scrollBy(0, %d)`, distance)
	}

	err := t.runAction(ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return ErrorResult(fmt.Sprintf("scroll failed: %v", err))
	}

	target := "page"
	if selector != "" {
		target = selector
	}
	return SilentResult(fmt.Sprintf("Scrolled %s %s by %dpx", target, direction, abs(distance)))
}

func (t *BrowserTool) doHover(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return ErrorResult("selector is required for hover action")
	}

	err := t.runAction(ctx,
		chromedp.WaitVisible(selector),
		chromedp.MouseClickXY(0, 0, chromedp.ButtonNone), // reset position
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Get element position and hover over it
			var nodes []*cdp.Node
			if err := chromedp.Nodes(selector, &nodes, chromedp.AtLeast(1)).Do(ctx); err != nil {
				return err
			}
			return chromedp.MouseClickNode(nodes[0], chromedp.ButtonNone).Do(ctx)
		}),
	)
	if err != nil {
		// Fallback: use JS-based hover via dispatchEvent
		jsHover := fmt.Sprintf(`
			var el = document.querySelector(%q);
			if (el) {
				el.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
				el.dispatchEvent(new MouseEvent('mouseenter', {bubbles: true}));
				'hovered'
			} else {
				'element not found'
			}
		`, selector)
		var result string
		err2 := t.runAction(ctx, chromedp.Evaluate(jsHover, &result))
		if err2 != nil {
			return ErrorResult(fmt.Sprintf("hover failed on '%s': %v", selector, err))
		}
		if result == "element not found" {
			return ErrorResult(fmt.Sprintf("hover failed: element '%s' not found", selector))
		}
	}

	return SilentResult(fmt.Sprintf("Hovered over: %s", selector))
}

func (t *BrowserTool) doSelect(ctx context.Context, args map[string]interface{}) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return ErrorResult("selector is required for select action")
	}

	value, ok := args["value"].(string)
	if !ok || value == "" {
		return ErrorResult("value is required for select action")
	}

	err := t.runAction(ctx,
		chromedp.WaitVisible(selector),
		chromedp.SetValue(selector, value),
		// Trigger change event so JS frameworks pick up the selection
		chromedp.Evaluate(fmt.Sprintf(
			`var el = document.querySelector(%q); if(el) el.dispatchEvent(new Event('change', {bubbles:true}))`,
			selector,
		), nil),
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("select failed on '%s': %v", selector, err))
	}

	return SilentResult(fmt.Sprintf("Selected value '%s' in: %s", value, selector))
}

func (t *BrowserTool) doPDF(ctx context.Context, args map[string]interface{}) *ToolResult {
	browserCtx := t.getBrowserCtx()
	if browserCtx == nil {
		return ErrorResult("browser not connected")
	}

	actionCtx, cancel := context.WithTimeout(browserCtx, t.actionTimeout)
	defer cancel()

	var buf []byte
	err := chromedp.Run(actionCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = page.PrintToPDF().
			WithPrintBackground(true).
			WithPreferCSSPageSize(true).
			Do(ctx)
		return err
	}))
	if err != nil {
		return ErrorResult(fmt.Sprintf("pdf failed: %v", err))
	}

	encoded := base64.StdEncoding.EncodeToString(buf)

	llmContent := fmt.Sprintf("PDF generated (%d bytes, %d chars base64)", len(buf), len(encoded))

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: fmt.Sprintf("data:application/pdf;base64,%s", encoded),
		Silent:  false,
	}
}

func (t *BrowserTool) doCookies(ctx context.Context, args map[string]interface{}) *ToolResult {
	cookieAction, _ := args["cookie_action"].(string)
	if cookieAction == "" {
		cookieAction = "get"
	}

	browserCtx := t.getBrowserCtx()
	if browserCtx == nil {
		return ErrorResult("browser not connected")
	}

	actionCtx, cancel := context.WithTimeout(browserCtx, t.actionTimeout)
	defer cancel()

	switch cookieAction {
	case "get":
		var cookies []*network.Cookie
		err := chromedp.Run(actionCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}))
		if err != nil {
			return ErrorResult(fmt.Sprintf("get cookies failed: %v", err))
		}

		type cookieSummary struct {
			Name     string `json:"name"`
			Value    string `json:"value"`
			Domain   string `json:"domain"`
			Path     string `json:"path"`
			Secure   bool   `json:"secure"`
			HTTPOnly bool   `json:"httpOnly"`
		}

		summaries := make([]cookieSummary, 0, len(cookies))
		for _, c := range cookies {
			summaries = append(summaries, cookieSummary{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Secure:   c.Secure,
				HTTPOnly: c.HTTPOnly,
			})
		}

		data, _ := json.MarshalIndent(summaries, "", "  ")
		return SilentResult(fmt.Sprintf("Cookies (%d):\n%s", len(cookies), string(data)))

	case "set":
		name, _ := args["cookie_name"].(string)
		value, _ := args["cookie_value"].(string)
		domain, _ := args["cookie_domain"].(string)

		if name == "" {
			return ErrorResult("cookie_name is required for cookies set")
		}

		err := chromedp.Run(actionCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			cp := &network.CookieParam{
				Name:  name,
				Value: value,
			}
			if domain != "" {
				cp.Domain = domain
			}
			return network.SetCookies([]*network.CookieParam{cp}).Do(ctx)
		}))
		if err != nil {
			return ErrorResult(fmt.Sprintf("set cookie failed: %v", err))
		}
		return SilentResult(fmt.Sprintf("Cookie set: %s=%s", name, value))

	case "delete":
		name, _ := args["cookie_name"].(string)
		domain, _ := args["cookie_domain"].(string)

		if name == "" {
			return ErrorResult("cookie_name is required for cookies delete")
		}

		err := chromedp.Run(actionCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			dp := &network.DeleteCookiesParams{Name: name}
			if domain != "" {
				dp.Domain = domain
			}
			return dp.Do(ctx)
		}))
		if err != nil {
			return ErrorResult(fmt.Sprintf("delete cookie failed: %v", err))
		}
		return SilentResult(fmt.Sprintf("Cookie deleted: %s", name))

	case "clear":
		err := chromedp.Run(actionCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			return network.ClearBrowserCookies().Do(ctx)
		}))
		if err != nil {
			return ErrorResult(fmt.Sprintf("clear cookies failed: %v", err))
		}
		return SilentResult("All cookies cleared")

	default:
		return ErrorResult(fmt.Sprintf("unknown cookie_action: %s (use get/set/delete/clear)", cookieAction))
	}
}

func (t *BrowserTool) doClose() *ToolResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return SilentResult("Browser already closed")
	}

	t.cleanupLocked()
	logger.InfoCF("browser", "Browser session closed", nil)
	return SilentResult("Browser session closed")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

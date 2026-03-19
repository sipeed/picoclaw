package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"jane/pkg/logger"
)

type BrowserActionTool struct {
	mu      sync.Mutex
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	page    playwright.Page
}

func NewBrowserActionTool() *BrowserActionTool {
	return &BrowserActionTool{}
}

func (t *BrowserActionTool) Name() string {
	return "browser_action"
}

func (t *BrowserActionTool) Description() string {
	return "Interact with a web browser. Actions: 'navigate' (requires url), 'click' (requires selector), 'type' (requires selector, text), 'extract' (returns page text), 'screenshot' (returns base64 image), 'wait' (requires selector). Use this for complex web tasks that require JavaScript rendering or interaction."
}

func (t *BrowserActionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform: 'navigate', 'click', 'type', 'extract', 'screenshot', 'wait'",
				"enum":        []string{"navigate", "click", "type", "extract", "screenshot", "wait"},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL to navigate to (used with 'navigate' action)",
			},
			"selector": map[string]any{
				"type":        "string",
				"description": "CSS selector or XPath for the element to interact with (used with 'click', 'type', 'wait' actions)",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text to type into the element (used with 'type' action)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserActionTool) ensureBrowser() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pw != nil && t.browser != nil && t.context != nil && t.page != nil {
		return nil
	}

	err := playwright.Install()
	if err != nil {
		logger.WarnCF("tool", "Playwright install warning/error", map[string]any{"error": err.Error()})
		// Continue even if install returns an error, as it might already be installed
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	t.pw = pw

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	t.browser = browser

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not create context: %w", err)
	}
	t.context = context

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}
	t.page = page

	return nil
}

func (t *BrowserActionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	if err := t.ensureBrowser(); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to initialize browser: %v", err))
	}

	// We lock around the actual playwright interactions to avoid concurrent page mutations
	t.mu.Lock()
	defer t.mu.Unlock()

	switch action {
	case "navigate":
		urlStr, ok := args["url"].(string)
		if !ok || urlStr == "" {
			return ErrorResult("url is required for navigate action")
		}

		// If the URL doesn't have a scheme, prepend https://
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			urlStr = "https://" + urlStr
		}

		if _, err := t.page.Goto(urlStr, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(30000), // 30s timeout
		}); err != nil {
			return ErrorResult(fmt.Sprintf("Failed to navigate to %s: %v", urlStr, err))
		}

		title, _ := t.page.Title()
		return SilentResult(fmt.Sprintf("Navigated to %s. Page title: %s", urlStr, title))

	case "click":
		selector, ok := args["selector"].(string)
		if !ok || selector == "" {
			return ErrorResult("selector is required for click action")
		}
		if err := t.page.Click(selector, playwright.PageClickOptions{
			Timeout: playwright.Float(10000),
		}); err != nil {
			return ErrorResult(fmt.Sprintf("Failed to click %s: %v", selector, err))
		}

		// Wait a bit for navigation or changes after click
		time.Sleep(1 * time.Second)
		url := t.page.URL()
		return SilentResult(fmt.Sprintf("Clicked element %s. Current URL: %s", selector, url))

	case "type":
		selector, ok := args["selector"].(string)
		if !ok || selector == "" {
			return ErrorResult("selector is required for type action")
		}
		text, ok := args["text"].(string)
		if !ok {
			return ErrorResult("text is required for type action")
		}
		if err := t.page.Fill(selector, text, playwright.PageFillOptions{
			Timeout: playwright.Float(10000),
		}); err != nil {
			return ErrorResult(fmt.Sprintf("Failed to type into %s: %v", selector, err))
		}
		return SilentResult(fmt.Sprintf("Typed text into %s", selector))

	case "wait":
		selector, ok := args["selector"].(string)
		if !ok || selector == "" {
			return ErrorResult("selector is required for wait action")
		}
		_, err := t.page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(15000),
		})
		if err != nil {
			return ErrorResult(fmt.Sprintf("Timed out waiting for selector %s: %v", selector, err))
		}
		return SilentResult(fmt.Sprintf("Selector %s is now visible", selector))

	case "extract":
		text, err := t.page.Evaluate(`() => {
			// Basic text extraction, preferring readable content over scripts/styles
			const extractText = (node) => {
				if (node.nodeType === Node.TEXT_NODE) {
					return node.textContent;
				}
				if (node.nodeType !== Node.ELEMENT_NODE) {
					return '';
				}
				const tag = node.tagName.toLowerCase();
				if (tag === 'script' || tag === 'style' || tag === 'noscript') {
					return '';
				}
				let text = '';
				for (const child of node.childNodes) {
					text += extractText(child);
				}
				return text;
			};
			return extractText(document.body).replace(/\s+/g, ' ').trim();
		}`)
		if err != nil {
			return ErrorResult(fmt.Sprintf("Failed to extract text: %v", err))
		}

		extractedText, ok := text.(string)
		if !ok {
			return ErrorResult("Failed to parse extracted text")
		}

		// Truncate if too long (similar to web_fetch)
		maxChars := 10000
		if len(extractedText) > maxChars {
			extractedText = extractedText[:maxChars] + fmt.Sprintf("\n... (truncated, %d more chars)", len(extractedText)-maxChars)
		}

		url := t.page.URL()
		title, _ := t.page.Title()
		return &ToolResult{
			ForLLM:  fmt.Sprintf("URL: %s\nTitle: %s\n\nContent:\n%s", url, title, extractedText),
			ForUser: fmt.Sprintf("Extracted %d chars from %s", len(extractedText), url),
		}

	case "screenshot":
		return ErrorResult("Screenshot action is not fully implemented for this environment yet (requires media handling).")

	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}

// Close gracefully shuts down the browser instance
func (t *BrowserActionTool) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.browser != nil {
		t.browser.Close()
		t.browser = nil
	}
	if t.pw != nil {
		t.pw.Stop()
		t.pw = nil
	}
}

func (t *BrowserActionTool) RequiresApproval() bool {
	return false
}

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// actionbookBinary caches the resolved path to the actionbook CLI.
var (
	actionbookBinary string
	actionbookOnce   sync.Once
	actionbookErr    error
)

func resolveActionbook() (string, error) {
	actionbookOnce.Do(func() {
		var path string
		path, actionbookErr = exec.LookPath("actionbook")
		if actionbookErr != nil {
			actionbookErr = fmt.Errorf("actionbook CLI not found in PATH: %w", actionbookErr)
			return
		}
		actionbookBinary = path
	})
	if actionbookErr != nil {
		return "", actionbookErr
	}
	return actionbookBinary, nil
}

// runActionbook executes an actionbook CLI command with a timeout.
func runActionbook(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	bin, err := resolveActionbook()
	if err != nil {
		return "", err
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, bin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("actionbook command timed out after %v", timeout)
		}
		// Many actionbook commands write useful output even on non-zero exit.
		// Return stdout+stderr as output with the error.
		out := stdout.String()
		if stderr.Len() > 0 {
			out += "\nSTDERR: " + stderr.String()
		}
		if out != "" {
			return out, nil // treat as non-fatal; LLM can interpret
		}
		return "", fmt.Errorf("actionbook failed: %w — %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// truncateOutput limits output length to avoid blowing up context.
func truncateOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... (truncated, %d more chars)", len(s)-max)
}

// ---------------------------------------------------------------------------
// BrowserSearchTool — actionbook search
// ---------------------------------------------------------------------------

// BrowserSearchTool wraps `actionbook search` to find action manuals.
type BrowserSearchTool struct {
}

func NewBrowserSearchTool() *BrowserSearchTool {
	return &BrowserSearchTool{}
}

func (t *BrowserSearchTool) Name() string { return "browser_search" }
func (t *BrowserSearchTool) Description() string {
	return "Search ActionBook for browser action manuals matching a task. Returns action IDs, descriptions, URLs, and health scores. Use the returned ID with browser_get to retrieve selectors."
}

func (t *BrowserSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query describing the task (e.g. 'airbnb search listings Tokyo')",
			},
			"domain": map[string]interface{}{
				"type":        "string",
				"description": "Optional domain to filter results (e.g. 'airbnb.com')",
			},
		},
		"required": []string{"query"},
	}
}

func (t *BrowserSearchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return ErrorResult("query is required")
	}

	cmdArgs := []string{"search", query, "--json"}
	if domain, ok := args["domain"].(string); ok && domain != "" {
		cmdArgs = append(cmdArgs, "--domain", domain)
	}

	output, err := runActionbook(ctx, 30*time.Second, cmdArgs...)
	if err != nil {
		return ErrorResult(fmt.Sprintf("browser_search failed: %v", err))
	}

	output = truncateOutput(output, 10000)
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

// ---------------------------------------------------------------------------
// BrowserGetTool — actionbook get
// ---------------------------------------------------------------------------

// BrowserGetTool wraps `actionbook get` to retrieve action details.
type BrowserGetTool struct{}

func NewBrowserGetTool() *BrowserGetTool { return &BrowserGetTool{} }

func (t *BrowserGetTool) Name() string { return "browser_get" }
func (t *BrowserGetTool) Description() string {
	return "Retrieve a specific ActionBook action manual by its ID. Returns page structure with CSS selectors, element types, and allowed methods. Use the ID from browser_search results."
}

func (t *BrowserGetTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action_id": map[string]interface{}{
				"type":        "string",
				"description": "Action manual ID from browser_search (e.g. 'airbnb.com:/:default')",
			},
		},
		"required": []string{"action_id"},
	}
}

func (t *BrowserGetTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	actionID, ok := args["action_id"].(string)
	if !ok || actionID == "" {
		return ErrorResult("action_id is required")
	}

	output, err := runActionbook(ctx, 30*time.Second, "get", actionID, "--json")
	if err != nil {
		return ErrorResult(fmt.Sprintf("browser_get failed: %v", err))
	}

	output = truncateOutput(output, 15000)
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

// ---------------------------------------------------------------------------
// BrowserTool — actionbook browser *
// ---------------------------------------------------------------------------

// allowedBrowserActions is the set of browser subcommands the LLM may invoke.
var allowedBrowserActions = map[string]bool{
	"open":       true,
	"goto":       true,
	"click":      true,
	"fill":       true,
	"type":       true,
	"select":     true,
	"hover":      true,
	"focus":      true,
	"press":      true,
	"text":       true,
	"snapshot":   true,
	"screenshot": true,
	"wait":       true,
	"wait-nav":   true,
	"back":       true,
	"forward":    true,
	"reload":     true,
	"close":      true,
	"pages":      true,
	"switch":     true,
	"eval":       true,
	"html":       true,
	"pdf":        true,
	"cookies":    true,
	"status":     true,
	"viewport":   true,
	"scroll":     true,
}

// BrowserTool wraps `actionbook browser` for browser automation.
type BrowserTool struct {
	headless bool
}

func NewBrowserTool(headless bool) *BrowserTool {
	return &BrowserTool{headless: headless}
}

func (t *BrowserTool) Name() string { return "browser" }
func (t *BrowserTool) Description() string {
	return `Execute browser automation commands via ActionBook. Supported actions: open, goto, click, fill, type, select, hover, focus, press, text, scroll, snapshot, screenshot, wait, wait-nav, back, forward, reload, close, pages, switch, eval, html, pdf, cookies, status, viewport. Typical workflow: browser_search → browser_get → browser (open → interact → close).`
}

func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Browser action to perform (e.g. 'open', 'click', 'fill', 'text', 'snapshot', 'close')",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL for open/goto actions",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for click/fill/type/select/hover/focus/wait/text/html actions",
			},
			"value": map[string]interface{}{
				"type":        "string",
				"description": "Value for fill/type/select/press/eval/switch/scroll actions",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in milliseconds for wait/wait-nav actions (default: 30000)",
			},
			"full_page": map[string]interface{}{
				"type":        "boolean",
				"description": "For screenshot: capture full page instead of viewport",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return ErrorResult("action is required")
	}

	action = strings.TrimSpace(strings.ToLower(action))
	if !allowedBrowserActions[action] {
		return ErrorResult(fmt.Sprintf("unknown browser action %q — allowed: open, goto, click, fill, type, select, hover, focus, press, text, scroll, snapshot, screenshot, wait, wait-nav, back, forward, reload, close, pages, switch, eval, html, pdf, cookies, status, viewport", action))
	}

	// Build argument list
	cmdArgs := []string{}

	// Global flags
	if t.headless {
		cmdArgs = append(cmdArgs, "--headless")
	}

	cmdArgs = append(cmdArgs, "browser", action)

	switch action {
	case "open", "goto":
		if u, ok := args["url"].(string); ok && u != "" {
			cmdArgs = append(cmdArgs, u)
		} else {
			return ErrorResult(fmt.Sprintf("url is required for %s action", action))
		}
		if timeout, ok := getIntArg(args, "timeout"); ok {
			cmdArgs = append(cmdArgs, "--timeout", fmt.Sprintf("%d", timeout))
		}

	case "click", "hover", "focus":
		sel, ok := args["selector"].(string)
		if !ok || sel == "" {
			return ErrorResult(fmt.Sprintf("selector is required for %s action", action))
		}
		cmdArgs = append(cmdArgs, sel)
		if timeout, ok := getIntArg(args, "timeout"); ok {
			cmdArgs = append(cmdArgs, "--wait", fmt.Sprintf("%d", timeout))
		}

	case "fill", "type":
		sel, ok := args["selector"].(string)
		if !ok || sel == "" {
			return ErrorResult(fmt.Sprintf("selector is required for %s action", action))
		}
		val, ok := args["value"].(string)
		if !ok {
			return ErrorResult(fmt.Sprintf("value is required for %s action", action))
		}
		cmdArgs = append(cmdArgs, sel, val)
		if timeout, ok := getIntArg(args, "timeout"); ok {
			cmdArgs = append(cmdArgs, "--wait", fmt.Sprintf("%d", timeout))
		}

	case "select":
		sel, ok := args["selector"].(string)
		if !ok || sel == "" {
			return ErrorResult("selector is required for select action")
		}
		val, ok := args["value"].(string)
		if !ok || val == "" {
			return ErrorResult("value is required for select action")
		}
		cmdArgs = append(cmdArgs, sel, val)

	case "press":
		key, ok := args["value"].(string)
		if !ok || key == "" {
			return ErrorResult("value is required for press action (e.g. 'Enter', 'Tab')")
		}
		cmdArgs = append(cmdArgs, key)

	case "scroll":
		val, ok := args["value"].(string)
		if !ok || val == "" {
			return ErrorResult("value is required for scroll action (e.g. 'down', 'up', 'bottom', 'top', or pixels)")
		}
		cmdArgs = append(cmdArgs, val)

	case "wait":
		sel, ok := args["selector"].(string)
		if !ok || sel == "" {
			return ErrorResult("selector is required for wait action")
		}
		cmdArgs = append(cmdArgs, sel)
		if timeout, ok := getIntArg(args, "timeout"); ok {
			cmdArgs = append(cmdArgs, "--timeout", fmt.Sprintf("%d", timeout))
		}

	case "wait-nav":
		if timeout, ok := getIntArg(args, "timeout"); ok {
			cmdArgs = append(cmdArgs, "--timeout", fmt.Sprintf("%d", timeout))
		}

	case "switch":
		pageID, ok := args["value"].(string)
		if !ok || pageID == "" {
			return ErrorResult("value (page_id) is required for switch action")
		}
		cmdArgs = append(cmdArgs, pageID)

	case "eval":
		script, ok := args["value"].(string)
		if !ok || script == "" {
			return ErrorResult("value (javascript expression) is required for eval action")
		}
		cmdArgs = append(cmdArgs, script)

	case "text", "html":
		if sel, ok := args["selector"].(string); ok && sel != "" {
			cmdArgs = append(cmdArgs, sel)
		}

	case "screenshot":
		if fullPage, ok := args["full_page"].(bool); ok && fullPage {
			cmdArgs = append(cmdArgs, "--full-page")
		}

	case "cookies":
		// cookies sub-actions via value: list, get <name>, set <name> <val>, delete <name>, clear
		if val, ok := args["value"].(string); ok && val != "" {
			cmdArgs = append(cmdArgs, strings.Fields(val)...)
		}

		// For actions with no extra args: back, forward, reload, close, pages, snapshot, status, viewport, pdf
		// — nothing extra needed, cmdArgs already has "browser" and action.
	}

	// Determine timeout
	cmdTimeout := 30 * time.Second
	if action == "screenshot" || action == "pdf" {
		cmdTimeout = 60 * time.Second
	}

	output, err := runActionbook(ctx, cmdTimeout, cmdArgs...)
	if err != nil {
		return ErrorResult(fmt.Sprintf("browser %s failed: %v", action, err))
	}

	if output == "" {
		output = "(no output)"
	}

	output = truncateOutput(output, 15000)
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

// getIntArg extracts an integer argument from the args map.
// JSON numbers arrive as float64.
func getIntArg(args map[string]interface{}, key string) (int, bool) {
	if v, ok := args[key].(float64); ok {
		return int(v), true
	}
	return 0, false
}

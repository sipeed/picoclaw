package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BrowserToolOptions configures the BrowserTool.
type BrowserToolOptions struct {
	Session  string // Session name for isolation
	Headless bool   // Run in headless mode (default true)
	Timeout  int    // Command timeout in seconds (default 30)
	CDPPort  int    // Chrome DevTools Protocol port (default 9222)
}

// BrowserTool wraps the agent-browser CLI for headless browser automation.
// It delegates all browser complexity to the external `agent-browser` binary.
type BrowserTool struct {
	session  string
	headless bool
	timeout  time.Duration
	cdpPort  int
}

// NewBrowserTool creates a new BrowserTool with the given options.
func NewBrowserTool(opts BrowserToolOptions) *BrowserTool {
	timeout := 30
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	cdpPort := 9222
	if opts.CDPPort > 0 {
		cdpPort = opts.CDPPort
	}
	return &BrowserTool{
		session:  opts.Session,
		headless: opts.Headless,
		timeout:  time.Duration(timeout) * time.Second,
		cdpPort:  cdpPort,
	}
}

func (t *BrowserTool) Name() string {
	return "browser"
}

func (t *BrowserTool) Description() string {
	return `Automate a headless browser via agent-browser CLI. Pass the subcommand as 'command'.
The browser daemon persists between calls — open a page first, then interact with it.

Core workflow:
  browser open <url>           → Navigate to URL
  browser snapshot -i          → Get interactive elements with refs (@e1, @e2, ...)
  browser click @e2            → Click element by ref
  browser fill @e3 "text"      → Fill input by ref
  browser type @e3 "text"      → Type into element
  browser press Enter          → Press a key
  browser screenshot [path]    → Take screenshot
  browser get text @e1         → Get text content of element
  browser get title            → Get page title
  browser get url              → Get current URL
  browser eval "js code"       → Run JavaScript
  browser scroll down [px]     → Scroll page
  browser wait <selector|ms>   → Wait for element or time
  browser close                → Close browser

CSS selectors also work: browser click "#submit"

Examples:
  command: "open https://example.com"
  command: "snapshot -i"
  command: "click @e2"
  command: "fill @e3 \"user@example.com\""
  command: "get title"
  command: "screenshot /tmp/page.png"
  command: "close"`
}

func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The agent-browser subcommand to execute (e.g. 'open https://example.com', 'snapshot -i', 'click @e2')",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	command, ok := args["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return ErrorResult("command is required (e.g. 'open https://example.com')")
	}

	// Build the full agent-browser command line
	cmdArgs := t.buildArgs(command)

	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "agent-browser", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		errOut := stderr.String()
		// Filter out noise from stderr (daemon startup messages, etc.)
		if !strings.Contains(errOut, "Daemon started") {
			if output != "" {
				output += "\n"
			}
			output += errOut
		}
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("Browser command timed out after %v: %s", t.timeout, command)
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
			}
		}
		// Include output even on error — agent-browser often puts useful info in stdout
		if output == "" {
			output = fmt.Sprintf("command failed: %v", err)
		} else {
			output += fmt.Sprintf("\nExit code: %v", err)
		}
	}

	if output == "" {
		output = "(no output)"
	}

	// Truncate long output
	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	if err != nil {
		return &ToolResult{
			ForLLM:  output,
			ForUser: output,
			IsError: true,
		}
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
		IsError: false,
	}
}

// buildArgs constructs the argument list for the agent-browser command.
// It splits the user command string and prepends global flags.
func (t *BrowserTool) buildArgs(command string) []string {
	var globalArgs []string

	// Add CDP port
	globalArgs = append(globalArgs, "--cdp", fmt.Sprintf("%d", t.cdpPort))

	// Add session flag if configured
	if t.session != "" {
		globalArgs = append(globalArgs, "--session", t.session)
	}

	// Add --headed if not headless (agent-browser defaults to headless)
	if !t.headless {
		globalArgs = append(globalArgs, "--headed")
	}

	// Add --json for machine-readable output
	globalArgs = append(globalArgs, "--json")

	// Parse the command string into arguments, respecting quotes
	cmdArgs := splitCommand(command)

	return append(globalArgs, cmdArgs...)
}

// splitCommand splits a command string into arguments, respecting quoted strings.
func splitCommand(command string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(command); i++ {
		ch := command[i]
		switch {
		case inQuote:
			if ch == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(ch)
			}
		case ch == '"' || ch == '\'':
			inQuote = true
			quoteChar = ch
		case ch == ' ' || ch == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

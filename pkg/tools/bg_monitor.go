package tools

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	bgWatchPollInterval   = 100 * time.Millisecond
	bgWatchDefaultTimeout = 30 * time.Second
	bgTailDefaultLines    = 20
)

// BgMonitorTool monitors and inspects background processes managed by ExecTool.
type BgMonitorTool struct {
	exec *ExecTool
}

// NewBgMonitorTool creates a new BgMonitorTool that accesses bg processes from the given ExecTool.
func NewBgMonitorTool(exec *ExecTool) *BgMonitorTool {
	return &BgMonitorTool{exec: exec}
}

func (t *BgMonitorTool) Name() string {
	return "bg_monitor"
}

func (t *BgMonitorTool) Description() string {
	return "Monitor and inspect background processes. Use 'list' to see all, 'watch' to wait for output pattern, 'tail' to get recent log lines."
}

func (t *BgMonitorTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "watch", "tail"},
				"description": "Action: 'list' all bg processes, 'watch' for a pattern in output, 'tail' recent output lines.",
			},
			"bg_id": map[string]any{
				"type":        "string",
				"description": "Background process ID (e.g. 'bg-1'). Required for watch and tail.",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to watch for in output (used with action='watch').",
			},
			"lines": map[string]any{
				"type":        "number",
				"description": "Number of recent lines to return (used with action='tail', default 20).",
			},
			"watch_timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds for watch action (default 30).",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BgMonitorTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	switch action {
	case "list":
		return t.actionList()
	case "watch":
		return t.actionWatch(ctx, args)
	case "tail":
		return t.actionTail(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action %q (use 'list', 'watch', or 'tail')", action))
	}
}

func (t *BgMonitorTool) actionList() *ToolResult {
	procs := t.exec.BgProcesses()
	if len(procs) == 0 {
		return &ToolResult{
			ForLLM:  "No background processes.",
			ForUser: "No background processes.",
		}
	}

	ids := make([]string, 0, len(procs))
	for id := range procs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var sb strings.Builder
	sb.WriteString("Background Processes:\n\n")
	for _, id := range ids {
		bp := procs[id]
		if bp.isRunning() {
			uptime := time.Since(bp.startedAt).Truncate(time.Second)
			fmt.Fprintf(&sb, "  [%s] pid=%d running  (uptime: %s, max: %s) %s\n",
				id, bp.pid, uptime, getBgMaxLifetime(), bp.command)
		} else {
			ran := time.Since(bp.startedAt).Truncate(time.Second)
			if bp.exitErr != nil {
				fmt.Fprintf(&sb, "  [%s] pid=%d exited=err (ran: %s) %s\n",
					id, bp.pid, ran, bp.command)
			} else {
				fmt.Fprintf(&sb, "  [%s] pid=%d exited=0 (ran: %s) %s\n",
					id, bp.pid, ran, bp.command)
			}
		}
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}

func (t *BgMonitorTool) actionWatch(ctx context.Context, args map[string]any) *ToolResult {
	bgID, _ := args["bg_id"].(string)
	if bgID == "" {
		return ErrorResult("bg_id is required for watch action")
	}

	patternStr, _ := args["pattern"].(string)
	if patternStr == "" {
		return ErrorResult("pattern is required for watch action")
	}

	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid regex pattern %q: %v", patternStr, err))
	}

	timeout := bgWatchDefaultTimeout
	if t, ok := args["watch_timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	procs := t.exec.BgProcesses()
	bp, ok := procs[bgID]
	if !ok {
		return ErrorResult(fmt.Sprintf("background process %q not found", bgID))
	}

	deadline := time.After(timeout)
	ticker := time.NewTicker(bgWatchPollInterval)
	defer ticker.Stop()

	for {
		// Check for pattern match
		if match := bp.output.Match(pattern); match != "" {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("Match found in [%s]: %s", bgID, match),
				ForUser: fmt.Sprintf("Match found in [%s]: %s", bgID, match),
			}
		}

		// Check if process exited
		if !bp.isRunning() {
			output := bp.output.String()
			tail := lastNLines(output, 10)
			var sb strings.Builder
			fmt.Fprintf(&sb, "Process %s exited before pattern matched.\n", bgID)
			if bp.exitErr != nil {
				fmt.Fprintf(&sb, "Exit: %v\n", bp.exitErr)
			} else {
				fmt.Fprintf(&sb, "Exit: 0\n")
			}
			fmt.Fprintf(&sb, "\nLast output:\n%s", tail)
			return &ToolResult{
				ForLLM:  sb.String(),
				ForUser: sb.String(),
				IsError: true,
			}
		}

		select {
		case <-deadline:
			// Timeout
			output := bp.output.String()
			tail := lastNLines(output, 10)
			var sb strings.Builder
			fmt.Fprintf(&sb, "Watch timed out after %s waiting for pattern %q in [%s].\n", timeout, patternStr, bgID)
			fmt.Fprintf(&sb, "\nLast output:\n%s", tail)
			return &ToolResult{
				ForLLM:  sb.String(),
				ForUser: sb.String(),
				IsError: true,
			}
		case <-ctx.Done():
			return ErrorResult("watch cancelled")
		case <-ticker.C:
			// Continue polling
		}
	}
}

func (t *BgMonitorTool) actionTail(args map[string]any) *ToolResult {
	bgID, _ := args["bg_id"].(string)
	if bgID == "" {
		return ErrorResult("bg_id is required for tail action")
	}

	n := bgTailDefaultLines
	if lines, ok := args["lines"].(float64); ok && lines > 0 {
		n = int(lines)
	}

	procs := t.exec.BgProcesses()
	bp, ok := procs[bgID]
	if !ok {
		return ErrorResult(fmt.Sprintf("background process %q not found", bgID))
	}

	lines := bp.output.Lines(n)

	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] pid=%d %s\n", bp.id, bp.pid, bp.command)
	if bp.isRunning() {
		fmt.Fprintf(&sb, "Status: running\n")
	} else {
		if bp.exitErr != nil {
			fmt.Fprintf(&sb, "Status: exited (%v)\n", bp.exitErr)
		} else {
			fmt.Fprintf(&sb, "Status: exited=0\n")
		}
	}
	fmt.Fprintf(&sb, "\nLast %d lines:\n", n)
	for _, line := range lines {
		fmt.Fprintf(&sb, "%s\n", line)
	}

	if len(lines) == 0 {
		fmt.Fprintf(&sb, "(no output)\n")
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}

// lastNLines returns the last n lines from a string.
func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if n >= len(lines) {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

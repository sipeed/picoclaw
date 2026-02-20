package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type VerifyTool struct {
	workspace    string
	restrict     bool
	denyPatterns []*regexp.Regexp
}

func NewVerifyTool(workspace string, restrict bool) *VerifyTool {
	return &VerifyTool{
		workspace:    workspace,
		restrict:     restrict,
		denyPatterns: defaultDenyPatterns, // Reusing from shell.go (they are in the same package)
	}
}

func (t *VerifyTool) Name() string {
	return "verify"
}

func (t *VerifyTool) Description() string {
	return "Verify the results of your work by running a check command (e.g., 'go test', 'build'). Use this to ensure your changes didn't break anything."
}

func (t *VerifyTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The verification command to run",
			},
			"label": map[string]interface{}{
				"type":        "string",
				"description": "A short label for the verification step (e.g., 'Run unit tests')",
			},
		},
		"required": []string{"command"},
	}
}

func (t *VerifyTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
		return ErrorResult("command is required")
	}

	label, _ := args["label"].(string)
	if label == "" {
		label = "Verification"
	}

	// Safety check (reusing logic from shell.go)
	if guardError := t.guardCommand(command, t.workspace); guardError != "" {
		return ErrorResult(guardError)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = t.workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outputStr := stdout.String()
	if stderr.Len() > 0 {
		outputStr += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		return &ToolResult{
			Err:     fmt.Errorf("%s failed: %w", label, err),
			ForLLM:  fmt.Sprintf("%s FAILED\n\nOutput:\n%s", label, outputStr),
			ForUser: fmt.Sprintf("❌ %s failed.\n```\n%s\n```", label, outputStr),
		}
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("%s PASSED\n\nOutput:\n%s", label, outputStr),
		ForUser: fmt.Sprintf("✅ %s passed successfully.", label),
	}
}

func (t *VerifyTool) guardCommand(command, cwd string) string {
	cmdText := strings.TrimSpace(command)
	lower := strings.ToLower(cmdText)

	for _, pattern := range t.denyPatterns {
		if pattern.MatchString(lower) {
			return "Command blocked by safety guard (dangerous pattern detected)"
		}
	}

	if t.restrict {
		if strings.Contains(cmdText, "..\\") || strings.Contains(cmdText, "../") {
			return "Command blocked by safety guard (path traversal detected)"
		}

		cwdPath, err := filepath.Abs(cwd)
		if err != nil {
			return ""
		}

		pathPattern := regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)
		matches := pathPattern.FindAllString(cmdText, -1)

		for _, raw := range matches {
			p, err := filepath.Abs(raw)
			if err != nil {
				continue
			}

			rel, err := filepath.Rel(cwdPath, p)
			if err != nil {
				continue
			}

			if strings.HasPrefix(rel, "..") {
				return "Command blocked by safety guard (path outside working dir)"
			}
		}
	}

	return ""
}

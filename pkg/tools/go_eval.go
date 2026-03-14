package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type GoEvalTool struct {
	workspace string
	timeout   time.Duration
}

func NewGoEvalTool(workspace string) *GoEvalTool {
	return &GoEvalTool{
		workspace: workspace,
		timeout:   60 * time.Second, // Default timeout
	}
}

func (t *GoEvalTool) Name() string {
	return "go_eval"
}

func (t *GoEvalTool) Description() string {
	return "Executes Go code dynamically. Provide valid Go source code containing a 'main' function. The code will be saved to a temporary file, compiled, and executed. Useful for complex logic or tasks that require writing a Go script."
}

func (t *GoEvalTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Valid Go source code to execute (must include 'package main' and 'func main()').",
			},
		},
		"required": []string{"code"},
	}
}

func (t *GoEvalTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	code, ok := args["code"].(string)
	if !ok || code == "" {
		return ErrorResult("code is required")
	}

	// Create a temporary directory for the Go code
	tmpDir, err := os.MkdirTemp(t.workspace, "go_eval_*")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create temp dir: %v", err))
	}
	defer os.RemoveAll(tmpDir)

	// Write code to main.go
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(code), 0600); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write code to file: %v", err))
	}

	// Initialize a go module to allow imports from standard library
	cmdMod := exec.Command("go", "mod", "init", "goeval")
	cmdMod.Dir = tmpDir
	if out, err := cmdMod.CombinedOutput(); err != nil {
		return ErrorResult(fmt.Sprintf("failed to init go mod: %v\nOutput: %s", err, string(out)))
	}

	// Tidy the module to fetch any dependencies
	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = tmpDir
	if out, err := cmdTidy.CombinedOutput(); err != nil {
		// Log the error but don't fail immediately, it might just be standard library
		_ = out
	}

	// Run the code
	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmdRun := exec.CommandContext(cmdCtx, "go", "run", "main.go")
	cmdRun.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmdRun.Stdout = &stdout
	cmdRun.Stderr = &stderr

	err = cmdRun.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\nSTDERR:\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("Execution timed out after %v.\nOutput so far:\n%s", t.timeout, output),
				ForUser: "Execution timed out.",
				IsError: true,
			}
		}

		return &ToolResult{
			ForLLM:  fmt.Sprintf("Execution failed: %v\nOutput:\n%s", err, output),
			ForUser: "Execution failed.",
			IsError: true,
		}
	}

	if output == "" {
		output = "(no output)"
	}

	// Truncate output if necessary
	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

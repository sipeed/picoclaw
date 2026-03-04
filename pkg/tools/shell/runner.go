package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// RunConfig holds all parameters for a single command execution.
type RunConfig struct {
	Command      string
	Dir          string
	Timeout      time.Duration
	Restrict     bool
	WorkspaceDir string

	RiskThreshold     RiskLevel
	RiskOverrides     map[string]string
	ExtraArgModifiers map[string][]ArgModifier // user-defined, appended after built-ins
	EnvAllowlist      []string
	EnvSet            map[string]string
}

// RunResult contains the output of a command execution.
type RunResult struct {
	Output  string
	IsError bool
}

// Run parses and executes a shell command using the in-process interpreter.
func Run(ctx context.Context, cfg RunConfig) RunResult {
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	prog, err := parser.Parse(strings.NewReader(cfg.Command), "")
	if err != nil {
		return RunResult{
			Output:  fmt.Sprintf("Failed to parse command: %v", err),
			IsError: true,
		}
	}

	var runCtx context.Context
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	env := BuildSanitizedEnv(cfg.EnvAllowlist, cfg.EnvSet)

	var stdout, stderr bytes.Buffer

	opts := []interp.RunnerOption{
		interp.Env(env),
		interp.StdIO(nil, &stdout, &stderr),
	}

	if cfg.Dir != "" {
		opts = append(opts, interp.Dir(cfg.Dir))
	}

	opts = append(
		opts,
		interp.ExecHandlers(
			pathAwareExecHandler(env),
			riskExecHandler(cfg.RiskThreshold, cfg.RiskOverrides, cfg.ExtraArgModifiers),
		),
	)

	if cfg.Restrict && cfg.WorkspaceDir != "" {
		opts = append(opts, interp.OpenHandler(SandboxedOpenHandler(cfg.WorkspaceDir)))
	}

	runner, err := interp.New(opts...)
	if err != nil {
		return RunResult{
			Output:  fmt.Sprintf("Failed to create interpreter: %v", err),
			IsError: true,
		}
	}

	err = runner.Run(runCtx, prog)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("Command timed out after %v", cfg.Timeout)
			return RunResult{Output: msg, IsError: true}
		}

		errStr := err.Error()
		if strings.Contains(errStr, "Command blocked by risk classifier") {
			return RunResult{Output: errStr, IsError: true}
		}

		output += fmt.Sprintf("\nExit code: %v", err)
	}

	if output == "" {
		output = "(no output)"
	}

	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	return RunResult{
		Output:  output,
		IsError: err != nil,
	}
}

// riskExecHandler returns an ExecHandlers middleware that classifies each resolved
// command against the risk table before delegating to the default handler.
func riskExecHandler(
	threshold RiskLevel,
	overrides map[string]string,
	extraMods map[string][]ArgModifier,
) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			level := ClassifyCommand(args, overrides, extraMods)
			if !IsAllowed(level, threshold) {
				reason := "command risk exceeds configured threshold"
				return fmt.Errorf("%s", BlockedCommandError(args, level, threshold, reason))
			}

			return next(ctx, args)
		}
	}
}

// pathAwareExecHandler returns an ExecHandlerFunc that looks up commands
// using the interpreter's environment (not os.Getenv). This ensures that
// PATH from the sanitized environment is used for command resolution.
func pathAwareExecHandler(env expand.Environ) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Look up command using interpreter's PATH
			path, err := lookPath(env, args[0])
			if err != nil {
				// Command not found in PATH, try the default handler
				return next(ctx, args)
			}

			// Replace command with full path and continue
			fullArgs := append([]string{path}, args[1:]...)
			return next(ctx, fullArgs)
		}
	}
}

// lookPath searches for an executable named cmd in the directories
// listed in the PATH variable from the given environment.
func lookPath(env expand.Environ, cmd string) (string, error) {
	// If command contains a slash, it's a path - return as-is
	if strings.Contains(cmd, "/") {
		return cmd, nil
	}

	// Get PATH from interpreter environment
	pathVar := env.Get("PATH")
	if !pathVar.Set {
		// PATH not set in environment - let default handler try
		return "", fmt.Errorf("PATH not set")
	}
	if pathVar.Str == "" {
		return "", fmt.Errorf("PATH is empty")
	}

	// Search each directory in PATH
	for _, dir := range filepath.SplitList(pathVar.Str) {
		if dir == "" {
			dir = "."
		}
		fullPath := filepath.Join(dir, cmd)
		// Check if file exists and is executable
		if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
			// On Unix, check executable bit
			if stat.Mode()&0o111 != 0 {
				return fullPath, nil
			}
		}
	}

	// Not found - let the default handler handle it
	return "", fmt.Errorf("command %q not found in PATH", cmd)
}

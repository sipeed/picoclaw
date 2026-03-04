package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools/shell"
)

// Compile-time interface checks.
var (
	_ Tool      = (*ExecTool)(nil)
	_ AsyncTool = (*ExecTool)(nil)
)

// ExecTool executes shell commands using an in-process interpreter
// with AST-based risk classification, env sanitization, and file-access sandboxing.
//
// ExecTool implements AsyncTool. When the LLM passes background=true the
// command runs in a goroutine and the result is delivered via the callback
// injected by the tool registry.
type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	restrictToWorkspace bool

	riskThreshold shell.RiskLevel
	riskOverrides map[string]string
	argModifiers  map[string][]shell.ArgModifier
	envAllowlist  []string
	envSet        map[string]string

	callback AsyncCallback
}

func NewExecTool(workingDir string, restrict bool) (*ExecTool, error) {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(workingDir string, restrict bool, cfg *config.Config) (*ExecTool, error) {
	t := &ExecTool{
		workingDir:          workingDir,
		timeout:             60 * time.Second,
		restrictToWorkspace: restrict,
		riskThreshold:       shell.RiskMedium,
	}

	if cfg != nil {
		execCfg := cfg.Tools.Exec

		warnDeprecatedExecConfig(execCfg)

		if execCfg.RiskThreshold != "" {
			level, err := shell.ParseRiskLevel(execCfg.RiskThreshold)
			if err != nil {
				fmt.Printf("Warning: invalid risk_threshold %q: %v. Using medium.\n", execCfg.RiskThreshold, err)
			} else {
				t.riskThreshold = level
			}
		}
		t.riskOverrides = execCfg.RiskOverrides
		t.argModifiers = parseArgModifiers(execCfg.ArgModifiers)
		t.envAllowlist = execCfg.EnvAllowlist
		t.envSet = execCfg.EnvSet
	}

	return t, nil
}

func warnDeprecatedExecConfig(cfg config.ExecConfig) {
	if len(cfg.CustomDenyPatterns) > 0 {
		fmt.Println("Warning: 'custom_deny_patterns' is deprecated and ignored. " +
			"Use 'risk_overrides' to adjust per-command risk levels.")
	}
	if len(cfg.CustomAllowPatterns) > 0 {
		fmt.Println("Warning: 'custom_allow_patterns' is deprecated and ignored. " +
			"Use 'risk_overrides' to lower the risk level of specific commands.")
	}
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Use with caution."
}

// SetCallback implements AsyncTool.
func (t *ExecTool) SetCallback(cb AsyncCallback) {
	t.callback = cb
}

func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "Run the command in the background. Returns immediately; result is delivered asynchronously.",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
		return ErrorResult("command is required")
	}

	cwd := t.workingDir
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		if t.restrictToWorkspace && t.workingDir != "" {
			resolvedWD, err := validatePath(wd, t.workingDir, true)
			if err != nil {
				return ErrorResult("Command blocked by safety guard (" + err.Error() + ")")
			}
			cwd = resolvedWD
		} else {
			cwd = wd
		}
	}

	if cwd == "" {
		wd, err := os.Getwd()
		if err == nil {
			cwd = wd
		}
	}

	cfg := shell.RunConfig{
		Command:           command,
		Dir:               cwd,
		Timeout:           t.timeout,
		Restrict:          t.restrictToWorkspace,
		WorkspaceDir:      t.workingDir,
		RiskThreshold:     t.riskThreshold,
		RiskOverrides:     t.riskOverrides,
		ExtraArgModifiers: t.argModifiers,
		EnvAllowlist:      t.envAllowlist,
		EnvSet:            t.envSet,
	}

	background, _ := args["background"].(bool)
	if background && t.callback != nil {
		return t.executeAsync(ctx, cfg)
	}

	result := shell.Run(ctx, cfg)
	return &ToolResult{
		ForLLM:  result.Output,
		ForUser: result.Output,
		IsError: result.IsError,
	}
}

// executeAsync launches the command in a goroutine and delivers the result
// through the AsyncCallback. The parent ctx is used for cancellation so the
// goroutine respects agent shutdown.
func (t *ExecTool) executeAsync(ctx context.Context, cfg shell.RunConfig) *ToolResult {
	cb := t.callback // capture before goroutine
	go func() {
		result := shell.Run(ctx, cfg)
		cb(ctx, &ToolResult{
			ForLLM:  result.Output,
			ForUser: result.Output,
			IsError: result.IsError,
		})
	}()
	return AsyncResult(fmt.Sprintf("Running `%s` in background", cfg.Command))
}

func (t *ExecTool) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *ExecTool) SetRestrictToWorkspace(restrict bool) {
	t.restrictToWorkspace = restrict
}

func (t *ExecTool) SetRiskThreshold(level shell.RiskLevel) {
	t.riskThreshold = level
}

// parseArgModifiers converts config.ArgModifierConfig entries into shell.ArgModifier.
func parseArgModifiers(raw map[string][]config.ArgModifierConfig) map[string][]shell.ArgModifier {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string][]shell.ArgModifier, len(raw))
	for cmd, entries := range raw {
		for _, e := range entries {
			level, err := shell.ParseRiskLevel(e.Level)
			if err != nil {
				fmt.Printf(
					"Warning: invalid risk level %q for command %q: %v. Skipping this modifier.\n",
					e.Level,
					cmd,
					err,
				)
				continue
			}
			out[cmd] = append(out[cmd], shell.ArgModifier{
				Args:  e.Args,
				Level: level,
			})
		}
	}
	return out
}

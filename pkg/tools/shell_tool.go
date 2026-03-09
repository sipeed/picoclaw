package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools/shell"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// Compile-time interface checks.
var (
	_ Tool          = (*ExecTool)(nil)
	_ AsyncExecutor = (*ExecTool)(nil)
)

// ExecTool executes shell commands using an in-process interpreter
// with AST-based risk classification, env sanitization, and file-access sandboxing.
//
// ExecTool implements AsyncExecutor. When the LLM passes background=true,
// the command runs in a goroutine and the result is delivered via the
// AsyncCallback provided by the registry. Without a callback,
// background=true falls back to synchronous execution.
type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	restrictToWorkspace bool

	riskThreshold shell.RiskLevel
	riskOverrides map[string]string
	argProfiles   map[string]shell.FlagProfile
	argModifiers  map[string][]shell.ArgModifier
	envAllowlist  []string
	envSet        map[string]string
}

func NewExecTool(workingDir string, restrict bool) (*ExecTool, error) {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(
	workingDir string,
	restrict bool,
	cfg *config.Config,
) (*ExecTool, error) {
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
		t.riskOverrides = shell.NormalizeCommandKeys(execCfg.RiskOverrides)
		t.argProfiles = shell.NormalizeCommandKeys(parseArgProfiles(execCfg.ArgProfiles))
		t.argModifiers = shell.NormalizeCommandKeys(parseArgModifiers(execCfg.ArgModifiers))
		t.envAllowlist = execCfg.EnvAllowlist
		t.envSet = execCfg.EnvSet
	}

	return t, nil
}

func warnDeprecatedExecConfig(cfg config.ExecConfig) {
	if cfg.EnableDenyPatterns != nil {
		if !*cfg.EnableDenyPatterns {
			fmt.Println("Warning: 'enable_deny_patterns: false' is deprecated and ignored. " +
				"Previously this disabled all command filtering. The new risk-based system " +
				"is now always active (default threshold=medium). " +
				"To allow all commands, set 'risk_threshold: critical'.")
		} else {
			fmt.Println("Warning: 'enable_deny_patterns' is deprecated and ignored. " +
				"Command filtering is now always active via the risk-based classifier. " +
				"Remove this field from your config.")
		}
	}
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
	cfg, err := t.buildConfig(args)
	if err != nil {
		return err
	}

	result := shell.Run(ctx, cfg)
	return &ToolResult{
		ForLLM:  result.Output,
		ForUser: result.Output,
		IsError: result.IsError,
	}
}

// ExecuteAsync implements AsyncExecutor. The registry calls this for
// parallel tool dispatch. When background=true and a callback is provided,
// the command runs in a goroutine and the result is delivered via cb.
// Without a callback, background=true falls back to synchronous execution.
func (t *ExecTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	background, _ := args["background"].(bool)
	if !background || cb == nil {
		return t.Execute(ctx, args)
	}

	cfg, errResult := t.buildConfig(args)
	if errResult != nil {
		return errResult
	}

	go func() {
		result := shell.Run(ctx, cfg)

		var content string
		if result.IsError {
			content = fmt.Sprintf("Background command failed: `%s`\n\n%s",
				utils.Truncate(cfg.Command, 100), result.Output)
		} else {
			content = fmt.Sprintf("Background command completed: `%s`\n\n%s",
				utils.Truncate(cfg.Command, 100), result.Output)
		}

		if cb != nil {
			cb(context.Background(), &ToolResult{
				ForLLM:  content,
				ForUser: content,
				IsError: result.IsError,
			})
		}
	}()
	return AsyncResult(fmt.Sprintf("Running `%s` in background", utils.Truncate(cfg.Command, 100)))
}

// buildConfig extracts args and returns a RunConfig. Returns a *ToolResult
// error if validation fails (bad command, sandbox violation, etc.).
func (t *ExecTool) buildConfig(args map[string]any) (shell.RunConfig, *ToolResult) {
	command, ok := args["command"].(string)
	if !ok {
		return shell.RunConfig{}, ErrorResult("command is required")
	}

	cwd := t.workingDir
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		if t.restrictToWorkspace && t.workingDir != "" {
			resolvedWD, err := validatePath(wd, t.workingDir, true)
			if err != nil {
				return shell.RunConfig{}, ErrorResult("Command blocked by safety guard (" + err.Error() + ")")
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

	return shell.RunConfig{
		Command:           command,
		Dir:               cwd,
		Timeout:           t.timeout,
		Restrict:          t.restrictToWorkspace,
		WorkspaceDir:      t.workingDir,
		RiskThreshold:     t.riskThreshold,
		RiskOverrides:     t.riskOverrides,
		ExtraFlagProfiles: t.argProfiles,
		ExtraArgModifiers: t.argModifiers,
		EnvAllowlist:      t.envAllowlist,
		EnvSet:            t.envSet,
	}, nil
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

func parseArgProfiles(raw map[string]config.ArgProfileConfig) map[string]shell.FlagProfile {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]shell.FlagProfile, len(raw))
	for cmd, profile := range raw {
		parsed := shell.FlagProfile{
			SplitCombinedShort: profile.SplitCombinedShort,
			SplitLongEquals:    profile.SplitLongEquals,
		}

		if transforms := parseFlagTransforms(
			cmd,
			"short_attached_value_flags",
			profile.ShortAttachedValue,
		); len(
			transforms,
		) > 0 {
			parsed.ShortAttachedValue = transforms
		}
		if transforms := parseFlagTransforms(
			cmd,
			"separate_value_flags",
			profile.SeparateValueFlags,
		); len(
			transforms,
		) > 0 {
			parsed.SeparateValueFlags = transforms
		}

		out[cmd] = parsed
	}
	return out
}

func parseFlagTransforms(cmd, field string, raw map[string]string) map[string]shell.FlagValueTransform {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]shell.FlagValueTransform, len(raw))
	for flag, name := range raw {
		transform, err := shell.ParseFlagValueTransform(name)
		if err != nil {
			fmt.Printf(
				"Warning: invalid %s transform %q for command %q flag %q: %v. Skipping this flag.\n",
				field,
				name,
				cmd,
				flag,
				err,
			)
			continue
		}
		out[flag] = transform
	}
	return out
}

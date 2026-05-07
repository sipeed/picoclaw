//go:build linux

package isolation

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func applyPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	if !isolation.Enabled {
		return nil
	}
	// Bubblewrap is the only supported Linux backend right now. Fail closed when
	// it is unavailable instead of silently running the child process unisolated.
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		hint := bwrapInstallHint()
		disableHint := `set "isolation.enabled": false in config.json`
		logger.WarnCF("isolation", "bubblewrap is required for Linux isolation",
			map[string]any{
				"binary":            "bwrap",
				"install":           hint,
				"disable_isolation": disableHint,
				"risk":              "disabling isolation lets child processes run without Linux filesystem isolation",
			})
		return fmt.Errorf(
			"linux isolation requires bwrap and does not fall back automatically: %w; install bubblewrap with one of: %s; or disable isolation by setting %s; disabling isolation means child processes can run without Linux filesystem isolation and may access or modify more host files",
			err,
			hint,
			disableHint,
		)
	}
	if cmd == nil || cmd.Path == "" || len(cmd.Args) == 0 {
		return nil
	}

	originalPath := cmd.Path
	originalArgs := append([]string{}, cmd.Args...)
	_, execDir, err := resolveLinuxWorkingDir(cmd.Dir, originalPath)
	if err != nil {
		return err
	}
	resolvedPath, err := resolveLinuxCommandPath(originalPath, execDir)
	if err != nil {
		return err
	}

	// Start from the configured mount plan, then add only the executable, its
	// resolved path, the effective working directory, and any absolute path
	// arguments needed to preserve the original command semantics.
	plan := BuildLinuxMountPlan(root, isolation.ExposePaths)
	plan = ensureLinuxMountRule(plan, resolvedPath, resolvedPath, "ro")
	plan = ensureLinuxMountRule(plan, filepath.Dir(resolvedPath), filepath.Dir(resolvedPath), "ro")
	if resolved, resolveErr := filepath.EvalSymlinks(resolvedPath); resolveErr == nil && resolved != resolvedPath {
		plan = ensureLinuxMountRule(plan, resolved, resolved, "ro")
		plan = ensureLinuxMountRule(plan, filepath.Dir(resolved), filepath.Dir(resolved), "ro")
	}
	if execDir != "" {
		plan = ensureLinuxMountRule(plan, execDir, execDir, "rw")
		if resolved, resolveErr := filepath.EvalSymlinks(execDir); resolveErr == nil && resolved != execDir {
			plan = ensureLinuxMountRule(plan, resolved, resolved, "rw")
		}
	}
	plan = appendLinuxArgumentMounts(plan, originalArgs[1:])
	logger.DebugCF("isolation", "linux isolation mount plan",
		map[string]any{
			"root":        root,
			"command":     resolvedPath,
			"working_dir": execDir,
			"mounts":      formatLinuxMountPlan(plan),
		})
	bwrapArgs, err := buildLinuxBwrapArgs(originalPath, resolvedPath, originalArgs, execDir, plan)
	if err != nil {
		return err
	}

	cmd.Path = bwrapPath
	cmd.Args = bwrapArgs
	cmd.Dir = ""
	return nil
}

func bwrapInstallHint() string {
	return "apt install bubblewrap; dnf install bubblewrap; yum install bubblewrap; pacman -S bubblewrap; apk add bubblewrap"
}

// formatLinuxMountPlan reshapes the internal plan for structured logging.
func formatLinuxMountPlan(plan []MountRule) []map[string]string {
	formatted := make([]map[string]string, 0, len(plan))
	for _, rule := range plan {
		formatted = append(formatted, map[string]string{
			"source": rule.Source,
			"target": rule.Target,
			"mode":   rule.Mode,
		})
	}
	return formatted
}

func postStartPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	return nil
}

func cleanupPendingPlatformResources(cmd *exec.Cmd) {
}

// buildLinuxBwrapArgs translates the mount plan into the bubblewrap command
// line that re-executes the original process inside the isolated mount view.
func buildLinuxBwrapArgs(
	originalPath string,
	resolvedPath string,
	originalArgs []string,
	execDir string,
	plan []MountRule,
) ([]string, error) {
	bwrapArgs := []string{
		"bwrap",
		"--die-with-parent",
		"--unshare-ipc",
		"--proc", "/proc",
		"--dev", "/dev",
	}
	for _, rule := range plan {
		flag, err := linuxBindFlag(rule)
		if err != nil {
			return nil, err
		}
		bwrapArgs = append(bwrapArgs, flag, rule.Source, rule.Target)
	}
	if execDir != "" {
		bwrapArgs = append(bwrapArgs, "--chdir", execDir)
	}
	execPath := originalPath
	if isRelativeCommandPath(originalPath) {
		execPath = resolvedPath
	}
	bwrapArgs = append(bwrapArgs, "--", execPath)
	if len(originalArgs) > 1 {
		bwrapArgs = append(bwrapArgs, originalArgs[1:]...)
	}
	return bwrapArgs, nil
}

func resolveLinuxWorkingDir(originalDir, originalPath string) (string, string, error) {
	if originalDir != "" {
		resolved, err := filepath.Abs(originalDir)
		if err != nil {
			return "", "", fmt.Errorf("resolve command dir %s: %w", originalDir, err)
		}
		return resolved, resolved, nil
	}
	if !isRelativeCommandPath(originalPath) {
		return "", "", nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("resolve current working dir: %w", err)
	}
	return "", wd, nil
}

func resolveLinuxCommandPath(originalPath, execDir string) (string, error) {
	if filepath.IsAbs(originalPath) || !isRelativeCommandPath(originalPath) {
		return filepath.Clean(originalPath), nil
	}
	base := execDir
	if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve current working dir: %w", err)
		}
	}
	return filepath.Clean(filepath.Join(base, originalPath)), nil
}

func appendLinuxArgumentMounts(plan []MountRule, args []string) []MountRule {
	for _, arg := range args {
		path, ok := linuxArgumentPath(arg)
		if !ok {
			continue
		}
		clean := filepath.Clean(path)
		if info, err := os.Stat(clean); err == nil {
			mode := "ro"
			if info.IsDir() {
				mode = "rw"
			}
			plan = ensureLinuxMountRule(plan, clean, clean, mode)
			if resolved, resolveErr := filepath.EvalSymlinks(clean); resolveErr == nil && resolved != clean {
				plan = ensureLinuxMountRule(plan, resolved, resolved, mode)
			}
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			continue
		}
		parent := filepath.Dir(clean)
		if parent == clean {
			continue
		}
		if _, err := os.Stat(parent); err == nil {
			plan = ensureLinuxMountRule(plan, parent, parent, "rw")
		}
	}
	return plan
}

func linuxArgumentPath(arg string) (string, bool) {
	if filepath.IsAbs(arg) {
		return arg, true
	}
	idx := strings.IndexRune(arg, '=')
	if idx <= 0 || idx == len(arg)-1 {
		return "", false
	}
	value := arg[idx+1:]
	if !filepath.IsAbs(value) {
		return "", false
	}
	return value, true
}

func isRelativeCommandPath(path string) bool {
	return !filepath.IsAbs(path) && strings.ContainsRune(path, filepath.Separator)
}

// ensureLinuxMountRule appends a mount rule unless another rule already owns
// the same target path.
func ensureLinuxMountRule(plan []MountRule, source, target, mode string) []MountRule {
	cleanSource := filepath.Clean(source)
	cleanTarget := filepath.Clean(target)
	for _, rule := range plan {
		if filepath.Clean(rule.Target) == cleanTarget {
			return plan
		}
	}
	return append(plan, MountRule{Source: cleanSource, Target: cleanTarget, Mode: mode})
}

// linuxBindFlag selects the correct bubblewrap bind flag based on mount mode.
func linuxBindFlag(rule MountRule) (string, error) {
	info, err := os.Stat(rule.Source)
	if err != nil {
		return "", fmt.Errorf("stat linux mount source %s: %w", rule.Source, err)
	}
	if !info.IsDir() {
		if rule.Mode == "rw" {
			return "--bind", nil
		}
		return "--ro-bind", nil
	}
	if rule.Mode == "rw" {
		return "--bind", nil
	}
	return "--ro-bind", nil
}

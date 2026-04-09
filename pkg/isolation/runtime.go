package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg"
	"github.com/sipeed/picoclaw/pkg/config"
)

// MountRule describes a source-to-target mount exposed inside the Linux
// isolation view.
type MountRule struct {
	Source string
	Target string
	Mode   string
}

// AccessRule describes the effective Windows-side access rule for a host path.
type AccessRule struct {
	Path string
	Mode string
}

// UserEnv contains the redirected per-instance user directories injected into
// isolated child processes.
type UserEnv struct {
	Home         string
	Tmp          string
	Config       string
	Cache        string
	State        string
	AppData      string
	LocalAppData string
}

var (
	isolationMu      sync.RWMutex
	currentIsolation = config.DefaultConfig().Isolation
)

// Configure updates the process-wide isolation state used by subsequent child
// process launches.
func Configure(cfg *config.Config) {
	isolationMu.Lock()
	defer isolationMu.Unlock()
	if cfg == nil {
		defaults := config.DefaultConfig()
		currentIsolation = defaults.Isolation
		return
	}
	currentIsolation = cfg.Isolation
}

// CurrentConfig returns the currently active isolation settings.
func CurrentConfig() config.IsolationConfig {
	isolationMu.RLock()
	defer isolationMu.RUnlock()
	return currentIsolation
}

// ResolveInstanceRoot resolves the instance root used to build the isolated
// filesystem and redirected user environment.
func ResolveInstanceRoot() (string, error) {
	root := filepath.Clean(config.GetHome())
	if root == "." {
		return "", fmt.Errorf("instance root resolved to current directory")
	}
	return root, nil
}

// PrepareInstanceRoot creates the directories required by the isolation runtime.
func PrepareInstanceRoot(root string) error {
	for _, dir := range InstanceDirs(root) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("prepare instance dir %s: %w", dir, err)
		}
	}
	return nil
}

// InstanceDirs returns the directories that must exist under the instance root
// for isolation-aware child processes.
func InstanceDirs(root string) []string {
	dirs := []string{
		root,
		filepath.Join(root, "skills"),
		filepath.Join(root, "logs"),
		filepath.Join(root, "cache"),
		filepath.Join(root, "state"),
		filepath.Join(root, "runtime-user-env"),
		filepath.Join(root, "runtime-user-env", "home"),
		filepath.Join(root, "runtime-user-env", "tmp"),
		filepath.Join(root, "runtime-user-env", "config"),
		filepath.Join(root, "runtime-user-env", "cache"),
		filepath.Join(root, "runtime-user-env", "state"),
	}
	dirs = append(dirs, filepath.Join(root, pkg.WorkspaceName))
	if runtime.GOOS == "windows" {
		dirs = append(dirs,
			filepath.Join(root, "runtime-user-env", "AppData", "Roaming"),
			filepath.Join(root, "runtime-user-env", "AppData", "Local"),
		)
	}
	return dirs
}

// ResolveUserEnv derives the redirected user directories rooted under the
// instance runtime area.
func ResolveUserEnv(root string) UserEnv {
	base := filepath.Join(root, "runtime-user-env")
	return UserEnv{
		Home:         filepath.Join(base, "home"),
		Tmp:          filepath.Join(base, "tmp"),
		Config:       filepath.Join(base, "config"),
		Cache:        filepath.Join(base, "cache"),
		State:        filepath.Join(base, "state"),
		AppData:      filepath.Join(base, "AppData", "Roaming"),
		LocalAppData: filepath.Join(base, "AppData", "Local"),
	}
}

// ApplyUserEnv rewrites the child process environment so home, temp, and
// platform-specific user-data directories point into the instance root.
func ApplyUserEnv(cmd *exec.Cmd, root string) {
	userEnv := ResolveUserEnv(root)
	envMap := make(map[string]string)
	for _, item := range cmd.Environ() {
		if idx := strings.IndexRune(item, '='); idx > 0 {
			envMap[item[:idx]] = item[idx+1:]
		}
	}

	if runtime.GOOS == "windows" {
		envMap["USERPROFILE"] = userEnv.Home
		envMap["HOME"] = userEnv.Home
		envMap["TEMP"] = userEnv.Tmp
		envMap["TMP"] = userEnv.Tmp
		envMap["APPDATA"] = userEnv.AppData
		envMap["LOCALAPPDATA"] = userEnv.LocalAppData
	} else {
		envMap["HOME"] = userEnv.Home
		envMap["TMPDIR"] = userEnv.Tmp
		envMap["XDG_CONFIG_HOME"] = userEnv.Config
		envMap["XDG_CACHE_HOME"] = userEnv.Cache
		envMap["XDG_STATE_HOME"] = userEnv.State
	}

	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
}

// ValidateExposePaths verifies the user-supplied path exposure rules before a
// child process is started.
func ValidateExposePaths(items []config.ExposePath) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		if item.Source == "" {
			return fmt.Errorf("source is required")
		}
		if item.Mode != "ro" && item.Mode != "rw" {
			return fmt.Errorf("invalid expose_paths mode: %s", item.Mode)
		}

		source := filepath.Clean(item.Source)
		target := item.Target
		if target == "" {
			target = source
		}
		target = filepath.Clean(target)

		if !filepath.IsAbs(source) || !filepath.IsAbs(target) {
			return fmt.Errorf("source and target must be absolute paths")
		}
		if _, ok := seen[target]; ok {
			return fmt.Errorf("duplicate expose_path target: %s", target)
		}
		seen[target] = struct{}{}
	}
	return nil
}

// NormalizeExposePath fills implicit defaults and cleans path values so merge
// and validation logic can work with canonical paths.
func NormalizeExposePath(item config.ExposePath) config.ExposePath {
	source := filepath.Clean(item.Source)
	target := item.Target
	if target == "" {
		target = source
	}
	return config.ExposePath{
		Source: source,
		Target: filepath.Clean(target),
		Mode:   item.Mode,
	}
}

// DefaultExposePaths returns the minimum built-in host paths required for the
// current platform to run isolated child processes.
func DefaultExposePaths(root string) []config.ExposePath {
	items := []config.ExposePath{{
		Source: root,
		Target: root,
		Mode:   "rw",
	}}
	if runtime.GOOS == "linux" {
		items = append(items, defaultLinuxSystemExposePaths()...)
	}
	return items
}

func defaultLinuxSystemExposePaths() []config.ExposePath {
	return existingExposePaths([]config.ExposePath{
		{Source: "/usr", Target: "/usr", Mode: "ro"},
		{Source: "/bin", Target: "/bin", Mode: "ro"},
		{Source: "/lib", Target: "/lib", Mode: "ro"},
		{Source: "/lib64", Target: "/lib64", Mode: "ro"},
		{Source: "/etc/resolv.conf", Target: "/etc/resolv.conf", Mode: "ro"},
		{Source: "/etc/hosts", Target: "/etc/hosts", Mode: "ro"},
		{Source: "/etc/nsswitch.conf", Target: "/etc/nsswitch.conf", Mode: "ro"},
		{Source: "/etc/passwd", Target: "/etc/passwd", Mode: "ro"},
		{Source: "/etc/group", Target: "/etc/group", Mode: "ro"},
		{Source: "/etc/ssl", Target: "/etc/ssl", Mode: "ro"},
		{Source: "/etc/pki", Target: "/etc/pki", Mode: "ro"},
		{Source: "/etc/ca-certificates", Target: "/etc/ca-certificates", Mode: "ro"},
		{Source: "/usr/share/ca-certificates", Target: "/usr/share/ca-certificates", Mode: "ro"},
		{Source: "/usr/local/share/ca-certificates", Target: "/usr/local/share/ca-certificates", Mode: "ro"},
		{Source: "/etc/alternatives", Target: "/etc/alternatives", Mode: "ro"},
		{Source: "/usr/share/zoneinfo", Target: "/usr/share/zoneinfo", Mode: "ro"},
		{Source: "/etc/localtime", Target: "/etc/localtime", Mode: "ro"},
	})
}

// existingExposePaths keeps only the builtin host paths that exist on the
// current machine so Linux isolation does not fail on distro-specific paths.
func existingExposePaths(items []config.ExposePath) []config.ExposePath {
	filtered := make([]config.ExposePath, 0, len(items))
	for _, item := range items {
		if _, err := os.Stat(item.Source); err == nil {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// MergeExposePaths merges built-in rules with user overrides. Rules are keyed
// by target path so later entries replace earlier ones for the same target.
func MergeExposePaths(defaults []config.ExposePath, overrides []config.ExposePath) []config.ExposePath {
	merged := make([]config.ExposePath, 0, len(defaults)+len(overrides))
	indexByTarget := make(map[string]int, len(defaults)+len(overrides))
	appendOrReplace := func(item config.ExposePath) {
		normalized := NormalizeExposePath(item)
		if idx, ok := indexByTarget[normalized.Target]; ok {
			merged[idx] = normalized
			return
		}
		indexByTarget[normalized.Target] = len(merged)
		merged = append(merged, normalized)
	}
	for _, item := range defaults {
		appendOrReplace(item)
	}
	for _, item := range overrides {
		appendOrReplace(item)
	}
	return merged
}

// BuildLinuxMountPlan converts the merged expose-path configuration into the
// mount rules consumed by the Linux bubblewrap backend.
func BuildLinuxMountPlan(root string, overrides []config.ExposePath) []MountRule {
	merged := MergeExposePaths(DefaultExposePaths(root), overrides)
	plan := make([]MountRule, 0, len(merged))
	for _, item := range merged {
		plan = append(plan, MountRule{Source: item.Source, Target: item.Target, Mode: item.Mode})
	}
	return plan
}

// BuildWindowsAccessRules derives the host-path access policy used by the
// Windows restricted-token backend.
func BuildWindowsAccessRules(root string, overrides []config.ExposePath) []AccessRule {
	merged := MergeExposePaths(nil, overrides)
	rules := make([]AccessRule, 0, len(merged)+1)
	rules = append(rules, AccessRule{Path: root, Mode: "rw"})
	for _, item := range merged {
		rules = append(rules, AccessRule{Path: item.Source, Mode: item.Mode})
	}
	return rules
}

func validateWindowsExposePaths(items []config.ExposePath) error {
	if len(items) == 0 {
		return nil
	}
	return fmt.Errorf("windows isolation does not yet support expose_paths filesystem rules")
}

// IsSupported reports whether the current platform has an implemented isolation
// backend.
func IsSupported() bool {
	return isSupportedOn(runtime.GOOS)
}

func isSupportedOn(goos string) bool {
	switch goos {
	case "linux", "windows":
		return true
	default:
		return false
	}
}

// Preflight validates the configured isolation state and prepares the instance
// runtime directories before any child process is launched.
func Preflight() error {
	isolation := CurrentConfig()
	if !isolation.Enabled {
		return nil
	}
	if !IsSupported() {
		return fmt.Errorf("subprocess isolation is not supported on %s", runtime.GOOS)
	}
	root, err := ResolveInstanceRoot()
	if err != nil {
		return err
	}
	if err := PrepareInstanceRoot(root); err != nil {
		return err
	}
	if err := ValidateExposePaths(isolation.ExposePaths); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		for _, rule := range BuildLinuxMountPlan(root, isolation.ExposePaths) {
			if rule.Source == "" || rule.Target == "" {
				return fmt.Errorf("invalid linux mount rule")
			}
		}
	}
	if runtime.GOOS == "windows" {
		if err := validateWindowsExposePaths(isolation.ExposePaths); err != nil {
			return err
		}
		for _, rule := range BuildWindowsAccessRules(root, isolation.ExposePaths) {
			if rule.Path == "" {
				return fmt.Errorf("invalid windows access rule")
			}
		}
	}
	return nil
}

// Start prepares isolation for the command, starts it, and applies any
// post-start platform hooks required by the active backend.
func Start(cmd *exec.Cmd) error {
	if err := PrepareCommand(cmd); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		cleanupPendingPlatformResources(cmd)
		return err
	}
	isolation := CurrentConfig()
	root := ""
	if isolation.Enabled {
		var err error
		root, err = ResolveInstanceRoot()
		if err != nil {
			terminateStartedCommand(cmd)
			return err
		}
	}
	if err := postStartPlatformIsolation(cmd, isolation, root); err != nil {
		terminateStartedCommand(cmd)
		return err
	}
	return nil
}

// Run is the Start-and-Wait helper that keeps the same isolation behavior as
// Start while returning the command's final exit status.
func Run(cmd *exec.Cmd) error {
	if err := PrepareCommand(cmd); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		cleanupPendingPlatformResources(cmd)
		return err
	}
	isolation := CurrentConfig()
	root := ""
	if isolation.Enabled {
		var err error
		root, err = ResolveInstanceRoot()
		if err != nil {
			terminateStartedCommand(cmd)
			return err
		}
	}
	if err := postStartPlatformIsolation(cmd, isolation, root); err != nil {
		terminateStartedCommand(cmd)
		return err
	}
	return cmd.Wait()
}

func terminateStartedCommand(cmd *exec.Cmd) {
	cleanupPendingPlatformResources(cmd)
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

// PrepareCommand mutates the command in-place so it inherits the configured
// isolated environment before being started by the caller.
func PrepareCommand(cmd *exec.Cmd) error {
	isolation := CurrentConfig()
	if err := Preflight(); err != nil {
		return err
	}
	if isolation.Enabled {
		root, err := ResolveInstanceRoot()
		if err != nil {
			return err
		}
		ApplyUserEnv(cmd, root)
		if err := applyPlatformIsolation(cmd, isolation, root); err != nil {
			return err
		}
	}
	return nil
}

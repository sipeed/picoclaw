package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultEnvAllowlist is the set of environment variable names that are safe
// to propagate to child processes. Everything else is stripped to prevent
// accidental credential leakage.
//
// To add a new variable:
// 1. Add to this map if it's safe to pass through
// 2. Or add a prefix to defaultEnvAllowPrefixes for pattern matching
// Note: Do NOT add wildcard patterns like "*_API_KEY" here - use explicit names
// to avoid accidentally leaking secrets.
var DefaultEnvAllowlist = map[string]bool{
	"PATH":            true,
	"HOME":            true,
	"USER":            true,
	"LANG":            true,
	"SHELL":           true,
	"TERM":            true,
	"PWD":             true,
	"OLDPWD":          true,
	"HOSTNAME":        true,
	"LOGNAME":         true,
	"TZ":              true,
	"DISPLAY":         true,
	"TMPDIR":       true,
	"EDITOR":       true,
	"PAGER":        true,
	"HTTP_PROXY":   true,
	"HTTPS_PROXY":  true,
	"NO_PROXY":     true,
}

// defaultEnvAllowPrefixes are env var prefixes that are always allowed.
var defaultEnvAllowPrefixes = []string{
	"LC_",
}

// LLMBlocklist is the set of environment variable names that the LLM
// cannot override, even if passed via the env parameter. These vars
// control fundamental process behavior and could be exploited.
var LLMBlocklist = map[string]bool{
	"PATH":            true, // Could hijack command resolution
	"HOME":            true, // Could redirect file access
	"USER":            true, // Could impersonate user
	"LOGNAME":        true, // Could impersonate user
	"SHELL":           true, // Could change shell behavior
	"LD_PRELOAD":      true, // Could inject code
	"LD_LIBRARY_PATH": true, // Could hijack library resolution
	"LD_AUDIT":       true, // Could inject code
	"LD_DEBUG":       true, // Could leak info
}

// windowsEnvAllowlist contains additional variables needed on Windows.
var windowsEnvAllowlist = map[string]bool{
	"PATHEXT":     true,
	"SYSTEMROOT":  true,
	"SYSTEMDRIVE": true,
	"COMSPEC":     true,
	"APPDATA":     true,
	"USERPROFILE": true,
	"HOMEDRIVE":   true,
	"HOMEPATH":    true,
}

// BuildSanitizedEnv constructs a sanitized environment []string suitable for
// exec.Cmd.Env. It filters the inherited environment to only allowlisted variables.
//
// baseEnv is the inherited environment (e.g., from os.Environ() or cached).
// If nil, os.Environ() will be used for backwards compatibility.
// extraAllowlist adds additional variable names to the default allowlist.
// envSet provides explicit key=value pairs from config (override inherited).
// extraEnv provides additional key=value pairs from tool call (merged with envSet).
func BuildSanitizedEnv(baseEnv []string, extraAllowlist []string, envSet, extraEnv map[string]string) []string {
	// Use provided env or fall back to os.Environ
	inherited := baseEnv
	if inherited == nil {
		inherited = os.Environ()
	}

	allowed := make(map[string]bool, len(DefaultEnvAllowlist)+len(extraAllowlist)+len(windowsEnvAllowlist))
	for k := range DefaultEnvAllowlist {
		allowed[envKey(k)] = true
	}
	if runtime.GOOS == "windows" {
		for k := range windowsEnvAllowlist {
			allowed[envKey(k)] = true
		}
	}
	for _, k := range extraAllowlist {
		allowed[envKey(k)] = true
	}

	vars := make(map[string]string, len(allowed)+len(envSet)+len(extraEnv))

	for _, entry := range inherited {
		k, v, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		norm := envKey(k)
		if allowed[norm] || isAllowedPrefix(norm) {
			vars[norm] = v
		}
	}

	if envSet != nil {
		for k, v := range envSet {
			vars[envKey(k)] = v
		}
	}

	// Merge extraEnv (tool call) - highest priority
	// Filter against LLM blocklist to prevent override of sensitive vars
	if extraEnv != nil {
		for k, v := range extraEnv {
			if LLMBlocklist[envKey(k)] {
				continue // Skip blocked vars
			}
			vars[envKey(k)] = v
		}
	}

	// Convert to []string for exec.Cmd.Env
	result := make([]string, 0, len(vars))
	for k, v := range vars {
		result = append(result, k+"="+v)
	}
	return result
}

// envKey normalizes an environment variable name. On Windows, where env
// vars are case-insensitive, it uppercases the key so that "Path" and
// "PATH" map to the same entry. On other platforms it's a no-op.
func envKey(k string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(k)
	}
	return k
}

func isAllowedPrefix(name string) bool {
	for _, prefix := range defaultEnvAllowPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// WithPicoclawEnvVars ensures PICOCLAW_* vars are set in envSet.
// These are needed for child processes to locate config, workspace, etc.
func WithPicoclawEnvVars(envSet map[string]string, workspace string) map[string]string {
	if envSet == nil {
		envSet = make(map[string]string)
	}

	// Always compute PICOCLAW_* vars - priority: env var > default
	if v := os.Getenv("PICOCLAW_HOME"); v != "" {
		envSet["PICOCLAW_HOME"] = v
	} else if home, _ := os.UserHomeDir(); home != "" {
		envSet["PICOCLAW_HOME"] = filepath.Join(home, ".picoclaw")
	}

	if v := os.Getenv("PICOCLAW_CONFIG"); v != "" {
		envSet["PICOCLAW_CONFIG"] = v
	} else if home := envSet["PICOCLAW_HOME"]; home != "" {
		envSet["PICOCLAW_CONFIG"] = filepath.Join(home, "config.json")
	}

	// Workspace - this is the agent's working directory
	if workspace != "" {
		envSet["PICOCLAW_AGENT_WORKSPACE"] = workspace
	}

	if exe, err := os.Executable(); err == nil {
		envSet["PICOCLAW_EXE"] = exe
	}

	if v := os.Getenv("PICOCLAW_SERVICE_NAME"); v != "" {
		envSet["PICOCLAW_SERVICE_NAME"] = v
	} else {
		envSet["PICOCLAW_SERVICE_NAME"] = "picoclaw"
	}

	return envSet
}

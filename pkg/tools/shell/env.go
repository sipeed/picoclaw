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
	"PATH":        true,
	"HOME":        true,
	"USER":        true,
	"LANG":        true,
	"SHELL":       true,
	"TERM":        true,
	"PWD":         true,
	"OLDPWD":      true,
	"HOSTNAME":    true,
	"LOGNAME":     true,
	"TZ":          true,
	"DISPLAY":     true,
	"TMPDIR":      true,
	"EDITOR":      true,
	"PAGER":       true,
	"HTTP_PROXY":  true,
	"HTTPS_PROXY": true,
	"NO_PROXY":    true,

	// Locale
	"LC_ALL":            true,
	"LC_CTYPE":          true,
	"LC_MESSAGES":       true,
	"LC_MONETARY":       true,
	"LC_NUMERIC":        true,
	"LC_TIME":           true,
	"LC_PAPER":          true,
	"LC_NAME":           true,
	"LC_ADDRESS":        true,
	"LC_TELEPHONE":      true,
	"LC_MEASUREMENT":    true,
	"LC_IDENTIFICATION": true,
	"LC_COLLATE":        true,
}

// LLMBlocklist is the set of environment variable names that the LLM
// cannot override, even if passed via the env parameter. These vars
// control fundamental process behavior and could be exploited.
var LLMBlocklist = map[string]bool{
	"PATH":            true, // Could hijack command resolution
	"HOME":            true, // Could redirect file access
	"USER":            true, // Could impersonate user
	"LOGNAME":         true, // Could impersonate user
	"SHELL":           true, // Could change shell behavior
	"LD_PRELOAD":      true, // Could inject code
	"LD_LIBRARY_PATH": true, // Could hijack library resolution
	"LD_AUDIT":        true, // Could inject code
	"LD_DEBUG":        true, // Could leak info

	// PICOCLAW_* vars - controlled by the agent, not LLM
	"PICOCLAW_HOME":            true,
	"PICOCLAW_CONFIG":          true,
	"PICOCLAW_AGENT_WORKSPACE": true,
	"PICOCLAW_EXE":             true,
	"PICOCLAW_SERVICE_NAME":    true,
	"PICOCLAW_EXEC_TIME":       true,
	"PICOCLAW_EXEC_TIMEOUT":    true,
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

// WithAllowedEnv builds a map of allowed environment variables by looking them up.
// This is more efficient than filtering os.Environ() with string parsing.
// It starts with the provided env map, then adds allowed inherited vars (if not set).
// extraAllowlist adds to the default allowlist.
func WithAllowedEnv(envSet map[string]string, extraAllowlist []string) map[string]string {
	// Start with provided envSet map
	result := envSet
	if result == nil {
		result = make(map[string]string)
	}

	// Add default allowlist (only if not already set)
	for k := range DefaultEnvAllowlist {
		if _, exists := result[k]; !exists {
			if val := os.Getenv(k); val != "" {
				result[k] = val
			}
		}
	}
	// Add Windows-specific vars
	if runtime.GOOS == "windows" {
		for k := range windowsEnvAllowlist {
			if _, exists := result[k]; !exists {
				if val := os.Getenv(k); val != "" {
					result[k] = val
				}
			}
		}
	}
	// Add extra allowlist from config
	for _, k := range extraAllowlist {
		if _, exists := result[k]; !exists {
			if val := os.Getenv(k); val != "" {
				result[k] = val
			}
		}
	}

	return result
}

// LLMBlocklistPrefixes are env var prefixes that the LLM cannot override.
var LLMBlocklistPrefixes = []string{
	"PICOCLAW_",
}

// isBlocked returns true if the key is in the blocklist or matches a blocked prefix.
func isBlocked(key string) bool {
	norm := envKey(key)
	if LLMBlocklist[norm] {
		return true
	}
	for _, prefix := range LLMBlocklistPrefixes {
		if strings.HasPrefix(norm, prefix) {
			return true
		}
	}
	return false
}

// MergeEnvVars merges multiple env sources into a map.
// baseEnv is the cached map from AllowedEnv.
// envSet provides explicit key=value pairs (config, not filtered).
// extraEnv provides additional key=value pairs from LLM (filtered by blocklist).
func MergeEnvVars(baseEnv map[string]string, envSet, extraEnv map[string]string) map[string]string {
	vars := make(map[string]string, len(baseEnv)+len(envSet)+len(extraEnv))

	// Start with base env (already filtered)
	for k, v := range baseEnv {
		vars[envKey(k)] = v
	}

	// Add envSet (config-provided, not filtered)
	if envSet != nil {
		for k, v := range envSet {
			vars[envKey(k)] = v
		}
	}

	// Merge extraEnv (LLM-provided) - filtered by blocklist
	if extraEnv != nil {
		for k, v := range extraEnv {
			if isBlocked(k) {
				continue // Skip blocked vars
			}
			vars[envKey(k)] = v
		}
	}

	return vars
}

// MapToEnvSlice converts a map of environment variables to a []string
// in the format "KEY=value" suitable for exec.Cmd.Env.
func MapToEnvSlice(vars map[string]string) []string {
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

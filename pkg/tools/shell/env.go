package shell

import (
	"os"
	"runtime"
	"strings"

	"mvdan.cc/sh/v3/expand"
)

// DefaultEnvAllowlist is the set of environment variable names that are safe
// to propagate to child processes. Everything else is stripped to prevent
// accidental credential leakage.
var DefaultEnvAllowlist = map[string]bool{
	"PATH":     true,
	"HOME":     true,
	"USER":     true,
	"LANG":     true,
	"SHELL":    true,
	"TERM":     true,
	"PWD":      true,
	"OLDPWD":   true,
	"HOSTNAME": true,
	"LOGNAME":  true,
	"TZ":       true,
	"DISPLAY":  true,
	"TMPDIR":   true,
	"EDITOR":   true,
	"PAGER":    true,
}

// defaultEnvAllowPrefixes are env var prefixes that are always allowed.
var defaultEnvAllowPrefixes = []string{
	"LC_",
}

// windowsEnvAllowlist contains additional variables needed on Windows.
// Without SYSTEMROOT, many Windows system calls fail. PATHEXT is required
// for correct executable lookup.
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

// BuildSanitizedEnv constructs an expand.Environ from the current process
// environment, filtering to only allowlisted variables.
//
// extraAllowlist adds additional variable names to the default allowlist.
// envSet provides explicit key=value pairs that override any inherited value.
func BuildSanitizedEnv(extraAllowlist []string, envSet map[string]string) expand.Environ {
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

	vars := make(map[string]string, len(allowed)+len(envSet))

	for _, entry := range os.Environ() {
		k, v, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		norm := envKey(k)
		if allowed[norm] || isAllowedPrefix(norm) {
			vars[norm] = v
		}
	}

	for k, v := range envSet {
		vars[envKey(k)] = v
	}

	return &sanitizedEnv{vars: vars}
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

// sanitizedEnv implements expand.Environ backed by a simple map.
type sanitizedEnv struct {
	vars map[string]string
}

func (e *sanitizedEnv) Get(name string) expand.Variable {
	val, ok := e.vars[envKey(name)]
	if !ok {
		return expand.Variable{}
	}
	return expand.Variable{
		Set:      true,
		Exported: true,
		Kind:     expand.String,
		Str:      val,
	}
}

func (e *sanitizedEnv) Each(fn func(name string, vr expand.Variable) bool) {
	for k, v := range e.vars {
		vr := expand.Variable{
			Set:      true,
			Exported: true,
			Kind:     expand.String,
			Str:      v,
		}
		if !fn(k, vr) {
			return
		}
	}
}

package shellguard

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Decision is a structured command validation result.
type Decision struct {
	Allowed  bool
	Reason   string
	Category string
}

// Config contains the policy inputs needed to validate a shell command.
type Config struct {
	DenyPatterns        []*regexp.Regexp
	AllowPatterns       []*regexp.Regexp
	CustomAllowPatterns []*regexp.Regexp
	AllowedPathPatterns []*regexp.Regexp
	RestrictToWorkspace bool
}

// Validator validates shell commands before execution.
type Validator struct {
	config Config
}

// New returns a reusable shell command validator.
func New(config Config) *Validator {
	return &Validator{config: config}
}

// Validate applies deny/allow/path checks without executing the command.
func (v *Validator) Validate(command, cwd string) Decision {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)
	sanitizedForGuards := StripQuotedHeredocBodies(cmd)
	lowerForDeny := strings.ToLower(sanitizedForGuards)

	if !v.explicitlyAllowed(lower) {
		for _, pattern := range v.config.DenyPatterns {
			if pattern.MatchString(lowerForDeny) {
				return Decision{
					Allowed:  false,
					Reason:   "Command blocked by safety guard (dangerous pattern detected)",
					Category: "dangerous_pattern",
				}
			}
		}
	}

	if len(v.config.AllowPatterns) > 0 && !matchesAny(v.config.AllowPatterns, lower) {
		return Decision{
			Allowed:  false,
			Reason:   "Command blocked by safety guard (not in allowlist)",
			Category: "not_allowlisted",
		}
	}

	if v.config.RestrictToWorkspace {
		if decision := v.validateWorkspacePaths(sanitizedForGuards, cwd); !decision.Allowed {
			return decision
		}
	}

	return Decision{Allowed: true, Reason: "allowed", Category: "allowed"}
}

func (v *Validator) explicitlyAllowed(lowerCommand string) bool {
	return matchesAny(v.config.CustomAllowPatterns, lowerCommand)
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func (v *Validator) validateWorkspacePaths(command, cwd string) Decision {
	if regexp.MustCompile(`\.\.(?:[\\/]\.\.)*[\\/]`).MatchString(command) {
		return Decision{
			Allowed:  false,
			Reason:   "Command blocked by safety guard (path traversal detected)",
			Category: "path_traversal",
		}
	}

	cwdPath, err := filepath.Abs(cwd)
	if err != nil {
		return Decision{Allowed: true, Reason: "allowed", Category: "allowed"}
	}

	if runtime.GOOS == "windows" {
		command = ExpandPowerShellEnvVars(command)
		if home, err := os.UserHomeDir(); err == nil {
			command = strings.ReplaceAll(command, "~", filepath.FromSlash(home))
		}
	}

	for _, loc := range absolutePathPattern.FindAllStringIndex(command, -1) {
		raw := command[loc[0]:loc[1]]
		if skipPathCandidate(command, raw, loc[0]) {
			continue
		}

		p, err := filepath.Abs(raw)
		if err != nil {
			continue
		}

		if runtime.GOOS == "windows" {
			p = strings.TrimPrefix(p, `\\?\`)
			if idx := strings.Index(p, ":"); idx > 1 {
				p = p[:idx]
			}
		}

		resolved, resolveErr := filepath.EvalSymlinks(p)
		if resolveErr == nil {
			p = resolved
		}

		if safePaths[p] || isAllowedPath(p, v.config.AllowedPathPatterns) {
			continue
		}

		rel, err := filepath.Rel(cwdPath, p)
		if err != nil {
			continue
		}

		if strings.HasPrefix(rel, "..") {
			return Decision{
				Allowed:  false,
				Reason:   "Command blocked by safety guard (path outside working dir)",
				Category: "path_outside_working_dir",
			}
		}
	}

	return Decision{Allowed: true, Reason: "allowed", Category: "allowed"}
}

func skipPathCandidate(command, raw string, start int) bool {
	if strings.HasPrefix(raw, "/") && start > 0 {
		prev := command[start-1]
		if (prev >= 'a' && prev <= 'z') ||
			(prev >= 'A' && prev <= 'Z') ||
			(prev >= '0' && prev <= '9') ||
			prev == '_' || prev == '.' || prev == '-' {
			return true
		}
	}

	if strings.HasPrefix(raw, "//") && start > 0 {
		before := command[:start]
		for _, scheme := range webSchemes {
			if strings.HasSuffix(before, scheme) {
				return true
			}
		}
	}

	return false
}

func isAllowedPath(path string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

var (
	absolutePathPattern = regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)

	safePaths = map[string]bool{
		"/dev/null":    true,
		"/dev/zero":    true,
		"/dev/random":  true,
		"/dev/urandom": true,
		"/dev/stdin":   true,
		"/dev/stdout":  true,
		"/dev/stderr":  true,
	}

	webSchemes = []string{"http:", "https:", "ftp:", "ftps:", "sftp:", "ssh:", "git:"}

	quotedHeredocStartPattern = regexp.MustCompile(`<<-?\s*(?:'([^'\s]+)'|"([^"\s]+)")`)
)

// StripQuotedHeredocBodies removes quoted heredoc bodies before deny-pattern checks.
func StripQuotedHeredocBodies(command string) string {
	sanitized := command
	searchFrom := 0

	for {
		loc := quotedHeredocStartPattern.FindStringSubmatchIndex(sanitized[searchFrom:])
		if loc == nil {
			return sanitized
		}

		matchEnd := searchFrom + loc[1]
		delimStart := searchFrom + loc[2]
		delimEnd := searchFrom + loc[3]
		if delimStart == searchFrom-1 || delimEnd == searchFrom-1 {
			delimStart = searchFrom + loc[4]
			delimEnd = searchFrom + loc[5]
		}
		delim := sanitized[delimStart:delimEnd]
		allowTabs := strings.Contains(sanitized[searchFrom+loc[0]:matchEnd], "<<-")

		newlineRel := strings.IndexByte(sanitized[matchEnd:], '\n')
		if newlineRel == -1 {
			return sanitized
		}
		bodyStart := matchEnd + newlineRel + 1
		lineStart := bodyStart
		foundEnd := false

		for lineStart <= len(sanitized) {
			lineEnd := len(sanitized)
			if nextNewline := strings.IndexByte(sanitized[lineStart:], '\n'); nextNewline >= 0 {
				lineEnd = lineStart + nextNewline
			}

			line := sanitized[lineStart:lineEnd]
			compare := strings.TrimSuffix(line, "\r")
			if allowTabs {
				compare = strings.TrimLeft(compare, "\t")
			}
			if compare == delim {
				sanitized = sanitized[:bodyStart] + "[quoted heredoc omitted]\n" + sanitized[lineStart:]
				searchFrom = bodyStart + len("[quoted heredoc omitted]\n")
				foundEnd = true
				break
			}

			if lineEnd == len(sanitized) {
				lineStart = len(sanitized) + 1
			} else {
				lineStart = lineEnd + 1
			}
		}

		if !foundEnd {
			return sanitized
		}
	}
}

// ExpandPowerShellEnvVars expands PowerShell/CMD environment variable syntax.
func ExpandPowerShellEnvVars(cmd string) string {
	rePs := regexp.MustCompile(`\$\{?env:(\w+)\}?`)
	cmd = rePs.ReplaceAllStringFunc(cmd, func(match string) string {
		varName := rePs.FindStringSubmatch(match)[1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})

	reCmd := regexp.MustCompile(`%([^%]+)%`)
	return reCmd.ReplaceAllStringFunc(cmd, func(match string) string {
		varName := reCmd.FindStringSubmatch(match)[1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})
}

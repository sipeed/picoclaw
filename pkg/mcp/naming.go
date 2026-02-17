package mcp

import "strings"

const qualifiedNameMaxLen = 64

// QualifiedToolName creates a stable, provider-safe function name.
func QualifiedToolName(serverName, toolName string) string {
	prefix := "mcp_" + sanitizeName(serverName) + "__"
	tool := sanitizeName(toolName)
	maxToolLen := qualifiedNameMaxLen - len(prefix)
	if maxToolLen <= 0 {
		return prefix[:qualifiedNameMaxLen]
	}
	if len(tool) > maxToolLen {
		tool = tool[:maxToolLen]
	}
	return prefix + tool
}

func sanitizeName(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "unknown"
	}

	var b strings.Builder
	b.Grow(len(trimmed))

	lastUnderscore := false
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			b.WriteByte(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}

	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "unknown"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "t_" + s
	}
	return s
}

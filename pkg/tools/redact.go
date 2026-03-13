package tools

import (
	"regexp"
	"strings"
)

// secretPatterns matches common API key and credential formats.
// Each pattern is compiled once at init and reused across calls.
var secretPatterns = []*regexp.Regexp{
	// OpenAI / OpenAI-compatible (sk-...)
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
	// OpenAI project keys (sk-proj-...)
	regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`),
	// Anthropic (sk-ant-...)
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`),
	// OpenRouter (sk-or-v1-...)
	regexp.MustCompile(`sk-or-v1-[A-Za-z0-9_-]{20,}`),
	// Google AI / Gemini (AIza...)
	regexp.MustCompile(`AIza[A-Za-z0-9_-]{30,}`),
	// GitHub tokens (ghp_, gho_, ghu_, ghs_, ghr_)
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{30,}`),
	// Slack tokens (xoxb-, xoxp-, xoxs-, xoxa-)
	regexp.MustCompile(`xox[bpsa]-[A-Za-z0-9-]{20,}`),
	// Discord bot tokens (base64.base64.base64)
	regexp.MustCompile(`[MN][A-Za-z0-9]{23,}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,}`),
	// Generic long bearer/api tokens (40+ hex or base64 chars)
	regexp.MustCompile(`\b[A-Fa-f0-9]{40,}\b`),
	// AWS access keys (AKIA...)
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),
	// AWS secret keys (40 char base64-ish after known prefixes)
	regexp.MustCompile(`(?i)aws[_\-]?secret[_\-]?access[_\-]?key["'\s:=]+[A-Za-z0-9/+=]{40}`),
}

// jsonSecretFields matches JSON keys that typically hold secrets.
// Captures the key and value so we can redact the value in-place.
var jsonSecretFields = regexp.MustCompile(
	`(?i)("(?:api_key|apikey|secret|token|password|access_token|auth_token|bot_token|app_token|channel_secret|channel_access_token|corp_secret|client_secret|verification_token|nickserv_password|sasl_password)")\s*:\s*"([^"]{8,})"`,
)

// RedactSecrets replaces known secret patterns in s with a redacted placeholder.
// It operates on both well-known token formats and JSON secret field values.
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}

	// Pass 1: redact JSON secret field values (preserves key for context).
	// e.g. "api_key": "sk-abc123..." → "api_key": "[REDACTED]"
	result := jsonSecretFields.ReplaceAllString(s, `$1: "[REDACTED]"`)

	// Pass 2: redact well-known token patterns anywhere in the text.
	for _, pat := range secretPatterns {
		result = pat.ReplaceAllStringFunc(result, redactToken)
	}

	return result
}

// redactToken replaces a matched token with a hint showing the prefix.
func redactToken(token string) string {
	if len(token) <= 8 {
		return "[REDACTED]"
	}
	// Show first 4 chars for identification, redact the rest.
	return token[:4] + "..." + "[REDACTED]"
}

// SanitizeResult applies secret redaction to both ForLLM and ForUser fields
// of a ToolResult. Returns the same pointer for convenience.
func SanitizeResult(r *ToolResult) *ToolResult {
	if r == nil {
		return r
	}
	r.ForLLM = RedactSecrets(r.ForLLM)
	if r.ForUser != "" {
		r.ForUser = RedactSecrets(r.ForUser)
	}
	// Also scrub error messages — provider errors sometimes include keys.
	if r.Err != nil {
		cleaned := RedactSecrets(r.Err.Error())
		if cleaned != r.Err.Error() {
			r.Err = &redactedError{msg: cleaned}
		}
	}
	return r
}

// redactedError wraps a redacted error message.
type redactedError struct {
	msg string
}

func (e *redactedError) Error() string {
	return e.msg
}

// ContainsSecret checks whether a string contains any known secret pattern.
// Useful for pre-flight validation before logging or outputting content.
func ContainsSecret(s string) bool {
	if jsonSecretFields.MatchString(s) {
		return true
	}
	for _, pat := range secretPatterns {
		if pat.MatchString(s) {
			return true
		}
	}
	return false
}

// envSecretKeys lists environment variable name patterns whose values should
// never appear in tool output. Used by the shell tool guard.
var envSecretKeys = []string{
	"API_KEY", "SECRET", "TOKEN", "PASSWORD", "CREDENTIAL",
	"ACCESS_KEY", "PRIVATE_KEY", "AUTH",
}

// IsSecretEnvVar returns true if the environment variable name looks like it
// holds a secret value (case-insensitive substring match).
func IsSecretEnvVar(name string) bool {
	upper := strings.ToUpper(name)
	for _, key := range envSecretKeys {
		if strings.Contains(upper, key) {
			return true
		}
	}
	return false
}

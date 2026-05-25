package common

import "strings"

// ModelOmitsTemperature reports whether an LLM model rejects requests that
// include the temperature parameter.
//
// Affected models:
//   - Anthropic claude-opus-4-7 family: returns HTTP 400 invalid_request_error
//     "temperature is deprecated for this model".
//   - OpenAI gpt-5.5 family: returns HTTP 400
//     "Unsupported value: 'temperature' does not support … with this model.
//     Only the default (1) value is supported." Setting temperature=1 works,
//     but omitting it is simpler and matches the documented default.
//
// Checks are prefix matches so dated variants (e.g. claude-opus-4-7-20260601,
// gpt-5.5-20260601) are covered without further code changes.
func ModelOmitsTemperature(model string) bool {
	lower := strings.ToLower(strings.TrimSpace(model))
	if lower == "" {
		return false
	}
	switch {
	case strings.HasPrefix(lower, "claude-opus-4-7"):
		return true
	case strings.HasPrefix(lower, "gpt-5.5"):
		return true
	}
	return false
}

// NormalizeBaseURL ensures the Anthropic base URL is properly formatted.
// It removes a trailing /v1 suffix if present (to avoid duplication), then
// re-appends /v1 when appendV1Suffix is true. An empty apiBase falls back to
// defaultBaseURL.
func NormalizeBaseURL(apiBase, defaultBaseURL string, appendV1Suffix bool) string {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		return defaultBaseURL
	}

	base = strings.TrimRight(base, "/")
	if before, ok := strings.CutSuffix(base, "/v1"); ok {
		base = before
	}
	if base == "" {
		return defaultBaseURL
	}

	if appendV1Suffix {
		return base + "/v1"
	}
	return base
}

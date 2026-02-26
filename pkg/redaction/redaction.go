// Package redaction provides privacy protection through sensitive data redaction.
// It automatically detects and masks API keys, tokens, passwords, and PII.
package redaction

import (
	"regexp"
	"strings"
	"sync"
)

// Config holds redaction configuration.
type Config struct {
	// Enabled controls whether redaction is active.
	Enabled bool `json:"enabled"`

	// RedactAPIKeys redacts API keys and tokens.
	RedactAPIKeys bool `json:"redact_api_keys"`

	// RedactPasswords redacts password fields.
	RedactPasswords bool `json:"redact_passwords"`

	// RedactEmails redacts email addresses.
	RedactEmails bool `json:"redact_emails"`

	// RedactPhoneNumbers redacts phone numbers.
	RedactPhoneNumbers bool `json:"redact_phone_numbers"`

	// RedactIPAddresses redacts IP addresses.
	RedactIPAddresses bool `json:"redact_ip_addresses"`

	// CustomPatterns allows additional regex patterns to redact.
	CustomPatterns []string `json:"custom_patterns"`

	// Replacement is the string used to replace sensitive data.
	Replacement string `json:"replacement"`
}

// DefaultConfig returns the default redaction configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:            true,
		RedactAPIKeys:      true,
		RedactPasswords:    true,
		RedactEmails:       true,
		RedactPhoneNumbers: true,
		RedactIPAddresses:  false, // Off by default as it may redact useful info
		Replacement:        "[REDACTED]",
	}
}

// Redactor provides sensitive data redaction capabilities.
type Redactor struct {
	config          Config
	compiledCustom  []*regexp.Regexp
	compiledBuiltin map[string]*regexp.Regexp
	mu              sync.RWMutex
}

// NewRedactor creates a new Redactor with the given configuration.
func NewRedactor(config Config) *Redactor {
	r := &Redactor{
		config:          config,
		compiledBuiltin: make(map[string]*regexp.Regexp),
	}

	// Compile builtin patterns
	r.compileBuiltinPatterns()

	// Compile custom patterns
	if len(config.CustomPatterns) > 0 {
		r.compiledCustom = make([]*regexp.Regexp, 0, len(config.CustomPatterns))
		for _, pattern := range config.CustomPatterns {
			re, err := regexp.Compile(pattern)
			if err == nil {
				r.compiledCustom = append(r.compiledCustom, re)
			}
		}
	}

	return r
}

// compileBuiltinPatterns compiles the builtin redaction patterns.
func (r *Redactor) compileBuiltinPatterns() {
	// API Key patterns - various formats
	r.compiledBuiltin["api_key"] = regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)\s*[=:]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`)
	r.compiledBuiltin["bearer_token"] = regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9_\-\.]{20,})`)
	r.compiledBuiltin["auth_token"] = regexp.MustCompile(`(?i)(auth[_-]?token|access[_-]?token|refresh[_-]?token)\s*[=:]\s*['"]?([a-zA-Z0-9_\-\.]{20,})['"]?`)
	r.compiledBuiltin["secret_key"] = regexp.MustCompile(`(?i)(secret[_-]?key|secretkey|private[_-]?key)\s*[=:]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`)

	// OpenAI-style keys
	r.compiledBuiltin["openai_key"] = regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`)
	r.compiledBuiltin["anthropic_key"] = regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-]{20,}`)

	// Generic token patterns
	r.compiledBuiltin["jwt"] = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`)
	r.compiledBuiltin["uuid"] = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

	// Password patterns
	r.compiledBuiltin["password"] = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*['"]?([^'"\s]{4,})['"]?`)

	// Email pattern
	r.compiledBuiltin["email"] = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// Phone number patterns (various formats)
	r.compiledBuiltin["phone_intl"] = regexp.MustCompile(`\+\d{1,3}[\s\-]?\d{1,4}[\s\-]?\d{1,4}[\s\-]?\d{1,9}`)
	r.compiledBuiltin["phone_us"] = regexp.MustCompile(`\(\d{3}\)\s*\d{3}[\s\-]?\d{4}`)
	r.compiledBuiltin["phone_simple"] = regexp.MustCompile(`\b\d{3}[\s\-]?\d{3}[\s\-]?\d{4}\b`)

	// IP Address patterns
	r.compiledBuiltin["ipv4"] = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
	r.compiledBuiltin["ipv6"] = regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)

	// AWS keys
	r.compiledBuiltin["aws_access_key"] = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	r.compiledBuiltin["aws_secret"] = regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[=:]\s*['"]?([a-zA-Z0-9/+=]{40})['"]?`)

	// Generic secrets in JSON/config
	r.compiledBuiltin["json_secret"] = regexp.MustCompile(`"(?:api_key|apikey|secret|password|token|private_key)"\s*:\s*"([^"]+)"`)
}

// Redact applies all configured redaction rules to the input string.
func (r *Redactor) Redact(input string) string {
	if !r.config.Enabled {
		return input
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := input

	// Redact API keys
	if r.config.RedactAPIKeys {
		result = r.redactPatterns(result,
			"api_key", "bearer_token", "auth_token", "secret_key",
			"openai_key", "anthropic_key", "jwt", "aws_access_key", "aws_secret",
		)
		// Redact JSON secrets with special handling
		result = r.redactJSONSecrets(result)
	}

	// Redact passwords
	if r.config.RedactPasswords {
		result = r.redactPatterns(result, "password")
	}

	// Redact emails
	if r.config.RedactEmails {
		result = r.redactPatternsWithPartial(result, "email", r.maskEmail)
	}

	// Redact phone numbers
	if r.config.RedactPhoneNumbers {
		result = r.redactPatterns(result, "phone_intl", "phone_us", "phone_simple")
	}

	// Redact IP addresses
	if r.config.RedactIPAddresses {
		result = r.redactPatterns(result, "ipv4", "ipv6")
	}

	// Apply custom patterns
	for _, re := range r.compiledCustom {
		result = re.ReplaceAllString(result, r.config.Replacement)
	}

	return result
}

// redactPatterns applies redaction for the specified patterns.
func (r *Redactor) redactPatterns(input string, patternNames ...string) string {
	result := input
	for _, name := range patternNames {
		if re, ok := r.compiledBuiltin[name]; ok {
			// For patterns with capture groups, only redact the captured content
			result = re.ReplaceAllStringFunc(result, func(match string) string {
				// Find submatches
				submatches := re.FindStringSubmatch(match)
				if len(submatches) > 1 {
					// Redact only the captured group(s), preserve the rest
					redacted := match
					for i := len(submatches) - 1; i >= 1; i-- {
						if submatches[i] != "" {
							redacted = strings.Replace(redacted, submatches[i], r.config.Replacement, 1)
						}
					}
					return redacted
				}
				return r.config.Replacement
			})
		}
	}
	return result
}

// redactPatternsWithPartial applies partial redaction (like masking) for patterns.
func (r *Redactor) redactPatternsWithPartial(input string, patternName string, maskFn func(string) string) string {
	re, ok := r.compiledBuiltin[patternName]
	if !ok {
		return input
	}

	return re.ReplaceAllStringFunc(input, func(match string) string {
		return maskFn(match)
	})
}

// redactJSONSecrets handles JSON key-value pairs specially.
func (r *Redactor) redactJSONSecrets(input string) string {
	re := r.compiledBuiltin["json_secret"]
	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) > 1 {
			return strings.Replace(match, submatches[1], r.config.Replacement, 1)
		}
		return match
	})
}

// maskEmail masks an email address, showing only first char and domain.
func (r *Redactor) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return r.config.Replacement
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return string(local[0]) + "***@" + domain
	}

	return string(local[0]) + "***@" + domain
}

// RedactFields redacts sensitive values in a map.
func (r *Redactor) RedactFields(fields map[string]any) map[string]any {
	if !r.config.Enabled {
		return fields
	}

	result := make(map[string]any, len(fields))
	for k, v := range fields {
		// Check if key name suggests sensitive data
		lowerKey := strings.ToLower(k)
		if r.isSensitiveKey(lowerKey) {
			result[k] = r.config.Replacement
		} else {
			// Recursively redact string values
			switch val := v.(type) {
			case string:
				result[k] = r.Redact(val)
			case map[string]any:
				result[k] = r.RedactFields(val)
			default:
				result[k] = v
			}
		}
	}
	return result
}

// isSensitiveKey checks if a key name suggests sensitive data.
func (r *Redactor) isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"api_key", "apikey", "api_secret",
		"secret", "secret_key", "private_key",
		"token", "access_token", "refresh_token", "auth_token",
		"credential", "credentials",
		"api_key_id", "secret_access_key",
	}

	for _, sk := range sensitiveKeys {
		if strings.Contains(key, sk) {
			return true
		}
	}
	return false
}

// SetEnabled enables or disables redaction at runtime.
func (r *Redactor) SetEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config.Enabled = enabled
}

// AddCustomPattern adds a custom redaction pattern at runtime.
func (r *Redactor) AddCustomPattern(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	r.compiledCustom = append(r.compiledCustom, re)
	return nil
}

// Global redactor instance with default config
var globalRedactor = NewRedactor(DefaultConfig())

// Redact applies redaction using the global redactor.
func Redact(input string) string {
	return globalRedactor.Redact(input)
}

// RedactFields redacts fields using the global redactor.
func RedactFields(fields map[string]any) map[string]any {
	return globalRedactor.RedactFields(fields)
}

// SetGlobalConfig sets the configuration for the global redactor.
func SetGlobalConfig(config Config) {
	globalRedactor = NewRedactor(config)
}

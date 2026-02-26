// Package injection provides prompt injection defense mechanisms.
// It detects and mitigates attempts to manipulate LLM behavior through user input.
package injection

import (
	"regexp"
	"strings"
	"sync"
)

// Config holds prompt injection defense configuration.
type Config struct {
	Enabled                 bool
	SanitizeUserInput       bool
	DetectInjectionPatterns bool
	CustomBlockPatterns     []string
}

// DefaultConfig returns the default prompt injection defense configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:                 true,
		SanitizeUserInput:       true,
		DetectInjectionPatterns: true,
		CustomBlockPatterns:     []string{},
	}
}

// Defender provides prompt injection defense capabilities.
type Defender struct {
	config           Config
	compiledPatterns []*regexp.Regexp
	mu               sync.RWMutex
}

// InjectionResult represents the result of injection detection.
type InjectionResult struct {
	Detected        bool     `json:"detected"`
	Confidence      float64  `json:"confidence"` // 0.0 to 1.0
	MatchedPatterns []string `json:"matched_patterns,omitempty"`
	SanitizedInput  string   `json:"sanitized_input,omitempty"`
}

// NewDefender creates a new prompt injection defender.
func NewDefender(config Config) *Defender {
	d := &Defender{
		config: config,
	}

	// Compile default patterns
	d.compileDefaultPatterns()

	// Compile custom patterns
	if len(config.CustomBlockPatterns) > 0 {
		for _, pattern := range config.CustomBlockPatterns {
			re, err := regexp.Compile(pattern)
			if err == nil {
				d.compiledPatterns = append(d.compiledPatterns, re)
			}
		}
	}

	return d
}

// compileDefaultPatterns compiles the default injection detection patterns.
func (d *Defender) compileDefaultPatterns() {
	// Common prompt injection patterns
	patterns := []string{
		// System prompt override attempts
		`(?i)ignore\s+(all\s+)?(previous|above)\s*(instructions|prompts?|rules)?`,
		`(?i)forget\s+(everything|all|previous)`,
		`(?i)disregard\s+(all|any|previous)\s*(instructions|rules)?`,
		`(?i)system\s*:\s*`,
		`(?i)assistant\s*:\s*`,
		`(?i)user\s*:\s*`,

		// Role manipulation
		`(?i)you\s+are\s+now\s+`,
		`(?i)act\s+as\s+(if|a|an)\s+`,
		`(?i)pretend\s+(to\s+be|that)\s+`,
		`(?i)role[\s-]*play\s+as`,
		`(?i)simulate\s+(being|a|an)\s+`,

		// Instruction injection
		`(?i)new\s+instructions?\s*:`,
		`(?i)override\s+(previous|default)\s*(instructions|settings)`,
		`(?i)change\s+(your|the)\s+(behavior|mode|persona)`,

		// Output manipulation
		`(?i)print\s+(exactly|the\s+following)`,
		`(?i)output\s+(only|exactly|the\s+following)`,
		`(?i)respond\s+(only\s+with|with\s+exactly)`,
		`(?i)repeat\s+(after\s+me|the\s+following)`,

		// Delimiter injection
		`-{3,}`,
		`={3,}`,
		`#{3,}`,
		`\[\[`,
		`\]\]`,
		`<<`,
		`>>`,

		// Escape attempts
		`(?i)escape\s*(the\s+)?(context|prompt|rules)`,
		`(?i)break\s*(out\s+of|the\s+)?(character|role|context)`,
		`(?i)bypass\s*(the\s+)?(filter|restrictions?|rules)`,

		// Common jailbreak phrases
		`(?i)do\s+anything\s+now`,
		`(?i)developer\s+mode`,
		`(?i)debug\s+mode`,
		`(?i)admin\s+mode`,
		`(?i)sudo\s+mode`,
		`(?i)dan\s+(mode|prompt)`,

		// Tool/function manipulation
		`(?i)(call|invoke|execute)\s+(tool|function)\s*:`,
		`(?i)use\s+(the\s+)?tool\s+`,

		// Special tokens
		`<\|`,
		`\|>`,
		`<\s*/?\s*(system|user|assistant|im_start|im_end)\s*>`,

		// Base64/encoded content hints
		`(?i)(base64|decode|decrypt)\s*:`,

		// Common attack patterns
		`(?i)prompt\s+injection`,
		`(?i)jailbreak`,
	}

	d.compiledPatterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			d.compiledPatterns = append(d.compiledPatterns, re)
		}
	}
}

// Detect checks if the input contains potential prompt injection attempts.
func (d *Defender) Detect(input string) InjectionResult {
	if !d.config.Enabled || !d.config.DetectInjectionPatterns {
		return InjectionResult{
			Detected:       false,
			Confidence:     0,
			SanitizedInput: input,
		}
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var matchedPatterns []string
	confidence := 0.0

	// Check each pattern
	for _, re := range d.compiledPatterns {
		if re.MatchString(input) {
			matchedPatterns = append(matchedPatterns, re.String())
			confidence += 0.1 // Each match adds to confidence
		}
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Additional heuristics
	confidence = d.applyHeuristics(input, confidence, &matchedPatterns)

	// Lower threshold for detection - any single pattern match should trigger
	detected := confidence >= 0.1 || len(matchedPatterns) > 0

	return InjectionResult{
		Detected:        detected,
		Confidence:      confidence,
		MatchedPatterns: matchedPatterns,
		SanitizedInput:  d.sanitize(input),
	}
}

// applyHeuristics applies additional detection heuristics.
func (d *Defender) applyHeuristics(input string, confidence float64, matchedPatterns *[]string) float64 {
	// Each matched pattern adds significant confidence
	confidence += float64(len(*matchedPatterns)) * 0.2

	// Check for unusual repetition
	words := strings.Fields(input)
	if len(words) > 10 {
		wordCount := make(map[string]int)
		for _, w := range words {
			wordCount[strings.ToLower(w)]++
		}
		for w, count := range wordCount {
			if count > 5 && len(w) > 3 {
				confidence += 0.1
				*matchedPatterns = append(*matchedPatterns, "repetition_heuristic:"+w)
			}
		}
	}

	// Check for mixed language/scripts (potential obfuscation)
	hasLatin := false
	hasNonLatin := false
	for _, r := range input {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			hasLatin = true
		} else if r > 127 {
			hasNonLatin = true
		}
	}
	if hasLatin && hasNonLatin && len(input) < 100 {
		confidence += 0.05
	}

	// Check for unusual capitalization patterns
	upperCount := 0
	lowerCount := 0
	for _, r := range input {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		} else if r >= 'a' && r <= 'z' {
			lowerCount++
		}
	}
	if upperCount > 0 && lowerCount > 0 {
		ratio := float64(upperCount) / float64(upperCount+lowerCount)
		if ratio > 0.7 || ratio < 0.3 {
			confidence += 0.03
		}
	}

	return confidence
}

// sanitize applies sanitization to user input.
func (d *Defender) sanitize(input string) string {
	if !d.config.Enabled || !d.config.SanitizeUserInput {
		return input
	}

	// Remove or escape potentially dangerous content
	result := input

	// Escape XML-like tags
	result = regexp.MustCompile(`<([^>]+)>`).ReplaceAllString(result, `&lt;$1&gt;`)

	// Normalize whitespace
	result = strings.TrimSpace(result)

	// Remove null bytes and control characters
	result = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, result)

	return result
}

// WrapInBoundary wraps user input in structured boundaries to prevent injection.
func (d *Defender) WrapInBoundary(input string) string {
	if !d.config.Enabled {
		return input
	}

	// Use XML-style boundaries that are clear and parseable
	// This helps the model distinguish user content from instructions
	return `<user_input>
` + input + `
</user_input>`
}

// SanitizeAndWrap combines sanitization and boundary wrapping.
func (d *Defender) SanitizeAndWrap(input string) (string, InjectionResult) {
	result := d.Detect(input)
	sanitized := d.sanitize(input)
	wrapped := d.WrapInBoundary(sanitized)
	return wrapped, result
}

// AddCustomPattern adds a custom detection pattern.
func (d *Defender) AddCustomPattern(pattern string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	d.compiledPatterns = append(d.compiledPatterns, re)
	return nil
}

// SetEnabled enables or disables the defender.
func (d *Defender) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config.Enabled = enabled
}

// IsEnabled returns whether the defender is enabled.
func (d *Defender) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Enabled
}

// Global defender instance
var globalDefender *Defender
var globalOnce sync.Once

// InitGlobal initializes the global defender.
func InitGlobal(config Config) {
	globalOnce.Do(func() {
		globalDefender = NewDefender(config)
	})
}

// Detect uses the global defender to detect injection.
func Detect(input string) InjectionResult {
	if globalDefender == nil {
		return InjectionResult{
			Detected:       false,
			Confidence:     0,
			SanitizedInput: input,
		}
	}
	return globalDefender.Detect(input)
}

// Sanitize uses the global defender to sanitize input.
func Sanitize(input string) string {
	if globalDefender == nil {
		return input
	}
	return globalDefender.sanitize(input)
}

// WrapInBoundary uses the global defender to wrap input.
func WrapInBoundary(input string) string {
	if globalDefender == nil {
		return input
	}
	return globalDefender.WrapInBoundary(input)
}

// SanitizeAndWrap uses the global defender to sanitize and wrap.
func SanitizeAndWrap(input string) (string, InjectionResult) {
	if globalDefender == nil {
		return input, InjectionResult{Detected: false, Confidence: 0, SanitizedInput: input}
	}
	return globalDefender.SanitizeAndWrap(input)
}

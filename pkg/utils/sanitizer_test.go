// PicoClaw - Ultra-lightweight personal AI assistant
// License: MIT

package utils

import (
	"testing"
)

func TestSanitizerBasic(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Test email sanitization
	text := "My email is test@example.com"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("Email should be sanitized, got: %s", result.Sanitized)
	}
	if _, ok := result.Mappings["[EMAIL]"]; !ok {
		t.Errorf("Should have [EMAIL] mapping, got: %v", result.Mappings)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerPhone(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Test Chinese phone number
	text := "Call me at 13812345678"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("Phone should be sanitized, got: %s", result.Sanitized)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerIDCard(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Test Chinese ID card
	text := "My ID is 110101199003077654"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("ID card should be sanitized, got: %s", result.Sanitized)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerAPIKey(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Test API key
	text := "API key: sk-abcdefghijklmnopqrstuvwxyz123456"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("API key should be sanitized, got: %s", result.Sanitized)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerMultiple(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Test multiple sensitive data
	text := "Email: test@example.com, Phone: 13812345678, ID: 110101199003077654"
	result := s.Sanitize(text)

	// Should have 3 mappings
	if len(result.Mappings) < 3 {
		t.Errorf("Should have at least 3 mappings, got: %d", len(result.Mappings))
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerKeywords(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
		Keywords: []KeywordRule{
			{Word: "secret_password", Tag: "[PASSWORD]"},
		},
	}

	s := NewSanitizer(config)

	text := "My password is secret_password"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("Keyword should be sanitized, got: %s", result.Sanitized)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerCustomPattern(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
		CustomPatterns: []CustomPatternRule{
			{Name: "employee_id", Pattern: `EMP\d{6}`, Tag: "[EMP_ID]"},
		},
	}

	s := NewSanitizer(config)

	text := "Employee ID: EMP123456"
	result := s.Sanitize(text)
	if result.Sanitized == text {
		t.Errorf("Custom pattern should be sanitized, got: %s", result.Sanitized)
	}

	// Test restore
	restored := s.Restore(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Restore failed, expected: %s, got: %s", text, restored)
	}
}

func TestSanitizerDisabled(t *testing.T) {
	config := SanitizerConfig{
		Enabled: false,
	}

	s := NewSanitizer(config)

	text := "My email is test@example.com"
	result := s.Sanitize(text)
	if result.Sanitized != text {
		t.Errorf("When disabled, text should not be modified, got: %s", result.Sanitized)
	}
}

func TestSanitizerNil(t *testing.T) {
	var s *Sanitizer

	text := "My email is test@example.com"
	result := s.Sanitize(text)
	if result.Sanitized != text {
		t.Errorf("Nil sanitizer should not modify text, got: %s", result.Sanitized)
	}

	restored := s.Restore(text, map[string]string{"[EMAIL]": "test@example.com"})
	if restored != text {
		t.Errorf("Nil sanitizer restore should not modify text, got: %s", restored)
	}
}

func TestSanitizerNoDoubleSanitize(t *testing.T) {
	config := SanitizerConfig{
		Enabled: true,
	}

	s := NewSanitizer(config)

	// Already sanitized text should not be double-sanitized
	text := "My email is [EMAIL]"
	result := s.Sanitize(text)
	if result.Sanitized != text {
		t.Errorf("Already sanitized text should not be modified, got: %s", result.Sanitized)
	}
}

func TestGlobalSanitizer(t *testing.T) {
	// Test global sanitizer functions
	InitGlobalSanitizer(SanitizerConfig{
		Enabled: true,
	})

	text := "Email: test@example.com"
	result := SanitizeText(text)
	if result.Sanitized == text {
		t.Errorf("Global sanitize should work, got: %s", result.Sanitized)
	}

	restored := RestoreText(result.Sanitized, result.Mappings)
	if restored != text {
		t.Errorf("Global restore should work, got: %s", restored)
	}
}
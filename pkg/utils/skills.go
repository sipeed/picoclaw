package utils

import (
	"fmt"
	"strings"
)

// ValidateSkillIdentifier validates that the given skill identifier (slug or registry name) is non-empty
// and does not contain path separators ("/", "\\") or ".." for security.
func ValidateSkillIdentifier(identifier string) error {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return fmt.Errorf("identifier is required and must be a non-empty string")
	}
	if strings.ContainsAny(trimmed, "/\\") || strings.Contains(trimmed, "..") {
		return fmt.Errorf("identifier must not contain path separators or '..' to prevent directory traversal")
	}
	return nil
}

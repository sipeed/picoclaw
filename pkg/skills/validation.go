package skills

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/utils"
)

func ValidateSkillName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("skill name is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("skill name must not be an absolute path")
	}
	if err := utils.ValidateSkillIdentifier(trimmed); err != nil {
		return fmt.Errorf("skill name is invalid: %w", err)
	}
	if len(trimmed) > MaxNameLength {
		return fmt.Errorf("skill name exceeds %d characters", MaxNameLength)
	}
	if !namePattern.MatchString(trimmed) {
		return fmt.Errorf("skill name must be alphanumeric with hyphens")
	}
	return nil
}

package skills

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxContentSize is the maximum allowed size for SKILL.md content (~36k tokens).
const MaxContentSize = 100_000

// ValidateName checks that a skill name follows PicoClaw's naming convention:
// alphanumeric segments separated by hyphens (e.g. "my-skill", "go-code-improve").
// Uses the same namePattern as loader.go.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name must not be empty")
	}
	if len(name) > MaxNameLength {
		return fmt.Errorf("skill name too long: %d chars (max %d)", len(name), MaxNameLength)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid skill name %q: must be alphanumeric with hyphens", name)
	}
	return nil
}

// ValidateFrontmatter checks that SKILL.md content has valid YAML
// frontmatter with required "name" and "description" fields.
func ValidateFrontmatter(content string) error {
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return fmt.Errorf("SKILL.md frontmatter not closed (missing ---)")
	}

	yamlBlock := content[4 : 4+end]

	var meta map[string]any
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	name, ok := meta["name"]
	if !ok || fmt.Sprint(name) == "" {
		return fmt.Errorf("frontmatter missing required field: name")
	}

	desc, ok := meta["description"]
	if !ok || fmt.Sprint(desc) == "" {
		return fmt.Errorf("frontmatter missing required field: description")
	}

	if len(fmt.Sprint(desc)) > MaxDescriptionLength {
		return fmt.Errorf("description too long: %d chars (max %d)", len(fmt.Sprint(desc)), MaxDescriptionLength)
	}

	return nil
}

// ValidateSize checks that content doesn't exceed the maximum size.
func ValidateSize(content string) error {
	if len(content) > MaxContentSize {
		return fmt.Errorf("content too large: %d chars (max %d)", len(content), MaxContentSize)
	}
	return nil
}

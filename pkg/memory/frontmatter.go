package memory

import (
	"strings"
	"gopkg.in/yaml.v3"
)

// ParseFrontmatter extracts YAML frontmatter delimited by --- from content
// Returns: frontmatter map, remaining body, error
func ParseFrontmatter(content string) (map[string]interface{}, string, error) {
	result := make(map[string]interface{})
	body := content

	// Check for frontmatter delimiters
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		// No frontmatter, return empty map and original content
		return result, body, nil
	}

	// Parse YAML
	fmText := parts[1]
	if err := yaml.Unmarshal([]byte(fmText), &result); err != nil {
		return nil, "", err
	}

	body = parts[2]
	return result, body, nil
}

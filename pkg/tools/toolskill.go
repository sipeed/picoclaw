// pkg/tools/toolskill.go
package tools

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ToolSkill struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Type        string                 `yaml:"type"`
	Tags        []string               `yaml:"tags"`
	Version     string                 `yaml:"version"`
	UsageCount  int                    `yaml:"usage_count"`
	LastUsed    string                 `yaml:"last_used"`
	Metadata    map[string]interface{} `yaml:",inline"`
}

// LoadToolSkill reads a tool skill file and parses it
func LoadToolSkill(filePath string) (*ToolSkill, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var skill ToolSkill
	if err := yaml.Unmarshal(data, &skill); err != nil {
		return nil, err
	}

	// Extract tags from content if not in frontmatter
	if len(skill.Tags) == 0 {
		skill.Tags = extractTagsFromContent(string(data))
	}

	return &skill, nil
}

// extractTagsFromContent extracts tags from markdown content
func extractTagsFromContent(content string) []string {
	// Look for #tag patterns
	re := strings.NewReplacer(`#([a-zA-Z0-9_-]+)`, `$1`)
	matches := re.FindAllStringSubmatch(content, -1)
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

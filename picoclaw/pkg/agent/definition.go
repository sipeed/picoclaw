package agent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AgentDefinitionSource identifies which agent bootstrap file produced the definition.
type AgentDefinitionSource string

const (
	// AgentDefinitionSourceAgent indicates the new AGENT.md format.
	AgentDefinitionSourceAgent AgentDefinitionSource = "AGENT.md"
	// AgentDefinitionSourceAgents indicates the legacy AGENTS.md format.
	AgentDefinitionSourceAgents AgentDefinitionSource = "AGENTS.md"
)

// AgentFrontmatter holds machine-readable AGENT.md configuration.
//
// Known fields are exposed directly for convenience. Fields keeps the full
// parsed frontmatter so future refactors can read additional keys without
// changing the loader contract again.
type AgentFrontmatter struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Tools       []string       `json:"tools,omitempty"`
	Model       string         `json:"model,omitempty"`
	MaxTurns    *int           `json:"maxTurns,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
	MCPServers  []string       `json:"mcpServers,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
}

// AgentPromptDefinition represents the parsed AGENT.md or AGENTS.md prompt file.
type AgentPromptDefinition struct {
	Path           string           `json:"path"`
	Raw            string           `json:"raw"`
	Body           string           `json:"body"`
	RawFrontmatter string           `json:"raw_frontmatter,omitempty"`
	Frontmatter    AgentFrontmatter `json:"frontmatter"`
}

// SoulDefinition represents the resolved SOUL.md file linked to the agent.
type SoulDefinition struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// UserDefinition represents the resolved USER.md file linked to the workspace.
type UserDefinition struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// AgentContextDefinition captures the workspace agent definition in a runtime-friendly shape.
type AgentContextDefinition struct {
	Source AgentDefinitionSource  `json:"source,omitempty"`
	Agent  *AgentPromptDefinition `json:"agent,omitempty"`
	Soul   *SoulDefinition        `json:"soul,omitempty"`
	User   *UserDefinition        `json:"user,omitempty"`
}

// LoadAgentDefinition parses the workspace agent bootstrap files.
//
// It prefers the new AGENT.md format and its paired SOUL.md file. When the
// structured files are absent, it falls back to the legacy AGENTS.md layout so
// the current runtime can transition incrementally.
func (cb *ContextBuilder) LoadAgentDefinition() AgentContextDefinition {
	return loadAgentDefinition(cb.workspace)
}

func loadAgentDefinition(workspace string) AgentContextDefinition {
	definition := AgentContextDefinition{}
	definition.User = loadUserDefinition(workspace)
	agentPath := filepath.Join(workspace, string(AgentDefinitionSourceAgent))
	if content, err := os.ReadFile(agentPath); err == nil {
		prompt := parseAgentPromptDefinition(agentPath, string(content))
		definition.Source = AgentDefinitionSourceAgent
		definition.Agent = &prompt
		soulPath := filepath.Join(workspace, "SOUL.md")
		if content, err := os.ReadFile(soulPath); err == nil {
			definition.Soul = &SoulDefinition{
				Path:    soulPath,
				Content: string(content),
			}
		}
		return definition
	}

	legacyPath := filepath.Join(workspace, string(AgentDefinitionSourceAgents))
	if content, err := os.ReadFile(legacyPath); err == nil {
		definition.Source = AgentDefinitionSourceAgents
		definition.Agent = &AgentPromptDefinition{
			Path: legacyPath,
			Raw:  string(content),
			Body: string(content),
		}
	}

	defaultSoulPath := filepath.Join(workspace, "SOUL.md")
	if definition.Source != "" || fileExists(defaultSoulPath) {
		if content, err := os.ReadFile(defaultSoulPath); err == nil {
			definition.Soul = &SoulDefinition{
				Path:    defaultSoulPath,
				Content: string(content),
			}
		}
	}

	return definition
}

func (definition AgentContextDefinition) trackedPaths(workspace string) []string {
	paths := []string{
		filepath.Join(workspace, string(AgentDefinitionSourceAgent)),
		filepath.Join(workspace, "SOUL.md"),
		filepath.Join(workspace, "USER.md"),
	}
	if definition.Source != AgentDefinitionSourceAgent {
		paths = append(paths,
			filepath.Join(workspace, string(AgentDefinitionSourceAgents)),
			filepath.Join(workspace, "IDENTITY.md"),
		)
	}
	return uniquePaths(paths)
}

func loadUserDefinition(workspace string) *UserDefinition {
	userPath := filepath.Join(workspace, "USER.md")
	if content, err := os.ReadFile(userPath); err == nil {
		return &UserDefinition{
			Path:    userPath,
			Content: string(content),
		}
	}

	return nil
}

func parseAgentPromptDefinition(path, content string) AgentPromptDefinition {
	frontmatter, body := splitAgentFrontmatter(content)
	return AgentPromptDefinition{
		Path:           path,
		Raw:            content,
		Body:           body,
		RawFrontmatter: frontmatter,
		Frontmatter:    parseAgentFrontmatter(path, frontmatter),
	}
}

func parseAgentFrontmatter(path, frontmatter string) AgentFrontmatter {
	frontmatter = strings.TrimSpace(frontmatter)
	if frontmatter == "" {
		return AgentFrontmatter{}
	}

	rawFields := make(map[string]any)
	if err := yaml.Unmarshal([]byte(frontmatter), &rawFields); err != nil {
		logger.WarnCF("agent", "Failed to parse AGENT.md frontmatter", map[string]any{
			"path":  path,
			"error": err.Error(),
		})
		return AgentFrontmatter{}
	}

	var typed struct {
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Tools       []string `yaml:"tools"`
		Model       string   `yaml:"model"`
		MaxTurns    *int     `yaml:"maxTurns"`
		Skills      []string `yaml:"skills"`
		MCPServers  []string `yaml:"mcpServers"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &typed); err != nil {
		logger.WarnCF("agent", "Failed to decode AGENT.md frontmatter fields", map[string]any{
			"path":  path,
			"error": err.Error(),
		})
		return AgentFrontmatter{}
	}

	return AgentFrontmatter{
		Name:        strings.TrimSpace(typed.Name),
		Description: strings.TrimSpace(typed.Description),
		Tools:       append([]string(nil), typed.Tools...),
		Model:       strings.TrimSpace(typed.Model),
		MaxTurns:    typed.MaxTurns,
		Skills:      append([]string(nil), typed.Skills...),
		MCPServers:  append([]string(nil), typed.MCPServers...),
		Fields:      rawFields,
	}
}

func splitAgentFrontmatter(content string) (frontmatter, body string) {
	normalized := string(parser.NormalizeNewlines([]byte(content)))
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", content
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", content
	}

	frontmatter = strings.Join(lines[1:end], "\n")
	body = strings.Join(lines[end+1:], "\n")
	body = strings.TrimLeft(body, "\n")
	return frontmatter, body
}

func relativeWorkspacePath(workspace, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	relativePath, err := filepath.Rel(workspace, path)
	if err == nil && relativePath != "." && !strings.HasPrefix(relativePath, "..") {
		return filepath.ToSlash(relativePath)
	}
	return filepath.Clean(path)
}

func uniquePaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		cleaned := filepath.Clean(path)
		if slices.Contains(result, cleaned) {
			continue
		}
		result = append(result, cleaned)
	}
	return result
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

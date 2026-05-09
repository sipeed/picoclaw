package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultWorkspacePath = "~/.picoclaw/workspace/agents"

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type Manager struct {
	workspacePath string
}

func NewManager(workspacePath string) *Manager {
	if workspacePath == "" {
		workspacePath = DefaultWorkspacePath
	}
	return &Manager{workspacePath: expandPath(workspacePath)}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func (m *Manager) ensureDir() error {
	if err := os.MkdirAll(m.workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}
	return nil
}

func (m *Manager) ListAgents() ([]*Agent, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(m.workspacePath)
	if err != nil {
		return nil, err
	}

	var agents []*Agent
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}
		agent, err := m.readAgentFile(file.Name())
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

func (m *Manager) GetAgent(slug string) (*Agent, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	filename := slug + ".md"
	fp := filepath.Join(m.workspacePath, filename)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return nil, fmt.Errorf("agent not found: %s", slug)
	}
	return m.readAgentFile(filename)
}

func (m *Manager) CreateAgent(req AgentCreateRequest) (*Agent, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	slug := m.slugify(req.Name)
	fp := filepath.Join(m.workspacePath, slug+".md")

	if _, err := os.Stat(fp); err == nil {
		return nil, fmt.Errorf("agent already exists: %s", slug)
	}

	agent := &Agent{
		Slug:            slug,
		Name:            req.Name,
		Description:     req.Description,
		SystemPrompt:    req.SystemPrompt,
		Model:           req.Model,
		ToolPermissions: req.ToolPermissions,
		Status:          AgentStatusEnabled,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := m.writeAgentFile(agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (m *Manager) UpdateAgent(slug string, req AgentUpdateRequest) (*Agent, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	fp := filepath.Join(m.workspacePath, slug+".md")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return nil, fmt.Errorf("agent not found: %s", slug)
	}

	agent, err := m.readAgentFile(slug + ".md")
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		agent.Name = req.Name
		agent.Slug = m.slugify(req.Name)
		fp = filepath.Join(m.workspacePath, agent.Slug+".md")
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.ToolPermissions != nil {
		agent.ToolPermissions = req.ToolPermissions
	}
	if req.Status != "" {
		agent.Status = AgentStatus(req.Status)
	}
	agent.UpdatedAt = time.Now()

	if err := m.writeAgentFile(agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (m *Manager) DeleteAgent(slug string) error {
	if err := m.ensureDir(); err != nil {
		return err
	}

	fp := filepath.Join(m.workspacePath, slug+".md")
	if err := os.Remove(fp); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}
	return nil
}

func (m *Manager) ImportAgent(content string) (*Agent, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	agent, err := m.parseAgentFromContent(content)
	if err != nil {
		return nil, err
	}

	fp := filepath.Join(m.workspacePath, agent.Slug+".md")
	if _, err := os.Stat(fp); err == nil {
		return nil, fmt.Errorf("agent already exists: %s", agent.Slug)
	}

	if err := m.writeAgentFile(agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (m *Manager) slugify(name string) string {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return slugRegex.FindString(slug)
}

func (m *Manager) parseAgentFromContent(content string) (*Agent, error) {
	var agent Agent

	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		agent.Name = "unknown"
		agent.Slug = m.slugify("unknown-" + fmt.Sprintf("%d", time.Now().Unix()))
		agent.SystemPrompt = content
		return &agent, nil
	}

	inFrontmatter := true
	var frontmatter strings.Builder
	contentStart := 0

	for i, line := range lines {
		if inFrontmatter && line == "---" {
			if contentStart == 0 {
				contentStart = i + 1
				inFrontmatter = false
			} else {
				break
			}
		} else if inFrontmatter {
			frontmatter.WriteString(line + "\n")
		}
	}

	yamlLines := strings.Split(frontmatter.String(), "\n")
	for _, line := range yamlLines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "name":
					agent.Name = strings.Trim(value, "\"'")
				case "description":
					agent.Description = value
				case "system_prompt":
					agent.SystemPrompt = value
				case "model":
					agent.Model = strings.Trim(value, "\"'")
				case "tool_permissions":
					// Parse array
				}
			}
		}
	}

	if agent.Slug == "" {
		agent.Slug = m.slugify(agent.Name)
	}
	if agent.Model == "" {
		agent.Model = "claude-3-5-sonnet"
	}

	return &agent, nil
}

func (m *Manager) readAgentFile(filename string) (*Agent, error) {
	fp := filepath.Join(m.workspacePath, filename)
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	agent, err := m.parseAgentFromContent(string(data))
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (m *Manager) writeAgentFile(agent *Agent) error {
	var content strings.Builder

	content.WriteString("---\n")
	content.WriteString(fmt.Sprintf("name: %s\n", agent.Name))
	if agent.Description != "" {
		content.WriteString(fmt.Sprintf("description: >\n  %s\n", agent.Description))
	}
	if agent.SystemPrompt != "" {
		content.WriteString(fmt.Sprintf("system_prompt: >\n  %s\n", agent.SystemPrompt))
	}
	content.WriteString(fmt.Sprintf("model: %s\n", agent.Model))
	content.WriteString(fmt.Sprintf("slug: %s\n", agent.Slug))
	content.WriteString("---\n\n")

	content.WriteString(agent.SystemPrompt)
	content.WriteString("\n")

	fp := filepath.Join(m.workspacePath, agent.Slug+".md")
	if err := os.WriteFile(fp, []byte(content.String()), 0644); err != nil {
		return err
	}
	return nil
}

package manager

import "time"

type AgentStatus string

const (
	AgentStatusEnabled  AgentStatus = "enabled"
	AgentStatusDisabled AgentStatus = "disabled"
)

type Agent struct {
	Slug            string      `json:"slug" yaml:"slug"`
	Name            string      `json:"name" yaml:"name"`
	Description     string      `json:"description" yaml:"description"`
	SystemPrompt    string      `json:"system_prompt" yaml:"system_prompt"`
	Model           string      `json:"model" yaml:"model"`
	ToolPermissions []string    `json:"tool_permissions" yaml:"tool_permissions"`
	Status          AgentStatus `json:"status" yaml:"status"`
	CreatedAt       time.Time   `json:"created_at" yaml:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" yaml:"updated_at"`
}

type AgentFile struct {
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

type AgentListResponse struct {
	Agents []*Agent `json:"agents"`
}

type AgentCreateRequest struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt" binding:"required"`
	Model           string   `json:"model" binding:"required"`
	ToolPermissions []string `json:"tool_permissions"`
}

type AgentUpdateRequest struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt"`
	Model           string   `json:"model"`
	ToolPermissions []string `json:"tool_permissions"`
	Status          string   `json:"status"`
}

// Package protoagent provides a behavior prototyping tool that transforms
// functional and non-functional requirements into working agent configurations,
// databases, interfaces, and communication channels.
package protoagent

import (
	"encoding/json"
	"time"
)

// InteractionMethod defines how the agent interacts with external systems.
type InteractionMethod string

const (
	InteractionAPI        InteractionMethod = "api"
	InteractionMCP        InteractionMethod = "mcp"
	InteractionUI         InteractionMethod = "ui"
	InteractionMessaging  InteractionMethod = "messaging"
	InteractionWebhook    InteractionMethod = "webhook"
	InteractionCLI        InteractionMethod = "cli"
	InteractionDatabase   InteractionMethod = "database"
	InteractionFile       InteractionMethod = "file"
	InteractionEventBus   InteractionMethod = "eventbus"
)

// FunctionalRequirement describes what the system should do.
type FunctionalRequirement struct {
	ID                string            `json:"id" yaml:"id"`
	Type              string            `json:"type" yaml:"type"` // action, operation, actor, resource
	Name              string            `json:"name" yaml:"name"`
	Description       string            `json:"description" yaml:"description"`
	Inputs            []ParameterDef    `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs           []ParameterDef    `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Preconditions     []string          `json:"preconditions,omitempty" yaml:"preconditions,omitempty"`
	Postconditions    []string          `json:"postconditions,omitempty" yaml:"postconditions,omitempty"`
	InteractionMethods []InteractionMethod `json:"interactionMethods,omitempty" yaml:"interactionMethods,omitempty"`
	Priority          int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Tags              []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// NonFunctionalRequirement describes constraints and quality attributes.
type NonFunctionalRequirement struct {
	ID          string            `json:"id" yaml:"id"`
	Category    string            `json:"category" yaml:"category"` // security, performance, reliability, scalability
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Constraints map[string]string `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Metrics     []MetricDef       `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Priority    int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ParameterDef defines a parameter for inputs/outputs.
type ParameterDef struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
}

// MetricDef defines a measurable metric for NFRs.
type MetricDef struct {
	Name      string  `json:"name" yaml:"name"`
	Target    string  `json:"target" yaml:"target"`
	Threshold float64 `json:"threshold,omitempty" yaml:"threshold,omitempty"`
	Unit      string  `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// SecurityRequirement captures security-specific NFRs.
type SecurityRequirement struct {
	Permissions      []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Roles            []string `json:"roles,omitempty" yaml:"roles,omitempty"`
	Authorizations   []string `json:"authorizations,omitempty" yaml:"authorizations,omitempty"`
	SecurityControls []string `json:"securityControls,omitempty" yaml:"securityControls,omitempty"`
	DataClassification string `json:"dataClassification,omitempty" yaml:"dataClassification,omitempty"`
}

// PerformanceRequirement captures performance-specific NFRs.
type PerformanceRequirement struct {
	ResponseTime    time.Duration `json:"responseTime,omitempty" yaml:"responseTime,omitempty"`
	Throughput      float64       `json:"throughput,omitempty" yaml:"throughput,omitempty"`
	Concurrency     int           `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	ResourceLimits  ResourceLimit `json:"resourceLimits,omitempty" yaml:"resourceLimits,omitempty"`
}

// ResourceLimit defines resource constraints.
type ResourceLimit struct {
	Memory    string `json:"memory,omitempty" yaml:"memory,omitempty"`
	CPU       string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Storage   string `json:"storage,omitempty" yaml:"storage,omitempty"`
	Network   string `json:"network,omitempty" yaml:"network,omitempty"`
}

// RequirementsDocument is the complete specification input.
type RequirementsDocument struct {
	Version                 string                    `json:"version" yaml:"version"`
	Name                    string                    `json:"name" yaml:"name"`
	Description             string                    `json:"description" yaml:"description"`
	FunctionalRequirements  []FunctionalRequirement   `json:"functionalRequirements" yaml:"functionalRequirements"`
	NonFunctionalRequirements []NonFunctionalRequirement `json:"nonFunctionalRequirements" yaml:"nonFunctionalRequirements"`
	SecurityRequirements    []SecurityRequirement     `json:"securityRequirements,omitempty" yaml:"securityRequirements,omitempty"`
	PerformanceRequirements []PerformanceRequirement  `json:"performanceRequirements,omitempty" yaml:"performanceRequirements,omitempty"`
	Metadata                map[string]string         `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// GeneratedArtifacts represents all outputs from the prototyping process.
type GeneratedArtifacts struct {
	Timestamp       time.Time              `json:"timestamp" yaml:"timestamp"`
	AgentConfig     *AgentConfig           `json:"agentConfig,omitempty" yaml:"agentConfig,omitempty"`
	DatabaseSchemas []DatabaseSchema       `json:"databaseSchemas,omitempty" yaml:"databaseSchemas,omitempty"`
	Interfaces      []InterfaceDef         `json:"interfaces,omitempty" yaml:"interfaces,omitempty"`
	Channels        []ChannelConfig        `json:"channels,omitempty" yaml:"channels,omitempty"`
	Policies        []PolicyDefinition     `json:"policies,omitempty" yaml:"policies,omitempty"`
	Skills          []SkillDefinition      `json:"skills,omitempty" yaml:"skills,omitempty"`
	Tools           []ToolDefinition       `json:"tools,omitempty" yaml:"tools,omitempty"`
	MCPConfig       *MCPConfiguration      `json:"mcpConfig,omitempty" yaml:"mcpConfig,omitempty"`
	ValidationReport *ValidationReport     `json:"validationReport,omitempty" yaml:"validationReport,omitempty"`
}

// AgentConfig is the generated AGENT.md configuration.
type AgentConfig struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Tools       []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Model       string   `json:"model,omitempty" yaml:"model,omitempty"`
	MaxTurns    *int     `json:"maxTurns,omitempty" yaml:"maxTurns,omitempty"`
	Skills      []string `json:"skills,omitempty" yaml:"skills,omitempty"`
	MCPServers  []string `json:"mcpServers,omitempty" yaml:"mcpServers,omitempty"`
	Body        string   `json:"body" yaml:"body"`
}

// DatabaseSchema defines a database structure.
type DatabaseSchema struct {
	Name        string         `json:"name" yaml:"name"`
	Type        string         `json:"type" yaml:"type"` // sql, nosql, memory, file
	Tables      []TableDef     `json:"tables,omitempty" yaml:"tables,omitempty"`
	Collections []CollectionDef `json:"collections,omitempty" yaml:"collections,omitempty"`
	Indexes     []IndexDef     `json:"indexes,omitempty" yaml:"indexes,omitempty"`
	Migrations  []string       `json:"migrations,omitempty" yaml:"migrations,omitempty"`
}

// TableDef defines a SQL table.
type TableDef struct {
	Name    string       `json:"name" yaml:"name"`
	Columns []ColumnDef  `json:"columns" yaml:"columns"`
	Indexes []string     `json:"indexes,omitempty" yaml:"indexes,omitempty"`
}

// ColumnDef defines a table column.
type ColumnDef struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"`
	Nullable   bool   `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	PrimaryKey bool   `json:"primaryKey,omitempty" yaml:"primaryKey,omitempty"`
	Unique     bool   `json:"unique,omitempty" yaml:"unique,omitempty"`
	Default    string `json:"default,omitempty" yaml:"default,omitempty"`
}

// CollectionDef defines a NoSQL collection.
type CollectionDef struct {
	Name   string          `json:"name" yaml:"name"`
	Schema json.RawMessage `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// IndexDef defines a database index.
type IndexDef struct {
	Name    string   `json:"name" yaml:"name"`
	Table   string   `json:"table" yaml:"table"`
	Columns []string `json:"columns" yaml:"columns"`
	Unique  bool     `json:"unique,omitempty" yaml:"unique,omitempty"`
}

// InterfaceDef defines a user or system interface.
type InterfaceDef struct {
	Name        string            `json:"name" yaml:"name"`
	Type        string            `json:"type" yaml:"type"` // web, cli, api, gui
	Endpoints   []EndpointDef     `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Screens     []ScreenDef       `json:"screens,omitempty" yaml:"screens,omitempty"`
	Components  []ComponentDef    `json:"components,omitempty" yaml:"components,omitempty"`
}

// EndpointDef defines an API endpoint.
type EndpointDef struct {
	Path        string            `json:"path" yaml:"path"`
	Method      string            `json:"method" yaml:"method"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Inputs      []ParameterDef    `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs     []ParameterDef    `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Auth        []string          `json:"auth,omitempty" yaml:"auth,omitempty"`
	RateLimit   *RateLimitDef     `json:"rateLimit,omitempty" yaml:"rateLimit,omitempty"`
}

// ScreenDef defines a UI screen.
type ScreenDef struct {
	Name        string            `json:"name" yaml:"name"`
	Route       string            `json:"route,omitempty" yaml:"route,omitempty"`
	Components  []ComponentDef    `json:"components,omitempty" yaml:"components,omitempty"`
	Actions     []ActionDef       `json:"actions,omitempty" yaml:"actions,omitempty"`
}

// ComponentDef defines a UI component.
type ComponentDef struct {
	Name       string            `json:"name" yaml:"name"`
	Type       string            `json:"type" yaml:"type"`
	Properties map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// ActionDef defines a UI action.
type ActionDef struct {
	Name        string `json:"name" yaml:"name"`
	Trigger     string `json:"trigger" yaml:"trigger"`
	Handler     string `json:"handler" yaml:"handler"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ChannelConfig defines a communication channel.
type ChannelConfig struct {
	Name       string            `json:"name" yaml:"name"`
	Type       string            `json:"type" yaml:"type"` // telegram, discord, slack, webhook, etc.
	Config     map[string]string `json:"config" yaml:"config"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	Commands   []CommandDef      `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// CommandDef defines a channel command.
type CommandDef struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Handler     string `json:"handler" yaml:"handler"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// RateLimitDef defines rate limiting configuration.
type RateLimitDef struct {
	Requests int           `json:"requests" yaml:"requests"`
	Window   time.Duration `json:"window" yaml:"window"`
}

// PolicyDefinition defines an OPA policy.
type PolicyDefinition struct {
	Name        string            `json:"name" yaml:"name"`
	Package     string            `json:"package" yaml:"package"`
	Rules       []PolicyRule      `json:"rules,omitempty" yaml:"rules,omitempty"`
	Rego        string            `json:"rego" yaml:"rego"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// PolicyRule defines a single policy rule.
type PolicyRule struct {
	Name      string `json:"name" yaml:"name"`
	Condition string `json:"condition" yaml:"condition"`
	Effect    string `json:"effect" yaml:"effect"` // allow, deny
}

// SkillDefinition defines a skill to be generated.
type SkillDefinition struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Code        string            `json:"code" yaml:"code"`
	Dependencies []string         `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Triggers    []string          `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

// ToolDefinition defines a tool to be generated.
type ToolDefinition struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Type        string            `json:"type" yaml:"type"` // shell, api, mcp, custom
	Config      map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
	Code        string            `json:"code,omitempty" yaml:"code,omitempty"`
}

// MCPConfiguration defines MCP server configuration.
type MCPConfiguration struct {
	Servers []MCPServerConfig `json:"servers" yaml:"servers"`
}

// MCPServerConfig defines a single MCP server.
type MCPServerConfig struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type" yaml:"type"` // stdio, sse, websocket
	Command string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty"`
	URL     string            `json:"url,omitempty" yaml:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ValidationReport contains validation results.
type ValidationReport struct {
	Valid       bool               `json:"valid" yaml:"valid"`
	Errors      []ValidationError  `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings    []ValidationWarning `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Suggestions []string           `json:"suggestions,omitempty" yaml:"suggestions,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

// ValidationWarning represents a validation warning.
type ValidationWarning struct {
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

package protoagent

import (
"context"
"fmt"
"strings"
"time"
)

// Engine is the main prototyping engine that transforms requirements into artifacts.
type Engine struct {
config EngineConfig
}

// EngineConfig holds configuration for the prototyping engine.
type EngineConfig struct {
OutputDir       string `json:"outputDir" yaml:"outputDir"`
Workspace       string `json:"workspace" yaml:"workspace"`
EnableOPA       bool   `json:"enableOPA" yaml:"enableOPA"`
EnableAI        bool   `json:"enableAI" yaml:"enableAI"`
AIProvider      string `json:"aiProvider,omitempty" yaml:"aiProvider,omitempty"`
DryRun          bool   `json:"dryRun" yaml:"dryRun"`
Verbose         bool   `json:"verbose" yaml:"verbose"`
}

// NewEngine creates a new prototyping engine.
func NewEngine(config EngineConfig) *Engine {
return &Engine{
config: config,
}
}

// ProcessRequirements takes a requirements document and generates all artifacts.
func (e *Engine) ProcessRequirements(ctx context.Context, reqs *RequirementsDocument) (*GeneratedArtifacts, error) {
fmt.Printf("[protoagent] Starting requirements processing: %s (v%s)\n", reqs.Name, reqs.Version)

artifacts := &GeneratedArtifacts{
Timestamp: time.Now(),
}

// Validate requirements first
if err := e.validateRequirements(reqs); err != nil {
return nil, fmt.Errorf("validation failed: %w", err)
}

// Generate agent configuration
agentConfig, err := e.generateAgentConfig(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate agent config: %v\n", err)
} else {
artifacts.AgentConfig = agentConfig
}

// Generate database schemas
dbSchemas, err := e.generateDatabaseSchemas(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate database schemas: %v\n", err)
} else {
artifacts.DatabaseSchemas = dbSchemas
}

// Generate interfaces
interfaces, err := e.generateInterfaces(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate interfaces: %v\n", err)
} else {
artifacts.Interfaces = interfaces
}

// Generate communication channels
channels, err := e.generateChannels(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate channels: %v\n", err)
} else {
artifacts.Channels = channels
}

// Generate OPA policies if enabled
if e.config.EnableOPA {
policies, err := e.generateOPAPolicies(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate OPA policies: %v\n", err)
} else {
artifacts.Policies = policies
}
}

// Generate skills
skills, err := e.generateSkills(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate skills: %v\n", err)
} else {
artifacts.Skills = skills
}

// Generate tools
tools, err := e.generateTools(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate tools: %v\n", err)
} else {
artifacts.Tools = tools
}

// Generate MCP configuration
mcpConfig, err := e.generateMCPConfig(reqs)
if err != nil {
fmt.Printf("[protoagent] Warning: Failed to generate MCP config: %v\n", err)
} else {
artifacts.MCPConfig = mcpConfig
}

// Generate validation report
artifacts.ValidationReport = e.generateValidationReport(reqs, artifacts)

fmt.Printf("[protoagent] Requirements processing completed: %d artifacts generated\n", e.countArtifacts(artifacts))

return artifacts, nil
}

// validateRequirements performs validation on the requirements document.
func (e *Engine) validateRequirements(reqs *RequirementsDocument) error {
var errors []ValidationError
var warnings []ValidationWarning

// Check for required fields
if reqs.Name == "" {
errors = append(errors, ValidationError{
Field:   "name",
Message: "Name is required",
})
}

if len(reqs.FunctionalRequirements) == 0 {
warnings = append(warnings, ValidationWarning{
Field:   "functionalRequirements",
Message: "No functional requirements defined",
})
}

// Validate FR IDs are unique
frIDs := make(map[string]bool)
for i, fr := range reqs.FunctionalRequirements {
if fr.ID == "" {
errors = append(errors, ValidationError{
Field:   fmt.Sprintf("functionalRequirements[%d].id", i),
Message: "ID is required for each functional requirement",
})
} else if frIDs[fr.ID] {
errors = append(errors, ValidationError{
Field:   fmt.Sprintf("functionalRequirements[%d].id", i),
Message: fmt.Sprintf("Duplicate ID: %s", fr.ID),
})
}
frIDs[fr.ID] = true
}

// Validate NFR categories
validCategories := map[string]bool{
"security": true, "performance": true, "reliability": true,
"scalability": true, "availability": true, "maintainability": true,
}
for i, nfr := range reqs.NonFunctionalRequirements {
if nfr.Category != "" && !validCategories[nfr.Category] {
warnings = append(warnings, ValidationWarning{
Field:   fmt.Sprintf("nonFunctionalRequirements[%d].category", i),
Message: fmt.Sprintf("Unknown category: %s", nfr.Category),
})
}
}

// Check for missing interaction methods
for i, fr := range reqs.FunctionalRequirements {
if len(fr.InteractionMethods) == 0 {
warnings = append(warnings, ValidationWarning{
Field:   fmt.Sprintf("functionalRequirements[%d].interactionMethods", i),
Message: "No interaction methods specified",
})
}
}

if len(errors) > 0 {
return fmt.Errorf("validation failed with %d errors", len(errors))
}

return nil
}

// generateAgentConfig creates the agent configuration from requirements.
func (e *Engine) generateAgentConfig(reqs *RequirementsDocument) (*AgentConfig, error) {
config := &AgentConfig{
Name:        reqs.Name,
Description: reqs.Description,
}

// Extract tools from functional requirements
toolSet := make(map[string]bool)
for _, fr := range reqs.FunctionalRequirements {
for _, method := range fr.InteractionMethods {
switch method {
case InteractionAPI:
toolSet["api_client"] = true
case InteractionMCP:
toolSet["mcp_client"] = true
case InteractionMessaging:
toolSet["message_handler"] = true
case InteractionWebhook:
toolSet["webhook_handler"] = true
case InteractionDatabase:
toolSet["database_tool"] = true
case InteractionFile:
toolSet["file_tool"] = true
}
}
}

for tool := range toolSet {
config.Tools = append(config.Tools, tool)
}

// Build agent body from requirements
var body strings.Builder
body.WriteString(fmt.Sprintf("# %s Agent\n\n", config.Name))
body.WriteString(fmt.Sprintf("## Description\n\n%s\n\n", config.Description))

body.WriteString("## Generated Capabilities\n\n")
body.WriteString("This agent was automatically generated from requirements specification.\n\n")

body.WriteString("### Functional Requirements\n\n")
for _, fr := range reqs.FunctionalRequirements {
body.WriteString(fmt.Sprintf("- **%s**: %s\n", fr.Name, fr.Description))
}

body.WriteString("\n### Non-Functional Requirements\n\n")
for _, nfr := range reqs.NonFunctionalRequirements {
body.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", nfr.Name, nfr.Category, nfr.Description))
}

body.WriteString("\n## Instructions\n\n")
body.WriteString("Follow the generated policies and use the provided tools to fulfill the requirements.\n")

config.Body = body.String()

return config, nil
}

// countArtifacts returns the total count of generated artifacts.
func (e *Engine) countArtifacts(artifacts *GeneratedArtifacts) int {
count := 0
if artifacts.AgentConfig != nil {
count++
}
count += len(artifacts.DatabaseSchemas)
count += len(artifacts.Interfaces)
count += len(artifacts.Channels)
count += len(artifacts.Policies)
count += len(artifacts.Skills)
count += len(artifacts.Tools)
return count
}

// generateValidationReport creates a validation report for the generated artifacts.
func (e *Engine) generateValidationReport(reqs *RequirementsDocument, artifacts *GeneratedArtifacts) *ValidationReport {
report := &ValidationReport{
Valid: true,
}

// Check if essential artifacts were generated
if artifacts.AgentConfig == nil {
report.Valid = false
report.Errors = append(report.Errors, ValidationError{
Field:   "agentConfig",
Message: "Failed to generate agent configuration",
})
}

// Add suggestions based on requirements
if len(reqs.SecurityRequirements) > 0 && len(artifacts.Policies) == 0 {
report.Suggestions = append(report.Suggestions,
"Consider enabling OPA for security policy enforcement")
}

if len(reqs.FunctionalRequirements) > 10 && len(artifacts.Skills) == 0 {
report.Suggestions = append(report.Suggestions,
"Consider creating skills for complex functional requirements")
}

return report
}

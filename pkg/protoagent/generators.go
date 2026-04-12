package protoagent

import (
	"fmt"
	"strings"
)

// generateDatabaseSchemas creates database schemas from requirements.
func (e *Engine) generateDatabaseSchemas(reqs *RequirementsDocument) ([]DatabaseSchema, error) {
	var schemas []DatabaseSchema

	// Analyze requirements to determine data entities
	entities := e.extractDataEntities(reqs)
	
	if len(entities) == 0 {
		// Create a default schema if no entities detected
		schemas = append(schemas, DatabaseSchema{
			Name: "default",
			Type: "sql",
			Tables: []TableDef{
				{
					Name: "entities",
					Columns: []ColumnDef{
						{Name: "id", Type: "uuid", PrimaryKey: true},
						{Name: "name", Type: "varchar(255)", Nullable: false},
						{Name: "created_at", Type: "timestamp", Default: "CURRENT_TIMESTAMP"},
						{Name: "updated_at", Type: "timestamp"},
					},
				},
			},
		})
		return schemas, nil
	}

	// Generate schema for each entity
	for _, entity := range entities {
		schema := DatabaseSchema{
			Name: entity.Name,
			Type: "sql",
		}

		table := TableDef{
			Name: strings.ToLower(entity.Name) + "s",
			Columns: []ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "created_at", Type: "timestamp", Default: "CURRENT_TIMESTAMP"},
				{Name: "updated_at", Type: "timestamp"},
			},
		}

		// Add columns based on entity attributes
		for _, attr := range entity.Attributes {
			col := ColumnDef{
				Name:     strings.ToLower(attr.Name),
				Type:     e.mapTypeToSQL(attr.Type),
				Nullable: !attr.Required,
			}
			table.Columns = append(table.Columns, col)
		}

		schema.Tables = append(schema.Tables, table)
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// Entity represents a data entity extracted from requirements.
type Entity struct {
	Name       string
	Attributes []Attribute
}

// Attribute represents an entity attribute.
type Attribute struct {
	Name     string
	Type     string
	Required bool
}

// extractDataEntities analyzes requirements to find data entities.
func (e *Engine) extractDataEntities(reqs *RequirementsDocument) []Entity {
	entityMap := make(map[string]*Entity)

	// Extract entities from functional requirements
	for _, fr := range reqs.FunctionalRequirements {
		// Look for resource-related requirements
		if fr.Type == "resource" || strings.Contains(strings.ToLower(fr.Description), "store") ||
			strings.Contains(strings.ToLower(fr.Description), "manage") {
			
			entityName := e.extractEntityName(fr)
			if entityName != "" {
				if _, exists := entityMap[entityName]; !exists {
					entityMap[entityName] = &Entity{
						Name:       entityName,
						Attributes: []Attribute{},
					}
				}

				// Extract attributes from inputs/outputs
				for _, input := range fr.Inputs {
					attr := Attribute{
						Name:     input.Name,
						Type:     input.Type,
						Required: input.Required,
					}
					entityMap[entityName].Attributes = append(entityMap[entityName].Attributes, attr)
				}
			}
		}
	}

	// Convert map to slice
	var entities []Entity
	for _, entity := range entityMap {
		entities = append(entities, *entity)
	}

	return entities
}

// extractEntityName tries to extract an entity name from a requirement.
func (e *Engine) extractEntityName(fr FunctionalRequirement) string {
	// Try to extract from name
	name := strings.ToLower(fr.Name)
	
	// Common entity patterns
	patterns := []string{"user", "account", "order", "product", "item", "record", "data", "document"}
	for _, pattern := range patterns {
		if strings.Contains(name, pattern) {
			return strings.Title(pattern)
		}
	}

	// Use the requirement name as fallback
	if fr.Name != "" {
		return fr.Name
	}

	return ""
}

// mapTypeToSQL maps a generic type to SQL type.
func (e *Engine) mapTypeToSQL(t string) string {
	switch strings.ToLower(t) {
	case "string", "text":
		return "varchar(255)"
	case "int", "integer", "number":
		return "integer"
	case "float", "double", "decimal":
		return "decimal(10,2)"
	case "bool", "boolean":
		return "boolean"
	case "date":
		return "date"
	case "datetime", "timestamp":
		return "timestamp"
	case "json":
		return "jsonb"
	default:
		return "text"
	}
}

// generateInterfaces creates interface definitions from requirements.
func (e *Engine) generateInterfaces(reqs *RequirementsDocument) ([]InterfaceDef, error) {
	var interfaces []InterfaceDef

	// Check for UI interaction methods
	hasUI := false
	hasAPI := false
	for _, fr := range reqs.FunctionalRequirements {
		for _, method := range fr.InteractionMethods {
			if method == InteractionUI {
				hasUI = true
			}
			if method == InteractionAPI {
				hasAPI = true
			}
		}
	}

	// Generate API interface if needed
	if hasAPI {
		apiInterface := InterfaceDef{
			Name: "API",
			Type: "api",
		}

		// Create endpoints from functional requirements
		for _, fr := range reqs.FunctionalRequirements {
			endpoint := EndpointDef{
				Path:        fmt.Sprintf("/api/v1/%s", strings.ToLower(fr.Name)),
				Method:      "POST",
				Description: fr.Description,
				Inputs:      fr.Inputs,
				Outputs:     fr.Outputs,
			}
			apiInterface.Endpoints = append(apiInterface.Endpoints, endpoint)
		}

		interfaces = append(interfaces, apiInterface)
	}

	// Generate Web UI interface if needed
	if hasUI {
		webInterface := InterfaceDef{
			Name: "Web UI",
			Type: "web",
		}

		// Create screens from functional requirements
		for _, fr := range reqs.FunctionalRequirements {
			screen := ScreenDef{
				Name:   fr.Name,
				Route:  fmt.Sprintf("/%s", strings.ToLower(fr.Name)),
			}

			// Add components based on inputs
			for _, input := range fr.Inputs {
				component := ComponentDef{
					Name: input.Name,
					Type: e.inputTypeToComponent(input.Type),
					Properties: map[string]string{
						"label": input.Name,
						"required": fmt.Sprintf("%v", input.Required),
					},
				}
				screen.Components = append(screen.Components, component)
			}

			webInterface.Screens = append(webInterface.Screens, screen)
		}

		interfaces = append(interfaces, webInterface)
	}

	return interfaces, nil
}

// inputTypeToComponent maps input types to UI components.
func (e *Engine) inputTypeToComponent(t string) string {
	switch strings.ToLower(t) {
	case "string", "text":
		return "TextInput"
	case "int", "integer", "number", "float":
		return "NumberInput"
	case "bool", "boolean":
		return "Checkbox"
	case "date":
		return "DatePicker"
	case "datetime", "timestamp":
		return "DateTimePicker"
	default:
		return "TextInput"
	}
}

// generateChannels creates communication channel configurations.
func (e *Engine) generateChannels(reqs *RequirementsDocument) ([]ChannelConfig, error) {
	var channels []ChannelConfig

	// Check for messaging interaction methods
	for _, fr := range reqs.FunctionalRequirements {
		for _, method := range fr.InteractionMethods {
			if method == InteractionMessaging {
				// Add default channels based on requirements
				channels = append(channels, ChannelConfig{
					Name:    "telegram",
					Type:    "telegram",
					Enabled: true,
					Config: map[string]string{
						"token": "${TELEGRAM_BOT_TOKEN}",
					},
				})
				break
			}
		}
	}

	// Check for webhook requirements
	for _, fr := range reqs.FunctionalRequirements {
		for _, method := range fr.InteractionMethods {
			if method == InteractionWebhook {
				channels = append(channels, ChannelConfig{
					Name:    "webhook",
					Type:    "webhook",
					Enabled: true,
					Config: map[string]string{
						"path": "/webhook",
						"secret": "${WEBHOOK_SECRET}",
					},
				})
				break
			}
		}
	}

	return channels, nil
}

// generateSkills creates skill definitions from requirements.
func (e *Engine) generateSkills(reqs *RequirementsDocument) ([]SkillDefinition, error) {
	var skills []SkillDefinition

	// Generate skills for complex operations
	for _, fr := range reqs.FunctionalRequirements {
		if fr.Type == "operation" && len(fr.Preconditions) > 0 {
			skill := SkillDefinition{
				Name:        fmt.Sprintf("%s_skill", strings.ToLower(fr.Name)),
				Description: fr.Description,
				Triggers:    []string{fr.Name},
			}

			// Generate skill code template
			code := fmt.Sprintf(`// Auto-generated skill for: %s
package skills

import "context"

func %sSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// TODO: Implement skill logic
	// Preconditions: %v
	return nil, nil
}
`, fr.Description, strings.ToLower(fr.Name), fr.Preconditions)

			skill.Code = code
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// generateTools creates tool definitions from requirements.
func (e *Engine) generateTools(reqs *RequirementsDocument) ([]ToolDefinition, error) {
	var tools []ToolDefinition

	// Generate tools based on interaction methods
	toolSet := make(map[string]bool)
	for _, fr := range reqs.FunctionalRequirements {
		for _, method := range fr.InteractionMethods {
			toolKey := string(method)
			if !toolSet[toolKey] {
				toolSet[toolKey] = true
				
				tool := ToolDefinition{
					Name:        string(method) + "_tool",
					Description: fmt.Sprintf("Tool for %s interactions", method),
					Type:        "custom",
				}

				switch method {
				case InteractionAPI:
					tool.Config = map[string]string{
						"type": "http",
						"base_url": "${API_BASE_URL}",
					}
				case InteractionDatabase:
					tool.Config = map[string]string{
						"type": "database",
						"driver": "postgres",
						"dsn": "${DATABASE_URL}",
					}
				case InteractionFile:
					tool.Config = map[string]string{
						"type": "filesystem",
						"root": "${WORKSPACE_DIR}",
					}
				}

				tools = append(tools, tool)
			}
		}
	}

	return tools, nil
}

// generateMCPConfig creates MCP server configuration.
func (e *Engine) generateMCPConfig(reqs *RequirementsDocument) (*MCPConfiguration, error) {
	var mcpConfig MCPConfiguration

	// Check for MCP interaction requirements
	hasMCP := false
	for _, fr := range reqs.FunctionalRequirements {
		for _, method := range fr.InteractionMethods {
			if method == InteractionMCP {
				hasMCP = true
				break
			}
		}
		if hasMCP {
			break
		}
	}

	if hasMCP {
		mcpConfig.Servers = []MCPServerConfig{
			{
				Name: "default",
				Type: "stdio",
				Command: "mcp-server",
				Args: []string{"--config", "${MCP_CONFIG_PATH}"},
			},
		}
	}

	if len(mcpConfig.Servers) == 0 {
		return nil, nil
	}

	return &mcpConfig, nil
}

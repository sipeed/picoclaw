package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ToolVisibilityContext represents the context used to determine tool visibility.
// It contains information about the current request that can be used by filters
// to decide whether a tool should be visible to the LLM.
type ToolVisibilityContext struct {
	Channel   string         // Channel type (e.g., "telegram", "wecom", "slack")
	ChatID    string         // Unique chat identifier
	UserID    string         // User identifier (if available)
	UserRoles []string       // User roles/permissions (e.g., ["admin"], ["user"])
	Args      map[string]any // Tool execution arguments
	Timestamp time.Time      // Request timestamp
}

// ToolVisibilityFilter is a function type that determines whether a tool should be visible
// based on the provided context. Return true to make the tool visible, false to hide it.
//
// Example:
//
//	adminOnlyFilter := func(ctx ToolVisibilityContext) bool {
//	    for _, role := range ctx.UserRoles {
//	        if role == "admin" {
//	            return true
//	        }
//	    }
//	    return false
//	}
type ToolVisibilityFilter func(ctx ToolVisibilityContext) bool

type ToolRegistry struct {
	tools             map[string]Tool
	mu                sync.RWMutex
	visibilityFilters map[string]ToolVisibilityFilter // Filters per tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:             make(map[string]Tool),
		visibilityFilters: make(map[string]ToolVisibilityFilter),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
	// No filter means always visible (backward compatible)
	delete(r.visibilityFilters, tool.Name())
}

// RegisterWithFilter registers a tool with a visibility filter.
// The filter determines whether the tool definition is included when GetDefinitionsForContext is called.
// This enables fine-grained control over which tools are visible to the LLM based on context.
//
// Parameters:
//   - tool: The tool to register
//   - filter: A function that returns true if the tool should be visible, false otherwise
//
// Example:
//
//	adminTool := NewAdminTool()
//	registry.RegisterWithFilter(adminTool, func(ctx ToolVisibilityContext) bool {
//	    for _, role := range ctx.UserRoles {
//	        if role == "admin" {
//	            return true
//	        }
//	    }
//	    return false
//	})
func (r *ToolRegistry) RegisterWithFilter(tool Tool, filter ToolVisibilityFilter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
	if filter != nil {
		r.visibilityFilters[tool.Name()] = filter
	} else {
		delete(r.visibilityFilters, tool.Name())
	}
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]any) *ToolResult {
	return r.ExecuteWithContext(ctx, name, args, "", "", nil)
}

// ExecuteWithContext executes a tool with channel/chatID context and optional async callback.
// If the tool implements AsyncTool and a non-nil callback is provided,
// the callback will be set on the tool before execution.
func (r *ToolRegistry) ExecuteWithContext(
	ctx context.Context,
	name string,
	args map[string]any,
	channel, chatID string,
	asyncCallback AsyncCallback,
) *ToolResult {
	logger.InfoCF("tool", "Tool execution started",
		map[string]any{
			"tool": name,
			"args": args,
		})

	tool, ok := r.Get(name)
	if !ok {
		logger.ErrorCF("tool", "Tool not found",
			map[string]any{
				"tool": name,
			})
		return ErrorResult(fmt.Sprintf("tool %q not found", name)).WithError(fmt.Errorf("tool not found"))
	}

	// If tool implements ContextualTool, set context
	if contextualTool, ok := tool.(ContextualTool); ok && channel != "" && chatID != "" {
		contextualTool.SetContext(channel, chatID)
	}

	// If tool implements AsyncTool and callback is provided, set callback
	if asyncTool, ok := tool.(AsyncTool); ok && asyncCallback != nil {
		asyncTool.SetCallback(asyncCallback)
		logger.DebugCF("tool", "Async callback injected",
			map[string]any{
				"tool": name,
			})
	}

	start := time.Now()
	result := tool.Execute(ctx, args)
	duration := time.Since(start)

	// Log based on result type
	if result.IsError {
		logger.ErrorCF("tool", "Tool execution failed",
			map[string]any{
				"tool":     name,
				"duration": duration.Milliseconds(),
				"error":    result.ForLLM,
			})
	} else if result.Async {
		logger.InfoCF("tool", "Tool started (async)",
			map[string]any{
				"tool":     name,
				"duration": duration.Milliseconds(),
			})
	} else {
		logger.InfoCF("tool", "Tool execution completed",
			map[string]any{
				"tool":          name,
				"duration_ms":   duration.Milliseconds(),
				"result_length": len(result.ForLLM),
			})
	}

	return result
}

// sortedToolNames returns tool names in sorted order for deterministic iteration.
// This is critical for KV cache stability: non-deterministic map iteration would
// produce different system prompts and tool definitions on each call, invalidating
// the LLM's prefix cache even when no tools have changed.
func (r *ToolRegistry) sortedToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *ToolRegistry) GetDefinitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()
	definitions := make([]map[string]any, 0, len(sorted))
	for _, name := range sorted {
		definitions = append(definitions, ToolToSchema(r.tools[name]))
	}
	return definitions
}

// GetDefinitionsForContext returns tool definitions filtered by the provided context.
// Only tools whose filters return true (or have no filter) will be included.
// This allows different users/channels to see different sets of available tools.
//
// Parameters:
//   - ctx: The visibility context containing channel, chatID, user info, etc.
//
// Returns:
//   - Filtered list of tool definitions in schema format
func (r *ToolRegistry) GetDefinitionsForContext(ctx ToolVisibilityContext) []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()
	definitions := make([]map[string]any, 0, len(sorted))

	for _, name := range sorted {
		tool := r.tools[name]
		filter, hasFilter := r.visibilityFilters[name]

		// If no filter or filter returns true, include the tool
		if !hasFilter || filter(ctx) {
			definitions = append(definitions, ToolToSchema(tool))
		}
	}

	return definitions
}

// ToProviderDefs converts tool definitions to provider-compatible format.
// This is the format expected by LLM provider APIs.
func (r *ToolRegistry) ToProviderDefs() []providers.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()
	definitions := make([]providers.ToolDefinition, 0, len(sorted))
	for _, name := range sorted {
		tool := r.tools[name]
		schema := ToolToSchema(tool)

		// Safely extract nested values with type checks
		fn, ok := schema["function"].(map[string]any)
		if !ok {
			continue
		}

		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		params, _ := fn["parameters"].(map[string]any)

		definitions = append(definitions, providers.ToolDefinition{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}
	return definitions
}

// ToProviderDefsForContext converts filtered tool definitions to provider-compatible format.
// Similar to ToProviderDefs but applies context-based filtering.
func (r *ToolRegistry) ToProviderDefsForContext(ctx ToolVisibilityContext) []providers.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()
	definitions := make([]providers.ToolDefinition, 0, len(sorted))

	for _, name := range sorted {
		tool := r.tools[name]
		filter, hasFilter := r.visibilityFilters[name]

		// If no filter or filter returns true, include the tool
		if !hasFilter || filter(ctx) {
			schema := ToolToSchema(tool)

			// Safely extract nested values with type checks
			fn, ok := schema["function"].(map[string]any)
			if !ok {
				continue
			}

			name, _ := fn["name"].(string)
			desc, _ := fn["description"].(string)
			params, _ := fn["parameters"].(map[string]any)

			definitions = append(definitions, providers.ToolDefinition{
				Type: "function",
				Function: providers.ToolFunctionDefinition{
					Name:        name,
					Description: desc,
					Parameters:  params,
				},
			})
		}
	}
	return definitions
}

// List returns a list of all registered tool names.
func (r *ToolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.sortedToolNames()
}

// Count returns the number of registered tools.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// GetSummaries returns human-readable summaries of all registered tools.
// Returns a slice of "name - description" strings.
func (r *ToolRegistry) GetSummaries() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()
	summaries := make([]string, 0, len(sorted))
	for _, name := range sorted {
		tool := r.tools[name]
		summaries = append(summaries, fmt.Sprintf("- `%s` - %s", tool.Name(), tool.Description()))
	}
	return summaries
}

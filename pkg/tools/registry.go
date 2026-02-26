package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools/append_file"
	"github.com/sipeed/picoclaw/pkg/tools/edit_file"
	"github.com/sipeed/picoclaw/pkg/tools/exec"
	"github.com/sipeed/picoclaw/pkg/tools/find_skills"
	"github.com/sipeed/picoclaw/pkg/tools/i2c"
	"github.com/sipeed/picoclaw/pkg/tools/install_skill"
	"github.com/sipeed/picoclaw/pkg/tools/list_dir"
	"github.com/sipeed/picoclaw/pkg/tools/message"
	"github.com/sipeed/picoclaw/pkg/tools/read_file"
	"github.com/sipeed/picoclaw/pkg/tools/spi"
	"github.com/sipeed/picoclaw/pkg/tools/web_fetch"
	"github.com/sipeed/picoclaw/pkg/tools/web_search"
	"github.com/sipeed/picoclaw/pkg/tools/write_file"
)

type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewToolRegistry(cfg *config.Config, workspace string, restrict bool) *ToolRegistry {
	toolsRegistry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// Handle nil config (for testing)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// File tools - each with individual configuration
	if cfg.Tools.ReadFile.Enabled {
		toolsRegistry.Register(read_file.NewReadFileTool(workspace, restrict))
	}
	if cfg.Tools.WriteFile.Enabled {
		toolsRegistry.Register(write_file.NewWriteFileTool(workspace, restrict))
	}
	if cfg.Tools.EditFile.Enabled {
		toolsRegistry.Register(edit_file.NewEditFileTool(workspace, restrict))
	}
	if cfg.Tools.AppendFile.Enabled {
		toolsRegistry.Register(append_file.NewAppendFileTool(workspace, restrict))
	}
	if cfg.Tools.ListDir.Enabled {
		toolsRegistry.Register(list_dir.NewListDirTool(workspace, restrict))
	}

	// Exec tool
	if cfg.Tools.Exec.Enabled {
		toolsRegistry.Register(exec.NewExecToolWithConfig(workspace, restrict, cfg))
	}

	// Web tools
	if searchTool := web_search.NewWebSearchTool(web_search.WebSearchToolOptions{
		BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		TavilyAPIKey:         cfg.Tools.Web.Tavily.APIKey,
		TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
		TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
		TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
		PerplexityAPIKey:     cfg.Tools.Web.Perplexity.APIKey,
		PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
		PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
		Proxy:                cfg.Tools.Web.Proxy,
	}); searchTool != nil {
		toolsRegistry.Register(searchTool)
	}
	toolsRegistry.Register(web_fetch.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy))

	// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
	if cfg.Tools.I2C.Enabled {
		toolsRegistry.Register(i2c.NewI2CTool())
	}
	if cfg.Tools.SPI.Enabled {
		toolsRegistry.Register(spi.NewSPITool())
	}

	// Skill discovery and installation tools
	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})
	searchCache := skills.NewSearchCache(
		cfg.Tools.Skills.SearchCache.MaxSize,
		time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
	)
	if cfg.Tools.FindSkills.Enabled {
		toolsRegistry.Register(find_skills.NewFindSkillsTool(registryMgr, searchCache))
	}
	if cfg.Tools.InstallSkill.Enabled {
		toolsRegistry.Register(install_skill.NewInstallSkillTool(registryMgr, workspace))
	}

	// Message tool
	if cfg.Tools.Message.Enabled {
		toolsRegistry.Register(message.NewMessageTool())
	}

	// // Spawn tool
	// if cfg.Tools.Spawn.Enabled {
	// 	// Note: Spawn tool is registered separately in agent loop
	// }

	return toolsRegistry
}

// NewEmptyToolRegistry creates a tool registry without pre-registered tools.
// This is useful for testing.
func NewEmptyToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
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

func ToolToSchema(tool Tool) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
		},
	}
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

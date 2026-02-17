package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

type clientFactory func(config ServerConfig) Client

type managedServer struct {
	config ServerConfig
	client Client
}

// Manager owns MCP servers and maps discovered MCP tools to PicoClaw tools.
type Manager struct {
	mu sync.RWMutex

	servers map[string]*managedServer
	tools   map[string]RegisteredTool

	discovered bool
	newClient  clientFactory
}

func NewManager(configs map[string]ServerConfig) *Manager {
	servers := make(map[string]*managedServer, len(configs))
	for name, cfg := range configs {
		copied := cfg
		copied.Name = name
		servers[name] = &managedServer{config: copied}
	}
	return &Manager{
		servers:    servers,
		tools:      make(map[string]RegisteredTool),
		discovered: false,
		newClient: func(config ServerConfig) Client {
			return NewStdioClient(config)
		},
	}
}

// DiscoverTools starts configured MCP servers and returns discovered tool metadata.
func (m *Manager) DiscoverTools(ctx context.Context) ([]RegisteredTool, error) {
	m.mu.Lock()
	if m.discovered {
		tools := toolsFromMap(m.tools)
		m.mu.Unlock()
		return tools, nil
	}

	discoveryErrors := make([]string, 0)

	for serverName, server := range m.servers {
		client := m.newClient(server.config)
		if err := client.Start(ctx); err != nil {
			discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: %v", serverName, err))
			continue
		}

		remoteTools, err := client.ListTools(ctx)
		if err != nil {
			_ = client.Close()
			discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: %v", serverName, err))
			continue
		}

		server.client = client
		for _, remoteTool := range remoteTools {
			if !isToolAllowed(remoteTool.Name, server.config.IncludeTools, server.config.ExcludeTools) {
				continue
			}

			qualifiedName := m.makeUniqueToolName(serverName, remoteTool.Name)
			parameters := normalizeSchema(remoteTool.InputSchema)
			m.tools[qualifiedName] = RegisteredTool{
				QualifiedName: qualifiedName,
				ServerName:    serverName,
				ToolName:      remoteTool.Name,
				Description:   remoteTool.Description,
				Parameters:    parameters,
			}
		}
	}

	m.discovered = true
	tools := toolsFromMap(m.tools)
	m.mu.Unlock()

	if len(tools) == 0 && len(discoveryErrors) > 0 {
		return nil, fmt.Errorf("mcp tool discovery failed: %s", strings.Join(discoveryErrors, "; "))
	}
	return tools, nil
}

func (m *Manager) CallTool(ctx context.Context, qualifiedName string, args map[string]any) (CallResult, error) {
	m.mu.RLock()
	tool, ok := m.tools[qualifiedName]
	if !ok {
		m.mu.RUnlock()
		return CallResult{}, fmt.Errorf("mcp tool %q not found", qualifiedName)
	}

	server := m.servers[tool.ServerName]
	if server == nil || server.client == nil {
		m.mu.RUnlock()
		return CallResult{}, fmt.Errorf("mcp server %q is not active", tool.ServerName)
	}
	client := server.client
	toolName := tool.ToolName
	m.mu.RUnlock()

	if args == nil {
		args = map[string]any{}
	}
	return client.CallTool(ctx, toolName, args)
}

func (m *Manager) Close() error {
	m.mu.Lock()
	servers := make([]*managedServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	m.mu.Unlock()

	var firstErr error
	for _, server := range servers {
		if server.client == nil {
			continue
		}
		if err := server.client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) makeUniqueToolName(serverName, toolName string) string {
	base := QualifiedToolName(serverName, toolName)
	if _, exists := m.tools[base]; !exists {
		return base
	}

	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s_%d", base, index)
		if len(candidate) > qualifiedNameMaxLen {
			overflow := len(candidate) - qualifiedNameMaxLen
			if overflow < len(base) {
				candidate = base[:len(base)-overflow] + fmt.Sprintf("_%d", index)
			} else {
				candidate = candidate[:qualifiedNameMaxLen]
			}
		}
		if _, exists := m.tools[candidate]; !exists {
			return candidate
		}
	}
}

func normalizeSchema(schema map[string]any) map[string]any {
	if len(schema) == 0 {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	return schema
}

func isToolAllowed(name string, include, exclude []string) bool {
	if len(include) > 0 && !slices.Contains(include, name) {
		return false
	}
	if slices.Contains(exclude, name) {
		return false
	}
	return true
}

func toolsFromMap(tools map[string]RegisteredTool) []RegisteredTool {
	out := make([]RegisteredTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool)
	}
	return out
}

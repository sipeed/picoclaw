package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/cliui"
	"github.com/sipeed/picoclaw/pkg/config"
	picomcp "github.com/sipeed/picoclaw/pkg/mcp"
)

type toolDetail struct {
	Name        string
	Description string
	Parameters  []paramDetail
}

type paramDetail struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

var serverShowProbe = defaultServerShowProbe

func defaultServerShowProbe(
	ctx context.Context,
	name string,
	server config.MCPServerConfig,
	workspacePath string,
) ([]toolDetail, error) {
	mgr := picomcp.NewManager()
	defer func() { _ = mgr.Close() }()

	server.Enabled = true
	mcpCfg := config.MCPConfig{
		ToolConfig: config.ToolConfig{Enabled: true},
		Servers: map[string]config.MCPServerConfig{
			name: server,
		},
	}

	if err := mgr.LoadFromMCPConfig(ctx, mcpCfg, workspacePath); err != nil {
		return nil, err
	}

	conn, ok := mgr.GetServer(name)
	if !ok {
		return nil, fmt.Errorf("server %q did not register a connection", name)
	}

	details := make([]toolDetail, 0, len(conn.Tools))
	for _, tool := range conn.Tools {
		details = append(details, toolDetail{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  extractParameters(tool.InputSchema),
		})
	}
	return details, nil
}

func extractParameters(schema any) []paramDetail {
	schemaMap := normalizeSchema(schema)
	properties, ok := schemaMap["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return nil
	}

	required := make(map[string]struct{})
	switch raw := schemaMap["required"].(type) {
	case []string:
		for _, name := range raw {
			required[name] = struct{}{}
		}
	case []any:
		for _, value := range raw {
			if name, ok := value.(string); ok {
				required[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]paramDetail, 0, len(names))
	for _, name := range names {
		param := paramDetail{Name: name}
		if propMap, ok := properties[name].(map[string]any); ok {
			if typeName, ok := propMap["type"].(string); ok {
				param.Type = strings.TrimSpace(typeName)
			}
			if desc, ok := propMap["description"].(string); ok {
				param.Description = strings.TrimSpace(desc)
			}
		}
		_, param.Required = required[name]
		params = append(params, param)
	}
	return params
}

func normalizeSchema(schema any) map[string]any {
	if schema == nil {
		return map[string]any{}
	}
	if schemaMap, ok := schema.(map[string]any); ok {
		return schemaMap
	}

	var jsonData []byte
	switch raw := schema.(type) {
	case json.RawMessage:
		jsonData = raw
	case []byte:
		jsonData = raw
	default:
		var err error
		jsonData, err = json.Marshal(schema)
		if err != nil {
			return map[string]any{}
		}
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return map[string]any{}
	}
	return result
}

func newShowCommand() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show details and tools for a configured MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			name := args[0]
			server, exists := cfg.Tools.MCP.Servers[name]
			if !exists {
				return fmt.Errorf("MCP server %q not found", name)
			}

			serverInfo := buildServerInfo(name, server, cfg.Tools.MCP.Discovery.Enabled)

			if !server.Enabled {
				cliui.PrintMCPShow(cmd.OutOrStdout(), serverInfo, nil, true)
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			details, err := serverShowProbe(ctx, name, server, cfg.WorkspacePath())
			if err != nil {
				return fmt.Errorf("failed to connect to MCP server %q: %w", name, err)
			}

			tools := make([]cliui.MCPShowTool, 0, len(details))
			for _, d := range details {
				params := make([]cliui.MCPShowParam, 0, len(d.Parameters))
				for _, p := range d.Parameters {
					params = append(params, cliui.MCPShowParam{
						Name:        p.Name,
						Type:        p.Type,
						Description: p.Description,
						Required:    p.Required,
					})
				}
				tools = append(tools, cliui.MCPShowTool{
					Name:        d.Name,
					Description: d.Description,
					Parameters:  params,
				})
			}

			cliui.PrintMCPShow(cmd.OutOrStdout(), serverInfo, tools, false)
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Connection timeout")

	return cmd
}

func buildServerInfo(name string, server config.MCPServerConfig, discoveryEnabled bool) cliui.MCPShowServer {
	effectiveDeferred := discoveryEnabled
	deferredExplicit := server.Deferred != nil
	if deferredExplicit {
		effectiveDeferred = *server.Deferred
	}
	info := cliui.MCPShowServer{
		Name:              name,
		Type:              inferTransportType(server),
		Target:            renderServerTarget(server),
		Enabled:           server.Enabled,
		EffectiveDeferred: effectiveDeferred,
		DeferredExplicit:  deferredExplicit,
		EnvFile:           server.EnvFile,
	}
	if len(server.Env) > 0 {
		keys := make([]string, 0, len(server.Env))
		for k := range server.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		info.EnvKeys = keys
	}
	if len(server.Headers) > 0 {
		keys := make([]string, 0, len(server.Headers))
		for k := range server.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		info.Headers = keys
	}
	return info
}

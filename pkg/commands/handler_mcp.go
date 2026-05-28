package commands

import (
	"context"
	"fmt"
	"strings"
)

func listMCPServersHandler() Handler {
	return func(ctx context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.ListMCPServers == nil {
			return req.Reply(unavailableMsg)
		}

		servers := rt.ListMCPServers(ctx)
		if len(servers) == 0 {
			return req.Reply("No MCP servers configured")
		}

		header := "Configured MCP Servers:"
		if rt.Config != nil && !rt.Config.Tools.IsToolEnabled("mcp") {
			header = "Configured MCP Servers (integration disabled):"
		}

		lines := make([]string, 0, len(servers)*5+1)
		lines = append(lines, header)
		for idx, server := range servers {
			if idx > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, fmt.Sprintf("- `%s`", server.Name))
			lines = append(lines, fmt.Sprintf("  Enabled: %s", yesNo(server.Enabled)))
			lines = append(lines, fmt.Sprintf("  Deferred: %s", yesNo(server.Deferred)))
			lines = append(lines, fmt.Sprintf("  Connected: %s", yesNo(server.Connected)))
			if server.Connected {
				lines = append(lines, fmt.Sprintf("  Active tools: %d", server.ToolCount))
				continue
			}
			lines = append(lines, "  Active tools: unavailable")
		}

		return req.Reply(strings.Join(lines, "\n"))
	}
}

func showMCPToolsHandler() Handler {
	return func(ctx context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.ListMCPTools == nil {
			return req.Reply(unavailableMsg)
		}

		serverName := nthToken(req.Text, 2)
		if serverName == "" {
			return req.Reply("Usage: /show mcp <server>")
		}

		tools, err := rt.ListMCPTools(ctx, serverName)
		if err != nil {
			return req.Reply(err.Error())
		}
		if len(tools) == 0 {
			return req.Reply(fmt.Sprintf("MCP server '%s' has no active tools", serverName))
		}

		lines := make([]string, 0, len(tools)*6+1)
		lines = append(lines, fmt.Sprintf("Active MCP tools for `%s`:", serverName))
		for idx, tool := range tools {
			if idx > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, fmt.Sprintf("- `%s`", tool.Name))
			lines = append(lines, fmt.Sprintf("  Description: %s", tool.Description))
			if len(tool.Parameters) == 0 {
				lines = append(lines, "  Parameters: none")
				continue
			}

			lines = append(lines, "  Parameters:")
			for _, param := range tool.Parameters {
				line := fmt.Sprintf("    - `%s`", param.Name)
				if param.Type != "" {
					line += fmt.Sprintf(" (%s", param.Type)
					if param.Required {
						line += ", required"
					}
					line += ")"
				} else if param.Required {
					line += " (required)"
				}
				if param.Description != "" {
					line += ": " + param.Description
				}
				lines = append(lines, line)
			}
		}

		return req.Reply(strings.Join(lines, "\n"))
	}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

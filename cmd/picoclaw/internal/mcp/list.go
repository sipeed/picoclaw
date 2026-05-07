package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/cliui"
)

func newListCommand() *cobra.Command {
	var (
		includeStatus bool
		timeout       time.Duration
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if len(cfg.Tools.MCP.Servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
				return nil
			}

			rows := make([]cliui.MCPListRow, 0, len(cfg.Tools.MCP.Servers))
			for _, name := range sortedServerNames(cfg.Tools.MCP.Servers) {
				server := cfg.Tools.MCP.Servers[name]
				status := "disabled"
				if server.Enabled {
					status = "enabled"
				}

				if includeStatus && server.Enabled {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					result, probeErr := serverProbe(ctx, name, server, cfg.WorkspacePath())
					cancel()
					if probeErr != nil {
						status = "error"
					} else {
						status = fmt.Sprintf("ok (%d tools)", result.ToolCount)
					}
				}

				effectiveDeferred := cfg.Tools.MCP.Discovery.Enabled
				deferredExplicit := server.Deferred != nil
				if deferredExplicit {
					effectiveDeferred = *server.Deferred
				}

				rows = append(rows, cliui.MCPListRow{
					Name:              name,
					Type:              inferTransportType(server),
					Target:            renderServerTarget(server),
					Status:            status,
					EffectiveDeferred: effectiveDeferred,
					DeferredExplicit:  deferredExplicit,
				})
			}

			cliui.PrintMCPList(cmd.OutOrStdout(), rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&includeStatus, "status", false, "Ping enabled servers and show live status")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "Timeout for each live status check")

	return cmd
}

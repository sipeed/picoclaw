package tracer

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/tracer"
)

func NewTracerCommand() *cobra.Command {
	var port int
	var logPath string
	var frontendDir string

	cmd := &cobra.Command{
		Use:     "tracer",
		Aliases: []string{"t"},
		Short:   "Start the trace viewer UI",
		Long: `Start a web UI that shows real-time LLM traces from the running gateway.

Reads ~/.picoclaw/logs/gateway.log and displays per-turn LLM calls,
system prompts, messages, tool definitions, and tool executions.

Run alongside the gateway:
  picoclaw gateway --debug --no-truncate
  picoclaw tracer`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return tracer.Run(port, logPath, frontendDir)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 7331, "Port to listen on")
	cmd.Flags().StringVar(&logPath, "log", "", "Path to gateway.log (default: ~/.picoclaw/logs/gateway.log)")
	cmd.Flags().StringVar(&frontendDir, "frontend-dir", "", "Path to frontend/dist directory (dev only)")

	return cmd
}

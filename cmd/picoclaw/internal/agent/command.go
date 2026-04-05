package agent

import (
	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	var (
		message    string
		sessionKey string
		model      string
		debug      bool
		verbose    bool
		showTools  bool
		showThink  bool
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Interact with the agent directly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Use enhanced interactive mode with logging if any verbose flag is set
			if verbose || showTools || showThink {
				return agentCmdWithLogging(message, sessionKey, model, debug, verbose, showTools, showThink)
			}
			return agentCmd(message, sessionKey, model, debug)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose progress logs")
	cmd.Flags().BoolVar(&showTools, "show-tools", false, "Show tool execution logs")
	cmd.Flags().BoolVar(&showThink, "show-think", false, "Show thinking process indicators")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "cli:default", "Session key")
	cmd.Flags().StringVarP(&model, "model", "", "", "Model to use")

	cmd.SetUsageTemplate(`Usage:
  {{.CommandPath}} [flags]

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Examples:
  # Interactive mode (default)
  {{.CommandPath}}

  # Interactive mode with verbose logging
  {{.CommandPath}} --verbose

  # Show tool execution logs
  {{.CommandPath}} --show-tools

  # Show thinking process
  {{.CommandPath}} --show-think

  # Send a single message
  {{.CommandPath}} -m "What is the weather today?"

  # Combine multiple logging options
  {{.CommandPath}} --verbose --show-tools --show-think
`)

	return cmd
}

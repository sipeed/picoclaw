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
		remoteURL  string
		token      string
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Interact with the agent directly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if remoteURL != "" {
				remoteSessionKey := sessionKey
				if !cmd.Flags().Changed("session") {
					remoteSessionKey = ""
				}
				return remoteAgentCmd(
					cmd.Context(),
					remoteURL,
					token,
					message,
					remoteSessionKey,
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
				)
			}
			return agentCmd(message, sessionKey, model, debug)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "cli:default", "Session key")
	cmd.Flags().StringVarP(&model, "model", "", "", "Model to use")
	cmd.Flags().StringVar(&remoteURL, "remote", "", "Connect to a remote Pico WebSocket URL")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for remote Pico auth")

	return cmd
}

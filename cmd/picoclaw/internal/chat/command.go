package chat

import (
	"github.com/spf13/cobra"
)

func NewChatCommand() *cobra.Command {
	var (
		message    string
		sessionKey string
		gatewayURL string
		debug      bool
	)

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with a running picoclaw gateway",
		Long: `Connect to a running picoclaw gateway and send messages interactively.

The gateway must already be started with 'picoclaw gateway'.
By default the gateway URL is read from the config file (~/.picoclaw/config.json).

Examples:
  picoclaw chat                          # interactive REPL
  picoclaw chat -m "what time is it?"   # single message
  picoclaw chat --url http://host:18790  # connect to remote gateway`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return chatCmd(message, sessionKey, gatewayURL, debug)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "cli:default", "Session key for conversation history")
	cmd.Flags().StringVarP(&gatewayURL, "url", "u", "", "Gateway base URL (default: from config, e.g. http://127.0.0.1:18790)")
	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	return cmd
}




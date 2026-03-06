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
		// Workspace and config overrides
		workspace string
		configDir string
		tools     string
		skills    string
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Interact with the agent directly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return agentCmd(message, sessionKey, model, debug,
				workspace, configDir, tools, skills)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "", "Session key for conversation isolation (e.g. stackId:conversationId)")
	cmd.Flags().StringVarP(&model, "model", "", "", "Model to use")

	// Workspace and config overrides
	cmd.Flags().StringVar(&workspace, "workspace", "", "Override agent workspace directory")
	cmd.Flags().StringVar(&configDir, "config-dir", "", "Directory containing config.json (model/agent/tool overrides) and bootstrap files (AGENTS.md, IDENTITY.md, SOUL.md, USER.md)")
	cmd.Flags().StringVar(&tools, "tools", "", "Comma-separated tool allowlist (only these tools enabled)")
	cmd.Flags().StringVar(&skills, "skills", "", "Comma-separated skill filter (only these skills loaded)")

	return cmd
}

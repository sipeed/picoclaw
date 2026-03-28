package auth

import "github.com/spf13/cobra"

func newLoginCommand() *cobra.Command {
	var (
		provider      string
		useDeviceCode bool
		useOauth      bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login via OAuth or paste token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return authLoginCmd(provider, useDeviceCode, useOauth)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider to login with (openai, anthropic, google-antigravity, openrouter, groq, deepseek, mistral, …)")
	cmd.Flags().BoolVar(&useDeviceCode, "device-code", false, "Use device code flow (for headless environments)")
	cmd.Flags().BoolVar(
		&useOauth, "setup-token", false,
		"Use setup-token flow for Anthropic (from `claude setup-token`)",
	)
	_ = cmd.MarkFlagRequired("provider")

	return cmd
}

package auth

import "github.com/spf13/cobra"

func newModelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Show available models",
		RunE: func(_ *cobra.Command, _ []string) error {
			return authModelsCmd()
		},
	}

	return cmd
}

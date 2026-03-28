package status

import (
	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"s"},
		Short:   "Show Jane AI status",
		Run: func(cmd *cobra.Command, args []string) {
			statusCmd()
		},
	}

	return cmd
}

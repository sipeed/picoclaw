package evolutioncmd

import "github.com/spf13/cobra"

func NewEvolutionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evolution",
		Short: "Manage self-evolving skills",
	}

	cmd.AddCommand(
		newDraftsCommand(),
		newReviewCommand(),
		newApplyCommand(),
		newRollbackCommand(),
		newStatusCommand(),
		newRunOnceCommand(),
		newPruneCommand(),
	)
	return cmd
}

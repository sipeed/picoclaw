package skills

import (
	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search available skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			skillsSearchCmd(query)
			return nil
		},
	}

	return cmd
}

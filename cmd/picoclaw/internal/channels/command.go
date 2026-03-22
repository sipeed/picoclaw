package channels

import "github.com/spf13/cobra"

func NewChannelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channels",
		Aliases: []string{"channel"},
		Short:   "Manage chat channel logins and configuration helpers",
	}

	cmd.AddCommand(newWeixinCommand())

	return cmd
}

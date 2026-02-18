package cron

import "github.com/spf13/cobra"

func newEnableCommand(storePath func() string, disable bool) *cobra.Command {
	name := "enable"
	short := "Enable a job"
	if disable {
		name = "disable"
		short = "Disable a job"
	}

	return &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cronEnableCmd(storePath(), disable, args[0])
			return nil
		},
	}
}

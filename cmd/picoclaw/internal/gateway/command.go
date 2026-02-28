package gateway

import (
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	var (
		debug         bool
		orchestration bool
		enableStats   bool
	)

	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"g"},
		Short:   "Start picoclaw gateway",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return gatewayCmd(debug, orchestration, enableStats)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolVar(&orchestration, "orchestration", false, "Enable subagent orchestration")
	cmd.Flags().BoolVar(&enableStats, "stats", false, "Enable stats tracking")

	return cmd
}

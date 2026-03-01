package dashboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:embed web/index.html
var embeddedFiles embed.FS

func NewDashboardCommand() *cobra.Command {
	var host string
	var port int
	var noBrowser bool

	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "ui"},
		Short:   "Start the web-based configuration dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDashboard(host, port, !noBrowser)
		},
	}

	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host to bind to")
	cmd.Flags().IntVarP(&port, "port", "p", 18795, "Port to listen on")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open the browser automatically")

	return cmd
}

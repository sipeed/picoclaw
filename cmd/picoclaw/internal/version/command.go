package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"jane/cmd/picoclaw/internal"
	"jane/pkg/config"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			printVersion()
		},
	}

	return cmd
}

func printVersion() {
	fmt.Printf("%s picoclaw %s\n", internal.Logo, config.FormatVersion())
	build, goVer := config.FormatBuildInfo()
	if build != "" {
		fmt.Printf("  Build: %s\n", build)
	}
	if goVer != "" {
		fmt.Printf("  Go: %s\n", goVer)
	}
}

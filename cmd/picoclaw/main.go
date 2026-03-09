// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/agent"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/auth"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/cron"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/gateway"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/migrate"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/onboard"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/plugins"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/skills"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/status"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/version"
)

func NewPicoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:                "picoclaw",
		Short:              short,
		Example:            "picoclaw list",
		Args:               rootArgsValidator,
		ValidArgsFunction:  rootCompleteArgs,
		DisableFlagParsing: true, // Pass all args (including flags) directly to plugins/commands
		RunE: func(cmd *cobra.Command, args []string) error {
			// With DisableFlagParsing, --help is just an arg - show help
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
				return cmd.Help()
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorBlue = "\033[1;38;2;62;93;185m"
	colorRed  = "\033[1;38;2;213;70;70m"
	banner    = "\r\n" +
		colorBlue + "██████╗ ██╗ ██████╗ ██████╗ " + colorRed + " ██████╗██╗      █████╗ ██╗    ██╗\n" +
		colorBlue + "██╔══██╗██║██╔════╝██╔═══██╗" + colorRed + "██╔════╝██║     ██╔══██╗██║    ██║\n" +
		colorBlue + "██████╔╝██║██║     ██║   ██║" + colorRed + "██║     ██║     ███████║██║ █╗ ██║\n" +
		colorBlue + "██╔═══╝ ██║██║     ██║   ██║" + colorRed + "██║     ██║     ██╔══██║██║███╗██║\n" +
		colorBlue + "██║     ██║╚██████╗╚██████╔╝" + colorRed + "╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		colorBlue + "╚═╝     ╚═╝ ╚═════╝ ╚═════╝ " + colorRed + " ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n " +
		"\033[0m\r\n"
)

// rootArgsValidator checks if args should be routed to a plugin
func rootArgsValidator(cmd *cobra.Command, args []string) error {
	// No args - let cobra show help
	if len(args) == 0 {
		return nil
	}

	// Check if it's a known subcommand
	knownCommands := map[string]bool{
		"onboard": true,
		"agent":   true,
		"auth":    true,
		"gateway": true,
		"status":  true,
		"cron":    true,
		"migrate": true,
		"skills":  true,
		"version": true,
		"help":    true,
	}

	if knownCommands[args[0]] {
		return nil
	}

	// Try to find a plugin with this name
	pluginPath, err := plugins.FindPlugin(args[0])
	if err != nil {
		// If it looks like a flag (starts with "-"), let cobra handle it
		if len(args[0]) > 0 && args[0][0] == '-' {
			return nil
		}
		// Not a known command and not a plugin - print error and exit
		fmt.Fprintf(os.Stderr, "picoclaw: %q is not a picoclaw command. See 'picoclaw --help'.\n", args[0])
		os.Exit(1)
	}

	// Execute the plugin with ALL remaining args (including any flags like --help)
	plugins.ExecPlugin(pluginPath, args[1:])
	// Should not reach here
	return nil
}

// rootCompleteArgs provides completion for plugin names
func rootCompleteArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get list of plugins for completion
	pluginList, err := plugins.ListPlugins()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Also include built-in commands for shell completion
	builtIn := []string{
		"onboard", "agent", "auth", "gateway",
		"status", "cron", "migrate", "skills",
		"plugins", "version",
	}

	// Combine both lists
	result := append(builtIn, pluginList...)
	return result, cobra.ShellCompDirectiveNoFileComp
}

func main() {
	// Print banner unless PICOCLAW_NO_BANNER=1
	if os.Getenv("PICOCLAW_NO_BANNER") != "1" {
		fmt.Printf("%s", banner)
	}
	cmd := NewPicoclawCommand()

	// With DisableFlagParsing, intercept global flags before plugin routing
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" {
			// Convert --version to "version" subcommand
			newArgs := append([]string{os.Args[0], "version"}, os.Args[2:]...)
			cmd.SetArgs(newArgs[1:])
		}
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

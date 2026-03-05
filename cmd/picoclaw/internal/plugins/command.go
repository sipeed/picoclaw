package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

const pluginsDirName = "plugins"

// pluginArgsValidator validates args and runs plugins for unknown subcommands
func pluginArgsValidator(cmd *cobra.Command, args []string) error {
	// If no args, let cobra handle it (show help)
	if len(args) == 0 {
		return nil
	}

	// Check if it's a known subcommand (handled by cobra)
	knownSubcommands := map[string]bool{
		"list": true,
	}

	if knownSubcommands[args[0]] {
		return nil
	}

	// Try to find and run the plugin
	pluginPath, err := findPlugin(args[0])
	if err != nil {
		// Plugin not found - return error to show "unknown command"
		return fmt.Errorf("unknown command %q", args[0])
	}

	// Run the plugin directly (this will exit)
	execPlugin(pluginPath, args[1:])
	// Should not reach here
	return nil
}

func NewPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "plugins",
		Short:             "Manage and run plugins",
		Long:              pluginsLongHelp,
		Args:              pluginArgsValidator,
		ValidArgsFunction: completePluginArgs,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If we get here with args (after Args validation passed), show help
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
	)

	return cmd
}

const pluginsLongHelp = `Manage and run plugins from ~/.picoclaw/plugins

Routes commands to executable plugins in the plugins directory.
Each plugin is an executable file in ~/.picoclaw/plugins.

Examples:
  picoclaw plugins list                    # List available plugins
  picoclaw plugins service restart         # Run 'service' plugin with 'restart'
  picoclaw plugins service status         # Run 'service' plugin with 'status'`

func getPluginsDir() string {
	return filepath.Join(internal.GetPicoclawHome(), pluginsDirName)
}

func listPlugins() ([]string, error) {
	pluginsDir := getPluginsDir()

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if executable by anyone
		if info.Mode()&0o111 != 0 {
			plugins = append(plugins, entry.Name())
		}
	}

	return plugins, nil
}

func findPlugin(name string) (string, error) {
	// Don't treat flags as plugins - let caller handle them
	if len(name) > 0 && name[0] == '-' {
		return "", fmt.Errorf("not a plugin: %q", name)
	}

	pluginsDir := getPluginsDir()

	// Try exact match first
	pluginPath := filepath.Join(pluginsDir, name)
	if info, err := os.Stat(pluginPath); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		return pluginPath, nil
	}

	// Try common extensions
	extensions := []string{".sh", ".bash", ".py", ".go", ""}
	for _, ext := range extensions {
		extPath := pluginPath + ext
		if info, err := os.Stat(extPath); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return extPath, nil
		}
	}

	// Try prefix match - if user types "X", match "X-*" or "*-X"
	entries, err := os.ReadDir(pluginsDir)
	if err == nil {
		var matches []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue
			}
			pluginName := entry.Name()
			// Check if name is a prefix or suffix of plugin name
			if strings.HasPrefix(pluginName, name+"-") ||
				strings.HasPrefix(pluginName, name) ||
				strings.HasSuffix(pluginName, "-"+name) {
				matches = append(matches, pluginName)
			}
		}
		if len(matches) == 1 {
			return filepath.Join(pluginsDir, matches[0]), nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("multiple plugins match %q: %v", name, matches)
		}
	}

	return "", fmt.Errorf("plugin %q not found", name)
}

func execPlugin(pluginPath string, args []string) {
	execCmd := exec.Command(pluginPath, args...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Env = os.Environ()

	if err := execCmd.Run(); err != nil {
		// Check if it's an exit error
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		// Otherwise it's some other error
		fmt.Fprintf(os.Stderr, "Error running plugin: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func completePluginArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	plugins, err := listPlugins()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return plugins, cobra.ShellCompDirectiveNoFileComp
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plugins, err := listPlugins()
			if err != nil {
				return fmt.Errorf("failed to list plugins: %w", err)
			}

			cmd.Println("Available plugins in ", internal.GetConfigPath())
			cmd.Println("")

			// Show the plugins directory
			pluginsDir := getPluginsDir()
			cmd.Printf("Plugins directory: %s\n", pluginsDir)
			cmd.Println("")

			if len(plugins) == 0 {
				cmd.Println("  (no executable plugins found)")
				return nil
			}

			cmd.Println("Plugins:")
			for _, plugin := range plugins {
				cmd.Printf("  %s\n", plugin)
			}

			return nil
		},
	}
}

// FindPlugin returns the path to a plugin by name, or an error if not found
func FindPlugin(name string) (string, error) {
	return findPlugin(name)
}

// ExecPlugin executes a plugin with the given arguments
func ExecPlugin(pluginPath string, args []string) {
	execPlugin(pluginPath, args)
}

// ListPlugins returns a list of available plugin names
func ListPlugins() ([]string, error) {
	return listPlugins()
}

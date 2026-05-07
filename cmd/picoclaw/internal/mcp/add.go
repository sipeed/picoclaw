package mcp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/config"
)

type addOptions struct {
	Env       []string
	EnvFile   string
	Headers   []string
	Transport string
	Force     bool
	Deferred  *bool // nil = not set, true = deferred, false = not deferred
}

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "add [flags] <name> <command-or-url> [args...]",
		Short:              "Add or update an MCP server",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, name, target, targetArgs, showHelp, err := parseAddArgs(args)
			if showHelp {
				return cmd.Help()
			}
			if err != nil {
				return err
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if cfg.Tools.MCP.Servers == nil {
				cfg.Tools.MCP.Servers = make(map[string]config.MCPServerConfig)
			}

			if _, exists := cfg.Tools.MCP.Servers[name]; exists && !opts.Force {
				var overwrite bool

				overwrite, err = confirmOverwrite(cmd.InOrStdin(), cmd.OutOrStdout(), name)
				if err != nil {
					return fmt.Errorf("failed to confirm overwrite: %w", err)
				}
				if !overwrite {
					return fmt.Errorf("aborted: MCP server %q already exists", name)
				}
			}

			server, err := buildServerConfig(target, targetArgs, opts)
			if err != nil {
				return err
			}

			cfg.Tools.MCP.Enabled = true
			cfg.Tools.MCP.Servers[name] = server

			if err := saveValidatedConfig(cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server %q saved.\n", name)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringArrayP("env", "e", nil, "Environment variable in KEY=value format (repeatable, saved to config)")
	flags.String("env-file", "", "Path to an env file for stdio servers (recommended for secrets)")
	flags.StringArrayP("header", "H", nil, "HTTP header in 'Name: Value' or 'Name=Value' format (repeatable)")
	flags.StringP("transport", "t", "stdio", "Transport type: stdio, http, or sse")
	flags.BoolP("force", "f", false, "Overwrite an existing server without prompting")
	flags.Bool("deferred", false, "Mark server as deferred (tools hidden until explicitly activated)")
	flags.Bool("no-deferred", false, "Mark server as non-deferred (tools always active)")

	return cmd
}

func parseAddArgs(args []string) (addOptions, string, string, []string, bool, error) {
	opts := addOptions{Transport: "stdio"}
	var positional []string
	serverArgs := make([]string, 0)
	explicitCommand := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--help" || arg == "-h":
			return addOptions{}, "", "", nil, true, nil
		case arg == "--":
			if i+1 < len(args) {
				explicitCommand = append(explicitCommand, args[i+1:]...)
			}
			i = len(args)
		case arg == "--force" || arg == "-f":
			opts.Force = true
		case arg == "--deferred":
			t := true
			opts.Deferred = &t
		case arg == "--no-deferred":
			f := false
			opts.Deferred = &f
		case arg == "--transport" || arg == "-t":
			if i+1 >= len(args) {
				return addOptions{}, "", "", nil, false, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.Transport = args[i]
		case strings.HasPrefix(arg, "--transport="):
			opts.Transport = strings.TrimPrefix(arg, "--transport=")
		case arg == "--env" || arg == "-e":
			if i+1 >= len(args) {
				return addOptions{}, "", "", nil, false, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.Env = append(opts.Env, args[i])
		case arg == "--env-file":
			if i+1 >= len(args) {
				return addOptions{}, "", "", nil, false, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.EnvFile = args[i]
		case strings.HasPrefix(arg, "--env="):
			opts.Env = append(opts.Env, strings.TrimPrefix(arg, "--env="))
		case strings.HasPrefix(arg, "--env-file="):
			opts.EnvFile = strings.TrimPrefix(arg, "--env-file=")
		case arg == "--header" || arg == "-H":
			if i+1 >= len(args) {
				return addOptions{}, "", "", nil, false, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.Headers = append(opts.Headers, args[i])
		case strings.HasPrefix(arg, "--header="):
			opts.Headers = append(opts.Headers, strings.TrimPrefix(arg, "--header="))
		case strings.HasPrefix(arg, "-") && len(positional) >= 2:
			serverArgs = append(serverArgs, args[i:]...)
			i = len(args)
		default:
			positional = append(positional, arg)
		}
	}

	if len(explicitCommand) > 0 {
		if len(positional) != 1 {
			return addOptions{}, "", "", nil, false, fmt.Errorf(
				"usage: picoclaw mcp add [flags] <name> <command-or-url> [args...] or picoclaw mcp add [flags] <name> -- <command> [args...]",
			)
		}
		if len(explicitCommand) == 0 {
			return addOptions{}, "", "", nil, false, fmt.Errorf("missing stdio command after --")
		}
		return opts, positional[0], explicitCommand[0], explicitCommand[1:], false, nil
	}

	if len(positional) < 2 {
		return addOptions{}, "", "", nil, false, fmt.Errorf(
			"usage: picoclaw mcp add [flags] <name> <command-or-url> [args...] or picoclaw mcp add [flags] <name> -- <command> [args...]",
		)
	}

	targetArgs := make([]string, 0, len(positional)-2+len(serverArgs))
	targetArgs = append(targetArgs, positional[2:]...)
	targetArgs = append(targetArgs, serverArgs...)

	return opts, positional[0], positional[1], targetArgs, false, nil
}

func buildServerConfig(target string, args []string, opts addOptions) (config.MCPServerConfig, error) {
	transport := strings.ToLower(strings.TrimSpace(opts.Transport))
	if transport == "" {
		transport = "stdio"
	}
	switch transport {
	case "stdio", "http", "sse":
	default:
		return config.MCPServerConfig{}, fmt.Errorf("unsupported transport %q", opts.Transport)
	}

	env, err := parseEnvAssignments(opts.Env)
	if err != nil {
		return config.MCPServerConfig{}, err
	}
	headers, err := parseHeaderAssignments(opts.Headers)
	if err != nil {
		return config.MCPServerConfig{}, err
	}

	server := config.MCPServerConfig{
		Enabled:  true,
		Type:     transport,
		Deferred: opts.Deferred,
	}

	switch transport {
	case "http", "sse":
		if len(env) > 0 {
			return config.MCPServerConfig{}, fmt.Errorf("--env can only be used with stdio transport")
		}
		if strings.TrimSpace(opts.EnvFile) != "" {
			return config.MCPServerConfig{}, fmt.Errorf("--env-file can only be used with stdio transport")
		}
		if len(args) > 0 {
			return config.MCPServerConfig{}, fmt.Errorf("%s transport does not accept command arguments", transport)
		}
		parsedURL, err := url.ParseRequestURI(target)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return config.MCPServerConfig{}, fmt.Errorf("invalid MCP URL %q", target)
		}
		server.URL = target
		server.Headers = headers
		return server, nil
	}

	if len(headers) > 0 {
		return config.MCPServerConfig{}, fmt.Errorf("--header can only be used with http or sse transport")
	}

	if looksLikeRemoteURL(target) {
		return config.MCPServerConfig{}, fmt.Errorf(
			"target %q looks like a remote MCP URL, but transport is %q. Use --transport http or --transport sse",
			target,
			transport,
		)
	}

	command := target
	commandArgs := append([]string(nil), args...)

	if err := validateLocalCommandPath(target); err != nil {
		return config.MCPServerConfig{}, err
	}
	if isLocalCommandPath(command) {
		command = expandHomePath(command)
	}

	server.Command = command
	server.Args = commandArgs
	server.Env = env
	server.EnvFile = strings.TrimSpace(opts.EnvFile)

	return server, nil
}

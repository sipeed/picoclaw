package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	svcmgr "github.com/sipeed/picoclaw/pkg/service"
)

type serviceLogsOptions struct {
	Lines int
	Follow bool
}

func serviceCmd() {
	args := os.Args[2:]
	if len(args) == 0 {
		serviceHelp()
		return
	}

	sub := strings.ToLower(args[0])
	if sub == "help" || sub == "--help" || sub == "-h" {
		serviceHelp()
		return
	}

	exePath, err := resolveServiceExecutablePath(os.Args[0], exec.LookPath, os.Executable)
	if err != nil {
		fmt.Printf("Error resolving executable path: %v\n", err)
		os.Exit(1)
	}

	mgr, err := svcmgr.NewManager(exePath)
	if err != nil {
		fmt.Printf("Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	switch sub {
	case "install":
		prepareServiceInstallEnvPath()
		if err := mgr.Install(); err != nil {
			fmt.Printf("Service install failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service installed")
		fmt.Printf("  Start with: %s service start\n", invokedCLIName())
	case "refresh":
		if err := runServiceRefresh(mgr); err != nil {
			fmt.Printf("Service refresh failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service refreshed")
		fmt.Printf("  Reinstalled and restarted (run: %s service status)\n", invokedCLIName())
	case "uninstall", "remove":
		if err := mgr.Uninstall(); err != nil {
			fmt.Printf("Service uninstall failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service uninstalled")
	case "start":
		if err := mgr.Start(); err != nil {
			fmt.Printf("Service start failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service started")
	case "stop":
		if err := mgr.Stop(); err != nil {
			fmt.Printf("Service stop failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service stopped")
	case "restart":
		if err := mgr.Restart(); err != nil {
			fmt.Printf("Service restart failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Service restarted")
	case "status":
		st, err := mgr.Status()
		if err != nil {
			fmt.Printf("Service status check failed: %v\n", err)
			os.Exit(1)
		}
		printServiceStatus(st)
	case "logs":
		opts, showHelp, err := parseServiceLogsOptions(args[1:])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			serviceHelp()
			os.Exit(2)
		}
		if showHelp {
			serviceHelp()
			return
		}
		if opts.Follow {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			if err := mgr.LogsFollow(ctx, opts.Lines, os.Stdout); err != nil && ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "Service logs failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			out, err := mgr.Logs(opts.Lines)
			if err != nil {
				fmt.Printf("Service logs failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(out)
		}
	default:
		fmt.Printf("Unknown service command: %s\n", sub)
		serviceHelp()
		os.Exit(2)
	}
}

func prepareServiceInstallEnvPath() {
	cfg, err := loadConfig()
	if err != nil || cfg == nil {
		return
	}
	venvBin := strings.TrimSpace(workspaceVenvBinDir(cfg.WorkspacePath()))
	if venvBin == "" {
		return
	}
	if _, err := os.Stat(venvBin); err != nil {
		return
	}
	prependPathEnv(venvBin)
}

func prependPathEnv(pathEntry string) {
	pathEntry = strings.TrimSpace(pathEntry)
	if pathEntry == "" {
		return
	}
	sep := string(os.PathListSeparator)
	current := os.Getenv("PATH")
	parts := []string{pathEntry}
	for _, p := range strings.Split(current, sep) {
		p = strings.TrimSpace(p)
		if p == "" || p == pathEntry {
			continue
		}
		parts = append(parts, p)
	}
	_ = os.Setenv("PATH", strings.Join(parts, sep))
}

func runServiceRefresh(mgr svcmgr.Manager) error {
	prepareServiceInstallEnvPath()
	if err := mgr.Install(); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	if err := mgr.Restart(); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}
	return nil
}

func resolveServiceExecutablePath(
	argv0 string,
	lookPath func(string) (string, error),
	executable func() (string, error),
) (string, error) {
	arg0 := strings.TrimSpace(argv0)

	if arg0 != "" && (strings.Contains(arg0, "/") || strings.Contains(arg0, `\`)) {
		if abs, err := filepath.Abs(arg0); err == nil {
			return abs, nil
		}
		return arg0, nil
	}

	base := strings.TrimSpace(filepath.Base(arg0))
	if base != "" {
		if resolved, err := lookPath(base); err == nil && strings.TrimSpace(resolved) != "" {
			if abs, err := filepath.Abs(resolved); err == nil {
				return abs, nil
			}
			return resolved, nil
		}
	}

	return executable()
}

func serviceHelp() {
	commandName := invokedCLIName()
	fmt.Println("\nService commands:")
	fmt.Println("  install             Install background gateway service")
	fmt.Println("  refresh             Reinstall + restart service after upgrades")
	fmt.Println("  uninstall           Remove background gateway service")
	fmt.Println("  start               Start background gateway service")
	fmt.Println("  stop                Stop background gateway service")
	fmt.Println("  restart             Restart background gateway service")
	fmt.Println("  status              Show service install/runtime status")
	fmt.Println("  logs                Show recent service logs")
	fmt.Println()
	fmt.Println("Logs options:")
	fmt.Println("  -n, --lines <N>     Number of log lines to show (default: 100)")
	fmt.Println("  -f, --follow         Follow log output (like tail -f); Ctrl+C to stop")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s service install\n", commandName)
	fmt.Printf("  %s service refresh\n", commandName)
	fmt.Printf("  %s service start\n", commandName)
	fmt.Printf("  %s service status\n", commandName)
	fmt.Printf("  %s service logs --lines 200\n", commandName)
	fmt.Printf("  %s service logs -f\n", commandName)
}

func parseServiceLogsOptions(args []string) (serviceLogsOptions, bool, error) {
	opts := serviceLogsOptions{Lines: 100}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--lines":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("%s requires a value", args[i])
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return opts, false, fmt.Errorf("invalid value for %s: %q", args[i], args[i+1])
			}
			opts.Lines = n
			i++
		case "-f", "--follow":
			opts.Follow = true
		case "help", "--help", "-h":
			return opts, true, nil
		default:
			return opts, false, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, false, nil
}

func printServiceStatus(st svcmgr.Status) {
	yn := func(v bool) string {
		if v {
			return "yes"
		}
		return "no"
	}

	fmt.Println("\nGateway service status:")
	fmt.Printf("  Backend:   %s\n", st.Backend)
	fmt.Printf("  Installed: %s\n", yn(st.Installed))
	fmt.Printf("  Running:   %s\n", yn(st.Running))
	fmt.Printf("  Enabled:   %s\n", yn(st.Enabled))
	if strings.TrimSpace(st.Detail) != "" {
		fmt.Printf("  Detail:    %s\n", st.Detail)
	}
}

func workspaceVenvBinDir(workspace string) string {
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	return filepath.Join(workspace, ".venv", "bin")
}

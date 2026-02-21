// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/tui"
	"github.com/sipeed/picoclaw/pkg/update"
)

func agentCmd() {
	message := ""
	sessionKey := "cli:default"
	modelOverride := ""

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Interact with the agent directly")
			fmt.Println()
			fmt.Println("Usage: picoclaw agent [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  -m, --message <text>   Send a single message (non-interactive)")
			fmt.Println("  -s, --session <key>    Session key (default: cli:default)")
			fmt.Println("  --model <model>        Override the default model")
			fmt.Println("  -d, --debug            Enable debug logging")
			fmt.Println("  -h, --help             Show this help message")
			return
		case "--debug", "-d":
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-s", "--session":
			if i+1 < len(args) {
				sessionKey = args[i+1]
				i++
			}
		case "--model", "-model":
			if i+1 < len(args) {
				modelOverride = args[i+1]
				i++
			}
		default:
			fmt.Printf("Unknown flag: %s\n", args[i])
			fmt.Println("Run 'picoclaw agent --help' for usage.")
			os.Exit(1)
		}
	}

	// Check for updates in the background (non-blocking, at most once per 24h)
	updateHint := make(chan string, 1)
	go func() {
		updateHint <- update.CheckHint(version)
	}()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Run 'picoclaw doctor' to check for common problems.")
		os.Exit(1)
	}

	if modelOverride != "" {
		cfg.Agents.Defaults.Model = modelOverride
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		fmt.Println("Run 'picoclaw doctor' to check for common problems.")
		os.Exit(1)
	}
	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.Model = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Set up CLI permission prompt for workspace access
	cliPermFn := tools.NewCLIPermissionFunc(os.Stdin, os.Stdout)
	agentLoop.SetPermissionFuncFactory(func(channel, chatID string) tools.PermissionFunc {
		if channel == "cli" {
			return cliPermFn
		}
		return nil // Other channels fall back to LLM-driven flow
	})

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      startupInfo["tools"].(map[string]any)["count"],
			"skills_total":     startupInfo["skills"].(map[string]any)["total"],
			"skills_available": startupInfo["skills"].(map[string]any)["available"],
		})

	// Print update hint if available (non-blocking select)
	printUpdateHint := func() {
		select {
		case v := <-updateHint:
			if v != "" {
				fmt.Printf("\nA new version (%s) is available. Run 'picoclaw update' to upgrade.\n\n", v)
			}
		default:
		}
	}

	if message != "" {
		ctx := context.Background()
		spin := newSpinner("Thinking...")
		spin.Start()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		spin.Stop()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n%s %s\n", logo, response)
		printUpdateHint()
	} else {
		printUpdateHint()

		// Suppress Go's default log output during TUI mode.
		// The logger package uses log.Println which writes to stderr,
		// corrupting the bubbletea alt-screen rendering.
		log.SetOutput(io.Discard)
		defer log.SetOutput(os.Stderr)

		modelName := cfg.Agents.Defaults.Model
		p := tea.NewProgram(
			tui.NewModel(agentLoop, sessionKey, modelName),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v\n", err)
			os.Exit(1)
		}
	}
}

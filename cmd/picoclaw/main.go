// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/swarm"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
)

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

const logo = "🦞"

// formatVersion returns the version string with optional git commit
func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// formatBuildInfo returns build time and go version info
func formatBuildInfo() (build string, goVer string) {
	if buildTime != "" {
		build = buildTime
	}
	goVer = goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return
}

func printVersion() {
	fmt.Printf("%s picoclaw %s\n", logo, formatVersion())
	build, goVer := formatBuildInfo()
	if build != "" {
		fmt.Printf("  Build: %s\n", build)
	}
	if goVer != "" {
		fmt.Printf("  Go: %s\n", goVer)
	}
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "onboard":
		onboard()
	case "agent":
		agentCmd()
	case "gateway":
		gatewayCmd()
	case "status":
		statusCmd()
	case "migrate":
		migrateCmd()
	case "auth":
		authCmd()
	case "cron":
		cronCmd()
	case "swarm":
		swarmCmd()
	case "skills":
		if len(os.Args) < 3 {
			skillsHelp()
			return
		}

		subcommand := os.Args[2]

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		workspace := cfg.WorkspacePath()
		installer := skills.NewSkillInstaller(workspace)
		// 获取全局配置目录和内置 skills 目录
		globalDir := filepath.Dir(getConfigPath())
		globalSkillsDir := filepath.Join(globalDir, "skills")
		builtinSkillsDir := filepath.Join(globalDir, "picoclaw", "skills")
		skillsLoader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

		switch subcommand {
		case "list":
			skillsListCmd(skillsLoader)
		case "install":
			skillsInstallCmd(installer, cfg)
		case "remove", "uninstall":
			if len(os.Args) < 4 {
				fmt.Println("Usage: picoclaw skills remove <skill-name>")
				return
			}
			skillsRemoveCmd(installer, os.Args[3])
		case "install-builtin":
			skillsInstallBuiltinCmd(workspace)
		case "list-builtin":
			skillsListBuiltinCmd()
		case "search":
			skillsSearchCmd(installer)
		case "show":
			if len(os.Args) < 4 {
				fmt.Println("Usage: picoclaw skills show <skill-name>")
				return
			}
			skillsShowCmd(skillsLoader, os.Args[3])
		default:
			fmt.Printf("Unknown skills command: %s\n", subcommand)
			skillsHelp()
		}
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%s picoclaw - Personal AI Assistant v%s\n\n", logo, version)
	fmt.Println("Usage: picoclaw <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  onboard     Initialize picoclaw configuration and workspace")
	fmt.Println("  agent       Interact with the agent directly")
	fmt.Println("  auth        Manage authentication (login, logout, status)")
	fmt.Println("  gateway     Start picoclaw gateway")
	fmt.Println("  status      Show picoclaw status")
	fmt.Println("  cron        Manage scheduled tasks")
	fmt.Println("  migrate     Migrate from OpenClaw to PicoClaw")
	fmt.Println("  skills      Manage skills (install, list, remove)")
	fmt.Println("  swarm       Run in swarm mode (multi-instance collaboration)")
	fmt.Println("  version     Show version information")
}

func onboard() {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	fmt.Printf("%s picoclaw is ready!\n", logo)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your API key to", configPath)
	fmt.Println("     Get one at: https://openrouter.ai/keys")
	fmt.Println("  2. Chat: picoclaw agent -m \"Hello!\"")
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func migrateCmd() {
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		migrateHelp()
		return
	}

	opts := migrate.Options{}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--config-only":
			opts.ConfigOnly = true
		case "--workspace-only":
			opts.WorkspaceOnly = true
		case "--force":
			opts.Force = true
		case "--refresh":
			opts.Refresh = true
		case "--openclaw-home":
			if i+1 < len(args) {
				opts.OpenClawHome = args[i+1]
				i++
			}
		case "--picoclaw-home":
			if i+1 < len(args) {
				opts.PicoClawHome = args[i+1]
				i++
			}
		default:
			fmt.Printf("Unknown flag: %s\n", args[i])
			migrateHelp()
			os.Exit(1)
		}
	}

	result, err := migrate.Run(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !opts.DryRun {
		migrate.PrintSummary(result)
	}
}

func migrateHelp() {
	fmt.Println("\nMigrate from OpenClaw to PicoClaw")
	fmt.Println()
	fmt.Println("Usage: picoclaw migrate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --dry-run          Show what would be migrated without making changes")
	fmt.Println("  --refresh          Re-sync workspace files from OpenClaw (repeatable)")
	fmt.Println("  --config-only      Only migrate config, skip workspace files")
	fmt.Println("  --workspace-only   Only migrate workspace files, skip config")
	fmt.Println("  --force            Skip confirmation prompts")
	fmt.Println("  --openclaw-home    Override OpenClaw home directory (default: ~/.openclaw)")
	fmt.Println("  --picoclaw-home    Override PicoClaw home directory (default: ~/.picoclaw)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw migrate              Detect and migrate from OpenClaw")
	fmt.Println("  picoclaw migrate --dry-run    Show what would be migrated")
	fmt.Println("  picoclaw migrate --refresh    Re-sync workspace files")
	fmt.Println("  picoclaw migrate --force      Migrate without confirmation")
}

func agentCmd() {
	message := ""
	sessionKey := "cli:default"
	var hid, sid string

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
		case "--hid", "--identity-hid":
			if i+1 < len(args) {
				hid = args[i+1]
				i++
			}
		case "--sid", "--identity-sid":
			if i+1 < len(args) {
				sid = args[i+1]
				i++
			}
		case "--identity":
			if i+1 < len(args) {
				// Parse "hid/sid" format
				identityParts := strings.SplitN(args[i+1], "/", 2)
				hid = identityParts[0]
				if len(identityParts) > 1 {
					sid = identityParts[1]
				}
				i++
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Set identity if provided
	if hid != "" || sid != "" {
		agentLoop.SetIdentity(hid, sid)
		if sid != "" {
			logger.Info(fmt.Sprintf("Identity set: hid=%s, sid=%s", hid, sid))
		} else {
			logger.Info(fmt.Sprintf("Identity set: hid=%s", hid))
		}
	}

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]interface{}{
			"tools_count":      startupInfo["tools"].(map[string]interface{})["count"],
			"skills_total":     startupInfo["skills"].(map[string]interface{})["total"],
			"skills_available": startupInfo["skills"].(map[string]interface{})["available"],
		})

	if message != "" {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n%s %s\n", logo, response)
	} else {
		fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", logo)
		interactiveMode(agentLoop, sessionKey)
	}
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(fmt.Sprintf("%s You: ", logo))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

func gatewayCmd() {
	// Check for --debug flag
	args := os.Args[2:]
	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
			break
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]interface{})
	skillsInfo := startupInfo["skills"].(map[string]interface{})
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	// Log to file as well
	logger.InfoCF("agent", "Agent initialized",
		map[string]interface{}{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// Setup cron tool and service
	cronService := setupCronTool(agentLoop, msgBus, cfg.WorkspacePath())

	// Setup swarm info tool
	setupSwarmInfoTool(agentLoop, cfg)

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)
	heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		response, err := agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		// For heartbeat, always return silent - the subagent result will be
		// sent to user via processSystemMessage when the async task completes
		return tools.SilentResult(response)
	})

	channelManager, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		fmt.Printf("Error creating channel manager: %v\n", err)
		os.Exit(1)
	}

	var transcriber *voice.GroqTranscriber
	if cfg.Providers.Groq.APIKey != "" {
		transcriber = voice.NewGroqTranscriber(cfg.Providers.Groq.APIKey)
		logger.InfoC("voice", "Groq voice transcription enabled")
	}

	if transcriber != nil {
		if telegramChannel, ok := channelManager.GetChannel("telegram"); ok {
			if tc, ok := telegramChannel.(*channels.TelegramChannel); ok {
				tc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Telegram channel")
			}
		}
		if discordChannel, ok := channelManager.GetChannel("discord"); ok {
			if dc, ok := discordChannel.(*channels.DiscordChannel); ok {
				dc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Discord channel")
			}
		}
		if slackChannel, ok := channelManager.GetChannel("slack"); ok {
			if sc, ok := slackChannel.(*channels.SlackChannel); ok {
				sc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Slack channel")
			}
		}
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cronService.Start(); err != nil {
		fmt.Printf("Error starting cron service: %v\n", err)
	}
	fmt.Println("✓ Cron service started")

	if err := heartbeatService.Start(); err != nil {
		fmt.Printf("Error starting heartbeat service: %v\n", err)
	}
	fmt.Println("✓ Heartbeat service started")

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	fmt.Println("✓ Gateway stopped")
}

func statusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := getConfigPath()

	fmt.Printf("%s picoclaw Status\n", logo)
	fmt.Printf("Version: %s\n", formatVersion())
	build, _ := formatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.Model)

		hasOpenRouter := cfg.Providers.OpenRouter.APIKey != ""
		hasAnthropic := cfg.Providers.Anthropic.APIKey != ""
		hasOpenAI := cfg.Providers.OpenAI.APIKey != ""
		hasGemini := cfg.Providers.Gemini.APIKey != ""
		hasZhipu := cfg.Providers.Zhipu.APIKey != ""
		hasGroq := cfg.Providers.Groq.APIKey != ""
		hasVLLM := cfg.Providers.VLLM.APIBase != ""

		status := func(enabled bool) string {
			if enabled {
				return "✓"
			}
			return "not set"
		}
		fmt.Println("OpenRouter API:", status(hasOpenRouter))
		fmt.Println("Anthropic API:", status(hasAnthropic))
		fmt.Println("OpenAI API:", status(hasOpenAI))
		fmt.Println("Gemini API:", status(hasGemini))
		fmt.Println("Zhipu API:", status(hasZhipu))
		fmt.Println("Groq API:", status(hasGroq))
		if hasVLLM {
			fmt.Printf("vLLM/Local: ✓ %s\n", cfg.Providers.VLLM.APIBase)
		} else {
			fmt.Println("vLLM/Local: not set")
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}

func authCmd() {
	if len(os.Args) < 3 {
		authHelp()
		return
	}

	switch os.Args[2] {
	case "login":
		authLoginCmd()
	case "logout":
		authLogoutCmd()
	case "status":
		authStatusCmd()
	default:
		fmt.Printf("Unknown auth command: %s\n", os.Args[2])
		authHelp()
	}
}

func authHelp() {
	fmt.Println("\nAuth commands:")
	fmt.Println("  login       Login via OAuth or paste token")
	fmt.Println("  logout      Remove stored credentials")
	fmt.Println("  status      Show current auth status")
	fmt.Println()
	fmt.Println("Login options:")
	fmt.Println("  --provider <name>    Provider to login with (openai, anthropic)")
	fmt.Println("  --device-code        Use device code flow (for headless environments)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw auth login --provider openai")
	fmt.Println("  picoclaw auth login --provider openai --device-code")
	fmt.Println("  picoclaw auth login --provider anthropic")
	fmt.Println("  picoclaw auth logout --provider openai")
	fmt.Println("  picoclaw auth status")
}

func authLoginCmd() {
	provider := ""
	useDeviceCode := false

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--device-code":
			useDeviceCode = true
		}
	}

	if provider == "" {
		fmt.Println("Error: --provider is required")
		fmt.Println("Supported providers: openai, anthropic")
		return
	}

	switch provider {
	case "openai":
		authLoginOpenAI(useDeviceCode)
	case "anthropic":
		authLoginPasteToken(provider)
	default:
		fmt.Printf("Unsupported provider: %s\n", provider)
		fmt.Println("Supported providers: openai, anthropic")
	}
}

func authLoginOpenAI(useDeviceCode bool) {
	cfg := auth.OpenAIOAuthConfig()

	var cred *auth.AuthCredential
	var err error

	if useDeviceCode {
		cred, err = auth.LoginDeviceCode(cfg)
	} else {
		cred, err = auth.LoginBrowser(cfg)
	}

	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential("openai", cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		appCfg.Providers.OpenAI.AuthMethod = "oauth"
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("Login successful!")
	if cred.AccountID != "" {
		fmt.Printf("Account: %s\n", cred.AccountID)
	}
}

func authLoginPasteToken(provider string) {
	cred, err := auth.LoginPasteToken(provider, os.Stdin)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential(provider, cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		switch provider {
		case "anthropic":
			appCfg.Providers.Anthropic.AuthMethod = "token"
		case "openai":
			appCfg.Providers.OpenAI.AuthMethod = "token"
		}
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Printf("Token saved for %s!\n", provider)
}

func authLogoutCmd() {
	provider := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		}
	}

	if provider != "" {
		if err := auth.DeleteCredential(provider); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			switch provider {
			case "openai":
				appCfg.Providers.OpenAI.AuthMethod = ""
			case "anthropic":
				appCfg.Providers.Anthropic.AuthMethod = ""
			}
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Printf("Logged out from %s\n", provider)
	} else {
		if err := auth.DeleteAllCredentials(); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			appCfg.Providers.OpenAI.AuthMethod = ""
			appCfg.Providers.Anthropic.AuthMethod = ""
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Println("Logged out from all providers")
	}
}

func authStatusCmd() {
	store, err := auth.LoadStore()
	if err != nil {
		fmt.Printf("Error loading auth store: %v\n", err)
		return
	}

	if len(store.Credentials) == 0 {
		fmt.Println("No authenticated providers.")
		fmt.Println("Run: picoclaw auth login --provider <name>")
		return
	}

	fmt.Println("\nAuthenticated Providers:")
	fmt.Println("------------------------")
	for provider, cred := range store.Credentials {
		status := "active"
		if cred.IsExpired() {
			status = "expired"
		} else if cred.NeedsRefresh() {
			status = "needs refresh"
		}

		fmt.Printf("  %s:\n", provider)
		fmt.Printf("    Method: %s\n", cred.AuthMethod)
		fmt.Printf("    Status: %s\n", status)
		if cred.AccountID != "" {
			fmt.Printf("    Account: %s\n", cred.AccountID)
		}
		if !cred.ExpiresAt.IsZero() {
			fmt.Printf("    Expires: %s\n", cred.ExpiresAt.Format("2006-01-02 15:04"))
		}
	}
}

func getConfigPath() string {
	// Try current directory first: ./.picoclaw/config.json
	localConfigPath := filepath.Join(".picoclaw", "config.json")
	if _, err := os.Stat(localConfigPath); err == nil {
		return localConfigPath
	}

	// Fallback to home directory: ~/.picoclaw/config.json
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func setupCronTool(agentLoop *agent.AgentLoop, msgBus *bus.MessageBus, workspace string) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}

func setupSwarmInfoTool(agentLoop *agent.AgentLoop, cfg *config.Config) {
	// Create and register the swarm info tool
	swarmInfoTool := tools.NewSwarmInfoTool()

	// Add known workers
	swarmInfoTool.AddWorker("coordinator", "coordinator", []string{"orchestration", "scheduling"}, "/Users/dev/service/coordinator")
	swarmInfoTool.AddWorker("worker-a", "worker", []string{"code", "macos"}, "/Users/dev/service/worker-a")
	swarmInfoTool.AddWorker("worker-b", "worker", []string{"search", "windows"}, "/Users/dev/service/worker-b")

	// Register the tool
	agentLoop.RegisterTool(swarmInfoTool)

	logger.InfoC("swarm", "Swarm info tool registered")
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

func cronCmd() {
	if len(os.Args) < 3 {
		cronHelp()
		return
	}

	subcommand := os.Args[2]

	// Load config to get workspace path
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	cronStorePath := filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")

	switch subcommand {
	case "list":
		cronListCmd(cronStorePath)
	case "add":
		cronAddCmd(cronStorePath)
	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: picoclaw cron remove <job_id>")
			return
		}
		cronRemoveCmd(cronStorePath, os.Args[3])
	case "enable":
		cronEnableCmd(cronStorePath, false)
	case "disable":
		cronEnableCmd(cronStorePath, true)
	default:
		fmt.Printf("Unknown cron command: %s\n", subcommand)
		cronHelp()
	}
}

func cronHelp() {
	fmt.Println("\nCron commands:")
	fmt.Println("  list              List all scheduled jobs")
	fmt.Println("  add              Add a new scheduled job")
	fmt.Println("  remove <id>       Remove a job by ID")
	fmt.Println("  enable <id>      Enable a job")
	fmt.Println("  disable <id>     Disable a job")
	fmt.Println()
	fmt.Println("Add options:")
	fmt.Println("  -n, --name       Job name")
	fmt.Println("  -m, --message    Message for agent")
	fmt.Println("  -e, --every      Run every N seconds")
	fmt.Println("  -c, --cron       Cron expression (e.g. '0 9 * * *')")
	fmt.Println("  -d, --deliver     Deliver response to channel")
	fmt.Println("  --to             Recipient for delivery")
	fmt.Println("  --channel        Channel for delivery")
}

func cronListCmd(storePath string) {
	cs := cron.NewCronService(storePath, nil)
	jobs := cs.ListJobs(true) // Show all jobs, including disabled

	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return
	}

	fmt.Println("\nScheduled Jobs:")
	fmt.Println("----------------")
	for _, job := range jobs {
		var schedule string
		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			schedule = fmt.Sprintf("every %ds", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			schedule = job.Schedule.Expr
		} else {
			schedule = "one-time"
		}

		nextRun := "scheduled"
		if job.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*job.State.NextRunAtMS)
			nextRun = nextTime.Format("2006-01-02 15:04")
		}

		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}

		fmt.Printf("  %s (%s)\n", job.Name, job.ID)
		fmt.Printf("    Schedule: %s\n", schedule)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Next run: %s\n", nextRun)
	}
}

func cronAddCmd(storePath string) {
	name := ""
	message := ""
	var everySec *int64
	cronExpr := ""
	deliver := false
	channel := ""
	to := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-e", "--every":
			if i+1 < len(args) {
				var sec int64
				fmt.Sscanf(args[i+1], "%d", &sec)
				everySec = &sec
				i++
			}
		case "-c", "--cron":
			if i+1 < len(args) {
				cronExpr = args[i+1]
				i++
			}
		case "-d", "--deliver":
			deliver = true
		case "--to":
			if i+1 < len(args) {
				to = args[i+1]
				i++
			}
		case "--channel":
			if i+1 < len(args) {
				channel = args[i+1]
				i++
			}
		}
	}

	if name == "" {
		fmt.Println("Error: --name is required")
		return
	}

	if message == "" {
		fmt.Println("Error: --message is required")
		return
	}

	if everySec == nil && cronExpr == "" {
		fmt.Println("Error: Either --every or --cron must be specified")
		return
	}

	var schedule cron.CronSchedule
	if everySec != nil {
		everyMS := *everySec * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	}

	cs := cron.NewCronService(storePath, nil)
	job, err := cs.AddJob(name, schedule, message, deliver, channel, to)
	if err != nil {
		fmt.Printf("Error adding job: %v\n", err)
		return
	}

	fmt.Printf("✓ Added job '%s' (%s)\n", job.Name, job.ID)
}

func cronRemoveCmd(storePath, jobID string) {
	cs := cron.NewCronService(storePath, nil)
	if cs.RemoveJob(jobID) {
		fmt.Printf("✓ Removed job %s\n", jobID)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func cronEnableCmd(storePath string, disable bool) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: picoclaw cron enable/disable <job_id>")
		return
	}

	jobID := os.Args[3]
	cs := cron.NewCronService(storePath, nil)
	enabled := !disable

	job := cs.EnableJob(jobID, enabled)
	if job != nil {
		status := "enabled"
		if disable {
			status = "disabled"
		}
		fmt.Printf("✓ Job '%s' %s\n", job.Name, status)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func skillsHelp() {
	fmt.Println("\nSkills commands:")
	fmt.Println("  list                    List installed skills")
	fmt.Println("  install <repo>          Install skill from GitHub")
	fmt.Println("  install-builtin          Install all builtin skills to workspace")
	fmt.Println("  list-builtin             List available builtin skills")
	fmt.Println("  remove <name>           Remove installed skill")
	fmt.Println("  search                  Search available skills")
	fmt.Println("  show <name>             Show skill details")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw skills list")
	fmt.Println("  picoclaw skills install sipeed/picoclaw-skills/weather")
	fmt.Println("  picoclaw skills install-builtin")
	fmt.Println("  picoclaw skills list-builtin")
	fmt.Println("  picoclaw skills remove weather")
}

func skillsListCmd(loader *skills.SkillsLoader) {
	allSkills := loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
}

func skillsInstallCmd(installer *skills.SkillInstaller) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: picoclaw skills install <github-repo>")
		fmt.Println("Example: picoclaw skills install sipeed/picoclaw-skills/weather")
		return
	}

	repo := os.Args[3]
	fmt.Printf("Installing skill from %s...\n", repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := installer.InstallFromGitHub(ctx, repo); err != nil {
		fmt.Printf("✗ Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' installed successfully!\n", filepath.Base(repo))
}

func skillsRemoveCmd(installer *skills.SkillInstaller, skillName string) {
	fmt.Printf("Removing skill '%s'...\n", skillName)

	if err := installer.Uninstall(skillName); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
}

func skillsInstallBuiltinCmd(workspace string) {
	builtinSkillsDir := "./picoclaw/skills"
	workspaceSkillsDir := filepath.Join(workspace, "skills")

	fmt.Printf("Copying builtin skills to workspace...\n")

	skillsToInstall := []string{
		"weather",
		"news",
		"stock",
		"calculator",
	}

	for _, skillName := range skillsToInstall {
		builtinPath := filepath.Join(builtinSkillsDir, skillName)
		workspacePath := filepath.Join(workspaceSkillsDir, skillName)

		if _, err := os.Stat(builtinPath); err != nil {
			fmt.Printf("⊘ Builtin skill '%s' not found: %v\n", skillName, err)
			continue
		}

		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			fmt.Printf("✗ Failed to create directory for %s: %v\n", skillName, err)
			continue
		}

		if err := copyDirectory(builtinPath, workspacePath); err != nil {
			fmt.Printf("✗ Failed to copy %s: %v\n", skillName, err)
		}
	}

	fmt.Println("\n✓ All builtin skills installed!")
	fmt.Println("Now you can use them in your workspace.")
}

func skillsListBuiltinCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	builtinSkillsDir := filepath.Join(filepath.Dir(cfg.WorkspacePath()), "picoclaw", "skills")

	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		fmt.Printf("Error reading builtin skills: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No builtin skills available.")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillName := entry.Name()
			skillFile := filepath.Join(builtinSkillsDir, skillName, "SKILL.md")

			description := "No description"
			if _, err := os.Stat(skillFile); err == nil {
				data, err := os.ReadFile(skillFile)
				if err == nil {
					content := string(data)
					if idx := strings.Index(content, "\n"); idx > 0 {
						firstLine := content[:idx]
						if strings.Contains(firstLine, "description:") {
							descLine := strings.Index(content[idx:], "\n")
							if descLine > 0 {
								description = strings.TrimSpace(content[idx+descLine : idx+descLine])
							}
						}
					}
				}
			}
			status := "✓"
			fmt.Printf("  %s  %s\n", status, entry.Name())
			if description != "" {
				fmt.Printf("     %s\n", description)
			}
		}
	}
}

func skillsSearchCmd(installer *skills.SkillInstaller) {
	fmt.Println("Searching for available skills...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	availableSkills, err := installer.ListAvailableSkills(ctx)
	if err != nil {
		fmt.Printf("✗ Failed to fetch skills list: %v\n", err)
		return
	}

	if len(availableSkills) == 0 {
		fmt.Println("No skills available.")
		return
	}

	fmt.Printf("\nAvailable Skills (%d):\n", len(availableSkills))
	fmt.Println("--------------------")
	for _, skill := range availableSkills {
		fmt.Printf("  📦 %s\n", skill.Name)
		fmt.Printf("     %s\n", skill.Description)
		fmt.Printf("     Repo: %s\n", skill.Repository)
		if skill.Author != "" {
			fmt.Printf("     Author: %s\n", skill.Author)
		}
		if len(skill.Tags) > 0 {
			fmt.Printf("     Tags: %v\n", skill.Tags)
		}
		fmt.Println()
	}
}

func skillsShowCmd(loader *skills.SkillsLoader, skillName string) {
	content, ok := loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
}

// ==================== Swarm Commands ====================

func swarmCmd() {
	if len(os.Args) < 3 {
		swarmHelp()
		return
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "start":
		swarmStartCmd()
	case "stop":
		swarmStopCmd()
	case "dispatch":
		swarmDispatchCmd()
	case "status":
		swarmStatusCmd()
	case "nodes":
		swarmNodesCmd()
	case "result":
		swarmResultCmd()
	default:
		fmt.Printf("Unknown swarm command: %s\n", subcommand)
		swarmHelp()
	}
}

func swarmHelp() {
	fmt.Println("\nSwarm commands:")
	fmt.Println("  start       Start swarm node")
	fmt.Println("  stop        Stop running swarm node")
	fmt.Println("  dispatch    Submit a task to the swarm")
	fmt.Println("  status      Show swarm configuration")
	fmt.Println("  nodes       List discovered nodes (requires running node)")
	fmt.Println()
	fmt.Println("Start options:")
	fmt.Println("  --role <role>         Node role: coordinator, worker, specialist")
	fmt.Println("  --capabilities <list> Comma-separated capabilities")
	fmt.Println("  --embedded            Use embedded NATS server (development mode)")
	fmt.Println("  --debug               Enable debug logging")
	fmt.Println("  --hid <id>            Human/Owner ID (tenant identifier)")
	fmt.Println("  --sid <id>            Shrimp/Service ID (instance identifier)")
	fmt.Println("  --identity <hid/sid>  Both IDs in one parameter")
	fmt.Println()
	fmt.Println("Dispatch options:")
	fmt.Println("  --type <type>         Task type: direct, workflow, broadcast")
	fmt.Println("  --capability <cap>    Required capability for routing")
	fmt.Println("  --timeout <ms>        Task timeout in milliseconds")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw swarm start --role coordinator --embedded")
	fmt.Println("  picoclaw swarm start --role worker --capabilities code,search")
	fmt.Println("  picoclaw swarm start --role worker --hid alice --sid worker1")
	fmt.Println("  picoclaw swarm start --role worker --identity alice/worker1")
	fmt.Println("  picoclaw swarm dispatch --type direct 'Analyze this code' --capability code")
	fmt.Println("  picoclaw swarm status")
}

func swarmStartCmd() {
	// Parse flags
	role := "worker"
	capabilities := []string{}
	embedded := false
	var hid, sid, natsServer, temporalServer string

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--role", "-r":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--capabilities", "-c":
			if i+1 < len(args) {
				capabilities = strings.Split(args[i+1], ",")
				i++
			}
		case "--embedded":
			embedded = true
		case "--debug", "-d":
			logger.SetLevel(logger.DEBUG)
			fmt.Println("Debug mode enabled")
		case "--hid", "--identity-hid":
			if i+1 < len(args) {
				hid = args[i+1]
				i++
			}
		case "--sid", "--identity-sid":
			if i+1 < len(args) {
				sid = args[i+1]
				i++
			}
		case "--identity":
			if i+1 < len(args) {
				// Parse "hid/sid" format
				identityParts := strings.SplitN(args[i+1], "/", 2)
				hid = identityParts[0]
				if len(identityParts) > 1 {
					sid = identityParts[1]
				}
				i++
			}
		case "--nats-server", "--nats":
			if i+1 < len(args) {
				natsServer = args[i+1]
				i++
			}
		case "--temporal", "--temporal-server":
			if i+1 < len(args) {
				temporalServer = args[i+1]
				i++
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override config with CLI flags
	cfg.Swarm.Enabled = true
	cfg.Swarm.Role = role
	if len(capabilities) > 0 {
		cfg.Swarm.Capabilities = capabilities
	}
	cfg.Swarm.NATS.Embedded = embedded

	// Override NATS server if provided
	if natsServer != "" {
		cfg.Swarm.NATS.URLs = []string{"nats://" + natsServer}
	}

	// Override Temporal server if provided
	if temporalServer != "" {
		cfg.Swarm.Temporal.Host = temporalServer
	}

	// Set identity if provided
	if hid != "" {
		cfg.Swarm.HID = hid
	}
	if sid != "" {
		cfg.Swarm.SID = sid
	}

	// Create provider
	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	// Create message bus and agent loop
	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Register swarm info tool for worker/coordinator agents
	swarmInfoTool := tools.NewSwarmInfoTool()
	swarmInfoTool.AddWorker("coordinator", "coordinator", []string{"orchestration", "scheduling"}, "/Users/dev/service/coordinator")
	swarmInfoTool.AddWorker("worker-a", "worker", []string{"code", "macos"}, "/Users/dev/service/worker-a")
	swarmInfoTool.AddWorker("worker-b", "worker", []string{"search", "windows"}, "/Users/dev/service/worker-b")
	agentLoop.RegisterTool(swarmInfoTool)
	logger.InfoC("swarm", "Swarm info tool registered for worker")

	// Create and start swarm manager
	manager := swarm.NewManager(cfg, agentLoop, provider, msgBus)
	if manager == nil {
		fmt.Println("Error: failed to create swarm manager (invalid configuration)")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := manager.Start(ctx); err != nil {
		fmt.Printf("Error starting swarm: %v\n", err)
		os.Exit(1)
	}

	nodeInfo := manager.GetNodeInfo()
	fmt.Printf("%s Swarm node started\n", logo)
	fmt.Printf("  Node ID: %s\n", nodeInfo.ID)
	fmt.Printf("  Role: %s\n", nodeInfo.Role)
	fmt.Printf("  Capabilities: %v\n", nodeInfo.Capabilities)
	if embedded {
		fmt.Println("  Mode: Embedded NATS (development)")
	}
	fmt.Printf("  NATS: %v\n", manager.IsNATSConnected())
	fmt.Printf("  Temporal: %v\n", manager.IsTemporalConnected())
	fmt.Println("\nPress Ctrl+C to stop")

	// Start agent loop in background
	// For coordinator, disable auto-consume since coordinator handles message routing
	if role == "coordinator" {
		agentLoop.AutoConsume = false
	}
	go agentLoop.Run(ctx)

	// For coordinator role, also start channel manager (Telegram, etc.)
	if role == "coordinator" {
		channelManager, err := channels.NewManager(cfg, msgBus)
		if err != nil {
			fmt.Printf("Error creating channel manager: %v\n", err)
			os.Exit(1)
		}

		// Start channels in background
		if err := channelManager.StartAll(ctx); err != nil {
			fmt.Printf("Error starting channel manager: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			channelManager.StopAll(ctx)
		}()

		// Get enabled channels
		enabledChannels := channelManager.GetEnabledChannels()
		fmt.Printf("  Channels: %v\n", enabledChannels)
	}

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	manager.Stop()
	agentLoop.Stop()
	fmt.Printf("%s Swarm node stopped\n", logo)
}

func swarmStopCmd() {
	fmt.Printf("%s Stopping swarm node...\n", logo)

	// Find and stop swarm processes
	pids, err := findSwarmProcesses()
	if err != nil {
		fmt.Printf("Error finding swarm processes: %v\n", err)
		return
	}

	if len(pids) == 0 {
		fmt.Println("No running swarm nodes found")
		return
	}

	fmt.Printf("Found %d swarm node(s)\n", len(pids))
	for _, pid := range pids {
		fmt.Printf("  Stopping PID %d...\n", pid)
		if err := stopProcess(pid); err != nil {
			fmt.Printf("    Error: %v\n", err)
		} else {
			fmt.Printf("    Stopped\n")
		}
	}
	fmt.Printf("%s Swarm node(s) stopped\n", logo)
}

func swarmDispatchCmd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: picoclaw swarm dispatch <prompt> [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --type <type>         Task type: direct, workflow, broadcast (default: workflow)")
		fmt.Println("  --capability <cap>    Required capability for routing")
		fmt.Println("  --timeout <ms>        Task timeout in milliseconds (default: 300000)")
		fmt.Println("  --wait, -w            Wait for result and display it")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  picoclaw swarm dispatch 'Analyze all node files' --type workflow")
		fmt.Println("  picoclaw swarm dispatch 'Read coordinator-info.txt and worker-a-info.txt in parallel' --wait")
		return
	}

	// Parse arguments
	taskType := "workflow"  // 默认使用 workflow 来启用任务拆分
	capability := "general"
	timeout := 600000 // 10 minutes (默认超时)
	prompt := ""
	waitForResult := false

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				taskType = args[i+1]
				i++
			}
		case "--capability", "-c":
			if i+1 < len(args) {
				capability = args[i+1]
				i++
			}
		case "--timeout", "-t":
			if i+1 < len(args) {
				var ms int
				fmt.Sscanf(args[i+1], "%d", &ms)
				timeout = ms
				i++
			}
		case "--wait", "-w":
			waitForResult = true
		default:
			if prompt == "" {
				prompt = args[i]
			}
		}
	}

	if prompt == "" {
		fmt.Println("Error: prompt is required")
		return
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// For workflow type, use Temporal
	if taskType == "workflow" {
		dispatchWorkflowTask(cfg, prompt, capability, timeout, waitForResult)
		return
	}

	// For direct type, execute locally (fallback)
	dispatchLocalTask(cfg, prompt, capability, timeout)
}

func dispatchWorkflowTask(cfg *config.Config, prompt, capability string, timeout int, waitForResult bool) {
	fmt.Printf("%s Dispatching workflow task...\n", logo)
	fmt.Printf("  Type: workflow (with decomposition)")
	fmt.Printf("  Capability: %s\n", capability)
	fmt.Printf("  Timeout: %d ms\n", timeout)
	fmt.Printf("  Prompt: %s\n", truncateForDisplay(prompt, 60))
	fmt.Println()

	// Import temporal client packages
	// We'll use go-temporal client to start workflow
	workflowID := fmt.Sprintf("task-%d", time.Now().UnixNano())

	// Create task JSON
	taskJSON := fmt.Sprintf(`{"id":"%s","prompt":"%s","capability":"%s","type":"workflow"}`,
		workflowID, escapeJSON(prompt), capability)

	// Use temporal CLI to start workflow
	cmd := exec.Command("temporal", "workflow", "start",
		"--address", cfg.Swarm.Temporal.Host,
		"--namespace", cfg.Swarm.Temporal.Namespace,
		"--task-queue", cfg.Swarm.Temporal.TaskQueue,
		"--type", "SwarmWorkflow",
		"--input", taskJSON,
		"--workflow-id", workflowID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error starting workflow: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		return
	}

	fmt.Printf("\n✓ Workflow started\n")
	fmt.Printf("  Workflow ID: %s\n", workflowID)
	fmt.Printf("  Temporal UI: http://localhost:8088/namespaces/%s/workflows/%s\n",
		cfg.Swarm.Temporal.Namespace, workflowID)

	if waitForResult {
		fmt.Printf("\n⏳ Waiting for result...\n")
		waitForWorkflowCompletion(cfg, workflowID, timeout*2) // Double timeout for wait mode
	} else {
		fmt.Printf("\n💡 Use 'temporal workflow describe %s' to check status\n", workflowID)
		fmt.Printf("💡 Use 'picoclaw swarm result %s' to get result\n", workflowID)
	}
}

func waitForWorkflowCompletion(cfg *config.Config, workflowID string, timeout int) {
	start := time.Now()
	timeoutDuration := time.Duration(timeout) * time.Millisecond

	for time.Since(start) < timeoutDuration {
		cmd := exec.Command("temporal", "workflow", "describe",
			"--address", cfg.Swarm.Temporal.Host,
			"--namespace", cfg.Swarm.Temporal.Namespace,
			"--workflow-id", workflowID,
			"--output", "json")

		output, err := cmd.Output()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		// Parse JSON to check status
		var result map[string]interface{}
		if err := json.Unmarshal(output, &result); err == nil {
			if status, ok := result["workflowExecutionInfo"].(map[string]interface{})["status"].(string); ok {
				if status == "COMPLETED" {
					// Try multiple ways to extract result
					if rawResult, ok := result["result"].(map[string]interface{}); ok {
						if value, ok := rawResult["value"].(string); ok {
							fmt.Printf("\n%s Result:\n", logo)
							fmt.Println(value)
							return
						}
						if data, ok := rawResult["data"].(string); ok {
							fmt.Printf("\n%s Result:\n", logo)
							fmt.Println(data)
							return
						}
					}
					fmt.Printf("\n%s Result:\n", logo)
					fmt.Printf("  (Completed - use Temporal UI for full output)\n")
					return
				} else if status == "FAILED" {
					fmt.Printf("\n❌ Workflow failed\n")
					return
				} else if status == "CANCELED" {
					fmt.Printf("\n❌ Workflow canceled\n")
					return
				}
				// Still running
				fmt.Printf(".")
				time.Sleep(2 * time.Second)
			}
		}
	}
	fmt.Printf("\n⏱ Timeout waiting for result\n")
	fmt.Printf("💡 Check status: temporal workflow describe --namespace %s %s\n",
		cfg.Swarm.Temporal.Namespace, workflowID)
}

func dispatchLocalTask(cfg *config.Config, prompt, capability string, timeout int) {
	// Create provider and agent loop
	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		return
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Execute task locally
	fmt.Printf("%s Executing task locally...\n", logo)
	fmt.Printf("  Capability: %s\n", capability)
	fmt.Printf("  Timeout: %d ms\n", timeout)
	fmt.Printf("  Prompt: %s\n", prompt)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	response, err := agentLoop.ProcessDirect(ctx, prompt, "swarm:dispatch")

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n%s Result:\n", logo)
	fmt.Println(response)
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func swarmResultCmd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: picoclaw swarm result <workflow-id>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  picoclaw swarm result task-1234567890")
		return
	}

	workflowID := os.Args[3]

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	cmd := exec.Command("temporal", "workflow", "describe",
		"--address", cfg.Swarm.Temporal.Host,
		"--namespace", cfg.Swarm.Temporal.Namespace,
		"--workflow-id", workflowID,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error fetching workflow: %v\n", err)
		fmt.Printf("Make sure the workflow ID is correct.\n")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		fmt.Printf("Error parsing result: %v\n", err)
		return
	}

	info, ok := result["workflowExecutionInfo"].(map[string]interface{})
	if !ok {
		fmt.Printf("Error: invalid response format\n")
		return
	}

	status, _ := info["status"].(string)

	fmt.Printf("%s Workflow Result\n\n", logo)
	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Status: %s\n", status)

	if startTime, ok := info["startTime"].(string); ok {
		fmt.Printf("Started: %s\n", startTime)
	}

	if status == "COMPLETED" {
		if res, ok := result["result"].(map[string]interface{}); ok {
			if rawValue, ok := res["raw"].(string); ok {
				// Try to decode base64 if present
				fmt.Printf("\n--- Result ---\n%s\n--- End ---\n", rawValue)
			} else if value, ok := res["value"].(string); ok {
				fmt.Printf("\n--- Result ---\n%s\n--- End ---\n", value)
			} else if data, ok := res["data"].(string); ok {
				fmt.Printf("\n--- Result ---\n%s\n--- End ---\n", data)
			} else {
				fmt.Printf("\n--- Result ---\n%+v\n--- End ---\n", res)
			}
		}
	} else if status == "FAILED" {
		if res, ok := result["result"].(map[string]interface{}); ok {
			if value, ok := res["value"].(string); ok {
				fmt.Printf("\n--- Error ---\n%s\n--- End ---\n", value)
			}
		}
	} else if status == "RUNNING" {
		fmt.Printf("\n⏳ Workflow is still running...\n")
		fmt.Printf("Use --wait flag to wait for completion:\n")
		fmt.Printf("  picoclaw swarm result %s --wait\n", workflowID)
	}

	fmt.Printf("\nMore info:\n")
	fmt.Printf("  temporal workflow describe --namespace %s %s\n", cfg.Swarm.Temporal.Namespace, workflowID)
	fmt.Printf("  http://localhost:8088/namespaces/%s/workflows/%s\n", cfg.Swarm.Temporal.Namespace, workflowID)
}

// Helper functions for process management

func findSwarmProcesses() ([]int, error) {
	cmd := exec.Command("pgrep", "-f", "picoclaw swarm start")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func stopProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(os.Interrupt)
}

func swarmStatusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("%s Swarm Configuration\n\n", logo)
	fmt.Printf("Enabled: %v\n", cfg.Swarm.Enabled)
	fmt.Printf("Node ID: %s\n", cfg.Swarm.NodeID)
	fmt.Printf("Role: %s\n", cfg.Swarm.Role)
	fmt.Printf("Capabilities: %v\n", cfg.Swarm.Capabilities)
	fmt.Printf("Max Concurrent: %d\n", cfg.Swarm.MaxConcurrent)
	fmt.Println("\nNATS:")
	fmt.Printf("  URLs: %v\n", cfg.Swarm.NATS.URLs)
	fmt.Printf("  Embedded: %v\n", cfg.Swarm.NATS.Embedded)
	fmt.Printf("  Heartbeat: %s\n", cfg.Swarm.NATS.HeartbeatInterval)
	fmt.Printf("  Node Timeout: %s\n", cfg.Swarm.NATS.NodeTimeout)
	fmt.Println("\nTemporal:")
	fmt.Printf("  Host: %s\n", cfg.Swarm.Temporal.Host)
	fmt.Printf("  Namespace: %s\n", cfg.Swarm.Temporal.Namespace)
	fmt.Printf("  Task Queue: %s\n", cfg.Swarm.Temporal.TaskQueue)
}

func swarmNodesCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Get NATS URL from config or use default
	natsURL := "nats://localhost:4222"
	if len(cfg.Swarm.NATS.URLs) > 0 {
		natsURL = cfg.Swarm.NATS.URLs[0]
	}

	// Get HID for filtering
	hid := cfg.Swarm.HID

	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.Timeout(5*time.Second),
		nats.ReconnectWait(100*time.Millisecond),
		nats.MaxReconnects(2),
	)
	if err != nil {
		fmt.Printf("%s Swarm Nodes\n\n", logo)
		fmt.Printf("Failed to connect to NATS at %s\n", natsURL)
		fmt.Printf("Error: %v\n\n", err)
		fmt.Println("Make sure swarm nodes are running:")
		fmt.Println("  pm2 status")
		return
	}
	defer nc.Close()

	// Wait a bit for connection to be fully established
	time.Sleep(100 * time.Millisecond)

	// Node info structures matching the swarm package
	type Heartbeat struct {
		NodeID       string   `json:"node_id"`
		Timestamp    int64    `json:"timestamp"`
		Load         float64  `json:"load"`
		TasksRunning int      `json:"tasks_running"`
		Status       string   `json:"status"`
		Capabilities []string `json:"capabilities"`
	}

	type NodeInfo struct {
		ID           string   `json:"id"`
		NodeID       string   `json:"node_id"`
		Role         string   `json:"role"`
		Capabilities []string `json:"capabilities"`
		Status       string   `json:"status"`
		Load         float64  `json:"load"`
		TasksRunning int      `json:"tasks_running"`
		MaxTasks     int      `json:"max_tasks"`
		Model        string   `json:"model"`
		HID          string   `json:"hid"`
		SID          string   `json:"sid"`
	}

	type DiscoveryAnnounce struct {
		Node      NodeInfo `json:"node"`
		Timestamp int64    `json:"timestamp"`
	}

	nodes := make(map[string]NodeInfo)
	var mu sync.Mutex

	// Subscribe to heartbeat messages (picoclaw.swarm.heartbeat.>)
	sub1, err := nc.Subscribe("picoclaw.swarm.heartbeat.>", func(msg *nats.Msg) {
		var hb Heartbeat
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			return
		}
		mu.Lock()
		// Update existing node or add new one
		if node, ok := nodes[hb.NodeID]; ok {
			node.Load = hb.Load
			node.TasksRunning = hb.TasksRunning
			node.Status = hb.Status
			if len(hb.Capabilities) > 0 {
				node.Capabilities = hb.Capabilities
			}
			nodes[hb.NodeID] = node
		}
		mu.Unlock()
	})
	if err == nil {
		defer sub1.Unsubscribe()
	}

	// Subscribe to discovery announce messages (picoclaw.swarm.discovery.announce)
	sub2, err := nc.Subscribe("picoclaw.swarm.discovery.announce", func(msg *nats.Msg) {
		var announce DiscoveryAnnounce
		if err := json.Unmarshal(msg.Data, &announce); err != nil {
			return
		}
		node := announce.Node

		// Use ID or NodeID field
		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.NodeID
		}

		// Filter by HID if specified
		if hid != "" && node.HID != hid {
			return
		}

		// Ensure NodeID is set for lookup
		if node.NodeID == "" {
			node.NodeID = nodeID
		}

		mu.Lock()
		// Merge with existing node info if any
		if existing, ok := nodes[nodeID]; ok {
			// Keep heartbeat-updated fields
			if existing.Load > 0 {
				node.Load = existing.Load
			}
			if existing.TasksRunning > 0 {
				node.TasksRunning = existing.TasksRunning
			}
		}
		nodes[nodeID] = node
		mu.Unlock()
	})
	if err == nil {
		defer sub2.Unsubscribe()
	}

	// Also subscribe to discovery response (reply to query)
	sub3, err := nc.Subscribe("picoclaw.swarm.discovery.>", func(msg *nats.Msg) {
		var announce DiscoveryAnnounce
		if err := json.Unmarshal(msg.Data, &announce); err != nil {
			// Try direct NodeInfo
			var node NodeInfo
			if err2 := json.Unmarshal(msg.Data, &node); err2 == nil {
				mu.Lock()
				nodeID := node.ID
				if nodeID == "" {
					nodeID = node.NodeID
				}
				if node.NodeID == "" {
					node.NodeID = nodeID
				}
				if hid == "" || node.HID == hid {
					nodes[nodeID] = node
				}
				mu.Unlock()
			}
			return
		}
		node := announce.Node
		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.NodeID
		}
		if node.NodeID == "" {
			node.NodeID = nodeID
		}
		mu.Lock()
		if hid == "" || node.HID == hid {
			nodes[nodeID] = node
		}
		mu.Unlock()
	})
	if err == nil {
		defer sub3.Unsubscribe()
	}

	// Publish a discovery query to prompt nodes to respond
	queryMsg := map[string]interface{}{
		"requester_id": "picoclaw-cli-query",
		"timestamp":    time.Now().UnixMilli(),
	}
	queryData, _ := json.Marshal(queryMsg)

	// Use PublishRequest to allow nodes to respond via msg.Respond()
	inbox := nats.NewInbox()
	responseSub, _ := nc.Subscribe(inbox, func(msg *nats.Msg) {
		var node NodeInfo
		if err := json.Unmarshal(msg.Data, &node); err == nil {
			nodeID := node.ID
			if nodeID == "" {
				nodeID = node.NodeID
			}
			if nodeID != "" {
				mu.Lock()
				if hid == "" || node.HID == hid {
					if existing, ok := nodes[nodeID]; ok {
						// Update with more complete info
						if node.Role != "" && existing.Role == "" {
							existing.Role = node.Role
						}
						if node.Status != "" && existing.Status == "" {
							existing.Status = node.Status
						}
						if len(node.Capabilities) > 0 && len(existing.Capabilities) == 0 {
							existing.Capabilities = node.Capabilities
						}
						if node.MaxTasks > 0 && existing.MaxTasks == 0 {
							existing.MaxTasks = node.MaxTasks
						}
						if node.Model != "" && existing.Model == "" {
							existing.Model = node.Model
						}
						if node.HID != "" {
							existing.HID = node.HID
						}
						if node.SID != "" {
							existing.SID = node.SID
						}
						nodes[nodeID] = existing
					} else {
						// Ensure NodeID is set
						if node.NodeID == "" {
							node.NodeID = nodeID
						}
						nodes[nodeID] = node
					}
				}
				mu.Unlock()
			}
		}
	})
	defer responseSub.Unsubscribe()

	nc.PublishRequest("picoclaw.swarm.discovery.query", inbox, queryData)

	// Also try wildcard subscription to catch heartbeat messages
	debugSub, _ := nc.Subscribe("picoclaw.swarm.heartbeat.>", func(msg *nats.Msg) {
		var raw map[string]interface{}
		if err := json.Unmarshal(msg.Data, &raw); err == nil {
			if nodeID, ok := raw["node_id"].(string); ok {
				mu.Lock()
				if existing, ok := nodes[nodeID]; ok {
					// Update heartbeat fields
					if load, ok := raw["load"].(float64); ok {
						existing.Load = load
					}
					if tasksRunning, ok := raw["tasks_running"].(float64); ok {
						existing.TasksRunning = int(tasksRunning)
					}
					if status, ok := raw["status"].(string); ok {
						existing.Status = status
					}
					if caps, ok := raw["capabilities"].([]interface{}); ok && len(existing.Capabilities) == 0 {
						for _, c := range caps {
							if cs, ok := c.(string); ok {
								existing.Capabilities = append(existing.Capabilities, cs)
							}
						}
					}
					nodes[nodeID] = existing
				} else {
					// Create new node from heartbeat (for nodes that only send heartbeats)
					newNode := NodeInfo{
						ID:     nodeID,
						NodeID: nodeID,
						Status: "online",
						Role:   "worker", // Default to worker if not specified
					}
					if load, ok := raw["load"].(float64); ok {
						newNode.Load = load
					}
					if tasksRunning, ok := raw["tasks_running"].(float64); ok {
						newNode.TasksRunning = int(tasksRunning)
					}
					if status, ok := raw["status"].(string); ok {
						newNode.Status = status
					}
					if caps, ok := raw["capabilities"].([]interface{}); ok {
						for _, c := range caps {
							if cs, ok := c.(string); ok {
								newNode.Capabilities = append(newNode.Capabilities, cs)
							}
						}
					}
					nodes[nodeID] = newNode
				}
				mu.Unlock()
			}
		}
	})
	defer debugSub.Unsubscribe()

	// Wait longer for responses and heartbeats (heartbeat interval is 10s)
	// Wait at least 20 seconds to capture at least 2 heartbeat cycles from all nodes
	time.Sleep(20 * time.Second)

	mu.Lock()
	nodeList := make([]NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		// Only include nodes with valid IDs
		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.NodeID
		}
		if nodeID != "" {
			nodeList = append(nodeList, node)
		}
	}
	mu.Unlock()

	// Display results
	fmt.Printf("%s Swarm Nodes\n\n", logo)

	if len(nodeList) == 0 {
		fmt.Println("No nodes discovered.")
		fmt.Println("\nMake sure swarm nodes are running:")
		fmt.Println("  pm2 status")
		fmt.Println("\nOr start a swarm node:")
		fmt.Println("  picoclaw swarm start --role coordinator --embedded")
		return
	}

	// Count by role
	coordinators := 0
	workers := 0
	specialists := 0

	for _, node := range nodeList {
		switch node.Role {
		case "coordinator":
			coordinators++
		case "worker":
			workers++
		case "specialist":
			specialists++
		}
	}

	fmt.Printf("Total: %d node(s) found\n\n", len(nodeList))
	fmt.Printf("  • Coordinators: %d\n", coordinators)
	fmt.Printf("  • Workers: %d\n", workers)
	fmt.Printf("  • Specialists: %d\n", specialists)
	fmt.Println("\nNodes:")

	for _, node := range nodeList {
		statusIcon := "●"
		if node.Status != "online" {
			statusIcon = "○"
		}
		roleIcon := "C"
		if node.Role == "worker" {
			roleIcon = "W"
		} else if node.Role == "specialist" {
			roleIcon = "S"
		}

		loadPercent := int(node.Load * 100)

		// Use ID or NodeID for display
		displayID := node.ID
		if displayID == "" {
			displayID = node.NodeID
		}
		if len(displayID) > 20 {
			displayID = displayID[:17] + "..."
		}

		fmt.Printf("  %s %s %-20s [%2s] %s (load: %d%%, tasks: %d/%d)\n",
			statusIcon, roleIcon, displayID, node.Role,
			node.Status, loadPercent, node.TasksRunning, node.MaxTasks)

		if len(node.Capabilities) > 0 {
			fmt.Printf("      Capabilities: %s\n", strings.Join(node.Capabilities, ", "))
		}
		if node.SID != "" {
			fmt.Printf("      SID: %s\n", node.SID)
		}
	}

	fmt.Printf("\nNATS: %s\n", natsURL)
	if hid != "" {
		fmt.Printf("HID: %s (filtered)\n", hid)
	}
}

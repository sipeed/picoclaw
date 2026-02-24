// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/sipeed/picoclaw/pkg/devices"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/heartbeat"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
)

// gatewayRunner holds the initialized gateway components.
// This allows the gateway lifecycle to be managed externally (e.g., by daemon service).
type gatewayRunner struct {
	cfg              *config.Config
	provider         providers.LLMProvider
	msgBus           *bus.MessageBus
	agentLoop        *agent.AgentLoop
	cronService      *cron.CronService
	heartbeatService *heartbeat.HeartbeatService
	channelManager   *channels.Manager
	deviceService    *devices.Service
	healthServer     *health.Server
	stateManager     *state.Manager
	ctx              context.Context
	cancel           context.CancelFunc
}

// createGatewayRunner initializes all gateway components and returns a runner.
// This function does NOT start any services - it only initializes them.
// The caller is responsible for calling the returned start function.
func createGatewayRunner(isDaemon bool) (*gatewayRunner, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating provider: %w", err)
	}
	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Setup cron tool and service
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	cronService := setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)

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
		var response string
		response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
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
		return nil, fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager into agent loop for command handling
	agentLoop.SetChannelManager(channelManager)

	var transcriber *voice.GroqTranscriber
	groqAPIKey := cfg.Providers.Groq.APIKey
	if groqAPIKey == "" {
		for _, mc := range cfg.ModelList {
			if strings.HasPrefix(mc.Model, "groq/") && mc.APIKey != "" {
				groqAPIKey = mc.APIKey
				break
			}
		}
	}
	if groqAPIKey != "" {
		transcriber = voice.NewGroqTranscriber(groqAPIKey)
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

	ctx, cancel := context.WithCancel(context.Background())

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)

	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)

	return &gatewayRunner{
		cfg:              cfg,
		provider:         provider,
		msgBus:           msgBus,
		agentLoop:        agentLoop,
		cronService:      cronService,
		heartbeatService: heartbeatService,
		channelManager:   channelManager,
		deviceService:    deviceService,
		healthServer:     healthServer,
		stateManager:     stateManager,
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// run starts all gateway services and waits for context cancellation.
// This method blocks until the gateway is stopped.
func (r *gatewayRunner) run(isDaemon bool) error {
	// Print startup info only in foreground mode
	if !isDaemon {
		// Print agent startup info
		fmt.Println("\n📦 Agent Status:")
		startupInfo := r.agentLoop.GetStartupInfo()
		toolsInfo := startupInfo["tools"].(map[string]any)
		skillsInfo := startupInfo["skills"].(map[string]any)
		fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
		fmt.Printf("  • Skills: %d/%d available\n",
			skillsInfo["available"],
			skillsInfo["total"])

		// Log to file as well
		logger.InfoCF("agent", "Agent initialized",
			map[string]any{
				"tools_count":      toolsInfo["count"],
				"skills_total":     skillsInfo["total"],
				"skills_available": skillsInfo["available"],
			})

		enabledChannels := r.channelManager.GetEnabledChannels()
		if len(enabledChannels) > 0 {
			fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
		} else {
			fmt.Println("⚠ Warning: No channels enabled")
		}

		fmt.Printf("✓ Gateway started on %s:%d\n", r.cfg.Gateway.Host, r.cfg.Gateway.Port)
		fmt.Println("Press Ctrl+C to stop")
	}

	// Start cron service
	if err := r.cronService.Start(); err != nil {
		return fmt.Errorf("error starting cron service: %w", err)
	}
	if !isDaemon {
		fmt.Println("✓ Cron service started")
	}

	// Start heartbeat service
	if err := r.heartbeatService.Start(); err != nil {
		return fmt.Errorf("error starting heartbeat service: %w", err)
	}
	if !isDaemon {
		fmt.Println("✓ Heartbeat service started")
	}

	// Start device service
	if err := r.deviceService.Start(r.ctx); err != nil {
		logger.ErrorCF("device", "Error starting device service", map[string]any{"error": err.Error()})
	} else if r.cfg.Devices.Enabled && !isDaemon {
		fmt.Println("✓ Device event service started")
	}

	// Start channels
	if err := r.channelManager.StartAll(r.ctx); err != nil {
		return fmt.Errorf("error starting channels: %w", err)
	}

	// Start health server
	go func() {
		if err := r.healthServer.Start(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("health", "Health server error", map[string]any{"error": err.Error()})
		}
	}()
	if !isDaemon {
		fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", r.cfg.Gateway.Host, r.cfg.Gateway.Port)
	}

	// Start agent loop
	go r.agentLoop.Run(r.ctx)

	// Wait for context cancellation
	<-r.ctx.Done()

	return nil
}

// stop gracefully stops all gateway services.
func (r *gatewayRunner) stop() {
	logger.InfoC("gateway", "Shutting down...")

	if !isDaemonMode() {
		fmt.Println("\nShutting down...")
	}

	if cp, ok := r.provider.(providers.StatefulProvider); ok {
		cp.Close()
	}

	r.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r.healthServer.Stop(ctx)
	r.deviceService.Stop()
	r.heartbeatService.Stop()
	r.cronService.Stop()
	r.agentLoop.Stop()
	r.channelManager.StopAll(ctx)

	if !isDaemonMode() {
		fmt.Println("✓ Gateway stopped")
	}

	logger.InfoC("gateway", "Shutdown complete")
}

// isDaemonMode returns true if the process is running in daemon mode.
func isDaemonMode() bool {
	return os.Getenv("PICOCLAW_DAEMON") == "1"
}

// gatewayCmd runs the gateway in the foreground.
func gatewayCmd() {
	// Check for --debug flag
	args := os.Args[2:]
	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			logger.SetLevel(logger.DEBUG)
			if !isDaemonMode() {
				fmt.Println("🔍 Debug mode enabled")
			}
			break
		}
	}

	// Create gateway runner
	runner, err := createGatewayRunner(isDaemonMode())
	if err != nil {
		if isDaemonMode() {
			logger.ErrorCF("gateway", "Failed to initialize gateway", map[string]any{"error": err.Error()})
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the gateway
	go func() {
		if err := runner.run(isDaemonMode()); err != nil {
			logger.ErrorCF("gateway", "Gateway error", map[string]any{"error": err.Error()})
			runner.stop()
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	runner.stop()
}

func setupCronTool(
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
	workspace string,
	restrict bool,
	execTimeout time.Duration,
	cfg *config.Config,
) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}

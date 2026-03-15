package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	_ "github.com/sipeed/picoclaw/pkg/channels/dingtalk"
	_ "github.com/sipeed/picoclaw/pkg/channels/discord"
	_ "github.com/sipeed/picoclaw/pkg/channels/feishu"
	_ "github.com/sipeed/picoclaw/pkg/channels/irc"
	_ "github.com/sipeed/picoclaw/pkg/channels/line"
	_ "github.com/sipeed/picoclaw/pkg/channels/maixcam"
	_ "github.com/sipeed/picoclaw/pkg/channels/matrix"
	_ "github.com/sipeed/picoclaw/pkg/channels/onebot"
	_ "github.com/sipeed/picoclaw/pkg/channels/pico"
	_ "github.com/sipeed/picoclaw/pkg/channels/qq"
	_ "github.com/sipeed/picoclaw/pkg/channels/slack"
	_ "github.com/sipeed/picoclaw/pkg/channels/telegram"
	_ "github.com/sipeed/picoclaw/pkg/channels/wecom"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp_native"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/sipeed/picoclaw/pkg/devices"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/heartbeat"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/miniapp"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/research"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tailscale"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
)

// Timeout constants for service operations
const (
	serviceRestartTimeout   = 30 * time.Second
	serviceShutdownTimeout  = 30 * time.Second
	providerReloadTimeout   = 30 * time.Second
	gracefulShutdownTimeout = 15 * time.Second
)

// gatewayServices holds references to all running services
type gatewayServices struct {
	CronService      *cron.CronService
	HeartbeatService *heartbeat.HeartbeatService
	MediaStore       media.MediaStore
	ChannelManager   *channels.Manager
	DeviceService    *devices.Service
	HealthServer     *health.Server
}

func gatewayCmd(debug bool, orchestration bool, enableStats bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	configPath := internal.GetConfigPath()
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}
	if orchestration {
		cfg.Agents.Defaults.Orchestration = true
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider, enableStats)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	if wsProvider, ok := toolsInfo["web_search_provider"].(string); ok {
		fmt.Printf("  • Web search: %s\n", wsProvider)
	} else {
		fmt.Println("  • Web search: disabled")
	}
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

	// Setup and start all services
	services, err := setupAndStartServices(cfg, agentLoop, msgBus)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentLoop.Run(ctx)

	// Setup config file watcher for hot reload
	configReloadChan, stopWatch := setupConfigWatcherPolling(configPath, debug)
	defer stopWatch()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Main event loop - wait for signals or config changes
	for {
		select {
		case <-sigChan:
			logger.Info("Shutting down...")
			shutdownGateway(services, agentLoop, provider, true)
			return nil

		case newCfg := <-configReloadChan:
			err := handleConfigReload(ctx, agentLoop, newCfg, &provider, services, msgBus)
			if err != nil {
				logger.Errorf("Config reload failed: %v", err)
			}
		}
	}
}

// setupAndStartServices initializes and starts all services
func setupAndStartServices(
	cfg *config.Config,
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
) (*gatewayServices, error) {
	services := &gatewayServices{}

	// Setup cron tool and service
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	services.CronService = setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)
	if err := services.CronService.Start(); err != nil {
		return nil, fmt.Errorf("error starting cron service: %w", err)
	}
	fmt.Println("✓ Cron service started")

	// Setup heartbeat service
	services.HeartbeatService = heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	services.HeartbeatService.SetHeartbeatThreadID(cfg.Channels.Telegram.HeartbeatThreadID)
	services.HeartbeatService.SetBus(msgBus)
	agentLoop.SetHeartbeatThreadUpdater(services.HeartbeatService.SetHeartbeatThreadID)
	agentLoop.SetConfigSaver(func(c *config.Config) error {
		return config.SaveConfig(internal.GetConfigPath(), c)
	})
	services.HeartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		var response string
		var err error
		response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		// Always return SilentResult — the task completion message in runAgentLoop
		// already includes the LLM response in the same status bubble.
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		return tools.SilentResult(response)
	})
	if err := services.HeartbeatService.Start(); err != nil {
		return nil, fmt.Errorf("error starting heartbeat service: %w", err)
	}
	fmt.Println("✓ Heartbeat service started")

	// Reset heartbeat suppression when a real user message arrives
	agentLoop.OnUserMessage = services.HeartbeatService.ResetSuppression

	// Create media store for file lifecycle management with TTL cleanup
	services.MediaStore = media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	// Start the media store if it's a FileMediaStore with cleanup
	if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
		fms.Start()
	}

	// Create channel manager
	var err error
	services.ChannelManager, err = channels.NewManager(cfg, msgBus, services.MediaStore)
	if err != nil {
		// Stop the media store if it's a FileMediaStore with cleanup
		if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return nil, fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager and media store into agent loop
	agentLoop.SetChannelManager(services.ChannelManager)
	agentLoop.SetMediaStore(services.MediaStore)

	// Wire up voice transcription if a supported provider is configured.
	if transcriber := voice.DetectTranscriber(cfg); transcriber != nil {
		agentLoop.SetTranscriber(transcriber)
		logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
	}

	enabledChannels := services.ChannelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	// Setup shared HTTP server with health endpoints and webhook handlers
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	services.HealthServer = health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	services.ChannelManager.SetupHTTPServer(addr, services.HealthServer)

	if err := services.ChannelManager.StartAll(context.Background()); err != nil {
		return nil, fmt.Errorf("error starting channels: %w", err)
	}

	// Mini App setup: register routes and determine TLS mode
	useTLS := false
	var tlsCert, tlsKey string
	var miniappNotifier *miniapp.StateNotifier
	if cfg.Channels.Telegram.Enabled {
		webAppURL := cfg.Channels.Telegram.WebAppURL
		if webAppURL == "" {
			// Auto-detect Tailscale hostname and build the WebAppURL
			hostname, tsErr := tailscale.DetectHostname()
			if tsErr != nil {
				logger.InfoCF(
					"miniapp",
					"Tailscale not available, Mini App disabled",
					map[string]any{"error": tsErr.Error()},
				)
			} else {
				hostPort := net.JoinHostPort(hostname, strconv.Itoa(cfg.Gateway.Port))
				webAppURL = "https://" + hostPort + "/miniapp"
				cfg.Channels.Telegram.WebAppURL = webAppURL
			}
		}

		// When the URL is HTTPS, fetch a TLS certificate from Tailscale so
		// the server can actually serve over TLS. This covers both the
		// auto-detected case above and a manually configured https:// URL.
		if strings.HasPrefix(webAppURL, "https://") {
			hostname, tsErr := tailscale.DetectHostname()
			if tsErr != nil {
				logger.ErrorCF("miniapp", "HTTPS URL configured but Tailscale not available",
					map[string]any{"error": tsErr.Error()})
			} else {
				certDir := filepath.Join(cfg.WorkspacePath(), "state", "certs")
				certFile, keyFile, certErr := tailscale.FetchCert(hostname, certDir)
				if certErr != nil {
					logger.ErrorCF("miniapp", "Failed to fetch TLS cert", map[string]any{"error": certErr.Error()})
				} else {
					tlsCert, tlsKey = certFile, keyFile
					useTLS = true
				}
			}
		}

		if webAppURL != "" {
			dataProvider := &agentLoopDataProvider{loop: agentLoop, workspace: cfg.WorkspacePath()}
			sender := &telegramCommandSender{bus: msgBus}
			miniappNotifier = miniapp.NewStateNotifier()
			handler := miniapp.NewHandler(
				dataProvider,
				sender,
				cfg.Channels.Telegram.Token,
				miniappNotifier,
				cfg.Channels.Telegram.AllowFrom,
				cfg.WorkspacePath(),
			)
			agentLoop.OnStateChange = miniappNotifier.Notify
			if b := agentLoop.GetOrchBroadcaster(); b != nil {
				handler.SetOrchBroadcaster(b)
			}
			handler.RegisterRoutes(services.HealthServer.Mux())

			// Register dev preview tool for all agents
			devPreviewTool := tools.NewDevPreviewTool(handler)
			agentLoop.RegisterTool(devPreviewTool)

			// Research store + tool registration
			researchStore, rsErr := research.OpenResearchStore(
				filepath.Join(cfg.WorkspacePath(), "research.db"),
				cfg.WorkspacePath(),
			)
			if rsErr != nil {
				logger.ErrorCF("research", "Failed to open research store", map[string]any{"error": rsErr.Error()})
			} else {
				agentLoop.RegisterTool(tools.NewResearchTool(researchStore, cfg.WorkspacePath()))
				handler.SetResearchStore(researchStore)
				fmt.Println("✓ Research store initialized")
			}

			fmt.Printf("✓ Mini App registered at %s\n", webAppURL)
		}
	}

	go func() {
		var serverErr error
		if useTLS {
			serverErr = services.HealthServer.StartTLS(tlsCert, tlsKey)
		} else {
			serverErr = services.HealthServer.Start()
		}
		if serverErr != nil && serverErr != http.ErrServerClosed {
			logger.ErrorCF("health", "Health server error", map[string]any{"error": serverErr.Error()})
		}
	}()
	if useTLS {
		fmt.Printf(
			"✓ Health endpoints available at https://%s:%d/health and /ready (TLS)\n",
			cfg.Gateway.Host,
			cfg.Gateway.Port,
		)
	} else {
		fmt.Printf(
			"✓ Health endpoints available at http://%s:%d/health and /ready\n",
			cfg.Gateway.Host, cfg.Gateway.Port,
		)
	}

	// Setup state manager and device service
	stateManager := state.NewManager(cfg.WorkspacePath())
	services.DeviceService = devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	services.DeviceService.SetBus(msgBus)
	if err := services.DeviceService.Start(context.Background()); err != nil {
		logger.ErrorCF("device", "Error starting device service", map[string]any{"error": err.Error()})
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}
	return services, nil
}

// stopAndCleanupServices stops all services and cleans up resources
func stopAndCleanupServices(
	services *gatewayServices,
	shutdownTimeout time.Duration,
) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if services.ChannelManager != nil {
		services.ChannelManager.StopAll(shutdownCtx)
	}
	if services.HealthServer != nil {
		services.HealthServer.Stop(shutdownCtx)
	}
	if services.DeviceService != nil {
		services.DeviceService.Stop()
	}
	if services.HeartbeatService != nil {
		services.HeartbeatService.Stop()
	}
	if services.CronService != nil {
		services.CronService.Stop()
	}
	if services.MediaStore != nil {
		// Stop the media store if it's a FileMediaStore with cleanup
		if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
	}
}

// shutdownGateway performs a complete gateway shutdown
func shutdownGateway(
	services *gatewayServices,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
	fullShutdown bool,
) {
	if cp, ok := provider.(providers.StatefulProvider); ok && fullShutdown {
		cp.Close()
	}

	stopAndCleanupServices(services, gracefulShutdownTimeout)

	agentLoop.Stop()
	agentLoop.Close()

	logger.Info("✓ Gateway stopped")
}

// handleConfigReload handles config file reload by stopping all services,
// reloading the provider and config, and restarting services with the new config.
func handleConfigReload(
	ctx context.Context,
	al *agent.AgentLoop,
	newCfg *config.Config,
	providerRef *providers.LLMProvider,
	services *gatewayServices,
	msgBus *bus.MessageBus,
) error {
	logger.Info("🔄 Config file changed, reloading...")

	newModel := newCfg.Agents.Defaults.ModelName
	if newModel == "" {
		newModel = newCfg.Agents.Defaults.Model
	}

	logger.Infof(" New model is '%s', recreating provider...", newModel)

	// Stop all services before reloading
	logger.Info("  Stopping all services...")
	stopAndCleanupServices(services, serviceShutdownTimeout)

	// Create new provider from updated config first to ensure validity
	// This will use the correct API key and settings from newCfg.ModelList
	newProvider, newModelID, err := providers.CreateProvider(newCfg)
	if err != nil {
		logger.Errorf("  ⚠ Error creating new provider: %v", err)
		logger.Warn("  Attempting to restart services with old provider and config...")
		// Try to restart services with old configuration
		if restartErr := restartServices(al, services, msgBus); restartErr != nil {
			logger.Errorf("  ⚠ Failed to restart services: %v", restartErr)
		}
		return fmt.Errorf("error creating new provider: %w", err)
	}

	if newModelID != "" {
		newCfg.Agents.Defaults.ModelName = newModelID
	}

	// Use the atomic reload method on AgentLoop to safely swap provider and config.
	// This handles locking internally to prevent races with in-flight LLM calls
	// and concurrent reads of registry/config while the swap occurs.
	reloadCtx, reloadCancel := context.WithTimeout(context.Background(), providerReloadTimeout)
	defer reloadCancel()

	if err := al.ReloadProviderAndConfig(reloadCtx, newProvider, newCfg); err != nil {
		logger.Errorf("  ⚠ Error reloading agent loop: %v", err)
		// Close the newly created provider since it wasn't adopted
		if cp, ok := newProvider.(providers.StatefulProvider); ok {
			cp.Close()
		}
		logger.Warn("  Attempting to restart services with old provider and config...")
		if restartErr := restartServices(al, services, msgBus); restartErr != nil {
			logger.Errorf("  ⚠ Failed to restart services: %v", restartErr)
		}
		return fmt.Errorf("error reloading agent loop: %w", err)
	}

	// Update local provider reference only after successful atomic reload
	*providerRef = newProvider

	// Restart all services with new config
	logger.Info("  Restarting all services with new configuration...")
	if err := restartServices(al, services, msgBus); err != nil {
		logger.Errorf("  ⚠ Error restarting services: %v", err)
		return fmt.Errorf("error restarting services: %w", err)
	}

	logger.Info("  ✓ Provider, configuration, and services reloaded successfully (thread-safe)")
	return nil
}

// restartServices restarts all services after a config reload
func restartServices(
	al *agent.AgentLoop,
	services *gatewayServices,
	msgBus *bus.MessageBus,
) error {
	// Create an independent context with timeout for service restart
	// This prevents cancellation from the main loop context during reload
	ctx, cancel := context.WithTimeout(context.Background(), serviceRestartTimeout)
	defer cancel()

	// Get current config from agent loop (which has been updated if this is a reload)
	cfg := al.GetConfig()

	// Re-create and start cron service with new config
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	services.CronService = setupCronTool(
		al,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)
	if err := services.CronService.Start(); err != nil {
		return fmt.Errorf("error restarting cron service: %w", err)
	}
	fmt.Println("  ✓ Cron service restarted")

	// Re-create and start heartbeat service with new config
	services.HeartbeatService = heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	services.HeartbeatService.SetBus(msgBus)
	services.HeartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		var response string
		var err error
		response, err = al.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		return tools.SilentResult(response)
	})
	if err := services.HeartbeatService.Start(); err != nil {
		return fmt.Errorf("error restarting heartbeat service: %w", err)
	}
	fmt.Println("  ✓ Heartbeat service restarted")

	// Stop the old media store before creating a new one
	if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
		fms.Stop()
	}

	// Re-create media store with new config
	services.MediaStore = media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	// Start the media store if it's a FileMediaStore with cleanup
	if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
		fms.Start()
	}
	al.SetMediaStore(services.MediaStore)

	// Re-create channel manager with new config
	var err error
	services.ChannelManager, err = channels.NewManager(cfg, msgBus, services.MediaStore)
	if err != nil {
		// Stop the media store if it's a FileMediaStore with cleanup
		if fms, ok := services.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return fmt.Errorf("error recreating channel manager: %w", err)
	}
	al.SetChannelManager(services.ChannelManager)

	enabledChannels := services.ChannelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("  ✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("  ⚠ Warning: No channels enabled")
	}

	// Setup HTTP server with new config
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	services.HealthServer = health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	services.ChannelManager.SetupHTTPServer(addr, services.HealthServer)

	if err := services.ChannelManager.StartAll(ctx); err != nil {
		return fmt.Errorf("error restarting channels: %w", err)
	}
	fmt.Printf(
		"  ✓ Channels restarted, health endpoints at http://%s:%d/health and ready\n",
		cfg.Gateway.Host,
		cfg.Gateway.Port,
	)

	// Re-create device service with new config
	stateManager := state.NewManager(cfg.WorkspacePath())
	services.DeviceService = devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	services.DeviceService.SetBus(msgBus)
	if err := services.DeviceService.Start(ctx); err != nil {
		logger.WarnCF("device", "Failed to restart device service", map[string]any{"error": err.Error()})
	} else if cfg.Devices.Enabled {
		fmt.Println("  ✓ Device event service restarted")
	}

	// Wire up voice transcription with new config
	transcriber := voice.DetectTranscriber(cfg)
	al.SetTranscriber(transcriber) // This will set it to nil if disabled
	if transcriber != nil {
		logger.InfoCF("voice", "Transcription re-enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
	} else {
		logger.InfoCF("voice", "Transcription disabled", nil)
	}

	return nil
}

// setupConfigWatcherPolling sets up a simple polling-based config file watcher
// Returns a channel for config updates and a stop function
func setupConfigWatcherPolling(configPath string, debug bool) (chan *config.Config, func()) {
	configChan := make(chan *config.Config, 1)
	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get initial file info
		lastModTime := getFileModTime(configPath)
		lastSize := getFileSize(configPath)

		ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentModTime := getFileModTime(configPath)
				currentSize := getFileSize(configPath)

				// Check if file changed (modification time or size changed)
				if currentModTime.After(lastModTime) || currentSize != lastSize {
					if debug {
						logger.Debugf("🔍 Config file change detected")
					}

					// Debounce - wait a bit to ensure file write is complete
					time.Sleep(500 * time.Millisecond)

					// Validate and load new config
					newCfg, err := config.LoadConfig(configPath)
					if err != nil {
						logger.Errorf("⚠ Error loading new config: %v", err)
						logger.Warn("  Using previous valid config")
						continue
					}

					// Validate the new config
					if err := newCfg.ValidateModelList(); err != nil {
						logger.Errorf("  ⚠ New config validation failed: %v", err)
						logger.Warn("  Using previous valid config")
						continue
					}

					logger.Info("✓ Config file validated and loaded")

					// Update last known state
					lastModTime = currentModTime
					lastSize = currentSize

					// Send new config to main loop (non-blocking)
					select {
					case configChan <- newCfg:
					default:
						// Channel full, skip this update
						logger.Warn("⚠ Previous config reload still in progress, skipping")
					}
				}

			case <-stop:
				return
			}
		}
	}()

	stopFunc := func() {
		close(stop)
		wg.Wait()
	}

	return configChan, stopFunc
}

// getFileModTime returns the modification time of a file, or zero time if file doesn't exist
func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// getFileSize returns the size of a file, or 0 if file doesn't exist
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
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

	// Create and register CronTool if enabled
	var cronTool *tools.CronTool
	if cfg.Tools.IsToolEnabled("cron") {
		var err error
		cronTool, err = tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
		if err != nil {
			logger.Fatalf("Critical error during CronTool initialization: %v", err)
		}

		agentLoop.RegisterTool(cronTool)
	}

	// Set onJob handler
	if cronTool != nil {
		cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
			result := cronTool.ExecuteJob(context.Background(), job)
			return result, nil
		})
	}

	return cronService
}

// agentLoopDataProvider adapts AgentLoop to the miniapp.DataProvider interface.
type agentLoopDataProvider struct {
	loop      *agent.AgentLoop
	workspace string

	gitReposCache   []miniapp.GitRepoSummary
	gitReposCacheAt time.Time
	gitDetailCache  map[string]gitDetailEntry
}

type gitDetailEntry struct {
	info miniapp.GitInfo
	at   time.Time
}

const gitCacheTTL = 5 * time.Minute

func (p *agentLoopDataProvider) ListSkills() []skills.SkillInfo {
	return p.loop.ListSkills()
}

func (p *agentLoopDataProvider) GetPlanInfo() miniapp.PlanInfo {
	hasPlan, status, currentPhase, totalPhases, display, memory := p.loop.GetPlanInfo()

	// Convert agent.PlanPhase -> miniapp.PlanPhase
	agentPhases := p.loop.GetPlanPhases()
	phases := make([]miniapp.PlanPhase, 0, len(agentPhases))
	for _, ap := range agentPhases {
		steps := make([]miniapp.PlanStep, 0, len(ap.Steps))
		for _, as := range ap.Steps {
			steps = append(steps, miniapp.PlanStep{
				Index:       as.Index,
				Description: as.Description,
				Done:        as.Done,
			})
		}
		phases = append(phases, miniapp.PlanPhase{
			Number: ap.Number,
			Title:  ap.Title,
			Steps:  steps,
		})
	}

	return miniapp.PlanInfo{
		HasPlan:      hasPlan,
		Status:       status,
		CurrentPhase: currentPhase,
		TotalPhases:  totalPhases,
		Display:      display,
		Phases:       phases,
		Memory:       memory,
	}
}

func (p *agentLoopDataProvider) GetSessionStats() *stats.Stats {
	return p.loop.GetSessionStats()
}

func (p *agentLoopDataProvider) GetActiveSessions() []miniapp.SessionInfo {
	entries := p.loop.GetActiveSessions()
	result := make([]miniapp.SessionInfo, len(entries))
	for i, e := range entries {
		result[i] = miniapp.SessionInfo{
			SessionKey:  e.SessionKey,
			Channel:     e.Channel,
			ChatID:      e.ChatID,
			TouchDir:    e.TouchDir,
			ProjectPath: e.ProjectPath,
			Purpose:     e.Purpose,
			Branch:      e.Branch,
			LastSeenAt:  e.LastSeenAt.Format(time.RFC3339),
			AgeSec:      int(time.Since(e.LastSeenAt).Seconds()),
		}
	}
	return result
}

func (p *agentLoopDataProvider) GetSessionGraph() *miniapp.SessionGraphData {
	nodes := p.loop.GetSessionGraph()
	if len(nodes) == 0 {
		return &miniapp.SessionGraphData{
			Nodes: []miniapp.SessionGraphNode{},
			Edges: []miniapp.SessionGraphEdge{},
		}
	}

	gNodes := make([]miniapp.SessionGraphNode, 0, len(nodes))
	var edges []miniapp.SessionGraphEdge

	for _, n := range nodes {
		sk := gatewayShortKey(n.Key)
		label := n.Label
		if label == "" {
			label = sk
		}
		gNodes = append(gNodes, miniapp.SessionGraphNode{
			Key:        n.Key,
			ShortKey:   sk,
			Label:      label,
			Status:     n.Status,
			TurnCount:  n.TurnCount,
			CreatedAt:  n.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  n.UpdatedAt.Format(time.RFC3339),
			Summary:    n.Summary,
			ForkTurnID: n.ForkTurnID,
		})
		if n.ParentKey != "" {
			edges = append(edges, miniapp.SessionGraphEdge{
				From:       n.ParentKey,
				To:         n.Key,
				ForkTurnID: n.ForkTurnID,
			})
		}
	}
	if edges == nil {
		edges = []miniapp.SessionGraphEdge{}
	}
	return &miniapp.SessionGraphData{Nodes: gNodes, Edges: edges}
}

// gatewayShortKey abbreviates long session keys for display.
func gatewayShortKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) > 2 {
		return strings.Join(parts[2:], ":")
	}
	return key
}

func (p *agentLoopDataProvider) GetContextInfo() miniapp.ContextInfo {
	workDir, planWorkDir, workspace, bootstrap := p.loop.GetContextInfo()
	files := make([]miniapp.BootstrapFileInfo, len(bootstrap))
	for i, b := range bootstrap {
		files[i] = miniapp.BootstrapFileInfo{Name: b.Name, Path: b.Path, Scope: b.Scope}
	}
	return miniapp.ContextInfo{
		WorkDir:     workDir,
		PlanWorkDir: planWorkDir,
		Workspace:   workspace,
		Bootstrap:   files,
	}
}

func (p *agentLoopDataProvider) GetSystemPrompt() string {
	return p.loop.GetSystemPrompt()
}

func (p *agentLoopDataProvider) GetGitRepos() []miniapp.GitRepoSummary {
	if time.Since(p.gitReposCacheAt) < gitCacheTTL {
		return p.gitReposCache
	}

	if p.workspace == "" {
		return nil
	}

	// Find workspace's own git root to exclude it
	workspaceGitRoot := ""
	if out, err := exec.Command("git", "-C", p.workspace, "rev-parse", "--show-toplevel").Output(); err == nil {
		workspaceGitRoot = strings.TrimSpace(string(out))
	}

	// Scan for .git dirs up to 2 levels deep under workspace
	seen := map[string]bool{}
	var repos []miniapp.GitRepoSummary
	for _, pattern := range []string{
		filepath.Join(p.workspace, "*", ".git"),
		filepath.Join(p.workspace, "*", "*", ".git"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			repoDir := filepath.Dir(m)
			if repoDir == workspaceGitRoot || seen[repoDir] {
				continue
			}
			seen[repoDir] = true
			name := filepath.Base(repoDir)
			branch := ""
			out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
			if err == nil {
				branch = strings.TrimSpace(string(out))
			}
			repos = append(repos, miniapp.GitRepoSummary{Name: name, Branch: branch})
		}
	}

	p.gitReposCache = repos
	p.gitReposCacheAt = time.Now()
	return repos
}

func (p *agentLoopDataProvider) GetGitRepoDetail(name string) miniapp.GitInfo {
	// Path traversal prevention
	if name == "" || filepath.Base(name) != name {
		return miniapp.GitInfo{Name: name}
	}

	// Check detail cache
	if p.gitDetailCache != nil {
		if entry, ok := p.gitDetailCache[name]; ok && time.Since(entry.at) < gitCacheTTL {
			return entry.info
		}
	}

	if p.workspace == "" {
		return miniapp.GitInfo{Name: name}
	}

	// Resolve repo path: try 1-level and 2-level deep
	var repoDir string
	for _, pattern := range []string{
		filepath.Join(p.workspace, name, ".git"),
		filepath.Join(p.workspace, "*", name, ".git"),
	} {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			repoDir = filepath.Dir(matches[0])
			break
		}
	}
	if repoDir == "" {
		return miniapp.GitInfo{Name: name}
	}

	info := collectGitRepoInfo(repoDir)

	if p.gitDetailCache == nil {
		p.gitDetailCache = make(map[string]gitDetailEntry)
	}
	p.gitDetailCache[name] = gitDetailEntry{info: info, at: time.Now()}
	return info
}

func collectGitRepoInfo(gitRoot string) miniapp.GitInfo {
	info := miniapp.GitInfo{Name: filepath.Base(gitRoot)}

	// Current branch
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		info.Branch = strings.TrimSpace(string(out))
	}

	// Recent commits (20 entries)
	out, err = exec.Command("git", "-C", gitRoot, "log", "--pretty=format:%h\x1f%s\x1f%an\x1f%cr", "-20").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			parts := strings.SplitN(line, "\x1f", 4)
			if len(parts) == 4 {
				info.Commits = append(info.Commits, miniapp.GitCommit{
					Hash: parts[0], Subject: parts[1], Author: parts[2], Date: parts[3],
				})
			}
		}
	}

	// Modified/untracked files
	out, err = exec.Command("git", "-C", gitRoot, "status", "--porcelain").Output()
	if err == nil && len(out) > 0 {
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			if len(line) < 4 {
				continue
			}
			info.Modified = append(info.Modified, miniapp.GitChange{
				Status: strings.TrimSpace(line[:2]),
				Path:   line[3:],
			})
		}
	}

	return info
}

// telegramCommandSender injects Mini App commands into the message bus.
type telegramCommandSender struct {
	bus *bus.MessageBus
}

func (s *telegramCommandSender) SendCommand(senderID, chatID, command string) {
	s.bus.PublishInbound(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: senderID,
		ChatID:   chatID,
		Content:  command,
		Metadata: map[string]string{
			"source": "webapp",
		},
	})
}

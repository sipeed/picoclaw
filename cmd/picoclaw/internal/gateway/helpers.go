package gateway

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	_ "github.com/sipeed/picoclaw/pkg/channels/dingtalk"
	_ "github.com/sipeed/picoclaw/pkg/channels/discord"
	_ "github.com/sipeed/picoclaw/pkg/channels/feishu"
	_ "github.com/sipeed/picoclaw/pkg/channels/line"
	_ "github.com/sipeed/picoclaw/pkg/channels/maixcam"
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
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
)

func gatewayCmd(debug bool) error {
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

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
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

	// Create media store for file lifecycle management with TTL cleanup
	mediaStore := media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	mediaStore.Start()

	channelManager, err := channels.NewManager(cfg, msgBus, mediaStore)
	if err != nil {
		mediaStore.Stop()
		return fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager and media store into agent loop
	agentLoop.SetChannelManager(channelManager)
	agentLoop.SetMediaStore(mediaStore)

	// Wire up voice transcription if a supported provider is configured.
	if transcriber := voice.DetectTranscriber(cfg); transcriber != nil {
		agentLoop.SetTranscriber(transcriber)
		logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
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

	// Setup shared HTTP server with health endpoints and webhook handlers
	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	channelManager.SetupHTTPServer(addr, healthServer)

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
		return err
	}

	fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)

	go agentLoop.Run(ctx)

	// Setup config file watcher for hot reload
	configWatcher, configReloadChan, watchErr := setupConfigWatcher(configPath, debug)
	if watchErr != nil {
		logger.Errorf("⚠ Warning: Could not start config file watcher: %v", watchErr)
		logger.Warn("  Config changes will require manual restart")
	} else {
		logger.Info("✓ Config file watcher started (auto-reload on change)")
		defer configWatcher.Close()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Main event loop - wait for signals or config changes
	for {
		select {
		case <-sigChan:
			logger.Info("Shutting down...")
			if cp, ok := provider.(providers.StatefulProvider); ok {
				cp.Close()
			}
			cancel()
			msgBus.Close()

			// Use a fresh context with timeout for graceful shutdown,
			// since the original ctx is already canceled.
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer shutdownCancel()

			channelManager.StopAll(shutdownCtx)
			deviceService.Stop()
			heartbeatService.Stop()
			cronService.Stop()
			mediaStore.Stop()
			agentLoop.Stop()
			logger.Info("✓ Gateway stopped")

			return nil

		case newCfg := <-configReloadChan:
			logger.Info("🔄 Config file changed, reloading...")

			newModel := newCfg.Agents.Defaults.ModelName
			if newModel == "" {
				newModel = newCfg.Agents.Defaults.Model
			}

			logger.Infof(" New model is '%s', recreating provider...", newModel)
			if cp, ok := provider.(providers.StatefulProvider); ok {
				cp.Close()
			}

			// Create new provider from updated config
			// This will use the correct API key and settings from newCfg.ModelList
			newProvider, newModelID, err := providers.CreateProvider(newCfg)
			if err != nil {
				logger.Errorf("  ⚠ Error creating new provider: %v", err)
				logger.Warn("  Continuing with old provider")
				continue
			}

			provider = newProvider
			if newModelID != "" {
				newCfg.Agents.Defaults.ModelName = newModelID
			}

			// Update agent loop provider and models
			agentLoop.SetProvider(provider, newCfg)

			logger.Info("  ✓ Provider and agents updated successfully")

			// Update the config reference for other operations
			// Note: Some changes (like channel configs) may require restart to take full effect
			cfg = newCfg
			logger.Info("  ✓ Configuration reloaded successfully")
		}
	}
}

// setupConfigWatcher sets up a file watcher for the config file
// Returns the watcher, a channel for config updates, and any error
func setupConfigWatcher(configPath string, debug bool) (*fsnotify.Watcher, chan *config.Config, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	configDir := filepath.Dir(configPath)
	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return nil, nil, err
	}

	configChan := make(chan *config.Config, 1)
	var mu sync.Mutex
	var debounceTimer *time.Timer

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only process config.json changes
				if event.Name != configPath {
					continue
				}

				// Debounce rapid file changes (some editors write multiple times)
				mu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
					mu.Unlock()

					if debug {
						logger.DebugSF("  🔍 Config file event: %v", event)
					}

					// Validate and load new config
					newCfg, err := config.LoadConfig(configPath)
					if err != nil {
						logger.Errorf("  ⚠ Error loading new config: %v", err)
						logger.Warn("  Using previous valid config")
						return
					}

					// Validate the new config
					if err := newCfg.ValidateModelList(); err != nil {
						logger.Errorf("  ⚠ New config validation failed: %v", err)
						logger.Warn("  Using previous valid config")
						return
					}

					logger.Info("  ✓ Config file validated and loaded")

					// Send new config to main loop (non-blocking)
					select {
					case configChan <- newCfg:
					default:
						// Channel full, skip this update
						logger.Warn("  ⚠ Previous config reload still in progress, skipping")
					}
				})
				mu.Lock() // Keep lock until timer is set
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Errorf("  ⚠ Config watcher error: %v", err)
			}
		}
	}()

	return watcher, configChan, nil
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
	cronTool, err := tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
	if err != nil {
		logger.Fatalf("Critical error during CronTool initialization: %v", err)
	}

	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}

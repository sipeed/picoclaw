package gateway

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/audio/asr"
	"github.com/sipeed/picoclaw/pkg/audio/tts"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	_ "github.com/sipeed/picoclaw/pkg/channels/dingtalk"
	_ "github.com/sipeed/picoclaw/pkg/channels/discord"
	_ "github.com/sipeed/picoclaw/pkg/channels/feishu"
	_ "github.com/sipeed/picoclaw/pkg/channels/irc"
	_ "github.com/sipeed/picoclaw/pkg/channels/line"
	_ "github.com/sipeed/picoclaw/pkg/channels/maixcam"
	_ "github.com/sipeed/picoclaw/pkg/channels/onebot"
	_ "github.com/sipeed/picoclaw/pkg/channels/pico"
	_ "github.com/sipeed/picoclaw/pkg/channels/qq"
	_ "github.com/sipeed/picoclaw/pkg/channels/slack"
	_ "github.com/sipeed/picoclaw/pkg/channels/teams_webhook"
	_ "github.com/sipeed/picoclaw/pkg/channels/telegram"
	_ "github.com/sipeed/picoclaw/pkg/channels/vk"
	_ "github.com/sipeed/picoclaw/pkg/channels/wecom"
	_ "github.com/sipeed/picoclaw/pkg/channels/weixin"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp_native"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/sipeed/picoclaw/pkg/devices"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/heartbeat"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	serviceShutdownTimeout  = 30 * time.Second
	providerReloadTimeout   = 30 * time.Second
	gracefulShutdownTimeout = 15 * time.Second

	logPath   = "logs"
	panicFile = "gateway_panic.log"
	logFile   = "gateway.log"
)

type services struct {
	CronService      *cron.CronService
	HeartbeatService *heartbeat.HeartbeatService
	MediaStore       media.MediaStore
	ChannelManager   *channels.Manager
	DeviceService    *devices.Service
	HealthServer     *health.Server
	VoiceAgentCancel context.CancelFunc
	manualReloadChan chan struct{}
	reloading        atomic.Bool
	authToken        string
}

type startupBlockedProvider struct {
	reason string
}

func logChannelVoiceCapabilities(cm *channels.Manager, asrAvailable bool, ttsAvailable bool) {
	if cm == nil {
		return
	}

	names := cm.GetEnabledChannels()
	sort.Strings(names)
	for _, name := range names {
		ch, ok := cm.GetChannel(name)
		if !ok {
			continue
		}
		caps := channels.DetectVoiceCapabilities(name, ch, asrAvailable, ttsAvailable)
		logger.InfoCF("voice", "Channel voice capabilities", map[string]any{
			"channel": name,
			"asr":     caps.ASR,
			"tts":     caps.TTS,
		})
	}
}

func (p *startupBlockedProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	return nil, fmt.Errorf("%s", p.reason)
}

func (p *startupBlockedProvider) GetDefaultModel() string {
	return ""
}

// Run starts the gateway runtime using the configuration loaded from configPath.
func Run(debug bool, homePath, configPath string, allowEmptyStartup bool) (runErr error) {
	startedAt := time.Now()
	panicPath := filepath.Join(homePath, logPath, panicFile)
	panicFunc, err := logger.InitPanic(panicPath)
	if err != nil {
		return fmt.Errorf("error initializing panic log: %w", err)
	}
	defer panicFunc()

	if err = logger.EnableFileLogging(filepath.Join(homePath, logPath, logFile)); err != nil {
		logger.Fatal(fmt.Sprintf("error enabling file logging: %v", err))
	}
	defer logger.DisableFileLogging()

	if debug {
		logger.SetLevel(logger.DEBUG)
	} else {
		logger.SetLevelFromString(config.ResolveGatewayLogLevel(configPath))
	}
	defer func() {
		if runErr != nil {
			logger.ErrorCF("gateway", "Gateway startup failed", map[string]any{
				"config_path": configPath,
				"error":       runErr.Error(),
				"home_path":   homePath,
				"allow_empty": allowEmptyStartup,
				"debug":       debug,
			})
		}
	}()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if err = preCheckConfig(cfg); err != nil {
		return fmt.Errorf("config pre-check failed: %w", err)
	}

	// Debug mode permanently overrides the config log level to DEBUG.
	if debug {
		fmt.Println("🔍 Debug mode enabled")
	} else {
		effectiveLogLevel := config.EffectiveGatewayLogLevel(cfg)
		logger.SetLevelFromString(effectiveLogLevel)
		logger.Infof("Log level set to %q", effectiveLogLevel)
	}

	bindPlan, listenResult, err := openGatewayListeners(cfg.Gateway.Host, cfg.Gateway.Port)
	if err != nil {
		return fmt.Errorf("error opening gateway listeners: %w", err)
	}

	// Enforce singleton: write PID file with generated token.
	pidData, err := pid.WritePidFile(homePath, bindPlan.ProbeHost, cfg.Gateway.Port)
	if err != nil {
		logger.Warnf("write pid file failed: %v", err)
		for _, ln := range listenResult.Listeners {
			_ = ln.Close()
		}
		return fmt.Errorf("singleton check failed: %w", err)
	}
	defer pid.RemovePidFile(homePath)
	closeListeners := true
	defer func() {
		if !closeListeners {
			return
		}
		for _, ln := range listenResult.Listeners {
			_ = ln.Close()
		}
	}()

	provider, modelID, err := createStartupProvider(cfg, allowEmptyStartup)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
	msgBus.SetEventPublisher(agentLoop.RuntimeEventBus())
	publishGatewayEvent(agentLoop, runtimeevents.KindGatewayStart, startedAt, nil)

	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n", skillsInfo["available"], skillsInfo["total"])

	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	runningServices, err := setupAndStartServices(cfg, agentLoop, msgBus, pidData.Token, listenResult)
	if err != nil {
		return err
	}
	publishGatewayEvent(agentLoop, runtimeevents.KindGatewayReady, startedAt, nil)
	closeListeners = false

	// Setup manual reload channel for /reload endpoint
	manualReloadChan := make(chan struct{}, 1)
	runningServices.manualReloadChan = manualReloadChan
	reloadTrigger := func() error {
		if !runningServices.reloading.CompareAndSwap(false, true) {
			return fmt.Errorf("reload already in progress")
		}
		select {
		case manualReloadChan <- struct{}{}:
			return nil
		default:
			// Should not happen, but reset flag if channel is full
			runningServices.reloading.Store(false)
			return fmt.Errorf("reload already queued")
		}
	}
	runningServices.HealthServer.SetReloadFunc(reloadTrigger)
	agentLoop.SetReloadFunc(reloadTrigger)

	// Wire permission grant function to allow runtime permission grants via health endpoint
	runningServices.HealthServer.SetPermissionGrantFunc(func(agentID, path, duration string) error {
		return agentLoop.GrantPermission(agentID, path, duration)
	})
	runningServices.HealthServer.SetSubagentStatusFunc(func(channel, chatID string) (any, error) {
		return map[string]any{
			"channel": channel,
			"chat_id": chatID,
			"tasks":   agentLoop.GetMainSubagentTasks(channel, chatID),
		}, nil
	})

	for _, bindHost := range listenResult.BindHosts {
		fmt.Printf("✓ Gateway started on %s\n", net.JoinHostPort(bindHost, strconv.Itoa(cfg.Gateway.Port)))
	}
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentLoop.Run(ctx)

	var configReloadChan <-chan *config.Config
	stopWatch := func() {}
	if cfg.Gateway.HotReload {
		configReloadChan, stopWatch = setupConfigWatcherPolling(configPath, debug)
		logger.Info("Config hot reload enabled")
	}
	defer stopWatch()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			logger.Info("Shutting down...")
			shutdownGateway(runningServices, agentLoop, provider, msgBus, true)
			return nil
		case newCfg := <-configReloadChan:
			if !runningServices.reloading.CompareAndSwap(false, true) {
				logger.Warn("Config reload skipped: another reload is in progress")
				continue
			}
			err := executeReload(ctx, agentLoop, newCfg, &provider, runningServices, msgBus, allowEmptyStartup, debug)
			if err != nil {
				logger.Errorf("Config reload failed: %v", err)
			}
		case <-manualReloadChan:
			logger.Info("Manual reload triggered via /reload endpoint")
			newCfg, err := config.LoadConfig(configPath)
			if err != nil {
				logger.Errorf("Error loading config for manual reload: %v", err)
				runningServices.reloading.Store(false)
				continue
			}
			if err = newCfg.ValidateModelList(); err != nil {
				logger.Errorf("Config validation failed: %v", err)
				runningServices.reloading.Store(false)
				continue
			}
			err = executeReload(ctx, agentLoop, newCfg, &provider, runningServices, msgBus, allowEmptyStartup, debug)
			if err != nil {
				logger.Errorf("Manual reload failed: %v", err)
			} else {
				logger.Info("Manual reload completed successfully")
			}
		}
	}
}

func preCheckConfig(cfg *config.Config) error {
	if cfg.Gateway.Port <= 0 || cfg.Gateway.Port > 65535 {
		return fmt.Errorf("invalid gateway port: %d, port must be between 1 and 65535", cfg.Gateway.Port)
	}
	return nil
}

func executeReload(
	ctx context.Context,
	agentLoop *agent.AgentLoop,
	newCfg *config.Config,
	provider *providers.LLMProvider,
	runningServices *services,
	msgBus *bus.MessageBus,
	allowEmptyStartup bool,
	debug bool,
) (err error) {
	startedAt := time.Now()
	publishGatewayEvent(agentLoop, runtimeevents.KindGatewayReloadStarted, startedAt, nil)
	defer runningServices.reloading.Store(false)
	defer func() {
		if err != nil {
			publishGatewayEvent(agentLoop, runtimeevents.KindGatewayReloadFailed, startedAt, err)
			return
		}
		publishGatewayEvent(agentLoop, runtimeevents.KindGatewayReloadCompleted, startedAt, nil)
	}()

	err = handleConfigReload(ctx, agentLoop, newCfg, provider, runningServices, msgBus, allowEmptyStartup, debug)
	return err
}

func createStartupProvider(
	cfg *config.Config,
	allowEmptyStartup bool,
) (providers.LLMProvider, string, error) {
	modelName := cfg.Agents.Defaults.GetModelName()
	if modelName == "" && allowEmptyStartup {
		reason := "no default model configured; gateway started in limited mode"
		fmt.Printf("⚠ Warning: %s\n", reason)
		logger.WarnCF("gateway", "Gateway started without default model", map[string]any{
			"limited_mode": true,
		})
		return &startupBlockedProvider{reason: reason}, "", nil
	}

	return providers.CreateProvider(cfg)
}

func setupAndStartServices(
	cfg *config.Config,
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
	authToken string,
	listenResult netbind.OpenResult,
) (*services, error) {
	runningServices := &services{}

	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	var err error
	runningServices.CronService, err = setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("error setting up cron service: %w", err)
	}
	if err = runningServices.CronService.Start(); err != nil {
		return nil, fmt.Errorf("error starting cron service: %w", err)
	}
	fmt.Println("✓ Cron service started")

	runningServices.HeartbeatService = heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	runningServices.HeartbeatService.SetBus(msgBus)
	runningServices.HeartbeatService.SetHandler(createHeartbeatHandler(agentLoop))
	if err = runningServices.HeartbeatService.Start(); err != nil {
		return nil, fmt.Errorf("error starting heartbeat service: %w", err)
	}
	fmt.Println("✓ Heartbeat service started")

	runningServices.MediaStore = media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
		fms.Start()
	}

	runningServices.ChannelManager, err = channels.NewManager(
		cfg,
		msgBus,
		runningServices.MediaStore,
		channels.WithRuntimeEvents(agentLoop.RuntimeEventBus()),
	)
	if err != nil {
		if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return nil, fmt.Errorf("error creating channel manager: %w", err)
	}

	agentLoop.SetChannelManager(runningServices.ChannelManager)
	agentLoop.SetMediaStore(runningServices.MediaStore)

	transcriber := asr.DetectTranscriber(cfg)
	if transcriber != nil {
		agentLoop.SetTranscriber(transcriber)
		logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
	}

	ttsAvailable := tts.DetectTTS(cfg) != nil

	enabledChannels := runningServices.ChannelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	runningServices.authToken = authToken
	runningServices.HealthServer = health.NewServer(listenResult.ProbeHost, cfg.Gateway.Port, authToken)

	var listenAddr string
	if len(listenResult.Listeners) > 0 {
		listenAddr = listenResult.Listeners[0].Addr().String()
	} else {
		listenAddr = net.JoinHostPort(listenResult.ProbeHost, strconv.Itoa(cfg.Gateway.Port))
	}
	runningServices.ChannelManager.SetupHTTPServerListeners(
		listenResult.Listeners,
		listenAddr,
		runningServices.HealthServer,
	)

	if err = runningServices.ChannelManager.StartAll(context.Background()); err != nil {
		return nil, fmt.Errorf("error starting channels: %w", err)
	}

	logChannelVoiceCapabilities(runningServices.ChannelManager, transcriber != nil, ttsAvailable)

	if transcriber != nil {
		// Start Voice Agent Orchestrator after channels are ready.
		vaCtx, vaCancel := context.WithCancel(context.Background())
		runningServices.VoiceAgentCancel = vaCancel
		voiceAgent := asr.NewAgent(msgBus, transcriber)
		voiceAgent.Start(vaCtx)
	}

	healthAddr := net.JoinHostPort(listenResult.ProbeHost, strconv.Itoa(cfg.Gateway.Port))
	fmt.Printf(
		"✓ Health endpoints available at http://%s/health, /ready and /reload (POST)\n",
		healthAddr,
	)

	stateManager := state.NewManager(cfg.WorkspacePath())
	runningServices.DeviceService = devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	runningServices.DeviceService.SetBus(msgBus)
	if err = runningServices.DeviceService.Start(context.Background()); err != nil {
		logger.ErrorCF("device", "Error starting device service", map[string]any{"error": err.Error()})
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	return runningServices, nil
}

func stopAndCleanupServices(runningServices *services, shutdownTimeout time.Duration, isReload bool) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// reload should not stop channel manager
	if !isReload && runningServices.ChannelManager != nil {
		runningServices.ChannelManager.StopAll(shutdownCtx)
	}
	if runningServices.VoiceAgentCancel != nil {
		runningServices.VoiceAgentCancel()
	}
	if runningServices.DeviceService != nil {
		runningServices.DeviceService.Stop()
	}
	if runningServices.HeartbeatService != nil {
		runningServices.HeartbeatService.Stop()
	}
	if runningServices.CronService != nil {
		runningServices.CronService.Stop()
	}
	if runningServices.MediaStore != nil {
		if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
	}
}

func shutdownGateway(
	runningServices *services,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
	msgBus *bus.MessageBus,
	fullShutdown bool,
) {
	publishGatewayEvent(agentLoop, runtimeevents.KindGatewayShutdown, time.Time{}, nil)

	if cp, ok := provider.(providers.StatefulProvider); ok && fullShutdown {
		cp.Close()
	}

	stopAndCleanupServices(runningServices, gracefulShutdownTimeout, false)

	if fullShutdown && msgBus != nil {
		msgBus.Close()
	}

	agentLoop.Stop()
	agentLoop.Close()

	logger.Info("✓ Gateway stopped")
}

func handleConfigReload(
	ctx context.Context,
	al *agent.AgentLoop,
	newCfg *config.Config,
	providerRef *providers.LLMProvider,
	runningServices *services,
	msgBus *bus.MessageBus,
	allowEmptyStartup bool,
	debug bool,
) error {
	logger.Info("🔄 Config file changed, reloading...")

	newModel := newCfg.Agents.Defaults.ModelName

	logger.Infof(" New model is '%s', recreating provider...", newModel)

	logger.Info("  Stopping all services...")
	stopAndCleanupServices(runningServices, serviceShutdownTimeout, true)

	newProvider, newModelID, err := createStartupProvider(newCfg, allowEmptyStartup)
	if err != nil {
		logger.Errorf("  ⚠ Error creating new provider: %v", err)
		logger.Warn("  Attempting to restart services with old provider and config...")
		if restartErr := restartServices(al, runningServices, msgBus); restartErr != nil {
			logger.Errorf("  ⚠ Failed to restart services: %v", restartErr)
		}
		return fmt.Errorf("error creating new provider: %w", err)
	}

	if newModelID != "" {
		newCfg.Agents.Defaults.ModelName = newModelID
	}

	reloadCtx, reloadCancel := context.WithTimeout(context.Background(), providerReloadTimeout)
	defer reloadCancel()

	if err := al.ReloadProviderAndConfig(reloadCtx, newProvider, newCfg); err != nil {
		logger.Errorf("  ⚠ Error reloading agent loop: %v", err)
		if cp, ok := newProvider.(providers.StatefulProvider); ok {
			cp.Close()
		}
		logger.Warn("  Attempting to restart services with old provider and config...")
		if restartErr := restartServices(al, runningServices, msgBus); restartErr != nil {
			logger.Errorf("  ⚠ Failed to restart services: %v", restartErr)
		}
		return fmt.Errorf("error reloading agent loop: %w", err)
	}

	*providerRef = newProvider

	logger.Info("  Restarting all services with new configuration...")
	if err := restartServices(al, runningServices, msgBus); err != nil {
		logger.Errorf("  ⚠ Error restarting services: %v", err)
		return fmt.Errorf("error restarting services: %w", err)
	}

	logger.Info("  ✓ Provider, configuration, and services reloaded successfully (thread-safe)")

	// Debug mode permanently overrides the config log level to DEBUG.
	if !debug {
		// Update log level last so that reload-related info/warn logs above are not suppressed.
		effectiveLogLevel := config.EffectiveGatewayLogLevel(newCfg)
		logger.SetLevelFromString(effectiveLogLevel)
		logger.Infof("Log level changing from current to %q", effectiveLogLevel)
	}

	return nil
}

func restartServices(
	al *agent.AgentLoop,
	runningServices *services,
	msgBus *bus.MessageBus,
) error {
	cfg := al.GetConfig()

	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	var err error
	runningServices.CronService, err = setupCronTool(
		al,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)
	if err != nil {
		return fmt.Errorf("error restarting cron service: %w", err)
	}
	if err = runningServices.CronService.Start(); err != nil {
		return fmt.Errorf("error restarting cron service: %w", err)
	}
	fmt.Println("  ✓ Cron service restarted")

	runningServices.HeartbeatService = heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	runningServices.HeartbeatService.SetBus(msgBus)
	runningServices.HeartbeatService.SetHandler(createHeartbeatHandler(al))
	if err = runningServices.HeartbeatService.Start(); err != nil {
		return fmt.Errorf("error restarting heartbeat service: %w", err)
	}
	fmt.Println("  ✓ Heartbeat service restarted")

	runningServices.MediaStore = media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
		fms.Start()
	}
	al.SetMediaStore(runningServices.MediaStore)

	al.SetChannelManager(runningServices.ChannelManager)

	if err = runningServices.ChannelManager.Reload(context.Background(), cfg); err != nil {
		return fmt.Errorf("error reload channels: %w", err)
	}
	fmt.Println("  ✓ Channels restarted.")

	enabledChannels := runningServices.ChannelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("  ✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("  ⚠ Warning: No channels enabled")
	}

	stateManager := state.NewManager(cfg.WorkspacePath())
	runningServices.DeviceService = devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	runningServices.DeviceService.SetBus(msgBus)
	if err := runningServices.DeviceService.Start(context.Background()); err != nil {
		logger.WarnCF("device", "Failed to restart device service", map[string]any{"error": err.Error()})
	} else if cfg.Devices.Enabled {
		fmt.Println("  ✓ Device event service restarted")
	}

	transcriber := asr.DetectTranscriber(cfg)
	al.SetTranscriber(transcriber)
	if transcriber != nil {
		logger.InfoCF("voice", "Transcription re-enabled (agent-level)", map[string]any{"provider": transcriber.Name()})

		// Start Voice Agent Orchestrator on reload
		vaCtx, vaCancel := context.WithCancel(context.Background())
		runningServices.VoiceAgentCancel = vaCancel
		voiceAgent := asr.NewAgent(msgBus, transcriber)
		voiceAgent.Start(vaCtx)
	} else {
		logger.InfoCF("voice", "Transcription disabled", nil)
	}

	ttsAvailable := tts.DetectTTS(cfg) != nil
	logChannelVoiceCapabilities(runningServices.ChannelManager, transcriber != nil, ttsAvailable)
	// NOTE: PID file is written once at startup and not updated on reload.
	// Changing the gateway listen address requires a full restart.

	return nil
}

func setupConfigWatcherPolling(configPath string, debug bool) (chan *config.Config, func()) {
	configChan := make(chan *config.Config, 1)
	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		lastModTime := getFileModTime(configPath)
		lastSize := getFileSize(configPath)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentModTime := getFileModTime(configPath)
				currentSize := getFileSize(configPath)

				if currentModTime.After(lastModTime) || currentSize != lastSize {
					if debug {
						logger.Debugf("🔍 Config file change detected")
					}

					time.Sleep(500 * time.Millisecond)

					lastModTime = currentModTime
					lastSize = currentSize

					newCfg, err := config.LoadConfig(configPath)
					if err != nil {
						logger.Errorf("⚠ Error loading new config: %v", err)
						logger.Warn("  Using previous valid config")
						continue
					}

					if err := newCfg.ValidateModelList(); err != nil {
						logger.Errorf("  ⚠ New config validation failed: %v", err)
						logger.Warn("  Using previous valid config")
						continue
					}

					logger.Info("✓ Config file validated and loaded")

					select {
					case configChan <- newCfg:
					default:
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

func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

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
) (*cron.CronService, error) {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	cronService := cron.NewCronService(cronStorePath, nil)

	var cronTool *tools.CronTool
	if cfg.Tools.IsToolEnabled("cron") {
		var err error
		cronTool, err = tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
		if err != nil {
			return nil, fmt.Errorf("critical error during CronTool initialization: %w", err)
		}

		agentLoop.RegisterTool(cronTool)
	}

	if cronTool != nil {
		cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
			result := cronTool.ExecuteJob(context.Background(), job)
			return result, nil
		})
	}

	return cronService, nil
}

func createHeartbeatHandler(agentLoop *agent.AgentLoop) func(prompt, channel, chatID string) *tools.ToolResult {
	return func(prompt, channel, chatID string) *tools.ToolResult {
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}

		response, err := agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		return tools.SilentResult(response)
	}
}

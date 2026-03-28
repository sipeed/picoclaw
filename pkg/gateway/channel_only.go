package gateway

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/voice"
)

const (
	channelPanicFile = "channel_panic.log"
	channelLogFile   = "channel.log"
)

type channelServices struct {
	MediaStore     media.MediaStore
	ChannelManager *channels.Manager
	HealthServer   *health.Server
	ListenHost     string
	ListenPort     int
}

// RunChannelsOnly starts channel and agent loop runtime without gateway side services.
func RunChannelsOnly(debug bool, homePath, configPath string, allowEmptyStartup bool) error {
	panicPath := filepath.Join(homePath, logPath, channelPanicFile)
	panicFunc, err := logger.InitPanic(panicPath)
	if err != nil {
		return fmt.Errorf("error initializing panic log: %w", err)
	}
	defer panicFunc()

	if err = logger.EnableFileLogging(filepath.Join(homePath, logPath, channelLogFile)); err != nil {
		panic(fmt.Sprintf("error enabling file logging: %v", err))
	}
	defer logger.DisableFileLogging()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	logger.SetLevelFromString(cfg.Gateway.LogLevel)

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	provider, modelID, err := createStartupProvider(cfg, allowEmptyStartup)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	if allowEmptyStartup {
		fmt.Println(" ⚠ Channel-only runtime started in limited mode")
	}
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

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

	runningServices, err := setupAndStartChannelServices(cfg, agentLoop, msgBus)
	if err != nil {
		return err
	}

	if runningServices.ListenHost != "" {
		fmt.Printf("✓ Channel runtime started on %s:%d\n", runningServices.ListenHost, runningServices.ListenPort)
	} else {
		fmt.Println("✓ Channel runtime started (shared HTTP server disabled)")
	}
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down channel runtime...")
	shutdownChannelRuntime(runningServices, agentLoop, provider)
	return nil
}

func setupAndStartChannelServices(
	cfg *config.Config,
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
) (*channelServices, error) {
	runningServices := &channelServices{}

	runningServices.MediaStore = media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
		fms.Start()
	}

	var err error
	runningServices.ChannelManager, err = channels.NewManager(cfg, msgBus, runningServices.MediaStore)
	if err != nil {
		if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return nil, fmt.Errorf("error creating channel manager: %w", err)
	}

	agentLoop.SetChannelManager(runningServices.ChannelManager)
	agentLoop.SetMediaStore(runningServices.MediaStore)

	if transcriber := voice.DetectTranscriber(cfg); transcriber != nil {
		agentLoop.SetTranscriber(transcriber)
		logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
	}

	enabledChannels := runningServices.ChannelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", strings.Join(enabledChannels, ", "))
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	listenHost, resolveErr := resolveChannelOnlyListenHost(cfg.Gateway.Host, cfg.Gateway.Port)
	if resolveErr != nil {
		logger.WarnCF("channels", "Shared HTTP server disabled in channel-only mode", map[string]any{
			"host":  cfg.Gateway.Host,
			"port":  cfg.Gateway.Port,
			"error": resolveErr.Error(),
		})
	} else {
		addr := net.JoinHostPort(listenHost, strconv.Itoa(cfg.Gateway.Port))
		runningServices.ListenHost = listenHost
		runningServices.ListenPort = cfg.Gateway.Port
		runningServices.HealthServer = health.NewServer(listenHost, cfg.Gateway.Port)
		runningServices.ChannelManager.SetupHTTPServer(addr, runningServices.HealthServer)
	}

	if err = runningServices.ChannelManager.StartAll(context.Background()); err != nil {
		if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return nil, fmt.Errorf("error starting channels: %w", err)
	}

	if runningServices.ListenHost != "" {
		fmt.Printf(
			"✓ Health endpoints available at http://%s:%d/health and /ready\n",
			runningServices.ListenHost,
			runningServices.ListenPort,
		)
	} else {
		fmt.Println("⚠ Shared HTTP server disabled; /health and webhook endpoints are unavailable")
	}

	return runningServices, nil
}

func resolveChannelOnlyListenHost(host string, port int) (string, error) {
	if err := probeTCPBind(host, port); err == nil {
		return host, nil
	} else if isLoopbackHost(host) {
		fallbackErr := probeTCPBind("0.0.0.0", port)
		if fallbackErr == nil {
			logger.WarnCF(
				"channels",
				"Loopback host unavailable in channel-only mode, fallback to wildcard",
				map[string]any{
					"host": host,
					"port": port,
				},
			)
			return "0.0.0.0", nil
		}
		return "", fmt.Errorf(
			"bind fallback 0.0.0.0:%d failed after %s:%d failed: %w (original error: %v)",
			port,
			host,
			port,
			fallbackErr,
			err,
		)
	} else {
		return "", fmt.Errorf("bind %s:%d failed: %w", host, port, err)
	}
}

func probeTCPBind(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	_ = ln.Close()
	return nil
}

func isLoopbackHost(host string) bool {
	normalized := strings.TrimSpace(strings.ToLower(host))
	if normalized == "localhost" {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func shutdownChannelRuntime(
	runningServices *channelServices,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
) {
	if cp, ok := provider.(providers.StatefulProvider); ok {
		cp.Close()
	}

	if runningServices != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		if runningServices.ChannelManager != nil {
			runningServices.ChannelManager.StopAll(shutdownCtx)
		}

		if runningServices.MediaStore != nil {
			if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
				fms.Stop()
			}
		}
	}

	if agentLoop != nil {
		agentLoop.Stop()
		agentLoop.Close()
	}
}

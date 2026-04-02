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
	"github.com/sipeed/picoclaw/pkg/audio/asr"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	channelPanicFile = "channel_panic.log"
	channelLogFile   = "channel.log"
)

type channelServices struct {
	MediaStore       media.MediaStore
	ChannelManager   *channels.Manager
	HealthServer     *health.Server
	VoiceAgentCancel context.CancelFunc
	ListenHost       string
	ListenPort       int
	ListenAddr       string
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
		return fmt.Errorf("error enabling file logging: %w", err)
	}
	defer logger.DisableFileLogging()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if err = preCheckConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	} else {
		effectiveLogLevel := config.EffectiveGatewayLogLevel(cfg)
		logger.SetLevelFromString(effectiveLogLevel)
		logger.Infof("Log level set to %q", effectiveLogLevel)
	}

	provider, modelID, err := createStartupProviderForRuntime(cfg, allowEmptyStartup, "channel-only runtime")
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()

	var toolsCount int
	if toolsRaw, ok := startupInfo["tools"].(map[string]any); ok && toolsRaw != nil {
		if v, ok := toolsRaw["count"].(int); ok {
			toolsCount = v
		}
	}

	var skillsAvailable, skillsTotal int
	if skillsRaw, ok := startupInfo["skills"].(map[string]any); ok && skillsRaw != nil {
		if v, ok := skillsRaw["available"].(int); ok {
			skillsAvailable = v
		}
		if v, ok := skillsRaw["total"].(int); ok {
			skillsTotal = v
		}
	}

	if toolsCount == 0 && skillsAvailable == 0 && skillsTotal == 0 {
		fmt.Println("  • Agent startup info not available")
	} else {
		fmt.Printf("  • Tools: %d loaded\n", toolsCount)
		fmt.Printf("  • Skills: %d/%d available\n", skillsAvailable, skillsTotal)
	}

	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsCount,
			"skills_total":     skillsTotal,
			"skills_available": skillsAvailable,
		})

	runningServices, err := setupAndStartChannelServices(cfg, agentLoop, msgBus)
	if err != nil {
		return err
	}

	if runningServices.HealthServer != nil {
		if runningServices.ListenHost == "" {
			fmt.Printf(
				"✓ Channel runtime started (shared HTTP server enabled on all interfaces, port %d)\n",
				runningServices.ListenPort,
			)
		} else {
			fmt.Printf("✓ Channel runtime started on %s\n", runningServices.ListenAddr)
		}
	} else {
		fmt.Println("✓ Channel runtime started (shared HTTP server disabled)")
	}
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentLoop.Run(ctx)

	httpErrCh := runningServices.ChannelManager.HTTPServerErrors()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		logger.Info("Shutting down channel runtime...")
		shutdownChannelRuntime(runningServices, agentLoop, provider)
		return nil
	case err := <-httpErrCh:
		logger.Errorf("Shared HTTP server stopped in channel-only mode: %v", err)
		shutdownChannelRuntime(runningServices, agentLoop, provider)
		return fmt.Errorf("shared HTTP server failed: %w", err)
	}
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

	var transcriber asr.Transcriber
	transcriber = asr.DetectTranscriber(cfg)
	if transcriber != nil {
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
		runningServices.ListenAddr = addr
		runningServices.HealthServer = health.NewServer(listenHost, cfg.Gateway.Port, "")
		runningServices.ChannelManager.SetupHTTPServer(addr, runningServices.HealthServer)
	}

	if err = runningServices.ChannelManager.StartAll(context.Background()); err != nil {
		if fms, ok := runningServices.MediaStore.(*media.FileMediaStore); ok {
			fms.Stop()
		}
		return nil, fmt.Errorf("error starting channels: %w", err)
	}

	if transcriber != nil {
		vaCtx, vaCancel := context.WithCancel(context.Background())
		runningServices.VoiceAgentCancel = vaCancel
		voiceAgent := asr.NewAgent(msgBus, transcriber)
		voiceAgent.Start(vaCtx)
	}

	if runningServices.HealthServer != nil {
		runningServices.HealthServer.SetReady(true)
	}

	if runningServices.HealthServer != nil {
		if runningServices.ListenHost == "" {
			fmt.Printf(
				"✓ Health endpoints available on all interfaces at port %d (/health and /ready; /reload returns 503 when not configured)\n",
				runningServices.ListenPort,
			)
		} else {
			fmt.Printf(
				"✓ Health endpoints available at http://%s/health and /ready (/reload returns 503 when not configured)\n",
				runningServices.ListenAddr,
			)
		}
	} else {
		fmt.Println("⚠ Shared HTTP server disabled; /health and webhook endpoints are unavailable")
	}

	return runningServices, nil
}

func resolveChannelOnlyListenHost(host string, port int) (string, error) {
	if err := probeTCPBind(host, port); err == nil {
		return host, nil
	} else if isLoopbackHost(host) {
		// Keep loopback scope when fallback is needed to avoid widening exposure.
		ipv6FallbackErr := probeTCPBind("::1", port)
		if ipv6FallbackErr == nil {
			logger.WarnCF(
				"channels",
				"Loopback host unavailable in channel-only mode, fallback to IPv6 loopback",
				map[string]any{
					"host": host,
					"port": port,
				},
			)
			return "::1", nil
		}

		ipv4FallbackErr := probeTCPBind("127.0.0.1", port)
		if ipv4FallbackErr == nil {
			logger.WarnCF(
				"channels",
				"Loopback host unavailable in channel-only mode, fallback to IPv4 loopback",
				map[string]any{
					"host": host,
					"port": port,
				},
			)
			return "127.0.0.1", nil
		}
		return "", fmt.Errorf(
			"bind fallback [::1]:%d and 127.0.0.1:%d failed after %s:%d failed: ipv6 error: %v; ipv4 error: %v; original error: %v",
			port,
			port,
			host,
			port,
			ipv6FallbackErr,
			ipv4FallbackErr,
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
		if runningServices.HealthServer != nil {
			runningServices.HealthServer.SetReady(false)
		}

		if runningServices.VoiceAgentCancel != nil {
			runningServices.VoiceAgentCancel()
		}

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

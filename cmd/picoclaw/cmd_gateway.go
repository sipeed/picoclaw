// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
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
	"github.com/sipeed/picoclaw/pkg/miniapp"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tailscale"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
)

func gatewayCmd() {
	// Check for --debug and --stats flags
	args := os.Args[2:]
	enableStats := false
	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
		}
		if arg == "--stats" {
			enableStats = true
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}
	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.Model = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider, enableStats)

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
		// Deliver response to user when a plan interview/review needs resuming.
		// For async tasks (spawn), results are delivered separately via processSystemMessage.
		if status := agentLoop.GetPlanStatus(); status == "interviewing" || status == "review" {
			return tools.UserResult(response)
		}
		return tools.SilentResult(response)
	})

	// Reset heartbeat suppression when a real user message arrives
	agentLoop.OnUserMessage = heartbeatService.ResetSuppression

	channelManager, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		fmt.Printf("Error creating channel manager: %v\n", err)
		os.Exit(1)
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

	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)

	// Mini App setup: register routes and determine TLS mode
	useTLS := false
	var tlsCert, tlsKey string
	var miniappNotifier *miniapp.StateNotifier
	if cfg.Channels.Telegram.Enabled {
		webAppURL := cfg.Channels.Telegram.WebAppURL
		if webAppURL == "" {
			// Auto-detect Tailscale hostname and fetch TLS cert
			hostname, tsErr := tailscale.DetectHostname()
			if tsErr != nil {
				logger.InfoCF("miniapp", "Tailscale not available, Mini App disabled", map[string]any{"error": tsErr.Error()})
			} else {
				certDir := filepath.Join(cfg.WorkspacePath(), "state", "certs")
				certFile, keyFile, certErr := tailscale.FetchCert(hostname, certDir)
				if certErr != nil {
					logger.ErrorCF("miniapp", "Failed to fetch TLS cert", map[string]any{"error": certErr.Error()})
				} else {
					webAppURL = fmt.Sprintf("https://%s:%d/miniapp", hostname, cfg.Gateway.Port)
					cfg.Channels.Telegram.WebAppURL = webAppURL
					tlsCert, tlsKey = certFile, keyFile
					useTLS = true
				}
			}
		}

		if webAppURL != "" {
			provider := &agentLoopDataProvider{loop: agentLoop, workspace: cfg.WorkspacePath()}
			sender := &telegramCommandSender{bus: msgBus}
			miniappNotifier = miniapp.NewStateNotifier()
			handler := miniapp.NewHandler(provider, sender, cfg.Channels.Telegram.Token, miniappNotifier, cfg.Channels.Telegram.AllowFrom, cfg.WorkspacePath())
			agentLoop.OnStateChange = miniappNotifier.Notify
			handler.RegisterRoutes(healthServer.Mux())

			// Register dev preview tool for all agents
			devPreviewTool := tools.NewDevPreviewTool(handler)
			agentLoop.RegisterTool(devPreviewTool)

			fmt.Printf("✓ Mini App registered at %s\n", webAppURL)
		}
	}

	go func() {
		var serverErr error
		if useTLS {
			serverErr = healthServer.StartTLS(tlsCert, tlsKey)
		} else {
			serverErr = healthServer.Start()
		}
		if serverErr != nil && serverErr != http.ErrServerClosed {
			logger.ErrorCF("health", "Health server error", map[string]any{"error": serverErr.Error()})
		}
	}()
	if useTLS {
		fmt.Printf("✓ Health endpoints available at https://%s:%d/health and /ready (TLS)\n", cfg.Gateway.Host, cfg.Gateway.Port)
	} else {
		fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)
	}

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	if miniappNotifier != nil {
		miniappNotifier.Close()
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	healthServer.Stop(shutdownCtx)
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	fmt.Println("✓ Gateway stopped")
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

	// Convert agent.PlanPhase → miniapp.PlanPhase
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
			SessionKey: e.SessionKey,
			Channel:    e.Channel,
			ChatID:     e.ChatID,
			TouchDir:   e.TouchDir,
			LastSeenAt: e.LastSeenAt.Format(time.RFC3339),
			AgeSec:     int(time.Since(e.LastSeenAt).Seconds()),
		}
	}
	return result
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
			if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
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
	s.bus.PublishInbound(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: senderID,
		ChatID:   chatID,
		Content:  command,
		Metadata: map[string]string{
			"source": "webapp",
		},
	})
}

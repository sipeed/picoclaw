// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// activeTask tracks a running agent task for live status and intervention.
type activeTask struct {
	Description    string
	Result         string // LLM response summary for completion notification
	Iteration      int
	MaxIter        int
	StartedAt      time.Time
	cancel         context.CancelFunc
	interrupt      chan string // buffered 1, for user message injection
	toolLog        []toolLogEntry
	lastError      *toolLogEntry // sticky: most recent error, persists across iterations
	projectDir     string        // detected from exec cd target (authoritative)
	fileCommonDir  string        // LCP of file paths relative to workspace (fallback)
	streamedChunks bool          // true after onChunk fires at least once
	messageContent string        // last content sent by the message tool (for inclusion in completion)
	mu             sync.Mutex
}

// toolLogEntry records a single tool call for the live terminal view.
type toolLogEntry struct {
	Name      string
	ArgsSnip  string // first ~80 chars of args
	Result    string // "✓ 4.9s" or "✗ 3.2s"
	ErrDetail string // non-empty on error — e.g. "Exit code: exit status 1"
}

// maxToolLogEntries limits the sliding window of tool log entries
// kept in memory and displayed in status messages.
const maxToolLogEntries = 5

// sessionSemaphore is a per-session mutex using a buffered channel.
type sessionSemaphore struct {
	ch chan struct{}
}

func newSessionSemaphore() *sessionSemaphore {
	s := &sessionSemaphore{ch: make(chan struct{}, 1)}
	s.ch <- struct{}{} // initially unlocked
	return s
}

type AgentLoop struct {
	bus                     *bus.MessageBus
	cfg                     *config.Config
	registry                *AgentRegistry
	state                   *state.Manager
	stats                   *stats.Tracker // nil when --stats not passed
	running                 atomic.Bool
	summarizing             sync.Map
	fallback                *providers.FallbackChain
	channelManager          *channels.Manager
	mediaStore              media.MediaStore
	providerCache           map[string]providers.LLMProvider
	planStartPending        bool     // set by /plan start to trigger LLM execution
	planClearHistory        bool     // set by /plan start clear to wipe history on transition
	sessionLocks            sync.Map // sessionKey → *sessionSemaphore
	activeTasks             sync.Map // sessionKey → *activeTask
	sessions                *SessionTracker
	lastSystemPrompt        atomic.Value // string — last system prompt sent to LLM
	promptDirty             atomic.Bool  // true = rebuild needed on next GetSystemPrompt read
	OnStateChange           func()       // called on plan/session/skills mutations
	OnUserMessage           func()       // called when a real user message is processed
	saveConfig              func(*config.Config) error
	onHeartbeatThreadUpdate func(int)
	orchBroadcaster         *orch.Broadcaster  // nil when --orchestration not set
	orchReporter            orch.AgentReporter // always non-nil (Noop when disabled)
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string // Session identifier for history/context
	Channel         string // Target channel for tool execution
	ChatID          string // Target chat ID for tool execution
	UserMessage     string // User message content (may include prefix)
	HistoryMessage  string // If set, save this to history instead of UserMessage (for skill compaction)
	DefaultResponse string // Response when LLM returns empty
	EnableSummary   bool   // Whether to trigger summarization
	SendResponse    bool   // Whether to send response via bus
	NoHistory       bool   // If true, don't load session history (for heartbeat)
	TaskID          string // Unique task ID for background task status tracking
	Background      bool   // If true, this is a background task (cron/heartbeat) — enables live task notifications
	SystemMessage   bool   // If true, this is a system message (subagent result) — skip placeholder and plan nudge
}

const defaultResponse = "I've completed processing but have no response to give. Increase `max_tool_iterations` in config.json."

func NewAgentLoop(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	provider providers.LLMProvider,
	enableStats ...bool,
) *AgentLoop {
	registry := NewAgentRegistry(cfg, provider)

	// Set up shared fallback chain
	cooldown := providers.NewCooldownTracker()
	fallbackChain := providers.NewFallbackChain(cooldown)

	// Create state manager using default agent's workspace for channel recording
	defaultAgent := registry.GetDefaultAgent()
	var stateManager *state.Manager
	if defaultAgent != nil {
		stateManager = state.NewManager(defaultAgent.Workspace)
	}

	providerCache := make(map[string]providers.LLMProvider)

	// Create stats tracker if enabled
	var statsTracker *stats.Tracker
	if len(enableStats) > 0 && enableStats[0] && defaultAgent != nil {
		statsTracker = stats.NewTracker(defaultAgent.Workspace)
	}

	// Determine if orchestration broadcaster is needed (any agent has subagents enabled).
	// Note: instance.go maps defaults.Orchestration → Subagents.Enabled, so --orchestration
	// is automatically reflected here.
	var orchBroadcaster *orch.Broadcaster
	var orchReporter orch.AgentReporter = orch.Noop
	for _, id := range registry.ListAgentIDs() {
		if a, ok := registry.GetAgent(id); ok && a.Subagents != nil && a.Subagents.Enabled {
			orchBroadcaster = orch.NewBroadcaster()
			orchReporter = orchBroadcaster
			break
		}
	}

	al := &AgentLoop{
		bus:             msgBus,
		cfg:             cfg,
		registry:        registry,
		state:           stateManager,
		stats:           statsTracker,
		summarizing:     sync.Map{},
		fallback:        fallbackChain,
		providerCache:   providerCache,
		sessions:        NewSessionTracker(),
		orchBroadcaster: orchBroadcaster,
		orchReporter:    orchReporter,
	}

	// Register shared tools to all agents (needs al for reporter injection).
	registerSharedTools(cfg, msgBus, registry, provider, al)

	return al
}

// reporter returns the active AgentReporter (never nil).
func (al *AgentLoop) reporter() orch.AgentReporter {
	if al.orchReporter == nil {
		return orch.Noop
	}
	return al.orchReporter
}

// SetOrchReporter wires a Broadcaster as the active reporter.
// Called from cmd_gateway.go when --orchestration is set.
// --orchestration なし → 呼ばれない → reporter() は Noop を返す。
func (al *AgentLoop) SetOrchReporter(b *orch.Broadcaster) {
	al.orchBroadcaster = b
	al.orchReporter = b
}

// GetOrchBroadcaster returns the concrete Broadcaster for miniapp wiring.
// Returns nil when orchestration is disabled.
func (al *AgentLoop) GetOrchBroadcaster() *orch.Broadcaster {
	return al.orchBroadcaster
}

func (al *AgentLoop) notifyStateChange() {
	al.promptDirty.Store(true)
	if al.OnStateChange != nil {
		al.OnStateChange()
	}
}

// SetConfigSaver registers a callback used by slash commands that persist runtime config changes.
func (al *AgentLoop) SetConfigSaver(fn func(*config.Config) error) {
	al.saveConfig = fn
}

// SetHeartbeatThreadUpdater registers a callback to apply runtime heartbeat thread updates.
func (al *AgentLoop) SetHeartbeatThreadUpdater(fn func(int)) {
	al.onHeartbeatThreadUpdate = fn
}

// registerSharedTools registers tools that are shared across all agents (web, message, spawn).
func registerSharedTools(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
	al *AgentLoop,
) {
	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		// Web tools
		searchTool, err := tools.NewWebSearchTool(tools.WebSearchToolOptions{
			BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
			BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
			BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
			TavilyAPIKey:         cfg.Tools.Web.Tavily.APIKey,
			TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
			TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
			TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
			DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
			DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
			PerplexityAPIKey:     cfg.Tools.Web.Perplexity.APIKey,
			PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
			PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
			Proxy:                cfg.Tools.Web.Proxy,
		})
		if err != nil {
			logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{
				"agent_id": agentID,
				"error":    err.Error(),
			})
		} else if searchTool != nil {
			agent.Tools.Register(searchTool)
			logger.InfoCF("agent", "Web search provider registered", map[string]any{
				"agent_id": agentID,
				"provider": searchTool.ProviderName(),
			})
		} else {
			logger.WarnCF("agent", "No web search provider configured", map[string]any{
				"agent_id": agentID,
			})
		}
		fetchTool, err := tools.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy)
		if err != nil {
			logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{
				"agent_id": agentID,
				"error":    err.Error(),
			})
		} else {
			agent.Tools.Register(fetchTool)
		}

		// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
		agent.Tools.Register(tools.NewI2CTool())
		agent.Tools.Register(tools.NewSPITool())

		// Message tool
		messageTool := tools.NewMessageTool()
		messageTool.SetSendCallback(func(channel, chatID, content string) error {
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pubCancel()
			return msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
				Channel: channel,
				ChatID:  chatID,
				Content: content,
			})
		})
		agent.Tools.Register(messageTool)

		// Skill discovery and installation tools
		registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
			MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
			ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
		})
		searchCache := skills.NewSearchCache(
			cfg.Tools.Skills.SearchCache.MaxSize,
			time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
		)
		agent.Tools.Register(tools.NewFindSkillsTool(registryMgr, searchCache))
		agent.Tools.Register(tools.NewInstallSkillTool(registryMgr, agent.Workspace))

		// Spawn tool — only registered when orchestration is explicitly enabled.
		if agent.Subagents != nil && agent.Subagents.Enabled {
			webSearchOpts := tools.WebSearchToolOptions{
				BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
				BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
				BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
				TavilyAPIKey:         cfg.Tools.Web.Tavily.APIKey,
				TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
				TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
				TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
				DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
				DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
				PerplexityAPIKey:     cfg.Tools.Web.Perplexity.APIKey,
				PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
				PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
			}
			subagentManager := tools.NewSubagentManager(
				provider,
				agent.Model,
				agent.Workspace,
				msgBus,
				al.reporter(),
				webSearchOpts,
			)
			subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)
			// Wire session recorder for DAG persistence.
			recorder := newSessionRecorder(agent.Sessions)
			conductorKey := routing.BuildAgentMainSessionKey(agent.ID)
			subagentManager.SetSessionRecorder(recorder, conductorKey)
			agent.SubagentMgr = subagentManager
			spawnTool := tools.NewSpawnTool(subagentManager)
			currentAgentID := agentID
			spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
				return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
			})
			agent.Tools.Register(spawnTool)
			// Register blocking subagent tool alongside spawn
			subagentTool := tools.NewSubagentTool(subagentManager)
			agent.Tools.Register(subagentTool)
		}

		// Update context builder with the complete tools registry
		agent.ContextBuilder.SetToolsRegistry(agent.Tools)

		// Set orchestration mode if enabled
		if agent.Subagents != nil && agent.Subagents.Enabled {
			agent.ContextBuilder.SetOrchestrationEnabled(true)
		}
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	// LLM work is dispatched to a background worker so the main loop
	// stays free to handle slash commands (/skills, …) instantly,
	// even while a long tool-call chain is running.
	llmQueue := make(chan bus.InboundMessage, 10)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		al.llmWorker(ctx, llmQueue)
	}()
	defer func() {
		close(llmQueue)
		<-workerDone
	}()

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, ok := al.bus.ConsumeInbound(ctx)
		if !ok {
			continue
		}

		// Echo commands sent from the Mini App so the user can see what was sent.
		if msg.Metadata["source"] == "webapp" && msg.Metadata["echoed"] == "" && msg.Content != "" {
			_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel:         msg.Channel,
				ChatID:          msg.ChatID,
				Content:         "via MiniApp: " + msg.Content,
				SkipPlaceholder: true,
			})
		}

		// Fast path: handle slash commands immediately without blocking the LLM worker.
		if response, handled := al.handleCommand(ctx, msg); handled {
			if response != "" {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel:         msg.Channel,
					ChatID:          msg.ChatID,
					Content:         response,
					SkipPlaceholder: true,
				})
			}
			// /plan start sets the flag — enqueue a synthetic message so
			// the LLM worker actually begins executing the plan.
			if al.planStartPending {
				al.planStartPending = false
				clearHistory := al.planClearHistory
				al.planClearHistory = false

				if clearHistory {
					if agent := al.registry.GetDefaultAgent(); agent != nil {
						agent.Sessions.SetHistory(msg.SessionKey, nil)
						agent.Sessions.SetSummary(msg.SessionKey, "")
						_ = agent.Sessions.Save(msg.SessionKey)
					}
				}

				// Activate worktree for the session's plan execution
				if agent := al.registry.GetDefaultAgent(); agent != nil {
					taskName := agent.ContextBuilder.Memory().GetPlanTaskName()
					if taskName == "" {
						taskName = "plan-execution"
					}
					planDir := agent.ContextBuilder.GetPlanWorkDir()
					if wt, err := agent.ActivateWorktree(msg.SessionKey, taskName, planDir); err != nil {
						logger.WarnCF("agent", "Worktree activation skipped", map[string]any{"error": err.Error()})
					} else {
						logger.InfoCF("agent", "Worktree activated", map[string]any{"branch": wt.Branch})
					}
				}

				syntheticMeta := map[string]string{"echoed": "1"}
				for k, v := range msg.Metadata {
					if k != "source" {
						syntheticMeta[k] = v
					}
				}
				select {
				case llmQueue <- bus.InboundMessage{
					Channel:    msg.Channel,
					ChatID:     msg.ChatID,
					SenderID:   msg.SenderID,
					SessionKey: msg.SessionKey,
					Content:    "The plan has been approved. Begin executing.",
					Metadata:   syntheticMeta,
				}:
				case <-ctx.Done():
					return nil
				}
			}
			continue
		}

		// Dispatch to LLM worker
		select {
		case llmQueue <- msg:
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

// llmWorker processes LLM messages sequentially in a background goroutine.
func (al *AgentLoop) llmWorker(ctx context.Context, queue <-chan bus.InboundMessage) {
	for msg := range queue {
		if ctx.Err() != nil {
			return
		}

		response, err := al.processMessage(ctx, msg)
		if err != nil {
			response = fmt.Sprintf("Error processing message: %v", err)
		}

		if response != "" {
			alreadySent := false
			defaultAgent := al.registry.GetDefaultAgent()
			if defaultAgent != nil {
				if tool, ok := defaultAgent.Tools.Get("message"); ok {
					if mt, ok := tool.(*tools.MessageTool); ok {
						alreadySent = mt.HasSentInRound()
					}
				}
			}

			if !alreadySent {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: response,
				})
			}
		}
	}
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
}

// Close releases resources held by the loop (e.g. flushes write-behind stats
// and dirty session data). Should be called during graceful shutdown.
func (al *AgentLoop) Close() {
	if al.stats != nil {
		al.stats.Close()
	}
	for _, agentID := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(agentID); ok {
			agent.Sessions.Close()
		}
	}
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	for _, agentID := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(agentID); ok {
			agent.Tools.Register(tool)
		}
	}
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
}

// resolveProvider returns the LLMProvider for the given provider/model pair.
// It caches created providers by "provider/model" key so each combination is
// only resolved once. Looks up model_list first (new format), then falls back
// to the legacy providers section via CreateProviderByName.
func (al *AgentLoop) resolveProvider(
	providerName, modelName string,
	fallback providers.LLMProvider,
) providers.LLMProvider {
	key := strings.ToLower(providerName + "/" + modelName)
	if key == "/" {
		return fallback
	}
	if p, ok := al.providerCache[key]; ok {
		return p
	}

	// Try model_list first (new config format).
	if mc := al.cfg.FindModelConfigByRef(providerName, modelName); mc != nil {
		p, _, err := providers.CreateProviderFromConfig(mc)
		if err == nil {
			al.providerCache[key] = p
			return p
		}
		logger.WarnCF("agent", "Failed to create provider from model_list, trying legacy",
			map[string]any{"provider": providerName, "model": modelName, "error": err.Error()})
	}

	// Fall back to legacy providers section.
	p, err := providers.CreateProviderByName(al.cfg, providerName)
	if err != nil {
		logger.WarnCF("agent", "Failed to create provider for fallback, using primary",
			map[string]any{"provider": providerName, "error": err.Error()})
		return fallback
	}
	al.providerCache[key] = p
	return p
}

// SetMediaStore injects a MediaStore for media lifecycle management.
func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s
}

// inferMediaType determines the media type ("image", "audio", "video", "file")
// from a filename and MIME content type.
func inferMediaType(filename, contentType string) string {
	ct := strings.ToLower(contentType)
	fn := strings.ToLower(filename)

	if strings.HasPrefix(ct, "image/") {
		return "image"
	}
	if strings.HasPrefix(ct, "audio/") || ct == "application/ogg" {
		return "audio"
	}
	if strings.HasPrefix(ct, "video/") {
		return "video"
	}

	// Fallback: infer from extension
	ext := filepath.Ext(fn)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	}

	return "file"
}

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChannel(channel string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChatID(chatID)
}

// RecordLastHeartbeatTarget records the latest heartbeat-safe destination.
// This is intentionally separate from LastChannel so heartbeat routing can be
// reasoned about and evolved without breaking generic last-activity tracking.
func (al *AgentLoop) RecordLastHeartbeatTarget(target string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastHeartbeatTarget(target)
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

func (al *AgentLoop) ProcessDirectWithChannel(
	ctx context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	msg := bus.InboundMessage{
		Channel:    channel,
		SenderID:   "cron",
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
		Metadata: map[string]string{
			"background": "true",
		},
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for heartbeat")
	}
	heartbeatThreadID := 0
	if al.cfg != nil {
		heartbeatThreadID = al.cfg.Channels.Telegram.HeartbeatThreadID
	}
	heartbeatChatID := al.withTelegramThread(channel, chatID, heartbeatThreadID)
	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          heartbeatChatID,
		UserMessage:     content,
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true, // Don't load session history for heartbeat
		Background:      true, // Enable live task notifications on Telegram
	})
}

func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Add message preview to log (show full content for error messages)
	var logContent string
	if strings.Contains(msg.Content, "Error:") || strings.Contains(msg.Content, "error") {
		logContent = msg.Content // Full content for errors
	} else {
		logContent = utils.Truncate(msg.Content, 80)
	}
	logger.InfoCF("agent", fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]any{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		})

	// Handle reply-based intervention for active tasks
	if taskID, ok := msg.Metadata["task_id"]; ok && taskID != "" {
		if val, found := al.activeTasks.Load(taskID); found {
			task := val.(*activeTask)
			content := strings.TrimSpace(msg.Content)
			lower := strings.ToLower(content)

			// Check for stop keywords
			stopKeywords := []string{
				"stop", "cancel", "abort",
				"停止", "中止", "やめて", //nolint:gosmopolitan // intentional CJK stop words
			}
			isStop := false
			for _, kw := range stopKeywords {
				if lower == kw {
					isStop = true
					break
				}
			}

			if isStop {
				task.cancel()
				logger.InfoCF("agent", "Task canceled by user intervention",
					map[string]any{"task_id": taskID})
				return "Task canceled.", nil
			}

			// Inject message into interrupt channel for the tool loop
			select {
			case task.interrupt <- content:
				logger.InfoCF("agent", "User intervention queued",
					map[string]any{"task_id": taskID, "content": utils.Truncate(content, 80)})
			default:
				logger.WarnCF("agent", "Interrupt channel full, message dropped",
					map[string]any{"task_id": taskID})
			}
			return "Intervention sent to running task.", nil
		}
		// Task not found — fall through to normal processing
	}

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Notify listeners that a real user message arrived (e.g. reset heartbeat suppression)
	if al.OnUserMessage != nil {
		al.OnUserMessage()
	}

	// Expand /skill command: inject SKILL.md content into message, then continue to LLM
	var expansionCompact string
	if expanded, compact, ok := al.expandSkillCommand(msg); ok {
		msg.Content = expanded
		expansionCompact = compact
	}

	// Expand /plan <task>: write interview seed, rewrite for LLM interview
	if expanded, compact, ok := al.expandPlanCommand(msg); ok {
		msg.Content = expanded
		expansionCompact = compact
	}

	// Check for commands
	if response, handled := al.handleCommand(ctx, msg); handled {
		return response, nil
	}

	// Route to determine agent and session key
	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel:    msg.Channel,
		AccountID:  msg.Metadata["account_id"],
		Peer:       extractPeer(msg),
		ParentPeer: extractParentPeer(msg),
		GuildID:    msg.Metadata["guild_id"],
		TeamID:     msg.Metadata["team_id"],
	})

	agent, ok := al.registry.GetAgent(route.AgentID)
	if !ok {
		agent = al.registry.GetDefaultAgent()
	}
	if agent == nil {
		return "", fmt.Errorf("no agent available for route (agent_id=%s)", route.AgentID)
	}

	// Reset message-tool state for this round so we don't skip publishing due to a previous round.
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(msg.Channel, msg.ChatID)
		}
	}

	// Use routed session key, but honor ANY pre-set session key (for ProcessDirect/cron)
	sessionKey := route.SessionKey
	if msg.SessionKey != "" {
		sessionKey = msg.SessionKey
	}

	logger.InfoCF("agent", "Routed message",
		map[string]any{
			"agent_id":    agent.ID,
			"session_key": sessionKey,
			"matched_by":  route.MatchedBy,
		})

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		HistoryMessage:  expansionCompact,
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
		SendResponse:    false,
		Background:      msg.Metadata["background"] == "true",
	})
}

func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	if msg.Channel != "system" {
		return "", fmt.Errorf("processSystemMessage called with non-system message channel: %s", msg.Channel)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]any{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel, originChatID string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
		originChatID = msg.ChatID[idx+1:]
	} else {
		originChannel = "cli"
		originChatID = msg.ChatID
	}

	// Extract subagent result from message content
	// Format: "Task 'label' completed.\n\nResult:\n<actual content>"
	content := msg.Content
	if idx := strings.Index(content, "Result:\n"); idx >= 0 {
		content = content[idx+8:] // Extract just the result part
	}

	// Skip internal channels - only log, don't send to user
	if constants.IsInternalChannel(originChannel) {
		logger.InfoCF("agent", "Subagent completed (internal channel)",
			map[string]any{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
			})
		return "", nil
	}

	// Inject subagent result into session history without running a full LLM loop.
	// The conductor will see the result on its next turn. This avoids:
	// - Flooding the chat with a response for every subagent completion
	// - Consuming the Telegram "Thinking..." placeholder
	// - Wasting LLM tokens on processing each result individually
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for system message")
	}

	sessionKey := routing.BuildAgentMainSessionKey(agent.ID)
	historyMsg := fmt.Sprintf("[System: %s] %s", msg.SenderID, msg.Content)
	// Write as TurnReport to the store for DAG tracking, with legacy fallback.
	subagentSessionKey := routing.BuildSubagentSessionKey(extractTaskID(msg.SenderID))
	store := agent.Sessions.Store()
	reportTurn := &session.Turn{
		Kind:      session.TurnReport,
		OriginKey: subagentSessionKey,
		Author:    msg.SenderID,
		Messages:  []providers.Message{{Role: "user", Content: historyMsg}},
	}
	if err := store.Append(sessionKey, reportTurn); err != nil {
		logger.ErrorCF("agent", "Failed to record report turn, falling back to legacy",
			map[string]any{"error": err.Error()})
		agent.Sessions.AddMessage(sessionKey, "user", historyMsg)
		agent.Sessions.MarkDirty(sessionKey)
	} else {
		// Update in-memory cache so conductor sees the message on next turn.
		agent.Sessions.AddFullMessage(sessionKey, providers.Message{Role: "user", Content: historyMsg})
		agent.Sessions.AdvanceStored(sessionKey, 1)
	}

	// Send a brief notification (SkipPlaceholder to avoid corrupting status messages)
	label := msg.SenderID
	if idx := strings.LastIndex(label, ":"); idx >= 0 {
		label = label[idx+1:]
	}
	notification := formatSubagentCompletion(label, msg.Metadata)
	subagentThreadID := 0
	if al.cfg != nil {
		subagentThreadID = al.cfg.Channels.Telegram.SubagentThreadID
	}
	notifyChatID := al.withTelegramThread(originChannel, originChatID, subagentThreadID)
	_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
		Channel:         originChannel,
		ChatID:          notifyChatID,
		Content:         notification,
		SkipPlaceholder: true,
	})

	logger.InfoCF("agent", "Subagent result injected into session history",
		map[string]any{
			"sender_id":   msg.SenderID,
			"session_key": sessionKey,
			"content_len": len(content),
		})

	return "", nil
}

// extractTaskID extracts the task ID from a sender ID like "subagent:subagent-1".
func extractTaskID(senderID string) string {
	if idx := strings.LastIndex(senderID, ":"); idx >= 0 {
		return senderID[idx+1:]
	}
	return senderID
}

// formatSubagentCompletion builds the user-facing notification for a completed subagent.
// If metadata contains duration_ms and tool_calls it produces e.g.:
//
//	"📋 scout-1 completed (3.2s, 5 tool calls)."
//
// Without metadata it falls back to the plain "📋 scout-1 completed." format.
func formatSubagentCompletion(label string, metadata map[string]string) string {
	if len(metadata) == 0 {
		return fmt.Sprintf("📋 %s completed.", label)
	}
	durationMs, _ := strconv.ParseInt(metadata["duration_ms"], 10, 64)
	toolCalls, _ := strconv.Atoi(metadata["tool_calls"])

	if durationMs <= 0 && toolCalls <= 0 {
		return fmt.Sprintf("📋 %s completed.", label)
	}

	parts := make([]string, 0, 2)
	if durationMs > 0 {
		parts = append(parts, formatDurationMs(durationMs))
	}
	if toolCalls > 0 {
		if toolCalls == 1 {
			parts = append(parts, "1 tool call")
		} else {
			parts = append(parts, fmt.Sprintf("%d tool calls", toolCalls))
		}
	}
	return fmt.Sprintf("📋 %s completed (%s).", label, strings.Join(parts, ", "))
}

// formatDurationMs converts milliseconds to a human-readable duration string.
// Examples: 800 → "0.8s", 1200 → "1.2s", 65000 → "1m5s", 3661000 → "61m1s".
func formatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	totalSec := ms / 1000
	if totalSec < 60 {
		tenths := (ms % 1000) / 100
		return fmt.Sprintf("%d.%ds", totalSec, tenths)
	}
	mins := totalSec / 60
	sec := totalSec % 60
	if sec == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dm%ds", mins, sec)
}

func (al *AgentLoop) withTelegramThread(channel, chatID string, threadID int) string {
	if channel != "telegram" || threadID <= 0 || chatID == "" {
		return chatID
	}

	baseChatID := chatID
	if slash := strings.Index(baseChatID, "/"); slash >= 0 {
		baseChatID = baseChatID[:slash]
	}
	if baseChatID == "" {
		return chatID
	}

	return fmt.Sprintf("%s/%d", baseChatID, threadID)
}

// acquireSessionLock gets or creates a per-session semaphore and acquires it.
// Returns false if the context is canceled before the lock is acquired.
func (al *AgentLoop) acquireSessionLock(ctx context.Context, sessionKey string) bool {
	val, _ := al.sessionLocks.LoadOrStore(sessionKey, newSessionSemaphore())
	sem := val.(*sessionSemaphore)
	select {
	case <-sem.ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// releaseSessionLock releases the per-session semaphore.
func (al *AgentLoop) releaseSessionLock(sessionKey string) {
	if val, ok := al.sessionLocks.Load(sessionKey); ok {
		sem := val.(*sessionSemaphore)
		sem.ch <- struct{}{}
	}
}

// runAgentLoop is the core message processing logic.
func (al *AgentLoop) runAgentLoop(ctx context.Context, agent *AgentInstance, opts processOptions) (string, error) {
	// -1. Acquire per-session lock to prevent concurrent access on the same session
	if !al.acquireSessionLock(ctx, opts.SessionKey) {
		return "", fmt.Errorf("context canceled while waiting for session lock")
	}
	defer al.releaseSessionLock(opts.SessionKey)

	// Report session lifecycle to canvas.
	al.reporter().ReportSpawn(opts.SessionKey, opts.Channel, opts.UserMessage)
	defer al.reporter().ReportGC(opts.SessionKey, "completed")

	// -0. Create cancelable child context and register active task
	taskCtx, taskCancel := context.WithCancel(ctx)
	defer taskCancel()

	task := &activeTask{
		Description: utils.Truncate(opts.UserMessage, 80),
		MaxIter:     agent.MaxIterations,
		StartedAt:   time.Now(),
		cancel:      taskCancel,
		interrupt:   make(chan string, 1),
	}

	// Guarantee heartbeat worktree cleanup on ALL exit paths (error, panic, normal).
	// Wait for spawned subagents first so they aren't killed mid-flight.
	// After auto-commit, attempt to merge the worktree branch into main.
	defer func() {
		if opts.Background {
			if agent.SubagentMgr != nil {
				agent.SubagentMgr.WaitAll(35 * time.Minute) // slightly above spawnTimeout
			}
			wt := agent.GetWorktree(opts.SessionKey)
			if wt != nil {
				// 1. Auto-commit uncommitted changes in worktree
				if git.HasUncommittedChanges(wt.Path) {
					_ = git.AutoCommit(wt.Path, "heartbeat: auto-save")
				}

				// 2. Check if there are unique commits worth merging
				repoRoot := git.FindRepoRoot(agent.Workspace)
				ahead := git.CommitsAhead(repoRoot, wt.BaseBranch, wt.Branch)

				if ahead > 0 && repoRoot != "" {
					// 3. Try fast-forward merge into base branch
					mr := git.MergeWorktreeBranch(repoRoot, wt)

					// 4. Notify based on merge result
					if !constants.IsInternalChannel(opts.Channel) {
						cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
						if mr.Merged {
							_ = al.bus.PublishOutbound(cleanupCtx, bus.OutboundMessage{
								Channel: opts.Channel,
								ChatID:  opts.ChatID,
								Content: fmt.Sprintf("Heartbeat: merged %d commit(s) to %s.",
									ahead, wt.BaseBranch),
							})
						} else if mr.Conflict {
							_ = al.bus.PublishOutbound(cleanupCtx, bus.OutboundMessage{
								Channel: opts.Channel,
								ChatID:  opts.ChatID,
								Content: fmt.Sprintf("Heartbeat: merge conflict on branch `%s` — manual merge needed.",
									mr.Branch),
							})
						}
						cleanupCancel()
					}
				}

				// 5. Dispose worktree (branch auto-deleted if merged, kept if conflict)
				agent.DeactivateWorktree(opts.SessionKey, "", false)
			}
		}
	}()

	// For background tasks (cron/heartbeat), generate a TaskID and send notification
	isBackgroundTask := opts.Background && al.state != nil
	if isBackgroundTask && opts.TaskID == "" {
		opts.TaskID = fmt.Sprintf("task-%s-%d", opts.SessionKey, time.Now().UnixMilli())

		// Determine notification channel: use opts.Channel if already a real channel,
		// otherwise resolve from last active channel
		notifyChannel := opts.Channel
		notifyChatID := opts.ChatID
		if constants.IsInternalChannel(notifyChannel) || notifyChannel == "" {
			if lastChannel := al.state.GetLastChannel(); lastChannel != "" {
				if idx := strings.Index(lastChannel, ":"); idx > 0 {
					notifyChannel = lastChannel[:idx]
					notifyChatID = lastChannel[idx+1:]
				}
			}
		}

		if notifyChannel != "" && notifyChatID != "" && !constants.IsInternalChannel(notifyChannel) {
			// Override opts channel/chatID for status updates
			opts.Channel = notifyChannel
			opts.ChatID = notifyChatID

			// Send initial task notification
			_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel:      notifyChannel,
				ChatID:       notifyChatID,
				Content:      fmt.Sprintf("\U0001F916 Background task started\n%s", task.Description),
				IsTaskStatus: true,
				TaskID:       opts.TaskID,
			})
		}
	}

	// Shared variable for capturing LLM's final response. The defer below reads it
	// to include the response in the task completion message.
	var finalContent string

	// Use TaskID as key if available (for background tasks), else sessionKey
	taskKey := opts.SessionKey
	if opts.TaskID != "" {
		taskKey = opts.TaskID
	}
	al.activeTasks.Store(taskKey, task)
	defer func() {
		al.activeTasks.Delete(taskKey)

		// Publish final task status on completion for background tasks.
		// Include finalContent so the LLM response appears in the same bubble
		// as the completion status, avoiding duplicate messages.
		if opts.TaskID != "" {
			elapsed := time.Since(task.StartedAt)
			completionMsg := fmt.Sprintf("\u2705 Task completed (%.1fs)", elapsed.Seconds())

			// Determine the best content to show in the completion bubble.
			// Priority: message tool content > finalContent > task.Result
			task.mu.Lock()
			msgContent := task.messageContent
			task.mu.Unlock()

			var resultContent string
			switch {
			case msgContent != "":
				// The message tool already sent this to the user via the
				// task bubble; re-include it so the completion doesn't erase it.
				resultContent = msgContent
			case finalContent != "" && finalContent != defaultResponse && finalContent != "HEARTBEAT_OK":
				resultContent = finalContent
			default:
				summary := task.Result
				if summary == "" {
					summary = task.Description
				}
				resultContent = summary
			}

			if resultContent != "" {
				combined := completionMsg + "\n\n" + resultContent
				if len([]rune(combined)) <= 4096 {
					completionMsg = combined
				} else {
					// Too long for one bubble: send header as task status,
					// body as regular message (auto-split by SplitMessage).
					doneCtx, doneCancel := context.WithTimeout(context.Background(), 5*time.Second)
					_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
						Channel:      opts.Channel,
						ChatID:       opts.ChatID,
						Content:      completionMsg,
						IsTaskStatus: true,
						TaskID:       opts.TaskID,
						Final:        true,
					})
					_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: resultContent,
					})
					doneCancel()
					return
				}
			}
			doneCtx, doneCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
				Channel:      opts.Channel,
				ChatID:       opts.ChatID,
				Content:      completionMsg,
				IsTaskStatus: true,
				TaskID:       opts.TaskID,
				Final:        true,
			})
			doneCancel()
		}
	}()

	// Replace ctx with the cancelable child context
	ctx = taskCtx

	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last channel", map[string]any{"error": err.Error()})
			}
			if err := al.RecordLastHeartbeatTarget(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last heartbeat target", map[string]any{"error": err.Error()})
			}
		}
	}

	// 1. Update tool contexts
	al.updateToolContexts(agent, opts.Channel, opts.ChatID)

	// 1-bis. For background tasks that don't send a final response (e.g. heartbeat),
	// redirect the message tool to publish as IsTaskStatus so its output lands in
	// the same bubble as the task status instead of creating a separate message.
	if opts.Background && !opts.SendResponse && opts.TaskID != "" {
		if tool, ok := agent.Tools.Get("message"); ok {
			if mt, ok := tool.(*tools.MessageTool); ok {
				taskID := opts.TaskID
				mt.SetSendCallback(func(channel, chatID, content string) error {
					// Capture the message tool's content so the completion
					// defer can include it instead of losing it to an overwrite.
					if task != nil {
						task.mu.Lock()
						task.messageContent = content
						task.mu.Unlock()
					}
					pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer pubCancel()
					return al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
						Channel:      channel,
						ChatID:       chatID,
						Content:      content,
						IsTaskStatus: true,
						TaskID:       taskID,
					})
				})
			}
		}
	}

	// 1a. Set session-specific working directory for bootstrap file lookup.
	// Prefer the tool-detected project directory (touch_dir) from the session tracker,
	// resolved as an absolute path under workspace. Fall back to worktree or workspace.
	if active := al.sessions.ListActive(); len(active) > 0 && active[0].SessionKey == opts.SessionKey &&
		active[0].TouchDir != "" {
		agent.ContextBuilder.SetWorkDir(filepath.Join(agent.Workspace, active[0].TouchDir))
	} else {
		agent.ContextBuilder.SetWorkDir(agent.EffectiveWorkspace(opts.SessionKey))
	}

	// 1b. Inject peer session awareness into system prompt
	projectPath := agent.ContextBuilder.GetPlanWorkDir()
	if projectPath == "" {
		projectPath = agent.Workspace
	}
	peers := al.sessions.GetPeerPurposes(opts.SessionKey, projectPath)
	if len(peers) > 0 {
		var peerNote strings.Builder
		peerNote.WriteString("Other sessions working on this project:\n")
		for _, p := range peers {
			peerNote.WriteString(fmt.Sprintf("- %s: %s (branch: %s)\n", p.SessionKey, p.Purpose, p.Branch))
		}
		peerNote.WriteString("\nAvoid conflicting changes with these sessions.")
		agent.ContextBuilder.SetPeerNote(peerNote.String())
	} else {
		agent.ContextBuilder.SetPeerNote("")
	}

	// 2. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = agent.Sessions.GetHistory(opts.SessionKey)
		summary = agent.Sessions.GetSummary(opts.SessionKey)

		// Sanitize history to remove orphaned tool calls (from crashes/session collisions)
		var removedCount int
		history, removedCount = session.SanitizeHistory(history)
		if removedCount > 0 {
			logger.WarnCF("agent", "Sanitized session history: removed orphaned messages",
				map[string]any{
					"session_key":   opts.SessionKey,
					"removed_count": removedCount,
				})
			// Persist the sanitized history
			agent.Sessions.SetHistory(opts.SessionKey, history)
			_ = agent.Sessions.Save(opts.SessionKey)
		}
	}
	messages := agent.ContextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		nil,
		opts.Channel,
		opts.ChatID,
	)

	// 2b. Interview staleness nudge: if MEMORY.md hasn't been updated for
	// several consecutive turns, inject a reminder so the AI writes its findings.
	const interviewStaleThreshold = 2
	if agent.ContextBuilder.GetPlanStatus() == "interviewing" && agent.interviewStaleCount >= interviewStaleThreshold {
		messages = append(messages, providers.Message{
			Role:    "user",
			Content: "[System] You have been interviewing for several turns without updating memory/MEMORY.md. Please use edit_file now to save your findings to the ## Context section, or organize the plan into ## Phase sections with `- [ ]` checkbox steps if you have enough information.",
		})
	}

	// 2c. Background plan preamble: append to system prompt (high attention)
	// so the LLM knows from the start that it must mark steps [x].
	// Skip if a chat session is actively working on the plan directory.
	if opts.Background && agent.ContextBuilder.HasActivePlan() && agent.ContextBuilder.GetPlanStatus() == "executing" {
		planDir := agent.ContextBuilder.GetPlanWorkDir()
		skipPreamble := planDir != "" && al.sessions.IsActiveInDir(planDir, "heartbeat")
		if !skipPreamble && len(messages) > 0 && messages[0].Role == "system" {
			var sb strings.Builder
			sb.WriteString(messages[0].Content)
			sb.WriteString("\n\n## Background Execution\n")
			sb.WriteString("You are running as a background heartbeat with no conversation history. ")
			sb.WriteString("MEMORY.md is the only shared state between heartbeats. ")
			sb.WriteString(
				"After completing each plan step, immediately use edit_file to mark it [x] in memory/MEMORY.md.",
			)
			messages[0].Content = sb.String()
		}
	}

	// 2d. Snapshot plan status and MEMORY.md size before LLM iteration.
	preStatus := agent.ContextBuilder.GetPlanStatus()
	var preMemoryLen int
	if preStatus == "interviewing" {
		preMemoryLen = len(agent.ContextBuilder.ReadMemory())
	}

	// 3. Save user message to session (use compact form if available)
	historyMsg := opts.UserMessage
	if opts.HistoryMessage != "" {
		historyMsg = opts.HistoryMessage
	}
	agent.Sessions.AddMessage(opts.SessionKey, "user", historyMsg)

	// 4. Record user prompt for stats
	if al.stats != nil {
		al.stats.RecordPrompt()
	}

	// Capture the finalized system prompt for Mini App inspection
	if len(messages) > 0 {
		al.lastSystemPrompt.Store(messages[0].Content)
		al.promptDirty.Store(false)
	}

	// 5. Run LLM iteration loop (with automatic phase transitions)
	var iteration int
	const maxPhaseTransitions = 10

	for phaseLoop := 0; ; phaseLoop++ {
		// On phase transition: rebuild system prompt with new phase context + nudge
		if phaseLoop > 0 {
			messages = agent.ContextBuilder.BuildMessages(
				agent.Sessions.GetHistory(opts.SessionKey),
				agent.Sessions.GetSummary(opts.SessionKey),
				"", nil, opts.Channel, opts.ChatID,
			)
			messages = append(messages, providers.Message{
				Role: "user",
				Content: fmt.Sprintf(
					"[System] Phase %d is now active. Continue working on the next steps.",
					agent.ContextBuilder.GetCurrentPhase(),
				),
			})
			if len(messages) > 0 {
				al.lastSystemPrompt.Store(messages[0].Content)
			}
		}

		curPlanStatus := preStatus
		if phaseLoop > 0 {
			curPlanStatus = agent.ContextBuilder.GetPlanStatus()
		}

		var err error
		finalContent, iteration, err = al.runLLMIteration(ctx, agent, messages, opts, task, curPlanStatus)
		if err != nil {
			return "", err
		}

		// 5a. Auto-advance plan phases after LLM iteration
		postStatus := agent.ContextBuilder.GetPlanStatus()
		if !agent.ContextBuilder.HasActivePlan() ||
			!(postStatus == "executing" || postStatus == "review" || postStatus == "completed") {
			break
		}

		// Intercept: if AI changed status to executing or review without user approval
		// (from interviewing or review), validate and hold at "review".
		if preStatus == "interviewing" || (preStatus == "review" && postStatus == "executing") {
			if err := agent.ContextBuilder.ValidatePlanStructure(); err != nil {
				_ = agent.ContextBuilder.SetPlanStatus("interviewing")
				logger.WarnCF("agent", "Reverted plan to interviewing: "+err.Error(),
					map[string]any{"agent_id": agent.ID})
				rejectionMsg := "[System] Plan rejected: " + err.Error() + ". Fix and try again."
				agent.Sessions.AddMessage(opts.SessionKey, "user", rejectionMsg)
			} else {
				_ = agent.ContextBuilder.SetPlanStatus("review")
				al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStatePlanReview, "")
				if !constants.IsInternalChannel(opts.Channel) {
					planDisplay := agent.ContextBuilder.FormatPlanDisplay()
					_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel:         opts.Channel,
						ChatID:          opts.ChatID,
						Content:         planDisplay + "\n\nUse /plan start to approve, or continue chatting to refine.",
						SkipPlaceholder: true,
					})
				}
			}
			break
		}

		if postStatus == "executing" && agent.ContextBuilder.GetTotalPhases() == 0 {
			_ = agent.ContextBuilder.SetPlanStatus("interviewing")
			logger.WarnCF("agent", "Reverted plan to interviewing: no phases defined",
				map[string]any{"agent_id": agent.ID})
			break
		}

		if agent.ContextBuilder.IsPlanComplete() {
			total := agent.ContextBuilder.GetTotalPhases()
			_ = agent.ContextBuilder.SetCurrentPhase(total)
			if preStatus != "completed" {
				_ = agent.ContextBuilder.SetPlanStatus("completed")
				al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStatePlanCompleted, "")

				// Deactivate worktree on plan completion
				commitMsg := "plan: " + agent.ContextBuilder.Memory().GetPlanTaskName()
				wtResult, _ := agent.DeactivateWorktree(opts.SessionKey, commitMsg, false)

				if !constants.IsInternalChannel(opts.Channel) {
					msg := "\u2705 Plan completed!"
					if wtResult != nil && wtResult.CommitsAhead > 0 {
						msg += fmt.Sprintf("\nBranch `%s` retained (%d commits). To merge: `git merge %s`",
							wtResult.Branch, wtResult.CommitsAhead, wtResult.Branch)
					}
					_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel:         opts.Channel,
						ChatID:          opts.ChatID,
						Content:         msg,
						SkipPlaceholder: true,
					})
				}
			}
			break
		}

		if agent.ContextBuilder.IsCurrentPhaseComplete() {
			if phaseLoop >= maxPhaseTransitions {
				logger.WarnCF("agent", "Max phase transitions reached, stopping",
					map[string]any{"agent_id": agent.ID, "transitions": phaseLoop})
				break
			}
			prev := agent.ContextBuilder.GetCurrentPhase()
			_ = agent.ContextBuilder.AdvancePhase()
			next := agent.ContextBuilder.GetCurrentPhase()
			if !constants.IsInternalChannel(opts.Channel) {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel:         opts.Channel,
					ChatID:          opts.ChatID,
					Content:         fmt.Sprintf("Phase %d complete. Moving to Phase %d.", prev, next),
					SkipPlaceholder: true,
				})
			}
			al.notifyStateChange()
			continue
		}

		break
	}

	al.notifyStateChange()

	// 5b. Interview staleness detection: compare MEMORY.md size after iteration.
	if agent.ContextBuilder.GetPlanStatus() == "interviewing" {
		postMemoryLen := len(agent.ContextBuilder.ReadMemory())
		if postMemoryLen == preMemoryLen {
			agent.interviewStaleCount++
		} else {
			agent.interviewStaleCount = 0
		}
		agent.interviewMemoryLen = postMemoryLen
	} else {
		// Reset counter when not interviewing.
		agent.interviewStaleCount = 0
	}

	// 5c. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 5d. Store result summary for task completion notification
	if task != nil {
		task.Result = utils.Truncate(finalContent, 280)
	}

	// 6. Save final assistant message to session (deferred write-behind)
	agent.Sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	agent.Sessions.MarkDirty(opts.SessionKey)

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 8. Optional: send response via bus
	if opts.SendResponse {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:         opts.Channel,
			ChatID:          opts.ChatID,
			Content:         finalContent,
			SkipPlaceholder: opts.SystemMessage, // suppress Telegram "Thinking..." for system messages
		})
	}

	// 9. Log response
	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]any{
			"agent_id":     agent.ID,
			"session_key":  opts.SessionKey,
			"iterations":   iteration,
			"final_length": len(finalContent),
		})

	return finalContent, nil
}

// Task reminder constants and helpers.
const (
	taskReminderMaxChars = 500
	blockerMaxChars      = 200
)

func shouldInjectReminder(iteration, interval int) bool {
	if interval <= 0 {
		return false
	}
	return iteration > 1 && iteration%interval == 0
}

func buildTaskReminder(userMessage string, lastBlocker string) providers.Message {
	truncatedTask := utils.Truncate(userMessage, taskReminderMaxChars)

	var content string
	if lastBlocker != "" {
		truncatedBlocker := utils.Truncate(lastBlocker, blockerMaxChars)
		content = fmt.Sprintf(
			"[TASK REMINDER]\nOriginal task:\n---\n%s\n---\nLast blocker:\n---\n%s\n---\nFix the blocker if essential, or find an alternative. If all steps are complete, move on.",
			truncatedTask,
			truncatedBlocker,
		)
	} else {
		content = fmt.Sprintf(
			"[TASK REMINDER]\nOriginal task:\n---\n%s\n---\nIf all steps of the original task are complete, move on. Otherwise, continue with the next step.",
			truncatedTask,
		)
	}

	return providers.Message{
		Role:    "user",
		Content: content,
	}
}

// interviewRejectMessage is the fixed rejection text injected when tool calls
// are blocked during the interview phase.  It is deliberately short to avoid
// wasting tokens, and ends with a purpose reminder to steer the LLM back.
const interviewRejectMessage = "[System] Tool call rejected. " +
	"You are in interview mode — ask the user questions and update MEMORY.md. " +
	"Do not execute, edit, or write project files."

// buildPlanReminder returns a reminder message for plan pre-execution states
// (interviewing / review) to keep the AI focused on the interview workflow
// during tool-call iterations.
func buildPlanReminder(planStatus string) (providers.Message, bool) {
	var content string
	switch planStatus {
	case "interviewing":
		content = "[System] You are interviewing the user to build a plan. " +
			"Ask clarifying questions and save findings to ## Context in memory/MEMORY.md using edit_file. " +
			"When you have enough information, write ## Phase sections with `- [ ]` checkbox steps, and ## Commands section. " +
			"Then change > Status: to review. Do NOT set it to executing."
	case "review":
		content = "[System] The plan is under review. " +
			"Wait for the user to approve or request changes. Do not proceed with execution."
	default:
		return providers.Message{}, false
	}
	return providers.Message{Role: "user", Content: content}, true
}

// buildOrchReminder returns a reminder to use spawn/subagent during plan execution.
// Fires on first iteration and every 3rd iteration to reinforce delegation behavior.
func buildOrchReminder(iteration int) (providers.Message, bool) {
	if iteration != 1 && iteration%3 != 0 {
		return providers.Message{}, false
	}
	content := `[System] ORCHESTRATION mode active. You MUST delegate plan steps to subagents.
Use spawn (non-blocking, returns immediately) or subagent (blocking, waits for result).
Do NOT implement steps inline unless they are a single trivial tool call.

To delegate, call the tool with JSON arguments:
  Tool: spawn  Arguments: {"task": "...", "preset": "scout", "label": "..."}
  Tool: subagent  Arguments: {"task": "...", "label": "..."}

Spawn multiple independent steps in parallel for maximum throughput.`
	return providers.Message{Role: "user", Content: content}, true
}

// cdPrefixPattern matches "cd /some/path && " at the start of a shell command.
// Group 1 captures the target directory path.
var cdPrefixPattern = regexp.MustCompile(`^cd\s+(\S+)\s*&&\s*`)

// optFlagPattern matches option flags like --verbose, -v, --timeout=60, -q.
// Only standalone flags are removed; flags whose value is the next positional
// argument (e.g. "-A 20") are kept because removing them would lose context.
var optFlagPattern = regexp.MustCompile(`\s+--?\w[\w-]*(=\S*)?`)

// extractExecProjectDir extracts the basename of an exec cd target.
// Returns "" if the command has no cd prefix.
func extractExecProjectDir(args map[string]any) string {
	cmd, _ := args["command"].(string)
	if cmd == "" {
		return ""
	}
	m := cdPrefixPattern.FindStringSubmatch(cmd)
	if len(m) < 2 {
		return ""
	}
	cdPath := strings.TrimRight(m[1], "/\\")
	if idx := strings.LastIndex(cdPath, "/"); idx >= 0 {
		return cdPath[idx+1:]
	}
	if idx := strings.LastIndex(cdPath, "\\"); idx >= 0 {
		return cdPath[idx+1:]
	}
	return cdPath
}

// fileParentRelDir returns the parent directory of a file path, relative to
// workspace. Returns "" if the path is not under workspace or has no parent.
func fileParentRelDir(filePath, workspace string) string {
	ws := strings.TrimRight(workspace, "/\\")
	if ws == "" {
		return ""
	}
	rest := strings.TrimPrefix(filePath, ws)
	if rest == filePath {
		return "" // not under workspace
	}
	rest = strings.TrimLeft(rest, "/\\")
	// Remove the filename — keep only the directory part
	if idx := strings.LastIndexAny(rest, "/\\"); idx >= 0 {
		return rest[:idx]
	}
	return "" // file is directly under workspace, no meaningful dir
}

// commonDirPrefix computes the longest common directory prefix of two
// slash-separated paths. Returns "" if there is no common component.
func commonDirPrefix(a, b string) string {
	partsA := strings.Split(a, "/")
	partsB := strings.Split(b, "/")
	n := len(partsA)
	if len(partsB) < n {
		n = len(partsB)
	}
	common := 0
	for i := 0; i < n; i++ {
		if partsA[i] != partsB[i] {
			break
		}
		common = i + 1
	}
	if common == 0 {
		return ""
	}
	return strings.Join(partsA[:common], "/")
}

// displayProjectDir returns the project directory name for status display.
// Prefers the authoritative exec-based projectDir; falls back to the
// basename of the file-based common directory.
func displayProjectDir(task *activeTask) string {
	if task.projectDir != "" {
		return task.projectDir
	}
	if task.fileCommonDir != "" {
		dir := task.fileCommonDir
		if idx := strings.LastIndex(dir, "/"); idx >= 0 {
			return dir[idx+1:]
		}
		return dir
	}
	return ""
}

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

// buildArgsSnippet produces a human-friendly snippet for the tool log.
// For exec: extracts the command and strips the leading "cd <workspace> && ".
// For file tools: extracts the path and strips the workspace prefix.
// Falls back to raw JSON truncation.
func buildArgsSnippet(toolName string, args map[string]any, workspace string) string {
	switch toolName {
	case "exec":
		cmd, _ := args["command"].(string)
		if cmd == "" {
			break
		}
		cmd = cdPrefixPattern.ReplaceAllString(cmd, "")
		cmd = optFlagPattern.ReplaceAllString(cmd, "")
		return utils.Truncate(cmd, 80)

	case "read_file", "write_file", "edit_file", "append_file", "list_dir":
		path, _ := args["path"].(string)
		if path == "" {
			break
		}
		if workspace != "" {
			path = strings.TrimPrefix(path, workspace)
			path = strings.TrimPrefix(path, "/")
		}
		// Prioritize filename: if path is too long, show "…/filename"
		const maxPath = 60
		if runes := []rune(path); len(runes) > maxPath {
			// Find last slash to extract filename
			if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
				filename := path[lastSlash:]                     // includes "/"
				dirBudget := maxPath - len([]rune(filename)) - 1 // 1 for "…"
				if dirBudget > 0 {
					dir := []rune(path[:lastSlash])
					if len(dir) > dirBudget {
						dir = dir[:dirBudget]
					}
					path = string(dir) + "\u2026" + filename
				} else {
					path = "\u2026" + filename
				}
			} else {
				path = utils.Truncate(path, maxPath)
			}
		}
		return path
	}

	// Default: raw JSON truncated
	argsJSON, _ := json.Marshal(args)
	return utils.Truncate(string(argsJSON), 80)
}

// maxEntryLineWidth is the max rune count for a single-line log entry.
// Telegram chat bubbles on mobile are roughly 40-45 chars wide.
const maxEntryLineWidth = 42

// isFileToolEntry returns true if the entry name contains a file-operation tool.
func isFileToolEntry(name string) bool {
	for _, t := range []string{"read_file", "write_file", "edit_file", "append_file", "list_dir"} {
		if strings.Contains(name, t) {
			return true
		}
	}
	return false
}

// formatCompactEntry formats a finished tool log entry as a fixed single line.
// The result marker (✓/✗) is always shown at the end regardless of truncation.
// File tools omit duration (always near-instant); paths truncate from the
// start so the filename is always visible.
func formatCompactEntry(entry toolLogEntry) string {
	result := entry.Result
	if result == "" {
		result = "\u23F3" // ⏳
	}

	// File tools: strip duration, keep only marker (✓/✗/⏳)
	isFile := isFileToolEntry(entry.Name)
	if isFile {
		if r := []rune(result); len(r) > 0 {
			result = string(r[0:1]) // just the symbol
		}
	}

	// Budget for ArgsSnip: total - name - " " - " " - result
	nameLen := utf8.RuneCountInString(entry.Name)
	resultLen := utf8.RuneCountInString(result)
	argsBudget := maxEntryLineWidth - nameLen - 1 - 1 - resultLen

	args := entry.ArgsSnip
	if args != "" && argsBudget > 3 {
		argsRunes := []rune(args)
		if len(argsRunes) > argsBudget {
			// Paths: truncate from the start, keeping the filename visible
			if strings.Contains(args, "/") {
				args = "\u2026" + string(argsRunes[len(argsRunes)-argsBudget+1:])
			} else {
				args = string(argsRunes[:argsBudget-1]) + "\u2026"
			}
		}
		var sb strings.Builder
		sb.Grow(len(entry.Name) + 1 + len(args) + 1 + len(result))
		sb.WriteString(entry.Name)
		sb.WriteByte(' ')
		sb.WriteString(args)
		sb.WriteByte(' ')
		sb.WriteString(result)
		return sb.String()
	}

	// No room for args or args empty
	var sb strings.Builder
	sb.Grow(len(entry.Name) + 1 + len(result))
	sb.WriteString(entry.Name)
	sb.WriteByte(' ')
	sb.WriteString(result)
	return sb.String()
}

// formatLatestEntry formats the latest entry command without its result marker.
// Since the result goes on the next line, the full width is available for the command.
func formatLatestEntry(entry toolLogEntry) string {
	nameLen := utf8.RuneCountInString(entry.Name)
	argsBudget := maxEntryLineWidth - nameLen - 1 // name + space + args (no result)

	args := entry.ArgsSnip
	if args != "" && argsBudget > 3 {
		argsRunes := []rune(args)
		if len(argsRunes) > argsBudget {
			if strings.Contains(args, "/") {
				args = "\u2026" + string(argsRunes[len(argsRunes)-argsBudget+1:])
			} else {
				args = string(argsRunes[:argsBudget-1]) + "\u2026"
			}
		}
		var sb strings.Builder
		sb.Grow(len(entry.Name) + 1 + len(args))
		sb.WriteString(entry.Name)
		sb.WriteByte(' ')
		sb.WriteString(args)
		return sb.String()
	}
	return entry.Name
}

// compressRepeats reduces runs of 3+ identical non-alphanumeric, non-space
// characters to just 2. e.g. "======" → "==", "---" → "--".
func compressRepeats(s string) string {
	runes := []rune(s)
	if len(runes) < 3 {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	i := 0
	for i < len(runes) {
		r := runes[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			j := i + 1
			for j < len(runes) && runes[j] == r {
				j++
			}
			if j-i >= 3 {
				sb.WriteRune(r)
				sb.WriteRune(r)
				i = j
				continue
			}
		}
		sb.WriteRune(r)
		i++
	}
	return sb.String()
}

// Display layout constants.
const (
	displayPastEntries    = 4 // number of compact 1-line past entries
	displayErrorLines     = 5 // content lines inside the error code block
	statusSeparator       = "\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n"
	streamingDisplayLines = 17 // line count matching buildRichStatus output
)

// buildRichStatus builds a fixed-height terminal-like status display.
//
// Layout (always the same number of lines):
//
//	🔄 Task in progress (N/M)          header
//	📁 workspace-path                   header
//	━━━━━━━━━━                          separator
//	[N] compact-past-1 ✓ Xs            past (1 line each)
//	[N] compact-past-2 ✗ Xs            past
//	[N] compact-past-3 ✓ Xs            past
//	[N] compact-past-4 ✓ Xs            past
//	[N] latest-command                  latest (no result, wider args)
//	  ⏳                                latest result
//	                                    reserved
//	```                                 error fence
//	err-line / placeholder              error body (5 lines)
//	```                                 error fence
//	↩️ Reply to intervene               footer (background only)
func buildRichStatus(task *activeTask, isBackground bool, workspace string) string {
	task.mu.Lock()
	defer task.mu.Unlock()

	var sb strings.Builder

	// --- Header ---
	sb.WriteString("\U0001F504 Task in progress (")
	sb.WriteString(strconv.Itoa(task.Iteration))
	sb.WriteByte('/')
	sb.WriteString(strconv.Itoa(task.MaxIter))
	sb.WriteString(")\n")
	// Project directory: exec cd (authoritative) → file LCP → workspace basename
	sb.WriteString("\U0001F4C1 ")
	if dir := displayProjectDir(task); dir != "" {
		sb.WriteString(dir)
	} else if workspace != "" {
		project := strings.TrimRight(workspace, "/\\")
		if idx := strings.LastIndex(project, "/"); idx >= 0 {
			project = project[idx+1:]
		} else if idx := strings.LastIndex(project, "\\"); idx >= 0 {
			project = project[idx+1:]
		}
		sb.WriteString(project)
	}
	sb.WriteByte('\n')
	sb.WriteString(statusSeparator)

	// --- Task entries (displayPastEntries + 2 lines for latest) ---
	entries := task.toolLog
	if len(entries) > maxToolLogEntries {
		entries = entries[len(entries)-maxToolLogEntries:]
	}

	var pastEntries []toolLogEntry
	var latest *toolLogEntry
	if len(entries) > 0 {
		latest = &entries[len(entries)-1]
		if len(entries) > 1 {
			start := len(entries) - 1 - displayPastEntries
			if start < 0 {
				start = 0
			}
			pastEntries = entries[start : len(entries)-1]
		}
	}

	// Past entries: exactly displayPastEntries lines (pad if fewer)
	for i := 0; i < displayPastEntries; i++ {
		if i < len(pastEntries) {
			sb.WriteString(formatCompactEntry(pastEntries[i]))
		} else {
			sb.WriteString("\u2800")
		}
		sb.WriteByte('\n')
	}

	// Latest entry: command on one line, result on next
	if latest != nil {
		sb.WriteString(formatLatestEntry(*latest))
		sb.WriteByte('\n')
		sb.WriteString("  ")
		if latest.Result != "" {
			sb.WriteString(latest.Result)
		} else {
			sb.WriteString("\u23F3")
		}
		sb.WriteByte('\n')
	} else {
		sb.WriteString("\u23F3 waiting...\n")
		sb.WriteString("\u2800\n")
	}

	// Reserved (1 line)
	sb.WriteString("\u2800\n")

	// --- Error region (code fence, no separator) ---
	sb.WriteString("```\n")

	errEntry := task.lastError
	if errEntry != nil {
		sb.WriteString("\u274C ")
		sb.WriteString(formatCompactEntry(*errEntry))
		sb.WriteByte('\n')

		var detailLines []string
		if errEntry.ErrDetail != "" {
			detailLines = strings.Split(errEntry.ErrDetail, "\n")
		}
		for i := 0; i < displayErrorLines-1; i++ {
			if i < len(detailLines) {
				line := compressRepeats(detailLines[i])
				if runes := []rune(line); len(runes) > maxEntryLineWidth {
					line = string(runes[:maxEntryLineWidth-1]) + "\u2026"
				}
				sb.WriteString(line)
			} else {
				sb.WriteString("\u2800")
			}
			sb.WriteByte('\n')
		}
	} else {
		sb.WriteString("\u2714 No errors\n")
		for i := 0; i < displayErrorLines-1; i++ {
			sb.WriteString("\u2800\n")
		}
	}

	sb.WriteString("```\n")

	if isBackground {
		sb.WriteString("\u21A9\uFE0F Reply to intervene")
	}
	return sb.String()
}

func (al *AgentLoop) handleReasoning(ctx context.Context, reasoningContent, channelName, channelID string) {
	if reasoningContent == "" || channelName == "" || channelID == "" {
		return
	}

	// Check context cancellation before attempting to publish,
	// since PublishOutbound's select may race between send and ctx.Done().
	if ctx.Err() != nil {
		return
	}

	// Use a short timeout so the goroutine does not block indefinitely when
	// the outbound bus is full.  Reasoning output is best-effort; dropping it
	// is acceptable to avoid goroutine accumulation.
	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: channelName,
		ChatID:  channelID,
		Content: reasoningContent,
	}); err != nil {
		// Treat context.DeadlineExceeded / context.Canceled as expected
		// (bus full under load, or parent canceled).  Check the error
		// itself rather than ctx.Err(), because pubCtx may time out
		// (5 s) while the parent ctx is still active.
		// Also treat ErrBusClosed as expected — it occurs during normal
		// shutdown when the bus is closed before all goroutines finish.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		}
	}
}

// streamingReasoningLines is the number of lines reserved for reasoning
// in the streaming display.  The remaining lines go to content.
const streamingReasoningLines = 6

// buildStreamingDisplay builds a fixed-height status bubble for streaming.
//
// Layout when reasoning is active (reasoning only or both):
//
//	🧠 Thinking...
//	━━━━━━━━━━
//	<reasoning tail — streamingReasoningLines lines>
//	━━━━━━━━━━
//	<content tail — remaining lines>  (or blank if content is empty)
//	█
//
// Layout when no reasoning (content only):
//
//	<content tail — streamingDisplayLines lines>
//	█
func buildStreamingDisplay(content, reasoning string) string {
	if reasoning == "" {
		// No reasoning — full window for content.
		return utils.TailPad(content, streamingDisplayLines, maxEntryLineWidth) + " \u2589"
	}

	var sb strings.Builder

	// Header
	if content == "" {
		sb.WriteString("\U0001f9e0 Thinking...\n")
	} else {
		sb.WriteString("\U0001f9e0 Thought, now responding...\n")
	}
	sb.WriteString(statusSeparator)

	// Reasoning window
	headerLines := 2 // header + separator
	footerLines := 1 // separator before content
	contentLines := streamingDisplayLines - headerLines - footerLines - streamingReasoningLines
	if contentLines < 3 {
		contentLines = 3
	}
	rLines := streamingDisplayLines - headerLines - footerLines - contentLines

	sb.WriteString(utils.TailPad(reasoning, rLines, maxEntryLineWidth))
	sb.WriteByte('\n')
	sb.WriteString(statusSeparator)

	// Content window (may be blank padding if content hasn't started)
	sb.WriteString(utils.TailPad(content, contentLines, maxEntryLineWidth))
	sb.WriteString(" \u2589")

	return sb.String()
}

// runLLMIteration executes the LLM call loop with tool handling.
// consumeStreamWithRepetitionDetection reads StreamEvents from ch, accumulates
// content and tool calls, and runs repetition detection every checkInterval runes.
// If repetition is detected, cancelFn is called to abort the HTTP request and
// the function returns the partial response with detected=true.
func consumeStreamWithRepetitionDetection(
	ch <-chan protocoltypes.StreamEvent,
	cancelFn context.CancelFunc,
	checkInterval int,
	onChunk func(content, reasoning string),
) (*providers.LLMResponse, bool, error) {
	var content strings.Builder
	var reasoning strings.Builder
	var toolCalls []streamToolCallAcc
	var finishReason string
	var usage *providers.UsageInfo
	runesSinceLastCheck := 0

	for ev := range ch {
		if ev.Err != nil {
			return nil, false, ev.Err
		}
		updated := false
		if ev.ContentDelta != "" {
			content.WriteString(ev.ContentDelta)
			runesSinceLastCheck += utf8.RuneCountInString(ev.ContentDelta)
			updated = true
		}
		if ev.ReasoningDelta != "" {
			reasoning.WriteString(ev.ReasoningDelta)
			updated = true
		}
		if updated && onChunk != nil {
			onChunk(content.String(), reasoning.String())
		}
		if ev.FinishReason != "" {
			finishReason = ev.FinishReason
		}
		if ev.Usage != nil {
			usage = ev.Usage
		}
		for _, tc := range ev.ToolCallDeltas {
			for len(toolCalls) <= tc.Index {
				toolCalls = append(toolCalls, streamToolCallAcc{})
			}
			if tc.ID != "" {
				toolCalls[tc.Index].id = tc.ID
			}
			if tc.Name != "" {
				toolCalls[tc.Index].name = tc.Name
			}
			toolCalls[tc.Index].args.WriteString(tc.ArgumentsDelta)
		}

		// Run repetition detection periodically on accumulated content.
		if runesSinceLastCheck >= checkInterval && content.Len() > 2000 {
			runesSinceLastCheck = 0
			if utils.DetectRepetitionLoop(content.String()) {
				cancelFn()
				// Drain remaining events so the producer goroutine can exit.
				for range ch {
				}
				resp := buildAccumulatedResponse(content.String(), reasoning.String(), toolCalls, finishReason, usage)
				return resp, true, nil
			}
		}
	}

	resp := buildAccumulatedResponse(content.String(), reasoning.String(), toolCalls, finishReason, usage)
	return resp, false, nil
}

// streamToolCallAcc accumulates streamed tool call fragments.
type streamToolCallAcc struct {
	id   string
	name string
	args strings.Builder
}

// buildAccumulatedResponse constructs an LLMResponse from accumulated stream data.
func buildAccumulatedResponse(
	content, reasoning string,
	toolCalls []streamToolCallAcc,
	finishReason string,
	usage *providers.UsageInfo,
) *providers.LLMResponse {
	resp := &providers.LLMResponse{
		Content:      content,
		Reasoning:    reasoning,
		FinishReason: finishReason,
		Usage:        usage,
	}
	for _, tc := range toolCalls {
		arguments := make(map[string]any)
		argStr := tc.args.String()
		if argStr != "" {
			if err := json.Unmarshal([]byte(argStr), &arguments); err != nil {
				arguments["raw"] = argStr
			}
		}
		resp.ToolCalls = append(resp.ToolCalls, providers.ToolCall{
			ID:        tc.id,
			Name:      tc.name,
			Arguments: arguments,
		})
	}
	return resp
}

func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
	task *activeTask,
	planSnapshot string,
) (string, int, error) {
	iteration := 0
	var finalContent string
	lastReminderIdx := -1
	planMarkNudged := false // true after we've already nudged once for [x] marking

	maxIter := agent.MaxIterations

	// Snapshot unchecked step count before tool loop so we can detect progress.
	preUnchecked := -1 // -1 = not tracking
	if planSnapshot == "executing" {
		preUnchecked = strings.Count(agent.ContextBuilder.ReadMemory(), "- [ ]")
	}

	// Determine if this is a background task (cron, heartbeat, etc.)
	isBackground := opts.TaskID != ""

	for iteration < maxIter {
		iteration++

		// Update active task iteration
		if task != nil {
			task.mu.Lock()
			task.Iteration = iteration
			task.mu.Unlock()
		}

		// Check for user intervention via interrupt channel
		if task != nil {
			select {
			case msg := <-task.interrupt:
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: "[User Intervention] " + msg,
				})
				logger.InfoCF("agent", "User intervention injected",
					map[string]any{"agent_id": agent.ID, "iteration": iteration})
			default:
			}
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       maxIter,
			})

		// Build tool definitions
		providerToolDefs := agent.Tools.ToProviderDefs()

		// Interview mode: strip tool definitions the LLM must not use,
		// reducing token cost and preventing wasted reject-retry cycles.
		if isPlanPreExecution(planSnapshot) {
			providerToolDefs = filterInterviewTools(providerToolDefs)
		}

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          agent.ID,
				"iteration":         iteration,
				"model":             agent.Model,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        agent.MaxTokens,
				"temperature":       agent.Temperature,
				"system_prompt_len": len(messages[0].Content),
			})

		// Log full messages (detailed)
		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// Call LLM with fallback chain if candidates are configured.
		var response *providers.LLMResponse
		var err error

		// Build onChunk callback for streaming preview.
		// Instead of a fixed-interval throttle, use a Go channel with
		// latest-value semantics: a consumer goroutine publishes status
		// updates as fast as the bus → manager → channel pipeline allows.
		// Backpressure is provided naturally by the per-channel rate limiter
		// (e.g. 20 msg/s for Telegram's SendDraft, 1 msg/s for Discord's EditMessage).
		type streamUpdate struct{ accumulated, reasoning string }
		var onChunk func(string, string)
		var streamCh chan streamUpdate
		var streamDone chan struct{}
		if !constants.IsInternalChannel(opts.Channel) {
			streamCh = make(chan streamUpdate, 1)
			streamDone = make(chan struct{})
			go func() {
				defer close(streamDone)
				for up := range streamCh {
					display := buildStreamingDisplay(up.accumulated, up.reasoning)
					outMsg := bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: display,
					}
					// For background tasks, publish streaming preview as
					// IsTaskStatus so it shares the same bubble as task
					// progress/completion (avoids a second bubble).
					if opts.Background && opts.TaskID != "" {
						outMsg.IsTaskStatus = true
						outMsg.TaskID = opts.TaskID
					} else {
						outMsg.IsStatus = true
					}
					_ = al.bus.PublishOutbound(ctx, outMsg)
				}
			}()
			onChunk = func(accumulated, reasoning string) {
				if task != nil {
					task.streamedChunks = true
				}
				up := streamUpdate{accumulated, reasoning}
				// Non-blocking latest-value send: if the consumer hasn't
				// drained the previous update, replace it with the latest.
				select {
				case streamCh <- up:
				default:
					// Channel full — drain stale value, then send latest.
					select {
					case <-streamCh:
					default:
					}
					select {
					case streamCh <- up:
					default:
					}
				}
			}
		}

		// doCall invokes a single LLM provider, using streaming with
		// early repetition detection when the provider supports it.
		opts_ := map[string]any{
			"max_tokens":       agent.MaxTokens,
			"temperature":      agent.Temperature,
			"prompt_cache_key": agent.ID,
		}
		doCall := func(ctx context.Context, p providers.LLMProvider, model string) (*providers.LLMResponse, error) {
			if sp, ok := p.(providers.StreamingProvider); ok && sp.CanStream() {
				streamCtx, streamCancel := context.WithCancel(ctx)
				defer streamCancel()
				ch, sErr := sp.ChatStream(streamCtx, messages, providerToolDefs, model, opts_)
				if sErr != nil {
					return nil, sErr
				}
				resp, repetition, sErr := consumeStreamWithRepetitionDetection(ch, streamCancel, 1000, onChunk)
				if sErr != nil {
					return nil, sErr
				}
				if repetition {
					resp.FinishReason = "repetition_detected"
				}
				return resp, nil
			}
			return p.Chat(ctx, messages, providerToolDefs, model, opts_)
		}

		callLLM := func() (*providers.LLMResponse, error) {
			// Plan model switching: use plan model during interviewing/review phases
			candidates := agent.Candidates
			primaryModel := agent.Model
			if isPlanPreExecution(planSnapshot) && agent.PlanModel != "" {
				candidates = agent.PlanCandidates
				primaryModel = agent.PlanModel
				logger.InfoCF("agent", "Using plan model",
					map[string]any{"agent_id": agent.ID, "plan_model": agent.PlanModel})
			}

			if len(candidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(ctx, candidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						p := al.resolveProvider(provider, model, agent.Provider)
						return doCall(ctx, p, model)
					},
				)
				if fbErr != nil {
					return nil, fbErr
				}
				if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
					logger.InfoCF("agent", fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
						fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
						map[string]any{"agent_id": agent.ID, "iteration": iteration})
				}
				return fbResult.Response, nil
			}
			if len(candidates) > 0 {
				c := candidates[0]
				p := al.resolveProvider(c.Provider, c.Model, agent.Provider)
				return doCall(ctx, p, c.Model)
			}
			return doCall(ctx, agent.Provider, primaryModel)
		}

		// Report waiting state to canvas before each LLM call.
		al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStateWaiting, "")

		// Retry loop for context/token errors
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM()
			if err == nil {
				break
			}

			errMsg := strings.ToLower(err.Error())

			// Check if this is a network/HTTP timeout — not a context window error.
			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			// Detect real context window / token limit errors, excluding network timeouts.
			isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
				strings.Contains(errMsg, "context window") ||
				strings.Contains(errMsg, "maximum context length") ||
				strings.Contains(errMsg, "token limit") ||
				strings.Contains(errMsg, "too many tokens") ||
				strings.Contains(errMsg, "max_tokens") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "prompt is too long") ||
				strings.Contains(errMsg, "request too large"))

			if isTimeoutError && retry < maxRetries {
				backoff := time.Duration(retry+1) * 5 * time.Second
				logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
					"error":   err.Error(),
					"retry":   retry,
					"backoff": backoff.String(),
				})
				time.Sleep(backoff)
				continue
			}

			if isContextError && retry < maxRetries {
				logger.WarnCF("agent", "Context window error detected, attempting compression", map[string]any{
					"error": err.Error(),
					"retry": retry,
				})

				if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
					_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: "Context window exceeded. Compressing history and retrying...",
					})
				}

				al.forceCompression(agent, opts.SessionKey)
				newHistory := agent.Sessions.GetHistory(opts.SessionKey)
				newSummary := agent.Sessions.GetSummary(opts.SessionKey)
				messages = agent.ContextBuilder.BuildMessages(
					newHistory, newSummary, "",
					nil, opts.Channel, opts.ChatID,
				)
				continue
			}
			break
		}

		// Streaming finished — close the stream goroutine so it flushes
		// the last update and exits cleanly before we process the response.
		if streamDone != nil {
			// onChunk is captured by doCall closures; nil it to avoid
			// writes after the channel is closed during retries.
			onChunk = nil
			close(streamCh)
			<-streamDone
			streamDone = nil
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  agent.ID,
					"iteration": iteration,
					"error":     err.Error(),
				})
			return "", iteration, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		// Record token usage
		if response.Usage != nil && al.stats != nil {
			al.stats.RecordUsage(
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens,
			)
		}

		// Handle reasoning output (best-effort, non-blocking)
		go al.handleReasoning(ctx, response.Reasoning, opts.Channel, al.targetReasoningChannelID(opts.Channel))

		logger.DebugCF("agent", "LLM response",
			map[string]any{
				"agent_id":       agent.ID,
				"iteration":      iteration,
				"content_chars":  len(response.Content),
				"tool_calls":     len(response.ToolCalls),
				"reasoning":      response.Reasoning,
				"target_channel": al.targetReasoningChannelID(opts.Channel),
				"channel":        opts.Channel,
			})

		// Detect repetition loop on raw text (before stripping think
		// blocks so loops inside <think> are caught).  Skip when the
		// provider already returned native tool calls.
		// Streaming providers may have already flagged repetition via
		// FinishReason="repetition_detected" — honor that too.
		if response.FinishReason == "repetition_detected" ||
			(len(response.ToolCalls) == 0 && utils.DetectRepetitionLoop(response.Content)) {
			logger.WarnCF("agent", "Repetition loop detected in LLM response, retrying",
				map[string]any{
					"agent_id":       agent.ID,
					"iteration":      iteration,
					"finish_reason":  response.FinishReason,
					"content_length": len(response.Content),
				})

			// Retry once: inject nudge message and re-call
			savedMsgs := messages
			messages = append(append([]providers.Message(nil), messages...),
				providers.Message{
					Role:    "user",
					Content: "[System] Your previous response contained degenerate repetition and was discarded. Please respond normally without repeating yourself.",
				})
			response, err = callLLM()
			messages = savedMsgs // restore original messages

			if err != nil {
				return "", iteration, fmt.Errorf("LLM retry after repetition failed: %w", err)
			}

			// Re-check on raw text; if still repeating give up
			if utils.DetectRepetitionLoop(response.Content) {
				logger.ErrorCF("agent", "Repetition persists after retry, returning empty",
					map[string]any{"agent_id": agent.ID})
				response.Content = ""
			}
		}

		// Strip think blocks before extracting XML tool calls so
		// extraction operates on clean content.
		response.Content = utils.StripThinkBlocks(response.Content)

		// Recover XML tool calls emitted as plain text by some providers.
		if len(response.ToolCalls) == 0 {
			if xmlCalls := providers.ExtractXMLToolCalls(response.Content); len(xmlCalls) > 0 {
				response.ToolCalls = xmlCalls
			}
		}
		response.Content = providers.StripXMLToolCalls(response.Content)

		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
			// Plan continuation: if unchecked steps remain, nudge the LLM to
			// either mark completed steps or continue working on them.
			// This fires for both foreground and background plan execution,
			// ensuring the loop doesn't exit prematurely after marking a step.
			curUnchecked := 0
			if preUnchecked > 0 {
				curUnchecked = strings.Count(agent.ContextBuilder.ReadMemory(), "- [ ]")
			}
			if curUnchecked > 0 && !planMarkNudged &&
				planSnapshot == "executing" {
				planMarkNudged = true
				messages = append(messages, providers.Message{
					Role:    "assistant",
					Content: response.Content,
				})
				var nudgeMsg string
				if curUnchecked == preUnchecked {
					nudgeMsg = fmt.Sprintf("[System] %d unchecked steps remain in MEMORY.md and "+
						"none were marked [x] during this session. "+
						"If you completed any steps, use edit_file to mark them [x] now. "+
						"If steps are still in progress, continue working on them.", curUnchecked)
				} else {
					nudgeMsg = fmt.Sprintf("[System] Progress recorded. %d unchecked steps remain. "+
						"Continue working on the next step.", curUnchecked)
				}
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: nudgeMsg,
				})
				logger.InfoCF("agent", "Nudging plan execution: continue plan steps",
					map[string]any{"agent_id": agent.ID, "iteration": iteration, "unchecked": curUnchecked})
				continue
			}

			finalContent = response.Content
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		// --- Interview mode: reject disallowed tool calls before they
		// enter messages or session history.  Rejected calls are stripped
		// from normalizedToolCalls so they never reach the assistant
		// message, the tool-result list, or the session store.
		// A single compact rejection message is injected instead.
		var interviewRejected []string
		if isPlanPreExecution(planSnapshot) {
			allowed := normalizedToolCalls[:0] // reuse backing array
			for _, tc := range normalizedToolCalls {
				if isToolAllowedDuringInterview(tc.Name, tc.Arguments) {
					allowed = append(allowed, tc)
				} else {
					interviewRejected = append(interviewRejected, tc.Name)
				}
			}
			normalizedToolCalls = allowed
			if len(interviewRejected) > 0 {
				logger.InfoCF("agent", "Interview mode: rejected tool calls",
					map[string]any{
						"agent_id": agent.ID,
						"rejected": interviewRejected,
					})
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: interviewRejectMessage,
				})
			}
			// If all tool calls were rejected, skip to next iteration.
			if len(normalizedToolCalls) == 0 {
				continue
			}
		}

		// Log tool calls
		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		// Publish rich status update
		if !constants.IsInternalChannel(opts.Channel) && task != nil {
			// Add pending entries to tool log for the current tool calls
			task.mu.Lock()
			for _, tc := range normalizedToolCalls {
				task.toolLog = append(task.toolLog, toolLogEntry{
					Name:     fmt.Sprintf("[%d] %s", iteration, tc.Name),
					ArgsSnip: buildArgsSnippet(tc.Name, tc.Arguments, agent.Workspace),
					Result:   "\u23F3",
				})
				// Detect project directory
				if task.projectDir == "" && tc.Name == "exec" {
					task.projectDir = extractExecProjectDir(tc.Arguments)
				}
				switch tc.Name {
				case "read_file", "write_file", "edit_file", "append_file", "list_dir":
					if p, _ := tc.Arguments["path"].(string); p != "" {
						if rel := fileParentRelDir(p, agent.Workspace); rel != "" {
							if task.fileCommonDir == "" {
								task.fileCommonDir = rel
							} else {
								task.fileCommonDir = commonDirPrefix(task.fileCommonDir, rel)
							}
						}
					}
				}
			}
			task.mu.Unlock()

			statusContent := buildRichStatus(task, isBackground, agent.Workspace)
			if isBackground {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel:      opts.Channel,
					ChatID:       opts.ChatID,
					Content:      statusContent,
					IsTaskStatus: true,
					TaskID:       opts.TaskID,
				})
			} else {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel:  opts.Channel,
					ChatID:   opts.ChatID,
					Content:  statusContent,
					IsStatus: true,
				})
			}
		}

		// Record session activity for heartbeat/plan coordination
		for _, tc := range normalizedToolCalls {
			var detectedDir string
			if tc.Name == "exec" {
				detectedDir = extractExecProjectDir(tc.Arguments)
			}
			if detectedDir == "" {
				switch tc.Name {
				case "read_file", "write_file", "edit_file", "append_file", "list_dir":
					if p, _ := tc.Arguments["path"].(string); p != "" {
						detectedDir = fileParentRelDir(p, agent.Workspace)
					}
				}
			}
			if detectedDir != "" {
				meta := &TouchMeta{
					ProjectPath: agent.ContextBuilder.GetPlanWorkDir(),
					Purpose:     utils.Truncate(opts.UserMessage, 80),
					Branch:      agent.GetWorktreeBranch(opts.SessionKey),
				}
				if meta.ProjectPath == "" {
					meta.ProjectPath = agent.Workspace
				}
				al.sessions.Touch(opts.SessionKey, opts.Channel, opts.ChatID, detectedDir, meta)
			}
		}

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			// Copy ExtraContent to ensure thought_signature is persisted for Gemini 3
			extraContent := tc.ExtraContent
			thoughtSignature := ""
			if tc.Function != nil {
				thoughtSignature = tc.Function.ThoughtSignature
			}

			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Name: tc.Name,
				Function: &providers.FunctionCall{
					Name:             tc.Name,
					Arguments:        string(argumentsJSON),
					ThoughtSignature: thoughtSignature,
				},
				ExtraContent:     extraContent,
				ThoughtSignature: thoughtSignature,
			})
		}
		messages = append(messages, assistantMsg)

		// Save assistant message with tool calls to session
		agent.Sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls
		var lastBlocker string
		for tcIdx, tc := range normalizedToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]any{
					"agent_id":  agent.ID,
					"tool":      tc.Name,
					"iteration": iteration,
				})

			// Heartbeat lazy worktree: create worktree on first write-tool call
			if opts.Background && isWriteTool(tc.Name) && !agent.IsInWorktree(opts.SessionKey) {
				taskName := "heartbeat-" + time.Now().Format("20060102")
				hbDir := agent.ContextBuilder.GetPlanWorkDir()
				if wt, err := agent.ActivateWorktree(opts.SessionKey, taskName, hbDir); err == nil {
					logger.InfoCF("agent", "Heartbeat worktree created", map[string]any{"branch": wt.Branch})
				}
			}

			// Create async callback for tools that implement AsyncTool.
			// The callback publishes a system inbound message so processSystemMessage
			// injects the result into the conductor's session history. The conductor
			// sees it on its next turn and decides whether to notify the user.
			toolName := tc.Name // capture for goroutine
			asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
				content := result.ForLLM
				if content == "" {
					content = result.ForUser
				}
				if content == "" {
					return
				}

				logger.InfoCF("agent", "Async tool completed, publishing to conductor",
					map[string]any{
						"tool":        toolName,
						"content_len": len(content),
						"is_error":    result.IsError,
					})

				pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pubCancel()
				_ = al.bus.PublishInbound(pubCtx, bus.InboundMessage{
					Channel:  "system",
					SenderID: fmt.Sprintf("async:%s", toolName),
					ChatID:   fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID),
					Content:  fmt.Sprintf("Async tool '%s' completed.\n\nResult:\n%s", toolName, content),
				})
			}

			// Report toolcall state to canvas.
			al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStateToolCall, tc.Name)

			toolStart := time.Now()
			toolCtx := ctx
			if wt := agent.GetWorktree(opts.SessionKey); wt != nil {
				toolCtx = tools.WithWorkspaceOverride(toolCtx, wt.Path)
				toolCtx = tools.WithWorktreeInfo(toolCtx, wt)
			}
			toolResult := agent.Tools.ExecuteWithContext(
				toolCtx,
				tc.Name,
				tc.Arguments,
				opts.Channel,
				opts.ChatID,
				asyncCallback,
			)
			toolDuration := time.Since(toolStart)

			// Update tool log entry with result
			if task != nil {
				task.mu.Lock()
				// Find the matching pending entry (added earlier in this iteration)
				logIdx := len(task.toolLog) - len(normalizedToolCalls) + tcIdx
				if logIdx >= 0 && logIdx < len(task.toolLog) {
					if toolResult.IsError || toolResult.Err != nil {
						task.toolLog[logIdx].Result = fmt.Sprintf("\u2717 %.1fs", toolDuration.Seconds())
						// Extract error detail for block display
						if toolResult.Err != nil {
							task.toolLog[logIdx].ErrDetail = utils.Truncate(toolResult.Err.Error(), 300)
						} else if toolResult.ForLLM != "" {
							// exec returns IsError with exit info in ForLLM, not Err
							// Show last few lines (stderr / exit code)
							lines := strings.Split(strings.TrimSpace(toolResult.ForLLM), "\n")
							start := len(lines) - 3
							if start < 0 {
								start = 0
							}
							task.toolLog[logIdx].ErrDetail = utils.Truncate(
								strings.Join(lines[start:], "\n"), 300)
						}
						// Sticky error: remember most recent error for persistent display
						entry := task.toolLog[logIdx]
						task.lastError = &entry
					} else {
						task.toolLog[logIdx].Result = fmt.Sprintf("\u2713 %.1fs", toolDuration.Seconds())
					}
				}
				task.mu.Unlock()
			}

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: toolResult.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]any{
						"tool":        tc.Name,
						"content_len": len(toolResult.ForUser),
					})
			}

			// If tool returned media refs, publish them as outbound media
			if len(toolResult.Media) > 0 && opts.SendResponse {
				parts := make([]bus.MediaPart, 0, len(toolResult.Media))
				for _, ref := range toolResult.Media {
					part := bus.MediaPart{Ref: ref}
					// Populate metadata from MediaStore when available
					if al.mediaStore != nil {
						if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
							part.Filename = meta.Filename
							part.ContentType = meta.ContentType
							part.Type = inferMediaType(meta.Filename, meta.ContentType)
						}
					}
					parts = append(parts, part)
				}
				al.bus.PublishOutboundMedia(ctx, bus.OutboundMediaMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Parts:   parts,
				})
			}

			// Determine content for LLM based on tool result
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			// Track blockers for task reminder
			if toolResult.IsError || toolResult.Err != nil {
				lastBlocker = contentForLLM
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
		}

		// Trim tool log sliding window to prevent unbounded growth
		if task != nil {
			task.mu.Lock()
			if len(task.toolLog) > maxToolLogEntries {
				task.toolLog = task.toolLog[len(task.toolLog)-maxToolLogEntries:]
			}
			task.mu.Unlock()
		}

		// Inject ephemeral task reminder to prevent focus drift.
		// Remove previous reminder and re-append at the tail so it stays
		// close to the LLM's attention window.
		if shouldInjectReminder(iteration, agent.TaskReminderInterval) && !opts.NoHistory {
			if lastReminderIdx >= 0 && lastReminderIdx < len(messages) {
				messages = append(messages[:lastReminderIdx], messages[lastReminderIdx+1:]...)
			}
			reminderMsg := buildTaskReminder(opts.UserMessage, lastBlocker)
			messages = append(messages, reminderMsg)
			lastReminderIdx = len(messages) - 1
			logger.DebugCF("agent", "Injected task reminder",
				map[string]any{
					"agent_id":    agent.ID,
					"iteration":   iteration,
					"has_blocker": lastBlocker != "",
				})
		}

		// Inject plan-mode reminder to keep AI focused on interview/review workflow.
		if iteration > 1 && isPlanPreExecution(planSnapshot) {
			if reminder, ok := buildPlanReminder(planSnapshot); ok {
				messages = append(messages, reminder)
				logger.DebugCF("agent", "Injected plan reminder",
					map[string]any{
						"agent_id":    agent.ID,
						"iteration":   iteration,
						"plan_status": planSnapshot,
					})
			}
		}

		// Inject orchestration nudge during plan execution to encourage spawn usage.
		if planSnapshot == "executing" && agent.Subagents != nil && agent.Subagents.Enabled {
			if reminder, ok := buildOrchReminder(iteration); ok {
				messages = append(messages, reminder)
				logger.DebugCF("agent", "Injected orchestration nudge",
					map[string]any{
						"agent_id":  agent.ID,
						"iteration": iteration,
					})
			}
		}

		// Refresh system prompt: tool execution may have changed workDir,
		// memory, plan status, etc.  Update messages[0] so the next LLM
		// call sees the current state.
		if touchDir := al.sessions.GetTouchDir(opts.SessionKey); touchDir != "" {
			agent.ContextBuilder.SetWorkDir(filepath.Join(agent.Workspace, touchDir))
		}
		if newPrompt := agent.ContextBuilder.BuildSystemPrompt(); len(messages) > 0 &&
			messages[0].Content != newPrompt {
			messages[0].Content = newPrompt
			al.lastSystemPrompt.Store(newPrompt)
			al.promptDirty.Store(false)
		}
	}

	// If max iterations exhausted with tool calls still pending,
	// make one final LLM call without tools to force a text response.
	if finalContent == "" && iteration >= maxIter {
		logger.WarnCF("agent", "Max iterations reached, forcing final response without tools",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
			})
		forceResp, forceErr := agent.Provider.Chat(ctx, messages, nil, agent.Model, map[string]any{
			"max_tokens":       agent.MaxTokens,
			"temperature":      agent.Temperature,
			"prompt_cache_key": agent.ID,
		})
		if forceErr == nil && forceResp.Content != "" {
			finalContent = utils.StripThinkBlocks(forceResp.Content)
			if forceResp.Usage != nil && al.stats != nil {
				al.stats.RecordUsage(
					forceResp.Usage.PromptTokens,
					forceResp.Usage.CompletionTokens,
					forceResp.Usage.TotalTokens,
				)
			}
		}
	}

	return finalContent, iteration, nil
}

// updateToolContexts updates the context for tools that need channel/chatID info.
func (al *AgentLoop) updateToolContexts(agent *AgentInstance, channel, chatID string) {
	// Use ContextualTool interface instead of type assertions
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(channel, chatID)
		}
	}
	if tool, ok := agent.Tools.Get("spawn"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
	if tool, ok := agent.Tools.Get("subagent"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
}

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := agent.ContextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey
		if _, loading := al.summarizing.LoadOrStore(summarizeKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(summarizeKey)
				logger.InfoCF("agent", "Memory threshold reached, optimizing conversation history",
					map[string]any{
						"session_key":    sessionKey,
						"history_len":    len(newHistory),
						"token_estimate": tokenEstimate,
					})
				al.summarizeSession(agent, sessionKey)
			}()
		}
	}
}

// forceCompression aggressively reduces context when the limit is hit.
// It drops the oldest 50% of messages (keeping system prompt and last user message).
func (al *AgentLoop) forceCompression(agent *AgentInstance, sessionKey string) {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) <= 4 {
		return
	}

	// Keep system prompt (usually [0]) and the very last message (user's trigger)
	// We want to drop the oldest half of the *conversation*
	// Assuming [0] is system, [1:] is conversation
	conversation := history[1 : len(history)-1]
	if len(conversation) == 0 {
		return
	}

	// Helper to find the mid-point of the conversation
	mid := len(conversation) / 2

	// New history structure:
	// 1. System Prompt (with compression note appended)
	// 2. Second half of conversation
	// 3. Last message

	droppedCount := mid
	keptConversation := conversation[mid:]

	newHistory := make([]providers.Message, 0, 1+len(keptConversation)+1)

	// Append compression note to the original system prompt instead of adding a new system message
	// This avoids having two consecutive system messages which some APIs (like Zhipu) reject
	compressionNote := fmt.Sprintf(
		"\n\n[System Note: Emergency compression dropped %d oldest messages due to context limit]",
		droppedCount,
	)
	enhancedSystemPrompt := history[0]
	enhancedSystemPrompt.Content = enhancedSystemPrompt.Content + compressionNote
	newHistory = append(newHistory, enhancedSystemPrompt)

	newHistory = append(newHistory, keptConversation...)
	newHistory = append(newHistory, history[len(history)-1]) // Last message

	// Update session
	agent.Sessions.SetHistory(sessionKey, newHistory)
	agent.Sessions.Save(sessionKey)

	logger.WarnCF("agent", "Forced compression executed", map[string]any{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(newHistory),
	})
}

// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return info
	}

	// Tools info
	toolsList := agent.Tools.List()
	toolsMap := map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}
	// Report web search provider if registered
	if t, ok := agent.Tools.Get("web_search"); ok {
		if wst, ok := t.(*tools.WebSearchTool); ok {
			toolsMap["web_search_provider"] = wst.ProviderName()
		}
	}
	info["tools"] = toolsMap

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(al.registry.ListAgentIDs()),
		"ids":   al.registry.ListAgentIDs(),
	}

	return info
}

// ListSkills returns all available skills from the default agent.
func (al *AgentLoop) ListSkills() []skills.SkillInfo {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return nil
	}
	return agent.ContextBuilder.ListSkills()
}

// GetPlanInfo returns plan state from the default agent's memory store.
func (al *AgentLoop) GetPlanInfo() (hasPlan bool, status string, currentPhase, totalPhases int, display string, memory string) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return false, "", 0, 0, "No agent available.", ""
	}
	mem := agent.ContextBuilder.Memory()
	if mem == nil {
		return false, "", 0, 0, "No memory store.", ""
	}
	hasPlan = mem.HasActivePlan()
	status = mem.GetPlanStatus()
	currentPhase = mem.GetCurrentPhase()
	totalPhases = mem.GetTotalPhases()
	display = mem.FormatPlanDisplay()
	memory = mem.ReadLongTerm()
	return hasPlan, status, currentPhase, totalPhases, display, memory
}

// GetPlanStatus returns the current plan status ("interviewing", "executing", "review", etc.) or "".
func (al *AgentLoop) GetPlanStatus() string {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return ""
	}
	return agent.ContextBuilder.GetPlanStatus()
}

// GetPlanPhases returns structured phase/step data from the default agent's plan.
func (al *AgentLoop) GetPlanPhases() []PlanPhase {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return nil
	}
	mem := agent.ContextBuilder.Memory()
	if mem == nil {
		return nil
	}
	return mem.GetPlanPhases()
}

// GetActiveSessions returns currently active sessions for the mini app API.
func (al *AgentLoop) GetActiveSessions() []SessionEntry {
	return al.sessions.ListActive()
}

// GetSessionStats returns the current session statistics snapshot, or nil if stats tracking is disabled.
func (al *AgentLoop) GetSessionStats() *stats.Stats {
	if al.stats == nil {
		return nil
	}
	s := al.stats.GetStats()
	return &s
}

// GetContextInfo returns the bootstrap file resolution and directory context for the default agent.
func (al *AgentLoop) GetContextInfo() (workDir, planWorkDir, workspace string, bootstrap []BootstrapFileInfo) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", "", "", nil
	}
	workspace = agent.Workspace
	planWorkDir = agent.ContextBuilder.GetPlanWorkDir()
	// Use the most recent active session's touch_dir (tool-detected project directory)
	if active := al.sessions.ListActive(); len(active) > 0 && active[0].TouchDir != "" {
		workDir = active[0].TouchDir
	} else {
		workDir = agent.ContextBuilder.workDir
	}
	bootstrap = agent.ContextBuilder.ResolveBootstrapPaths()
	return workDir, planWorkDir, workspace, bootstrap
}

// GetSystemPrompt returns the system prompt last sent to the LLM.
// If the prompt is dirty (state changed since last capture), it rebuilds
// from current state. Falls back to building if no LLM call has occurred yet.
func (al *AgentLoop) GetSystemPrompt() string {
	if !al.promptDirty.Load() {
		if v := al.lastSystemPrompt.Load(); v != nil {
			return v.(string)
		}
	}
	// Rebuild from current state
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return ""
	}
	prompt := agent.ContextBuilder.BuildSystemPrompt()
	al.lastSystemPrompt.Store(prompt)
	al.promptDirty.Store(false)
	return prompt
}

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					fmt.Fprintf(&sb, "      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			fmt.Fprintf(&sb, "  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, tool := range toolDefs {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(&sb, "      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// summarizeSession summarizes the conversation history for a session.
func (al *AgentLoop) summarizeSession(agent *AgentInstance, sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := agent.Sessions.GetHistory(sessionKey)
	summary := agent.Sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard
	maxMessageTokens := agent.ContextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		msgTokens := len(m.Content) / 2
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization
	var finalSummary string
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, agent, part1, "")
		s2, _ := al.summarizeBatch(ctx, agent, part2, "")

		mergePrompt := fmt.Sprintf(
			"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
			s1,
			s2,
		)
		resp, err := agent.Provider.Chat(
			ctx,
			[]providers.Message{{Role: "user", Content: mergePrompt}},
			nil,
			agent.Model,
			map[string]any{
				"max_tokens":       1024,
				"temperature":      0.3,
				"prompt_cache_key": agent.ID,
			},
		)
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		agent.Sessions.SetSummary(sessionKey, finalSummary)
		agent.Sessions.TruncateHistory(sessionKey, 4)
		agent.Sessions.Save(sessionKey)
	}
}

// summarizeBatch summarizes a batch of messages.
func (al *AgentLoop) summarizeBatch(
	ctx context.Context,
	agent *AgentInstance,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	var sb strings.Builder
	sb.WriteString("Provide a concise summary of this conversation segment, preserving core context and key points.\n")
	if agent.ContextBuilder.HasActivePlan() {
		sb.WriteString("Note: Active plan in MEMORY.md. Preserve plan progress references.\n")
	}
	if existingSummary != "" {
		sb.WriteString("Existing context: ")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCONVERSATION:\n")
	for _, m := range batch {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}
	prompt := sb.String()

	response, err := agent.Provider.Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: prompt}},
		nil,
		agent.Model,
		map[string]any{
			"max_tokens":       1024,
			"temperature":      0.3,
			"prompt_cache_key": agent.ID,
		},
	)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// estimateTokens estimates the number of tokens in a message list.
// Uses a safe heuristic of 2.5 characters per token to account for CJK and other
// overheads better than the previous 3 chars/token.
func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	totalChars := 0
	for _, m := range messages {
		totalChars += utf8.RuneCountInString(m.Content)
	}
	// 2.5 chars per token = totalChars * 2 / 5
	return totalChars * 2 / 5
}

func (al *AgentLoop) handleCommand(ctx context.Context, msg bus.InboundMessage) (string, bool) {
	content := strings.TrimSpace(msg.Content)
	if !strings.HasPrefix(content, "/") {
		return "", false
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/show":
		if len(args) < 1 {
			return "Usage: /show [model|channel|agents]", true
		}
		switch args[0] {
		case "model":
			defaultAgent := al.registry.GetDefaultAgent()
			if defaultAgent == nil {
				return "No default agent configured", true
			}
			return fmt.Sprintf("Current model: %s", defaultAgent.Model), true
		case "channel":
			return fmt.Sprintf("Current channel: %s", msg.Channel), true
		case "agents":
			agentIDs := al.registry.ListAgentIDs()
			return fmt.Sprintf("Registered agents: %s", strings.Join(agentIDs, ", ")), true
		default:
			return fmt.Sprintf("Unknown show target: %s", args[0]), true
		}

	case "/list":
		if len(args) < 1 {
			return "Usage: /list [models|channels|agents]", true
		}
		switch args[0] {
		case "models":
			return "Available models: configured in config.json per agent", true
		case "channels":
			if al.channelManager == nil {
				return "Channel manager not initialized", true
			}
			channels := al.channelManager.GetEnabledChannels()
			if len(channels) == 0 {
				return "No channels enabled", true
			}
			return fmt.Sprintf("Enabled channels: %s", strings.Join(channels, ", ")), true
		case "agents":
			agentIDs := al.registry.ListAgentIDs()
			return fmt.Sprintf("Registered agents: %s", strings.Join(agentIDs, ", ")), true
		default:
			return fmt.Sprintf("Unknown list target: %s", args[0]), true
		}

	case "/switch":
		if len(args) < 3 || args[1] != "to" {
			return "Usage: /switch [model|channel] to <name>", true
		}
		target := args[0]
		value := args[2]

		switch target {
		case "model":
			defaultAgent := al.registry.GetDefaultAgent()
			if defaultAgent == nil {
				return "No default agent configured", true
			}
			oldModel := defaultAgent.Model
			defaultAgent.Model = value
			return fmt.Sprintf("Switched model from %s to %s", oldModel, value), true
		case "channel":
			if al.channelManager == nil {
				return "Channel manager not initialized", true
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Sprintf("Channel '%s' not found or not enabled", value), true
			}
			return fmt.Sprintf("Switched target channel to %s", value), true
		default:
			return fmt.Sprintf("Unknown switch target: %s", target), true
		}

	case "/session":
		return al.handleSessionCommand(args), true

	case "/skills":
		return al.handleSkillsCommand(), true

	case "/plan":
		resp, handled := al.handlePlanCommand(args, msg.SessionKey)
		if handled {
			al.notifyStateChange()
		}
		return resp, handled

	case "/heartbeat":
		resp, handled := al.handleHeartbeatCommand(args, msg)
		if handled {
			al.notifyStateChange()
		}
		return resp, handled
	}

	return "", false
}

func (al *AgentLoop) handleHeartbeatCommand(args []string, msg bus.InboundMessage) (string, bool) {
	if len(args) == 0 {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if args[0] != "thread" {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if len(args) < 2 {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if msg.Channel != "telegram" {
		return "/heartbeat thread is only supported from Telegram chats.", true
	}

	baseChatID, currentThreadID := splitChatAndThread(msg.ChatID)
	if baseChatID == "" {
		return "Unable to detect Telegram chat ID for heartbeat routing.", true
	}

	arg := strings.ToLower(strings.TrimSpace(args[1]))
	var threadID int
	var err error

	switch arg {
	case "off", "disable", "clear":
		threadID = 0
	case "here", "this":
		if currentThreadID <= 0 {
			return "Current Telegram message is not in a thread. Usage: /heartbeat thread <thread_id>", true
		}
		threadID = currentThreadID
	default:
		threadID, err = strconv.Atoi(arg)
		if err != nil || threadID < 0 {
			return "Usage: /heartbeat thread [here|off|<thread_id>]", true
		}
	}

	al.cfg.Channels.Telegram.HeartbeatThreadID = threadID
	if al.state != nil {
		_ = al.state.SetHeartbeatTarget(fmt.Sprintf("telegram:%s", baseChatID))
	}
	if al.onHeartbeatThreadUpdate != nil {
		al.onHeartbeatThreadUpdate(threadID)
	}

	if al.saveConfig != nil {
		if err := al.saveConfig(al.cfg); err != nil {
			return fmt.Sprintf("Failed to persist config.json: %v", err), true
		}
	}

	if threadID == 0 {
		return fmt.Sprintf("Heartbeat thread routing disabled for chat %s and saved to config.json.", baseChatID), true
	}
	return fmt.Sprintf("Heartbeat thread set to %d for chat %s and saved to config.json.", threadID, baseChatID), true
}

func splitChatAndThread(chatID string) (baseChatID string, threadID int) {
	baseChatID = strings.TrimSpace(chatID)
	if baseChatID == "" {
		return "", 0
	}
	if slash := strings.Index(baseChatID, "/"); slash >= 0 {
		threadPart := strings.TrimSpace(baseChatID[slash+1:])
		baseChatID = strings.TrimSpace(baseChatID[:slash])
		if tid, err := strconv.Atoi(threadPart); err == nil && tid > 0 {
			threadID = tid
		}
	}
	return baseChatID, threadID
}

// handleSessionCommand returns usage statistics or resets them.
func (al *AgentLoop) handleSessionCommand(args []string) string {
	if al.stats == nil {
		return "Stats tracking is disabled. Start with --stats flag to enable.\nUsage: picoclaw gateway --stats"
	}

	if len(args) > 0 && args[0] == "reset" {
		al.stats.Reset()
		return "Session statistics have been reset."
	}

	s := al.stats.GetStats()
	return fmt.Sprintf(
		"Session Statistics\n\nToday (%s):\n  Prompts: %d\n  LLM calls: %d\n  Tokens: %s (in: %s, out: %s)\n\nAll time (since %s):\n  Prompts: %d\n  LLM calls: %d\n  Tokens: %s (in: %s, out: %s)",
		s.Today.Date,
		s.Today.Prompts,
		s.Today.Requests,
		stats.FormatTokenCount(s.Today.TotalTokens),
		stats.FormatTokenCount(s.Today.PromptTokens),
		stats.FormatTokenCount(s.Today.CompletionTokens),
		s.Since.Format("2006-01-02"),
		s.TotalPrompts,
		s.TotalRequests,
		stats.FormatTokenCount(s.TotalTokens),
		stats.FormatTokenCount(s.TotalPromptTokens),
		stats.FormatTokenCount(s.TotalCompletionTokens),
	)
}

// expandSkillCommand detects "/skill <name> [message]" and returns:
//   - expanded: full content with SKILL.md injected (for LLM)
//   - compact: skill name tag + user message only (for history)
//   - ok: whether expansion happened
func (al *AgentLoop) expandSkillCommand(msg bus.InboundMessage) (expanded string, compact string, ok bool) {
	content := strings.TrimSpace(msg.Content)
	if !strings.HasPrefix(content, "/skill ") {
		return "", "", false
	}

	// Parse: /skill <name> [message]
	rest := strings.TrimSpace(content[7:]) // len("/skill ") == 7
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	skillName := parts[0]
	userMessage := ""
	if len(parts) > 1 {
		userMessage = strings.TrimSpace(parts[1])
	}

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", "", false
	}

	skillContent, found := agent.ContextBuilder.LoadSkill(skillName)
	if !found {
		return "", "", false
	}

	tag := fmt.Sprintf("[Skill: %s]", skillName)

	// Build expanded message: skill instructions + user message (for LLM)
	var sb strings.Builder
	sb.WriteString(tag)
	sb.WriteString("\n\n")
	sb.WriteString(skillContent)
	if userMessage != "" {
		sb.WriteString("\n\n---\n\n")
		sb.WriteString(userMessage)
	}

	// Build compact form: skill name tag + user message only (for history)
	compactForm := tag
	if userMessage != "" {
		compactForm = tag + "\n" + userMessage
	}

	return sb.String(), compactForm, true
}

// handleSkillsCommand lists all available skills.
func (al *AgentLoop) handleSkillsCommand() string {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "No agent configured."
	}

	skillsList := agent.ContextBuilder.ListSkills()
	if len(skillsList) == 0 {
		return "No skills available.\nAdd skills to your workspace/skills/ directory."
	}

	var sb strings.Builder
	sb.WriteString("Available Skills\n\n")
	for _, s := range skillsList {
		fmt.Fprintf(&sb, "**%s** (%s)\n", s.Name, s.Source)
		if s.Description != "" {
			fmt.Fprintf(&sb, "```\n%s\n```\n", s.Description)
		}
	}
	sb.WriteString("\nUse: /skill <name> [message]")
	return sb.String()
}

// handlePlanCommand handles /plan subcommands that can be resolved instantly.
// Returns (response, handled). For "/plan <task>" (new plan), it returns
// ("", false) so the message falls through to the LLM queue, where
// expandPlanCommand writes the seed and rewrites the content.
func (al *AgentLoop) handlePlanCommand(args []string, sessionKey string) (string, bool) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "No agent configured.", true
	}

	if len(args) == 0 {
		// /plan — show current plan
		return agent.ContextBuilder.FormatPlanDisplay(), true
	}

	sub := args[0]
	switch sub {
	case "clear":
		if agent.ContextBuilder.ReadMemory() == "" {
			return "No active plan to clear.", true
		}
		// Deactivate worktree on plan clear
		if sessionKey != "" {
			agent.DeactivateWorktree(sessionKey, "", true)
		}
		if err := agent.ContextBuilder.ClearMemory(); err != nil {
			return fmt.Sprintf("Error clearing plan: %v", err), true
		}
		return "Plan cleared.", true

	case "done":
		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}
		if len(args) < 2 {
			return "Usage: /plan done <step number>", true
		}
		stepNum, err := strconv.Atoi(args[1])
		if err != nil || stepNum < 1 {
			return "Step number must be a positive integer.", true
		}
		phase := agent.ContextBuilder.GetCurrentPhase()
		if err := agent.ContextBuilder.MarkStep(phase, stepNum); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}
		return fmt.Sprintf("Marked step %d in phase %d as done.", stepNum, phase), true

	case "add":
		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}
		if len(args) < 2 {
			return "Usage: /plan add <step description>", true
		}
		desc := strings.Join(args[1:], " ")
		phase := agent.ContextBuilder.GetCurrentPhase()
		if err := agent.ContextBuilder.AddStep(phase, desc); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}
		return fmt.Sprintf("Added step to phase %d: %s", phase, desc), true

	case "start":
		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}
		status := agent.ContextBuilder.GetPlanStatus()
		if status == "executing" {
			return "Plan is already executing.", true
		}
		if status != "interviewing" && status != "review" {
			return fmt.Sprintf("Cannot start from status %q.", status), true
		}
		if agent.ContextBuilder.GetTotalPhases() == 0 {
			return "Cannot start: no phases defined yet. Complete the interview first.", true
		}
		if err := agent.ContextBuilder.SetPlanStatus("executing"); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}
		al.reporter().ReportStateChange(sessionKey, orch.AgentStatePlanExecuting, "")
		al.planStartPending = true
		clearHistory := len(args) > 1 && args[1] == "clear"
		al.planClearHistory = clearHistory
		if clearHistory {
			return "Plan approved. Executing with clean history.", true
		}
		return "Plan approved. Executing.", true

	case "next":
		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}
		if err := agent.ContextBuilder.AdvancePhase(); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}
		phase := agent.ContextBuilder.GetCurrentPhase()
		return fmt.Sprintf("Advanced to phase %d.", phase), true

	case "worktrees":
		return al.handlePlanWorktreesCommand(agent, args[1:]), true

	default:
		// /plan <task description> — start new plan
		// Block if a plan is already active (fast-path error).
		if agent.ContextBuilder.HasActivePlan() {
			return "A plan is already active. Use /plan clear first.", true
		}
		// Not handled here — let the message flow to the LLM queue.
		// expandPlanCommand will write the seed and rewrite the content.
		return "", false
	}
}

// isPlanPreExecution returns true if the plan is in a pre-execution state
// (interviewing or review) where tool restrictions and iteration caps apply.
func (al *AgentLoop) handlePlanWorktreesCommand(agent *AgentInstance, args []string) string {
	repoRoot := git.FindRepoRoot(agent.Workspace)
	if repoRoot == "" {
		return "Workspace is not a git repository."
	}
	worktreesDir := filepath.Join(agent.Workspace, ".worktrees")

	sub := "list"
	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch sub {
	case "", "list":
		items, err := git.ListManagedWorktrees(repoRoot, worktreesDir)
		if err != nil {
			return fmt.Sprintf("Error listing worktrees: %v", err)
		}
		if len(items) == 0 {
			return "No active worktrees in workspace/.worktrees."
		}

		var sb strings.Builder
		sb.WriteString("Active worktrees\n\n")
		for _, wt := range items {
			status := "clean"
			if wt.HasUncommitted {
				status = "dirty"
			}
			last := "(no commits)"
			if wt.LastCommitHash != "" {
				if wt.LastCommitAge != "" {
					last = fmt.Sprintf("%s %s (%s)", wt.LastCommitHash, wt.LastCommitSubject, wt.LastCommitAge)
				} else {
					last = fmt.Sprintf("%s %s", wt.LastCommitHash, wt.LastCommitSubject)
				}
			}
			fmt.Fprintf(&sb, "- %s\n  branch: %s\n  status: %s\n  last: %s\n", wt.Name, wt.Branch, status, last)
		}
		sb.WriteString("\nCommands:\n")
		sb.WriteString("/plan worktrees inspect <name>\n")
		sb.WriteString("/plan worktrees merge <name>\n")
		sb.WriteString("/plan worktrees dispose <name> [force]")
		return sb.String()

	case "inspect":
		if len(args) < 2 {
			return "Usage: /plan worktrees inspect <name>"
		}
		name := args[1]
		wt, err := git.GetManagedWorktree(repoRoot, worktreesDir, name)
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}
			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}
			return fmt.Sprintf("Error inspecting worktree %q: %v", name, err)
		}
		statusOut, _ := git.WorktreeStatusShort(wt.Path)
		diffOut, _ := git.WorktreeDiffStat(wt.Path)
		logOut, _ := git.WorktreeRecentLog(wt.Path, 10)
		if statusOut == "" {
			statusOut = "(clean)"
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Worktree: %s\nBranch: %s\nDirty: %t\n", wt.Name, wt.Branch, wt.HasUncommitted)
		if wt.LastCommitHash != "" {
			fmt.Fprintf(&sb, "Last commit: %s %s", wt.LastCommitHash, wt.LastCommitSubject)
			if wt.LastCommitAge != "" {
				fmt.Fprintf(&sb, " (%s)", wt.LastCommitAge)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\nStatus:\n```\n")
		sb.WriteString(statusOut)
		sb.WriteString("\n```\n")
		if diffOut != "" {
			sb.WriteString("\nDiff (stat):\n```\n")
			sb.WriteString(diffOut)
			sb.WriteString("\n```\n")
		}
		if logOut != "" {
			sb.WriteString("\nRecent commits:\n```\n")
			sb.WriteString(logOut)
			sb.WriteString("\n```")
		}
		return sb.String()

	case "merge":
		if len(args) < 2 {
			return "Usage: /plan worktrees merge <name>"
		}
		name := args[1]
		res, base, err := git.MergeManagedWorktree(repoRoot, worktreesDir, name, "")
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}
			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}
			return fmt.Sprintf("Error merging worktree %q: %v", name, err)
		}
		if res.Conflict {
			return fmt.Sprintf("Merge conflict while merging `%s` into `%s`. Merge was aborted.", res.Branch, base)
		}
		if res.Merged {
			return fmt.Sprintf("Merged `%s` into `%s`.", res.Branch, base)
		}
		return fmt.Sprintf("No merge was performed for `%s`.", name)

	case "dispose":
		if len(args) < 2 {
			return "Usage: /plan worktrees dispose <name> [force]"
		}
		name := args[1]
		force := len(args) > 2 && strings.EqualFold(args[2], "force")
		wt, err := git.GetManagedWorktree(repoRoot, worktreesDir, name)
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}
			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}
			return fmt.Sprintf("Error disposing worktree %q: %v", name, err)
		}
		if wt.HasUncommitted && !force {
			return fmt.Sprintf(
				"Worktree `%s` has uncommitted changes. Re-run with `/plan worktrees dispose %s force` to confirm.",
				name,
				name,
			)
		}
		res, err := git.DisposeManagedWorktree(repoRoot, worktreesDir, name, "")
		if err != nil {
			return fmt.Sprintf("Error disposing worktree %q: %v", name, err)
		}
		parts := []string{fmt.Sprintf("Disposed worktree `%s` (branch `%s`).", name, res.Branch)}
		if res.AutoCommitted {
			parts = append(parts, "Uncommitted changes were auto-committed.")
		}
		if res.CommitsAhead > 0 {
			parts = append(parts, fmt.Sprintf("Branch has %d unique commit(s); branch was kept.", res.CommitsAhead))
		}
		if res.BranchDeleted {
			parts = append(parts, "Branch was deleted (no unique commits).")
		}
		return strings.Join(parts, " ")
	}

	return "Usage: /plan worktrees [list|inspect <name>|merge <name>|dispose <name> [force]]"
}

func isPlanPreExecution(status string) bool {
	return status == "interviewing" || status == "review"
}

// interviewAllowedTools is the single source of truth for tool names that may
// be sent to the LLM (and subsequently invoked) during the interview phase.
// filterInterviewTools uses this to strip tool *definitions* before the LLM call,
// while isToolAllowedDuringInterview adds argument-level checks as a second gate.
var interviewAllowedTools = map[string]bool{
	"readfile":   true,
	"listdir":    true,
	"websearch":  true,
	"webfetch":   true,
	"message":    true,
	"editfile":   true,
	"appendfile": true,
	"writefile":  true,
	"exec":       true,
	"logs":       true,
}

// filterInterviewTools removes tool definitions that are not in the
// interviewAllowedTools whitelist, reducing token usage and preventing the
// LLM from attempting disallowed tool calls during the interview phase.
func filterInterviewTools(defs []providers.ToolDefinition) []providers.ToolDefinition {
	filtered := make([]providers.ToolDefinition, 0, len(defs))
	for _, d := range defs {
		if interviewAllowedTools[tools.NormalizeToolName(d.Function.Name)] {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// isToolAllowedDuringInterview checks whether a tool call is permitted while the
// plan is in a pre-execution state. Uses the shared interviewAllowedTools map for
// name-level gating, then applies argument-level constraints for write-type tools
// (MEMORY.md only) and exec (read-only commands only).
func isToolAllowedDuringInterview(toolName string, args map[string]any) bool {
	norm := tools.NormalizeToolName(toolName)
	if !interviewAllowedTools[norm] {
		return false
	}

	// Argument-level constraints
	switch norm {
	case "editfile", "appendfile", "writefile":
		path, _ := args["path"].(string)
		return strings.HasSuffix(path, "MEMORY.md")
	case "exec":
		cmd, _ := args["command"].(string)
		return isReadOnlyCommand(cmd)
	}
	return true
}

// isReadOnlyCommand returns true when cmd is a safe, read-only shell command
// that an LLM may run during the interview phase.
func isReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}

	// Reject write operators anywhere in the command
	for _, op := range []string{">", ">>", "| tee "} {
		if strings.Contains(cmd, op) {
			return false
		}
	}

	// Reject path traversal (defense in depth; ExecTool.guardCommand also enforces workspace restriction)
	if strings.Contains(cmd, "..") {
		return false
	}
	// Block absolute paths in arguments (allow "cd /path && cmd" which is stripped later)
	for _, field := range strings.Fields(cmd) {
		if strings.HasPrefix(field, "/") && !strings.HasPrefix(cmd, "cd ") {
			return false
		}
	}

	// Strip "cd /path &&" prefix (LLM habit)
	if strings.HasPrefix(cmd, "cd ") {
		if idx := strings.Index(cmd, "&&"); idx >= 0 {
			cmd = strings.TrimSpace(cmd[idx+2:])
		}
	}

	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	first := filepath.Base(fields[0])
	switch first {
	case "find", "ls", "cat", "head", "tail", "grep", "rg",
		"tree", "wc", "file", "which", "pwd",
		"uname", "df", "du", "stat", "realpath", "dirname",
		"basename", "date":
		return true
	}
	return false
}

// isWriteTool returns true if the tool can modify files.
func isWriteTool(name string) bool {
	switch tools.NormalizeToolName(name) {
	case "writefile", "editfile", "appendfile", "exec":
		return true
	}
	return false
}

// expandPlanCommand detects "/plan <task>" (new plan start) and:
//   - writes the interview seed to MEMORY.md
//   - rewrites the message content for the LLM
//   - returns a compact form for session history
//
// This follows the same pattern as expandSkillCommand: the message is
// rewritten before reaching the LLM, so the AI sees the task description
// while the system prompt contains the interview guide.
func (al *AgentLoop) expandPlanCommand(msg bus.InboundMessage) (expanded string, compact string, ok bool) {
	content := strings.TrimSpace(msg.Content)
	if !strings.HasPrefix(content, "/plan ") {
		return "", "", false
	}

	task := strings.TrimSpace(content[6:]) // len("/plan ") == 6
	if task == "" {
		return "", "", false
	}

	// Known subcommands are handled by handlePlanCommand (fast path).
	firstWord := strings.Fields(task)[0]
	switch firstWord {
	case "clear", "done", "add", "start", "next", "worktrees":
		return "", "", false
	}

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", "", false
	}

	// If a plan is already active, don't expand — handleCommand will
	// catch it and return the error on the fast path.
	if agent.ContextBuilder.HasActivePlan() {
		return "", "", false
	}

	// Write the interview seed
	seed := BuildInterviewSeed(task, agent.Workspace)
	if err := agent.ContextBuilder.WriteMemory(seed); err != nil {
		return "", "", false
	}
	al.notifyStateChange()

	// Expanded: the task description goes to LLM.
	// The system prompt already contains the interview guide.
	expanded = task
	compact = fmt.Sprintf("[Plan: %s]", utils.Truncate(task, 80))
	return expanded, compact, true
}

// extractPeer extracts the routing peer from the inbound message's structured Peer field.
func extractPeer(msg bus.InboundMessage) *routing.RoutePeer {
	if msg.Peer.Kind == "" {
		return nil
	}
	peerID := msg.Peer.ID
	if peerID == "" {
		if msg.Peer.Kind == "direct" {
			peerID = msg.SenderID
		} else {
			peerID = msg.ChatID
		}
	}
	return &routing.RoutePeer{Kind: msg.Peer.Kind, ID: peerID}
}

// extractParentPeer extracts the parent peer (reply-to) from inbound message metadata.
func extractParentPeer(msg bus.InboundMessage) *routing.RoutePeer {
	parentKind := msg.Metadata["parent_peer_kind"]
	parentID := msg.Metadata["parent_peer_id"]
	if parentKind == "" || parentID == "" {
		return nil
	}
	return &routing.RoutePeer{Kind: parentKind, ID: parentID}
}

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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AgentLoop struct {
	bus *bus.MessageBus

	cfg *config.Config

	registry *AgentRegistry

	state *state.Manager

	stats *stats.Tracker // nil when --stats not passed

	running atomic.Bool

	summarizing sync.Map

	fallback *providers.FallbackChain

	channelManager *channels.Manager

	mediaStore media.MediaStore

	providerCache map[string]providers.LLMProvider

	planStartPending bool // set by /plan start to trigger LLM execution

	planClearHistory bool // set by /plan start clear to wipe history on transition

	sessionLocks sync.Map // sessionKey → *sessionSemaphore

	activeTasks sync.Map // sessionKey → *activeTask

	sessions *SessionTracker

	lastSystemPrompt atomic.Value // string — last system prompt sent to LLM

	promptDirty atomic.Bool // true = rebuild needed on next GetSystemPrompt read

	OnStateChange func() // called on plan/session/skills mutations

	OnUserMessage func() // called when a real user message is processed

	saveConfig func(*config.Config) error

	onHeartbeatThreadUpdate func(int)

	orchBroadcaster *orch.Broadcaster // nil when --orchestration not set

	orchReporter orch.AgentReporter // always non-nil (Noop when disabled)

	done chan struct{} // closed by Close() to stop background goroutines
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey string // Session identifier for history/context

	Channel string // Target channel for tool execution

	ChatID string // Target chat ID for tool execution

	UserMessage string // User message content (may include prefix)

	HistoryMessage string // If set, save this to history instead of UserMessage (for skill compaction)

	DefaultResponse string // Response when LLM returns empty

	EnableSummary bool // Whether to trigger summarization

	SendResponse bool // Whether to send response via bus

	NoHistory bool // If true, don't load session history (for heartbeat)

	TaskID string // Unique task ID for background task status tracking

	Background bool // If true, this is a background task (cron/heartbeat) — enables live task notifications

	SystemMessage bool // If true, this is a system message (subagent result) — skip placeholder and plan nudge
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
		bus: msgBus,

		cfg: cfg,

		registry: registry,

		state: stateManager,

		stats: statsTracker,

		summarizing: sync.Map{},

		fallback: fallbackChain,

		providerCache: providerCache,

		sessions: NewSessionTracker(),

		orchBroadcaster: orchBroadcaster,

		orchReporter: orchReporter,

		done: make(chan struct{}),
	}

	// Register shared tools to all agents (needs al for reporter injection).

	registerSharedTools(cfg, msgBus, registry, provider, al)

	go al.gcLoop()

	return al
}

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
			BraveAPIKey: cfg.Tools.Web.Brave.APIKey,

			BraveMaxResults: cfg.Tools.Web.Brave.MaxResults,

			BraveEnabled: cfg.Tools.Web.Brave.Enabled,

			TavilyAPIKey: cfg.Tools.Web.Tavily.APIKey,

			TavilyBaseURL: cfg.Tools.Web.Tavily.BaseURL,

			TavilyMaxResults: cfg.Tools.Web.Tavily.MaxResults,

			TavilyEnabled: cfg.Tools.Web.Tavily.Enabled,

			DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,

			DuckDuckGoEnabled: cfg.Tools.Web.DuckDuckGo.Enabled,

			PerplexityAPIKey: cfg.Tools.Web.Perplexity.APIKey,

			PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,

			PerplexityEnabled: cfg.Tools.Web.Perplexity.Enabled,

			Proxy: cfg.Tools.Web.Proxy,
		})

		if err != nil {
			logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{
				"agent_id": agentID,

				"error": err.Error(),
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

				"error": err.Error(),
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

				ChatID: chatID,

				Content: content,
			})
		})

		agent.Tools.Register(messageTool)

		// Skill discovery and installation tools

		registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
			MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,

			ClawHub: skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
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
				BraveAPIKey: cfg.Tools.Web.Brave.APIKey,

				BraveMaxResults: cfg.Tools.Web.Brave.MaxResults,

				BraveEnabled: cfg.Tools.Web.Brave.Enabled,

				TavilyAPIKey: cfg.Tools.Web.Tavily.APIKey,

				TavilyBaseURL: cfg.Tools.Web.Tavily.BaseURL,

				TavilyMaxResults: cfg.Tools.Web.Tavily.MaxResults,

				TavilyEnabled: cfg.Tools.Web.Tavily.Enabled,

				DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,

				DuckDuckGoEnabled: cfg.Tools.Web.DuckDuckGo.Enabled,

				PerplexityAPIKey: cfg.Tools.Web.Perplexity.APIKey,

				PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,

				PerplexityEnabled: cfg.Tools.Web.Perplexity.Enabled,
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

			// Register conductor-side escalation tools (answer questions, review plans)

			agent.Tools.Register(tools.NewAnswerSubagentTool(subagentManager))

			agent.Tools.Register(tools.NewReviewSubagentPlanTool(subagentManager))
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
				Channel: msg.Channel,

				ChatID: msg.ChatID,

				Content: "via MiniApp: " + msg.Content,

				SkipPlaceholder: true,
			})
		}

		// Fast path: handle slash commands immediately without blocking the LLM worker.

		if response, handled := al.handleCommand(ctx, msg); handled {
			if response != "" {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: msg.Channel,

					ChatID: msg.ChatID,

					Content: response,

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
					Channel: msg.Channel,

					ChatID: msg.ChatID,

					SenderID: msg.SenderID,

					SessionKey: msg.SessionKey,

					Content: "The plan has been approved. Begin executing.",

					Metadata: syntheticMeta,
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

					ChatID: msg.ChatID,

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
	select {
	case <-al.done:

		// already closed

	default:

		close(al.done)
	}

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
		Channel: channel,

		SenderID: "cron",

		ChatID: chatID,

		Content: content,

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
		SessionKey: "heartbeat",

		Channel: channel,

		ChatID: heartbeatChatID,

		UserMessage: content,

		DefaultResponse: defaultResponse,

		EnableSummary: false,

		SendResponse: false,

		NoHistory: true, // Don't load session history for heartbeat

		Background: true, // Enable live task notifications on Telegram

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
			"channel": msg.Channel,

			"chat_id": msg.ChatID,

			"sender_id": msg.SenderID,

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
		Channel: msg.Channel,

		AccountID: msg.Metadata["account_id"],

		Peer: extractPeer(msg),

		ParentPeer: extractParentPeer(msg),

		GuildID: msg.Metadata["guild_id"],

		TeamID: msg.Metadata["team_id"],
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
			"agent_id": agent.ID,

			"session_key": sessionKey,

			"matched_by": route.MatchedBy,
		})

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey: sessionKey,

		Channel: msg.Channel,

		ChatID: msg.ChatID,

		UserMessage: msg.Content,

		HistoryMessage: expansionCompact,

		DefaultResponse: defaultResponse,

		EnableSummary: true,

		SendResponse: false,

		Background: msg.Metadata["background"] == "true",
	})
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

		MaxIter: agent.MaxIterations,

		StartedAt: time.Now(),

		cancel: taskCancel,

		interrupt: make(chan string, 1),
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

								ChatID: opts.ChatID,

								Content: fmt.Sprintf("Heartbeat: merged %d commit(s) to %s.",

									ahead, wt.BaseBranch),
							})
						} else if mr.Conflict {
							_ = al.bus.PublishOutbound(cleanupCtx, bus.OutboundMessage{
								Channel: opts.Channel,

								ChatID: opts.ChatID,

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
				Channel: notifyChannel,

				ChatID: notifyChatID,

				Content: fmt.Sprintf("\U0001F916 Background task started\n%s", task.Description),

				IsTaskStatus: true,

				TaskID: opts.TaskID,
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
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: completionMsg,

						IsTaskStatus: true,

						TaskID: opts.TaskID,

						Final: true,
					})

					_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: resultContent,
					})

					doneCancel()

					return
				}
			}

			doneCtx, doneCancel := context.WithTimeout(context.Background(), 5*time.Second)

			_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
				Channel: opts.Channel,

				ChatID: opts.ChatID,

				Content: completionMsg,

				IsTaskStatus: true,

				TaskID: opts.TaskID,

				Final: true,
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
						Channel: channel,

						ChatID: chatID,

						Content: content,

						IsTaskStatus: true,

						TaskID: taskID,
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
					"session_key": opts.SessionKey,

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
			Role: "user",

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
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: planDisplay + "\n\nUse /plan start to approve, or continue chatting to refine.",

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
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: msg,

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
					Channel: opts.Channel,

					ChatID: opts.ChatID,

					Content: fmt.Sprintf("Phase %d complete. Moving to Phase %d.", prev, next),

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
			Channel: opts.Channel,

			ChatID: opts.ChatID,

			Content: finalContent,

			SkipPlaceholder: opts.SystemMessage, // suppress Telegram "Thinking..." for system messages

		})
	}

	// 9. Log response

	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]any{
			"agent_id": agent.ID,

			"session_key": opts.SessionKey,

			"iterations": iteration,

			"final_length": len(finalContent),
		})

	return finalContent, nil
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

func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
	task *activeTask,
	planSnapshot string,
) (string, int, error) {
	hooks := al.buildHooks(agent, opts, task, planSnapshot)

	iteration := 0
	var finalContent string

	for iteration < agent.MaxIterations {
		iteration++

		// Hook: iteration start (task tracking, user intervention)
		if hooks.OnIterationStart != nil {
			if msg := hooks.OnIterationStart(iteration); msg != "" {
				messages = append(messages, providers.Message{Role: "user", Content: msg})
			}
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       agent.MaxIterations,
			})

		// Build tool definitions
		providerToolDefs := agent.Tools.ToProviderDefs()

		// Hook: tool filtering (interview mode)
		if hooks.FilterTools != nil {
			providerToolDefs = hooks.FilterTools(providerToolDefs)
		}

		// Resolve model and candidates for this call
		candidates := agent.Candidates
		activeModel := agent.Model
		if hooks.SelectModel != nil {
			if m, c := hooks.SelectModel(); m != "" {
				activeModel = m
				candidates = c
			}
		}

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          agent.ID,
				"iteration":         iteration,
				"model":             activeModel,
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

		// Hook: streaming setup
		var onChunk func(string, string)
		var streamCleanup func()
		if hooks.SetupStreaming != nil {
			onChunk, streamCleanup = hooks.SetupStreaming()
		}

		// Build LLM call functions
		var response *providers.LLMResponse
		var err error

		llmOpts := map[string]any{
			"max_tokens":       agent.MaxTokens,
			"temperature":      agent.Temperature,
			"prompt_cache_key": agent.ID,
		}

		doCall := func(ctx context.Context, p providers.LLMProvider, model string) (*providers.LLMResponse, error) {
			if sp, ok := p.(providers.StreamingProvider); ok && sp.CanStream() {
				streamCtx, streamCancel := context.WithCancel(ctx)
				defer streamCancel()
				ch, sErr := sp.ChatStream(streamCtx, messages, providerToolDefs, model, llmOpts)
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
			return p.Chat(ctx, messages, providerToolDefs, model, llmOpts)
		}

		callLLM := func() (*providers.LLMResponse, error) {
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

			return doCall(ctx, agent.Provider, activeModel)
		}

		// Hook: pre-LLM state reporting
		if hooks.OnPreLLMCall != nil {
			hooks.OnPreLLMCall()
		}

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

		// Streaming cleanup
		if streamCleanup != nil {
			onChunk = nil // prevent writes after close
			streamCleanup()
			streamCleanup = nil
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

		// Detect repetition loop
		if response.FinishReason == "repetition_detected" ||
			(len(response.ToolCalls) == 0 && utils.DetectRepetitionLoop(response.Content)) {
			logger.WarnCF("agent", "Repetition loop detected in LLM response, retrying",
				map[string]any{
					"agent_id":       agent.ID,
					"iteration":      iteration,
					"finish_reason":  response.FinishReason,
					"content_length": len(response.Content),
				})

			savedMsgs := messages
			messages = append(append([]providers.Message(nil), messages...),
				providers.Message{
					Role:    "user",
					Content: "[System] Your previous response contained degenerate repetition and was discarded. Please respond normally without repeating yourself.",
				})

			response, err = callLLM()
			messages = savedMsgs

			if err != nil {
				return "", iteration, fmt.Errorf("LLM retry after repetition failed: %w", err)
			}

			if utils.DetectRepetitionLoop(response.Content) {
				logger.ErrorCF("agent", "Repetition persists after retry, returning empty",
					map[string]any{"agent_id": agent.ID})
				response.Content = ""
			}
		}

		// Strip think blocks and extract XML tool calls
		response.Content = utils.StripThinkBlocks(response.Content)
		if len(response.ToolCalls) == 0 {
			if xmlCalls := providers.ExtractXMLToolCalls(response.Content); len(xmlCalls) > 0 {
				response.ToolCalls = xmlCalls
			}
		}
		response.Content = providers.StripXMLToolCalls(response.Content)

		// Check if no tool calls
		if len(response.ToolCalls) == 0 {
			// Hook: plan continuation nudge
			if hooks.OnNoToolCalls != nil {
				if nudge, cont := hooks.OnNoToolCalls(response.Content, iteration); cont {
					messages = append(messages,
						providers.Message{Role: "assistant", Content: response.Content},
						providers.Message{Role: "user", Content: nudge},
					)
					continue
				}
			}

			finalContent = response.Content
			if finalContent == "" && response.ReasoningContent != "" {
				finalContent = response.ReasoningContent
			}
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

		// Hook: interview rejection
		if hooks.FilterToolCalls != nil {
			filtered, rejMsg := hooks.FilterToolCalls(normalizedToolCalls)
			if len(filtered) < len(normalizedToolCalls) && rejMsg != "" {
				messages = append(messages, providers.Message{Role: "user", Content: rejMsg})
			}
			normalizedToolCalls = filtered
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

		// Hook: publish tool status and record session touches
		if hooks.OnToolsProcessed != nil {
			hooks.OnToolsProcessed(ctx, iteration, normalizedToolCalls)
		}

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			extraContent := tc.ExtraContent
			thoughtSignature := ""
			if tc.Function != nil {
				thoughtSignature = tc.Function.ThoughtSignature
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:               tc.ID,
				Type:             "function",
				Name:             tc.Name,
				Arguments:        tc.Arguments,
				Function: &providers.FunctionCall{
					Name:             tc.Name,
					Arguments:        tc.Arguments,
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
		for _, tc := range normalizedToolCalls {
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
				if wt, wtErr := agent.ActivateWorktree(opts.SessionKey, taskName, hbDir); wtErr == nil {
					logger.InfoCF("agent", "Heartbeat worktree created", map[string]any{"branch": wt.Branch})
				}
			}

			// Hook: pre-tool execution (async callback, orch state)
			var asyncCallback tools.AsyncCallback
			if hooks.OnPreToolExec != nil {
				asyncCallback = hooks.OnPreToolExec(ctx, tc)
			}

			toolStart := time.Now()
			toolCtx := ctx
			if wt := agent.GetWorktree(opts.SessionKey); wt != nil {
				toolCtx = tools.WithWorkspaceOverride(toolCtx, wt.Path)
				toolCtx = tools.WithWorktreeInfo(toolCtx, wt)
			}

			toolResult := agent.Tools.ExecuteWithContext(
				toolCtx, tc.Name, tc.Arguments,
				opts.Channel, opts.ChatID, asyncCallback,
			)
			toolDuration := time.Since(toolStart)

			// Hook: post-tool execution (task log update)
			if hooks.OnToolExecDone != nil {
				hooks.OnToolExecDone(tc, toolResult, toolDuration)
			}

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: toolResult.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]any{"tool": tc.Name, "content_len": len(toolResult.ForUser)})
			}

			// If tool returned media refs, publish them as outbound media
			if len(toolResult.Media) > 0 && opts.SendResponse {
				parts := make([]bus.MediaPart, 0, len(toolResult.Media))
				for _, ref := range toolResult.Media {
					part := bus.MediaPart{Ref: ref}
					if al.mediaStore != nil {
						if _, meta, mErr := al.mediaStore.ResolveWithMeta(ref); mErr == nil {
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

		// Hook: inject reminders and trim tool log
		if hooks.InjectReminders != nil {
			hooks.InjectReminders(iteration, &messages, lastBlocker)
		}

		// Hook: refresh system prompt
		if hooks.RefreshSystemPrompt != nil {
			hooks.RefreshSystemPrompt(messages)
		}
	}

	// If max iterations exhausted with tool calls still pending,
	// make one final LLM call without tools to force a text response.
	if finalContent == "" && iteration >= agent.MaxIterations {
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

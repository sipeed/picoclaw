// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
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
	Description string
	Iteration   int
	MaxIter     int
	StartedAt   time.Time
	cancel      context.CancelFunc
	interrupt   chan string // buffered 1, for user message injection
	toolLog     []toolLogEntry
	mu          sync.Mutex
}

// toolLogEntry records a single tool call for the live terminal view.
type toolLogEntry struct {
	Name     string
	ArgsSnip string // first ~80 chars of args
	Result   string // "✓ 4.9s" or "✗ 3.2s"
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
	bus            *bus.MessageBus
	cfg            *config.Config
	registry       *AgentRegistry
	state          *state.Manager
	stats          *stats.Tracker // nil when --stats not passed
	running        atomic.Bool
	summarizing    sync.Map
	fallback         *providers.FallbackChain
	channelManager   *channels.Manager
	providerCache    map[string]providers.LLMProvider
	planStartPending bool // set by /plan start to trigger LLM execution
	sessionLocks     sync.Map // sessionKey → *sessionSemaphore
	activeTasks      sync.Map // sessionKey → *activeTask
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
}

func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider, enableStats ...bool) *AgentLoop {
	registry := NewAgentRegistry(cfg, provider)

	// Register shared tools to all agents
	registerSharedTools(cfg, msgBus, registry, provider)

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

	return &AgentLoop{
		bus:           msgBus,
		cfg:           cfg,
		registry:      registry,
		state:         stateManager,
		stats:         statsTracker,
		summarizing:   sync.Map{},
		fallback:      fallbackChain,
		providerCache: providerCache,
	}
}

// registerSharedTools registers tools that are shared across all agents (web, message, spawn).
func registerSharedTools(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
) {
	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		// Web tools
		if searchTool := tools.NewWebSearchTool(tools.WebSearchToolOptions{
			BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
			BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
			BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
			DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
			DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
			PerplexityAPIKey:     cfg.Tools.Web.Perplexity.APIKey,
			PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
			PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
		}); searchTool != nil {
			agent.Tools.Register(searchTool)
		}
		agent.Tools.Register(tools.NewWebFetchTool(50000))

		// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
		agent.Tools.Register(tools.NewI2CTool())
		agent.Tools.Register(tools.NewSPITool())

		// Message tool
		messageTool := tools.NewMessageTool()
		messageTool.SetSendCallback(func(channel, chatID, content string) error {
			msgBus.PublishOutbound(bus.OutboundMessage{
				Channel: channel,
				ChatID:  chatID,
				Content: content,
			})
			return nil
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

		// Spawn tool with allowlist checker
		subagentManager := tools.NewSubagentManager(provider, agent.Model, agent.Workspace, msgBus)
		subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)
		spawnTool := tools.NewSpawnTool(subagentManager)
		currentAgentID := agentID
		spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
			return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
		})
		agent.Tools.Register(spawnTool)

		// Update context builder with the complete tools registry
		agent.ContextBuilder.SetToolsRegistry(agent.Tools)
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

		// Fast path: handle slash commands immediately without blocking the LLM worker.
		if response, handled := al.handleCommand(ctx, msg); handled {
			if response != "" {
				al.bus.PublishOutbound(bus.OutboundMessage{
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
				select {
				case llmQueue <- bus.InboundMessage{
					Channel:    msg.Channel,
					ChatID:     msg.ChatID,
					SenderID:   msg.SenderID,
					SessionKey: msg.SessionKey,
					Content:    "The plan has been approved. Begin executing.",
					Metadata:   msg.Metadata,
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
				al.bus.PublishOutbound(bus.OutboundMessage{
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
func (al *AgentLoop) resolveProvider(providerName, modelName string, fallback providers.LLMProvider) providers.LLMProvider {
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
			map[string]interface{}{"provider": providerName, "model": modelName, "error": err.Error()})
	}

	// Fall back to legacy providers section.
	p, err := providers.CreateProviderByName(al.cfg, providerName)
	if err != nil {
		logger.WarnCF("agent", "Failed to create provider for fallback, using primary",
			map[string]interface{}{"provider": providerName, "error": err.Error()})
		return fallback
	}
	al.providerCache[key] = p
	return p
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
	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true,  // Don't load session history for heartbeat
		Background:      true,  // Enable live task notifications on Telegram
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
			stopKeywords := []string{"stop", "cancel", "abort", "停止", "中止", "やめて"}
			isStop := false
			for _, kw := range stopKeywords {
				if lower == kw {
					isStop = true
					break
				}
			}

			if isStop {
				task.cancel()
				logger.InfoCF("agent", "Task cancelled by user intervention",
					map[string]any{"task_id": taskID})
				return "Task cancelled.", nil
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
		DefaultResponse: "I've completed processing but have no response to give.",
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

	// Use default agent for system messages
	agent := al.registry.GetDefaultAgent()

	// Use the origin session for context
	sessionKey := routing.BuildAgentMainSessionKey(agent.ID)

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         originChannel,
		ChatID:          originChatID,
		UserMessage:     fmt.Sprintf("[System: %s] %s", msg.SenderID, msg.Content),
		DefaultResponse: "Background task completed.",
		EnableSummary:   false,
		SendResponse:    true,
	})
}

// acquireSessionLock gets or creates a per-session semaphore and acquires it.
// Returns false if the context is cancelled before the lock is acquired.
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
		return "", fmt.Errorf("context cancelled while waiting for session lock")
	}
	defer al.releaseSessionLock(opts.SessionKey)

	// -0. Create cancellable child context and register active task
	taskCtx, taskCancel := context.WithCancel(ctx)
	defer taskCancel()

	task := &activeTask{
		Description: utils.Truncate(opts.UserMessage, 80),
		MaxIter:     agent.MaxIterations,
		StartedAt:   time.Now(),
		cancel:      taskCancel,
		interrupt:   make(chan string, 1),
	}

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
			al.bus.PublishOutbound(bus.OutboundMessage{
				Channel:      notifyChannel,
				ChatID:       notifyChatID,
				Content:      fmt.Sprintf("\U0001F916 Background task started\n%s", task.Description),
				IsTaskStatus: true,
				TaskID:       opts.TaskID,
			})
		}
	}

	// Use TaskID as key if available (for background tasks), else sessionKey
	taskKey := opts.SessionKey
	if opts.TaskID != "" {
		taskKey = opts.TaskID
	}
	al.activeTasks.Store(taskKey, task)
	defer func() {
		al.activeTasks.Delete(taskKey)

		// Publish final task status on completion for background tasks
		if opts.TaskID != "" {
			elapsed := time.Since(task.StartedAt)
			al.bus.PublishOutbound(bus.OutboundMessage{
				Channel:      opts.Channel,
				ChatID:       opts.ChatID,
				Content:      fmt.Sprintf("\u2705 Task completed (%.1fs)\n%s", elapsed.Seconds(), task.Description),
				IsTaskStatus: true,
				TaskID:       opts.TaskID,
			})
		}
	}()

	// Replace ctx with the cancellable child context
	ctx = taskCtx

	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last channel", map[string]any{"error": err.Error()})
			}
		}
	}

	// 1. Update tool contexts
	al.updateToolContexts(agent, opts.Channel, opts.ChatID)

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
					"session_key":    opts.SessionKey,
					"removed_count":  removedCount,
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
			Content: "[System] You have been interviewing for several turns without updating memory/MEMORY.md. Please use edit_file now to save your findings to the ## Context section, or organize the plan into Phases if you have enough information.",
		})
	}

	// 2c. Snapshot plan status and MEMORY.md size before LLM iteration.
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

	// 5. Run LLM iteration loop
	finalContent, iteration, err := al.runLLMIteration(ctx, agent, messages, opts, task)
	if err != nil {
		return "", err
	}

	// If last tool had ForUser content and we already sent it, we might not need to send final response
	// This is controlled by the tool's Silent flag and ForUser content

	// 5a. Auto-advance plan phases after LLM iteration
	postStatus := agent.ContextBuilder.GetPlanStatus()
	if agent.ContextBuilder.HasActivePlan() && postStatus == "executing" {
		// Intercept: if AI changed status from interviewing to executing,
		// hijack to "review" and show the plan for user approval.
		if preStatus == "interviewing" {
			if agent.ContextBuilder.GetTotalPhases() == 0 {
				_ = agent.ContextBuilder.SetPlanStatus("interviewing")
				logger.WarnCF("agent", "Reverted plan to interviewing: no phases defined",
					map[string]interface{}{"agent_id": agent.ID})
			} else {
				_ = agent.ContextBuilder.SetPlanStatus("review")
				if !constants.IsInternalChannel(opts.Channel) {
					planDisplay := agent.ContextBuilder.FormatPlanDisplay()
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel:         opts.Channel,
						ChatID:          opts.ChatID,
						Content:         planDisplay + "\n\nUse /plan start to approve, or continue chatting to refine.",
						SkipPlaceholder: true,
					})
				}
			}
		} else if agent.ContextBuilder.GetTotalPhases() == 0 {
			// Safeguard: executing but no phases (shouldn't happen, but be safe).
			_ = agent.ContextBuilder.SetPlanStatus("interviewing")
			logger.WarnCF("agent", "Reverted plan to interviewing: no phases defined",
				map[string]interface{}{"agent_id": agent.ID})
		} else if agent.ContextBuilder.IsPlanComplete() {
			_ = agent.ContextBuilder.ClearMemory()
			if !constants.IsInternalChannel(opts.Channel) {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel:         opts.Channel,
					ChatID:          opts.ChatID,
					Content:         "\u2705 Plan completed!",
					SkipPlaceholder: true,
				})
			}
		} else if agent.ContextBuilder.IsCurrentPhaseComplete() {
			prev := agent.ContextBuilder.GetCurrentPhase()
			_ = agent.ContextBuilder.AdvancePhase()
			next := agent.ContextBuilder.GetCurrentPhase()
			if !constants.IsInternalChannel(opts.Channel) {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel:         opts.Channel,
					ChatID:          opts.ChatID,
					Content:         fmt.Sprintf("Phase %d complete. Moving to Phase %d.", prev, next),
					SkipPlaceholder: true,
				})
			}
		}
	}

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

	// 6. Save final assistant message to session
	agent.Sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	agent.Sessions.Save(opts.SessionKey)

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 8. Optional: send response via bus
	if opts.SendResponse {
		al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalContent,
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
const taskReminderMaxChars = 500
const blockerMaxChars = 200

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
			truncatedTask, truncatedBlocker,
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

// buildPlanReminder returns a reminder message for plan pre-execution states
// (interviewing / review) to keep the AI focused on the interview workflow
// during tool-call iterations.
func buildPlanReminder(planStatus string) (providers.Message, bool) {
	var content string
	switch planStatus {
	case "interviewing":
		content = "[System] You are interviewing the user to build a plan. " +
			"Ask clarifying questions and save findings to ## Context in memory/MEMORY.md using edit_file. " +
			"When you have enough information, write ## Phases and ## Commands sections."
	case "review":
		content = "[System] The plan is under review. " +
			"Wait for the user to approve or request changes. Do not proceed with execution."
	default:
		return providers.Message{}, false
	}
	return providers.Message{Role: "user", Content: content}, true
}

// cdPrefixPattern matches "cd /some/path && " at the start of a shell command.
var cdPrefixPattern = regexp.MustCompile(`^cd\s+\S+\s*&&\s*`)

// buildArgsSnippet produces a human-friendly snippet for the tool log.
// For exec: extracts the command and strips the leading "cd <workspace> && ".
// For file tools: extracts the path and strips the workspace prefix.
// Falls back to raw JSON truncation.
func buildArgsSnippet(toolName string, args map[string]interface{}, workspace string) string {
	switch toolName {
	case "exec":
		cmd, _ := args["command"].(string)
		if cmd == "" {
			break
		}
		cmd = cdPrefixPattern.ReplaceAllString(cmd, "")
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
		extra := ""
		if toolName == "edit_file" {
			if old, ok := args["old_text"].(string); ok && old != "" {
				extra = "  old:" + utils.Truncate(old, 30)
			}
		}
		return utils.Truncate(path, 60) + extra
	}

	// Default: raw JSON truncated
	argsJSON, _ := json.Marshal(args)
	return utils.Truncate(string(argsJSON), 80)
}

// maxEntryLineWidth is the max rune count for a single-line log entry.
// Telegram chat bubbles on mobile are roughly 35-40 chars wide; keeping
// entries under this avoids line-wrapping that causes height jitter.
const maxEntryLineWidth = 36

// formatCompactEntry formats a finished tool log entry as a fixed single line.
// The result marker (✓/✗) is always shown at the end regardless of truncation.
func formatCompactEntry(entry toolLogEntry) string {
	// result is e.g. "✓ 1.2s" or "✗ 3.0s" — always 6-8 chars
	result := entry.Result
	if result == "" {
		result = "\u23F3" // ⏳
	}

	prefix := entry.Name
	if entry.ArgsSnip != "" {
		prefix += " " + entry.ArgsSnip
	}

	// Budget: total ≤ maxEntryLineWidth, need space for " " + result
	budget := maxEntryLineWidth - 1 - utf8.RuneCountInString(result)
	if budget < 4 {
		budget = 4
	}

	prefixRunes := []rune(prefix)
	if len(prefixRunes) > budget {
		prefix = string(prefixRunes[:budget-1]) + "\u2026" // …
	}

	return prefix + " " + result
}

// buildRichStatus builds a terminal-like status display from the active task's tool log.
// Layout is designed for fixed height: past entries are always 1 line,
// and the latest entry has reserved space for potential error detail.
func buildRichStatus(task *activeTask, isBackground bool, workspace string) string {
	task.mu.Lock()
	defer task.mu.Unlock()

	var sb strings.Builder
	fmt.Fprintf(&sb, "\U0001F504 Task in progress (%d/%d)\n", task.Iteration, task.MaxIter)
	// Show project directory name so user knows which workspace is active
	if workspace != "" {
		project := workspace
		if idx := strings.LastIndex(workspace, "/"); idx >= 0 {
			project = workspace[idx+1:]
		} else if idx := strings.LastIndex(workspace, "\\"); idx >= 0 {
			project = workspace[idx+1:]
		}
		if project != "" {
			fmt.Fprintf(&sb, "\U0001F4C2 %s\n", project)
		}
	}
	sb.WriteString("\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n")

	// Sliding window: show only the last maxToolLogEntries entries
	entries := task.toolLog
	if len(entries) > maxToolLogEntries {
		entries = entries[len(entries)-maxToolLogEntries:]
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1
		isErr := strings.HasPrefix(entry.Result, "\u2717")

		if !isLast {
			// Past entries: always exactly 1 compact line
			sb.WriteString(formatCompactEntry(entry))
			sb.WriteString("\n")
		} else {
			// Latest entry: reserved area for detail
			sb.WriteString(formatCompactEntry(entry))
			sb.WriteString("\n")

			if isErr && entry.ErrDetail != "" {
				// Error block in code-fence style
				sb.WriteString("```\n")
				for _, line := range strings.Split(entry.ErrDetail, "\n") {
					sb.WriteString(line)
					sb.WriteString("\n")
				}
				sb.WriteString("```\n")
			} else {
				// Reserve height: blank line so bubble doesn't shrink
				// when the next update adds error detail
				sb.WriteString("\n")
			}
		}
	}

	// If no entries yet, still reserve the space
	if len(entries) == 0 {
		sb.WriteString("\u23F3 waiting...\n\n")
	}

	sb.WriteString("\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n")
	if isBackground {
		sb.WriteString("\u21A9\uFE0F Reply to intervene")
	}
	return sb.String()
}

// runLLMIteration executes the LLM call loop with tool handling.
func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
	task *activeTask,
) (string, int, error) {
	iteration := 0
	var finalContent string
	lastReminderIdx := -1

	maxIter := agent.MaxIterations

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

		callLLM := func() (*providers.LLMResponse, error) {
			if len(agent.Candidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(ctx, agent.Candidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						p := al.resolveProvider(provider, model, agent.Provider)
						return p.Chat(ctx, messages, providerToolDefs, model, map[string]interface{}{
							"max_tokens":  agent.MaxTokens,
							"temperature": agent.Temperature,
						})
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
			return agent.Provider.Chat(ctx, messages, providerToolDefs, agent.Model, map[string]any{
				"max_tokens":  agent.MaxTokens,
				"temperature": agent.Temperature,
			})
		}

		// Retry loop for context/token errors
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM()
			if err == nil {
				break
			}

			errMsg := strings.ToLower(err.Error())
			isContextError := strings.Contains(errMsg, "token") ||
				strings.Contains(errMsg, "context") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "length")

			if isContextError && retry < maxRetries {
				logger.WarnCF("agent", "Context window error detected, attempting compression", map[string]any{
					"error": err.Error(),
					"retry": retry,
				})

				if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
					al.bus.PublishOutbound(bus.OutboundMessage{
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

		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
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
			}
			task.mu.Unlock()

			statusContent := buildRichStatus(task, isBackground, agent.Workspace)
			if isBackground {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel:      opts.Channel,
					ChatID:       opts.ChatID,
					Content:      statusContent,
					IsTaskStatus: true,
					TaskID:       opts.TaskID,
				})
			} else {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel:  opts.Channel,
					ChatID:   opts.ChatID,
					Content:  statusContent,
					IsStatus: true,
				})
			}
		}

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
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

			// Create async callback for tools that implement AsyncTool
			// NOTE: Following openclaw's design, async tools do NOT send results directly to users.
			// Instead, they notify the agent via PublishInbound, and the agent decides
			// whether to forward the result to the user (in processSystemMessage).
			asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
				// Log the async completion but don't send directly to user
				// The agent will handle user notification via processSystemMessage
				if !result.Silent && result.ForUser != "" {
					logger.InfoCF("agent", "Async tool completed, agent will handle notification",
						map[string]any{
							"tool":        tc.Name,
							"content_len": len(result.ForUser),
						})
				}
			}

			// Block non-allowed tools during plan interview mode.
			// Only read-type tools and MEMORY.md writes are permitted.
			toolStart := time.Now()
			var toolResult *tools.ToolResult
			if isPlanPreExecution(agent.ContextBuilder.GetPlanStatus()) && !isToolAllowedDuringInterview(tc.Name, tc.Arguments) {
				toolResult = tools.ErrorResult("Interview mode: only read tools and MEMORY.md edits are allowed. Focus on asking questions and updating the plan.")
			} else {
				toolResult = agent.Tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
			}
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
							task.toolLog[logIdx].ErrDetail = utils.Truncate(toolResult.Err.Error(), 120)
						} else if toolResult.ForLLM != "" {
							// exec returns IsError with exit info in ForLLM, not Err
							// Show last few lines (stderr / exit code)
							lines := strings.Split(strings.TrimSpace(toolResult.ForLLM), "\n")
							start := len(lines) - 3
							if start < 0 {
								start = 0
							}
							task.toolLog[logIdx].ErrDetail = utils.Truncate(
								strings.Join(lines[start:], "\n"), 200)
						}
					} else {
						task.toolLog[logIdx].Result = fmt.Sprintf("\u2713 %.1fs", toolDuration.Seconds())
					}
				}
				task.mu.Unlock()
			}

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(bus.OutboundMessage{
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
				map[string]interface{}{
					"agent_id":    agent.ID,
					"iteration":   iteration,
					"has_blocker": lastBlocker != "",
				})
		}

		// Inject plan-mode reminder to keep AI focused on interview/review workflow.
		if iteration > 1 && isPlanPreExecution(agent.ContextBuilder.GetPlanStatus()) {
			if reminder, ok := buildPlanReminder(agent.ContextBuilder.GetPlanStatus()); ok {
				messages = append(messages, reminder)
				logger.DebugCF("agent", "Injected plan reminder",
					map[string]interface{}{
						"agent_id":    agent.ID,
						"iteration":   iteration,
						"plan_status": agent.ContextBuilder.GetPlanStatus(),
					})
			}
		}
	}

	// If max iterations exhausted with tool calls still pending,
	// make one final LLM call without tools to force a text response.
	if finalContent == "" && iteration >= maxIter {
		logger.WarnCF("agent", "Max iterations reached, forcing final response without tools",
			map[string]interface{}{
				"agent_id":  agent.ID,
				"iteration": iteration,
			})
		forceResp, forceErr := agent.Provider.Chat(ctx, messages, nil, agent.Model, map[string]interface{}{
			"max_tokens":  agent.MaxTokens,
			"temperature": agent.Temperature,
		})
		if forceErr == nil && forceResp.Content != "" {
			finalContent = forceResp.Content
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
					map[string]interface{}{
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

	newHistory := make([]providers.Message, 0)

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
	info["tools"] = map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(al.registry.ListAgentIDs()),
		"ids":   al.registry.ListAgentIDs(),
	}

	return info
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
				"max_tokens":  1024,
				"temperature": 0.3,
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
			"max_tokens":  1024,
			"temperature": 0.3,
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
		resp, handled := al.handlePlanCommand(args)
		return resp, handled
	}

	return "", false
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
	return fmt.Sprintf("Session Statistics\n\nToday (%s):\n  Prompts: %d\n  LLM calls: %d\n  Tokens: %s (in: %s, out: %s)\n\nAll time (since %s):\n  Prompts: %d\n  LLM calls: %d\n  Tokens: %s (in: %s, out: %s)",
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
		sb.WriteString(fmt.Sprintf("**%s** (%s)\n", s.Name, s.Source))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n", s.Description))
		}
	}
	sb.WriteString("\nUse: /skill <name> [message]")
	return sb.String()
}

// handlePlanCommand handles /plan subcommands that can be resolved instantly.
// Returns (response, handled). For "/plan <task>" (new plan), it returns
// ("", false) so the message falls through to the LLM queue, where
// expandPlanCommand writes the seed and rewrites the content.
func (al *AgentLoop) handlePlanCommand(args []string) (string, bool) {
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
		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan to clear.", true
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
		al.planStartPending = true
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
func isPlanPreExecution(status string) bool {
	return status == "interviewing" || status == "review"
}

// isToolAllowedDuringInterview checks whether a tool call is permitted while the
// plan is in a pre-execution state. Read-type tools are always allowed. Write-type
// tools (edit_file, append_file, write_file) are only allowed when targeting MEMORY.md.
// Uses normalized names so "readfile" matches "read_file", etc.
func isToolAllowedDuringInterview(toolName string, args map[string]interface{}) bool {
	norm := tools.NormalizeToolName(toolName)

	// Read-type tools: always allowed
	switch norm {
	case "readfile", "listdir", "websearch", "webfetch":
		return true
	}

	// Write-type tools: allowed only when targeting MEMORY.md
	switch norm {
	case "editfile", "appendfile", "writefile":
		path, _ := args["path"].(string)
		return strings.HasSuffix(path, "MEMORY.md")
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
	case "clear", "done", "add", "start", "next":
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
	seed := BuildInterviewSeed(task)
	if err := agent.ContextBuilder.WriteMemory(seed); err != nil {
		return "", "", false
	}

	// Expanded: the task description goes to LLM.
	// The system prompt already contains the interview guide.
	expanded = task
	compact = fmt.Sprintf("[Plan: %s]", utils.Truncate(task, 80))
	return expanded, compact, true
}

// extractPeer extracts the routing peer from inbound message metadata.
func extractPeer(msg bus.InboundMessage) *routing.RoutePeer {
	peerKind := msg.Metadata["peer_kind"]
	if peerKind == "" {
		return nil
	}
	peerID := msg.Metadata["peer_id"]
	if peerID == "" {
		if peerKind == "direct" {
			peerID = msg.SenderID
		} else {
			peerID = msg.ChatID
		}
	}
	return &routing.RoutePeer{Kind: peerKind, ID: peerID}
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

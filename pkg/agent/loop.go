// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/research"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type AgentLoop struct {
	loopExt // fork-specific fields (see loop_ext.go)

	// Core dependencies
	bus      *bus.MessageBus
	cfg      *config.Config
	registry *AgentRegistry
	state    *state.Manager

	// Event system
	eventBus *EventBus
	hooks    *HookManager

	// Runtime state
	running        atomic.Bool
	summarizing    sync.Map
	fallback       *providers.FallbackChain
	channelManager *channels.Manager
	mediaStore     media.MediaStore
	transcriber    voice.Transcriber
	cmdRegistry    *commands.Registry
	mcp            mcpRuntime
	hookRuntime    hookRuntime
	steering       *steeringQueue
	pendingSkills  sync.Map // sessionKey → skillName (armed by /use <skill>)
	mu             sync.RWMutex

	providerCache map[string]providers.LLMProvider

	// Concurrent turn management
	activeTurnStates sync.Map     // key: sessionKey (string), value: *turnState
	subTurnCounter   atomic.Int64 // Counter for generating unique SubTurn IDs

	// Turn tracking
	turnSeq        atomic.Uint64
	activeRequests sync.WaitGroup

	lastSystemPrompt atomic.Value // string — last system prompt sent to LLM
	promptDirty      atomic.Bool  // true = rebuild needed on next GetSystemPrompt read

	OnStateChange func() // called on plan/session/skills mutations
	OnUserMessage func() // called when a real user message is processed

	reloadFunc func() error
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey              string              // Session identifier for history/context
	Channel                 string              // Target channel for tool execution
	ChatID                  string              // Target chat ID for tool execution
	SenderID                string              // Current sender ID for dynamic context
	SenderDisplayName       string              // Current sender display name for dynamic context
	UserMessage             string              // User message content (may include prefix)
	ForcedSkills            []string            // Skills explicitly requested for this message
	SystemPromptOverride    string              // Override the default system prompt (Used by SubTurns)
	Media                   []string            // media:// refs from inbound message
	InitialSteeringMessages []providers.Message // Steering messages from refactor/agent
	HistoryMessage          string              // If set, save this to history instead of UserMessage (for skill compaction)
	DefaultResponse         string              // Response when LLM returns empty
	EnableSummary           bool                // Whether to trigger summarization
	SendResponse            bool                // Whether to send response via bus
	NoHistory               bool                // If true, don't load session history (for heartbeat)
	SkipInitialSteeringPoll bool                // If true, skip the steering poll at loop start (used by Continue)
	TaskID                  string              // Unique task ID for background task status tracking
	Background              bool                // If true, this is a background task (cron/heartbeat) — enables live task notifications
	SystemMessage           bool                // If true, this is a system message (subagent result) — skip placeholder and plan nudge
}

const (
	defaultResponse           = "The model returned an empty response. This may indicate a provider error or token limit."
	toolLimitResponse         = "I've reached `max_tool_iterations` without a final response. Increase `max_tool_iterations` in config.json if this task needs more tool steps."
	sessionKeyAgentPrefix     = "agent:"
	metadataKeyAccountID      = "account_id"
	metadataKeyGuildID        = "guild_id"
	metadataKeyTeamID         = "team_id"
	metadataKeyParentPeerKind = "parent_peer_kind"
	metadataKeyParentPeerID   = "parent_peer_id"
)

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
	eventBus := NewEventBus()
	al := &AgentLoop{
		bus:         msgBus,
		cfg:         cfg,
		registry:    registry,
		state:       stateManager,
		eventBus:    eventBus,
		summarizing: sync.Map{},

		fallback: fallbackChain,

		providerCache: providerCache,

		cmdRegistry: commands.NewRegistry(commands.BuiltinDefinitions()),
		steering:    newSteeringQueue(parseSteeringMode(cfg.Agents.Defaults.SteeringMode)),
	}
	al.hooks = NewHookManager(eventBus)
	configureHookManagerFromConfig(al.hooks, cfg)

	// Register shared tools to all agents (now that al is created)
	registerSharedTools(al, cfg, msgBus, registry, provider)

	// Initialize fork-specific fields (stats, sessions, orchestration, gcLoop).
	al.initLoopExt(cfg, registry, len(enableStats) > 0 && enableStats[0])

	return al
}

// registerSharedTools registers tools that are shared across all agents (web, message, spawn).
func registerSharedTools(
	al *AgentLoop,
	cfg *config.Config,
	msgBus *bus.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
) {
	allowReadPaths := buildAllowReadPatterns(cfg)

	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		// Web tools
		if cfg.Tools.IsToolEnabled("web") {
			searchTool, err := tools.NewWebSearchTool(tools.WebSearchToolOptions{
				BraveAPIKeys:    config.MergeAPIKeys(cfg.Tools.Web.Brave.APIKey(), cfg.Tools.Web.Brave.APIKeys()),
				BraveMaxResults: cfg.Tools.Web.Brave.MaxResults,
				BraveEnabled:    cfg.Tools.Web.Brave.Enabled,
				TavilyAPIKeys: config.MergeAPIKeys(
					cfg.Tools.Web.Tavily.APIKey(),
					cfg.Tools.Web.Tavily.APIKeys(),
				),
				TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
				TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
				TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
				DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
				DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
				PerplexityAPIKeys: config.MergeAPIKeys(
					cfg.Tools.Web.Perplexity.APIKey(),
					cfg.Tools.Web.Perplexity.APIKeys(),
				),
				PerplexityMaxResults:  cfg.Tools.Web.Perplexity.MaxResults,
				PerplexityEnabled:     cfg.Tools.Web.Perplexity.Enabled,
				SearXNGBaseURL:        cfg.Tools.Web.SearXNG.BaseURL,
				SearXNGMaxResults:     cfg.Tools.Web.SearXNG.MaxResults,
				SearXNGEnabled:        cfg.Tools.Web.SearXNG.Enabled,
				GLMSearchAPIKey:       cfg.Tools.Web.GLMSearch.APIKey(),
				GLMSearchBaseURL:      cfg.Tools.Web.GLMSearch.BaseURL,
				GLMSearchEngine:       cfg.Tools.Web.GLMSearch.SearchEngine,
				GLMSearchMaxResults:   cfg.Tools.Web.GLMSearch.MaxResults,
				GLMSearchEnabled:      cfg.Tools.Web.GLMSearch.Enabled,
				BaiduSearchAPIKey:     cfg.Tools.Web.BaiduSearch.APIKey(),
				BaiduSearchBaseURL:    cfg.Tools.Web.BaiduSearch.BaseURL,
				BaiduSearchMaxResults: cfg.Tools.Web.BaiduSearch.MaxResults,
				BaiduSearchEnabled:    cfg.Tools.Web.BaiduSearch.Enabled,
				Proxy:                 cfg.Tools.Web.Proxy,
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
		}

		if cfg.Tools.IsToolEnabled("web_fetch") {
			fetchTool, err := tools.NewWebFetchToolWithProxy(
				50000,
				cfg.Tools.Web.Proxy,
				cfg.Tools.Web.Format,
				cfg.Tools.Web.FetchLimitBytes,
				cfg.Tools.Web.PrivateHostWhitelist)
			if err != nil {
				logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{
					"agent_id": agentID,
					"error":    err.Error(),
				})
			} else {
				agent.Tools.Register(fetchTool)
			}
		}

		// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
		if cfg.Tools.IsToolEnabled("i2c") {
			agent.Tools.Register(tools.NewI2CTool())
		}
		if cfg.Tools.IsToolEnabled("spi") {
			agent.Tools.Register(tools.NewSPITool())
		}

		// Message tool
		if cfg.Tools.IsToolEnabled("message") {
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
		}

		// Send file tool (outbound media via MediaStore — store injected later by SetMediaStore)
		if cfg.Tools.IsToolEnabled("send_file") {
			sendFileTool := tools.NewSendFileTool(
				agent.Workspace,
				cfg.Agents.Defaults.RestrictToWorkspace,
				cfg.Agents.Defaults.GetMaxMediaSize(),
				nil,
				allowReadPaths,
			)
			agent.Tools.Register(sendFileTool)
		}

		// Skill discovery and installation tools
		skills_enabled := cfg.Tools.IsToolEnabled("skills")
		find_skills_enable := cfg.Tools.IsToolEnabled("find_skills")
		install_skills_enable := cfg.Tools.IsToolEnabled("install_skill")
		if skills_enabled && (find_skills_enable || install_skills_enable) {
			clawHubConfig := cfg.Tools.Skills.Registries.ClawHub
			registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
				MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
				ClawHub: skills.ClawHubConfig{
					Enabled:         clawHubConfig.Enabled,
					BaseURL:         clawHubConfig.BaseURL,
					AuthToken:       clawHubConfig.AuthToken(),
					SearchPath:      clawHubConfig.SearchPath,
					SkillsPath:      clawHubConfig.SkillsPath,
					DownloadPath:    clawHubConfig.DownloadPath,
					Timeout:         clawHubConfig.Timeout,
					MaxZipSize:      clawHubConfig.MaxZipSize,
					MaxResponseSize: clawHubConfig.MaxResponseSize,
				},
			})

			if find_skills_enable {
				searchCache := skills.NewSearchCache(
					cfg.Tools.Skills.SearchCache.MaxSize,
					time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
				)
				agent.Tools.Register(tools.NewFindSkillsTool(registryMgr, searchCache))
			}

			if install_skills_enable {
				agent.Tools.Register(tools.NewInstallSkillTool(registryMgr, agent.Workspace))
			}
		}

		// Orchestration tools (spawn, subagent, answer, review_plan)
		registerOrchestrationTools(cfg, agent, agentID, registry, provider, msgBus, al)

		// Update context builder with the complete tools registry
		agent.ContextBuilder.SetToolsRegistry(agent.Tools)
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	if err := al.ensureHooksInitialized(ctx); err != nil {
		return err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return err
	}

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

	inbound := al.bus.InboundChan()
	for al.running.Load() {
		var msg bus.InboundMessage
		select {
		case <-ctx.Done():
			return nil
		case m, ok := <-inbound:
			if !ok {
				return nil
			}
			msg = m
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
		defaultAgent := al.registry.GetDefaultAgent()
		if response, handled := al.handleCommand(ctx, msg, defaultAgent, msg.SessionKey); handled {
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

		// PDF two-phase handling:
		//   Phase 1: wait briefly for OCR keyword follow-up ("figures"/"図版")
		//   Phase 2: buffer messages during OCR, support cancel
		if messageHasBareFile(msg) {
			al.llmWorkerPDF(ctx, msg, queue)
			continue
		}

		al.llmWorkerNormal(ctx, msg, queue)
	}
}

// llmWorkerNormal processes a single non-PDF message.
// queue is optional (nil when called outside Run loop).
func (al *AgentLoop) llmWorkerNormal(ctx context.Context, msg bus.InboundMessage, queue <-chan bus.InboundMessage) {
	al.activeRequests.Add(1)
	defer al.activeRequests.Done()

	// Ensure typing indicator is stopped when processing completes.
	if al.channelManager != nil {
		defer al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
	}

	// Reset per-round message-tool state so a previous round's
	// tool-sent flag does not suppress this round's response.
	if defaultAgent := al.registry.GetDefaultAgent(); defaultAgent != nil {
		if tool, ok := defaultAgent.Tools.Get("message"); ok {
			if mt, ok := tool.(*tools.MessageTool); ok {
				mt.ResetSentInRound()
			}
		}
	}

	response, err := al.processMessage(ctx, msg)
	if err != nil {
		response = fmt.Sprintf("Error processing message: %v", err)
	}

	// Auto-continue: if inbound messages arrived in the queue while this
	// turn was running, treat them as steering continuations so the agent
	// responds with the full context instead of two separate turns.
	if queue != nil {
		if drained := al.drainQueueAsSteering(queue, msg); len(drained) > 0 {
			agent := al.agentForSession(msg.SessionKey)
			if agent == nil {
				agent = al.registry.GetDefaultAgent()
			}
			if agent != nil {
				sessionKey := msg.SessionKey
				if sessionKey == "" {
					route, _, _ := al.resolveMessageRoute(msg)
					sessionKey = resolveScopeKey(route, msg.SessionKey)
				}
				contResp, contErr := al.continueWithSteeringMessages(
					ctx, agent, sessionKey, msg.Channel, msg.ChatID, drained,
				)
				if contErr == nil && contResp != "" {
					response = contResp
				}
			}
		}
	}

	al.sendResponseIfNeeded(ctx, msg, response)
}

// drainQueueAsSteering non-blocking drains pending messages from the
// llmQueue that belong to the same chat as the original message and
// returns them as steering-style provider messages.
func (al *AgentLoop) drainQueueAsSteering(
	queue <-chan bus.InboundMessage,
	orig bus.InboundMessage,
) []providers.Message {
	var msgs []providers.Message
	for {
		select {
		case m, ok := <-queue:
			if !ok {
				return msgs
			}
			if m.Channel == orig.Channel && m.ChatID == orig.ChatID {
				msgs = append(msgs, providers.Message{
					Role:    "user",
					Content: m.Content,
				})
			} else {
				// Re-queue by pushing to steering for later processing
				al.enqueueSteeringMessage("", "", providers.Message{
					Role:    "user",
					Content: m.Content,
				})
			}
		default:
			return msgs
		}
	}
}

// llmWorkerPDF handles a bare-PDF message with two-phase follow-up collection.
func (al *AgentLoop) llmWorkerPDF(ctx context.Context, msg bus.InboundMessage, queue <-chan bus.InboundMessage) {
	// Phase 1: wait for OCR keywords (figures/図版) — up to 5 seconds.
	var overflow []bus.InboundMessage
	msg, overflow = al.waitForPDFFollowUp(ctx, msg, queue)

	// Phase 2: run processMessage (OCR) concurrently while buffering
	// messages from the same chat. Cancel on "中止"/"cancel".
	al.resetMessageTool()

	response, err, buffered := al.processPDFWithBuffering(ctx, msg, queue, overflow)
	if err != nil {
		response = fmt.Sprintf("Error processing message: %v", err)
	}

	al.sendResponseIfNeeded(ctx, msg, response)

	// Process buffered same-chat messages as a single follow-up turn
	// so the LLM sees user instructions alongside the OCR result.
	followUpText := mergeBufferedMessages(buffered, msg.ChatID)
	if followUpText != "" {
		notice := formatBufferedNotice(len(buffered))
		if notice != "" {
			_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel:         msg.Channel,
				ChatID:          msg.ChatID,
				Content:         notice,
				SkipPlaceholder: true,
				IsStatus:        true,
			})
		}

		followUpMsg := bus.InboundMessage{
			Channel:    msg.Channel,
			SenderID:   msg.SenderID,
			Sender:     msg.Sender,
			ChatID:     msg.ChatID,
			Peer:       msg.Peer,
			SessionKey: msg.SessionKey,
			Content:    followUpText,
			Metadata:   msg.Metadata,
		}
		al.llmWorkerNormal(ctx, followUpMsg, nil)
	}

	// Re-queue messages from other chats that were buffered.
	otherMsgs := extractNonChatMessages(buffered, msg.ChatID)
	for _, other := range otherMsgs {
		al.llmWorkerNormal(ctx, other, nil)
	}
}

// sendResponseIfNeeded sends the LLM response unless the message tool
// already sent it during this round.
func (al *AgentLoop) sendResponseIfNeeded(ctx context.Context, msg bus.InboundMessage, response string) {
	if response == "" {
		return
	}

	alreadySent := false
	if defaultAgent := al.registry.GetDefaultAgent(); defaultAgent != nil {
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

// resetMessageTool resets the per-round message-tool state.
func (al *AgentLoop) resetMessageTool() {
	if defaultAgent := al.registry.GetDefaultAgent(); defaultAgent != nil {
		if tool, ok := defaultAgent.Tools.Get("message"); ok {
			if mt, ok := tool.(*tools.MessageTool); ok {
				mt.ResetSentInRound()
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
	// Wait for in-flight LLM requests to finish before releasing resources.
	al.activeRequests.Wait()

	al.closeExt()

	mcpManager := al.mcp.takeManager()
	if mcpManager != nil {
		if err := mcpManager.Close(); err != nil {
			logger.ErrorCF("agent", "Failed to close MCP manager",
				map[string]any{
					"error": err.Error(),
				})
		}
	}

	al.GetRegistry().Close()
	if al.hooks != nil {
		al.hooks.Close()
	}
	if al.eventBus != nil {
		al.eventBus.Close()
	}
}

// MountHook registers an in-process hook on the agent loop.
func (al *AgentLoop) MountHook(reg HookRegistration) error {
	if al == nil || al.hooks == nil {
		return fmt.Errorf("hook manager is not initialized")
	}
	return al.hooks.Mount(reg)
}

// UnmountHook removes a previously registered in-process hook.
func (al *AgentLoop) UnmountHook(name string) {
	if al == nil || al.hooks == nil {
		return
	}
	al.hooks.Unmount(name)
}

// SubscribeEvents registers a subscriber for agent-loop events.
func (al *AgentLoop) SubscribeEvents(buffer int) EventSubscription {
	if al == nil || al.eventBus == nil {
		ch := make(chan Event)
		close(ch)
		return EventSubscription{C: ch}
	}
	return al.eventBus.Subscribe(buffer)
}

// UnsubscribeEvents removes a previously registered event subscriber.
func (al *AgentLoop) UnsubscribeEvents(id uint64) {
	if al == nil || al.eventBus == nil {
		return
	}
	al.eventBus.Unsubscribe(id)
}

// EventDrops returns the number of dropped events for the given kind.
func (al *AgentLoop) EventDrops(kind EventKind) int64 {
	if al == nil || al.eventBus == nil {
		return 0
	}
	return al.eventBus.Dropped(kind)
}

type turnEventScope struct {
	agentID    string
	sessionKey string
	turnID     string
}

func (al *AgentLoop) newTurnEventScope(agentID, sessionKey string) turnEventScope {
	seq := al.turnSeq.Add(1)
	return turnEventScope{
		agentID:    agentID,
		sessionKey: sessionKey,
		turnID:     fmt.Sprintf("%s-turn-%d", agentID, seq),
	}
}

func (al *AgentLoop) emitEvent(kind EventKind, meta EventMeta, payload any) {
	evt := Event{
		Kind:    kind,
		Meta:    meta,
		Payload: payload,
	}

	if al == nil || al.eventBus == nil {
		return
	}

	al.logEvent(evt)

	al.eventBus.Emit(evt)
}

func (al *AgentLoop) logEvent(evt Event) {
	fields := map[string]any{
		"event_kind":  evt.Kind.String(),
		"agent_id":    evt.Meta.AgentID,
		"turn_id":     evt.Meta.TurnID,
		"session_key": evt.Meta.SessionKey,
		"iteration":   evt.Meta.Iteration,
	}

	if evt.Meta.TracePath != "" {
		fields["trace"] = evt.Meta.TracePath
	}
	if evt.Meta.Source != "" {
		fields["source"] = evt.Meta.Source
	}

	switch payload := evt.Payload.(type) {
	case TurnStartPayload:
		fields["channel"] = payload.Channel
		fields["chat_id"] = payload.ChatID
		fields["user_len"] = len(payload.UserMessage)
		fields["media_count"] = payload.MediaCount
	case TurnEndPayload:
		fields["status"] = payload.Status
		fields["iterations_total"] = payload.Iterations
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["final_len"] = payload.FinalContentLen
	case LLMRequestPayload:
		fields["model"] = payload.Model
		fields["messages"] = payload.MessagesCount
		fields["tools"] = payload.ToolsCount
		fields["max_tokens"] = payload.MaxTokens
	case LLMDeltaPayload:
		fields["content_delta_len"] = payload.ContentDeltaLen
		fields["reasoning_delta_len"] = payload.ReasoningDeltaLen
	case LLMResponsePayload:
		fields["content_len"] = payload.ContentLen
		fields["tool_calls"] = payload.ToolCalls
		fields["has_reasoning"] = payload.HasReasoning
	case LLMRetryPayload:
		fields["attempt"] = payload.Attempt
		fields["max_retries"] = payload.MaxRetries
		fields["reason"] = payload.Reason
		fields["error"] = payload.Error
		fields["backoff_ms"] = payload.Backoff.Milliseconds()
	case ContextCompressPayload:
		fields["reason"] = payload.Reason
		fields["dropped_messages"] = payload.DroppedMessages
		fields["remaining_messages"] = payload.RemainingMessages
	case SessionSummarizePayload:
		fields["summarized_messages"] = payload.SummarizedMessages
		fields["kept_messages"] = payload.KeptMessages
		fields["summary_len"] = payload.SummaryLen
		fields["omitted_oversized"] = payload.OmittedOversized
	case ToolExecStartPayload:
		fields["tool"] = payload.Tool
		fields["args_count"] = len(payload.Arguments)
	case ToolExecEndPayload:
		fields["tool"] = payload.Tool
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["for_llm_len"] = payload.ForLLMLen
		fields["for_user_len"] = payload.ForUserLen
		fields["is_error"] = payload.IsError
		fields["async"] = payload.Async
	case ToolExecSkippedPayload:
		fields["tool"] = payload.Tool
		fields["reason"] = payload.Reason
	case SteeringInjectedPayload:
		fields["count"] = payload.Count
		fields["total_content_len"] = payload.TotalContentLen
	case FollowUpQueuedPayload:
		fields["source_tool"] = payload.SourceTool
		fields["channel"] = payload.Channel
		fields["chat_id"] = payload.ChatID
		fields["content_len"] = payload.ContentLen
	case InterruptReceivedPayload:
		fields["interrupt_kind"] = payload.Kind
		fields["role"] = payload.Role
		fields["content_len"] = payload.ContentLen
		fields["queue_depth"] = payload.QueueDepth
		fields["hint_len"] = payload.HintLen
	case SubTurnSpawnPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["label"] = payload.Label
	case SubTurnEndPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["status"] = payload.Status
	case SubTurnResultDeliveredPayload:
		fields["target_channel"] = payload.TargetChannel
		fields["target_chat_id"] = payload.TargetChatID
		fields["content_len"] = payload.ContentLen
	case ErrorPayload:
		fields["stage"] = payload.Stage
		fields["error"] = payload.Message
	}

	logger.InfoCF("eventbus", fmt.Sprintf("Agent event: %s", evt.Kind.String()), fields)
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
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

// ReloadProviderAndConfig atomically swaps the provider and config with proper synchronization.
// It uses a context to allow timeout control from the caller.
// Returns an error if the reload fails or context is canceled.
func (al *AgentLoop) ReloadProviderAndConfig(
	ctx context.Context,
	provider providers.LLMProvider,
	cfg *config.Config,
) error {
	// Validate inputs
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Create new registry with updated config and provider
	// Wrap in defer/recover to handle any panics gracefully
	var registry *AgentRegistry
	var panicErr error
	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("panic during registry creation: %v", r)
				logger.ErrorCF("agent", "Panic during registry creation",
					map[string]any{"panic": r})
			}
			close(done)
		}()

		registry = NewAgentRegistry(cfg, provider)
	}()

	// Wait for completion or context cancellation
	select {
	case <-done:
		if registry == nil {
			if panicErr != nil {
				return fmt.Errorf("registry creation failed: %w", panicErr)
			}
			return fmt.Errorf("registry creation failed (nil result)")
		}
	case <-ctx.Done():
		return fmt.Errorf("context canceled during registry creation: %w", ctx.Err())
	}

	// Check context again before proceeding
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled after registry creation: %w", err)
	}

	// Ensure shared tools are re-registered on the new registry
	registerSharedTools(al, cfg, al.bus, registry, provider)

	// Atomically swap the config and registry under write lock
	// This ensures readers see a consistent pair
	al.mu.Lock()
	oldRegistry := al.registry

	// Store new values
	al.cfg = cfg
	al.registry = registry

	// Also update fallback chain with new config
	al.fallback = providers.NewFallbackChain(providers.NewCooldownTracker())

	al.mu.Unlock()

	al.hookRuntime.reset(al)
	configureHookManagerFromConfig(al.hooks, cfg)

	// Close old provider after releasing the lock
	// This prevents blocking readers while closing
	if oldProvider, ok := extractProvider(oldRegistry); ok {
		if stateful, ok := oldProvider.(providers.StatefulProvider); ok {
			// Give in-flight requests a moment to complete
			// Use a reasonable timeout that balances cleanup vs resource usage
			select {
			case <-time.After(100 * time.Millisecond):
				stateful.Close()
			case <-ctx.Done():
				// Context canceled, close immediately but log warning
				logger.WarnCF("agent", "Context canceled during provider cleanup, forcing close",
					map[string]any{"error": ctx.Err()})
				stateful.Close()
			}
		}
	}

	logger.InfoCF("agent", "Provider and config reloaded successfully",
		map[string]any{
			"model": cfg.Agents.Defaults.GetModelName(),
		})

	return nil
}

// GetRegistry returns the current registry (thread-safe)
func (al *AgentLoop) GetRegistry() *AgentRegistry {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.registry
}

// GetConfig returns the current config (thread-safe)
func (al *AgentLoop) GetConfig() *config.Config {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.cfg
}

// SetMediaStore injects a MediaStore for media lifecycle management.

func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s

	// Propagate store to send_file tools in all agents.
	registry := al.GetRegistry()
	registry.ForEachTool("send_file", func(t tools.Tool) {
		if sf, ok := t.(*tools.SendFileTool); ok {
			sf.SetMediaStore(s)
		}
	})
}

// SetTranscriber injects a voice transcriber for agent-level audio transcription.
func (al *AgentLoop) SetTranscriber(t voice.Transcriber) {
	al.transcriber = t
}

var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)

// transcribeAudioInMessage resolves audio media refs, transcribes them, and
// replaces audio annotations in msg.Content with the transcribed text.
// Returns the (possibly modified) message and true if audio was transcribed.
func (al *AgentLoop) transcribeAudioInMessage(ctx context.Context, msg bus.InboundMessage) (bus.InboundMessage, bool) {
	if al.transcriber == nil || al.mediaStore == nil || len(msg.Media) == 0 {
		return msg, false
	}

	// Transcribe each audio media ref in order.
	var transcriptions []string
	for _, ref := range msg.Media {
		path, meta, err := al.mediaStore.ResolveWithMeta(ref)
		if err != nil {
			logger.WarnCF("voice", "Failed to resolve media ref", map[string]any{"ref": ref, "error": err})
			continue
		}
		if !utils.IsAudioFile(meta.Filename, meta.ContentType) {
			continue
		}
		result, err := al.transcriber.Transcribe(ctx, path)
		if err != nil {
			logger.WarnCF("voice", "Transcription failed", map[string]any{"ref": ref, "error": err})
			transcriptions = append(transcriptions, "")
			continue
		}
		transcriptions = append(transcriptions, result.Text)
	}

	if len(transcriptions) == 0 {
		return msg, false
	}

	al.sendTranscriptionFeedback(ctx, msg.Channel, msg.ChatID, msg.MessageID, transcriptions)

	// Replace audio annotations sequentially with transcriptions.
	idx := 0
	newContent := audioAnnotationRe.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if idx >= len(transcriptions) {
			return match
		}
		text := transcriptions[idx]
		idx++
		return "[voice: " + text + "]"
	})

	// Append any remaining transcriptions not matched by an annotation.
	for ; idx < len(transcriptions); idx++ {
		newContent += "\n[voice: " + transcriptions[idx] + "]"
	}

	msg.Content = newContent
	return msg, true
}

// sendTranscriptionFeedback sends feedback to the user with the result of
// audio transcription if the option is enabled. It uses Manager.SendMessage
// which executes synchronously (rate limiting, splitting, retry) so that
// ordering with the subsequent placeholder is guaranteed.
func (al *AgentLoop) sendTranscriptionFeedback(
	ctx context.Context,
	channel, chatID, messageID string,
	validTexts []string,
) {
	if !al.cfg.Voice.EchoTranscription {
		return
	}
	if al.channelManager == nil {
		return
	}

	var nonEmpty []string
	for _, t := range validTexts {
		if t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}

	var feedbackMsg string
	if len(nonEmpty) > 0 {
		feedbackMsg = "Transcript: " + strings.Join(nonEmpty, "\n")
	} else {
		feedbackMsg = "No voice detected in the audio"
	}

	err := al.channelManager.SendMessage(ctx, bus.OutboundMessage{
		Channel:          channel,
		ChatID:           chatID,
		Content:          feedbackMsg,
		ReplyToMessageID: messageID,
	})
	if err != nil {
		logger.WarnCF("voice", "Failed to send transcription feedback", map[string]any{"error": err.Error()})
	}
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
	if err := al.ensureHooksInitialized(ctx); err != nil {
		return "", err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return "", err
	}

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

	response, err := al.processMessage(ctx, msg)
	if err != nil {
		return response, err
	}

	// If steering messages arrived during the LLM call, the initial
	// response is stale. Discard it and do a continuation turn that
	// includes the steering messages for a fresh response.
	steeringMsgs := al.dequeueSteeringMessagesForScopeWithFallback(sessionKey)
	if len(steeringMsgs) > 0 {
		agent := al.agentForSession(sessionKey)
		if agent == nil {
			agent = al.registry.GetDefaultAgent()
		}
		if agent != nil {
			contResp, contErr := al.continueWithSteeringMessages(
				ctx, agent, sessionKey, channel, chatID, steeringMsgs,
			)
			if contErr == nil && contResp != "" {
				return contResp, nil
			}
		}
	}

	return response, nil
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	ctx = tools.WithHeartbeatContext(ctx)
	ctx = tools.WithWebSearchQuota(ctx, research.DefaultHeartbeatSearchQuota)
	if err := al.ensureHooksInitialized(ctx); err != nil {
		return "", err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return "", err
	}

	agent := al.GetRegistry().GetDefaultAgent()
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

// resolveMessageRoute resolves the agent and routing info for an inbound message.
// It looks up the agent registry to determine which agent handles the message
// and resets the message tool context for the new round.
func (al *AgentLoop) resolveMessageRoute(msg bus.InboundMessage) (routing.ResolvedRoute, *AgentInstance, error) {
	registry := al.GetRegistry()
	route := registry.ResolveRoute(routing.RouteInput{
		Channel:    msg.Channel,
		AccountID:  msg.Metadata[metadataKeyAccountID],
		Peer:       extractPeer(msg),
		ParentPeer: extractParentPeer(msg),
		GuildID:    msg.Metadata[metadataKeyGuildID],
		TeamID:     msg.Metadata[metadataKeyTeamID],
	})

	agent, ok := registry.GetAgent(route.AgentID)
	if !ok {
		agent = registry.GetDefaultAgent()
	}
	if agent == nil {
		return route, nil, fmt.Errorf("no agent available for route (agent_id=%s)", route.AgentID)
	}

	// Reset message-tool state for this round
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(msg.Channel, msg.ChatID)
		}
	}

	logger.InfoCF("agent", "Routed message",
		map[string]any{
			"agent_id":    agent.ID,
			"session_key": route.SessionKey,
			"matched_by":  route.MatchedBy,
		})

	return route, agent, nil
}

// resolveScopeKey returns the session key to use: honors a pre-set key (from
// ProcessDirect/cron) over the route-resolved key.
func resolveScopeKey(route routing.ResolvedRoute, msgSessionKey string) string {
	if msgSessionKey != "" && strings.HasPrefix(msgSessionKey, sessionKeyAgentPrefix) {
		return msgSessionKey
	}
	return route.SessionKey
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

	// Transcribe audio in the message if a transcriber is configured.
	var hadAudio bool
	msg, hadAudio = al.transcribeAudioInMessage(ctx, msg)

	// For audio messages the placeholder was deferred by the channel.
	// Now that transcription (and optional feedback) is done, send it.
	if hadAudio && al.channelManager != nil {
		al.channelManager.SendPlaceholder(ctx, msg.Channel, msg.ChatID)
	}

	// Handle reply-based intervention for active tasks
	if response, handled := al.handleTaskIntervention(msg); handled {
		return response, nil
	}

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Notify listeners that a real user message arrived (e.g. reset heartbeat suppression)

	if al.OnUserMessage != nil {
		al.OnUserMessage()
	}

	// Expand fork-specific /skill, /use, and /plan commands
	expansionCompact, forcedSkills := al.expandForkCommands(&msg)

	// Check for commands (using default agent, before routing)
	if response, handled := al.handleCommand(ctx, msg, al.registry.GetDefaultAgent(), msg.SessionKey); handled {
		return response, nil
	}

	// Route to determine agent and session key
	route, agent, err := al.resolveMessageRoute(msg)
	if err != nil {
		return "", err
	}

	sessionKey := resolveScopeKey(route, msg.SessionKey)

	// Consume armed skill from a previous /use <skill> command
	armKey := msg.Channel + ":" + msg.ChatID
	if val, ok := al.pendingSkills.LoadAndDelete(sessionKey); ok {
		if skillName, ok := val.(string); ok {
			forcedSkills = append(forcedSkills, skillName)
		}
	} else if val, ok := al.pendingSkills.LoadAndDelete(armKey); ok {
		if skillName, ok := val.(string); ok {
			forcedSkills = append(forcedSkills, skillName)
		}
	}

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:        sessionKey,
		Channel:           msg.Channel,
		ChatID:            msg.ChatID,
		SenderID:          msg.SenderID,
		SenderDisplayName: msg.Sender.DisplayName,
		UserMessage:       msg.Content,
		ForcedSkills:      forcedSkills,
		Media:             msg.Media,
		HistoryMessage:    expansionCompact,
		DefaultResponse:   defaultResponse,
		EnableSummary:     true,
		SendResponse:      false,
		Background:        msg.Metadata["background"] == "true",
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

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

// callLLMWithRetry calls the LLM with streaming support, fallback chain,
// and retry logic for timeout and context window errors.
func (al *AgentLoop) callLLMWithRetry(
	ctx context.Context,
	agent *AgentInstance,
	messages *[]providers.Message,
	opts processOptions,
	toolDefs []providers.ToolDefinition,
	candidates []providers.FallbackCandidate,
	activeModel string,
	onChunk func(string, string),
	iteration int,
	scope ...turnEventScope,
) (*providers.LLMResponse, error) {
	llmOpts := map[string]any{
		"max_tokens":       agent.MaxTokens,
		"temperature":      agent.Temperature,
		"prompt_cache_key": agent.ID,
	}

	// parseThinkingLevel guarantees ThinkingOff for empty/unknown values,
	// so checking != ThinkingOff is sufficient.
	if agent.ThinkingLevel != ThinkingOff {
		if tc, ok := agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
			llmOpts["thinking_level"] = string(agent.ThinkingLevel)
		} else {
			logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
				map[string]any{"agent_id": agent.ID, "thinking_level": string(agent.ThinkingLevel)})
		}
	}

	doCall := func(ctx context.Context, p providers.LLMProvider, model string) (*providers.LLMResponse, error) {
		if sp, ok := p.(providers.StreamingProvider); ok && sp.CanStream() {
			streamCtx, streamCancel := context.WithCancel(ctx)
			defer streamCancel()
			ch, sErr := sp.ChatStream(streamCtx, *messages, toolDefs, model, llmOpts)
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
		return p.Chat(ctx, *messages, toolDefs, model, llmOpts)
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

	// Hook: pre-LLM state reporting (called via hooks in the caller)

	maxRetries := 2
	var response *providers.LLMResponse
	var err error

	for retry := 0; retry <= maxRetries; retry++ {
		response, err = callLLM()
		if err == nil {
			return response, nil
		}

		errMsg := strings.ToLower(err.Error())

		isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(errMsg, "deadline exceeded") ||
			strings.Contains(errMsg, "client.timeout") ||
			strings.Contains(errMsg, "timed out") ||
			strings.Contains(errMsg, "timeout exceeded")

		isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
			strings.Contains(errMsg, "context window") ||
			strings.Contains(errMsg, "maximum context length") ||
			strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "too many tokens") ||
			strings.Contains(errMsg, "max_tokens") ||
			strings.Contains(errMsg, "invalidparameter") ||
			strings.Contains(errMsg, "prompt is too long") ||
			strings.Contains(errMsg, "request too large"))

		// Helper to emit events if scope was provided
		emitRetryEvent := func(kind EventKind, payload any) {
			if len(scope) > 0 {
				meta := EventMeta{
					AgentID:    agent.ID,
					TurnID:     scope[0].turnID,
					SessionKey: opts.SessionKey,
					Iteration:  iteration,
				}
				al.emitEvent(kind, meta, payload)
			}
		}

		if isTimeoutError && retry < maxRetries {
			backoff := time.Duration(retry+1) * 5 * time.Second
			logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
				"error":   err.Error(),
				"retry":   retry,
				"backoff": backoff.String(),
			})
			emitRetryEvent(EventKindLLMRetry, LLMRetryPayload{
				Attempt: retry + 1, MaxRetries: maxRetries,
				Reason: "timeout", Error: err.Error(), Backoff: backoff,
			})
			time.Sleep(backoff)
			continue
		}

		if isContextError && retry < maxRetries {
			logger.WarnCF("agent", "Context window error detected, attempting compression", map[string]any{
				"error": err.Error(),
				"retry": retry,
			})
			emitRetryEvent(EventKindLLMRetry, LLMRetryPayload{
				Attempt: retry + 1, MaxRetries: maxRetries,
				Reason: "context_limit", Error: err.Error(),
			})
			if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: "Context window exceeded. Compressing history and retrying...",
				})
			}
			prevCount := len(*messages)
			al.forceCompression(agent, opts.SessionKey)
			newHistory := agent.Sessions.GetHistory(opts.SessionKey)
			newSummary := agent.Sessions.GetSummary(opts.SessionKey)
			*messages = agent.ContextBuilder.BuildMessages(
				newHistory, newSummary, "",
				nil, opts.Channel, opts.ChatID,
				opts.SenderID, opts.SenderDisplayName,
			)
			emitRetryEvent(EventKindContextCompress, ContextCompressPayload{
				Reason:            ContextCompressReasonRetry,
				DroppedMessages:   prevCount - len(*messages),
				RemainingMessages: len(*messages),
			})
			continue
		}
		break
	}
	return nil, err
}

// cleanLLMResponse handles repetition detection, think block stripping,
// and XML tool call extraction on the raw LLM response.
func (al *AgentLoop) cleanLLMResponse(
	ctx context.Context,
	response *providers.LLMResponse,
	messages *[]providers.Message,
	agent *AgentInstance,
	iteration int,
	toolDefs []providers.ToolDefinition,
	candidates []providers.FallbackCandidate,
	activeModel string,
	onChunk func(string, string),
) *providers.LLMResponse {
	if response.FinishReason == "repetition_detected" ||
		(len(response.ToolCalls) == 0 && utils.DetectRepetitionLoop(response.Content)) {
		logger.WarnCF("agent", "Repetition loop detected in LLM response, retrying",
			map[string]any{
				"agent_id":       agent.ID,
				"iteration":      iteration,
				"finish_reason":  response.FinishReason,
				"content_length": len(response.Content),
			})

		savedMsgs := *messages
		*messages = append(append([]providers.Message(nil), *messages...),
			providers.Message{
				Role:    "user",
				Content: "[System] Your previous response contained degenerate repetition and was discarded. Please respond normally without repeating yourself.",
			})

		retryResp, retryErr := al.callLLMWithRetry(ctx, agent, messages, processOptions{},
			toolDefs, candidates, activeModel, onChunk, iteration)
		*messages = savedMsgs

		if retryErr == nil {
			response = retryResp
		}
		if utils.DetectRepetitionLoop(response.Content) {
			logger.ErrorCF("agent", "Repetition persists after retry, returning empty",
				map[string]any{"agent_id": agent.ID})
			response.Content = ""
		}
	}

	response.Content = utils.StripThinkBlocks(response.Content)
	if len(response.ToolCalls) == 0 {
		if xmlCalls := providers.ExtractXMLToolCalls(response.Content); len(xmlCalls) > 0 {
			response.ToolCalls = xmlCalls
		}
	}
	response.Content = providers.StripXMLToolCalls(response.Content)
	return response
}

// buildAssistantMessage constructs the assistant message with tool calls.
func buildAssistantMessage(response *providers.LLMResponse, toolCalls []providers.ToolCall) providers.Message {
	msg := providers.Message{
		Role:             "assistant",
		Content:          response.Content,
		ReasoningContent: response.ReasoningContent,
	}
	for _, tc := range toolCalls {
		extraContent := tc.ExtraContent
		thoughtSignature := ""
		if tc.Function != nil {
			thoughtSignature = tc.Function.ThoughtSignature
		}
		msg.ToolCalls = append(msg.ToolCalls, providers.ToolCall{
			ID:        tc.ID,
			Type:      "function",
			Name:      tc.Name,
			Arguments: tc.Arguments,
			Function: &providers.FunctionCall{
				Name:             tc.Name,
				Arguments:        tc.Arguments,
				ThoughtSignature: thoughtSignature,
			},
			ExtraContent:     extraContent,
			ThoughtSignature: thoughtSignature,
		})
	}
	return msg
}

// publishToolMedia publishes media refs from a tool result as outbound media.
func (al *AgentLoop) publishToolMedia(ctx context.Context, result *tools.ToolResult, opts processOptions) {
	parts := make([]bus.MediaPart, 0, len(result.Media))
	for _, ref := range result.Media {
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

// isNativeSearchProvider reports whether the given LLM provider implements
// NativeSearchCapable and returns true for SupportsNativeSearch.
func isNativeSearchProvider(p providers.LLMProvider) bool {
	if ns, ok := p.(providers.NativeSearchCapable); ok {
		return ns.SupportsNativeSearch()
	}
	return false
}

// filterClientWebSearch returns a copy of tools with the client-side
// web_search tool removed. Used when native provider search is preferred.
func filterClientWebSearch(tools []providers.ToolDefinition) []providers.ToolDefinition {
	result := make([]providers.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if strings.EqualFold(t.Function.Name, "web_search") {
			continue
		}
		result = append(result, t)
	}
	return result
}

// Helper to extract provider from registry for cleanup
func extractProvider(registry *AgentRegistry) (providers.LLMProvider, bool) {
	if registry == nil {
		return nil, false
	}
	// Get any agent to access the provider
	defaultAgent := registry.GetDefaultAgent()
	if defaultAgent == nil {
		return nil, false
	}
	return defaultAgent.Provider, true
}

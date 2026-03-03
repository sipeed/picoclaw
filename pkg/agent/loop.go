// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
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
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AgentLoop struct {
	bus            *bus.MessageBus
	cfg            *config.Config
	registry       *AgentRegistry
	state          *state.Manager
	running        atomic.Bool
	msgSeqId       atomic.Uint64
	summarizing    sync.Map
	fallback       *providers.FallbackChain
	channelManager *channels.Manager
	mediaStore     media.MediaStore
	// Phase 3 infrastructure (M1-M4)
	turnStore      *TurnStore             // per-workspace turns.db
	activeCtx      *ActiveContextStore    // per channel:chatID context
	memoryDigest   *MemoryDigestWorker    // background memory distillation
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string // Session identifier for history/context
	Channel         string // Target channel for tool execution
	ChatID          string // Target chat ID for tool execution
	UserMessage     string // User message content (may include prefix)
	DefaultResponse string // Response when LLM returns empty
	EnableSummary   bool   // Whether to trigger summarization
	SendResponse    bool   // Whether to send response via bus
	NoHistory       bool   // If true, don't load session history (for heartbeat)
	MsgSeqId        uint64 // Global message sequence number
}

const defaultResponse = "I've completed processing but have no response to give. Increase `max_tool_iterations` in config.json."

func NewAgentLoop(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	provider providers.LLMProvider,
) *AgentLoop {
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

	// Initialise Phase 3 infrastructure.
	activeCtxStore := NewActiveContextStore()
	var ts *TurnStore
	var digestWorker *MemoryDigestWorker
	if defaultAgent != nil {
		var tsErr error
		ts, tsErr = NewTurnStore(defaultAgent.Workspace)
		if tsErr != nil {
			logger.ErrorCF("agent", "Failed to create TurnStore", map[string]any{"error": tsErr.Error()})
		} else {
			mem := defaultAgent.ContextBuilder.GetMemory()
			digestModel := cfg.Agents.Defaults.GetDigestModel()
			digestWorker = NewMemoryDigestWorker(ts, mem, provider, digestModel)
		}
	}

	return &AgentLoop{
		bus:          msgBus,
		cfg:          cfg,
		registry:     registry,
		state:        stateManager,
		summarizing:  sync.Map{},
		fallback:     fallbackChain,
		turnStore:    ts,
		activeCtx:    activeCtxStore,
		memoryDigest: digestWorker,
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
			logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{"error": err.Error()})
		} else if searchTool != nil {
			agent.Tools.Register(searchTool)
		}
		fetchTool, err := tools.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy, cfg.Tools.Web.FetchLimitBytes)
		if err != nil {
			logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
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

		// Spawn tool with allowlist checker
		subagentManager := tools.NewSubagentManager(provider, agent.Model, agent.Workspace, msgBus)
		subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)
		spawnTool := tools.NewSpawnTool(subagentManager)
		currentAgentID := agentID
		spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
			return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
		})
		agent.Tools.Register(spawnTool)
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	// Load persisted Active Context.
	if al.activeCtx != nil {
		defaultAgent := al.registry.GetDefaultAgent()
		if defaultAgent != nil {
			acPath := activeContextPath(defaultAgent.Workspace)
			if err := al.activeCtx.Load(acPath); err != nil {
				logger.WarnCF("agent", "Failed to load active context", map[string]any{"error": err.Error()})
			}
		}
	}

	// Start MemoryDigest background worker.
	if al.memoryDigest != nil {
		al.memoryDigest.Start(ctx)
	}

	// Initialize MCP servers for all agents
	if al.cfg.Tools.MCP.Enabled {
		mcpManager := mcp.NewManager()
		defaultAgent := al.registry.GetDefaultAgent()
		var workspacePath string
		if defaultAgent != nil && defaultAgent.Workspace != "" {
			workspacePath = defaultAgent.Workspace
		} else {
			workspacePath = al.cfg.WorkspacePath()
		}

		if err := mcpManager.LoadFromMCPConfig(ctx, al.cfg.Tools.MCP, workspacePath); err != nil {
			logger.WarnCF("agent", "Failed to load MCP servers, MCP tools will not be available",
				map[string]any{
					"error": err.Error(),
				})
		} else {
			// Ensure MCP connections are cleaned up on exit, only if initialization succeeded
			defer func() {
				if err := mcpManager.Close(); err != nil {
					logger.ErrorCF("agent", "Failed to close MCP manager",
						map[string]any{
							"error": err.Error(),
						})
				}
			}()

			// Register MCP tools for all agents
			servers := mcpManager.GetServers()
			uniqueTools := 0
			totalRegistrations := 0
			agentIDs := al.registry.ListAgentIDs()
			agentCount := len(agentIDs)

			for serverName, conn := range servers {
				uniqueTools += len(conn.Tools)
				for _, tool := range conn.Tools {
					for _, agentID := range agentIDs {
						agent, ok := al.registry.GetAgent(agentID)
						if !ok {
							continue
						}
						mcpTool := tools.NewMCPTool(mcpManager, serverName, tool)
						agent.Tools.Register(mcpTool)
						totalRegistrations++
						logger.DebugCF("agent", "Registered MCP tool",
							map[string]any{
								"agent_id": agentID,
								"server":   serverName,
								"tool":     tool.Name,
								"name":     mcpTool.Name(),
							})
					}
				}
			}
			logger.InfoCF("agent", "MCP tools registered successfully",
				map[string]any{
					"server_count":        len(servers),
					"unique_tools":        uniqueTools,
					"total_registrations": totalRegistrations,
					"agent_count":         agentCount,
				})
		}
	}

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			// Process message
			func() {
				// TODO: Re-enable media cleanup after inbound media is properly consumed by the agent.
				// Currently disabled because files are deleted before the LLM can access their content.
				// defer func() {
				// 	if al.mediaStore != nil && msg.MediaScope != "" {
				// 		if releaseErr := al.mediaStore.ReleaseAll(msg.MediaScope); releaseErr != nil {
				// 			logger.WarnCF("agent", "Failed to release media", map[string]any{
				// 				"scope": msg.MediaScope,
				// 				"error": releaseErr.Error(),
				// 			})
				// 		}
				// 	}
				// }()

				response, err := al.processMessage(ctx, msg)
				if err != nil {
					response = fmt.Sprintf("Error processing message: %v", err)
				}

				if response != "" {
					// Check if the message tool already sent a response during this round.
					// If so, skip publishing to avoid duplicate messages to the user.
					// Use default agent's tools to check (message tool is shared).
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
						al.bus.PublishOutbound(ctx, bus.OutboundMessage{
							Channel: msg.Channel,
							ChatID:  msg.ChatID,
							Content: response,
						})
						logger.InfoCF("agent", "Published outbound response",
							map[string]any{
								"channel":     msg.Channel,
								"chat_id":     msg.ChatID,
								"content_len": len(response),
							})
					} else {
						logger.DebugCF(
							"agent",
							"Skipped outbound (message tool already sent)",
							map[string]any{"channel": msg.Channel},
						)
					}
				}
			}()
		}
	}

	return nil
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
	// Flush Active Context to disk.
	if al.activeCtx != nil {
		defaultAgent := al.registry.GetDefaultAgent()
		if defaultAgent != nil {
			acPath := activeContextPath(defaultAgent.Workspace)
			if err := al.activeCtx.Flush(acPath); err != nil {
				logger.WarnCF("agent", "Failed to flush active context", map[string]any{"error": err.Error()})
			}
		}
	}
	// Close TurnStore.
	if al.turnStore != nil {
		if err := al.turnStore.Close(); err != nil {
			logger.WarnCF("agent", "Failed to close turn store", map[string]any{"error": err.Error()})
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
	// Wire agent info into all agent Runtimes so /show, /list, /switch work.
	for _, id := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(id); ok && agent != nil && agent.Reflector != nil {
			agent.Reflector.SetAgentInfo(al.registry, cm)
		}
	}
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

func (al *AgentLoop) ProcessDirect(
	ctx context.Context,
	content, sessionKey string,
) (string, error) {
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
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(
	ctx context.Context,
	content, channel, chatID string,
) (string, error) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for heartbeat")
	}
	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true, // Don't load session history for heartbeat
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
	logger.InfoCF(
		"agent",
		fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]any{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		},
	)

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Check for runtime commands (e.g. /memory, /cot, /runtime, /show, /list, /switch).
	if agent := al.registry.GetDefaultAgent(); agent != nil && agent.Reflector != nil {
		if response, handled := agent.Reflector.HandleCommand(msg.Content, agent.ContextBuilder.GetMemory()); handled {
			return response, nil
		}
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

	// Use routed session key, but honor pre-set agent-scoped keys (for ProcessDirect/cron)
	sessionKey := route.SessionKey
	if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
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
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
		SendResponse:    false,
	})
}

func (al *AgentLoop) processSystemMessage(
	ctx context.Context,
	msg bus.InboundMessage,
) (string, error) {
	if msg.Channel != "system" {
		return "", fmt.Errorf(
			"processSystemMessage called with non-system message channel: %s",
			msg.Channel,
		)
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
	if agent == nil {
		return "", fmt.Errorf("no default agent for system message")
	}

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

// runAgentLoop is the core message processing logic.
func (al *AgentLoop) runAgentLoop(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
) (string, error) {
	seq := al.msgSeqId.Add(1)
	opts.MsgSeqId = seq

	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF(
					"agent",
					"Failed to record last channel",
					map[string]any{"error": err.Error()},
				)
			}
		}
	}

	// 1. Update tool contexts
	al.updateToolContexts(agent, opts.Channel, opts.ChatID)

	// 2. Analyse intent + build Phase 2 messages.
	//
	// Two paths:
	//   A. Instant Memory (when Analyser + TurnStore are ready):
	//      - Phase 1 analyses intent/tags
	//      - BuildInstantMemory selects relevant historical turns from TurnStore
	//      - BuildPhase2Messages assembles KV cache friendly message array
	//   B. Legacy SessionManager (fallback):
	//      - Uses Session history directly via ContextBuilder.BuildMessages
	//
	var analyseResult AnalyseResult
	var messages []providers.Message
	channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)

	useInstantMemory := agent.Analyser != nil && al.turnStore != nil && !opts.NoHistory && opts.UserMessage != ""

	if useInstantMemory {
		// --- Path A: Instant Memory ---

		// Phase 1: analyse intent + tags.
		var actCtx *ActiveContext
		if al.activeCtx != nil {
			actCtx = al.activeCtx.Get(channelKey)
		}
		analyseResult = agent.Analyser.Analyse(ctx, opts.UserMessage, agent.ContextBuilder.GetMemory(), actCtx)

		// Build system prompt from ContextBuilder (cached static + dynamic context).
		staticPrompt := agent.ContextBuilder.BuildSystemPromptWithCache()
		dynamicCtx := agent.ContextBuilder.buildDynamicContext(opts.Channel, opts.ChatID)
		systemPrompt := staticPrompt + "\n\n---\n\n" + dynamicCtx

		// Enrich system prompt with CoT strategy.
		if analyseResult.CotPrompt != "" {
			systemPrompt += "\n\n---\n\n## Thinking Strategy\n\n" + analyseResult.CotPrompt
		}

		// Select relevant turns from TurnStore.
		cfg := DefaultInstantMemoryCfg(agent.ContextWindow)
		instantTurns := BuildInstantMemory(al.turnStore, analyseResult.Tags, channelKey, cfg)

		// Get long-term memory by tags.
		longTermMemory := analyseResult.MemoryContext

		// Assemble Phase 2 messages in KV cache friendly order.
		messages = BuildPhase2Messages(
			systemPrompt,
			longTermMemory,
			instantTurns,
			opts.UserMessage,
			cfg.HighScoreThreshold,
		)

		logger.InfoCF("agent", "Phase 2 messages built via instant memory",
			map[string]any{
				"seq":            seq,
				"agent_id":       agent.ID,
				"intent":         analyseResult.Intent,
				"tags":           analyseResult.Tags,
				"instant_turns":  len(instantTurns),
				"total_messages": len(messages),
				"has_cot":        analyseResult.CotPrompt != "",
				"has_memories":   longTermMemory != "",
			})
	} else {
		// --- Path B: Legacy SessionManager ---
		var history []providers.Message
		var summary string
		if !opts.NoHistory {
			history = agent.Sessions.GetHistory(opts.SessionKey)
			summary = agent.Sessions.GetSummary(opts.SessionKey)
		}
		messages = agent.ContextBuilder.BuildMessages(
			history,
			summary,
			opts.UserMessage,
			nil,
			opts.Channel,
			opts.ChatID,
		)

		// Optional Phase 1 enrichment (when Analyser exists but TurnStore not ready).
		if agent.Analyser != nil && !opts.NoHistory && opts.UserMessage != "" {
			var actCtx *ActiveContext
			if al.activeCtx != nil {
				actCtx = al.activeCtx.Get(channelKey)
			}
			analyseResult = agent.Analyser.Analyse(ctx, opts.UserMessage, agent.ContextBuilder.GetMemory(), actCtx)

			var enrichment strings.Builder
			if analyseResult.CotPrompt != "" {
				enrichment.WriteString("\n\n---\n\n## Thinking Strategy\n\n")
				enrichment.WriteString(analyseResult.CotPrompt)
			}
			if analyseResult.MemoryContext != "" {
				enrichment.WriteString("\n\n---\n\n# Contextual Memories (pre-analysed)\n\n")
				enrichment.WriteString(analyseResult.MemoryContext)
			}

			if enrichment.Len() > 0 && len(messages) > 0 && messages[0].Role == "system" {
				messages[0].Content += enrichment.String()
				if len(messages[0].SystemParts) > 0 {
					enrichBlock := providers.ContentBlock{
						Type: "text",
						Text: enrichment.String(),
					}
					messages[0].SystemParts = append(messages[0].SystemParts, enrichBlock)
				}
				logger.InfoCF("agent", "Pre-LLM enriched context (legacy path)",
					map[string]any{
						"seq":          seq,
						"agent_id":     agent.ID,
						"intent":       analyseResult.Intent,
						"tags":         analyseResult.Tags,
						"has_memories": analyseResult.MemoryContext != "",
						"has_cot":      analyseResult.CotPrompt != "",
					})
			}
		}
	}

	// 3. Save user message to session (kept for /memory, /show debug commands).
	agent.Sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)

	// 4. Run LLM iteration loop
	finalContent, iteration, toolRecords, err := al.runLLMIteration(ctx, agent, messages, opts)
	if err != nil {
		return "", err
	}

	// If last tool had ForUser content and we already sent it, we might not need to send final response
	// This is controlled by the tool's Silent flag and ForUser content

	// 5. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 6. Save final assistant message to session
	agent.Sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	agent.Sessions.Save(opts.SessionKey)

	// Build the runtime input used by Phase 3 stages.
	runtimeInput := RuntimeInput{
		UserMessage:    opts.UserMessage,
		AssistantReply: finalContent,
		Intent:         analyseResult.Intent,
		Tags:           analyseResult.Tags,
		CotPrompt:      analyseResult.CotPrompt,
		ToolCalls:      toolRecords,
		Iterations:     iteration,
		ChannelKey:     channelKey,
	}

	// 6.5. Phase 3 — Synchronous part (< 2ms): score + Active Context update.
	// MUST run before PublishOutbound so the next turn's Phase 1 sees fresh context.
	if agent.Reflector != nil && opts.UserMessage != "" {
		score := agent.Reflector.SyncPhase3(runtimeInput)
		runtimeInput.Score = score

		// Update Active Context for this channel.
		if al.activeCtx != nil {
			al.activeCtx.Update(channelKey, runtimeInput)
		}
	}

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 8. Optional: send response via bus (user receives reply here).
	if opts.SendResponse {
		al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalContent,
		})
	}

	// 6.6. Phase 3 — Async part: persist TurnRecord, run processors.
	// Runs AFTER PublishOutbound to not delay the user response.
	if agent.Reflector != nil && opts.UserMessage != "" {
		agent.Reflector.AsyncPhase3(runtimeInput, agent.ContextBuilder.GetMemory(), al.turnStore, al.activeCtx)
	}

	// 9. Log response
	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]any{
			"seq":          seq,
			"agent_id":     agent.ID,
			"session_key":  opts.SessionKey,
			"iterations":   iteration,
			"final_length": len(finalContent),
		})

	return finalContent, nil
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
	sb.WriteString(
		"Provide a concise summary of this conversation segment, preserving core context and key points.\n",
	)
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

// activeContextPath returns the full path for the active_context.json file
// stored inside the workspace directory.
func activeContextPath(workspace string) string {
	return filepath.Join(workspace, "active_context.json")
}


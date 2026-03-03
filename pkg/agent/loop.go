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
	"unicode/utf8"

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
	summarizing    sync.Map
	fallback       *providers.FallbackChain
	channelManager *channels.Manager
	mediaStore     media.MediaStore

	// Legacy interrupt handling (to be deprecated)
	interruptHandler InterruptHandler // Interrupt handler for dynamic task management
	taskManager      *TaskManager     // Task manager for concurrent task tracking (Phase 2)

	// New steering architecture (nanobot-inspired)
	enableSteering    bool                            // Opt-in flag for steering feature
	interruptCheckers map[string]*InterruptionChecker // Per-session interrupt queues
	checkersMu        sync.RWMutex                    // Protects interruptCheckers map
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

	al := &AgentLoop{
		bus:         msgBus,
		cfg:         cfg,
		registry:    registry,
		state:       stateManager,
		summarizing: sync.Map{},
		fallback:    fallbackChain,

		// New steering architecture (nanobot-inspired)
		enableSteering:    cfg.Agents.Defaults.EnableSteering,
		interruptCheckers: make(map[string]*InterruptionChecker),
	}

	// Legacy components (DEPRECATED - only initialized when new steering is disabled)
	if !cfg.Agents.Defaults.EnableSteering {
		// Legacy interrupt handler (Phase 1.5)
		al.interruptHandler = NewBusInterruptHandler(msgBus, DefaultInterruptionConfig())

		// Legacy TaskManager (Phase 2)
		maxConcurrent := cfg.Agents.Defaults.MaxConcurrentTasks
		if maxConcurrent < 0 {
			maxConcurrent = 0 // Ensure no negative values, 0 = unlimited
		}
		al.taskManager = NewTaskManager(maxConcurrent)
	}

	return al
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

	// Legacy: Start task cleanup and steering loop only when new steering is disabled
	if !al.enableSteering {
		// Phase 2: Start task cleanup goroutine
		go al.runTaskCleanup(ctx)

		// Phase 2: Start steering loop if enabled
		if al.cfg.Agents.Defaults.EnableSteeringLoop {
			go al.runSteeringLoop(ctx)
		}
	}

	for al.running.Load() {
		select {
		case <-ctx.Done():
			// Legacy: Wait for running tasks to complete (only if using TaskManager)
			if !al.enableSteering && al.taskManager != nil {
				al.waitForRunningTasks(5 * time.Second)
			}
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			// ===== NEW: Steering Architecture - Check for active session =====
			if al.enableSteering {
				// Get session key for this message (needs routing resolution)
				sessionKey := al.getSessionKeyForMessage(msg)

				// Check if this session has an active checker (task is running)
				if al.hasActiveChecker(sessionKey) {
					// Session is active, signal interruption instead of creating new task
					checker := al.getOrCreateChecker(sessionKey)
					signaled := checker.Signal(msg)

					if signaled {
						logger.InfoCF("agent", "Steering: signaled interruption for active session",
							map[string]any{
								"session_key":     sessionKey,
								"channel":         msg.Channel,
								"chat_id":         msg.ChatID,
								"content_preview": utils.Truncate(msg.Content, 60),
							})
						continue // Don't process as new message
					} else {
						// Grace period expired, treat as new message
						logger.WarnCF("agent", "Steering: session checker exists but grace period expired, processing as new message",
							map[string]any{
								"session_key": sessionKey,
								"channel":     msg.Channel,
								"chat_id":     msg.ChatID,
							})
						// Fall through to process as new message
					}
				}
			}

			// Phase 2: Process message asynchronously
			go func(msg bus.InboundMessage) {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorCF("agent", "Panic in message processing",
							map[string]any{"error": r, "channel": msg.Channel, "chat_id": msg.ChatID})
					}
				}()

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
					// Phase 2: Don't send error message if task was canceled
					// (user already received confirmation from /stop command)
					if errors.Is(err, context.Canceled) {
						logger.InfoCF("agent", "Task canceled, skipping error response",
							map[string]any{
								"channel": msg.Channel,
								"chat_id": msg.ChatID,
							})
						return // Silent return on cancellation
					}
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
			}(msg)
		}
	}

	return nil
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
}

// runTaskCleanup periodically cleans up old completed tasks (Phase 2)
func (al *AgentLoop) runTaskCleanup(ctx context.Context) {
	// Get cleanup interval from config, default to 5 minutes
	intervalMins := al.cfg.Agents.Defaults.TaskCleanupIntervalMins
	if intervalMins <= 0 {
		intervalMins = 5
	}

	// Get retention time from config, default to 1 hour
	retentionHours := al.cfg.Agents.Defaults.TaskRetentionHours
	if retentionHours <= 0 {
		retentionHours = 1
	}

	ticker := time.NewTicker(time.Duration(intervalMins) * time.Minute)
	defer ticker.Stop()

	logger.InfoCF("agent", "Task cleanup loop started",
		map[string]any{
			"interval_mins":   intervalMins,
			"retention_hours": retentionHours,
		})

	for {
		select {
		case <-ctx.Done():
			logger.InfoCF("agent", "Task cleanup loop stopped", nil)
			return
		case <-ticker.C:
			removed := al.taskManager.Cleanup(time.Duration(retentionHours) * time.Hour)
			if removed > 0 {
				logger.DebugCF("agent", "Cleaned up old tasks",
					map[string]any{"removed": removed})
			}
		}
	}
}

// waitForRunningTasks waits for all running tasks to complete or timeout (Phase 2)
func (al *AgentLoop) waitForRunningTasks(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		tasks := al.taskManager.GetRunningTasks()
		if len(tasks) == 0 {
			return
		}

		if time.Now().After(deadline) {
			logger.WarnCF("agent", "Timeout waiting for tasks, some may be abandoned",
				map[string]any{"running_tasks": len(tasks)})
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// runSteeringLoop monitors for interrupt signals and cancels tasks (Phase 2 Step 5)
func (al *AgentLoop) runSteeringLoop(ctx context.Context) {
	// Get interval from config, default to 500ms
	intervalMs := al.cfg.Agents.Defaults.SteeringLoopIntervalMs
	if intervalMs <= 0 {
		intervalMs = 500
	}

	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	logger.InfoCF("agent", "Steering loop started",
		map[string]any{"interval_ms": intervalMs})

	for {
		select {
		case <-ctx.Done():
			logger.InfoCF("agent", "Steering loop stopped", nil)
			return
		case <-ticker.C:
			al.checkAndHandleInterrupts(ctx)
		}
	}
}

// checkAndHandleInterrupts checks for interrupt signals and handles them (Phase 2 Step 5)
func (al *AgentLoop) checkAndHandleInterrupts(ctx context.Context) {
	if al.interruptHandler == nil {
		return
	}

	// Check for interruption signal
	signal, err := al.interruptHandler.CheckInterruption(ctx)
	if err != nil {
		logger.WarnCF("agent", "Interrupt check failed",
			map[string]any{"error": err.Error()})
		return
	}

	if signal == nil || signal.Priority < 8 {
		return // No interrupt or priority not high enough
	}

	// High-priority interrupt detected
	logger.InfoCF("agent", "High-priority interrupt detected by steering loop",
		map[string]any{
			"type":     signal.Type,
			"priority": signal.Priority,
			"source":   signal.Source,
		})

	// If it's a user message interrupt, check if we need to cancel existing tasks
	if signal.Type == "user_message" {
		if msg, ok := signal.Data.(bus.InboundMessage); ok {
			// Cancel all running tasks for the same session
			canceled := al.taskManager.CancelAllTasksForSession(msg.Channel, msg.ChatID)

			if canceled > 0 {
				logger.InfoCF("agent", "Canceled tasks for new high-priority message",
					map[string]any{
						"channel":        msg.Channel,
						"chat_id":        msg.ChatID,
						"canceled_count": canceled,
					})

				// Send notification to user
				al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: "Previous task interrupted. Processing your new request...",
				})
			}

			// Put the interrupt message back in the queue for normal processing
			go func() {
				pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				al.bus.PublishInbound(pubCtx, msg)
			}()
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

// ===== Steering Architecture: InterruptionChecker Management =====

// getOrCreateChecker gets or creates an interruption checker for a session.
// Thread-safe with double-checked locking pattern.
func (al *AgentLoop) getOrCreateChecker(sessionKey string) *InterruptionChecker {
	// Fast path: read lock
	al.checkersMu.RLock()
	checker, exists := al.interruptCheckers[sessionKey]
	al.checkersMu.RUnlock()

	if exists {
		return checker
	}

	// Slow path: write lock
	al.checkersMu.Lock()
	defer al.checkersMu.Unlock()

	// Double-check after acquiring write lock
	if checker, exists := al.interruptCheckers[sessionKey]; exists {
		return checker
	}

	// Create new checker
	checker = NewInterruptionChecker()
	al.interruptCheckers[sessionKey] = checker

	logger.DebugCF("agent", "Created interruption checker for session",
		map[string]any{"session_key": sessionKey})

	return checker
}

// formatInterruptionInjection formats pending interruption messages for injection into conversation.
// This follows nanobot's pattern of providing context to the LLM about the interruption.
func formatInterruptionInjection(pending []bus.InboundMessage) string {
	if len(pending) == 0 {
		return ""
	}

	var combined strings.Builder
	for i, msg := range pending {
		if i > 0 {
			combined.WriteString("\n\n---\n\n")
		}
		combined.WriteString(msg.Content)
	}

	injection := "[The user just sent a new message while you were working. " +
		"Read it and decide: continue current work, switch to the new request, or address both.]\n\n" +
		combined.String()

	return injection
}

// removeChecker removes a checker when session completes.
// This prevents memory leaks for long-running processes.
func (al *AgentLoop) removeChecker(sessionKey string) {
	al.checkersMu.Lock()
	defer al.checkersMu.Unlock()

	if _, exists := al.interruptCheckers[sessionKey]; exists {
		delete(al.interruptCheckers, sessionKey)
		logger.DebugCF("agent", "Removed interruption checker for session",
			map[string]any{"session_key": sessionKey})
	}
}

// hasActiveChecker checks if a session has an active interruption checker.
// This indicates the session is currently processing a message.
func (al *AgentLoop) hasActiveChecker(sessionKey string) bool {
	al.checkersMu.RLock()
	defer al.checkersMu.RUnlock()
	_, exists := al.interruptCheckers[sessionKey]
	return exists
}

// getSessionKeyForMessage resolves the session key for a message using routing logic.
// This is needed to check if the session has an active task before creating a new one.
func (al *AgentLoop) getSessionKeyForMessage(msg bus.InboundMessage) string {
	// If message already has an agent-scoped session key, use it
	if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
		return msg.SessionKey
	}

	// Otherwise, resolve via routing
	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel:    msg.Channel,
		AccountID:  msg.Metadata["account_id"],
		Peer:       extractPeer(msg),
		ParentPeer: extractParentPeer(msg),
		GuildID:    msg.Metadata["guild_id"],
		TeamID:     msg.Metadata["team_id"],
	})

	return route.SessionKey
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
}

// SetMediaStore injects a MediaStore for media lifecycle management.
func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s
}

// SetInterruptHandler sets or replaces the interrupt handler for the agent loop.
// This allows dynamic configuration of interruption behavior at runtime.
func (al *AgentLoop) SetInterruptHandler(handler InterruptHandler) {
	al.interruptHandler = handler
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

	// New steering architecture: Direct processing without task management
	if al.enableSteering {
		return al.processMessageDirect(ctx, msg)
	}

	// Legacy: Phase 2 task management
	priority := 5 // Default priority
	if busHandler, ok := al.interruptHandler.(*BusInterruptHandler); ok {
		priority = busHandler.calculatePriority(msg)
	}

	task := NewTask(msg, priority)
	if err := al.taskManager.AddTask(task); err != nil {
		return "", fmt.Errorf("failed to add task: %w", err)
	}

	// Start task (synchronous mode for now - Phase 2 Step 2)
	if err := al.taskManager.StartTask(task.ID, ctx); err != nil {
		return "", fmt.Errorf("failed to start task: %w", err)
	}

	// Use task's context for cancellation support
	taskCtx := task.Context()

	// Defer task completion/failure to ensure it's always updated
	defer func() {
		// Check if task is still running (not already completed/failed/canceled)
		if taskObj, exists := al.taskManager.GetTask(task.ID); exists {
			if taskObj.Status == TaskStatusRunning {
				al.taskManager.CompleteTask(task.ID)
			}
		}
	}()

	// Delegate to helper function for actual processing
	response, err := al.processMessageWithTask(taskCtx, task, msg)
	// Update task status based on result
	if err != nil {
		al.taskManager.FailTask(task.ID, err)
		return "", err
	}

	// Task will be completed by defer
	return response, nil
}

// processMessageDirect handles message processing for new steering architecture (no task management)
func (al *AgentLoop) processMessageDirect(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
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

	// Reset message-tool state for this round
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(msg.Channel, msg.ChatID)
		}
	}

	// Use routed session key, but honor pre-set agent-scoped keys
	sessionKey := route.SessionKey
	if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
		sessionKey = msg.SessionKey
	}

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:    sessionKey,
		Channel:       msg.Channel,
		ChatID:        msg.ChatID,
		UserMessage:   msg.Content,
		EnableSummary: true,
		SendResponse:  false,
	})
}

// processMessageWithTask handles the actual message processing logic with task context (LEGACY)
func (al *AgentLoop) processMessageWithTask(ctx context.Context, task *Task, msg bus.InboundMessage) (string, error) {
	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Check for commands
	if response, handled := al.handleCommand(ctx, msg); handled {
		// Mark task as a command task (for filtering in cancellation counts)
		task.Metadata["is_command"] = true
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

	// Reset message-tool state for this round
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(msg.Channel, msg.ChatID)
		}
	}

	// Use routed session key, but honor pre-set agent-scoped keys
	sessionKey := route.SessionKey
	if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
		sessionKey = msg.SessionKey
	}

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:    sessionKey,
		Channel:       msg.Channel,
		ChatID:        msg.ChatID,
		UserMessage:   msg.Content,
		EnableSummary: true,
		SendResponse:  false,
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
	// Phase 2 Step 4: Check if context is already canceled before starting
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("task canceled before execution: %w", ctx.Err())
	default:
	}

	// ===== NEW: Steering Architecture - Setup checker for this session =====
	if al.enableSteering {
		// Create checker to signal this session is active
		checker := al.getOrCreateChecker(opts.SessionKey)

		// Cleanup with grace period to handle race conditions
		defer func() {
			// Set grace period before checking for late arrivals
			checker.SetGracePeriod(2 * time.Second)

			// Wait briefly for any race-condition messages
			time.Sleep(150 * time.Millisecond)

			// Check one final time for pending interruptions
			finalPending := checker.DrainAll()
			if len(finalPending) > 0 {
				logger.InfoCF("agent", "Steering: found interruptions during grace period, reprocessing",
					map[string]any{
						"session_key":   opts.SessionKey,
						"pending_count": len(finalPending),
						"channel":       opts.Channel,
						"chat_id":       opts.ChatID,
					})

				// Re-trigger processing by publishing as new inbound message
				// Combine all pending messages
				injectionContent := formatInterruptionInjection(finalPending)
				al.bus.PublishInbound(ctx, bus.InboundMessage{
					Channel:    opts.Channel,
					ChatID:     opts.ChatID,
					SessionKey: opts.SessionKey,
					Content:    injectionContent,
					Metadata:   make(map[string]string), // Empty metadata for re-triggered message
				})
			}

			// Now safe to remove checker
			al.removeChecker(opts.SessionKey)
		}()
	}

	// 0a. Update interrupt handler context
	if busHandler, ok := al.interruptHandler.(*BusInterruptHandler); ok {
		busHandler.SetContext(opts.Channel, opts.ChatID)
	}

	// 0b. Record last channel for heartbeat notifications (skip internal channels)
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

	// 2. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = agent.Sessions.GetHistory(opts.SessionKey)
		summary = agent.Sessions.GetSummary(opts.SessionKey)

		// Sanitize history to remove incomplete tool call sequences
		// This prevents API errors when resuming after task interruption/cancellation
		sanitizedHistory := sanitizeIncompleteToolCalls(history)
		if len(sanitizedHistory) != len(history) {
			logger.WarnCF("agent", "Sanitized incomplete tool calls from session history",
				map[string]any{
					"session_key":     opts.SessionKey,
					"original_count":  len(history),
					"sanitized_count": len(sanitizedHistory),
				})
			// Update the session with sanitized history
			agent.Sessions.SetHistory(opts.SessionKey, sanitizedHistory)
			agent.Sessions.Save(opts.SessionKey)
			history = sanitizedHistory
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

	// 3. Save user message to session
	agent.Sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)

	// 4. Run LLM iteration loop
	finalContent, iteration, err := al.runLLMIteration(ctx, agent, messages, opts)
	if err != nil {
		// Special handling for interruption errors
		if errors.Is(err, ErrInterrupted) {
			logger.InfoCF("agent", "Agent execution interrupted by high-priority signal",
				map[string]any{
					"agent_id":   agent.ID,
					"session":    opts.SessionKey,
					"iterations": iteration,
				})
			// Save current state
			agent.Sessions.Save(opts.SessionKey)
			// Return a user-friendly message
			return "Task interrupted by higher priority request. Progress has been saved.", nil
		}
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

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 8. Optional: send response via bus
	if opts.SendResponse {
		al.bus.PublishOutbound(ctx, bus.OutboundMessage{
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

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

func (al *AgentLoop) handleReasoning(
	ctx context.Context,
	reasoningContent, channelName, channelID string,
) {
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

// runLLMIteration executes the LLM call loop with tool handling.
func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
) (string, int, error) {
	iteration := 0
	var finalContent string

	for iteration < agent.MaxIterations {
		iteration++

		// Phase 2 Step 4: Check for context cancellation at start of each iteration
		select {
		case <-ctx.Done():
			return "", iteration, fmt.Errorf("iteration canceled: %w", ctx.Err())
		default:
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       agent.MaxIterations,
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
				fbResult, fbErr := al.fallback.Execute(
					ctx,
					agent.Candidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return agent.Provider.Chat(
							ctx,
							messages,
							providerToolDefs,
							model,
							map[string]any{
								"max_tokens":  agent.MaxTokens,
								"temperature": agent.Temperature,
								// "prompt_cache_key": agent.ID,
							},
						)
					},
				)
				if fbErr != nil {
					return nil, fbErr
				}
				if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
					logger.InfoCF(
						"agent",
						fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
							fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
						map[string]any{"agent_id": agent.ID, "iteration": iteration},
					)
				}
				return fbResult.Response, nil
			}
			return agent.Provider.Chat(ctx, messages, providerToolDefs, agent.Model, map[string]any{
				"max_tokens":  agent.MaxTokens,
				"temperature": agent.Temperature,
				// "prompt_cache_key": agent.ID,
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
				logger.WarnCF(
					"agent",
					"Context window error detected, attempting compression",
					map[string]any{
						"error": err.Error(),
						"retry": retry,
					},
				)

				if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
					al.bus.PublishOutbound(ctx, bus.OutboundMessage{
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

		go al.handleReasoning(
			ctx,
			response.Reasoning,
			opts.Channel,
			al.targetReasoningChannelID(opts.Channel),
		)

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
		// Check if no tool calls - but first check for pending interruptions
		if len(response.ToolCalls) == 0 {
			// NEW: Check for pending interruptions before finishing
			if al.enableSteering {
				checker := al.getOrCreateChecker(opts.SessionKey)
				pending := checker.DrainAll()

				if len(pending) > 0 {
					logger.InfoCF("agent", "Steering: LLM finished but has pending interruptions, injecting",
						map[string]any{
							"session_key":   opts.SessionKey,
							"pending_count": len(pending),
							"iteration":     iteration,
						})

					// Save the assistant's response first
					assistantMsg := providers.Message{
						Role:    "assistant",
						Content: response.Content,
					}
					messages = append(messages, assistantMsg)
					agent.Sessions.AddMessage(opts.SessionKey, "assistant", response.Content)

					// Send the response to user
					if !constants.IsInternalChannel(opts.Channel) {
						al.bus.PublishOutbound(ctx, bus.OutboundMessage{
							Channel: opts.Channel,
							ChatID:  opts.ChatID,
							Content: response.Content,
						})
					}

					// Format and inject interruption
					injectionContent := formatInterruptionInjection(pending)
					injectionMsg := providers.Message{
						Role:    "user",
						Content: injectionContent,
					}
					messages = append(messages, injectionMsg)
					agent.Sessions.AddMessage(opts.SessionKey, "user", injectionContent)

					// Continue to handle the interruption
					continue
				}
			}

			// No interruptions, finish normally
			finalContent = response.Content
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})

			// FINAL SAFETY CHECK: One more check for race-condition interruptions
			// This catches messages that arrived while we were processing the final response
			if al.enableSteering {
				time.Sleep(100 * time.Millisecond) // Brief wait for any in-flight messages
				checker := al.getOrCreateChecker(opts.SessionKey)
				lastMinutePending := checker.DrainAll()

				if len(lastMinutePending) > 0 {
					logger.InfoCF("agent", "Steering: caught last-minute interruptions after final response",
						map[string]any{
							"session_key":   opts.SessionKey,
							"pending_count": len(lastMinutePending),
							"iteration":     iteration,
						})

					// Save assistant's response first
					assistantMsg := providers.Message{
						Role:    "assistant",
						Content: response.Content,
					}
					messages = append(messages, assistantMsg)
					agent.Sessions.AddMessage(opts.SessionKey, "assistant", response.Content)

					// Send response to user
					if !constants.IsInternalChannel(opts.Channel) {
						al.bus.PublishOutbound(ctx, bus.OutboundMessage{
							Channel: opts.Channel,
							ChatID:  opts.ChatID,
							Content: response.Content,
						})
					}

					// Inject last-minute interruptions
					injectionContent := formatInterruptionInjection(lastMinutePending)
					injectionMsg := providers.Message{
						Role:    "user",
						Content: injectionContent,
					}
					messages = append(messages, injectionMsg)
					agent.Sessions.AddMessage(opts.SessionKey, "user", injectionContent)

					// Continue to handle these interruptions
					continue
				}
			}

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
		for i, tc := range normalizedToolCalls {
			// Phase 2 Step 4: Check for context cancellation before each tool execution
			select {
			case <-ctx.Done():
				return "", iteration, fmt.Errorf("tool execution canceled: %w", ctx.Err())
			default:
			}

			// Check for interruption before each tool call
			if al.interruptHandler != nil {
				signal, checkErr := al.interruptHandler.CheckInterruption(ctx)
				if checkErr != nil {
					logger.WarnCF("agent", "Interruption check failed",
						map[string]any{"error": checkErr.Error()})
				}

				if signal != nil && signal.Priority >= 8 {
					logger.InfoCF("agent", "High-priority interruption detected during tool execution",
						map[string]any{
							"type":        signal.Type,
							"priority":    signal.Priority,
							"source":      signal.Source,
							"tool":        tc.Name,
							"tool_index":  i,
							"total_tools": len(normalizedToolCalls),
						})

					// Save current progress to session
					agent.Sessions.Save(opts.SessionKey)

					// If it's a user message interruption, put the message back
					if signal.Type == "user_message" && signal.Data != nil {
						if msg, ok := signal.Data.(bus.InboundMessage); ok {
							go func() {
								pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
								defer cancel()
								al.bus.PublishInbound(pubCtx, msg)
							}()
						}
					}

					// Return interruption error
					return "", iteration, ErrInterrupted
				}
			}

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

			toolResult := agent.Tools.ExecuteWithContext(
				ctx,
				tc.Name,
				tc.Arguments,
				opts.Channel,
				opts.ChatID,
				asyncCallback,
			)

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(ctx, bus.OutboundMessage{
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

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
		}

		// ===== NEW: Steering Architecture - Check for interruptions after tool execution =====
		if al.enableSteering {
			checker := al.getOrCreateChecker(opts.SessionKey)
			pending := checker.DrainAll()

			if len(pending) > 0 {
				logger.InfoCF("agent", "Steering: injecting interruption messages",
					map[string]any{
						"session_key":   opts.SessionKey,
						"pending_count": len(pending),
						"iteration":     iteration,
					})

				// Send progress update to user showing tool results before handling interruption
				if !constants.IsInternalChannel(opts.Channel) {
					// Build a brief summary of what just completed
					completedTools := []string{}
					for _, msg := range messages {
						if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
							for _, tc := range msg.ToolCalls {
								completedTools = append(completedTools, tc.Name)
							}
						}
					}

					progressMsg := fmt.Sprintf("⚡ Completed: %s\n📥 Processing new request...",
						utils.Truncate(fmt.Sprint(completedTools), 100))

					al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: progressMsg,
					})
				}

				// Format and inject interruption
				injectionContent := formatInterruptionInjection(pending)
				injectionMsg := providers.Message{
					Role:    "user",
					Content: injectionContent,
				}
				messages = append(messages, injectionMsg)

				// Save injection to session for context continuity
				agent.Sessions.AddMessage(opts.SessionKey, "user", injectionContent)

				// Continue to next iteration with injected message
				// The LLM will decide how to handle both the original task and the new request
				continue
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
				logger.Debug("Memory threshold reached. Optimizing conversation history...")
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
					fmt.Fprintf(
						&sb,
						"      Arguments: %s\n",
						utils.Truncate(tc.Function.Arguments, 200),
					)
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
			fmt.Fprintf(
				&sb,
				"      Parameters: %s\n",
				utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200),
			)
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
				// "prompt_cache_key": agent.ID,
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
			"max_tokens":  1024,
			"temperature": 0.3,
			// "prompt_cache_key": agent.ID,
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

	case "/stop", "/cancel", "/abort":
		// Phase 2: Cancel running tasks for the current session
		runningTasks := al.taskManager.GetRunningTasksForSession(msg.Channel, msg.ChatID)

		// Filter out command tasks from the count (exclude tasks with is_command metadata)
		nonCommandTasks := make([]*Task, 0)
		for _, task := range runningTasks {
			if isCmd, ok := task.Metadata["is_command"].(bool); !ok || !isCmd {
				nonCommandTasks = append(nonCommandTasks, task)
			}
		}

		if len(nonCommandTasks) == 0 {
			// No running tasks to cancel
			return "ℹ️ 没有正在运行的任务需要取消。\nNo running tasks to cancel.", true
		}

		// Cancel all tasks (including command tasks)
		canceled := al.taskManager.CancelAllTasksForSession(msg.Channel, msg.ChatID)

		// Sanitize session history to remove incomplete tool_use/tool_result pairs
		// This prevents API errors (e.g., "tool_use ids were found without tool_result blocks")
		// when resuming after cancellation
		route := al.registry.ResolveRoute(routing.RouteInput{
			Channel:    msg.Channel,
			AccountID:  msg.Metadata["account_id"],
			Peer:       extractPeer(msg),
			ParentPeer: extractParentPeer(msg),
			GuildID:    msg.Metadata["guild_id"],
			TeamID:     msg.Metadata["team_id"],
		})
		agent, _ := al.registry.GetAgent(route.AgentID)
		if agent == nil {
			agent = al.registry.GetDefaultAgent()
		}
		sessionSanitized := false
		if agent != nil {
			sessionKey := route.SessionKey
			if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
				sessionKey = msg.SessionKey
			}
			// Sanitize the session history to remove incomplete tool call sequences
			history := agent.Sessions.GetHistory(sessionKey)
			sanitizedHistory := sanitizeIncompleteToolCalls(history)
			agent.Sessions.SetHistory(sessionKey, sanitizedHistory)
			agent.Sessions.Save(sessionKey)
			sessionSanitized = len(history) != len(sanitizedHistory)
		}

		logger.InfoCF("agent", "User canceled running tasks",
			map[string]any{
				"channel":           msg.Channel,
				"chat_id":           msg.ChatID,
				"command":           cmd,
				"total_canceled":    canceled,
				"non_command_count": len(nonCommandTasks),
				"session_sanitized": sessionSanitized,
			})

		// Build response message using non-command task count
		var response string
		nonCommandCount := len(nonCommandTasks)
		if nonCommandCount == 1 {
			response = "🛑 已取消 1 个正在运行的任务。\n✅ 已就绪，可以处理新的请求。\n\nCanceled 1 running task. Ready for new requests."
		} else {
			response = fmt.Sprintf("🛑 已取消 %d 个正在运行的任务。\n✅ 已就绪，可以处理新的请求。\n\nCanceled %d running tasks. Ready for new requests.",
				nonCommandCount, nonCommandCount)
		}

		return response, true

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
	}

	return "", false
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

// sanitizeIncompleteToolCalls removes assistant messages with tool calls that don't have
// corresponding tool result messages. This prevents API errors when resuming after task cancellation.
//
// The function walks through the message history and:
// 1. Identifies assistant messages with tool calls
// 2. Checks if each tool call has a matching tool result in the next messages
// 3. Removes assistant messages with incomplete tool call sequences
//
// This is necessary because Claude's API requires every tool_use block to have a corresponding
// tool_result block immediately after in the conversation flow.
func sanitizeIncompleteToolCalls(messages []providers.Message) []providers.Message {
	if len(messages) == 0 {
		return messages
	}

	sanitized := make([]providers.Message, 0, len(messages))

	for i := 0; i < len(messages); i++ {
		msg := messages[i]

		// If this is an assistant message with tool calls, check if all tool calls have results
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Collect all tool call IDs from this message
			toolCallIDs := make(map[string]bool)
			for _, tc := range msg.ToolCalls {
				toolCallIDs[tc.ID] = true
			}

			// Look ahead to find tool results for these calls
			foundResults := make(map[string]bool)
			for j := i + 1; j < len(messages); j++ {
				nextMsg := messages[j]
				if nextMsg.Role == "tool" && nextMsg.ToolCallID != "" {
					if toolCallIDs[nextMsg.ToolCallID] {
						foundResults[nextMsg.ToolCallID] = true
					}
				}
				// Stop looking once we hit another assistant message or user message
				// (tool results must immediately follow the assistant message with tool calls)
				if nextMsg.Role == "assistant" || nextMsg.Role == "user" {
					break
				}
			}

			// Only keep this assistant message if ALL tool calls have results
			if len(foundResults) == len(toolCallIDs) {
				sanitized = append(sanitized, msg)
			} else {
				// Skip this assistant message and all its tool results
				logger.WarnCF("agent", "Removing incomplete tool call sequence",
					map[string]any{
						"expected_results": len(toolCallIDs),
						"found_results":    len(foundResults),
						"tool_call_ids":    getToolCallIDsList(msg.ToolCalls),
					})

				// Also skip the subsequent tool result messages that belong to this assistant message
				for j := i + 1; j < len(messages); j++ {
					nextMsg := messages[j]
					if nextMsg.Role == "tool" && toolCallIDs[nextMsg.ToolCallID] {
						i = j // Skip this tool result too
					} else {
						break
					}
				}
			}
		} else {
			// Keep non-assistant messages or assistant messages without tool calls
			sanitized = append(sanitized, msg)
		}
	}

	return sanitized
}

// getToolCallIDsList extracts tool call IDs for logging
func getToolCallIDsList(toolCalls []providers.ToolCall) []string {
	ids := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		ids[i] = tc.ID
	}
	return ids
}

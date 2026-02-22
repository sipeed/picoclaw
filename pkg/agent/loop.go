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
	"os"
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
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AgentLoop struct {
	bus            *bus.MessageBus
	provider       providers.LLMProvider
	workspace      string
	model          string
	maxTokens      int // Maximum tokens for API response
	contextWindow  int // Maximum context window size in tokens (for summarization)
	maxIterations  int
	sessions       *session.SessionManager
	state          *state.Manager
	contextBuilder *ContextBuilder
	tools          *tools.ToolRegistry
	running        atomic.Bool
	summarizing    sync.Map // Tracks which sessions are currently being summarized
	channelManager *channels.Manager
	rateLimiter    *rateLimiter
	mcpManager     *mcp.Manager
	activeProcs    map[string]*activeProcess
	procsMu        sync.Mutex
	mediaDir       string
}

type activeProcess struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string            // Session identifier for history/context
	Channel         string            // Target channel for tool execution
	ChatID          string            // Target chat ID for tool execution
	UserMessage     string            // User message content (may include prefix)
	Media           []string          // Base64 data URLs for images
	DefaultResponse string            // Response when LLM returns empty
	EnableSummary   bool              // Whether to trigger summarization
	SendResponse    bool              // Whether to send response via bus
	NoHistory       bool              // If true, don't load session history (for heartbeat)
	InputMode       string            // "voice" or "text"
	Metadata        map[string]string // Channel metadata (e.g. client_type)
}

// createToolRegistry creates a tool registry with common tools.
// This is shared between main agent and subagents.
func createToolRegistry(workspace string, restrict bool, cfg *config.Config, msgBus *bus.MessageBus, dataDir string) *tools.ToolRegistry {
	registry := tools.NewToolRegistry()

	// File system tools
	registry.Register(tools.NewReadFileTool(workspace, restrict))
	registry.Register(tools.NewWriteFileTool(workspace, restrict))
	registry.Register(tools.NewListDirTool(workspace, restrict))
	registry.Register(tools.NewEditFileTool(workspace, restrict))
	registry.Register(tools.NewAppendFileTool(workspace, restrict))

	// Copy file tool (allows copying from media dir to workspace)
	mediaDir := filepath.Join(dataDir, "media")
	registry.Register(tools.NewCopyFileTool(workspace, mediaDir, restrict))

	// Shell execution (disabled by default for security)
	if cfg.Tools.Exec.Enabled {
		registry.Register(tools.NewExecTool(workspace, restrict))
	}

	if searchTool := tools.NewWebSearchTool(tools.WebSearchToolOptions{
		BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
	}); searchTool != nil {
		registry.Register(searchTool)
	}
	registry.Register(tools.NewWebFetchTool(50000))

	// Hardware tools (I2C, SPI) - disabled by default for security
	if cfg.Tools.I2C.Enabled {
		registry.Register(tools.NewI2CTool())
	}
	if cfg.Tools.SPI.Enabled {
		registry.Register(tools.NewSPITool())
	}

	// Android device control tool
	sendCallbackWithType := func(channel, chatID, content, msgType string) error {
		msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
			Type:    msgType,
		})
		return nil
	}
	if cfg.Tools.Android.Enabled {
		androidTool := tools.NewAndroidTool()
		androidTool.SetSendCallback(sendCallbackWithType)
		registry.Register(androidTool)
	}

	// Message tool - available to both agent and subagent
	// Subagent uses it to communicate directly with user
	messageTool := tools.NewMessageTool()
	messageTool.SetSendCallback(func(channel, chatID, content string) error {
		msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
		})
		return nil
	})
	// StateResolver is injected later in NewAgentLoop after stateManager is created
	registry.Register(messageTool)

	return registry
}

func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
	workspace := cfg.WorkspacePath()
	dataDir := cfg.DataPath()
	os.MkdirAll(workspace, 0755)
	os.MkdirAll(dataDir, 0755)

	restrict := cfg.Agents.Defaults.RestrictToWorkspace

	// Create media directory for persisting images
	mediaDir := filepath.Join(dataDir, "media")
	os.MkdirAll(mediaDir, 0755)

	// Create tool registry for main agent
	toolsRegistry := createToolRegistry(workspace, restrict, cfg, msgBus, dataDir)

	// Create subagent manager with its own tool registry
	subagentManager := tools.NewSubagentManager(provider, cfg.Agents.Defaults.Model, workspace, msgBus)
	subagentTools := createToolRegistry(workspace, restrict, cfg, msgBus, dataDir)
	// Subagent doesn't need spawn/subagent tools to avoid recursion
	subagentManager.SetTools(subagentTools)

	// Register spawn tool (for main agent only)
	spawnTool := tools.NewSpawnTool(subagentManager)
	toolsRegistry.Register(spawnTool)

	// Register exit tool (for main agent only, voice/assistant mode)
	exitTool := tools.NewExitTool()
	exitTool.SetSendCallback(func(channel, chatID, content, msgType string) error {
		msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
			Type:    msgType,
		})
		return nil
	})
	toolsRegistry.Register(exitTool)

	// Register subagent tool (synchronous execution)
	subagentTool := tools.NewSubagentTool(subagentManager)
	toolsRegistry.Register(subagentTool)

	// Use dataDir for sessions and state (outside workspace for security)
	sessionsManager := session.NewSessionManager(filepath.Join(dataDir, "sessions"))

	// Create state manager for atomic state persistence
	stateManager := state.NewManager(dataDir)

	// Inject state resolver into message tools for cross-channel "app" alias
	if tool, ok := toolsRegistry.Get("message"); ok {
		if mt, ok := tool.(*tools.MessageTool); ok {
			mt.SetStateResolver(stateManager)
		}
	}
	if tool, ok := subagentTools.Get("message"); ok {
		if mt, ok := tool.(*tools.MessageTool); ok {
			mt.SetStateResolver(stateManager)
		}
	}

	// Create context builder and set tools registry
	contextBuilder := NewContextBuilder(workspace, dataDir)
	contextBuilder.SetToolsRegistry(toolsRegistry)

	// Register memory tool (conditionally based on config)
	contextBuilder.SetMemoryToolEnabled(cfg.Tools.Memory.Enabled)
	if cfg.Tools.Memory.Enabled {
		memoryTool := tools.NewMemoryTool(contextBuilder.GetMemory())
		toolsRegistry.Register(memoryTool)
		subagentTools.Register(memoryTool)
	}

	skillTool := tools.NewSkillTool(contextBuilder.GetSkillsLoader())
	toolsRegistry.Register(skillTool)
	subagentTools.Register(skillTool)

	// MCP server manager and bridge tool
	var mcpManager *mcp.Manager
	if len(cfg.Tools.MCP) > 0 {
		mcpManager = mcp.NewManager(cfg.Tools.MCP)
		mcpBridgeTool := tools.NewMCPBridgeTool(mcpManager)
		toolsRegistry.Register(mcpBridgeTool)
		subagentTools.Register(mcpBridgeTool)
		contextBuilder.SetMCPManager(mcpManager)
	}

	return &AgentLoop{
		bus:            msgBus,
		provider:       provider,
		workspace:      workspace,
		model:          cfg.Agents.Defaults.Model,
		maxTokens:      cfg.Agents.Defaults.MaxTokens,
		contextWindow:  cfg.Agents.Defaults.ContextWindow,
		maxIterations:  cfg.Agents.Defaults.MaxToolIterations,
		sessions:       sessionsManager,
		state:          stateManager,
		contextBuilder: contextBuilder,
		tools:          toolsRegistry,
		summarizing:    sync.Map{},
		rateLimiter:    newRateLimiter(cfg.RateLimits.MaxToolCallsPerMinute, cfg.RateLimits.MaxRequestsPerMinute),
		mcpManager:     mcpManager,
		activeProcs:    make(map[string]*activeProcess),
		mediaDir:       mediaDir,
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			sessionKey := msg.SessionKey
			if sessionKey == "" {
				sessionKey = fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
			}

			// Cancel active process for the same session
			al.procsMu.Lock()
			if active, exists := al.activeProcs[sessionKey]; exists {
				active.cancel()
				al.procsMu.Unlock()
				select {
				case <-active.done:
				case <-time.After(5 * time.Second):
					logger.WarnCF("agent", "Timed out waiting for cancelled process",
						map[string]interface{}{"session_key": sessionKey})
				}
				al.procsMu.Lock()
			}

			procCtx, procCancel := context.WithCancel(ctx)
			done := make(chan struct{})
			al.activeProcs[sessionKey] = &activeProcess{cancel: procCancel, done: done}
			al.procsMu.Unlock()

			go func(m bus.InboundMessage, sk string) {
				defer func() {
					// Clear status indicator on completion (normal, error, or cancel)
					if !constants.IsInternalChannel(m.Channel) {
						al.bus.PublishOutbound(bus.OutboundMessage{
							Channel: m.Channel, ChatID: m.ChatID,
							Type: "status_end",
						})
					}
					close(done)
					al.procsMu.Lock()
					if cur, ok := al.activeProcs[sk]; ok && cur.done == done {
						delete(al.activeProcs, sk)
					}
					al.procsMu.Unlock()
					procCancel()
				}()

				response, err := al.processMessage(procCtx, m)

				if procCtx.Err() != nil {
					return
				}
				if err != nil {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: m.Channel, ChatID: m.ChatID,
						Content: fmt.Sprintf("Error: %v", err), Type: "error",
					})
					return
				}
				if response != "" {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: m.Channel, ChatID: m.ChatID, Content: response,
					})
				}
			}(msg, sessionKey)
		}
	}

	return nil
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
	if al.mcpManager != nil {
		al.mcpManager.Stop()
	}
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	al.tools.Register(tool)
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm

	// Propagate enabled channels to context builder and message tools
	if cm != nil {
		channels := cm.GetEnabledChannels()
		al.contextBuilder.SetEnabledChannels(channels)
		if tool, ok := al.tools.Get("message"); ok {
			if mt, ok := tool.(*tools.MessageTool); ok {
				mt.SetEnabledChannels(channels)
			}
		}
	}
}

// StateManager returns the state manager used by this agent loop.
// This allows sharing the same instance with other services (e.g. heartbeat).
func (al *AgentLoop) StateManager() *state.Manager {
	return al.state
}

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChannel(channel string) error {
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	return al.state.SetLastChatID(chatID)
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

func (al *AgentLoop) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
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
func (al *AgentLoop) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: "I've completed processing but have no response to give.",
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
	logger.InfoCF("agent", fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]interface{}{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		})

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Check request rate limit
	if err := al.rateLimiter.checkRequest(); err != nil {
		logger.WarnCF("agent", "Request rate limited",
			map[string]interface{}{
				"channel":   msg.Channel,
				"sender_id": msg.SenderID,
			})
		return fmt.Sprintf("Rate limited: %v. Please try again later.", err), nil
	}

	// Check for commands
	if response, handled := al.handleCommand(ctx, msg); handled {
		return response, nil
	}

	// Extract input_mode from metadata
	inputMode := "text"
	if msg.Metadata != nil {
		if mode, ok := msg.Metadata["input_mode"]; ok && mode != "" {
			inputMode = mode
		}
	}

	// Process as user message
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      msg.SessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		Media:           msg.Media,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   true,
		SendResponse:    false,
		InputMode:       inputMode,
		Metadata:        msg.Metadata,
	})
}

func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Verify this is a system message
	if msg.Channel != "system" {
		return "", fmt.Errorf("processSystemMessage called with non-system message channel: %s", msg.Channel)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]interface{}{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
	} else {
		// Fallback
		originChannel = "cli"
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
			map[string]interface{}{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
			})
		return "", nil
	}

	// Agent acts as dispatcher only - subagent handles user interaction via message tool
	// Don't forward result here, subagent should use message tool to communicate with user
	logger.InfoCF("agent", "Subagent completed",
		map[string]interface{}{
			"sender_id":   msg.SenderID,
			"channel":     originChannel,
			"content_len": len(content),
		})

	// Agent only logs, does not respond to user
	return "", nil
}

// runAgentLoop is the core message processing logic.
// It handles context building, LLM calls, tool execution, and response handling.
func (al *AgentLoop) runAgentLoop(ctx context.Context, opts processOptions) (string, error) {
	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			clientType := ""
			if opts.Metadata != nil {
				clientType = opts.Metadata["client_type"]
			}
			if clientType != "" {
				if err := al.state.SetLastChannelWithType(channelKey, clientType); err != nil {
					logger.WarnCF("agent", "Failed to record last channel: %v", map[string]interface{}{"error": err.Error()})
				}
			} else {
				if err := al.RecordLastChannel(channelKey); err != nil {
					logger.WarnCF("agent", "Failed to record last channel: %v", map[string]interface{}{"error": err.Error()})
				}
			}
			// Record per-channel chatID for cross-channel messaging
			if err := al.state.SetChannelChatID(opts.Channel, opts.ChatID); err != nil {
				logger.WarnCF("agent", "Failed to record channel chatID", map[string]interface{}{"error": err.Error()})
			}
		}
	}

	// 1. Update tool contexts
	al.updateToolContexts(opts.Channel, opts.ChatID, opts.Metadata)

	// 2. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = al.sessions.GetHistory(opts.SessionKey)
		summary = al.sessions.GetSummary(opts.SessionKey)
	}
	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		opts.Media,
		opts.Channel,
		opts.ChatID,
		opts.InputMode,
	)

	// 3. Save user message to session (with media if present)
	userContent := opts.UserMessage
	if len(opts.Media) > 0 {
		paths := PersistMedia(opts.Media, al.mediaDir)
		for _, p := range paths {
			userContent += fmt.Sprintf("\n[Image: %s]", p)
		}
	}
	al.sessions.AddFullMessage(opts.SessionKey, providers.Message{
		Role:    "user",
		Content: userContent,
		Media:   opts.Media,
	})

	// 4. Emit thinking status
	if !constants.IsInternalChannel(opts.Channel) {
		al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: "思考中...",
			Type:    "status",
		})
	}

	// Start heartbeat goroutine - resend current status every 10s
	var currentStatus atomic.Value
	currentStatus.Store("思考中...")
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()

	if !constants.IsInternalChannel(opts.Channel) {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-heartbeatCtx.Done():
					return
				case <-ticker.C:
					if status, ok := currentStatus.Load().(string); ok && status != "" {
						al.bus.PublishOutbound(bus.OutboundMessage{
							Channel: opts.Channel,
							ChatID:  opts.ChatID,
							Content: status,
							Type:    "status",
						})
					}
				}
			}
		}()
	}

	// 5. Run LLM iteration loop
	finalContent, iteration, err := al.runLLMIteration(ctx, messages, opts, &currentStatus)

	if ctx.Err() != nil {
		// Processing was cancelled (e.g., new message from same session)
		al.sessions.AddMessage(opts.SessionKey, "assistant", "[応答は中断されました]")
		al.sessions.Save(opts.SessionKey)
		return "", nil
	}

	if err != nil {
		return "", err
	}

	// Handle NO_REPLY token — suppress sending to user
	if strings.TrimSpace(finalContent) == SilentReplyToken {
		al.sessions.AddMessage(opts.SessionKey, "assistant", "[silent]")
		al.sessions.Save(opts.SessionKey)
		if opts.EnableSummary {
			al.maybeSummarize(opts.SessionKey, opts.Channel, opts.ChatID)
		}
		return "", nil
	}

	// 5. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 6. Save final assistant message to session
	al.sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	al.sessions.Save(opts.SessionKey)

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(opts.SessionKey, opts.Channel, opts.ChatID)
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
		map[string]interface{}{
			"session_key":  opts.SessionKey,
			"iterations":   iteration,
			"final_length": len(finalContent),
		})

	return finalContent, nil
}

// runLLMIteration executes the LLM call loop with tool handling.
// Returns the final content, iteration count, and any error.
func (al *AgentLoop) runLLMIteration(ctx context.Context, messages []providers.Message, opts processOptions, currentStatus *atomic.Value) (string, int, error) {
	iteration := 0
	var finalContent string

	for iteration < al.maxIterations {
		iteration++

		// Cancellation checkpoint at start of each iteration
		select {
		case <-ctx.Done():
			return finalContent, iteration, ctx.Err()
		default:
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]interface{}{
				"iteration": iteration,
				"max":       al.maxIterations,
			})

		// Build tool definitions
		providerToolDefs := al.tools.ToProviderDefs()

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]interface{}{
				"iteration":         iteration,
				"model":             al.model,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        al.maxTokens,
				"temperature":       0.7,
				"system_prompt_len": len(messages[0].Content),
			})

		// Log full messages (detailed)
		logger.DebugCF("agent", "Full LLM request",
			map[string]interface{}{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		var response *providers.LLMResponse
		var err error

		// Retry loop for context/token errors
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
				"max_tokens":  al.maxTokens,
				"temperature": 0.7,
			})

			if err == nil {
				break // Success
			}

			errMsg := strings.ToLower(err.Error())

			// Request cancellation (e.g. user sent a new message) is not a context
			// window error — break out of the retry loop immediately.
			// The caller's ctx.Err() check (line ~530) will handle the cancel gracefully.
			if ctx.Err() != nil {
				break
			}

			// Check for context window errors (provider specific, but usually contain "token" or "invalid")
			isContextError := strings.Contains(errMsg, "token") ||
				strings.Contains(errMsg, "context") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "length")

			if isContextError && retry < maxRetries {
				logger.WarnCF("agent", "Context window error detected, attempting compression", map[string]interface{}{
					"error": err.Error(),
					"retry": retry,
				})

				// Notify user on first retry only
				if retry == 0 && !constants.IsInternalChannel(opts.Channel) && opts.SendResponse {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: "⚠️ Context window exceeded. Compressing history and retrying...",
						Type:    "warning",
					})
				}

				// Force compression
				al.forceCompression(opts.SessionKey)

				// Rebuild messages with compressed history
				// Note: We need to reload history from session manager because forceCompression changed it
				newHistory := al.sessions.GetHistory(opts.SessionKey)
				newSummary := al.sessions.GetSummary(opts.SessionKey)

				// Re-create messages for the next attempt
				// We keep the current user message (opts.UserMessage) effectively
				messages = al.contextBuilder.BuildMessages(
					newHistory,
					newSummary,
					opts.UserMessage,
					nil,
					opts.Channel,
					opts.ChatID,
					opts.InputMode,
				)

				// Important: If we are in the middle of a tool loop (iteration > 1),
				// rebuilding messages from session history might duplicate the flow or miss context
				// if intermediate steps weren't saved correctly.
				// However, al.sessions.AddFullMessage is called after every tool execution,
				// so GetHistory should reflect the current state including partial tool execution.
				// But we need to ensure we don't duplicate the user message which is appended in BuildMessages.
				// BuildMessages(history...) takes the stored history and appends the *current* user message.
				// If iteration > 1, the "current user message" was already added to history in step 3 of runAgentLoop.
				// So if we pass opts.UserMessage again, we might duplicate it?
				// Actually, step 3 is: al.sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)
				// So GetHistory ALREADY contains the user message!

				// CORRECTION:
				// BuildMessages combines: [System] + [History] + [CurrentMessage]
				// But Step 3 added CurrentMessage to History.
				// So if we use GetHistory now, it has the user message.
				// If we pass opts.UserMessage to BuildMessages, it adds it AGAIN.

				// For retry in the middle of a loop, we should rely on what's in the session.
				// BUT checking BuildMessages implementation:
				// It appends history... then appends currentMessage.

				// Logic fix for retry:
				// If iteration == 1, opts.UserMessage corresponds to the user input.
				// If iteration > 1, we are processing tool results. The "messages" passed to Chat
				// already accumulated tool outputs.
				// Rebuilding from session history is safest because it persists state.
				// Start fresh with rebuilt history.

				// Special case: standard BuildMessages appends "currentMessage".
				// If we are strictly retrying the *LLM call*, we want the exact same state as before but compressed.
				// However, the "messages" argument passed to runLLMIteration is constructed by the caller.
				// If we rebuild from Session, we need to know if "currentMessage" should be appended or is already in history.

				// In runAgentLoop:
				// 3. sessions.AddMessage(userMsg)
				// 4. runLLMIteration(..., UserMessage)

				// So History contains the user message.
				// BuildMessages typically appends the user message as a *new* pending message.
				// Wait, standard BuildMessages usage in runAgentLoop:
				// messages := BuildMessages(history (has old), UserMessage)
				// THEN AddMessage(UserMessage).
				// So "history" passed to BuildMessages does NOT contain the current UserMessage yet.

				// But here, inside the loop, we have already saved it.
				// So GetHistory() includes the current user message.
				// If we call BuildMessages(GetHistory(), UserMessage), we get duplicates.

				// Hack/Fix:
				// If we are retrying, we rebuild from Session History ONLY.
				// We pass empty string as "currentMessage" to BuildMessages
				// because the "current message" is already saved in history (step 3).

				messages = al.contextBuilder.BuildMessages(
					newHistory,
					newSummary,
					"", // Empty because history already contains the relevant messages
					nil,
					opts.Channel,
					opts.ChatID,
					opts.InputMode,
				)

				continue
			}

			// Real error or success, break loop
			break
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]interface{}{
					"iteration": iteration,
					"error":     err.Error(),
				})
			return "", iteration, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]interface{}{
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// Log tool calls
		toolNames := make([]string, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]interface{}{
				"tools":     toolNames,
				"count":     len(response.ToolCalls),
				"iteration": iteration,
			})

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}
		for _, tc := range response.ToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		// Save assistant message with tool calls to session
		al.sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls
		for _, tc := range response.ToolCalls {
			// Check tool call rate limit
			if err := al.rateLimiter.checkToolCall(); err != nil {
				logger.WarnCF("agent", "Tool call rate limited",
					map[string]interface{}{
						"tool":      tc.Name,
						"iteration": iteration,
					})
				toolResultMsg := providers.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("Rate limited: %v", err),
					ToolCallID: tc.ID,
				}
				messages = append(messages, toolResultMsg)
				al.sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
				continue
			}

			// Log tool call with arguments preview
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]interface{}{
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
						map[string]interface{}{
							"tool":        tc.Name,
							"content_len": len(result.ForUser),
						})
				}
			}

			// Emit tool use status indicator
			if !constants.IsInternalChannel(opts.Channel) {
				if label := statusLabel(tc.Name, tc.Arguments); label != "" {
					currentStatus.Store(label)
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: label,
						Type:    "status",
					})
				}
			}

			toolResult := al.tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: toolResult.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]interface{}{
						"tool":        tc.Name,
						"content_len": len(toolResult.ForUser),
					})
			}

			// Determine content for LLM based on tool result
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			// Persist media files from tool results (e.g. screenshots)
			if len(toolResult.Media) > 0 {
				paths := PersistMedia(toolResult.Media, al.mediaDir)
				for _, p := range paths {
					contentForLLM += fmt.Sprintf("\n[Image: %s]", p)
				}
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				Media:      toolResult.Media,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			al.sessions.AddFullMessage(opts.SessionKey, toolResultMsg)

			// Cancellation checkpoint after each tool execution
			select {
			case <-ctx.Done():
				return finalContent, iteration, ctx.Err()
			default:
			}
		}
	}

	return finalContent, iteration, nil
}

// updateToolContexts updates the context for tools that need channel/chatID info.
func (al *AgentLoop) updateToolContexts(channel, chatID string, metadata map[string]string) {
	// Use ContextualTool interface instead of type assertions
	if tool, ok := al.tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(channel, chatID)
		}
	}
	if tool, ok := al.tools.Get("spawn"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
	if tool, ok := al.tools.Get("subagent"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
	if tool, ok := al.tools.Get("android"); ok {
		if ct, ok := tool.(tools.ContextualTool); ok {
			ct.SetContext(channel, chatID)
		}
		if at, ok := tool.(*tools.AndroidTool); ok {
			if metadata != nil {
				at.SetClientType(metadata["client_type"])
			} else {
				at.SetClientType("")
			}
		}
	}
	if tool, ok := al.tools.Get("exit"); ok {
		if et, ok := tool.(*tools.ExitTool); ok {
			et.SetContext(channel, chatID)
			if metadata != nil {
				et.SetInputMode(metadata["input_mode"])
			} else {
				et.SetInputMode("")
			}
		}
	}
}

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(sessionKey, channel, chatID string) {
	newHistory := al.sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := al.contextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		if _, loading := al.summarizing.LoadOrStore(sessionKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(sessionKey)
				// Notify user about optimization if not an internal channel
				if !constants.IsInternalChannel(channel) {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: channel,
						ChatID:  chatID,
						Content: "⚠️ Memory threshold reached. Optimizing conversation history...",
						Type:    "warning",
					})
				}
				al.summarizeSession(sessionKey)
			}()
		}
	}
}

// forceCompression aggressively reduces context when the limit is hit.
// It drops the oldest 50% of messages (keeping system prompt and last user message).
func (al *AgentLoop) forceCompression(sessionKey string) {
	history := al.sessions.GetHistory(sessionKey)
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

	// Adjust mid so we don't split in the middle of a tool_calls/tool group.
	// Both cases move mid forward to include the entire group in the dropped
	// portion, which maximises context freed by forceCompression.
	//
	// Case 1: mid lands on a "tool" message — advance past all consecutive
	//         tool messages so the group is dropped together with its
	//         preceding assistant (already in the dropped half).
	// Case 2: mid lands on an "assistant" with tool_calls — advance past
	//         the subsequent tool responses so the whole group is dropped.
	if mid < len(conversation) && conversation[mid].Role == "tool" {
		for mid < len(conversation) && conversation[mid].Role == "tool" {
			mid++
		}
	} else if mid < len(conversation) && conversation[mid].Role == "assistant" && len(conversation[mid].ToolCalls) > 0 {
		mid++
		for mid < len(conversation) && conversation[mid].Role == "tool" {
			mid++
		}
	}

	// Clamp: ensure we keep at least something and drop at least something
	if mid <= 0 {
		mid = 1
	}
	if mid >= len(conversation) {
		mid = len(conversation) - 1
	}

	droppedCount := mid
	// Clean up media files from dropped messages
	CleanupMediaFiles(conversation[:mid])
	keptConversation := conversation[mid:]

	newHistory := make([]providers.Message, 0)
	newHistory = append(newHistory, history[0]) // System prompt

	// Add a note about compression
	compressionNote := fmt.Sprintf("[System: Emergency compression dropped %d oldest messages due to context limit]", droppedCount)
	// If there was an existing summary, we might lose it if it was in the dropped part (which is just messages).
	// The summary is stored separately in session.Summary, so it persists!
	// We just need to ensure the user knows there's a gap.

	// Use "user" role for the compression note because some providers reject
	// "system" messages that appear after the initial system prompt.
	newHistory = append(newHistory, providers.Message{
		Role:    "user",
		Content: compressionNote,
	})

	newHistory = append(newHistory, keptConversation...)
	newHistory = append(newHistory, history[len(history)-1]) // Last message

	// Update session
	al.sessions.SetHistory(sessionKey, newHistory)
	al.sessions.Save(sessionKey)

	logger.WarnCF("agent", "Forced compression executed", map[string]interface{}{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(newHistory),
	})
}

// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Tools info
	tools := al.tools.List()
	info["tools"] = map[string]interface{}{
		"count": len(tools),
		"names": tools,
	}

	// Skills info
	info["skills"] = al.contextBuilder.GetSkillsInfo()

	return info
}

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, msg := range messages {
		result += fmt.Sprintf("  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			result += "  ToolCalls:\n"
			for _, tc := range msg.ToolCalls {
				result += fmt.Sprintf("    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					result += fmt.Sprintf("      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			result += fmt.Sprintf("  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			result += fmt.Sprintf("  ToolCallID: %s\n", msg.ToolCallID)
		}
		result += "\n"
	}
	result += "]"
	return result
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(tools []providers.ToolDefinition) string {
	if len(tools) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, tool := range tools {
		result += fmt.Sprintf("  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		result += fmt.Sprintf("      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			result += fmt.Sprintf("      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	result += "]"
	return result
}

// summarizeSession summarizes the conversation history for a session.
func (al *AgentLoop) summarizeSession(sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := al.sessions.GetHistory(sessionKey)
	summary := al.sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard
	// Skip messages larger than 50% of context window to prevent summarizer overflow
	maxMessageTokens := al.contextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		// Estimate tokens for this message
		msgTokens := len(m.Content) / 2 // Use safer estimate here too (2.5 -> 2 for integer division safety)
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
	// Split into two parts if history is significant
	var finalSummary string
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, part1, "")
		s2, _ := al.summarizeBatch(ctx, part2, "")

		// Merge them
		mergePrompt := fmt.Sprintf("Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s", s1, s2)
		resp, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: mergePrompt}}, nil, al.model, map[string]interface{}{
			"max_tokens":  1024,
			"temperature": 0.3,
		})
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		// Clean up media files from messages being summarized
		CleanupMediaFiles(toSummarize)
		al.sessions.SetSummary(sessionKey, finalSummary)
		al.sessions.TruncateHistory(sessionKey, 4)
		al.sessions.Save(sessionKey)
	}
}

// summarizeBatch summarizes a batch of messages.
func (al *AgentLoop) summarizeBatch(ctx context.Context, batch []providers.Message, existingSummary string) (string, error) {
	prompt := "Provide a concise summary of this conversation segment, preserving core context and key points.\n"
	if existingSummary != "" {
		prompt += "Existing context: " + existingSummary + "\n"
	}
	prompt += "\nCONVERSATION:\n"
	for _, m := range batch {
		prompt += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	response, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: prompt}}, nil, al.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
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
			return "Usage: /show [model|channel]", true
		}
		switch args[0] {
		case "model":
			return fmt.Sprintf("Current model: %s", al.model), true
		case "channel":
			return fmt.Sprintf("Current channel: %s", msg.Channel), true
		default:
			return fmt.Sprintf("Unknown show target: %s", args[0]), true
		}

	case "/list":
		if len(args) < 1 {
			return "Usage: /list [models|channels]", true
		}
		switch args[0] {
		case "models":
			// TODO: Fetch available models dynamically if possible
			return "Available models: glm-4.7, claude-3-5-sonnet, gpt-4o (configured in config.json/env)", true
		case "channels":
			if al.channelManager == nil {
				return "Channel manager not initialized", true
			}
			channels := al.channelManager.GetEnabledChannels()
			if len(channels) == 0 {
				return "No channels enabled", true
			}
			return fmt.Sprintf("Enabled channels: %s", strings.Join(channels, ", ")), true
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
			oldModel := al.model
			al.model = value
			return fmt.Sprintf("Switched model from %s to %s", oldModel, value), true
		case "channel":
			// This changes the 'default' channel for some operations, or effectively redirects output?
			// For now, let's just validate if the channel exists
			if al.channelManager == nil {
				return "Channel manager not initialized", true
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Sprintf("Channel '%s' not found or not enabled", value), true
			}

			// If message came from CLI, maybe we want to redirect CLI output to this channel?
			// That would require state persistence about "redirected channel"
			// For now, just acknowledged.
			return fmt.Sprintf("Switched target channel to %s (Note: this currently only validates existence)", value), true
		default:
			return fmt.Sprintf("Unknown switch target: %s", target), true
		}
	}

	return "", false
}

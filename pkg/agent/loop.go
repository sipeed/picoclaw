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
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// Session mode constants
type sessionMode int

const (
	modePico sessionMode = iota // Default: messages → LLM
	modeCmd                     // Command mode: messages → shell
)

type AgentLoop struct {
	bus             *bus.MessageBus
	cfg             *config.Config
	registry        *AgentRegistry
	state           *state.Manager
	running         atomic.Bool
	summarizing     sync.Map
	fallback        *providers.FallbackChain
	channelManager  *channels.Manager
	sessionModes    sync.Map // per-session mode: sessionKey -> sessionMode
	sessionWorkDirs sync.Map // per-session working dir: sessionKey -> string
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
	WorkingDir      string // Current working directory override (for hipico from cmd mode)
}

func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
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

	return &AgentLoop{
		bus:         msgBus,
		cfg:         cfg,
		registry:    registry,
		state:       stateManager,
		summarizing: sync.Map{},
		fallback:    fallbackChain,
	}
}

func (al *AgentLoop) getSessionMode(sessionKey string) sessionMode {
	if v, ok := al.sessionModes.Load(sessionKey); ok {
		return v.(sessionMode)
	}
	return modePico
}

func (al *AgentLoop) setSessionMode(sessionKey string, mode sessionMode) {
	al.sessionModes.Store(sessionKey, mode)
}

func (al *AgentLoop) getSessionWorkDir(sessionKey string) string {
	if v, ok := al.sessionWorkDirs.Load(sessionKey); ok {
		return v.(string)
	}
	return ""
}

func (al *AgentLoop) setSessionWorkDir(sessionKey string, dir string) {
	al.sessionWorkDirs.Store(sessionKey, dir)
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
		}); searchTool != nil {
			agent.Tools.Register(searchTool)
		}
		agent.Tools.Register(tools.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy))

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

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

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
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: msg.Channel,
						ChatID:  msg.ChatID,
						Content: response,
					})
				}
			}
		}
	}

	return nil
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

// ProcessDirectWithWorkDir processes a message with an explicit working directory context.
// The workDir is injected into the system prompt so the AI resolves file paths relative to it.
func (al *AgentLoop) ProcessDirectWithWorkDir(ctx context.Context, content, sessionKey, workDir string) (string, error) {
	msg := bus.InboundMessage{
		Channel:    "cli",
		SenderID:   "cron",
		ChatID:     "direct",
		Content:    content,
		SessionKey: sessionKey,
		Metadata:   map[string]string{"work_dir": workDir},
	}
	return al.processMessage(ctx, msg)
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
		map[string]any{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		})

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

	// Handle mode-switching commands (:cmd, :pico, :hipico)
	content := strings.TrimSpace(msg.Content)
	if strings.HasPrefix(content, ":") {
		if response, handled := al.handleModeCommand(content, sessionKey, agent); handled {
			return response, nil
		}
		// :hipico <msg> falls through here — one-shot LLM call, stays in modeCmd
		if strings.HasPrefix(content, ":hipico") {
			userMessage := strings.TrimSpace(strings.TrimPrefix(content, ":hipico"))
			workDir := al.getSessionWorkDir(sessionKey)
			if workDir == "" {
				workDir = agent.Workspace
			}
			hipicoSessionKey := sessionKey + ":hipico"
			return al.runAgentLoop(ctx, agent, processOptions{
				SessionKey:      hipicoSessionKey,
				Channel:         msg.Channel,
				ChatID:          msg.ChatID,
				UserMessage:     userMessage,
				DefaultResponse: "I've completed processing but have no response to give.",
				EnableSummary:   false,
				SendResponse:    false,
				WorkingDir:      workDir,
			})
		}
	}

	// Dispatch based on current session mode
	switch al.getSessionMode(sessionKey) {
	case modeCmd:
		return al.executeCmdMode(ctx, agent, content, sessionKey, msg.Channel, msg.ChatID)

	default: // modePico
		return al.runAgentLoop(ctx, agent, processOptions{
			SessionKey:      sessionKey,
			Channel:         msg.Channel,
			ChatID:          msg.ChatID,
			UserMessage:     msg.Content,
			DefaultResponse: "I've completed processing but have no response to give.",
			EnableSummary:   true,
			SendResponse:    false,
			WorkingDir:      msg.Metadata["work_dir"],
		})
	}
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

// runAgentLoop is the core message processing logic.
func (al *AgentLoop) runAgentLoop(ctx context.Context, agent *AgentInstance, opts processOptions) (string, error) {
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
	}
	messages := agent.ContextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		nil,
		opts.Channel,
		opts.ChatID,
	)

	// 2b. Inject current working directory into system prompt if set
	if opts.WorkingDir != "" && len(messages) > 0 && messages[0].Role == "system" {
		messages[0].Content += fmt.Sprintf(
			"\n\n## Current Working Directory\nThe user is currently working in: %s\n"+
				"When the user refers to files or directories, resolve them relative to this path, not the workspace root.",
			opts.WorkingDir,
		)
	}

	// 3. Save user message to session
	agent.Sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)

	// 4. Run LLM iteration loop
	finalContent, iteration, err := al.runLLMIteration(ctx, agent, messages, opts)
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
				fbResult, fbErr := al.fallback.Execute(ctx, agent.Candidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return agent.Provider.Chat(ctx, messages, providerToolDefs, model, map[string]any{
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

		// Accumulate token usage
		agent.AddUsage(response.Usage)

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
		for _, tc := range normalizedToolCalls {
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

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
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
				if !constants.IsInternalChannel(channel) {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: channel,
						ChatID:  chatID,
						Content: "Memory threshold reached. Optimizing conversation history...",
					})
				}
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

// GetUsageInfo returns accumulated token usage for the default agent.
func (al *AgentLoop) GetUsageInfo() map[string]any {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return nil
	}
	promptTokens := agent.TotalPromptTokens.Load()
	completionTokens := agent.TotalCompletionTokens.Load()
	return map[string]any{
		"model":             agent.Model,
		"max_tokens":        agent.MaxTokens,
		"temperature":       agent.Temperature,
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
		"requests":          agent.TotalRequests.Load(),
	}
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

	// Handle : prefixed extension commands (work across all channels)
	if strings.HasPrefix(content, ":") {
		return al.handleExtensionCommand(content)
	}

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
	}

	return "", false
}

// handleExtensionCommand handles : prefixed commands that work across all channels.
func (al *AgentLoop) handleExtensionCommand(content string) (string, bool) {
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", false
	}

	cmd := parts[0]

	switch cmd {
	case ":cmd", ":pico", ":hipico", ":edit":
		// Pass through to processMessage for mode handling (needs sessionKey from routing)
		return "", false

	case ":help":
		return `:help - Show this help message
:usage - Show model info and token usage
:cmd - Switch to command mode (execute shell commands)
:pico - Switch to chat mode (default, AI conversation)
:hipico <msg> - Ask AI for help (from command mode, one-shot)
:edit <file> - View/edit files (cmd mode)
/show [model|channel|agents] - Show current configuration
/list [models|channels|agents] - List available options
/switch [model|channel] to <name> - Switch model or channel`, true

	case ":usage":
		agent := al.registry.GetDefaultAgent()
		if agent == nil {
			return "No agent available.", true
		}
		promptTokens := agent.TotalPromptTokens.Load()
		completionTokens := agent.TotalCompletionTokens.Load()
		return fmt.Sprintf(`Model: %s
Max tokens: %d
Temperature: %.1f

Token usage (this session):
  Prompt tokens: %d
  Completion tokens: %d
  Total tokens: %d
  Requests: %d`,
			agent.Model,
			agent.MaxTokens,
			agent.Temperature,
			promptTokens,
			completionTokens,
			promptTokens+completionTokens,
			agent.TotalRequests.Load(),
		), true

	default:
		return fmt.Sprintf("Unknown command: %s\nType :help for available commands.", cmd), true
	}
}

// handleModeCommand processes mode-switching commands (:cmd, :pico, :hipico).
// Returns (response, handled). If handled is true, the caller should return the response directly.
// For :hipico with a message, it returns ("", false) so processMessage continues with a one-shot LLM call.
func (al *AgentLoop) handleModeCommand(content, sessionKey string, agent *AgentInstance) (string, bool) {
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", false
	}

	cmd := parts[0]

	switch cmd {
	case ":cmd":
		al.setSessionMode(sessionKey, modeCmd)
		workDir := al.getSessionWorkDir(sessionKey)
		if workDir == "" {
			workDir = agent.Workspace
			al.setSessionWorkDir(sessionKey, workDir)
		}
		displayDir := shortenHomePath(workDir)
		return fmt.Sprintf("```\n%s$\n```\nType `:pico` to return to chat mode.", displayDir), true

	case ":pico":
		al.setSessionMode(sessionKey, modePico)
		return "Switched to chat mode. Type :cmd to enter command mode.", true

	case ":hipico":
		msg := strings.TrimSpace(strings.TrimPrefix(content, ":hipico"))
		if msg == "" {
			return "Usage: :hipico <message>\nExample: :hipico check the log files for errors", true
		}
		// Stay in modeCmd, just flag for one-shot LLM call — processMessage handles it
		return "", false
	}

	return "", false
}

// executeCmdMode executes a shell command in command mode via ExecTool.
// Output is formatted as a console code block for channel display.
func (al *AgentLoop) executeCmdMode(ctx context.Context, agent *AgentInstance, content, sessionKey, channel, chatID string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", nil
	}

	// Handle cd command specially
	if content == "cd" || strings.HasPrefix(content, "cd ") {
		return al.handleCdCommand(content, sessionKey, agent), nil
	}

	// Handle :edit command
	if content == ":edit" || strings.HasPrefix(content, ":edit ") {
		workDir := al.getSessionWorkDir(sessionKey)
		if workDir == "" {
			workDir = agent.Workspace
		}
		return al.handleEditCommand(content, workDir), nil
	}

	// Intercept interactive editors
	if msg := interceptEditor(content); msg != "" {
		return msg, nil
	}

	// Get working directory
	workDir := al.getSessionWorkDir(sessionKey)
	if workDir == "" {
		workDir = agent.Workspace
	}

	// For ls commands, ensure -l flag so we can parse file types
	execCmd := content
	if isLsCommand(content) {
		execCmd = ensureLsLong(content)
	}

	// Execute via ExecTool
	result := agent.Tools.ExecuteWithContext(ctx, "exec", map[string]any{
		"command":     execCmd,
		"working_dir": workDir,
	}, channel, chatID, nil)

	displayDir := shortenHomePath(workDir)
	output := result.ForLLM
	if output == "" {
		output = "(no output)"
	}

	// Colorize ls output with emoji type indicators
	if isLsCommand(content) {
		output = formatLsOutput(output)
	}

	// Format as console code block: prompt line + output (show original command, not modified)
	return fmt.Sprintf("```\n%s$ %s\n%s\n```", displayDir, content, output), nil
}

// handleCdCommand handles the cd command in command mode, updating per-session working directory.
// Special paths (cd, cd ~, cd /, cd /xxx) are redirected to the workspace directory for safety.
func (al *AgentLoop) handleCdCommand(content, sessionKey string, agent *AgentInstance) string {
	parts := strings.Fields(content)
	workspace := agent.Workspace
	var target string

	if len(parts) < 2 || parts[1] == "~" || parts[1] == "/" {
		// cd, cd ~, cd / → always go to workspace
		target = workspace
	} else {
		target = parts[1]
		// Expand ~ prefix: treat ~ as workspace root (not $HOME)
		if strings.HasPrefix(target, "~/") {
			target = workspace + target[1:]
		}
		// Absolute paths (e.g. cd /etc) → redirect to workspace
		if filepath.IsAbs(target) {
			target = workspace
		}
		// Resolve relative paths
		if !filepath.IsAbs(target) {
			currentDir := al.getSessionWorkDir(sessionKey)
			if currentDir == "" {
				currentDir = workspace
			}
			target = filepath.Join(currentDir, target)
		}
	}

	target = filepath.Clean(target)

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Sprintf("cd: %s: No such file or directory", target)
	}
	if !info.IsDir() {
		return fmt.Sprintf("cd: %s: Not a directory", target)
	}

	al.setSessionWorkDir(sessionKey, target)
	return fmt.Sprintf("```\n%s$\n```", shortenHomePath(target))
}

// shortenHomePath replaces the user's home directory prefix with ~ for display.
func shortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

// handleEditCommand processes :edit commands for file viewing and editing in cmd mode.
// Syntax:
//
//	:edit                           → show usage
//	:edit <file>                    → show file with line numbers
//	:edit <file> <N> <text>         → replace line N
//	:edit <file> +<N> <text>        → insert after line N
//	:edit <file> -<N>               → delete line N
//	:edit <file> -m """<content>""" → write full content (create if needed)
func (al *AgentLoop) handleEditCommand(content, workDir string) string {
	raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(content), ":edit"))
	if raw == "" {
		return editUsage()
	}

	// Split on first newline to get the command line
	firstLine := raw
	if idx := strings.Index(raw, "\n"); idx != -1 {
		firstLine = raw[:idx]
	}

	parts := strings.Fields(firstLine)
	if len(parts) == 0 {
		return editUsage()
	}

	filename := resolveEditPath(parts[0], workDir)

	// :edit <file> — show file content
	if len(parts) == 1 && !strings.Contains(raw, "\n") {
		return editShowFile(filename)
	}

	// :edit <file> -m """..."""
	if len(parts) >= 2 && parts[1] == "-m" {
		return editMultiline(filename, raw)
	}

	// Line operations: N text, +N text, -N
	if len(parts) >= 2 {
		// Get raw text after the line-op token (preserves original spacing)
		afterFile := strings.TrimSpace(firstLine[len(parts[0]):])
		return editLineOp(filename, afterFile)
	}

	return editUsage()
}

func resolveEditPath(name, workDir string) string {
	if strings.HasPrefix(name, "~/") {
		home, _ := os.UserHomeDir()
		return home + name[1:]
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(workDir, name)
}

func editUsage() string {
	return "Usage:\n" +
		"  :edit <file>              — view file\n" +
		"  :edit <file> <N> <text>   — replace line N\n" +
		"  :edit <file> +<N> <text>  — insert after line N\n" +
		"  :edit <file> -<N>         — delete line N\n" +
		"  :edit <file> -m \"\"\"       — write content\n" +
		"  <content>\n" +
		"  \"\"\""
}

func editShowFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("File not found: %s\nUse :edit %s -m \"\"\" to create it.", shortenHomePath(path), filepath.Base(path))
		}
		return fmt.Sprintf("Error reading file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line that Split produces
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	const maxLines = 50
	var b strings.Builder
	b.WriteString(fmt.Sprintf("``` %s (%d lines)\n", filepath.Base(path), len(lines)))
	if len(lines) <= maxLines {
		for i, line := range lines {
			b.WriteString(fmt.Sprintf("%4d│ %s\n", i+1, line))
		}
	} else {
		for i := 0; i < maxLines; i++ {
			b.WriteString(fmt.Sprintf("%4d│ %s\n", i+1, lines[i]))
		}
		b.WriteString(fmt.Sprintf("  ...│ (%d more lines)\n", len(lines)-maxLines))
	}
	b.WriteString("```")
	return b.String()
}

func editMultiline(filename, raw string) string {
	// raw = `<file> -m """..."""`
	start := strings.Index(raw, `"""`)
	if start == -1 {
		return editUsage()
	}
	rest := raw[start+3:]
	// Trim leading newline after opening """
	rest = strings.TrimPrefix(rest, "\n")

	// Find closing """
	end := strings.LastIndex(rest, `"""`)
	if end == -1 || end == 0 {
		// No closing triple-quote — use entire rest as content
		end = len(rest)
	}
	content := rest[:end]

	// Ensure trailing newline
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Create parent dirs if needed
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	lineCount := strings.Count(content, "\n")
	return fmt.Sprintf("```\n✓ Wrote %d lines → %s\n```", lineCount, shortenHomePath(filename))
}

func editLineOp(filename, rawArgs string) string {
	rawArgs = strings.TrimSpace(rawArgs)
	// Split into op token and text
	spaceIdx := strings.IndexByte(rawArgs, ' ')
	var op, text string
	if spaceIdx == -1 {
		op = rawArgs
	} else {
		op = rawArgs[:spaceIdx]
		text = rawArgs[spaceIdx+1:]
	}

	var lineNum int
	var action string // "replace", "insert", "delete"
	var err error

	if strings.HasPrefix(op, "+") {
		action = "insert"
		lineNum, err = strconv.Atoi(op[1:])
	} else if strings.HasPrefix(op, "-") {
		action = "delete"
		lineNum, err = strconv.Atoi(op[1:])
	} else {
		action = "replace"
		lineNum, err = strconv.Atoi(op)
	}
	if err != nil || lineNum < 1 {
		return "Invalid line number. Use a positive integer."
	}

	// Read existing file
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("File not found: %s", shortenHomePath(filename))
		}
		return fmt.Sprintf("Error reading file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	switch action {
	case "delete":
		if lineNum > len(lines) {
			return fmt.Sprintf("Line %d out of range (file has %d lines).", lineNum, len(lines))
		}
		deleted := lines[lineNum-1]
		lines = append(lines[:lineNum-1], lines[lineNum:]...)
		if err := os.WriteFile(filename, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			return fmt.Sprintf("Error writing file: %v", err)
		}
		return fmt.Sprintf("```\n✓ Deleted line %d: %s\n(%d lines remaining)\n```", lineNum, deleted, len(lines))

	case "replace":
		if text == "" {
			return "Usage: :edit <file> <N> <text>"
		}
		if lineNum > len(lines) {
			return fmt.Sprintf("Line %d out of range (file has %d lines).", lineNum, len(lines))
		}
		old := lines[lineNum-1]
		lines[lineNum-1] = text
		if err := os.WriteFile(filename, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			return fmt.Sprintf("Error writing file: %v", err)
		}
		return fmt.Sprintf("```\n✓ Line %d replaced\n  was: %s\n  now: %s\n```", lineNum, old, text)

	case "insert":
		if text == "" {
			return "Usage: :edit <file> +<N> <text>"
		}
		if lineNum > len(lines) {
			lineNum = len(lines) // insert at end
		}
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:lineNum]...)
		newLines = append(newLines, text)
		newLines = append(newLines, lines[lineNum:]...)
		if err := os.WriteFile(filename, []byte(strings.Join(newLines, "\n")+"\n"), 0o644); err != nil {
			return fmt.Sprintf("Error writing file: %v", err)
		}
		return fmt.Sprintf("```\n✓ Inserted after line %d: %s\n(%d lines total)\n```", lineNum, text, len(newLines))
	}

	return editUsage()
}

// interceptEditor detects interactive editor commands and returns a helpful redirect message.
func interceptEditor(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	name := parts[0]
	switch name {
	case "vim", "vi", "nvim", "nano", "emacs", "pico", "joe", "mcedit":
		return fmt.Sprintf("⚠ %s requires a terminal and cannot run here.\nUse :edit instead:\n\n"+
			":edit <file>              — view file\n"+
			":edit <file> -m \"\"\"       — write content\n"+
			"<content>\n"+
			"\"\"\"\n\n"+
			"Type :help for all commands.", name)
	}
	return ""
}

// isLsCommand checks if a shell command is an ls invocation.
func isLsCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	return cmd == "ls" || strings.HasPrefix(cmd, "ls ")
}

// ensureLsLong injects -l into an ls command if not already present,
// so the output always contains permission strings for type detection.
func ensureLsLong(cmd string) string {
	parts := strings.Fields(cmd)
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "-") && !strings.HasPrefix(p, "--") && strings.ContainsRune(p, 'l') {
			return cmd // already has -l
		}
	}
	// "ls" → "ls -l", "ls -a /tmp" → "ls -l -a /tmp"
	if len(parts) == 1 {
		return "ls -l"
	}
	return "ls -l " + strings.Join(parts[1:], " ")
}

// formatLsOutput adds emoji type indicators to ls -l style output lines.
func formatLsOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = formatLsLine(line)
	}
	return strings.Join(lines, "\n")
}

// formatLsLine adds an emoji prefix to a single ls -l output line based on file type.
func formatLsLine(line string) string {
	// Skip empty lines, "total" line, and lines too short to be ls -l
	if line == "" || strings.HasPrefix(line, "total ") || len(line) < 10 {
		return line
	}

	// Check if line starts with a permission string (e.g. drwxr-xr-x)
	perms := line[:10]
	if !isPermString(perms) {
		return line
	}

	fileType := perms[0]
	var emoji string
	switch fileType {
	case 'd':
		emoji = "\U0001F4C1" // 📁
	case 'l':
		emoji = "\U0001F517" // 🔗
	case 'b', 'c':
		emoji = "\U0001F4BE" // 💾
	case 'p', 's':
		emoji = "\U0001F50C" // 🔌
	default:
		// Regular file: check executable bit (owner/group/other x positions)
		if perms[3] == 'x' || perms[6] == 'x' || perms[9] == 'x' {
			emoji = "\u26A1" // ⚡
		} else {
			emoji = fileEmojiByExt(line)
		}
	}

	return emoji + " " + line
}

// isPermString checks if a 10-char string looks like a Unix permission string.
func isPermString(s string) bool {
	if len(s) != 10 {
		return false
	}
	// First char: file type
	switch s[0] {
	case '-', 'd', 'l', 'b', 'c', 'p', 's':
	default:
		return false
	}
	// Remaining 9 chars: rwx or - (plus s/S/t/T for setuid/setgid/sticky)
	for _, c := range s[1:] {
		switch c {
		case 'r', 'w', 'x', '-', 's', 'S', 't', 'T':
		default:
			return false
		}
	}
	return true
}

// fileEmojiByExt returns an emoji based on the file extension found in an ls -l line.
func fileEmojiByExt(line string) string {
	// Extract filename: last whitespace-delimited field (for symlinks, take before " -> ")
	name := line
	if idx := strings.LastIndex(line, " -> "); idx != -1 {
		name = line[:idx]
	}
	if idx := strings.LastIndex(name, " "); idx != -1 {
		name = name[idx+1:]
	}
	name = strings.ToLower(name)

	ext := filepath.Ext(name)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".bmp", ".ico", ".tiff":
		return "\U0001F5BC" // 🖼
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a":
		return "\U0001F3B5" // 🎵
	case ".mp4", ".avi", ".mkv", ".mov", ".webm", ".flv", ".wmv":
		return "\U0001F3AC" // 🎬
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".zst", ".tgz":
		return "\U0001F4E6" // 📦
	default:
		return "\U0001F4C4" // 📄
	}
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

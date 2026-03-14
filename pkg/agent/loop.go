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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
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
	"github.com/sipeed/picoclaw/pkg/voice"
)

type AgentLoop struct {
	bus                   *bus.MessageBus
	cfg                   *config.Config
	registry              *AgentRegistry
	state                 *state.Manager
	running               atomic.Bool
	summarizing           sync.Map
	fallback              *providers.FallbackChain
	channelManager        *channels.Manager
	mediaStore            media.MediaStore
	transcriber           voice.Transcriber
	synthesizer           voice.Synthesizer
	cmdRegistry           *commands.Registry
	perAgentToolFactories []func(agentID string, agent *AgentInstance) tools.Tool
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string   // Session identifier for history/context
	Channel         string   // Target channel for tool execution
	ChatID          string   // Target chat ID for tool execution
	UserMessage     string   // User message content (may include prefix)
	Media           []string // media:// refs from inbound message
	DefaultResponse string   // Response when LLM returns empty
	EnableSummary   bool     // Whether to trigger summarization
	SendResponse    bool     // Whether to send response via bus
	NoHistory       bool     // If true, don't load session history (for heartbeat)
}

const (
	defaultResponse           = "I've completed processing but have no response to give. Increase `max_tool_iterations` in config.json."
	sessionKeyAgentPrefix     = "agent:"
	metadataKeyAccountID      = "account_id"
	metadataKeyGuildID        = "guild_id"
	metadataKeyTeamID         = "team_id"
	metadataKeyParentPeerKind = "parent_peer_kind"
	metadataKeyParentPeerID   = "parent_peer_id"
	metadataKeyUserProfileKey = "user_profile_key"
	metadataKeyAdminEscalated = "admin_escalated"
)

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
		cmdRegistry: commands.NewRegistry(commands.BuiltinDefinitions()),
	}

	return al
}

func normalizeOutboundChannel(channel string, cfg *config.Config) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	switch channel {
	case "wa", "whatsapp":
		if cfg != nil && cfg.Channels.WhatsApp.UseNative {
			return "whatsapp_native"
		}
		return "whatsapp"
	default:
		return channel
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
		registerSharedToolsForAgent(cfg, msgBus, registry, provider, agentID, agent)
	}
}

func registerSharedToolsForAgent(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
	agentID string,
	agent *AgentInstance,
) {
	if agent == nil {
		return
	}

	// Web tools
	if cfg.Tools.IsToolEnabled("web") {
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
			SearXNGBaseURL:       cfg.Tools.Web.SearXNG.BaseURL,
			SearXNGMaxResults:    cfg.Tools.Web.SearXNG.MaxResults,
			SearXNGEnabled:       cfg.Tools.Web.SearXNG.Enabled,
			GLMSearchAPIKey:      cfg.Tools.Web.GLMSearch.APIKey,
			GLMSearchBaseURL:     cfg.Tools.Web.GLMSearch.BaseURL,
			GLMSearchEngine:      cfg.Tools.Web.GLMSearch.SearchEngine,
			GLMSearchMaxResults:  cfg.Tools.Web.GLMSearch.MaxResults,
			GLMSearchEnabled:     cfg.Tools.Web.GLMSearch.Enabled,
			Proxy:                cfg.Tools.Web.Proxy,
		})
		if err != nil {
			logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{"error": err.Error()})
		} else if searchTool != nil {
			agent.Tools.Register(searchTool)
		}
	}
	if cfg.Tools.IsToolEnabled("web_fetch") {
		fetchTool, err := tools.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy, cfg.Tools.Web.FetchLimitBytes)
		if err != nil {
			logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
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
			channel = normalizeOutboundChannel(channel, cfg)
			if strings.TrimSpace(channel) == "" {
				return fmt.Errorf("message target channel is required")
			}
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pubCancel()
			return msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
				Channel: channel,
				ChatID:  chatID,
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
		)
		agent.Tools.Register(sendFileTool)
	}

	// Skill discovery and installation tools
	skillsEnabled := cfg.Tools.IsToolEnabled("skills")
	findSkillsEnabled := cfg.Tools.IsToolEnabled("find_skills")
	installSkillsEnabled := cfg.Tools.IsToolEnabled("install_skill")
	if skillsEnabled && (findSkillsEnabled || installSkillsEnabled) {
		registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
			MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
			ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
		})

		if findSkillsEnabled {
			searchCache := skills.NewSearchCache(
				cfg.Tools.Skills.SearchCache.MaxSize,
				time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
			)
			agent.Tools.Register(tools.NewFindSkillsTool(registryMgr, searchCache))
		}

		if installSkillsEnabled {
			agent.Tools.Register(tools.NewInstallSkillTool(registryMgr, agent.Workspace))
		}
	}

	// Spawn tool with allowlist checker
	if cfg.Tools.IsToolEnabled("spawn") {
		if cfg.Tools.IsToolEnabled("subagent") {
			subagentManager := tools.NewSubagentManager(provider, agent.Model, agent.Workspace)
			subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)
			spawnTool := tools.NewSpawnTool(subagentManager)
			currentAgentID := agentID
			spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
				return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
			})
			agent.Tools.Register(spawnTool)
		} else {
			logger.WarnCF("agent", "spawn tool requires subagent to be enabled", nil)
		}
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	// Initialize MCP servers for all agents
	if al.cfg.Tools.IsToolEnabled("mcp") {
		mcpManager := mcp.NewManager()
		// Ensure MCP connections are cleaned up on exit, regardless of initialization success
		// This fixes resource leak when LoadFromMCPConfig partially succeeds then fails
		defer func() {
			if err := mcpManager.Close(); err != nil {
				logger.ErrorCF("agent", "Failed to close MCP manager",
					map[string]any{
						"error": err.Error(),
					})
			}
		}()

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
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	for _, agentID := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(agentID); ok {
			agent.Tools.Register(tool)
		}
	}
}

func (al *AgentLoop) RegisterPerAgentTool(factory func(agentID string, agent *AgentInstance) tools.Tool) {
	if factory == nil {
		return
	}
	al.perAgentToolFactories = append(al.perAgentToolFactories, factory)
	for _, agentID := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(agentID); ok {
			al.applyPerAgentToolFactory(agentID, agent, factory)
		}
	}
}

func (al *AgentLoop) applyPerAgentToolFactory(agentID string, agent *AgentInstance, factory func(agentID string, agent *AgentInstance) tools.Tool) {
	if agent == nil || factory == nil {
		return
	}
	if tool := factory(agentID, agent); tool != nil {
		agent.Tools.Register(tool)
	}
}

func (al *AgentLoop) applyPerAgentTools(agent *AgentInstance) {
	if agent == nil {
		return
	}
	for _, factory := range al.perAgentToolFactories {
		al.applyPerAgentToolFactory(agent.ID, agent, factory)
	}
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
	al.restoreRecentPreviews()
}

// SetMediaStore injects a MediaStore for media lifecycle management.
func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s

	// Propagate store to send_file tools in all agents.
	al.registry.ForEachTool("send_file", func(t tools.Tool) {
		if sf, ok := t.(*tools.SendFileTool); ok {
			sf.SetMediaStore(s)
		}
	})
}

// SetTranscriber injects a voice transcriber for agent-level audio transcription.
func (al *AgentLoop) SetTranscriber(t voice.Transcriber) {
	al.transcriber = t
}

func (al *AgentLoop) SetSynthesizer(s voice.Synthesizer) {
	al.synthesizer = s
}

var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)
var voiceURLRe = regexp.MustCompile(`https?://[^\s)]+`)
var voiceCodeBlockRe = regexp.MustCompile("(?s)```.*?```")
var voiceInlineCodeRe = regexp.MustCompile("`[^`]+`")
var voiceWhitespaceRe = regexp.MustCompile(`\s+`)

// transcribeAudioInMessage resolves audio media refs, transcribes them, and
// replaces audio annotations in msg.Content with the transcribed text.
func (al *AgentLoop) transcribeAudioInMessage(ctx context.Context, msg bus.InboundMessage) bus.InboundMessage {
	if al.transcriber == nil || al.mediaStore == nil || len(msg.Media) == 0 {
		return msg
	}

	var transcriptions []string
	remainingMedia := make([]string, 0, len(msg.Media))
	for _, ref := range msg.Media {
		path, meta, err := al.mediaStore.ResolveWithMeta(ref)
		if err != nil {
			logger.WarnCF("voice", "Failed to resolve media ref", map[string]any{"ref": ref, "error": err})
			remainingMedia = append(remainingMedia, ref)
			continue
		}
		if !utils.IsAudioFile(meta.Filename, meta.ContentType) {
			remainingMedia = append(remainingMedia, ref)
			continue
		}
		result, err := al.transcriber.Transcribe(ctx, path)
		if err != nil {
			logger.WarnCF("voice", "Transcription failed", map[string]any{"ref": ref, "error": err})
			remainingMedia = append(remainingMedia, ref)
			continue
		}
		if text := strings.TrimSpace(result.Text); text != "" {
			transcriptions = append(transcriptions, text)
			continue
		}
		remainingMedia = append(remainingMedia, ref)
	}

	if len(transcriptions) == 0 {
		return msg
	}

	msg.Media = remainingMedia
	msg.Content = mergeVoiceTranscriptions(msg.Content, transcriptions)
	return msg
}

func mergeVoiceTranscriptions(content string, transcriptions []string) string {
	if len(transcriptions) == 0 {
		return content
	}

	idx := 0
	newContent := audioAnnotationRe.ReplaceAllStringFunc(content, func(match string) string {
		if idx >= len(transcriptions) {
			return match
		}
		text := strings.TrimSpace(transcriptions[idx])
		idx++
		if text == "" {
			return match
		}
		return "Voice note transcript: " + text
	})

	for ; idx < len(transcriptions); idx++ {
		text := strings.TrimSpace(transcriptions[idx])
		if text == "" {
			continue
		}
		if strings.TrimSpace(newContent) != "" {
			newContent += "\n"
		}
		newContent += "Voice note transcript: " + text
	}

	return strings.TrimSpace(newContent)
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

	msg = al.transcribeAudioInMessage(ctx, msg)

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	route, agent, routeErr := al.resolveMessageRoute(msg)
	if routeErr == nil {
		agent, routeErr = al.resolveEffectiveAgent(msg, route, agent)
	}

	// Commands are checked before requiring a successful route.
	// Global commands (/help, /show, /switch) work even when routing fails;
	// context-dependent commands check their own Runtime fields and report
	// "unavailable" when the required capability is nil.
	commandSessionKey := msg.SessionKey
	if routeErr == nil {
		commandSessionKey = resolveScopeKey(route, msg.SessionKey, isAdminEscalation(msg))
	}
	if response, handled := al.handleCommand(ctx, msg, agent, commandSessionKey); handled {
		return response, nil
	}

	if routeErr != nil {
		return "", routeErr
	}

	// Reset message-tool state for this round so we don't skip publishing due to a previous round.
	if tool, ok := agent.Tools.Get("message"); ok {
		if resetter, ok := tool.(interface{ ResetSentInRound() }); ok {
			resetter.ResetSentInRound()
		}
	}

	// Resolve session key from route, while preserving explicit agent-scoped keys.
	scopeKey := resolveScopeKey(route, msg.SessionKey, isAdminEscalation(msg))
	sessionKey := scopeKey

	logger.InfoCF("agent", "Routed message",
		map[string]any{
			"agent_id":      agent.ID,
			"scope_key":     scopeKey,
			"session_key":   sessionKey,
			"matched_by":    route.MatchedBy,
			"route_agent":   route.AgentID,
			"route_channel": route.Channel,
		})

	response, err := al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		Media:           msg.Media,
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
		SendResponse:    false,
	})
	if err != nil {
		return "", err
	}
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(interface{ HasSentToCurrentRound() bool }); ok && mt.HasSentToCurrentRound() {
			logger.InfoCF("agent", "Suppressing duplicate final response after same-chat message tool send", map[string]any{
				"agent_id": agent.ID,
				"channel":  msg.Channel,
				"chat_id":  msg.ChatID,
			})
			return "", nil
		}
	}
	return response, nil
}

func (al *AgentLoop) resolveMessageRoute(msg bus.InboundMessage) (routing.ResolvedRoute, *AgentInstance, error) {
	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel:    msg.Channel,
		AccountID:  inboundMetadata(msg, metadataKeyAccountID),
		Peer:       extractPeer(msg),
		ParentPeer: extractParentPeer(msg),
		GuildID:    inboundMetadata(msg, metadataKeyGuildID),
		TeamID:     inboundMetadata(msg, metadataKeyTeamID),
	})

	agent, ok := al.registry.GetAgent(route.AgentID)
	if !ok {
		agent = al.registry.GetDefaultAgent()
	}
	if agent == nil {
		return routing.ResolvedRoute{}, nil, fmt.Errorf("no agent available for route (agent_id=%s)", route.AgentID)
	}

	return route, agent, nil
}

func resolveScopeKey(route routing.ResolvedRoute, msgSessionKey string, adminEscalated bool) string {
	if msgSessionKey != "" && strings.HasPrefix(msgSessionKey, sessionKeyAgentPrefix) {
		return msgSessionKey
	}
	if adminEscalated && route.MainSessionKey != "" {
		return route.MainSessionKey
	}
	return route.SessionKey
}

func (al *AgentLoop) resolveEffectiveAgent(
	msg bus.InboundMessage,
	route routing.ResolvedRoute,
	baseAgent *AgentInstance,
) (*AgentInstance, error) {
	if baseAgent == nil {
		return nil, fmt.Errorf("no base agent available")
	}
	if isAdminEscalation(msg) {
		return baseAgent, nil
	}

	profileKey := inboundMetadata(msg, metadataKeyUserProfileKey)
	if profileKey == "" {
		profileKey = deriveUserProfileKey(msg)
	}
	if profileKey == "" {
		return baseAgent, nil
	}

	baseAgentID := baseAgent.ID
	if baseAgentID == "" {
		baseAgentID = route.AgentID
	}
	profileAgent, created, err := al.registry.GetOrCreateProfileAgent(baseAgentID, profileKey)
	if err != nil {
		return nil, err
	}
	if created {
		registerSharedToolsForAgent(al.cfg, al.bus, al.registry, baseAgent.Provider, baseAgentID, profileAgent)
		if al.mediaStore != nil {
			if tool, ok := profileAgent.Tools.Get("send_file"); ok {
				if sendFileTool, ok := tool.(*tools.SendFileTool); ok {
					sendFileTool.SetMediaStore(al.mediaStore)
				}
			}
		}
		al.applyPerAgentTools(profileAgent)
		logger.InfoCF("agent", "Created isolated profile agent", map[string]any{
			"agent_id":       baseAgentID,
			"profile_key":    profileKey,
			"profile_folder": profileAgent.Workspace,
		})
	}
	return profileAgent, nil
}

func isAdminEscalation(msg bus.InboundMessage) bool {
	return strings.EqualFold(strings.TrimSpace(inboundMetadata(msg, metadataKeyAdminEscalated)), "true")
}

func deriveUserProfileKey(msg bus.InboundMessage) string {
	if msg.Sender.CanonicalID != "" {
		return msg.Sender.CanonicalID
	}
	if msg.Sender.Platform != "" && msg.Sender.PlatformID != "" {
		return fmt.Sprintf("%s:%s", msg.Sender.Platform, msg.Sender.PlatformID)
	}
	if msg.Channel != "" && inboundMetadata(msg, "user_id") != "" {
		return fmt.Sprintf("%s:%s", msg.Channel, inboundMetadata(msg, "user_id"))
	}
	return ""
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
	// 0. Record last channel for heartbeat notifications (skip internal channels and cli)
	if opts.Channel != "" && opts.ChatID != "" {
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

	// 1. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = agent.Sessions.GetHistory(opts.SessionKey)
		summary = agent.Sessions.GetSummary(opts.SessionKey)
	}
	attachmentCtx := buildAttachmentContext(opts.Media, al.mediaStore)
	userMessageForModel := opts.UserMessage
	if attachmentCtx != "" {
		if strings.TrimSpace(userMessageForModel) != "" {
			userMessageForModel = strings.TrimSpace(userMessageForModel) + "\n\n" + attachmentCtx
		} else {
			userMessageForModel = attachmentCtx
		}
	}
	messages := agent.ContextBuilder.BuildMessages(
		history,
		summary,
		userMessageForModel,
		opts.Media,
		opts.Channel,
		opts.ChatID,
	)

	// Resolve media:// refs to base64 data URLs (streaming)
	maxMediaSize := al.cfg.Agents.Defaults.GetMaxMediaSize()
	messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)

	// 2. Save user message to session
	agent.Sessions.AddMessage(opts.SessionKey, "user", userMessageForModel)

	// 3. Run LLM iteration loop
	finalContent, iteration, err := al.runLLMIteration(ctx, agent, messages, opts)
	if err != nil {
		return "", err
	}

	// If last tool had ForUser content and we already sent it, we might not need to send final response
	// This is controlled by the tool's Silent flag and ForUser content

	// 4. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// Guard against raw tool-call payload leakage in Telegram responses.
	if opts.Channel == "telegram" {
		finalContent = sanitizeLeakedToolPayload(finalContent)
	}

	// 5. Save final assistant message to session
	agent.Sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	agent.Sessions.Save(opts.SessionKey)

	// 6. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 7. Optional: send response via bus
	if opts.SendResponse {
		voiceSent, supplemental := al.maybeSendVoiceReply(ctx, opts, finalContent)
		if !voiceSent {
			al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: finalContent,
			})
		} else if strings.TrimSpace(supplemental) != "" {
			al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: supplemental,
			})
		}
	}

	// 8. Log response
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

	// Determine effective model tier for this conversation turn.
	// selectCandidates evaluates routing once and the decision is sticky for
	// all tool-follow-up iterations within the same turn so that a multi-step
	// tool chain doesn't switch models mid-way through.
	activeCandidates, activeModel := al.selectCandidates(agent, opts.UserMessage, messages, opts.Media)

	maxIterations := effectiveConversationIterations(opts.Channel, agent.MaxIterations)

	for iteration < maxIterations {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       maxIterations,
			})

		// Build tool definitions
		providerToolDefs := agent.Tools.ToProviderDefs()

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

		// Call LLM with fallback chain if multiple candidates are configured.
		var response *providers.LLMResponse
		var err error

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

		callLLM := func() (*providers.LLMResponse, error) {
			if len(activeCandidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(
					ctx,
					activeCandidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return al.callCandidateChat(ctx, agent, messages, providerToolDefs, provider, model, llmOpts)
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
			primaryProvider := al.cfg.Agents.Defaults.Provider
			if len(activeCandidates) > 0 && strings.TrimSpace(activeCandidates[0].Provider) != "" {
				primaryProvider = activeCandidates[0].Provider
			}
			return al.callCandidateChat(ctx, agent, messages, providerToolDefs, primaryProvider, activeModel, llmOpts)
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

				if retry == 0 {
					logger.InfoCF("agent", "Suppressing user-facing compression notice", map[string]any{
						"channel": opts.Channel,
						"chat_id": opts.ChatID,
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
		// Some providers occasionally emit wrapped tool-call payloads in text,
		// e.g. CALL>[{"name":"read_file","arguments":{...}}]ALL>.
		// Convert those payloads into executable tool calls instead of replying with raw wrappers.
		if len(response.ToolCalls) == 0 {
			if wrapped := extractWrappedToolCalls(response.Content); len(wrapped) > 0 {
				response.ToolCalls = wrapped
				response.Content = ""
			}
		}

		// Check if no tool calls - then check reasoning content if any
		if len(response.ToolCalls) == 0 {
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
			norm := providers.NormalizeToolCall(tc)
			norm.Arguments = rewriteToolArguments(norm.Name, norm.Arguments, agent, opts.Media, al.mediaStore)
			normalizedToolCalls = append(normalizedToolCalls, norm)
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

		// Execute tool calls in parallel
		type indexedAgentResult struct {
			result *tools.ToolResult
			tc     providers.ToolCall
		}

		agentResults := make([]indexedAgentResult, len(normalizedToolCalls))
		var wg sync.WaitGroup

		for i, tc := range normalizedToolCalls {
			agentResults[i].tc = tc

			wg.Add(1)
			go func(idx int, tc providers.ToolCall) {
				defer wg.Done()

				argsJSON, _ := json.Marshal(tc.Arguments)
				argsPreview := utils.Truncate(string(argsJSON), 200)
				logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
					map[string]any{
						"agent_id":  agent.ID,
						"tool":      tc.Name,
						"iteration": iteration,
					})

				// Create async callback for tools that implement AsyncExecutor.
				// When the background work completes, this publishes the result
				// as an inbound system message so processSystemMessage routes it
				// back to the user via the normal agent loop.
				asyncCallback := func(_ context.Context, result *tools.ToolResult) {
					// Send ForUser content directly to the user (immediate feedback),
					// mirroring the synchronous tool execution path.
					if !result.Silent && result.ForUser != "" {
						outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer outCancel()
						_ = al.bus.PublishOutbound(outCtx, bus.OutboundMessage{
							Channel: opts.Channel,
							ChatID:  opts.ChatID,
							Content: result.ForUser,
						})
					}

					// Determine content for the agent loop (ForLLM or error).
					content := result.ForLLM
					if content == "" && result.Err != nil {
						content = result.Err.Error()
					}
					if content == "" {
						return
					}
					if opts.NoHistory {
						logger.InfoCF("agent", "Async tool completed during no-history run; suppressing publish", map[string]any{
							"tool":        tc.Name,
							"content_len": len(content),
							"channel":     opts.Channel,
						})
						return
					}

					logger.InfoCF("agent", "Async tool completed, publishing result",
						map[string]any{
							"tool":        tc.Name,
							"content_len": len(content),
							"channel":     opts.Channel,
						})

					pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer pubCancel()
					_ = al.bus.PublishInbound(pubCtx, bus.InboundMessage{
						Channel:  "system",
						SenderID: fmt.Sprintf("async:%s", tc.Name),
						ChatID:   fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID),
						Content:  content,
					})
				}

				toolResult := agent.Tools.ExecuteWithContext(
					ctx,
					tc.Name,
					tc.Arguments,
					opts.Channel,
					opts.ChatID,
					asyncCallback,
				)
				agentResults[idx].result = toolResult
			}(i, tc)
		}
		wg.Wait()

		// Process results in original order (send to user, save to session)
		for _, r := range agentResults {
			// Send ForUser content to user immediately if not Silent
			if !r.result.Silent && r.result.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: r.result.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]any{
						"tool":        r.tc.Name,
						"content_len": len(r.result.ForUser),
					})
			}

			// If tool returned media refs, publish them as outbound media
			if len(r.result.Media) > 0 {
				parts := make([]bus.MediaPart, 0, len(r.result.Media))
				for _, ref := range r.result.Media {
					part := bus.MediaPart{Ref: ref}
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
			contentForLLM := r.result.ForLLM
			if contentForLLM == "" && r.result.Err != nil {
				contentForLLM = r.result.Err.Error()
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: r.tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
		}
	}

	return finalContent, iteration, nil
}

// selectCandidates returns the model candidates and resolved model name to use
// for a conversation turn. When model routing is configured and the incoming
// message scores below the complexity threshold, it returns the light model
// candidates instead of the primary ones.
//
// The returned (candidates, model) pair is used for all LLM calls within one
// turn — tool follow-up iterations use the same tier as the initial call so
// that a multi-step tool chain doesn't switch models mid-way.
func (al *AgentLoop) selectCandidates(
	agent *AgentInstance,
	userMsg string,
	history []providers.Message,
	currentMedia []string,
) (candidates []providers.FallbackCandidate, model string) {
	if len(agent.ImageCandidates) > 0 && turnContainsImageMedia(currentMedia, al.mediaStore) {
		imageModel := agent.ImageModel
		if imageModel == "" {
			imageModel = agent.ImageCandidates[0].Model
		}
		logger.InfoCF("agent", "Model routing: image model selected",
			map[string]any{
				"agent_id":    agent.ID,
				"image_model": imageModel,
			})
		return agent.ImageCandidates, imageModel
	}

	if agent.Router == nil || len(agent.LightCandidates) == 0 {
		return agent.Candidates, agent.Model
	}

	_, usedLight, score := agent.Router.SelectModel(userMsg, history, agent.Model)
	if !usedLight {
		logger.DebugCF("agent", "Model routing: primary model selected",
			map[string]any{
				"agent_id":  agent.ID,
				"score":     score,
				"threshold": agent.Router.Threshold(),
			})
		return agent.Candidates, agent.Model
	}

	logger.InfoCF("agent", "Model routing: light model selected",
		map[string]any{
			"agent_id":    agent.ID,
			"light_model": agent.Router.LightModel(),
			"score":       score,
			"threshold":   agent.Router.Threshold(),
		})
	return agent.LightCandidates, agent.Router.LightModel()
}

func turnContainsImageMedia(refs []string, store media.MediaStore) bool {
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "data:image/") {
			return true
		}
		if !strings.HasPrefix(lower, "media://") || store == nil {
			continue
		}
		_, meta, err := store.ResolveWithMeta(trimmed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(meta.ContentType)), "image/") {
			return true
		}
		switch strings.ToLower(filepath.Ext(strings.TrimSpace(meta.Filename))) {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".heic", ".heif":
			return true
		}
	}
	return false
}

func (al *AgentLoop) callCandidateChat(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	providerName string,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	provider, resolvedModel, created, err := al.providerForCandidate(providerName, model)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		if agent == nil || agent.Provider == nil {
			return nil, fmt.Errorf("provider not available for %s/%s", providerName, model)
		}
		provider = agent.Provider
		if strings.TrimSpace(resolvedModel) == "" {
			resolvedModel = model
		}
	}
	if created {
		if stateful, ok := provider.(providers.StatefulProvider); ok {
			defer stateful.Close()
		}
	}
	return provider.Chat(ctx, messages, toolDefs, resolvedModel, options)
}

func (al *AgentLoop) providerForCandidate(providerName, model string) (providers.LLMProvider, string, bool, error) {
	modelCfg, err := al.modelConfigForCandidate(providerName, model)
	if err != nil {
		return nil, "", false, err
	}
	provider, resolvedModel, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", false, err
	}
	return provider, resolvedModel, true, nil
}

func (al *AgentLoop) modelConfigForCandidate(providerName, model string) (*config.ModelConfig, error) {
	normalizedProvider := providers.NormalizeProvider(providerName)
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		return nil, fmt.Errorf("candidate model is empty for provider %q", providerName)
	}

	for i := range al.cfg.ModelList {
		entry := al.cfg.ModelList[i]
		protocol, modelID := providers.ExtractProtocol(strings.TrimSpace(entry.Model))
		if providers.NormalizeProvider(protocol) == normalizedProvider && strings.TrimSpace(modelID) == trimmedModel {
			copy := entry
			al.applyProviderDefaults(&copy, normalizedProvider)
			return &copy, nil
		}
	}

	modelCfg := &config.ModelConfig{
		ModelName: normalizedProvider + "-" + sanitizeModelName(trimmedModel),
		Model:     normalizedProvider + "/" + trimmedModel,
	}

	switch normalizedProvider {
	case "openrouter":
		modelCfg.APIKey = al.cfg.Providers.OpenRouter.APIKey
		modelCfg.APIBase = al.cfg.Providers.OpenRouter.APIBase
		modelCfg.Proxy = al.cfg.Providers.OpenRouter.Proxy
	case "gemini":
		modelCfg.APIKey = al.cfg.Providers.Gemini.APIKey
		modelCfg.APIBase = al.cfg.Providers.Gemini.APIBase
		modelCfg.Proxy = al.cfg.Providers.Gemini.Proxy
	case "deepseek":
		modelCfg.APIKey = al.cfg.Providers.DeepSeek.APIKey
		modelCfg.APIBase = al.cfg.Providers.DeepSeek.APIBase
		modelCfg.Proxy = al.cfg.Providers.DeepSeek.Proxy
	case "groq":
		modelCfg.APIKey = al.cfg.Providers.Groq.APIKey
		modelCfg.APIBase = al.cfg.Providers.Groq.APIBase
		modelCfg.Proxy = al.cfg.Providers.Groq.Proxy
	case "openai":
		modelCfg.APIKey = al.cfg.Providers.OpenAI.APIKey
		modelCfg.APIBase = al.cfg.Providers.OpenAI.APIBase
		modelCfg.Proxy = al.cfg.Providers.OpenAI.Proxy
		modelCfg.AuthMethod = al.cfg.Providers.OpenAI.AuthMethod
	case "anthropic":
		modelCfg.APIKey = al.cfg.Providers.Anthropic.APIKey
		modelCfg.APIBase = al.cfg.Providers.Anthropic.APIBase
		modelCfg.Proxy = al.cfg.Providers.Anthropic.Proxy
		modelCfg.AuthMethod = al.cfg.Providers.Anthropic.AuthMethod
	default:
		return nil, fmt.Errorf("no model_list entry for candidate %s/%s", normalizedProvider, trimmedModel)
	}

	if strings.TrimSpace(modelCfg.APIKey) == "" && strings.TrimSpace(modelCfg.APIBase) == "" && strings.TrimSpace(modelCfg.AuthMethod) == "" {
		return nil, fmt.Errorf("credentials not configured for candidate %s/%s", normalizedProvider, trimmedModel)
	}
	return modelCfg, nil
}

func (al *AgentLoop) applyProviderDefaults(modelCfg *config.ModelConfig, providerName string) {
	if modelCfg == nil {
		return
	}
	switch providers.NormalizeProvider(providerName) {
	case "openrouter":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.OpenRouter.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.OpenRouter.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.OpenRouter.Proxy
		}
	case "gemini":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.Gemini.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.Gemini.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.Gemini.Proxy
		}
	case "deepseek":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.DeepSeek.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.DeepSeek.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.DeepSeek.Proxy
		}
	case "groq":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.Groq.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.Groq.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.Groq.Proxy
		}
	case "openai":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.OpenAI.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.OpenAI.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.OpenAI.Proxy
		}
		if strings.TrimSpace(modelCfg.AuthMethod) == "" {
			modelCfg.AuthMethod = al.cfg.Providers.OpenAI.AuthMethod
		}
	case "anthropic":
		if strings.TrimSpace(modelCfg.APIKey) == "" {
			modelCfg.APIKey = al.cfg.Providers.Anthropic.APIKey
		}
		if strings.TrimSpace(modelCfg.APIBase) == "" {
			modelCfg.APIBase = al.cfg.Providers.Anthropic.APIBase
		}
		if strings.TrimSpace(modelCfg.Proxy) == "" {
			modelCfg.Proxy = al.cfg.Providers.Anthropic.Proxy
		}
		if strings.TrimSpace(modelCfg.AuthMethod) == "" {
			modelCfg.AuthMethod = al.cfg.Providers.Anthropic.AuthMethod
		}
	}
}

func sanitizeModelName(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return "model"
	}
	replacer := strings.NewReplacer("/", "-", ":", "-", ".", "-", " ", "-", "_", "-")
	model = replacer.Replace(model)
	model = strings.Trim(model, "-")
	if model == "" {
		return "model"
	}
	return model
}

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := agent.ContextWindow * agent.SummarizeTokenPercent / 100

	if len(newHistory) > agent.SummarizeMessageThreshold || tokenEstimate > threshold {
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

func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	agent *AgentInstance,
	sessionKey string,
) (string, bool) {
	commandText := msg.Content
	if !commands.HasCommandPrefix(commandText) {
		if inferred := al.inferImplicitTelegramCommand(msg, commandText, agent); inferred != "" {
			commandText = inferred
		} else if shouldAutoRunShellInTelegram(msg, commandText) {
			commandText = "/run " + strings.TrimSpace(commandText)
		}
	}
	if !commands.HasCommandPrefix(commandText) {
		return "", false
	}

	if al.cmdRegistry == nil {
		return "", false
	}

	rt := al.buildCommandsRuntime(agent, sessionKey)
	executor := commands.NewExecutor(al.cmdRegistry, rt)

	var commandReply string
	result := executor.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     commandText,
		Reply: func(text string) error {
			commandReply = text
			return nil
		},
	})

	switch result.Outcome {
	case commands.OutcomeHandled:
		if result.Err != nil {
			return mapCommandError(result), true
		}
		if commandReply != "" {
			return commandReply, true
		}
		return "", true
	default: // OutcomePassthrough — let the message fall through to LLM
		return "", false
	}
}

func (al *AgentLoop) buildCommandsRuntime(agent *AgentInstance, sessionKey string) *commands.Runtime {
	rt := &commands.Runtime{
		Config:          al.cfg,
		ListAgentIDs:    al.registry.ListAgentIDs,
		ListDefinitions: al.cmdRegistry.Definitions,
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel %s not found or not enabled", value)
			}
			return nil
		},
		ExecuteShell: func(ctx context.Context, command string) (string, error) {
			if !strings.EqualFold(strings.TrimSpace(os.Getenv("PICOCLAW_DASHBOARD_ALLOW_SHELL")), "true") {
				return "", fmt.Errorf("shell execution disabled; set PICOCLAW_DASHBOARD_ALLOW_SHELL=true")
			}
			if strings.TrimSpace(command) == "" {
				return "", fmt.Errorf("command is required")
			}
			cmdCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(cmdCtx, "bash", "-lc", command)
			if agent != nil && strings.TrimSpace(agent.Workspace) != "" {
				cmd.Dir = agent.Workspace
			}
			out, err := cmd.CombinedOutput()
			text := formatShellOutputForChat(command, strings.TrimSpace(string(out)))
			if len(text) > 6000 {
				text = text[:6000] + "\n... (truncated)"
			}
			return text, err
		},
		GetRecentPreviews: func() []commands.PreviewInfo {
			return al.recentPreviewInfos(agent)
		},
		ClearHistory: func() error {
			if strings.TrimSpace(sessionKey) == "" {
				return fmt.Errorf("session key is empty")
			}
			if agent == nil || agent.Sessions == nil {
				return fmt.Errorf("sessions not initialized for agent")
			}
			agent.Sessions.SetHistory(sessionKey, make([]providers.Message, 0))
			agent.Sessions.SetSummary(sessionKey, "")
			agent.Sessions.Save(sessionKey)
			return nil
		},
	}
	if agent != nil {
		rt.GetModelInfo = func() (string, string) {
			return agent.Model, al.cfg.Agents.Defaults.Provider
		}
		rt.SwitchModel = func(value string) (string, error) {
			oldModel := agent.Model
			candidates := resolveAgentCandidates(al.cfg, al.cfg.Agents.Defaults.Provider, value, agent.Fallbacks)
			if len(candidates) == 0 {
				return oldModel, fmt.Errorf("model %q not found", value)
			}
			agent.Model = candidates[0].Model
			agent.Candidates = candidates
			return oldModel, nil
		}
	}
	return rt
}

func (al *AgentLoop) recentPreviewInfos(agent *AgentInstance) []commands.PreviewInfo {
	if agent == nil || strings.TrimSpace(agent.Workspace) == "" {
		return nil
	}
	items, err := tools.LoadRecentPreviews(agent.Workspace)
	if err != nil || len(items) == 0 {
		return nil
	}
	result := make([]commands.PreviewInfo, 0, len(items))
	for _, item := range items {
		refreshed, ok := al.refreshRecentPreview(agent.Workspace, item)
		if !ok {
			continue
		}
		result = append(result, commands.PreviewInfo{
			Slug:         refreshed.Slug,
			LocalURL:     refreshed.LocalURL,
			TailscaleURL: refreshed.TailscaleURL,
			Root:         refreshed.Root,
			Entry:        refreshed.Entry,
		})
	}
	return result
}

func (al *AgentLoop) restoreRecentPreviews() {
	if al == nil || al.channelManager == nil || al.registry == nil {
		return
	}

	seen := make(map[string]struct{})
	for _, workspace := range al.recentPreviewWorkspaces() {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			continue
		}
		if _, ok := seen[workspace]; ok {
			continue
		}
		seen[workspace] = struct{}{}

		items, err := tools.LoadRecentPreviews(workspace)
		if err != nil || len(items) == 0 {
			continue
		}
		for _, item := range items {
			al.refreshRecentPreview(workspace, item)
		}
	}
}

func (al *AgentLoop) recentPreviewWorkspaces() []string {
	if al == nil || al.registry == nil {
		return nil
	}

	var workspaces []string
	appendWorkspace := func(workspace string) {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			return
		}
		workspaces = append(workspaces, workspace)
		profilesDir := filepath.Join(workspace, "profiles")
		entries, err := os.ReadDir(profilesDir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			workspaces = append(workspaces, filepath.Join(profilesDir, entry.Name()))
		}
	}

	al.registry.mu.RLock()
	defer al.registry.mu.RUnlock()
	for _, agent := range al.registry.agents {
		if agent != nil {
			appendWorkspace(agent.Workspace)
		}
	}
	for _, agent := range al.registry.profileAgents {
		if agent != nil {
			appendWorkspace(agent.Workspace)
		}
	}
	return workspaces
}

func (al *AgentLoop) refreshRecentPreview(workspace string, item tools.HostedPreview) (tools.HostedPreview, bool) {
	root := strings.TrimSpace(item.Root)
	workspace = strings.TrimSpace(workspace)
	if workspace == "" || root == "" || root == string(filepath.Separator) || al.channelManager == nil {
		return tools.HostedPreview{}, false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return tools.HostedPreview{}, false
	}

	slug := strings.TrimSpace(item.Slug)
	entry := strings.TrimSpace(item.Entry)
	actualSlug, tailscaleURL, localURL, err := al.channelManager.PublishPreview(root, entry, slug)
	if err != nil {
		return tools.HostedPreview{}, false
	}

	refreshed := item
	refreshed.Slug = actualSlug
	refreshed.Root = root
	refreshed.Entry = entry
	refreshed.LocalURL = localURL
	refreshed.TailscaleURL = tailscaleURL
	refreshed.UpdatedAt = time.Now().Format(time.RFC3339)
	_ = tools.SaveRecentPreview(workspace, &refreshed)
	return refreshed, true
}

func formatShellOutputForChat(command, output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return trimmed
	}
	lowerCommand := strings.ToLower(command)
	switch {
	case strings.Contains(lowerCommand, "gws gmail +send"):
		return formatGWSSendOutput(command, trimmed)
	case strings.Contains(lowerCommand, "gws gmail +triage"):
		if formatted := formatGWSTableOutput(trimmed); formatted != "" {
			return formatted
		}
	}
	return trimmed
}

func formatGWSSendOutput(command, output string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return output
	}
	lines := []string{"Mail sent."}
	if to := strings.TrimSpace(extractCLIFlagValue(command, "--to")); to != "" {
		lines = append(lines, "To: "+to)
	}
	if subject := strings.TrimSpace(extractCLIFlagValue(command, "--subject")); subject != "" {
		lines = append(lines, "Subject: "+subject)
	}
	if id := strings.TrimSpace(fmt.Sprint(payload["id"])); id != "" && id != "<nil>" {
		lines = append(lines, "ID: "+id)
	}
	if threadID := strings.TrimSpace(fmt.Sprint(payload["threadId"])); threadID != "" && threadID != "<nil>" {
		lines = append(lines, "Thread: "+threadID)
	}
	return strings.Join(lines, "\n")
}

func extractCLIFlagValue(command, flag string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(flag) + `\s+(?:"([^"]+)"|'([^']+)'|(\S+))`)
	match := re.FindStringSubmatch(command)
	if len(match) == 0 {
		return ""
	}
	for _, candidate := range match[1:] {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func formatGWSTableOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		return ""
	}
	header := strings.ToLower(lines[0])
	if !strings.Contains(header, "date") || !strings.Contains(header, "from") || !strings.Contains(header, "subject") {
		return ""
	}
	splitter := regexp.MustCompile(`\s{2,}`)
	formatted := []string{"Latest emails:"}
	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := splitter.Split(line, 4)
		if len(parts) < 4 {
			continue
		}
		formatted = append(formatted, fmt.Sprintf("• %s\nFrom: %s\nSubject: %s\nID: %s", compactWhitespace(parts[0]), compactWhitespace(parts[1]), compactWhitespace(parts[3]), compactWhitespace(parts[2])))
	}
	if len(formatted) == 1 {
		return ""
	}
	return strings.Join(formatted, "\n")
}

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func shouldSendVoiceReply(opts processOptions) bool {
	if !strings.EqualFold(strings.TrimSpace(opts.Channel), "telegram") {
		return false
	}
	return strings.Contains(opts.UserMessage, "Voice note transcript:")
}

func splitVoiceReplyContent(content string) (string, string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", ""
	}
	urls := uniqueStrings(voiceURLRe.FindAllString(trimmed, -1))
	codeBlocks := uniqueStrings(voiceCodeBlockRe.FindAllString(trimmed, -1))
	inlineCodes := uniqueStrings(voiceInlineCodeRe.FindAllString(trimmed, -1))

	spoken := voiceCodeBlockRe.ReplaceAllString(trimmed, " ")
	spoken = voiceInlineCodeRe.ReplaceAllString(spoken, " ")
	spoken = voiceURLRe.ReplaceAllString(spoken, " ")
	spoken = strings.NewReplacer("**", "", "__", "", "~~", "", "#", "", ">", "").Replace(spoken)
	spoken = voiceWhitespaceRe.ReplaceAllString(strings.TrimSpace(spoken), " ")
	spoken = trimSpeechText(spoken, 900)

	var supplemental []string
	if len(urls) > 0 {
		supplemental = append(supplemental, "Links:\n• "+strings.Join(urls, "\n• "))
	}
	var details []string
	for _, block := range codeBlocks {
		block = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(block, "```"), "```"))
		if block != "" {
			details = append(details, block)
		}
	}
	for _, code := range inlineCodes {
		code = strings.Trim(strings.TrimSpace(code), "`")
		if code != "" {
			details = append(details, code)
		}
	}
	if len(details) > 0 {
		supplemental = append(supplemental, "Code/details:\n```\n"+strings.Join(details, "\n\n")+"\n```")
	}
	return spoken, strings.TrimSpace(strings.Join(supplemental, "\n\n"))
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func trimSpeechText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	runes := []rune(text)
	cut := string(runes[:limit])
	if idx := strings.LastIndexAny(cut, ".!?\n"); idx > 120 {
		cut = cut[:idx+1]
	}
	return strings.TrimSpace(cut)
}

func (al *AgentLoop) maybeSendVoiceReply(ctx context.Context, opts processOptions, finalContent string) (bool, string) {
	if !shouldSendVoiceReply(opts) || al.synthesizer == nil || al.mediaStore == nil || al.bus == nil {
		return false, ""
	}
	spoken, supplemental := splitVoiceReplyContent(finalContent)
	if strings.TrimSpace(spoken) == "" {
		return false, supplemental
	}
	resp, err := al.synthesizer.Synthesize(ctx, spoken)
	if err != nil {
		logger.WarnCF("voice", "Voice reply synthesis failed", map[string]any{"channel": opts.Channel, "chat_id": opts.ChatID, "error": err.Error()})
		return false, supplemental
	}
	scope := channels.BuildMediaScope(opts.Channel, opts.ChatID, fmt.Sprintf("voice-reply-%d", time.Now().UnixNano()))
	ref, err := al.mediaStore.Store(resp.AudioFilePath, media.MediaMeta{Filename: resp.Filename, ContentType: resp.ContentType, Source: "tool:voice-reply:" + al.synthesizer.Name()}, scope)
	if err != nil {
		logger.WarnCF("voice", "Failed to store synthesized voice reply", map[string]any{"error": err.Error()})
		return false, supplemental
	}
	if err := al.bus.PublishOutboundMedia(ctx, bus.OutboundMediaMessage{Channel: opts.Channel, ChatID: opts.ChatID, Parts: []bus.MediaPart{{Type: "voice", Ref: ref, Filename: resp.Filename, ContentType: resp.ContentType}}}); err != nil {
		logger.WarnCF("voice", "Failed to publish synthesized voice reply", map[string]any{"error": err.Error()})
		return false, supplemental
	}
	return true, supplemental
}

func extractWrappedToolCalls(content string) []providers.ToolCall {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	lower := strings.ToLower(trimmed)
	start := strings.Index(lower, "call>")
	end := strings.LastIndex(lower, "all>")
	if start == -1 || end == -1 || end <= start+5 {
		return nil
	}

	inner := strings.TrimSpace(trimmed[start+5 : end])
	if strings.HasPrefix(inner, ">") {
		inner = strings.TrimSpace(strings.TrimPrefix(inner, ">"))
	}
	if inner == "" {
		return nil
	}

	type wrappedCall struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	var calls []wrappedCall
	if err := json.Unmarshal([]byte(inner), &calls); err != nil || len(calls) == 0 {
		// Try single object form.
		var single wrappedCall
		if err2 := json.Unmarshal([]byte(inner), &single); err2 != nil || strings.TrimSpace(single.Name) == "" {
			return nil
		}
		calls = []wrappedCall{single}
	}

	out := make([]providers.ToolCall, 0, len(calls))
	for i, c := range calls {
		name := strings.TrimSpace(strings.ToLower(c.Name))
		switch name {
		case "readfile":
			name = "read_file"
		case "writefile":
			name = "write_file"
		case "listdir":
			name = "list_dir"
		}
		if name == "" {
			continue
		}
		if c.Arguments == nil {
			c.Arguments = map[string]any{}
		}
		out = append(out, providers.ToolCall{
			ID:        fmt.Sprintf("wrapped_%d", i+1),
			Type:      "function",
			Name:      name,
			Arguments: c.Arguments,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func preferToolCapableCandidates(
	candidates []providers.FallbackCandidate,
	model string,
	messages []providers.Message,
	hasToolCalls bool,
) ([]providers.FallbackCandidate, string, bool) {
	if len(candidates) < 2 {
		return nil, "", false
	}
	if !hasToolCalls && !messagesContainMedia(messages) {
		return nil, "", false
	}
	if !isOpenRouterFreeCandidate(candidates[0]) {
		return nil, "", false
	}

	reordered := append([]providers.FallbackCandidate(nil), candidates...)
	first := reordered[0]
	reordered = append(reordered[1:], first)
	return reordered, reordered[0].Model, true
}

func messagesContainMedia(messages []providers.Message) bool {
	for _, msg := range messages {
		if len(msg.Media) > 0 {
			return true
		}
	}
	return false
}

func isOpenRouterFreeCandidate(candidate providers.FallbackCandidate) bool {
	return strings.EqualFold(strings.TrimSpace(candidate.Provider), "openrouter") &&
		strings.EqualFold(strings.TrimSpace(candidate.Model), "openrouter/free")
}

func rewriteToolArguments(name string, args map[string]any, agent *AgentInstance, mediaRefs []string, store media.MediaStore) map[string]any {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "read_file":
		return rewriteReadFileArgsFromMedia(args, mediaRefs, store)
	case "host_preview":
		return rewriteHostPreviewArgs(args, agent)
	case "send_file":
		return rewriteSendFileArgs(args, agent, mediaRefs, store)
	case "message":
		return rewriteMessageArgs(args)
	default:
		return args
	}
}

func rewriteHostPreviewArgs(args map[string]any, agent *AgentInstance) map[string]any {
	if len(args) == 0 {
		return args
	}
	if pathVal, _ := args["path"].(string); strings.TrimSpace(pathVal) != "" {
		if info, err := os.Stat(strings.TrimSpace(pathVal)); err == nil && info.IsDir() {
			return args
		}
	}
	raw := rawArgumentString(args)
	if raw == "" {
		return args
	}
	if candidate := extractExistingPathFromRaw(raw, candidateSearchRoots(agent), true); candidate != "" {
		args["path"] = candidate
		if _, ok := args["entry"]; !ok && strings.Contains(strings.ToLower(raw), "index.html") {
			args["entry"] = "index.html"
		}
		delete(args, "raw")
	}
	return args
}

func rewriteSendFileArgs(args map[string]any, agent *AgentInstance, mediaRefs []string, store media.MediaStore) map[string]any {
	args = rewriteReadFileArgsFromMedia(args, mediaRefs, store)
	if len(args) == 0 {
		return args
	}
	if pathVal, _ := args["path"].(string); strings.TrimSpace(pathVal) != "" {
		if _, err := os.Stat(strings.TrimSpace(pathVal)); err == nil {
			if _, ok := args["filename"]; !ok {
				args["filename"] = filepath.Base(strings.TrimSpace(pathVal))
			}
			delete(args, "raw")
			return args
		}
	}
	raw := rawArgumentString(args)
	if raw == "" {
		return args
	}
	if candidate := extractExistingPathFromRaw(raw, candidateSearchRoots(agent), false); candidate != "" {
		args["path"] = candidate
		if _, ok := args["filename"]; !ok {
			args["filename"] = filepath.Base(candidate)
		}
		delete(args, "raw")
	}
	return args
}

func rewriteMessageArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return args
	}
	raw := rawArgumentString(args)
	if raw == "" {
		return args
	}
	for _, key := range []string{"channel", "chat_id", "content"} {
		if value, _ := args[key].(string); strings.TrimSpace(value) == "" {
			if decoded := extractSimpleStringField(raw, key); decoded != "" {
				args[key] = decoded
			}
		}
	}
	if content, _ := args["content"].(string); strings.TrimSpace(content) != "" {
		delete(args, "raw")
	}
	return args
}

func rawArgumentString(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	if raw, _ := args["raw"].(string); strings.TrimSpace(raw) != "" {
		return strings.TrimSpace(raw)
	}
	buf, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(buf))
}

func extractSimpleStringField(raw, key string) string {
	re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*"((?:\\.|[^"])*)"`, regexp.QuoteMeta(key)))
	match := re.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	encoded := match[1]
	var decoded string
	if err := json.Unmarshal([]byte(`"`+encoded+`"`), &decoded); err == nil {
		return strings.TrimSpace(decoded)
	}
	return strings.TrimSpace(strings.ReplaceAll(encoded, `\"`, `"`))
}

func extractExistingPathFromRaw(raw string, roots []string, wantDir bool) string {
	pathPattern := regexp.MustCompile(`/(?:root|tmp|home)[^"'{}\s,]+`)
	for _, match := range pathPattern.FindAllString(raw, -1) {
		candidate := strings.TrimSpace(match)
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() == wantDir {
			return candidate
		}
	}
	basePattern := regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._-]{2,}`)
	for _, token := range basePattern.FindAllString(raw, -1) {
		if strings.Contains(token, ".") || strings.Contains(token, "-") || strings.Contains(token, "_") {
			if candidate := searchPathByBaseName(token, roots, wantDir); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func candidateSearchRoots(agent *AgentInstance) []string {
	roots := []string{}
	seen := map[string]bool{}
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			seen[path] = true
			roots = append(roots, path)
		}
	}
	if agent != nil {
		add(agent.Workspace)
		add(filepath.Join(agent.Workspace, "projects"))
		parent := filepath.Dir(agent.Workspace)
		add(parent)
		add(filepath.Join(parent, "projects"))
		grandparent := filepath.Dir(parent)
		add(grandparent)
		add(filepath.Join(grandparent, "projects"))
	}
	add("/root/.picoclaw/workspace")
	add("/root/.picoclaw/workspace/projects")
	return roots
}

func searchPathByBaseName(name string, roots []string, wantDir bool) string {
	for _, root := range roots {
		var found string
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() != wantDir {
				return nil
			}
			if strings.EqualFold(info.Name(), name) {
				found = path
				return filepath.SkipDir
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

func rewriteReadFileArgsFromMedia(args map[string]any, mediaRefs []string, store media.MediaStore) map[string]any {
	if len(args) == 0 || len(mediaRefs) == 0 || store == nil {
		return args
	}

	pathVal, _ := args["path"].(string)
	pathVal = strings.TrimSpace(pathVal)
	rawVal, _ := args["raw"].(string)
	rawVal = strings.TrimSpace(rawVal)
	if pathVal == "" && rawVal == "" {
		return args
	}

	type mediaCandidate struct {
		ref  string
		path string
		name string
	}
	cands := make([]mediaCandidate, 0, len(mediaRefs))
	for _, ref := range mediaRefs {
		if !strings.HasPrefix(ref, "media://") {
			continue
		}
		localPath, meta, err := store.ResolveWithMeta(ref)
		if err != nil || strings.TrimSpace(localPath) == "" {
			continue
		}
		name := strings.TrimSpace(meta.Filename)
		if name == "" {
			name = filepath.Base(localPath)
		}
		cands = append(cands, mediaCandidate{ref: ref, path: localPath, name: name})
	}
	if len(cands) == 0 {
		return args
	}

	if rawVal != "" {
		lowerRaw := strings.ToLower(rawVal)
		for _, c := range cands {
			if strings.Contains(lowerRaw, strings.ToLower(c.name)) ||
				strings.Contains(lowerRaw, strings.ToLower(filepath.Base(c.path))) ||
				strings.Contains(lowerRaw, strings.ToLower(c.path)) {
				args["path"] = c.path
				delete(args, "raw")
				return args
			}
		}
		if len(cands) == 1 {
			args["path"] = cands[0].path
			delete(args, "raw")
			return args
		}
	}

	if pathVal == "" {
		if len(cands) == 1 {
			args["path"] = cands[0].path
		}
		return args
	}

	if strings.HasPrefix(pathVal, "media://") {
		for _, c := range cands {
			if c.ref == pathVal {
				args["path"] = c.path
				return args
			}
		}
	}

	base := strings.ToLower(filepath.Base(pathVal))
	for _, c := range cands {
		if base != "" && strings.ToLower(filepath.Base(c.path)) == base {
			args["path"] = c.path
			return args
		}
		if base != "" && strings.ToLower(c.name) == base {
			args["path"] = c.path
			return args
		}
	}

	if len(cands) == 1 {
		args["path"] = cands[0].path
	}

	return args
}

func effectiveConversationIterations(channel string, base int) int {
	if base <= 0 {
		base = 1
	}
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "telegram", "whatsapp", "whatsapp_native":
		if base < 14 {
			return 14
		}
	}
	return base
}

func sanitizeLeakedToolPayload(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return content
	}

	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "tool_call") || strings.Contains(lower, `{"tool_calls"`) {
		return ""
	}

	// Catch wrapper formats like CALL>[{...}]ALL> / OLCALL>
	if strings.Contains(lower, "call>") || strings.Contains(lower, "all>") {
		if strings.Contains(lower, `"name"`) && strings.Contains(lower, `"arguments"`) {
			return ""
		}
	}
	if strings.EqualFold(trimmed, "OLCALL>") {
		return ""
	}

	if strings.Contains(trimmed, `"name"`) && strings.Contains(trimmed, `"arguments"`) {
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, ">") || strings.HasPrefix(lower, "call>") {
			return ""
		}
	}

	return content
}

func (al *AgentLoop) inferImplicitTelegramCommand(msg bus.InboundMessage, text string, agent *AgentInstance) string {
	if msg.Channel != "telegram" || msg.Peer.Kind != "direct" {
		return ""
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" || commands.HasCommandPrefix(trimmed) {
		return ""
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case "gws", "gws?", "google workspace", "google workspace?", "gmail?", "mail?":
		return "/check gws"
	}

	if al.shouldRecallRecentPreview(lower, agent) || al.shouldRecallRecentPreviewFromConversation(lower, agent) {
		return "/show previews"
	}

	if !googleWorkspaceAvailable() {
		return ""
	}

	if recipient, ok := inferEmailSendRecipient(trimmed, lower); ok {
		subject := "Pico test mail"
		body := "Test mail sent by Pico."
		return fmt.Sprintf("/exec gws gmail +send --to %q --subject %q --body %q", recipient, subject, body)
	}

	if strings.Contains(lower, "email") || strings.Contains(lower, "emails") || strings.Contains(lower, "mail") || strings.Contains(lower, "mails") || strings.Contains(lower, "inbox") {
		maxItems := extractRequestedCount(lower, 5)
		query := "in:anywhere"
		if strings.Contains(lower, "unread") {
			query = "is:unread"
		}
		return fmt.Sprintf("/exec gws gmail +triage --max %d --query %q --format table", maxItems, query)
	}

	if strings.Contains(lower, "agenda") || strings.Contains(lower, "calendar") || strings.Contains(lower, "events") {
		args := []string{"/exec", "gws", "calendar", "+agenda", "--format", "table"}
		switch {
		case strings.Contains(lower, "today"):
			args = append(args, "--today")
		case strings.Contains(lower, "tomorrow"):
			args = append(args, "--tomorrow")
		case strings.Contains(lower, "week"):
			args = append(args, "--week")
		default:
			args = append(args, "--days", fmt.Sprintf("%d", extractRequestedCount(lower, 3)))
		}
		return strings.Join(args, " ")
	}

	if strings.Contains(lower, "drive") && (strings.Contains(lower, "file") || strings.Contains(lower, "files")) {
		pageSize := extractRequestedCount(lower, 10)
		params := fmt.Sprintf("{\"pageSize\":%d}", pageSize)
		return fmt.Sprintf("/exec gws drive files list --params %q --format table", params)
	}

	return ""
}

func (al *AgentLoop) shouldRecallRecentPreview(lower string, agent *AgentInstance) bool {
	if agent == nil || strings.TrimSpace(agent.Workspace) == "" {
		return false
	}
	wantsURL := strings.Contains(lower, "url") || strings.Contains(lower, "urls") || strings.Contains(lower, "link") || strings.Contains(lower, "links")
	wantsRecent := strings.Contains(lower, "recent") || strings.Contains(lower, "latest") || strings.Contains(lower, "most recent") || strings.Contains(lower, "current")
	wantsHost := strings.Contains(lower, "host") || strings.Contains(lower, "serve") || strings.Contains(lower, "open")
	wantsPreview := strings.Contains(lower, "preview") || strings.Contains(lower, "site") || strings.Contains(lower, "website") || strings.Contains(lower, "app") || strings.Contains(lower, "build")
	wantsStatus := strings.Contains(lower, "status") || strings.Contains(lower, "going") || strings.Contains(lower, "howd") || strings.Contains(lower, "hows") || strings.Contains(lower, "how's")
	if !wantsPreview {
		return false
	}
	if !wantsURL && !(wantsRecent && wantsHost) && !wantsStatus {
		return false
	}
	return len(al.recentPreviewInfos(agent)) > 0
}

func (al *AgentLoop) shouldRecallRecentPreviewFromConversation(lower string, agent *AgentInstance) bool {
	if agent == nil || strings.TrimSpace(agent.Workspace) == "" {
		return false
	}
	wantsURL := strings.Contains(lower, "url") || strings.Contains(lower, "urls") || strings.Contains(lower, "link") || strings.Contains(lower, "links")
	wantsRecent := strings.Contains(lower, "recent") || strings.Contains(lower, "latest") || strings.Contains(lower, "most recent") || strings.Contains(lower, "current")
	wantsHost := strings.Contains(lower, "host") || strings.Contains(lower, "serve") || strings.Contains(lower, "open")
	wantsPreview := strings.Contains(lower, "preview") || strings.Contains(lower, "site") || strings.Contains(lower, "website") || strings.Contains(lower, "app") || strings.Contains(lower, "build")
	wantsStatus := wantsPreview && (strings.Contains(lower, "status") || strings.Contains(lower, "going") || strings.Contains(lower, "howd") || strings.Contains(lower, "hows") || strings.Contains(lower, "how's"))
	if !wantsURL && !(wantsRecent && wantsHost) && !wantsStatus {
		return false
	}
	hints := strings.Contains(lower, "talked") || strings.Contains(lower, "previous") || strings.Contains(lower, "built") || strings.Contains(lower, "that site") || strings.Contains(lower, "that website") || wantsRecent || wantsStatus
	if !hints {
		return false
	}
	return len(al.recentPreviewInfos(agent)) > 0
}

func inferEmailSendRecipient(trimmed, lower string) (string, bool) {
	if !strings.Contains(lower, "send") {
		return "", false
	}
	if !(strings.Contains(lower, "mail") || strings.Contains(lower, "email")) {
		return "", false
	}
	match := regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`).FindString(trimmed)
	if strings.TrimSpace(match) == "" {
		return "", false
	}
	return match, true
}

func extractRequestedCount(text string, fallback int) int {
	match := regexp.MustCompile(`\b(\d{1,2})\b`).FindStringSubmatch(text)
	if len(match) < 2 {
		return fallback
	}
	value := fallback
	fmt.Sscanf(match[1], "%d", &value)
	if value <= 0 {
		return fallback
	}
	if value > 50 {
		return 50
	}
	return value
}

func googleWorkspaceAvailable() bool {
	if _, err := exec.LookPath("gws"); err != nil {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".config", "gws", "credentials.json"))
	return err == nil
}

func shouldAutoRunShellInTelegram(msg bus.InboundMessage, text string) bool {
	if msg.Channel != "telegram" || msg.Peer.Kind != "direct" {
		return false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || commands.HasCommandPrefix(trimmed) {
		return false
	}
	first := strings.ToLower(strings.Fields(trimmed)[0])
	allowed := map[string]bool{
		"gws": true, "gcloud": true, "tailscale": true, "systemctl": true,
		"redis-cli": true, "curl": true, "ssh": true, "ls": true,
		"cat": true, "test": true, "mkdir": true, "chmod": true,
		"chown": true, "journalctl": true, "picoclaw": true, "hostname": true,
	}
	return allowed[first]
}

func mapCommandError(result commands.ExecuteResult) string {
	if result.Command == "" {
		return fmt.Sprintf("Failed to execute command: %v", result.Err)
	}
	return fmt.Sprintf("Failed to execute /%s: %v", result.Command, result.Err)
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

func inboundMetadata(msg bus.InboundMessage, key string) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata[key]
}

// extractParentPeer extracts the parent peer (reply-to) from inbound message metadata.
func extractParentPeer(msg bus.InboundMessage) *routing.RoutePeer {
	parentKind := inboundMetadata(msg, metadataKeyParentPeerKind)
	parentID := inboundMetadata(msg, metadataKeyParentPeerID)
	if parentKind == "" || parentID == "" {
		return nil
	}
	return &routing.RoutePeer{Kind: parentKind, ID: parentID}
}

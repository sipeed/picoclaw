// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent/interfaces"
	"github.com/sipeed/picoclaw/pkg/audio/asr"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AgentLoop struct {
	// Core dependencies
	bus      interfaces.MessageBus
	cfg      *config.Config
	registry *AgentRegistry
	state    *state.Manager

	// Runtime event system
	runtimeEvents      runtimeevents.Bus
	ownsRuntimeEvents  bool
	runtimeEventLogMu  sync.RWMutex
	runtimeEventLogger *runtimeEventLogger
	runtimeEventLogSub runtimeevents.Subscription
	hooks              *HookManager

	// Runtime state
	running        atomic.Bool
	contextManager ContextManager
	fallback       *providers.FallbackChain
	channelManager interfaces.ChannelManager
	mediaStore     media.MediaStore
	transcriber    asr.Transcriber
	cmdRegistry    *commands.Registry
	mcp            mcpRuntime
	hookRuntime    hookRuntime
	steering       *steeringQueue
	pendingSkills  sync.Map
	pendingStops   sync.Map
	mu             sync.RWMutex

	// workerSem limits concurrent turn processing workers.
	workerSem chan struct{}

	// activeTurnStates tracks active turns per session to prevent duplicates.
	activeTurnStates sync.Map
	subTurnCounter   atomic.Int64

	turnSeq        atomic.Uint64
	activeRequests sync.WaitGroup

	reloadFunc func() error

	providerFactory func(*config.ModelConfig) (providers.LLMProvider, string, error)
}

// processOptions configures how a message is processed
type processOptions struct {
	Dispatch                DispatchRequest        // Normalized routed request boundary for this turn
	SessionKey              string                 // Session identifier for history/context
	SessionAliases          []string               // Compatibility aliases for the session key
	Channel                 string                 // Target channel for tool execution
	ChatID                  string                 // Target chat ID for tool execution
	MessageID               string                 // Current inbound platform message ID
	ReplyToMessageID        string                 // Current inbound reply target message ID
	SenderID                string                 // Current sender ID for dynamic context
	SenderDisplayName       string                 // Current sender display name for dynamic context
	UserMessage             string                 // User message content (may include prefix)
	ForcedSkills            []string               // Skills explicitly requested for this message
	SystemPromptOverride    string                 // Override the default system prompt (Used by SubTurns)
	Media                   []string               // media:// refs from inbound message
	InitialSteeringMessages []providers.Message    // Steering messages from refactor/agent
	DefaultResponse         string                 // Response when LLM returns empty
	EnableSummary           bool                   // Whether to trigger summarization
	SendResponse            bool                   // Whether to send response via bus
	AllowInterimPicoPublish bool                   // Whether pico tool-call interim text can be published when SendResponse is false
	SuppressToolFeedback    bool                   // Whether to suppress inline tool feedback messages
	NoHistory               bool                   // If true, don't load session history (for heartbeat)
	SkipInitialSteeringPoll bool                   // If true, skip the steering poll at loop start (used by Continue)
	InboundContext          *bus.InboundContext    // Normalized inbound facts for events/hooks
	RouteResult             *routing.ResolvedRoute // Route decision snapshot for events/hooks
	SessionScope            *session.SessionScope  // Session scope snapshot for events/hooks
}

type continuationTarget struct {
	SessionKey string
	Channel    string
	ChatID     string
}

const (
	defaultResponse            = "The model returned an empty response. This may indicate a provider error or token limit."
	toolLimitResponse          = "I've reached `max_tool_iterations` without a final response. Increase `max_tool_iterations` in config.json if this task needs more tool steps."
	handledToolResponseSummary = "Requested output delivered via tool attachment."
	sessionKeyAgentPrefix      = "agent:"
	pendingTurnPrefix          = "pending-"
	metadataKeyMessageKind     = "message_kind"
	metadataKeyToolCalls       = "tool_calls"
	messageKindThought         = "thought"
	messageKindToolFeedback    = "tool_feedback"
	messageKindToolCalls       = "tool_calls"
	metadataKeyAccountID       = "account_id"
	metadataKeyGuildID         = "guild_id"
	metadataKeyTeamID          = "team_id"
	metadataKeyReplyToMessage  = "reply_to_message_id"
	metadataKeyParentPeerKind  = "parent_peer_kind"
	metadataKeyParentPeerID    = "parent_peer_id"
)

// registerSharedTools registers tools that are shared across all agents (web, message, spawn).

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	if err := al.ensureHooksInitialized(ctx); err != nil {
		return err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return err
	}

	idleTicker := time.NewTicker(100 * time.Millisecond)
	defer idleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-idleTicker.C:
			if !al.running.Load() {
				return nil
			}
		case msg, ok := <-al.bus.InboundChan():
			if !ok {
				return nil
			}

			// Resolve the session key for this message
			sessionKey, agentID, ok := al.resolveSteeringTarget(msg)
			if !ok {
				// Non-routable message (e.g., system) — process immediately.
				// Note: system messages are processed in the main goroutine,
				// so they block the receive loop but guarantee session serialization.
				al.processMessageSync(ctx, msg)
				continue
			}

			// Atomically claim the session key with a unique placeholder sentinel
			// to prevent a TOCTOU race where multiple messages for the same session
			// pass the Load check before either registers.
			// The placeholder ensures GetActiveTurnBySession() never returns nil
			// during turn setup. Each placeholder has a unique turnID to prevent
			// cross-worker cleanup issues.
			placeholder := &turnState{
				turnID: makePendingTurnID(sessionKey, al.turnSeq.Add(1)),
				phase:  TurnPhaseSetup,
			}
			if _, loaded := al.activeTurnStates.LoadOrStore(sessionKey, placeholder); loaded {
				if al.tryHandleStopCommand(ctx, msg, sessionKey) {
					continue
				}

				// Another turn is already active (or reserved) for this session — enqueue
				if err := al.enqueueSteeringMessage(sessionKey, agentID, providers.Message{
					Role:    "user",
					Content: msg.Content,
					Media:   append([]string(nil), msg.Media...),
				}); err != nil {
					logger.WarnCF("agent", "Failed to enqueue steering message",
						map[string]any{
							"error":       err.Error(),
							"channel":     msg.Channel,
							"chat_id":     msg.ChatID,
							"session_key": sessionKey,
						})
				}
				continue
			}

			// Session claimed — spawn a worker goroutine that acquires a semaphore
			// slot. The goroutine is spawned immediately so the main loop keeps
			// draining the inbound channel. The goroutine blocks on the semaphore.
			go func(m bus.InboundMessage) {
				// Acquire semaphore slot (blocks if at capacity)
				select {
				case al.workerSem <- struct{}{}:
					// Got slot, start worker
				case <-ctx.Done():
					// Context canceled while waiting for a slot — clean up the
					// placeholder to prevent session-level deadlock.
					al.activeTurnStates.Delete(sessionKey)
					return
				}

				// Safety-net cleanup: if the placeholder was never replaced by a real
				// turnState (e.g., error before runTurn), delete it here. When runTurn
				// completes normally, clearActiveTurn deletes the real turnState and
				// this becomes a no-op (the key is already gone).
				defer func() {
					if actual, ok := al.activeTurnStates.Load(sessionKey); ok {
						if ts, ok := actual.(*turnState); ok && strings.HasPrefix(ts.turnID, pendingTurnPrefix) {
							// Placeholder still present — runTurn never replaced it.
							al.activeTurnStates.Delete(sessionKey)
						}
					}
				}()

				defer func() {
					if r := recover(); r != nil {
						logger.RecoverPanicNoExit(r)
						logger.ErrorCF("agent", "Worker goroutine panicked",
							map[string]any{
								"session_key": sessionKey,
								"channel":     m.Channel,
								"chat_id":     m.ChatID,
								"panic":       fmt.Sprintf("%v", r),
							})
					}
				}()
				defer func() { <-al.workerSem }() // Release slot

				if al.channelManager != nil {
					defer al.channelManager.InvokeTypingStop(m.Channel, m.ChatID)
				}

				if al.takePendingStop(sessionKey) {
					al.activeTurnStates.Delete(sessionKey)
					target := &continuationTarget{
						SessionKey: sessionKey,
						Channel:    m.Channel,
						ChatID:     m.ChatID,
					}
					continued, continueErr := al.drainQueuedSteeringContinuations(ctx, target)
					if continueErr != nil {
						al.maybePublishError(ctx, m.Channel, m.ChatID, sessionKey, continueErr)
						return
					}
					if continued != "" {
						al.PublishResponseIfNeeded(ctx, target.Channel, target.ChatID, target.SessionKey, continued)
					}
					return
				}

				al.runTurnWithSteering(ctx, m)
			}(msg)

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
		}
	}
}

// processMessageSync processes a message synchronously (for non-routable/system messages).

// runTurnWithSteering runs a complete turn for a message and drains its steering queue.

// maybePublishError publishes an error response unless the error is context.Canceled.
// Returns true if processing should continue (non-cancellation error or no error),
// false if context was canceled and the caller should return.

// publishResponseOrError publishes the response, or an error message if processing failed.

func (al *AgentLoop) Stop() {
	al.running.Store(false)
}

// Close releases resources held by agent session stores. Call after Stop.
func (al *AgentLoop) Close() {
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
	al.closeRuntimeEventLogger()
	if al.runtimeEvents != nil && al.ownsRuntimeEvents {
		if err := al.runtimeEvents.Close(); err != nil {
			logger.ErrorCF("agent", "Failed to close runtime event bus",
				map[string]any{
					"error": err.Error(),
				})
		}
	}
}

// MountHook registers an in-process hook on the agent loop.

// UnmountHook removes a previously registered in-process hook.

type turnEventScope struct {
	agentID    string
	sessionKey string
	turnID     string
	context    *TurnContext
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
				logger.RecoverPanicNoExit(r)
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

	// Also update fallback chain with new config; rebuild rate limiter registry.
	newRL := providers.NewRateLimiterRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			newRL.RegisterCandidates(agent.Candidates)
			newRL.RegisterCandidates(agent.LightCandidates)
		}
	}
	al.fallback = providers.NewFallbackChain(providers.NewCooldownTracker(), newRL)

	al.mu.Unlock()
	al.refreshRuntimeEventLogger(cfg)

	oldMCPManager := al.mcp.reset()
	al.hookRuntime.reset(al)
	configureHookManagerFromConfig(al.hooks, cfg)
	if err := al.ensureHooksInitialized(ctx); err != nil {
		logger.WarnCF("agent", "Configured hooks failed to reinitialize after reload",
			map[string]any{"error": err.Error()})
	}
	if oldMCPManager != nil {
		if err := oldMCPManager.Close(); err != nil {
			logger.WarnCF("agent", "Failed to close previous MCP manager during reload",
				map[string]any{"error": err.Error()})
		}
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		logger.WarnCF("agent", "MCP failed to reinitialize after reload",
			map[string]any{"error": err.Error()})
	}

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

// GetConfig returns the current config (thread-safe)

// SetMediaStore injects a MediaStore for media lifecycle management.

// SetTranscriber injects a voice transcriber for agent-level audio transcription.

// SetReloadFunc sets the callback function for triggering config reload.

var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)

// transcribeAudioInMessage resolves audio media refs, transcribes them, and
// replaces audio annotations in msg.Content with the transcribed text.
// Returns the (possibly modified) message and true if audio was transcribed.

// sendTranscriptionFeedback sends feedback to the user with the result of
// audio transcription if the option is enabled. It uses Manager.SendMessage
// which executes synchronously (rate limiting, splitting, retry) so that
// ordering with the subsequent placeholder is guaranteed.

// inferMediaType determines the media type ("image", "audio", "video", "file")
// from a filename and MIME content type.

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.

// runAgentLoop remains the top-level shell that starts a turn and publishes
// any post-turn work. runTurn owns the full turn lifecycle.
func (al *AgentLoop) runAgentLoop(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
) (string, error) {
	opts = normalizeProcessOptions(opts)

	// Record last channel for heartbeat notifications (skip internal channels and cli)
	if opts.Dispatch.Channel() != "" &&
		opts.Dispatch.ChatID() != "" &&
		!constants.IsInternalChannel(opts.Dispatch.Channel()) {
		channelKey := fmt.Sprintf("%s:%s", opts.Dispatch.Channel(), opts.Dispatch.ChatID())
		if err := al.RecordLastChannel(channelKey); err != nil {
			logger.WarnCF(
				"agent",
				"Failed to record last channel",
				map[string]any{"error": err.Error()},
			)
		}
	}

	ensureSessionMetadata(
		agent.Sessions,
		opts.Dispatch.SessionKey,
		opts.Dispatch.SessionScope,
		opts.Dispatch.SessionAliases,
	)

	turnScope := al.newTurnEventScope(
		agent.ID,
		opts.Dispatch.SessionKey,
		newTurnContext(opts.Dispatch.InboundContext, opts.Dispatch.RouteResult, opts.Dispatch.SessionScope),
	)
	ts := newTurnState(agent, opts, turnScope)
	pipeline := NewPipeline(al)
	result, err := al.runTurn(ctx, ts, pipeline)
	if err != nil {
		return "", err
	}
	if result.status == TurnEndStatusAborted {
		return "", nil
	}

	for _, followUp := range result.followUps {
		if pubErr := al.bus.PublishInbound(ctx, followUp); pubErr != nil {
			logger.WarnCF("agent", "Failed to publish follow-up after turn",
				map[string]any{
					"turn_id": ts.turnID,
					"error":   pubErr.Error(),
				})
		}
	}

	if opts.SendResponse && result.finalContent != "" {
		agentID, sessionKey, scope := outboundTurnMetadata(
			agent.ID,
			opts.Dispatch.SessionKey,
			opts.Dispatch.SessionScope,
		)
		al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Context: outboundContextFromInbound(
				opts.Dispatch.InboundContext,
				opts.Dispatch.Channel(),
				opts.Dispatch.ChatID(),
				opts.Dispatch.ReplyToMessageID(),
			),
			AgentID:      agentID,
			SessionKey:   sessionKey,
			Scope:        scope,
			Content:      result.finalContent,
			ContextUsage: computeContextUsage(agent, opts.Dispatch.SessionKey),
		})
	}

	if result.finalContent != "" {
		responsePreview := utils.Truncate(result.finalContent, 120)
		logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
			map[string]any{
				"agent_id":     agent.ID,
				"session_key":  opts.Dispatch.SessionKey,
				"iterations":   ts.currentIteration(),
				"final_length": len(result.finalContent),
			})
	}

	return result.finalContent, nil
}

// selectCandidates returns the model candidates and resolved model name to use
// for a conversation turn. When model routing is configured and the incoming
// message scores below the complexity threshold, it returns the light model
// candidates instead of the primary ones.
//
// The returned (candidates, model) pair is used for all LLM calls within one
// turn — tool follow-up iterations use the same tier as the initial call so
// that a multi-step tool chain doesn't switch models mid-way.

// resolveContextManager selects the ContextManager implementation based on config.

// GetStartupInfo returns information about loaded tools and skills for logging.

// formatMessagesForLog formats messages for logging

// formatToolsForLog formats tool definitions for logging

// summarizeSession summarizes the conversation history for a session.
// findNearestUserMessage finds the nearest user message to the given index.
// It searches backward first, then forward if no user message is found.
// retryLLMCall calls the LLM with retry logic.
// summarizeBatch summarizes a batch of messages.
// estimateTokens estimates the number of tokens in a message list.
// Counts Content, ToolCalls arguments, and ToolCallID metadata so that
// tool-heavy conversations are not systematically undercounted.

// askSideQuestion handles /btw commands by creating an isolated provider instance
// that doesn't share state with the main conversation provider.

// shallowCloneLLMOptions creates a shallow copy of LLM options map.
// Note: This is a shallow copy - nested maps/slices are shared.

// hasMediaRefs checks if any message has media references.

// isolatedSideQuestionProvider creates a separate provider instance for /btw commands
// to avoid sharing state with the main conversation provider.

// sideQuestionModelConfig resolves the model config for side questions.

// sideQuestionModelName determines which model name to use for side questions.

// modelNameFromIdentityKey extracts the model name from an identity key.

// closeProviderIfStateful closes a provider if it implements StatefulProvider.

// makePendingTurnID generates a unique turn ID for placeholder turns.
// Format: "pending-{sessionKey}-{sequence}"

// isNativeSearchProvider reports whether the given LLM provider implements
// NativeSearchCapable and returns true for SupportsNativeSearch.

// filterClientWebSearch returns a copy of tools with the client-side
// web_search tool removed. Used when native provider search is preferred.

// Helper to extract provider from registry for cleanup

package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
)

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

func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	if msg.Channel != "system" {
		return "", fmt.Errorf("processSystemMessage called with non-system message channel: %s", msg.Channel)
	}

	logger.InfoCF("agent", "Processing system message",

		map[string]any{
			"sender_id": msg.SenderID,

			"chat_id": msg.ChatID,
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
				"sender_id": msg.SenderID,

				"content_len": len(content),

				"channel": originChannel,
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
		Kind: session.TurnReport,

		OriginKey: subagentSessionKey,

		Author: msg.SenderID,

		Messages: []providers.Message{{Role: "user", Content: historyMsg}},
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
		Channel: originChannel,

		ChatID: notifyChatID,

		Content: notification,

		SkipPlaceholder: true,
	})

	logger.InfoCF("agent", "Subagent result injected into session history",

		map[string]any{
			"sender_id": msg.SenderID,

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

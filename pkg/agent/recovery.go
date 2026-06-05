package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

// RecoverUnansweredSessions resumes sessions whose durable history ends with a
// user message. This closes the small gap left after a steering message has
// crossed from ingress spool into session history but the process exits before
// producing the assistant response.
func (al *AgentLoop) RecoverUnansweredSessions(ctx context.Context) int {
	registry := al.GetRegistry()
	if registry == nil {
		return 0
	}

	blockedSessions := al.sessionsWithUnackedInbound(ctx)
	recovered := 0
	agentIDs := registry.ListAgentIDs()
	sort.Strings(agentIDs)
	for _, agentID := range agentIDs {
		agent, ok := registry.GetAgent(agentID)
		if !ok || agent == nil || agent.Sessions == nil {
			continue
		}
		sessionKeys := agent.Sessions.ListSessions()
		sort.Strings(sessionKeys)
		for _, sessionKey := range sessionKeys {
			if ctx.Err() != nil {
				return recovered
			}
			if !sessionNeedsUnansweredRecovery(agent.Sessions.GetHistory(sessionKey)) {
				continue
			}
			if _, blocked := blockedSessions[sessionKey]; blocked {
				logger.DebugCF("agent", "Skipping unanswered recovery for session with unacked inbound spool entry",
					map[string]any{"agent_id": agent.ID, "session_key": sessionKey})
				continue
			}
			if err := al.recoverUnansweredSession(ctx, agent, sessionKey); err != nil {
				logger.WarnCF("agent", "Failed to recover unanswered session", map[string]any{
					"agent_id":     agent.ID,
					"session_key":  sessionKey,
					"error":        err.Error(),
					"recover_kind": "unanswered_session",
				})
				continue
			}
			recovered++
		}
	}
	if recovered > 0 {
		logger.InfoCF("agent", "Recovered unanswered sessions", map[string]any{
			"count": recovered,
		})
	}
	return recovered
}

func (al *AgentLoop) recoverUnansweredSession(
	ctx context.Context,
	agent *AgentInstance,
	sessionKey string,
) error {
	scope := sessionScopeForRecovery(agent.Sessions, sessionKey)
	inbound, ok := inboundContextFromSessionScope(scope)
	if !ok {
		return fmt.Errorf("session %q has no recoverable channel/chat scope", sessionKey)
	}

	placeholder := &turnState{
		turnID: "pending-recovery-" + sessionKey + "-" + fmt.Sprintf("%d", al.turnSeq.Add(1)),
		phase:  TurnPhaseSetup,
	}
	if _, loaded := al.activeTurnStates.LoadOrStore(sessionKey, placeholder); loaded {
		return nil
	}
	defer al.clearRecoveryPlaceholder(sessionKey, placeholder)

	if tool, ok := agent.Tools.Get("message"); ok {
		if resetter, ok := tool.(interface{ ResetSentInRound(sessionKey string) }); ok {
			resetter.ResetSentInRound(sessionKey)
		}
	}

	_, err := al.runAgentLoop(ctx, agent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:     sessionKey,
			InboundContext: &inbound,
			SessionScope:   session.CloneScope(scope),
			// Leave UserMessage empty. The unanswered user message is already
			// durable session history; adding it again would duplicate history.
		},
		DefaultResponse:         defaultResponse,
		EnableSummary:           true,
		SendResponse:            true,
		AllowInterimPicoPublish: true,
	})
	return err
}

func (al *AgentLoop) clearRecoveryPlaceholder(sessionKey string, placeholder *turnState) {
	if current, ok := al.activeTurnStates.Load(sessionKey); ok && current == placeholder {
		al.activeTurnStates.Delete(sessionKey)
	}
}

func (al *AgentLoop) sessionsWithUnackedInbound(ctx context.Context) map[string]struct{} {
	blocked := make(map[string]struct{})
	if al == nil || al.bus == nil {
		return blocked
	}
	msgs, err := al.bus.PendingInboundSpool(ctx)
	if err != nil {
		logger.WarnCF("agent", "Failed to inspect pending inbound spool before recovery",
			map[string]any{"error": err.Error()})
		return blocked
	}
	for _, msg := range msgs {
		if sessionKey := al.sessionKeyForInboundRecoveryBlock(msg); sessionKey != "" {
			blocked[sessionKey] = struct{}{}
		}
	}
	return blocked
}

func (al *AgentLoop) sessionKeyForInboundRecoveryBlock(msg bus.InboundMessage) string {
	if strings.TrimSpace(msg.SessionKey) != "" {
		return strings.TrimSpace(msg.SessionKey)
	}
	route, _, err := al.resolveMessageRoute(msg)
	if err != nil {
		return ""
	}
	allocation := al.allocateRouteSession(route, msg)
	return al.resolveEffectiveSessionKey(allocation.SessionKey, msg.SessionKey)
}

func sessionNeedsUnansweredRecovery(history []providers.Message) bool {
	for i := len(history) - 1; i >= 0; i-- {
		role := strings.TrimSpace(history[i].Role)
		switch role {
		case "":
			continue
		case "user":
			if isPassiveObservedHistoryMessage(history[i]) {
				return false
			}
			return strings.TrimSpace(history[i].Content) != "" || len(history[i].Media) > 0
		case "assistant", "tool":
			return false
		default:
			return false
		}
	}
	return false
}

func isPassiveObservedHistoryMessage(msg providers.Message) bool {
	content := strings.TrimSpace(msg.Content)
	return strings.HasPrefix(content, "[observed group message ") &&
		strings.Contains(content, " no reply requested;")
}

func sessionScopeForRecovery(store session.SessionStore, sessionKey string) *session.SessionScope {
	metaStore, ok := store.(session.MetadataAwareSessionStore)
	if !ok {
		return nil
	}
	return metaStore.GetSessionScope(sessionKey)
}

func inboundContextFromSessionScope(scope *session.SessionScope) (bus.InboundContext, bool) {
	if scope == nil {
		return bus.InboundContext{}, false
	}
	channel := strings.TrimSpace(scope.Channel)
	if channel == "" || len(scope.Values) == 0 {
		return bus.InboundContext{}, false
	}

	chatValue := strings.TrimSpace(scope.Values["chat"])
	if chatValue == "" {
		return bus.InboundContext{}, false
	}
	chatType, chatID, ok := strings.Cut(chatValue, ":")
	if !ok {
		return bus.InboundContext{}, false
	}
	chatType = strings.TrimSpace(chatType)
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return bus.InboundContext{}, false
	}

	topicID := ""
	if idx := strings.LastIndex(chatID, "/"); idx >= 0 {
		topicID = strings.TrimSpace(chatID[idx+1:])
		chatID = strings.TrimSpace(chatID[:idx])
	}
	if topicValue := strings.TrimSpace(scope.Values["topic"]); topicValue != "" {
		if _, topic, topicOK := strings.Cut(topicValue, ":"); topicOK {
			topicID = strings.TrimSpace(topic)
		}
	}

	return bus.NormalizeInboundMessage(bus.InboundMessage{Context: bus.InboundContext{
		Channel:  channel,
		Account:  strings.TrimSpace(scope.Account),
		ChatID:   chatID,
		ChatType: chatType,
		TopicID:  topicID,
		SenderID: "recovery",
	}}).Context, true
}

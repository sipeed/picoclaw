package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/session"
)

func buildResetSessionKey(agentID, routeSessionKey string) string {
	alias := fmt.Sprintf(
		"agent:%s:reset:%s:%d",
		strings.ToLower(strings.TrimSpace(agentID)),
		strings.ToLower(strings.TrimSpace(routeSessionKey)),
		time.Now().UnixNano(),
	)
	return session.BuildOpaqueSessionKey(alias)
}

func (al *AgentLoop) getSessionOverride(routeSessionKey string) string {
	if al == nil || al.state == nil {
		return ""
	}
	return al.state.GetSessionOverride(routeSessionKey)
}

func (al *AgentLoop) setSessionOverride(routeSessionKey, sessionKey string) error {
	if al == nil || al.state == nil {
		return fmt.Errorf("state manager not initialized")
	}
	return al.state.SetSessionOverride(routeSessionKey, sessionKey)
}

func (al *AgentLoop) clearSessionOverride(routeSessionKey string) error {
	if al == nil || al.state == nil {
		return fmt.Errorf("state manager not initialized")
	}
	return al.state.ClearSessionOverride(routeSessionKey)
}

func (al *AgentLoop) resolveEffectiveSessionKey(routeSessionKey, msgSessionKey string) string {
	if isExplicitSessionKey(msgSessionKey) {
		return msgSessionKey
	}
	if override := al.getSessionOverride(routeSessionKey); override != "" {
		return override
	}
	return routeSessionKey
}

func sessionAliasCandidates(routeSessionKey, effectiveSessionKey string, routeAliases []string, msgSessionKey string) []string {
	if isExplicitSessionKey(msgSessionKey) {
		return []string{msgSessionKey}
	}
	if strings.TrimSpace(routeSessionKey) == strings.TrimSpace(effectiveSessionKey) {
		return routeAliases
	}
	return nil
}

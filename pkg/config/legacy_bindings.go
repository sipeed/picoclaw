package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const legacyDefaultAccountID = "default"

type legacyBindingsEnvelope struct {
	Bindings json.RawMessage `json:"bindings"`
}

type legacyAgentBinding struct {
	AgentID string             `json:"agent_id"`
	Match   legacyBindingMatch `json:"match"`
}

type legacyBindingMatch struct {
	Channel   string           `json:"channel"`
	AccountID string           `json:"account_id,omitempty"`
	Peer      *legacyPeerMatch `json:"peer,omitempty"`
	GuildID   string           `json:"guild_id,omitempty"`
	TeamID    string           `json:"team_id,omitempty"`
}

type legacyPeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func applyLegacyBindingsMigration(data []byte, cfg *Config) {
	if cfg == nil {
		return
	}

	bindings, found, err := decodeLegacyBindings(data)
	if err != nil {
		logger.WarnF(
			"legacy bindings config detected but could not be decoded",
			map[string]any{"error": err},
		)
		return
	}
	if !found {
		return
	}

	if cfg.Agents.Dispatch != nil && len(cfg.Agents.Dispatch.Rules) > 0 {
		logger.WarnF(
			"legacy bindings config is deprecated and ignored because agents.dispatch.rules is configured",
			map[string]any{"bindings": len(bindings), "dispatch_rules": len(cfg.Agents.Dispatch.Rules)},
		)
		return
	}

	rules, dropped := migrateLegacyBindings(bindings, cfg.Session.IdentityLinks)
	if len(rules) == 0 {
		logger.WarnF(
			"legacy bindings config is deprecated and could not be migrated",
			map[string]any{"bindings": len(bindings), "dropped_bindings": dropped},
		)
		return
	}

	if cfg.Agents.Dispatch == nil {
		cfg.Agents.Dispatch = &DispatchConfig{}
	}
	cfg.Agents.Dispatch.Rules = rules

	fields := map[string]any{
		"bindings":       len(bindings),
		"dispatch_rules": len(rules),
	}
	if dropped > 0 {
		fields["dropped_bindings"] = dropped
	}
	logger.WarnF("legacy bindings config is deprecated; migrated to agents.dispatch.rules in memory", fields)
}

func decodeLegacyBindings(data []byte) ([]legacyAgentBinding, bool, error) {
	var envelope legacyBindingsEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, false, err
	}
	if len(envelope.Bindings) == 0 {
		return nil, false, nil
	}

	var bindings []legacyAgentBinding
	if err := json.Unmarshal(envelope.Bindings, &bindings); err != nil {
		return nil, true, err
	}
	return bindings, true, nil
}

func migrateLegacyBindings(bindings []legacyAgentBinding, identityLinks map[string][]string) ([]DispatchRule, int) {
	if len(bindings) == 0 {
		return nil, 0
	}

	type prioritizedRule struct {
		rule  DispatchRule
		index int
		kind  int
	}

	prioritized := make([]prioritizedRule, 0, len(bindings))
	dropped := 0
	for i, binding := range bindings {
		rule, kind, ok := migrateLegacyBinding(binding, i, identityLinks)
		if !ok {
			dropped++
			continue
		}
		prioritized = append(prioritized, prioritizedRule{rule: rule, index: i, kind: kind})
	}
	if len(prioritized) == 0 {
		return nil, dropped
	}

	rules := make([]DispatchRule, 0, len(prioritized))
	for kind := 0; kind <= 4; kind++ {
		for _, item := range prioritized {
			if item.kind == kind {
				rules = append(rules, item.rule)
			}
		}
	}
	return rules, dropped
}

func migrateLegacyBinding(
	binding legacyAgentBinding,
	index int,
	identityLinks map[string][]string,
) (DispatchRule, int, bool) {
	channel := strings.ToLower(strings.TrimSpace(binding.Match.Channel))
	agentID := strings.TrimSpace(binding.AgentID)
	if channel == "" || agentID == "" {
		return DispatchRule{}, 0, false
	}

	rule := DispatchRule{
		Name:  fmt.Sprintf("legacy-binding-%d", index+1),
		Agent: agentID,
		When: DispatchSelector{
			Channel: channel,
		},
	}

	switch normalizeLegacyAccountSelector(binding.Match.AccountID) {
	case "":
	case "*":
	default:
		rule.When.Account = normalizeLegacyAccountSelector(binding.Match.AccountID)
	}

	if peer := binding.Match.Peer; peer != nil {
		peerKind := strings.ToLower(strings.TrimSpace(peer.Kind))
		peerID := strings.TrimSpace(peer.ID)
		if peerID == "" {
			return DispatchRule{}, 0, false
		}
		switch peerKind {
		case "direct":
			rule.When.Sender = canonicalLegacyBindingSenderID(channel, peerID, identityLinks)
			return rule, 0, true
		case "group", "channel":
			rule.When.Chat = peerKind + ":" + peerID
			return rule, 0, true
		case "topic":
			rule.When.Topic = "topic:" + peerID
			return rule, 0, true
		default:
			return DispatchRule{}, 0, false
		}
	}

	if guildID := strings.TrimSpace(binding.Match.GuildID); guildID != "" {
		rule.When.Space = "guild:" + guildID
		return rule, 1, true
	}

	if teamID := strings.TrimSpace(binding.Match.TeamID); teamID != "" {
		rule.When.Space = "team:" + teamID
		return rule, 2, true
	}

	accountSelector := normalizeLegacyAccountSelector(binding.Match.AccountID)
	if accountSelector == "*" {
		rule.When.Account = ""
		return rule, 4, true
	}

	rule.When.Account = accountSelector
	return rule, 3, true
}

func normalizeLegacyAccountSelector(accountID string) string {
	accountID = strings.TrimSpace(accountID)
	switch accountID {
	case "":
		return legacyDefaultAccountID
	case "*":
		return "*"
	default:
		return strings.ToLower(accountID)
	}
}

func canonicalLegacyBindingSenderID(channel, peerID string, identityLinks map[string][]string) string {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return ""
	}

	if linked := resolveLegacyBindingLinkedID(identityLinks, channel, peerID); linked != "" {
		return strings.ToLower(linked)
	}

	return strings.ToLower(peerID)
}

func resolveLegacyBindingLinkedID(identityLinks map[string][]string, channel, peerID string) string {
	if len(identityLinks) == 0 {
		return ""
	}
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return ""
	}

	candidates := make(map[string]struct{})
	rawCandidate := strings.ToLower(peerID)
	if rawCandidate != "" {
		candidates[rawCandidate] = struct{}{}
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel != "" {
		candidates[channel+":"+rawCandidate] = struct{}{}
	}
	if idx := strings.Index(rawCandidate, ":"); idx > 0 && idx < len(rawCandidate)-1 {
		candidates[rawCandidate[idx+1:]] = struct{}{}
	}

	for canonical, ids := range identityLinks {
		canonical = strings.TrimSpace(canonical)
		if canonical == "" {
			continue
		}
		for _, id := range ids {
			normalized := strings.ToLower(strings.TrimSpace(id))
			if normalized == "" {
				continue
			}
			if _, ok := candidates[normalized]; ok {
				return canonical
			}
		}
	}

	return ""
}

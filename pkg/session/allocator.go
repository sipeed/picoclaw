package session

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// Allocation contains the concrete session keys selected for a routed turn.
// The current implementation intentionally preserves the legacy session-key
// layout while moving key construction out of the router.
type Allocation struct {
	Scope          SessionScope
	SessionKey     string
	SessionAliases []string
	MainSessionKey string
	MainAliases    []string
}

// AllocationInput contains the routing result and peer context needed to
// derive the session keys for a turn.
type AllocationInput struct {
	AgentID       string
	Context       bus.InboundContext
	SessionPolicy routing.SessionPolicy
}

// AllocateRouteSession maps a route decision onto a structured scope and the
// current opaque session-key format.
func AllocateRouteSession(input AllocationInput) Allocation {
	scope := buildSessionScope(input)
	legacySessionAliases := buildLegacySessionAliases(input)
	legacyMainSessionKey := strings.ToLower(BuildLegacyMainAlias(input.AgentID))
	return Allocation{
		Scope:          scope,
		SessionKey:     BuildSessionKey(scope),
		SessionAliases: legacySessionAliases,
		MainSessionKey: BuildOpaqueSessionKey(legacyMainSessionKey),
		MainAliases:    []string{legacyMainSessionKey},
	}
}

func buildSessionScope(input AllocationInput) SessionScope {
	inbound := input.Context
	includeTopicInChatDimension := shouldPreserveTelegramForumIsolation(input)
	scope := SessionScope{
		Version: ScopeVersionV1,
		AgentID: routing.NormalizeAgentID(input.AgentID),
		Channel: strings.ToLower(strings.TrimSpace(inbound.Channel)),
		Account: routing.NormalizeAccountID(inbound.Account),
	}
	if scope.Channel == "" {
		scope.Channel = "unknown"
	}

	dimensions := make([]string, 0, len(input.SessionPolicy.Dimensions))
	values := make(map[string]string, len(input.SessionPolicy.Dimensions))

	for _, dimension := range input.SessionPolicy.Dimensions {
		switch dimension {
		case "space":
			if spaceID := strings.TrimSpace(inbound.SpaceID); spaceID != "" {
				spaceType := strings.ToLower(strings.TrimSpace(inbound.SpaceType))
				if spaceType == "" {
					spaceType = "space"
				}
				dimensions = append(dimensions, "space")
				values["space"] = fmt.Sprintf("%s:%s", spaceType, strings.ToLower(spaceID))
			}
		case "chat":
			chatID := strings.TrimSpace(inbound.ChatID)
			if chatID == "" {
				continue
			}
			if includeTopicInChatDimension {
				if topicID := strings.TrimSpace(inbound.TopicID); topicID != "" {
					chatID = chatID + "/" + topicID
				}
			}
			chatType := strings.ToLower(strings.TrimSpace(inbound.ChatType))
			if chatType == "" {
				chatType = "direct"
			}
			dimensions = append(dimensions, "chat")
			values["chat"] = fmt.Sprintf("%s:%s", chatType, strings.ToLower(chatID))
		case "topic":
			if topicID := strings.TrimSpace(inbound.TopicID); topicID != "" {
				dimensions = append(dimensions, "topic")
				values["topic"] = "topic:" + strings.ToLower(topicID)
			}
		case "sender":
			senderID := CanonicalSessionIdentityID(
				inbound.Channel,
				inbound.SenderID,
				input.SessionPolicy.IdentityLinks,
			)
			if senderID == "" {
				continue
			}
			dimensions = append(dimensions, "sender")
			values["sender"] = senderID
		}
	}

	if len(dimensions) > 0 {
		scope.Dimensions = dimensions
		scope.Values = values
	}

	return scope
}

func buildLegacySessionAliases(input AllocationInput) []string {
	aliases := []string{strings.ToLower(BuildLegacyMainAlias(input.AgentID))}
	inbound := input.Context

	if strings.EqualFold(strings.TrimSpace(inbound.ChatType), "direct") {
		peerIDs := buildLegacyDirectPeerIDs(input)
		if len(peerIDs) == 0 {
			return uniqueAliases(aliases)
		}
		for _, peerID := range peerIDs {
			aliases = append(
				aliases,
				BuildLegacyDirectAliases(input.AgentID, inbound.Channel, inbound.Account, peerID)...,
			)
		}
		return uniqueAliases(aliases)
	}

	peerID := strings.TrimSpace(inbound.ChatID)
	if peerID == "" {
		return uniqueAliases(aliases)
	}
	if topicID := strings.TrimSpace(inbound.TopicID); topicID != "" {
		peerID = peerID + "/" + topicID
	}
	aliases = append(aliases, BuildLegacyPeerAlias(
		input.AgentID,
		inbound.Channel,
		strings.ToLower(strings.TrimSpace(inbound.ChatType)),
		peerID,
	))

	return uniqueAliases(aliases)
}

func shouldPreserveTelegramForumIsolation(input AllocationInput) bool {
	inbound := input.Context
	if !strings.EqualFold(strings.TrimSpace(inbound.Channel), "telegram") {
		return false
	}
	if strings.TrimSpace(inbound.TopicID) == "" {
		return false
	}
	for _, dimension := range input.SessionPolicy.Dimensions {
		if strings.EqualFold(strings.TrimSpace(dimension), "topic") {
			return false
		}
	}
	return true
}

func buildLegacyDirectPeerIDs(input AllocationInput) []string {
	inbound := input.Context
	peerIDs := make([]string, 0, 3)

	rawSenderID := strings.TrimSpace(inbound.SenderID)
	if rawSenderID != "" {
		peerIDs = append(peerIDs, strings.ToLower(rawSenderID))
	}

	canonicalSenderID := CanonicalSessionIdentityID(
		inbound.Channel,
		inbound.SenderID,
		input.SessionPolicy.IdentityLinks,
	)
	if canonicalSenderID != "" {
		peerIDs = append(peerIDs, canonicalSenderID)
	}

	chatID := strings.TrimSpace(inbound.ChatID)
	if chatID != "" {
		peerIDs = append(peerIDs, strings.ToLower(chatID))
	}

	return uniqueAliases(peerIDs)
}

func uniqueAliases(aliases []string) []string {
	if len(aliases) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(aliases))
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(strings.ToLower(alias))
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		normalized = append(normalized, alias)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

package routing

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// SessionPolicy describes how a routed message should be mapped to a session.
type SessionPolicy struct {
	Dimensions    []string
	IdentityLinks map[string][]string
}

// ResolvedRoute is the result of agent routing.
type ResolvedRoute struct {
	AgentID       string
	Channel       string
	AccountID     string
	SessionPolicy SessionPolicy
	MatchedBy     string
}

// RouteResolver determines which agent handles a message.
type RouteResolver struct {
	cfg *config.Config
}

// NewRouteResolver creates a new route resolver.
func NewRouteResolver(cfg *config.Config) *RouteResolver {
	return &RouteResolver{cfg: cfg}
}

// ResolveRoute determines which agent handles the message from a normalized
// inbound context and returns the session policy that should be used to
// allocate session state.
func (r *RouteResolver) ResolveRoute(inbound bus.InboundContext) ResolvedRoute {
	channel := strings.ToLower(strings.TrimSpace(inbound.Channel))
	accountID := NormalizeAccountID(inbound.Account)
	identityLinks := cloneIdentityLinks(r.cfg.Session.IdentityLinks)
	view := buildDispatchView(inbound, identityLinks)

	if rule := r.matchDispatchRule(view); rule != nil {
		return ResolvedRoute{
			AgentID:       r.pickAgentID(rule.Agent),
			Channel:       channel,
			AccountID:     accountID,
			SessionPolicy: r.sessionPolicy(rule),
			MatchedBy:     matchedByForRule(rule),
		}
	}

	return ResolvedRoute{
		AgentID:       r.pickAgentID(r.resolveDefaultAgentID()),
		Channel:       channel,
		AccountID:     accountID,
		SessionPolicy: r.sessionPolicy(nil),
		MatchedBy:     "default",
	}
}

func (r *RouteResolver) pickAgentID(agentID string) string {
	trimmed := strings.TrimSpace(agentID)
	if trimmed == "" {
		return NormalizeAgentID(r.resolveDefaultAgentID())
	}
	normalized := NormalizeAgentID(trimmed)
	agents := r.cfg.Agents.List
	if len(agents) == 0 {
		return normalized
	}
	for _, a := range agents {
		if NormalizeAgentID(a.ID) == normalized {
			return normalized
		}
	}
	return NormalizeAgentID(r.resolveDefaultAgentID())
}

func (r *RouteResolver) resolveDefaultAgentID() string {
	agents := r.cfg.Agents.List
	if len(agents) == 0 {
		return DefaultAgentID
	}
	for _, a := range agents {
		if a.Default {
			id := strings.TrimSpace(a.ID)
			if id != "" {
				return NormalizeAgentID(id)
			}
		}
	}
	if id := strings.TrimSpace(agents[0].ID); id != "" {
		return NormalizeAgentID(id)
	}
	return DefaultAgentID
}

func (r *RouteResolver) sessionPolicy(rule *config.DispatchRule) SessionPolicy {
	dimensions := r.cfg.Session.Dimensions
	if rule != nil && len(rule.SessionDimensions) > 0 {
		dimensions = rule.SessionDimensions
	}
	return SessionPolicy{
		Dimensions:    normalizeSessionDimensions(dimensions),
		IdentityLinks: cloneIdentityLinks(r.cfg.Session.IdentityLinks),
	}
}

func normalizeSessionDimensions(dimensions []string) []string {
	if len(dimensions) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(dimensions))
	seen := make(map[string]struct{}, len(dimensions))
	for _, dimension := range dimensions {
		dimension = strings.ToLower(strings.TrimSpace(dimension))
		switch dimension {
		case "space", "chat", "topic", "sender":
		default:
			continue
		}
		if _, ok := seen[dimension]; ok {
			continue
		}
		seen[dimension] = struct{}{}
		normalized = append(normalized, dimension)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func cloneIdentityLinks(src map[string][]string) map[string][]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(src))
	for canonical, ids := range src {
		dup := make([]string, len(ids))
		copy(dup, ids)
		cloned[canonical] = dup
	}
	return cloned
}

type dispatchView struct {
	Channel   string
	Account   string
	Space     string
	Chat      string
	Topic     string
	Sender    string
	Mentioned bool
}

func (r *RouteResolver) matchDispatchRule(view dispatchView) *config.DispatchRule {
	if r.cfg == nil || r.cfg.Agents.Dispatch == nil || len(r.cfg.Agents.Dispatch.Rules) == 0 {
		return nil
	}

	for i := range r.cfg.Agents.Dispatch.Rules {
		rule := &r.cfg.Agents.Dispatch.Rules[i]
		if !selectorHasAnyConstraint(rule.When) {
			continue
		}
		if ruleMatchesView(*rule, view) {
			return rule
		}
	}
	return nil
}

func ruleMatchesView(rule config.DispatchRule, view dispatchView) bool {
	when := normalizeDispatchSelector(rule.When)
	if when.Channel != "" && when.Channel != view.Channel {
		return false
	}
	if when.Account != "" && when.Account != view.Account {
		return false
	}
	if when.Space != "" && when.Space != view.Space {
		return false
	}
	if when.Chat != "" && when.Chat != view.Chat {
		return false
	}
	if when.Topic != "" && when.Topic != view.Topic {
		return false
	}
	if when.Sender != "" && when.Sender != view.Sender {
		return false
	}
	if when.Mentioned != nil && *when.Mentioned != view.Mentioned {
		return false
	}
	return true
}

func matchedByForRule(rule *config.DispatchRule) string {
	if rule == nil {
		return "default"
	}
	name := strings.TrimSpace(rule.Name)
	if name == "" {
		return "dispatch.rule"
	}
	return "dispatch.rule:" + strings.ToLower(name)
}

func buildDispatchView(inbound bus.InboundContext, identityLinks map[string][]string) dispatchView {
	view := dispatchView{
		Channel:   strings.ToLower(strings.TrimSpace(inbound.Channel)),
		Account:   NormalizeAccountID(inbound.Account),
		Mentioned: inbound.Mentioned,
	}

	if spaceID := strings.TrimSpace(inbound.SpaceID); spaceID != "" {
		spaceType := strings.ToLower(strings.TrimSpace(inbound.SpaceType))
		if spaceType == "" {
			spaceType = "space"
		}
		view.Space = fmt.Sprintf("%s:%s", spaceType, strings.ToLower(spaceID))
	}

	if chatID := strings.TrimSpace(inbound.ChatID); chatID != "" {
		chatType := strings.ToLower(strings.TrimSpace(inbound.ChatType))
		if chatType == "" {
			chatType = "direct"
		}
		view.Chat = fmt.Sprintf("%s:%s", chatType, strings.ToLower(chatID))
	}

	if topicID := strings.TrimSpace(inbound.TopicID); topicID != "" {
		view.Topic = "topic:" + strings.ToLower(topicID)
	}

	view.Sender = canonicalDispatchSenderID(inbound.Channel, inbound.SenderID, identityLinks)

	return view
}

func normalizeDispatchSelector(selector config.DispatchSelector) config.DispatchSelector {
	selector.Channel = strings.ToLower(strings.TrimSpace(selector.Channel))
	selector.Account = NormalizeAccountID(selector.Account)
	selector.Space = strings.ToLower(strings.TrimSpace(selector.Space))
	selector.Chat = strings.ToLower(strings.TrimSpace(selector.Chat))
	selector.Topic = strings.ToLower(strings.TrimSpace(selector.Topic))
	selector.Sender = strings.ToLower(strings.TrimSpace(selector.Sender))
	return selector
}

func selectorHasAnyConstraint(selector config.DispatchSelector) bool {
	return strings.TrimSpace(selector.Channel) != "" ||
		strings.TrimSpace(selector.Account) != "" ||
		strings.TrimSpace(selector.Space) != "" ||
		strings.TrimSpace(selector.Chat) != "" ||
		strings.TrimSpace(selector.Topic) != "" ||
		strings.TrimSpace(selector.Sender) != "" ||
		selector.Mentioned != nil
}

func canonicalDispatchSenderID(channel, rawID string, identityLinks map[string][]string) string {
	normalizedID := strings.TrimSpace(rawID)
	if normalizedID == "" {
		return ""
	}
	if linked := resolveLinkedDispatchID(identityLinks, channel, normalizedID); linked != "" {
		normalizedID = linked
	}
	return strings.ToLower(normalizedID)
}

func resolveLinkedDispatchID(identityLinks map[string][]string, channel, peerID string) string {
	if len(identityLinks) == 0 {
		return ""
	}
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return ""
	}

	candidates := make(map[string]bool)
	rawCandidate := strings.ToLower(peerID)
	if rawCandidate != "" {
		candidates[rawCandidate] = true
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel != "" {
		candidates[fmt.Sprintf("%s:%s", channel, rawCandidate)] = true
	}
	if idx := strings.Index(rawCandidate, ":"); idx > 0 && idx < len(rawCandidate)-1 {
		candidates[rawCandidate[idx+1:]] = true
	}

	for canonical, ids := range identityLinks {
		canonicalName := strings.TrimSpace(canonical)
		if canonicalName == "" {
			continue
		}
		for _, id := range ids {
			normalized := strings.ToLower(strings.TrimSpace(id))
			if normalized != "" && candidates[normalized] {
				return canonicalName
			}
		}
	}
	return ""
}

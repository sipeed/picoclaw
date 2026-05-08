package events

import "strings"

// Filter decides whether an event should pass through an EventChannel.
type Filter func(Event) bool

// ScopeFilter matches selected non-empty fields against Event.Scope.
type ScopeFilter struct {
	AgentID    string
	SessionKey string
	TurnID     string
	Channel    string
	ChatID     string
	MessageID  string
}

// MatchKind matches events whose kind is in kinds. Empty kinds match all events.
func MatchKind(kinds ...Kind) Filter {
	if len(kinds) == 0 {
		return matchAll
	}

	allowed := make(map[Kind]struct{}, len(kinds))
	for _, kind := range kinds {
		allowed[kind] = struct{}{}
	}

	return func(evt Event) bool {
		_, ok := allowed[evt.Kind]
		return ok
	}
}

// MatchKindPrefix matches events whose kind starts with prefix.
func MatchKindPrefix(prefix string) Filter {
	if prefix == "" {
		return matchAll
	}
	return func(evt Event) bool {
		return strings.HasPrefix(evt.Kind.String(), prefix)
	}
}

// MatchSource matches events emitted by component and, optionally, one of names.
func MatchSource(component string, names ...string) Filter {
	if component == "" && len(names) == 0 {
		return matchAll
	}

	allowedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		allowedNames[name] = struct{}{}
	}

	return func(evt Event) bool {
		if component != "" && evt.Source.Component != component {
			return false
		}
		if len(allowedNames) == 0 {
			return true
		}
		_, ok := allowedNames[evt.Source.Name]
		return ok
	}
}

// MatchScope matches events whose Scope contains all non-empty filter fields.
func MatchScope(scope ScopeFilter) Filter {
	if scope == (ScopeFilter{}) {
		return matchAll
	}

	return func(evt Event) bool {
		return matchesString(scope.AgentID, evt.Scope.AgentID) &&
			matchesString(scope.SessionKey, evt.Scope.SessionKey) &&
			matchesString(scope.TurnID, evt.Scope.TurnID) &&
			matchesString(scope.Channel, evt.Scope.Channel) &&
			matchesString(scope.ChatID, evt.Scope.ChatID) &&
			matchesString(scope.MessageID, evt.Scope.MessageID)
	}
}

// And combines filters and short-circuits on the first non-match.
func And(filters ...Filter) Filter {
	if len(filters) == 0 {
		return matchAll
	}

	return func(evt Event) bool {
		for _, filter := range filters {
			if filter != nil && !filter(evt) {
				return false
			}
		}
		return true
	}
}

// Or combines filters and short-circuits on the first match.
func Or(filters ...Filter) Filter {
	if len(filters) == 0 {
		return matchAll
	}

	return func(evt Event) bool {
		for _, filter := range filters {
			if filter == nil || filter(evt) {
				return true
			}
		}
		return false
	}
}

func matchAll(Event) bool {
	return true
}

func matchesString(want, got string) bool {
	return want == "" || want == got
}

func matchesFilters(filters []Filter, evt Event) bool {
	for _, filter := range filters {
		if filter != nil && !filter(evt) {
			return false
		}
	}
	return true
}

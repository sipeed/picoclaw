package channels

// AckReactionScope defines when to send acknowledgment reactions
type AckReactionScope string

const (
	// AckReactionScopeAll enables ack reactions for all messages
	AckReactionScopeAll AckReactionScope = "all"
	// AckReactionScopeDirect enables ack reactions only for direct messages
	AckReactionScopeDirect AckReactionScope = "direct"
	// AckReactionScopeGroupAll enables ack reactions for all group messages
	AckReactionScopeGroupAll AckReactionScope = "group-all"
	// AckReactionScopeGroupMentions enables ack reactions only when mentioned in groups
	AckReactionScopeGroupMentions AckReactionScope = "group-mentions"
	// AckReactionScopeOff disables ack reactions
	AckReactionScopeOff AckReactionScope = "off"
	// AckReactionScopeNone disables ack reactions (alias)
	AckReactionScopeNone AckReactionScope = "none"
)

// AckReactionParams contains parameters for determining whether to send an ack reaction
type AckReactionParams struct {
	// Scope is the configured ack reaction scope
	Scope AckReactionScope
	// IsDirect indicates if the message is a direct/private message
	IsDirect bool
	// IsGroup indicates if the message is from a group
	IsGroup bool
	// IsMentionableGroup indicates if the group supports mentions
	IsMentionableGroup bool
	// RequireMention indicates if the group requires mentioning the bot to respond
	RequireMention bool
	// CanDetectMention indicates if the platform can detect mentions
	CanDetectMention bool
	// WasMentioned indicates if the bot was mentioned in the message
	WasMentioned bool
	// ShouldBypassMention indicates if mention requirements should be bypassed
	ShouldBypassMention bool
}

// ShouldAckReaction determines whether an ack reaction should be sent based on parameters
// Reference: openclaw implementation
func ShouldAckReaction(params AckReactionParams) bool {
	// Default to group-mentions if not specified
	scope := params.Scope
	if scope == "" {
		scope = AckReactionScopeGroupMentions
	}

	// Disabled cases
	if scope == AckReactionScopeOff || scope == AckReactionScopeNone {
		return false
	}

	// All messages
	if scope == AckReactionScopeAll {
		return true
	}

	// Direct messages only
	if scope == AckReactionScopeDirect {
		return params.IsDirect
	}

	// All group messages
	if scope == AckReactionScopeGroupAll {
		return params.IsGroup
	}

	// Group mentions only
	if scope == AckReactionScopeGroupMentions {
		// Not a group message, don't ack
		if !params.IsGroup {
			return false
		}
		// Not a mentionable group, don't ack
		if !params.IsMentionableGroup {
			return false
		}
		// No mention required, don't ack (avoid over-acknowledging)
		if !params.RequireMention {
			return false
		}
		// Can't detect mentions, don't ack
		if !params.CanDetectMention {
			return false
		}
		// Mentioned or bypass required, ack
		return params.WasMentioned || params.ShouldBypassMention
	}

	return false
}

// AckReactionManager manages the lifecycle of acknowledgment reactions
type AckReactionManager struct {
	// ReactionValue is the current reaction value (emoji)
	ReactionValue string
	// Added indicates if the ack reaction has been added
	Added bool
}

// NewAckReactionManager creates a new ack reaction manager
func NewAckReactionManager(reaction string) *AckReactionManager {
	return &AckReactionManager{
		ReactionValue: reaction,
		Added:         false,
	}
}

// MarkAdded marks the ack reaction as added
func (m *AckReactionManager) MarkAdded() {
	m.Added = true
}

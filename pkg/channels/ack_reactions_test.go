package channels

import "testing"

func TestShouldAckReaction(t *testing.T) {
	tests := []struct {
		name   string
		params AckReactionParams
		want   bool
	}{
		// Scope: off/none (disabled)
		{
			name:   "scope off disables all",
			params: AckReactionParams{Scope: AckReactionScopeOff, IsDirect: true, IsGroup: true},
			want:   false,
		},
		{
			name:   "scope none alias disables all",
			params: AckReactionParams{Scope: AckReactionScopeNone, IsDirect: true, IsGroup: true},
			want:   false,
		},
		// Scope: all
		{
			name:   "scope all enables for direct messages",
			params: AckReactionParams{Scope: AckReactionScopeAll, IsDirect: true, IsGroup: false},
			want:   true,
		},
		{
			name:   "scope all enables for group messages",
			params: AckReactionParams{Scope: AckReactionScopeAll, IsDirect: false, IsGroup: true},
			want:   true,
		},
		// Scope: direct
		{
			name:   "scope direct enables for direct messages",
			params: AckReactionParams{Scope: AckReactionScopeDirect, IsDirect: true, IsGroup: false},
			want:   true,
		},
		{
			name:   "scope direct disables for group messages",
			params: AckReactionParams{Scope: AckReactionScopeDirect, IsDirect: false, IsGroup: true},
			want:   false,
		},
		// Scope: group-all
		{
			name:   "scope group-all enables for group messages",
			params: AckReactionParams{Scope: AckReactionScopeGroupAll, IsDirect: false, IsGroup: true},
			want:   true,
		},
		{
			name:   "scope group-all disables for direct messages",
			params: AckReactionParams{Scope: AckReactionScopeGroupAll, IsDirect: true, IsGroup: false},
			want:   false,
		},
		// Scope: group-mentions (mentionable group, require mention, can detect)
		{
			name: "scope group-mentions with all conditions met",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: true,
		},
		{
			name: "scope group-mentions bypass mention requirement",
			params: AckReactionParams{
				Scope:               AckReactionScopeGroupMentions,
				IsDirect:            false,
				IsGroup:             true,
				IsMentionableGroup:  true,
				RequireMention:      true,
				CanDetectMention:    true,
				ShouldBypassMention: true,
			},
			want: true,
		},
		{
			name: "scope group-mentions not in group",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           true,
				IsGroup:            false,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: false,
		},
		{
			name: "scope group-mentions not mentionable group",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: false,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: false,
		},
		{
			name: "scope group-mentions no mention required",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: true,
				RequireMention:     false,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: false,
		},
		{
			name: "scope group-mentions cannot detect mention",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   false,
				WasMentioned:       true,
			},
			want: false,
		},
		{
			name: "scope group-mentions not mentioned",
			params: AckReactionParams{
				Scope:              AckReactionScopeGroupMentions,
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       false,
			},
			want: false,
		},
		// Default scope (empty) - should default to group-mentions
		{
			name: "empty scope defaults to group-mentions with all conditions met",
			params: AckReactionParams{
				Scope:              "",
				IsDirect:           false,
				IsGroup:            true,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: true,
		},
		{
			name: "empty scope defaults to group-mentions not in group",
			params: AckReactionParams{
				Scope:              "",
				IsDirect:           true,
				IsGroup:            false,
				IsMentionableGroup: true,
				RequireMention:     true,
				CanDetectMention:   true,
				WasMentioned:       true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldAckReaction(tt.params); got != tt.want {
				t.Errorf("ShouldAckReaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAckReactionManager(t *testing.T) {
	t.Run("mark added", func(t *testing.T) {
		m := NewAckReactionManager("OK")
		if m.Added {
			t.Error("expected Added to be false initially")
		}
		m.MarkAdded()
		if !m.Added {
			t.Error("expected Added to be true after MarkAdded")
		}
	})
}

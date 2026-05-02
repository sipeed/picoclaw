// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"testing"
)

// TestUserPromptMessage_AttachesSanitizedSenderName verifies that
// userPromptMessage stores a sanitized sender attribution in
// providers.Message.Name when given a real human sender. This is the
// core wiring change for issue #2702.
func TestUserPromptMessage_AttachesSanitizedSenderName(t *testing.T) {
	tests := []struct {
		name     string
		senderID string
		wantName string
	}{
		{"slack-style id", "U07AB12C3DEF", "U07AB12C3DEF"},
		{"discord-style id sanitized", "alice#1234", "alice_1234"},
		{"telegram numeric", "141455495", "141455495"},
		{"unicode display becomes empty", "李华", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := userPromptMessage("hello", nil, tt.senderID)
			if msg.Role != "user" {
				t.Errorf("Role = %q, want user", msg.Role)
			}
			if msg.Content != "hello" {
				t.Errorf("Content = %q, want hello", msg.Content)
			}
			if msg.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", msg.Name, tt.wantName)
			}
		})
	}
}

// TestUserPromptMessage_SkipsAttributionForSystemSenders verifies that
// synthetic trigger sources (cron, heartbeat, async callbacks) do not
// produce per-message attribution. These are not distinct human users,
// so leaking them into Name would create spurious "user identities" in
// the OpenAI wire format and a confusing `[cron] [System: ...]` double
// prefix on Anthropic-style adapters.
func TestUserPromptMessage_SkipsAttributionForSystemSenders(t *testing.T) {
	systemIDs := []string{"", "cron", "heartbeat", "system", "async:read_file"}
	for _, id := range systemIDs {
		t.Run(id, func(t *testing.T) {
			msg := userPromptMessage("triggered work", nil, id)
			if msg.Name != "" {
				t.Errorf("Name = %q, want empty for synthetic sender %q", msg.Name, id)
			}
		})
	}
}

// TestUserPromptMessage_NoNameMeansNoAttribution covers the empty-sender
// path explicitly so future refactors don't accidentally start tagging
// anonymous direct-channel turns.
func TestUserPromptMessage_NoNameMeansNoAttribution(t *testing.T) {
	msg := userPromptMessage("anonymous message", nil, "")
	if msg.Name != "" {
		t.Errorf("Name = %q, want empty for anonymous sender", msg.Name)
	}
}

// TestUserPromptMessage_PreservesMedia is a regression guard: the
// existing media-passthrough behavior must continue to work after the
// signature change.
func TestUserPromptMessage_PreservesMedia(t *testing.T) {
	media := []string{"data:image/png;base64,abc"}
	msg := userPromptMessage("look", media, "alice")
	if len(msg.Media) != 1 || msg.Media[0] != media[0] {
		t.Errorf("Media = %v, want %v", msg.Media, media)
	}
	if msg.Name != "alice" {
		t.Errorf("Name = %q, want alice", msg.Name)
	}
}

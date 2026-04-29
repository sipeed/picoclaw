// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package messageutil

import (
	"strings"
	"testing"
)

func TestSanitizeMessageName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"plain ascii", "alice", "alice"},
		{"discord-style id", "alice#1234", "alice_1234"},
		{"telegram numeric id", "141455495", "141455495"},
		{"slack-style id", "U07AB12C3DEF", "U07AB12C3DEF"},
		{"already valid mixed case", "Alice_Bob-99", "Alice_Bob-99"},
		{"strips leading/trailing underscores", "@alice@", "alice"},
		{"collapses runs", "a@@@b", "a_b"},
		{"non-ascii becomes underscores", "李华", ""},
		{"non-ascii mixed", "alice 李", "alice"},
		{"all special chars collapses to empty", "@@@!!!", ""},
		{"truncates to 64", strings.Repeat("a", 100), strings.Repeat("a", 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeMessageName(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeMessageName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeMessageName_OutputAlwaysWireSafe(t *testing.T) {
	inputs := []string{
		"alice#1234", "U07AB12C3DEF", "141455495",
		"@@@", "李华", "alice 李", "Alice_Bob-99",
		strings.Repeat("a", 200),
	}
	for _, in := range inputs {
		got := SanitizeMessageName(in)
		if got == "" {
			continue
		}
		if len(got) > maxMessageNameLen {
			t.Errorf("SanitizeMessageName(%q) length %d exceeds max %d", in, len(got), maxMessageNameLen)
		}
		for _, r := range got {
			ok := (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') ||
				r == '_' || r == '-'
			if !ok {
				t.Errorf("SanitizeMessageName(%q) = %q contains invalid rune %q", in, got, r)
			}
		}
	}
}

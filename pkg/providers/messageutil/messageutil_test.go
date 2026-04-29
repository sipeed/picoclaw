// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package messageutil

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
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

func TestIsSystemSenderID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"   ", true},
		{"cron", true},
		{"CRON", true},
		{"heartbeat", true},
		{"system", true},
		{"async:tool_call", true},
		{"ASYNC:Foo", true},
		{"alice", false},
		{"U07AB12C3DEF", false},
		{"141455495", false},
		{"alice#1234", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := IsSystemSenderID(tt.in); got != tt.want {
				t.Errorf("IsSystemSenderID(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestApplyUserNamePrefix(t *testing.T) {
	tests := []struct {
		name string
		msg  protocoltypes.Message
		want string
	}{
		{
			name: "user with name",
			msg:  protocoltypes.Message{Role: "user", Name: "alice", Content: "hello"},
			want: "[alice] hello",
		},
		{
			name: "user without name",
			msg:  protocoltypes.Message{Role: "user", Content: "hello"},
			want: "hello",
		},
		{
			name: "user with name but empty content",
			msg:  protocoltypes.Message{Role: "user", Name: "alice", Content: ""},
			want: "",
		},
		{
			name: "user tool result is not prefixed",
			msg:  protocoltypes.Message{Role: "user", Name: "alice", Content: `{"ok":true}`, ToolCallID: "call_1"},
			want: `{"ok":true}`,
		},
		{
			name: "assistant with name not prefixed",
			msg:  protocoltypes.Message{Role: "assistant", Name: "alice", Content: "I am the assistant"},
			want: "I am the assistant",
		},
		{
			name: "tool role with name not prefixed",
			msg:  protocoltypes.Message{Role: "tool", Name: "alice", Content: `{"x":1}`},
			want: `{"x":1}`,
		},
		{
			name: "system role with name not prefixed",
			msg:  protocoltypes.Message{Role: "system", Name: "alice", Content: "instructions"},
			want: "instructions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyUserNamePrefix(tt.msg)
			if got != tt.want {
				t.Errorf("ApplyUserNamePrefix(%+v) = %q, want %q", tt.msg, got, tt.want)
			}
		})
	}
}

func TestApplyUserNamePrefix_DoesNotMutateInput(t *testing.T) {
	msg := protocoltypes.Message{Role: "user", Name: "alice", Content: "hello"}
	original := msg
	_ = ApplyUserNamePrefix(msg)
	if !reflect.DeepEqual(msg, original) {
		t.Errorf("ApplyUserNamePrefix mutated the input message: got %+v, want %+v", msg, original)
	}
}

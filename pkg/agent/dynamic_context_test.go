package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// stripRuntimeContext removes the tail-placed <runtime_context> block from a
// user message, returning the user's own text. Tests that care about the user
// text (not the placement) use this so they stay agnostic to the configured
// dynamic context position.
func stripRuntimeContext(content string) string {
	start := strings.Index(content, runtimeContextOpenTag)
	if start < 0 {
		return content
	}
	end := strings.Index(content, runtimeContextCloseTag)
	if end < 0 {
		return content
	}
	end += len(runtimeContextCloseTag)
	return strings.TrimSpace(content[:start] + content[end:])
}

// runtimeContextBlock returns the contents of the tail-placed <runtime_context>
// block, or "" when the message carries none.
func runtimeContextBlock(content string) string {
	start := strings.Index(content, runtimeContextOpenTag)
	if start < 0 {
		return ""
	}
	end := strings.Index(content, runtimeContextCloseTag)
	if end < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(content[start+len(runtimeContextOpenTag) : end])
}

func TestBuildMessages_TailPlacementKeepsSystemPromptStatic(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Identity\nTest agent.",
	})

	cb := NewContextBuilder(tmpDir)
	msgs := cb.BuildMessages(nil, "", "hello", nil, "discord", "chat1", "u1", "Alice")

	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}

	sys := msgs[0].Content
	for _, marker := range []string{"## Current Time", "## Runtime", "## Current Session", "## Current Sender"} {
		if strings.Contains(sys, marker) {
			t.Fatalf("system prompt must not contain %q with tail placement:\n%s", marker, sys)
		}
	}

	user := msgs[len(msgs)-1]
	if user.Role != "user" {
		t.Fatalf("last message role = %q, want user", user.Role)
	}
	if stripRuntimeContext(user.Content) != "hello" {
		t.Fatalf("user text = %q, want %q", stripRuntimeContext(user.Content), "hello")
	}
	block := runtimeContextBlock(user.Content)
	for _, marker := range []string{"## Current Time", "## Runtime", "## Current Session", "## Current Sender"} {
		if !strings.Contains(block, marker) {
			t.Fatalf("tail runtime context missing %q:\n%s", marker, block)
		}
	}
}

// The whole point of tail placement: a clock tick must leave the system prompt
// and the history byte-identical so a prefix-matching backend keeps its KV cache.
func TestBuildMessages_TailPlacementSystemPromptStableAcrossClockTick(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Identity\nTest agent.",
	})

	cb := NewContextBuilder(tmpDir)
	first := cb.BuildMessages(nil, "prior summary", "hello", nil, "discord", "chat1", "u1", "Alice")
	second := cb.BuildMessages(nil, "prior summary", "hello", nil, "discord", "chat1", "u1", "Alice")

	if first[0].Content != second[0].Content {
		t.Fatalf("system prompt differs between builds:\n%q\n%q", first[0].Content, second[0].Content)
	}
	if !strings.Contains(first[0].Content, "CONTEXT_SUMMARY:") {
		t.Fatal("system prompt missing summary")
	}
}

func TestBuildMessages_SystemPlacementRestoresLegacyLayout(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Identity\nTest agent.",
	})

	cb := NewContextBuilder(tmpDir).WithDynamicContext(config.EffectiveDynamicContext{
		Time:     config.DynamicContextTimeMinute,
		Position: config.DynamicContextPositionSystem,
	})
	msgs := cb.BuildMessages(nil, "", "hello", nil, "discord", "chat1", "u1", "Alice")

	sys := msgs[0].Content
	if !strings.Contains(sys, "## Current Time") {
		t.Fatalf("system placement must keep the time in the system prompt:\n%s", sys)
	}
	user := msgs[len(msgs)-1]
	if user.Content != "hello" {
		t.Fatalf("user content = %q, want unwrapped %q", user.Content, "hello")
	}
}

// Ordering inside the block is by volatility, least volatile first, so that a
// clock tick invalidates as short a suffix as possible under system placement.
func TestBuildDynamicContext_OrdersByVolatility(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	block := cb.buildDynamicContext("discord", "chat1", "u1", "Alice")

	order := []string{"## Runtime", "## Current Session", "## Current Sender", "## Current Time"}
	prev := -1
	for _, marker := range order {
		idx := strings.Index(block, marker)
		if idx < 0 {
			t.Fatalf("dynamic context missing %q:\n%s", marker, block)
		}
		if idx < prev {
			t.Fatalf("dynamic context section %q out of volatility order:\n%s", marker, block)
		}
		prev = idx
	}
}

func TestFormatCurrentTime_Precision(t *testing.T) {
	now := time.Date(2026, 8, 3, 14, 37, 12, 0, time.UTC)

	tests := []struct {
		name      string
		precision config.DynamicContextTime
		want      string
	}{
		{"default is minute", "", "2026-08-03 14:37 (Monday)"},
		{"minute", config.DynamicContextTimeMinute, "2026-08-03 14:37 (Monday)"},
		{"hour truncates minutes", config.DynamicContextTimeHour, "2026-08-03 14:00 (Monday)"},
		{"off omits the time", config.DynamicContextTimeOff, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCurrentTime(now, tt.precision); got != tt.want {
				t.Fatalf("formatCurrentTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDynamicContext_TimeOffOmitsSection(t *testing.T) {
	cb := NewContextBuilder(t.TempDir()).WithDynamicContext(config.EffectiveDynamicContext{
		Time:     config.DynamicContextTimeOff,
		Position: config.DynamicContextPositionTail,
	})
	block := cb.buildDynamicContext("discord", "chat1", "", "")

	if strings.Contains(block, "## Current Time") {
		t.Fatalf("time off must omit the section:\n%s", block)
	}
	if !strings.Contains(block, "## Runtime") {
		t.Fatalf("time off must keep the runtime section:\n%s", block)
	}
}

// A turn with no user text still needs to carry the block somewhere.
func TestBuildMessages_TailPlacementWithoutCurrentMessage(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)

	cb := NewContextBuilder(tmpDir)
	msgs := cb.BuildMessagesFromPrompt(PromptBuildRequest{
		History: []providers.Message{{Role: "user", Content: "earlier"}},
		Channel: "pico",
		ChatID:  "chat-1",
	})

	last := msgs[len(msgs)-1]
	if last.Role != "user" {
		t.Fatalf("last message role = %q, want user", last.Role)
	}
	if !strings.Contains(runtimeContextBlock(last.Content), "## Runtime") {
		t.Fatalf("trailing message missing runtime context:\n%s", last.Content)
	}
	if stripRuntimeContext(last.Content) != "" {
		t.Fatalf("trailing message should carry no user text, got %q", stripRuntimeContext(last.Content))
	}
}

func TestSuppressedSystemPromptSkipsDynamicContext(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	msgs := cb.BuildMessagesFromPrompt(PromptBuildRequest{
		CurrentMessage:              "hello",
		Channel:                     "pico",
		ChatID:                      "chat-1",
		SuppressDefaultSystemPrompt: true,
	})

	for _, msg := range msgs {
		if strings.Contains(msg.Content, runtimeContextOpenTag) {
			t.Fatalf("suppressed system prompt must not emit runtime context:\n%s", msg.Content)
		}
	}
}

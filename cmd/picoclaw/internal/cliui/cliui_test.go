package cliui

import (
	"testing"

	flag "github.com/spf13/pflag"
)

func init() {
	// Disable ANSI colors in tests so output is predictable plain text.
	Init(true)
}

// ---------------------------------------------------------------------------
// showErrHint
// ---------------------------------------------------------------------------

func TestShowErrHint(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		// Cobra flag errors — should show hint
		{"unknown flag: --foo", true},
		{"unknown shorthand flag: 'f' in -f", true},
		{"flag needs an argument: --output", true},
		{"required flag(s) \"model\" not set", true},
		// Generic invalid-argument errors — should show hint
		{"invalid argument \"abc\" for --count", true},
		// required flag errors — should show hint
		{"required flag(s) \"model\" not set", true},
		// usage: in message — should show hint
		{"bad input\nusage: picoclaw ...", true},
		// Should NOT false-positive on broad words
		{"connection flagged by remote", false},
		{"feature flag not set", false},
		{"invalid API key provided", false},
		{"authentication required", false},
		// Unrelated messages — no hint
		{"something went wrong", false},
		{"network timeout", false},
	}

	for _, tc := range cases {
		got := showErrHint(tc.msg)
		if got != tc.want {
			t.Errorf("showErrHint(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// styleUsageTokens
// ---------------------------------------------------------------------------

func TestStyleUsageTokensContainsTokens(t *testing.T) {
	cases := []struct {
		input    string
		contains []string // substrings that must appear in plain output
	}{
		{
			"picoclaw agent <message>",
			[]string{"picoclaw agent", "<message>"},
		},
		{
			"picoclaw [command] [flags]",
			[]string{"picoclaw", "[command]", "[flags]"},
		},
		{
			"picoclaw",
			[]string{"picoclaw"},
		},
		{
			"cmd <arg1> [--flag]",
			[]string{"cmd", "<arg1>", "[--flag]"},
		},
	}

	for _, tc := range cases {
		out := styleUsageTokens(tc.input)
		for _, sub := range tc.contains {
			if !containsStripped(out, sub) {
				t.Errorf("styleUsageTokens(%q): output %q does not contain %q", tc.input, out, sub)
			}
		}
	}
}

// containsStripped checks whether plain contains sub after stripping ANSI escapes.
// Since Init(true) sets Ascii profile, lipgloss emits no escape codes in tests,
// so this is just a plain substring check.
func containsStripped(plain, sub string) bool {
	return len(plain) >= len(sub) && findSubstring(plain, sub)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// collectFlagRows
// ---------------------------------------------------------------------------

func TestCollectFlagRows_Empty(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rows := collectFlagRows(fs)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for empty FlagSet, got %d", len(rows))
	}
}

func TestCollectFlagRows_BasicFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("output", "", "output file path")
	fs.Bool("verbose", false, "enable verbose mode")
	fs.Int("count", 1, "number of items")

	rows := collectFlagRows(fs)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Rows must be sorted alphabetically by flag name.
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r[0])
	}
	if names[0] > names[1] || names[1] > names[2] {
		t.Errorf("rows not sorted: %v", names)
	}
}

func TestCollectFlagRows_Shorthand(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.StringP("model", "m", "", "model name")

	rows := collectFlagRows(fs)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	left := rows[0][0]
	if !findSubstring(left, "-m") || !findSubstring(left, "--model") {
		t.Errorf("expected shorthand and long form in %q", left)
	}
}

func TestCollectFlagRows_HiddenFlagsExcluded(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("visible", "", "this shows up")
	hidden := fs.String("hidden", "", "this should not show up")
	_ = hidden
	_ = fs.MarkHidden("hidden")

	rows := collectFlagRows(fs)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (hidden excluded), got %d", len(rows))
	}
	if !findSubstring(rows[0][0], "visible") {
		t.Errorf("expected visible flag in rows, got %q", rows[0][0])
	}
}

func TestCollectFlagRows_UsageInRightColumn(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("format", "json", "output format: json or text")

	rows := collectFlagRows(fs)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][1] != "output format: json or text" {
		t.Errorf("expected usage in right column, got %q", rows[0][1])
	}
}

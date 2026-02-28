package utils

import (
	"strings"
	"testing"
)

// --- StripThinkBlocks ---

func TestStripThinkBlocks_ClosedBlock(t *testing.T) {
	in := "<think>\nsecret reasoning\n</think>\n\nVisible content"
	got := StripThinkBlocks(in)
	if got != "Visible content" {
		t.Fatalf("StripThinkBlocks() = %q, want %q", got, "Visible content")
	}
}

func TestStripThinkBlocks_UnclosedBlock(t *testing.T) {
	in := "<think>reasoning that never ends\nmore reasoning"
	got := StripThinkBlocks(in)
	if got != "" {
		t.Fatalf("StripThinkBlocks() = %q, want empty", got)
	}
}

func TestStripThinkBlocks_MultipleBlocks(t *testing.T) {
	in := "<think>first</think>middle<think>second</think>end"
	got := StripThinkBlocks(in)
	if got != "middleend" {
		t.Fatalf("StripThinkBlocks() = %q, want %q", got, "middleend")
	}
}

func TestStripThinkBlocks_NoBlocks(t *testing.T) {
	in := "plain text without think blocks"
	got := StripThinkBlocks(in)
	if got != in {
		t.Fatalf("StripThinkBlocks() = %q, want %q", got, in)
	}
}

func TestStripThinkBlocks_CaseInsensitive(t *testing.T) {
	in := "<THINK>upper case</THINK>visible"
	got := StripThinkBlocks(in)
	if got != "visible" {
		t.Fatalf("StripThinkBlocks() = %q, want %q", got, "visible")
	}
}

func TestStripThinkBlocks_ClosedThenUnclosed(t *testing.T) {
	in := "<think>closed</think>middle<think>unclosed tail"
	got := StripThinkBlocks(in)
	if got != "middle" {
		t.Fatalf("StripThinkBlocks() = %q, want %q", got, "middle")
	}
}

// --- DetectRepetitionLoop ---

func TestDetectRepetitionLoop_HighRepetition(t *testing.T) {
	// Repeat a short phrase many times → should be detected
	phrase := "結構本格的なコード"
	repeated := strings.Repeat(phrase, 300)
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for highly repetitive text")
	}
}

func TestDetectRepetitionLoop_NormalText(t *testing.T) {
	// Normal varied text should not trigger
	normal := "The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs. " +
		"How vexingly quick daft zebras jump. " +
		"Sphinx of black quartz, judge my vow. " +
		"Two driven jocks help fax my big quiz. " +
		"The five boxing wizards jump quickly. " +
		"Jackdaws love my big sphinx of quartz. " +
		"Grumpy wizards make a toxic brew for the jovial queen."
	// Extend to be long enough
	long := strings.Repeat(normal+" ", 10)
	if DetectRepetitionLoop(long) {
		t.Fatal("DetectRepetitionLoop should return false for normal text")
	}
}

func TestDetectRepetitionLoop_ShortText(t *testing.T) {
	// Text shorter than N-gram size should never trigger
	if DetectRepetitionLoop("short") {
		t.Fatal("DetectRepetitionLoop should return false for short text")
	}
}

func TestDetectRepetitionLoop_EmptyString(t *testing.T) {
	if DetectRepetitionLoop("") {
		t.Fatal("DetectRepetitionLoop should return false for empty string")
	}
}

func TestDetectRepetitionLoop_SingleCharRepeat(t *testing.T) {
	// "aaaa..." repeated → only 1 unique N-gram → detected
	repeated := strings.Repeat("あ", 2500)
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for single-char repetition")
	}
}

func TestDetectRepetitionLoop_BelowSampleSize(t *testing.T) {
	// Repetitive but under sample size still detected
	phrase := "abcdefghij"
	repeated := strings.Repeat(phrase, 50) // 500 chars
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for repetitive text below sample size")
	}
}

// --- TailPad ---

func TestTailPad_FewerThanN(t *testing.T) {
	got := TailPad("a\nb", 5, 80)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("TailPad line count = %d, want 5", len(lines))
	}
	for i := 0; i < 3; i++ {
		if lines[i] != "\u2800" {
			t.Errorf("TailPad line %d = %q, want padding", i, lines[i])
		}
	}
	if lines[3] != "a" || lines[4] != "b" {
		t.Errorf("TailPad content = %q %q, want a b", lines[3], lines[4])
	}
}

func TestTailPad_ExactlyN(t *testing.T) {
	in := "a\nb\nc"
	got := TailPad(in, 3, 80)
	if got != in {
		t.Fatalf("TailPad exact = %q, want %q", got, in)
	}
}

func TestTailPad_MoreThanN(t *testing.T) {
	got := TailPad("a\nb\nc\nd\ne", 3, 80)
	if got != "c\nd\ne" {
		t.Fatalf("TailPad tail = %q, want %q", got, "c\nd\ne")
	}
}

func TestTailPad_Empty(t *testing.T) {
	got := TailPad("", 4, 80)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("TailPad empty line count = %d, want 4", len(lines))
	}
	for i, l := range lines {
		if i == len(lines)-1 {
			if l != "" {
				t.Errorf("TailPad empty last line = %q, want empty", l)
			}
		} else if l != "\u2800" {
			t.Errorf("TailPad empty line %d = %q, want padding", i, l)
		}
	}
}

func TestTailPad_LongLineWraps(t *testing.T) {
	// One 10-char line wraps into 2 visual lines at width 5.
	got := TailPad("abcdefghij", 4, 5)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("TailPad wrap line count = %d, want 4", len(lines))
	}
	// 2 padding + "abcde" + "fghij"
	if lines[2] != "abcde" || lines[3] != "fghij" {
		t.Errorf("TailPad wrap content = %v", lines)
	}
}

func TestTailPad_WrapPushesOldLines(t *testing.T) {
	// "short" (1 visual) + "abcdefghij" (2 visual at width 5) = 3 visual.
	// With n=2, only tail 2 visual lines remain.
	got := TailPad("short\nabcdefghij", 2, 5)
	if got != "abcde\nfghij" {
		t.Fatalf("TailPad wrap push = %q, want %q", got, "abcde\nfghij")
	}
}

// --- Truncate ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hi",
			maxLen: 10,
			want:   "hi",
		},
		{
			name:   "exact length unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated with ellipsis",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "maxLen equals 4 leaves 1 char plus ellipsis",
			input:  "abcdef",
			maxLen: 4,
			want:   "a...",
		},
		{
			name:   "maxLen 3 returns first 3 chars without ellipsis",
			input:  "abcdef",
			maxLen: 3,
			want:   "abc",
		},
		{
			name:   "maxLen 2 returns first 2 chars",
			input:  "abcdef",
			maxLen: 2,
			want:   "ab",
		},
		{
			name:   "maxLen 1 returns first char",
			input:  "abcdef",
			maxLen: 1,
			want:   "a",
		},
		{
			name:   "maxLen 0 returns empty",
			input:  "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "negative maxLen returns empty",
			input:  "hello",
			maxLen: -1,
			want:   "",
		},
		{
			name:   "empty string unchanged",
			input:  "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "empty string with zero maxLen",
			input:  "",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "unicode truncated correctly",
			input:  "\U0001f600\U0001f601\U0001f602\U0001f603\U0001f604",
			maxLen: 4,
			want:   "\U0001f600...",
		},
		{
			name:   "unicode short enough",
			input:  "\u00e9\u00e8",
			maxLen: 5,
			want:   "\u00e9\u00e8",
		},
		{
			name:   "mixed ascii and unicode",
			input:  "Go\U0001f680\U0001f525\U0001f4a5\U0001f30d",
			maxLen: 5,
			want:   "Go...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSanitizeMessageContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text unchanged", "Hello world", "Hello world"},
		{"strip ZWSP", "Hello\u200bworld", "Helloworld"},
		{"strip RTL override", "Hi\u202eevil", "Hievil"},
		{"strip BOM", "\uFEFFcontent", "content"},
		{"strip multiple", "a\u200c\u202ab\u202cc", "abc"},
		{"unicode letters preserved", "café \u65e5\u672c\u8a9e", "café \u65e5\u672c\u8a9e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeMessageContent(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeMessageContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

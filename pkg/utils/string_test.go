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

// --- Truncate ---

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Errorf("Truncate short = %q", got)
	}
	if got := Truncate("hello world!", 8); got != "hello..." {
		t.Errorf("Truncate long = %q", got)
	}
}

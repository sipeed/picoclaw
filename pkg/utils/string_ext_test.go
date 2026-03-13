package utils

import (
	"strings"
	"testing"
)

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

func TestDetectRepetitionLoop_HighRepetition(t *testing.T) {
	phrase := "結構本格的なコード" //nolint:gosmopolitan
	repeated := strings.Repeat(phrase, 300)
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for highly repetitive text")
	}
}

func TestDetectRepetitionLoop_NormalText(t *testing.T) {
	normal := "The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs. " +
		"How vexingly quick daft zebras jump. " +
		"Sphinx of black quartz, judge my vow. " +
		"Two driven jocks help fax my big quiz. " +
		"The five boxing wizards jump quickly. " +
		"Jackdaws love my big sphinx of quartz. " +
		"Grumpy wizards make a toxic brew for the jovial queen."

	long := strings.Repeat(normal+" ", 10)
	if DetectRepetitionLoop(long) {
		t.Fatal("DetectRepetitionLoop should return false for normal text")
	}
}

func TestDetectRepetitionLoop_ShortText(t *testing.T) {
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
	repeated := strings.Repeat("あ", 2500)
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for single-char repetition")
	}
}

func TestDetectRepetitionLoop_BelowSampleSize(t *testing.T) {
	phrase := "abcdefghij"
	repeated := strings.Repeat(phrase, 50)
	if !DetectRepetitionLoop(repeated) {
		t.Fatal("DetectRepetitionLoop should return true for repetitive text below sample size")
	}
}

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
	got := TailPad("abcdefghij", 4, 5)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("TailPad wrap line count = %d, want 4", len(lines))
	}

	if lines[2] != "abcde" || lines[3] != "fghij" {
		t.Errorf("TailPad wrap content = %v", lines)
	}
}

func TestTailPad_WrapPushesOldLines(t *testing.T) {
	got := TailPad("short\nabcdefghij", 2, 5)
	if got != "abcde\nfghij" {
		t.Fatalf("TailPad wrap push = %q, want %q", got, "abcde\nfghij")
	}
}

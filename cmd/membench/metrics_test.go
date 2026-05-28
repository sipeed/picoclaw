package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestSplitEvidenceIDs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"D1:3", []string{"D1:3"}},
		{"D8:6; D9:17", []string{"D8:6", "D9:17"}},
		{"D9:1 D4:4 D4:6", []string{"D9:1", "D4:4", "D4:6"}},
		{"D22:1 D22:2 D9:10 D9:11", []string{"D22:1", "D22:2", "D9:10", "D9:11"}},
		{"D21:18 D21:22 D11:15 D11:19", []string{"D21:18", "D21:22", "D11:15", "D11:19"}},
		{"D30:05", []string{"D30:5"}},
		{"D", nil},
		{"D:", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SplitEvidenceIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitEvidenceIDs(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNormalizeDiaID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"D1:3", "D1:3"},
		{"D30:05", "D30:5"},
		{"D10:003", "D10:3"},
		{"D1:0", "D1:0"},
	}
	for _, tt := range tests {
		got := NormalizeDiaID(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeDiaID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTokenOverlapF1(t *testing.T) {
	tests := []struct {
		name       string
		prediction string
		reference  string
		want       float64
	}{
		{"exact match", "hello world", "hello world", 1.0},
		{"no overlap", "foo bar", "baz qux", 0.0},
		{"empty both", "", "", 1.0},
		{"empty prediction", "", "hello", 0.0},
		{"empty reference", "hello", "", 0.0},
		{"partial overlap", "the cat sat on the mat", "the cat on the floor", 8.0 / 11.0},
		{"case insensitive", "Hello World", "hello world", 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenOverlapF1(tt.prediction, tt.reference)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("TokenOverlapF1(%q, %q) = %.4f, want %.4f",
					tt.prediction, tt.reference, got, tt.want)
			}
		})
	}
}

func TestBudgetTruncate(t *testing.T) {
	t.Run("within budget returns all", func(t *testing.T) {
		msgs := []string{"short", "message", "here"}
		result, total := BudgetTruncate(msgs, 1000)
		if len(result) != 3 {
			t.Errorf("expected 3 messages, got %d", len(result))
		}
		if total == 0 {
			t.Error("expected non-zero token count")
		}
	})

	t.Run("over budget keeps best first", func(t *testing.T) {
		msgs := []string{
			"best message that is quite long and takes up tokens",
			"good message also fairly long content",
			"worst short",
		}
		result, _ := BudgetTruncate(msgs, 5) // very small budget
		if len(result) == 0 {
			t.Fatal("expected at least one message")
		}
		// Best-ranked (first) should be kept
		if result[0] != "best message that is quite long and takes up tokens" {
			t.Errorf("expected best message kept first, got %q", result[0])
		}
	})

	t.Run("over budget keeps best ranked first", func(t *testing.T) {
		// Messages are sorted by bm25 rank ascending (best/most-negative first).
		// When budget is insufficient, BudgetTruncate must keep the front
		// (best-ranked) messages, not the tail (worst-ranked).
		msgs := []string{
			"best ranked message with some content here",
			"second best message also has content",
			"third message here too",
			"worst ranked short",
		}
		// Budget only fits ~1 message (~10 tokens per message, budget=12)
		result, _ := BudgetTruncate(msgs, 12)
		if len(result) == 0 {
			t.Fatal("expected at least one message")
		}
		if result[0] != "best ranked message with some content here" {
			t.Errorf("expected best-ranked (first) message kept, got %q", result[0])
		}
		// Worst-ranked (last) must NOT appear
		for _, m := range result {
			if m == "worst ranked short" {
				t.Error("worst-ranked message should have been truncated")
			}
		}
	})

	t.Run("preserves original order", func(t *testing.T) {
		msgs := []string{"alpha", "beta", "gamma"}
		result, _ := BudgetTruncate(msgs, 100)
		for i, got := range result {
			if got != msgs[i] {
				t.Errorf("result[%d] = %q, want %q", i, got, msgs[i])
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result, total := BudgetTruncate(nil, 100)
		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
		if total != 0 {
			t.Errorf("expected 0 tokens, got %d", total)
		}
	})
}

func TestRecallHitRate(t *testing.T) {
	// Build a sample with known turns
	sample := &LocomoSample{
		SampleID: "test-sample",
		Conversation: map[string]json.RawMessage{
			"session_1": json.RawMessage(`[
				{"speaker":"A","dia_id":"D1:1","text":"hello world this is a test message with enough length"},
				{"speaker":"B","dia_id":"D1:2","text":"another message for testing recall computation purposes here"},
				{"speaker":"A","dia_id":"D1:3","text":"third turn with some more content to test"}
			]`),
		},
	}

	t.Run("all evidence found", func(t *testing.T) {
		retrieved := "hello world this is a test message with enough length another message for testing recall computation purposes here"
		got := RecallHitRate([]string{"D1:1", "D1:2"}, sample, retrieved)
		if math.Abs(got-1.0) > 1e-9 {
			t.Errorf("RecallHitRate all found = %.4f, want 1.0", got)
		}
	})

	t.Run("partial evidence found", func(t *testing.T) {
		retrieved := "hello world this is a test message with enough length"
		got := RecallHitRate([]string{"D1:1", "D1:2"}, sample, retrieved)
		if math.Abs(got-0.5) > 1e-9 {
			t.Errorf("RecallHitRate partial = %.4f, want 0.5", got)
		}
	})

	t.Run("no evidence required", func(t *testing.T) {
		got := RecallHitRate(nil, sample, "anything")
		if got != 1.0 {
			t.Errorf("RecallHitRate no evidence = %.4f, want 1.0", got)
		}
	})

	t.Run("missing turn excluded from denominator", func(t *testing.T) {
		// D1:1 is found, D99:1 does not exist in sample
		// Should only count resolvable turns in denominator
		retrieved := "hello world this is a test message with enough length"
		got := RecallHitRate([]string{"D1:1", "D99:1"}, sample, retrieved)
		if math.Abs(got-1.0) > 1e-9 {
			t.Errorf("RecallHitRate missing turn = %.4f, want 1.0 (unresolvable excluded)", got)
		}
	})
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "What is the capital of France", []string{"capital", "france"}},
		{
			"stops removed",
			"Who is the president of the United States",
			[]string{"president", "united", "states"},
		},
		{
			"max 6 keywords",
			"one two three four five six seven eight nine ten",
			[]string{"one", "two", "three", "four", "five", "six"},
		},
		{"short words filtered", "I am a go to the store", []string{"am", "store"}},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractKeywords(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

package stats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTracker_Persistence(t *testing.T) {
	dir := t.TempDir()

	tr := NewTracker(dir)
	tr.RecordUsage(100, 50, 150)
	tr.RecordPrompt()

	// Reload from disk
	tr2 := NewTracker(dir)
	s := tr2.GetStats()

	if s.TotalTokens != 150 {
		t.Errorf("expected TotalTokens=150, got %d", s.TotalTokens)
	}
	if s.TotalPromptTokens != 100 {
		t.Errorf("expected TotalPromptTokens=100, got %d", s.TotalPromptTokens)
	}
	if s.TotalCompletionTokens != 50 {
		t.Errorf("expected TotalCompletionTokens=50, got %d", s.TotalCompletionTokens)
	}
	if s.TotalRequests != 1 {
		t.Errorf("expected TotalRequests=1, got %d", s.TotalRequests)
	}
	if s.TotalPrompts != 1 {
		t.Errorf("expected TotalPrompts=1, got %d", s.TotalPrompts)
	}
}

func TestTracker_Accumulation(t *testing.T) {
	tr := NewTracker(t.TempDir())

	tr.RecordUsage(10, 5, 15)
	tr.RecordUsage(20, 10, 30)
	tr.RecordPrompt()
	tr.RecordPrompt()
	tr.RecordPrompt()

	s := tr.GetStats()
	if s.TotalTokens != 45 {
		t.Errorf("expected TotalTokens=45, got %d", s.TotalTokens)
	}
	if s.TotalRequests != 2 {
		t.Errorf("expected TotalRequests=2, got %d", s.TotalRequests)
	}
	if s.TotalPrompts != 3 {
		t.Errorf("expected TotalPrompts=3, got %d", s.TotalPrompts)
	}
	if s.Today.TotalTokens != 45 {
		t.Errorf("expected Today.TotalTokens=45, got %d", s.Today.TotalTokens)
	}
	if s.Today.Requests != 2 {
		t.Errorf("expected Today.Requests=2, got %d", s.Today.Requests)
	}
}

func TestTracker_Reset(t *testing.T) {
	tr := NewTracker(t.TempDir())

	tr.RecordUsage(100, 50, 150)
	tr.RecordPrompt()
	tr.Reset()

	s := tr.GetStats()
	if s.TotalTokens != 0 {
		t.Errorf("expected TotalTokens=0 after reset, got %d", s.TotalTokens)
	}
	if s.TotalRequests != 0 {
		t.Errorf("expected TotalRequests=0 after reset, got %d", s.TotalRequests)
	}
	if s.TotalPrompts != 0 {
		t.Errorf("expected TotalPrompts=0 after reset, got %d", s.TotalPrompts)
	}
	if s.Since.IsZero() {
		t.Error("expected Since to be set after reset")
	}
}

func TestTracker_DayRoll(t *testing.T) {
	dir := t.TempDir()
	tr := NewTracker(dir)

	tr.RecordUsage(100, 50, 150)

	// Manually set Today to a past date to simulate day change
	tr.mu.Lock()
	tr.stats.Today.Date = "2000-01-01"
	tr.mu.Unlock()

	tr.RecordUsage(10, 5, 15)
	s := tr.GetStats()

	// Today should only have the second call
	if s.Today.TotalTokens != 15 {
		t.Errorf("expected Today.TotalTokens=15 after day roll, got %d", s.Today.TotalTokens)
	}
	if s.Today.Requests != 1 {
		t.Errorf("expected Today.Requests=1 after day roll, got %d", s.Today.Requests)
	}

	// Totals should include both
	if s.TotalTokens != 165 {
		t.Errorf("expected TotalTokens=165, got %d", s.TotalTokens)
	}
}

func TestTracker_StateFileCreated(t *testing.T) {
	dir := t.TempDir()
	tr := NewTracker(dir)
	tr.RecordUsage(1, 1, 2)

	stateFile := filepath.Join(dir, "state", "stats.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("expected stats.json to be created")
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0K"},
		{1200, "1.2K"},
		{45200, "45.2K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1200000, "1.2M"},
		{3500000, "3.5M"},
	}
	for _, tt := range tests {
		got := FormatTokenCount(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

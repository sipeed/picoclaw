package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DayStats holds token usage for a single day.
type DayStats struct {
	Date             string `json:"date"`              // "2006-01-02"
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	Requests         int    `json:"requests"` // LLM call count
	Prompts          int    `json:"prompts"`  // user message count
}

// Stats is the full snapshot returned by GetStats.
type Stats struct {
	TotalPromptTokens     int64     `json:"total_prompt_tokens"`
	TotalCompletionTokens int64     `json:"total_completion_tokens"`
	TotalTokens           int64     `json:"total_tokens"`
	TotalRequests         int       `json:"total_requests"`
	TotalPrompts          int       `json:"total_prompts"`
	Since                 time.Time `json:"since"`
	Today                 DayStats  `json:"today"`
}

// Tracker accumulates LLM usage statistics with mutex-protected atomic persistence.
type Tracker struct {
	mu        sync.Mutex
	stats     Stats
	stateFile string
}

// NewTracker creates a tracker that persists to {workspace}/state/stats.json.
func NewTracker(workspace string) *Tracker {
	stateDir := filepath.Join(workspace, "state")
	os.MkdirAll(stateDir, 0755)

	t := &Tracker{
		stateFile: filepath.Join(stateDir, "stats.json"),
	}
	t.load()

	// Initialise Since if this is a fresh tracker
	if t.stats.Since.IsZero() {
		t.stats.Since = time.Now()
	}

	// Lazy day-roll on startup
	t.rollDay()
	return t
}

// RecordUsage records tokens from a single LLM call.
func (t *Tracker) RecordUsage(prompt, completion, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rollDay()

	t.stats.Today.PromptTokens += int64(prompt)
	t.stats.Today.CompletionTokens += int64(completion)
	t.stats.Today.TotalTokens += int64(total)
	t.stats.Today.Requests++

	t.stats.TotalPromptTokens += int64(prompt)
	t.stats.TotalCompletionTokens += int64(completion)
	t.stats.TotalTokens += int64(total)
	t.stats.TotalRequests++

	t.save()
}

// RecordPrompt increments the user-message counter.
func (t *Tracker) RecordPrompt() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rollDay()

	t.stats.Today.Prompts++
	t.stats.TotalPrompts++

	t.save()
}

// GetStats returns a snapshot of the current statistics.
func (t *Tracker) GetStats() Stats {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rollDay()
	return t.stats
}

// Reset zeroes all counters and re-initialises Since.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats = Stats{
		Since: time.Now(),
		Today: DayStats{Date: today()},
	}
	t.save()
}

// rollDay resets Today if the date has changed. Must be called with mu held.
func (t *Tracker) rollDay() {
	d := today()
	if t.stats.Today.Date != d {
		t.stats.Today = DayStats{Date: d}
	}
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// save persists via temp-file + rename (atomic). Must be called with mu held.
func (t *Tracker) save() {
	data, err := json.MarshalIndent(&t.stats, "", "  ")
	if err != nil {
		return
	}
	tmp := t.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	if err := os.Rename(tmp, t.stateFile); err != nil {
		os.Remove(tmp)
	}
}

// load reads the stats file from disk. Called once at init.
func (t *Tracker) load() {
	data, err := os.ReadFile(t.stateFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &t.stats)
}

// FormatTokenCount formats a token count for display (e.g. 1.2K, 3.5M).
func FormatTokenCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

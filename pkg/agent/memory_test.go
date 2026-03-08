package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteCompactionSummaryCreatesTimestampedFile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	store := NewMemoryStore(workspace)
	timestamp := time.Date(2026, time.March, 8, 14, 5, 9, 0, time.Local)

	path, err := store.WriteCompactionSummary(timestamp, "# Summary\n\nBody")
	if err != nil {
		t.Fatalf("WriteCompactionSummary failed: %v", err)
	}

	expected := filepath.Join(
		workspace,
		"memory",
		"202603",
		"20260308-140509.compactions.md",
	)
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "# Summary\n\nBody\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestGetRecentDailyNotesIgnoresCompactionFiles(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	store := NewMemoryStore(workspace)
	if err := store.AppendToday("## Daily\n\nKeep this in prompt context."); err != nil {
		t.Fatalf("AppendToday failed: %v", err)
	}

	if _, err := store.WriteCompactionSummary(
		time.Now(),
		"# Compaction Summary\n\nDo not inject this into the prompt.",
	); err != nil {
		t.Fatalf("WriteCompactionSummary failed: %v", err)
	}

	notes := store.GetRecentDailyNotes(1)
	if !strings.Contains(notes, "Keep this in prompt context.") {
		t.Fatalf("daily note missing from recent notes: %q", notes)
	}
	if strings.Contains(notes, "Do not inject this into the prompt.") {
		t.Fatalf("compaction note leaked into recent daily notes: %q", notes)
	}
}

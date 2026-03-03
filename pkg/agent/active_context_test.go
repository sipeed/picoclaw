package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveContextStore_UpdateAndGet(t *testing.T) {
	s := NewActiveContextStore()
	key := "telegram:12345"

	// Initially empty.
	ac := s.Get(key)
	if len(ac.CurrentFiles) != 0 || len(ac.RecentErrors) != 0 {
		t.Errorf("expected empty context, got %+v", ac)
	}

	// Add errors via Update.
	s.Update(key, RuntimeInput{
		ToolCalls: []ToolCallRecord{
			{Name: "exec", Error: "timeout after 30s"},
			{Name: "read_file", Error: ""},
		},
	})
	ac = s.Get(key)
	if len(ac.RecentErrors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(ac.RecentErrors), ac.RecentErrors)
	}
	if ac.RecentErrors[0] != "[exec] timeout after 30s" {
		t.Errorf("unexpected error: %s", ac.RecentErrors[0])
	}
}

func TestActiveContextStore_FileCapping(t *testing.T) {
	s := NewActiveContextStore()
	key := "cli:direct"

	// Add 7 file paths — should cap at 5, newest first.
	s.UpdateWithFiles(key, RuntimeInput{}, []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go"})
	ac := s.Get(key)
	if len(ac.CurrentFiles) != 5 {
		t.Fatalf("expected 5 files, got %d: %v", len(ac.CurrentFiles), ac.CurrentFiles)
	}
	// Last added (g.go) is prepended, so it should be first.
	if ac.CurrentFiles[0] != "g.go" {
		t.Errorf("expected g.go first, got %s (all: %v)", ac.CurrentFiles[0], ac.CurrentFiles)
	}
}

func TestActiveContextStore_ErrorCapping(t *testing.T) {
	s := NewActiveContextStore()
	key := "wecom:alice"

	for i := 0; i < 5; i++ {
		s.Update(key, RuntimeInput{
			ToolCalls: []ToolCallRecord{{Name: "exec", Error: "err"}},
		})
	}
	ac := s.Get(key)
	if len(ac.RecentErrors) > 3 {
		t.Errorf("expected max 3 errors, got %d", len(ac.RecentErrors))
	}
}

func TestActiveContextStore_FlushAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "active_context.json")

	s := NewActiveContextStore()
	key := "cli:direct"
	s.UpdateWithFiles(key, RuntimeInput{}, []string{"main.go"})
	s.Update(key, RuntimeInput{
		ToolCalls: []ToolCallRecord{{Name: "exec", Error: "failed"}},
	})

	if err := s.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// File must exist.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	// Load into new store.
	s2 := NewActiveContextStore()
	if err := s2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	ac := s2.Get(key)
	if len(ac.CurrentFiles) != 1 || ac.CurrentFiles[0] != "main.go" {
		t.Errorf("unexpected files after reload: %v", ac.CurrentFiles)
	}
	if len(ac.RecentErrors) != 1 {
		t.Errorf("unexpected errors after reload: %v", ac.RecentErrors)
	}
}

func TestActiveContextStore_LoadMissingFile(t *testing.T) {
	s := NewActiveContextStore()
	// Should not error on missing file.
	if err := s.Load("/nonexistent/path.json"); err != nil {
		t.Errorf("Load of missing file should return nil, got: %v", err)
	}
}

func TestActiveContext_Format(t *testing.T) {
	ac := &ActiveContext{
		CurrentFiles: []string{"main.go", "loop.go"},
		RecentErrors: []string{"[exec] timeout"},
	}
	formatted := ac.Format()
	if formatted == "" {
		t.Error("expected non-empty format")
	}
	if len(formatted) == 0 {
		t.Error("Format returned empty string")
	}
}

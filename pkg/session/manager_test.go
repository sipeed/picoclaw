package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}

func TestSessionIndex_BootstrapScopeAndPersist(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)
	scope := "agent:main:telegram:direct:user1"

	active, err := sm.ResolveActive(scope)
	if err != nil {
		t.Fatal(err)
	}
	if active != scope {
		t.Fatalf("active=%q, want %q", active, scope)
	}

	indexPath := filepath.Join(tmp, "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}

	sm2 := NewSessionManager(tmp)
	active2, err := sm2.ResolveActive(scope)
	if err != nil {
		t.Fatal(err)
	}
	if active2 != active {
		t.Fatalf("active2=%q, want %q", active2, active)
	}
}

func TestStartNew_CreatesMonotonicSessionKeys(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}

	s2, err := sm.StartNew(scope)
	if err != nil {
		t.Fatal(err)
	}
	if s2 != scope+"#2" {
		t.Fatalf("s2=%q, want %q", s2, scope+"#2")
	}

	s3, err := sm.StartNew(scope)
	if err != nil {
		t.Fatal(err)
	}
	if s3 != scope+"#3" {
		t.Fatalf("s3=%q, want %q", s3, scope+"#3")
	}
}

func TestListAndResume_ByScopeOrdinal(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}
	if _, err := sm.StartNew(scope); err != nil { // #2
		t.Fatal(err)
	}
	if _, err := sm.StartNew(scope); err != nil { // #3 (active)
		t.Fatal(err)
	}

	list, err := sm.List(scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("len(list)=%d, want 3", len(list))
	}
	if list[0].Ordinal != 1 || list[0].SessionKey != scope+"#3" || !list[0].Active {
		t.Fatalf("list[0]=%+v", list[0])
	}
	if list[2].Ordinal != 3 || list[2].SessionKey != scope {
		t.Fatalf("list[2]=%+v", list[2])
	}

	resumed, err := sm.Resume(scope, 3)
	if err != nil {
		t.Fatal(err)
	}
	if resumed != scope {
		t.Fatalf("resumed=%q, want %q", resumed, scope)
	}

	listAfter, err := sm.List(scope)
	if err != nil {
		t.Fatal(err)
	}
	if !listAfter[2].Active {
		t.Fatalf("listAfter[2] should be active: %+v", listAfter[2])
	}
}

func TestPrune_RemovesOldestFromMemoryAndDisk(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}
	if _, err := sm.StartNew(scope); err != nil { // #2
		t.Fatal(err)
	}
	if _, err := sm.StartNew(scope); err != nil { // #3 (active)
		t.Fatal(err)
	}

	keys := []string{scope, scope + "#2", scope + "#3"}
	for _, key := range keys {
		sm.AddMessage(key, "user", "hello")
		if err := sm.Save(key); err != nil {
			t.Fatalf("save %q: %v", key, err)
		}
	}

	pruned, err := sm.Prune(scope, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 1 || pruned[0] != scope {
		t.Fatalf("pruned=%v, want [%s]", pruned, scope)
	}

	if got := len(sm.GetHistory(scope)); got != 0 {
		t.Fatalf("expected deleted session history len=0, got %d", got)
	}

	removedFile := filepath.Join(dir, sanitizeFilename(scope)+".json")
	if _, err := os.Stat(removedFile); !os.IsNotExist(err) {
		t.Fatalf("expected %s deleted, stat err=%v", removedFile, err)
	}

	list, err := sm.List(scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list)=%d, want 2", len(list))
	}
	if list[0].SessionKey != scope+"#3" || list[1].SessionKey != scope+"#2" {
		t.Fatalf("list order after prune = %+v", list)
	}
}

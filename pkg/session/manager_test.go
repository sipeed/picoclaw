package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestStartNew_PersistsSessionFileWithoutManualSave(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}

	s2, err := sm.StartNew(scope)
	if err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(dir, sanitizeFilename(s2)+".json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("expected %s to exist: %v", sessionPath, err)
	}

	indexPath := filepath.Join(dir, sessionIndexFilename)
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}

	var idx sessionIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}

	scoped := idx.Scopes[scope]
	if scoped == nil {
		t.Fatalf("expected scope %q in index", scope)
	}
	if scoped.ActiveSessionKey != s2 {
		t.Fatalf("active=%q, want %q", scoped.ActiveSessionKey, s2)
	}
	if len(scoped.OrderedSessions) == 0 || scoped.OrderedSessions[0] != s2 {
		t.Fatalf("ordered_sessions=%v, want first=%q", scoped.OrderedSessions, s2)
	}
}

func TestStartNew_DoesNotMutateIndexWhenSessionPersistFails(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)
	scope := "../invalid/scope"

	_, err := sm.StartNew(scope)
	if err == nil {
		t.Fatalf("expected StartNew to fail for invalid persisted session key")
	}

	if _, exists := sm.index.Scopes[scope]; exists {
		t.Fatalf("scope %q should not be added to index on session persist failure", scope)
	}

	if _, exists := sm.sessions[scope+"#2"]; exists {
		t.Fatalf("session %q should not remain in memory on session persist failure", scope+"#2")
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	if len(files) != 0 {
		t.Fatalf("storage should stay untouched, found files: %v", files)
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

func TestLoadIndex_SelfHealsStaleReferences(t *testing.T) {
	dir := t.TempDir()
	scopeA := "agent:main:telegram:direct:user1"
	scopeB := "agent:main:telegram:direct:user2"
	validNewest := scopeA + "#3"
	validOlder := scopeA + "#2"

	seed := NewSessionManager(dir)
	seed.AddMessage(validNewest, "user", "hello")
	seed.AddMessage(validOlder, "user", "hello")
	if err := seed.Save(validNewest); err != nil {
		t.Fatalf("save %q: %v", validNewest, err)
	}
	if err := seed.Save(validOlder); err != nil {
		t.Fatalf("save %q: %v", validOlder, err)
	}

	stale := sessionIndex{
		Version: 1,
		Scopes: map[string]*scopeIndex{
			scopeA: {
				ActiveSessionKey: scopeA + "#999",
				OrderedSessions: []string{
					validNewest,
					validNewest,
					scopeA + "#404",
					validOlder,
				},
			},
			scopeB: {
				ActiveSessionKey: scopeB,
				OrderedSessions:  []string{scopeB},
			},
		},
	}

	raw, err := json.MarshalIndent(stale, "", "  ")
	if err != nil {
		t.Fatalf("marshal stale index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, sessionIndexFilename), raw, 0o644); err != nil {
		t.Fatalf("write stale index: %v", err)
	}

	reloaded := NewSessionManager(dir)

	indexRaw, err := os.ReadFile(filepath.Join(dir, sessionIndexFilename))
	if err != nil {
		t.Fatalf("read healed index: %v", err)
	}

	var healed sessionIndex
	if err := json.Unmarshal(indexRaw, &healed); err != nil {
		t.Fatalf("unmarshal healed index: %v", err)
	}

	scopeAHealed := healed.Scopes[scopeA]
	if scopeAHealed == nil {
		t.Fatalf("expected scope %q in healed index", scopeA)
	}
	if scopeAHealed.ActiveSessionKey != validNewest {
		t.Fatalf("active=%q, want %q", scopeAHealed.ActiveSessionKey, validNewest)
	}
	if len(scopeAHealed.OrderedSessions) != 2 {
		t.Fatalf("ordered_sessions=%v, want len=2", scopeAHealed.OrderedSessions)
	}
	if scopeAHealed.OrderedSessions[0] != validNewest || scopeAHealed.OrderedSessions[1] != validOlder {
		t.Fatalf("ordered_sessions=%v, want [%s %s]", scopeAHealed.OrderedSessions, validNewest, validOlder)
	}
	if _, exists := healed.Scopes[scopeB]; exists {
		t.Fatalf("expected stale scope %q removed", scopeB)
	}

	list, err := reloaded.List(scopeA)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list)=%d, want 2", len(list))
	}
	if !list[0].Active || list[0].SessionKey != validNewest {
		t.Fatalf("list[0]=%+v, want active newest", list[0])
	}
}

func TestDeleteSession_FileDeleteFailureIsDeferredAndRetriedOnStartup(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}
	sessionKey, err := sm.StartNew(scope)
	if err != nil {
		t.Fatal(err)
	}

	oldRemoveFile := removeFile
	oldWarningWriter := warningWriter
	var warnings bytes.Buffer
	warningWriter = &warnings
	removeFile = func(path string) error {
		if strings.HasSuffix(path, sanitizeFilename(sessionKey)+".json") {
			return errors.New("permission denied")
		}
		return oldRemoveFile(path)
	}
	t.Cleanup(func() {
		removeFile = oldRemoveFile
		warningWriter = oldWarningWriter
	})

	if err := sm.DeleteSession(sessionKey); err != nil {
		t.Fatalf("DeleteSession(%q) returned unexpected error: %v", sessionKey, err)
	}
	if got := warnings.String(); !strings.Contains(got, "deferred retry on startup") {
		t.Fatalf("expected deferred-delete warning, got: %q", got)
	}

	sessionPath := filepath.Join(dir, sanitizeFilename(sessionKey)+".json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("expected deferred file %q to still exist, err=%v", sessionPath, err)
	}

	list, err := sm.List(scope)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list {
		if item.SessionKey == sessionKey {
			t.Fatalf("deleted session key %q should not remain in index list", sessionKey)
		}
	}

	if len(sm.index.PendingDeletes) != 1 || sm.index.PendingDeletes[0] != sessionKey {
		t.Fatalf("pending_deletes=%v, want [%q]", sm.index.PendingDeletes, sessionKey)
	}

	removeFile = oldRemoveFile
	reloaded := NewSessionManager(dir)
	if len(reloaded.index.PendingDeletes) != 0 {
		t.Fatalf("pending_deletes should be drained on startup retry, got %v", reloaded.index.PendingDeletes)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be removed by startup retry, stat err=%v", sessionPath, err)
	}
	if _, exists := reloaded.sessions[sessionKey]; exists {
		t.Fatalf("session %q should not be present in memory after startup retry", sessionKey)
	}
}

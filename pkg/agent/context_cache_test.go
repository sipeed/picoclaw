package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// setupWorkspace creates a temporary workspace with standard directories and optional files.
// Returns the tmpDir path; caller should defer os.RemoveAll(tmpDir).
func setupWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "picoclaw-test-*")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(tmpDir, "memory"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "skills"), 0o755)
	for name, content := range files {
		dir := filepath.Dir(filepath.Join(tmpDir, name))
		os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return tmpDir
}

// TestSingleSystemMessage verifies that BuildMessages always produces exactly one
// system message regardless of summary/history variations.
// Fix: multiple system messages break Anthropic (top-level system param) and
// Codex (only reads last system message as instructions).
func TestSingleSystemMessage(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Identity\nTest agent.",
	})
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	tests := []struct {
		name    string
		history []providers.Message
		summary string
		message string
	}{
		{
			name:    "no summary, no history",
			summary: "",
			message: "hello",
		},
		{
			name:    "with summary",
			summary: "Previous conversation discussed X",
			message: "hello",
		},
		{
			name: "with history and summary",
			history: []providers.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
			},
			summary: strings.Repeat("Long summary text. ", 50),
			message: "new message",
		},
		{
			name: "system message in history is filtered",
			history: []providers.Message{
				{Role: "system", Content: "stale system prompt from previous session"},
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
			},
			summary: "",
			message: "new message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := cb.BuildMessages(tt.history, tt.summary, tt.message, nil, "test", "chat1")

			systemCount := 0
			for _, m := range msgs {
				if m.Role == "system" {
					systemCount++
				}
			}
			if systemCount != 1 {
				t.Errorf("expected exactly 1 system message, got %d", systemCount)
			}
			if msgs[0].Role != "system" {
				t.Errorf("first message should be system, got %s", msgs[0].Role)
			}
			if msgs[len(msgs)-1].Role != "user" {
				t.Errorf("last message should be user, got %s", msgs[len(msgs)-1].Role)
			}

			// System message must contain identity (static) and time (dynamic)
			sys := msgs[0].Content
			if !strings.Contains(sys, "picoclaw") {
				t.Error("system message missing identity")
			}
			if !strings.Contains(sys, "Current Time") {
				t.Error("system message missing dynamic time context")
			}

			// Summary handling
			if tt.summary != "" {
				if !strings.Contains(sys, "CONTEXT_SUMMARY:") {
					t.Error("summary present but CONTEXT_SUMMARY prefix missing")
				}
				if !strings.Contains(sys, tt.summary[:20]) {
					t.Error("summary content not found in system message")
				}
			} else {
				if strings.Contains(sys, "CONTEXT_SUMMARY:") {
					t.Error("CONTEXT_SUMMARY should not appear without summary")
				}
			}
		})
	}
}

// TestMtimeAutoInvalidation verifies that the cache detects source file changes
// via mtime without requiring explicit InvalidateCache().
// Fix: original implementation had no auto-invalidation — edits to bootstrap files,
// memory, or skills were invisible until process restart.
func TestMtimeAutoInvalidation(t *testing.T) {
	tests := []struct {
		name       string
		file       string // relative path inside workspace
		contentV1  string
		contentV2  string
		checkField string // substring to verify in rebuilt prompt
	}{
		{
			name:       "bootstrap file change",
			file:       "IDENTITY.md",
			contentV1:  "# Original Identity",
			contentV2:  "# Updated Identity",
			checkField: "Updated Identity",
		},
		{
			name:       "memory file change",
			file:       "memory/MEMORY.md",
			contentV1:  "# Memory\nUser likes Go.",
			contentV2:  "# Memory\nUser likes Rust.",
			checkField: "User likes Rust",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupWorkspace(t, map[string]string{tt.file: tt.contentV1})
			defer os.RemoveAll(tmpDir)

			cb := NewContextBuilder(tmpDir)

			sp1 := cb.BuildSystemPromptWithCache()

			// Overwrite file and set future mtime to ensure detection.
			// Use 2s offset for filesystem mtime resolution safety (some FS
			// have 1s or coarser granularity, especially in CI containers).
			fullPath := filepath.Join(tmpDir, tt.file)
			os.WriteFile(fullPath, []byte(tt.contentV2), 0o644)
			future := time.Now().Add(2 * time.Second)
			os.Chtimes(fullPath, future, future)

			// Verify sourceFilesChangedLocked detects the mtime change
			cb.systemPromptMutex.RLock()
			changed := cb.sourceFilesChangedLocked()
			cb.systemPromptMutex.RUnlock()
			if !changed {
				t.Fatalf("sourceFilesChangedLocked() should detect %s change", tt.file)
			}

			// Should auto-rebuild without explicit InvalidateCache()
			sp2 := cb.BuildSystemPromptWithCache()
			if sp1 == sp2 {
				t.Errorf("cache not rebuilt after %s change", tt.file)
			}
			if !strings.Contains(sp2, tt.checkField) {
				t.Errorf("rebuilt prompt missing expected content %q", tt.checkField)
			}
		})
	}

	// Skills directory mtime change
	t.Run("skills dir change", func(t *testing.T) {
		tmpDir := setupWorkspace(t, nil)
		defer os.RemoveAll(tmpDir)

		cb := NewContextBuilder(tmpDir)
		_ = cb.BuildSystemPromptWithCache() // populate cache

		// Touch skills directory (simulate new skill installed)
		skillsDir := filepath.Join(tmpDir, "skills")
		future := time.Now().Add(2 * time.Second)
		os.Chtimes(skillsDir, future, future)

		// Verify sourceFilesChangedLocked detects it (cache is rebuilt)
		// We confirm by checking internal state: a second call should rebuild.
		cb.systemPromptMutex.RLock()
		changed := cb.sourceFilesChangedLocked()
		cb.systemPromptMutex.RUnlock()
		if !changed {
			t.Error("sourceFilesChangedLocked() should detect skills dir mtime change")
		}
	})
}

// TestExplicitInvalidateCache verifies that InvalidateCache() forces a rebuild
// even when source files haven't changed (useful for tests and reload commands).
func TestExplicitInvalidateCache(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Test Identity",
	})
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	sp1 := cb.BuildSystemPromptWithCache()
	cb.InvalidateCache()
	sp2 := cb.BuildSystemPromptWithCache()

	if sp1 != sp2 {
		t.Error("prompt should be identical after invalidate+rebuild when files unchanged")
	}

	// Verify cachedAt was reset
	cb.InvalidateCache()
	cb.systemPromptMutex.RLock()
	if !cb.cachedAt.IsZero() {
		t.Error("cachedAt should be zero after InvalidateCache()")
	}
	cb.systemPromptMutex.RUnlock()
}

// TestCacheStability verifies that the static prompt is stable across repeated calls
// when no files change (regression test for issue #607).
func TestCacheStability(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md": "# Identity\nContent",
		"SOUL.md":     "# Soul\nContent",
	})
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	results := make([]string, 5)
	for i := range results {
		results[i] = cb.BuildSystemPromptWithCache()
	}
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("cached prompt changed between call 0 and %d", i)
		}
	}

	// Static prompt must NOT contain per-request data
	if strings.Contains(results[0], "Current Time") {
		t.Error("static cached prompt should not contain time (added dynamically)")
	}
}

// TestNewFileCreationInvalidatesCache verifies that creating a source file that
// did not exist when the cache was built triggers a cache rebuild.
// This catches the "from nothing to something" edge case that the old
// modifiedSince (return false on stat error) would miss.
func TestNewFileCreationInvalidatesCache(t *testing.T) {
	tests := []struct {
		name       string
		file       string // relative path inside workspace
		content    string
		checkField string // substring to verify in rebuilt prompt
	}{
		{
			name:       "new bootstrap file",
			file:       "SOUL.md",
			content:    "# Soul\nBe kind and helpful.",
			checkField: "Be kind and helpful",
		},
		{
			name:       "new memory file",
			file:       "memory/MEMORY.md",
			content:    "# Memory\nUser prefers dark mode.",
			checkField: "User prefers dark mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with an empty workspace (no bootstrap/memory files)
			tmpDir := setupWorkspace(t, nil)
			defer os.RemoveAll(tmpDir)

			cb := NewContextBuilder(tmpDir)

			// Populate cache — file does not exist yet
			sp1 := cb.BuildSystemPromptWithCache()
			if strings.Contains(sp1, tt.checkField) {
				t.Fatalf("prompt should not contain %q before file is created", tt.checkField)
			}

			// Create the file after cache was built
			fullPath := filepath.Join(tmpDir, tt.file)
			os.MkdirAll(filepath.Dir(fullPath), 0o755)
			if err := os.WriteFile(fullPath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			// Set future mtime to guarantee detection
			future := time.Now().Add(2 * time.Second)
			os.Chtimes(fullPath, future, future)

			// Cache should auto-invalidate because file went from absent -> present
			sp2 := cb.BuildSystemPromptWithCache()
			if !strings.Contains(sp2, tt.checkField) {
				t.Errorf("cache not invalidated on new file creation: expected %q in prompt", tt.checkField)
			}
		})
	}
}

// TestSkillFileContentChange verifies that modifying a skill file's content
// (not just the directory structure) invalidates the cache.
// This is the scenario where directory mtime alone is insufficient — on most
// filesystems, editing a file inside a directory does NOT update the parent
// directory's mtime.
func TestSkillFileContentChange(t *testing.T) {
	skillMD := `---
name: test-skill
description: "A test skill"
---
# Test Skill v1
Original content.`

	tmpDir := setupWorkspace(t, map[string]string{
		"skills/test-skill/SKILL.md": skillMD,
	})
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	// Populate cache
	sp1 := cb.BuildSystemPromptWithCache()
	_ = sp1 // cache is warm

	// Modify the skill file content (without touching the skills/ directory)
	updatedSkillMD := `---
name: test-skill
description: "An updated test skill"
---
# Test Skill v2
Updated content.`

	skillPath := filepath.Join(tmpDir, "skills", "test-skill", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(updatedSkillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set future mtime on the skill file only (NOT the directory)
	future := time.Now().Add(2 * time.Second)
	os.Chtimes(skillPath, future, future)

	// Verify that sourceFilesChangedLocked detects the content change
	cb.systemPromptMutex.RLock()
	changed := cb.sourceFilesChangedLocked()
	cb.systemPromptMutex.RUnlock()
	if !changed {
		t.Error("sourceFilesChangedLocked() should detect skill file content change")
	}

	// Verify cache is actually rebuilt with new content
	sp2 := cb.BuildSystemPromptWithCache()
	if sp1 == sp2 && strings.Contains(sp1, "test-skill") {
		// If the skill appeared in the prompt and the prompt didn't change,
		// the cache was not invalidated.
		t.Error("cache should be invalidated when skill file content changes")
	}
}

// TestConcurrentBuildSystemPromptWithCache verifies that multiple goroutines
// can safely call BuildSystemPromptWithCache concurrently without producing
// empty results, panics, or data races.
// Run with: go test -race ./pkg/agent/ -run TestConcurrentBuildSystemPromptWithCache
func TestConcurrentBuildSystemPromptWithCache(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"IDENTITY.md":          "# Identity\nConcurrency test agent.",
		"SOUL.md":              "# Soul\nBe helpful.",
		"memory/MEMORY.md":     "# Memory\nUser prefers Go.",
		"skills/demo/SKILL.md": "---\nname: demo\ndescription: \"demo skill\"\n---\n# Demo",
	})
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	errs := make(chan string, goroutines*iterations)

	for g := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := range iterations {
				result := cb.BuildSystemPromptWithCache()
				if result == "" {
					errs <- "empty prompt returned"
					return
				}
				if !strings.Contains(result, "picoclaw") {
					errs <- "prompt missing identity"
					return
				}

				// Also exercise BuildMessages concurrently
				msgs := cb.BuildMessages(nil, "", "hello", nil, "test", "chat")
				if len(msgs) < 2 {
					errs <- "BuildMessages returned fewer than 2 messages"
					return
				}
				if msgs[0].Role != "system" {
					errs <- "first message not system"
					return
				}

				// Occasionally invalidate to exercise the write path
				if i%10 == 0 {
					cb.InvalidateCache()
				}
			}
		}(g)
	}

	wg.Wait()
	close(errs)

	for errMsg := range errs {
		t.Errorf("concurrent access error: %s", errMsg)
	}
}

// BenchmarkBuildMessagesWithCache measures caching performance.

// TestEmptyWorkspaceBaselineDetectsNewFiles verifies that when the cache is
// built on an empty workspace (no tracked files exist), creating a file
// afterwards still triggers cache invalidation. This validates the
// time.Unix(1, 0) fallback for maxMtime: any real file's mtime is after epoch,
// so fileChangedSince correctly detects the absent -> present transition AND
// the mtime comparison succeeds even without artificially inflated Chtimes.
func TestEmptyWorkspaceBaselineDetectsNewFiles(t *testing.T) {
	// Empty workspace: no bootstrap files, no memory, no skills content.
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	// Build cache — all tracked files are absent, maxMtime falls back to epoch.
	sp1 := cb.BuildSystemPromptWithCache()

	// Create a bootstrap file with natural mtime (no Chtimes manipulation).
	// The file's mtime should be the current wall-clock time, which is
	// strictly after time.Unix(1, 0).
	soulPath := filepath.Join(tmpDir, "SOUL.md")
	if err := os.WriteFile(soulPath, []byte("# Soul\nNewly created."), 0o644); err != nil {
		t.Fatal(err)
	}

	// Cache should detect the new file via existedAtCache (absent -> present).
	cb.systemPromptMutex.RLock()
	changed := cb.sourceFilesChangedLocked()
	cb.systemPromptMutex.RUnlock()
	if !changed {
		t.Fatal("sourceFilesChangedLocked should detect newly created file on empty workspace")
	}

	sp2 := cb.BuildSystemPromptWithCache()
	if !strings.Contains(sp2, "Newly created") {
		t.Error("rebuilt prompt should contain new file content")
	}
	if sp1 == sp2 {
		t.Error("cache should have been invalidated after file creation")
	}
}

// BenchmarkBuildMessagesWithCache measures caching performance.
func BenchmarkBuildMessagesWithCache(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "picoclaw-bench-*")
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "memory"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "skills"), 0o755)
	for _, name := range []string{"IDENTITY.md", "SOUL.md", "USER.md"} {
		os.WriteFile(filepath.Join(tmpDir, name), []byte(strings.Repeat("Content.\n", 10)), 0o644)
	}

	cb := NewContextBuilder(tmpDir)
	history := []providers.Message{
		{Role: "user", Content: "previous message"},
		{Role: "assistant", Content: "previous response"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.BuildMessages(history, "summary", "new message", nil, "cli", "test")
	}
}

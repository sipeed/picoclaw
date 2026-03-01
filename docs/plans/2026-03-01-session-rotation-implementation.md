# Scope-Aware Session Rotation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `/new` (`/reset`) and `/session list|resume` with scope-aware active session switching and per-scope backlog pruning, without changing routing semantics.

**Architecture:** Extend `SessionManager` to own both session content and a persisted scope index (`sessions/index.json`), then resolve active session keys in `AgentLoop` before normal processing. Command handling remains in `AgentLoop.handleCommand` for cross-channel consistency, and all session operations stay in the session domain (not `state`).

**Tech Stack:** Go, stdlib `encoding/json`/`sync`/`os`, existing `pkg/session`, `pkg/agent`, `pkg/routing`, table-driven tests with `go test`.

---

**Required skills during execution:** @test-driven-development, @verification-before-completion, @requesting-code-review

### Task 1: Add Session Index Model and Persistence in SessionManager

**Files:**
- Modify: `pkg/session/manager.go`
- Test: `pkg/session/manager_test.go`

**Step 1: Write the failing test**

```go
func TestSessionIndex_BootstrapScopeAndPersist(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	active, err := sm.ResolveActive("agent:main:telegram:direct:user1")
	if err != nil {
		t.Fatal(err)
	}
	if active != "agent:main:telegram:direct:user1" {
		t.Fatalf("active=%q", active)
	}

	sm2 := NewSessionManager(tmp)
	active2, err := sm2.ResolveActive("agent:main:telegram:direct:user1")
	if err != nil || active2 != active {
		t.Fatalf("active2=%q err=%v", active2, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/session -run TestSessionIndex_BootstrapScopeAndPersist -v`  
Expected: FAIL with undefined `ResolveActive`.

**Step 3: Write minimal implementation**

```go
type scopeIndex struct {
	ActiveSessionKey string   `json:"active_session_key"`
	OrderedSessions  []string `json:"ordered_sessions"`
	UpdatedAt        string   `json:"updated_at"`
}
type sessionIndex struct {
	Version int                   `json:"version"`
	Scopes  map[string]*scopeIndex `json:"scopes"`
}
// Add indexPath/index fields to SessionManager and load/save helpers.
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/session -run TestSessionIndex_BootstrapScopeAndPersist -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/session/manager.go pkg/session/manager_test.go
git commit -m "feat(session): add persisted scope index model in session manager"
```

### Task 2: Add Active Session API and Session Rotation

**Files:**
- Modify: `pkg/session/manager.go`
- Test: `pkg/session/manager_test.go`

**Step 1: Write the failing test**

```go
func TestStartNew_CreatesMonotonicSessionKeys(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	scope := "agent:main:telegram:direct:user1"

	if _, err := sm.ResolveActive(scope); err != nil {
		t.Fatal(err)
	}
	s2, err := sm.StartNew(scope)
	if err != nil || s2 != scope+"#2" {
		t.Fatalf("s2=%q err=%v", s2, err)
	}
	s3, err := sm.StartNew(scope)
	if err != nil || s3 != scope+"#3" {
		t.Fatalf("s3=%q err=%v", s3, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/session -run TestStartNew_CreatesMonotonicSessionKeys -v`  
Expected: FAIL with undefined `StartNew`.

**Step 3: Write minimal implementation**

```go
func (sm *SessionManager) ResolveActive(scopeKey string) (string, error) { ... }
func (sm *SessionManager) StartNew(scopeKey string) (string, error) { ... } // append #n
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/session -run TestStartNew_CreatesMonotonicSessionKeys -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/session/manager.go pkg/session/manager_test.go
git commit -m "feat(session): add resolve-active and start-new session rotation APIs"
```

### Task 3: Add List/Resume API with Stable Numbering

**Files:**
- Modify: `pkg/session/manager.go`
- Test: `pkg/session/manager_test.go`

**Step 1: Write the failing test**

```go
func TestListAndResume_ByScopeOrdinal(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	scope := "agent:main:telegram:direct:user1"
	_, _ = sm.ResolveActive(scope)
	_, _ = sm.StartNew(scope) // #2
	_, _ = sm.StartNew(scope) // #3 (active)

	list, err := sm.List(scope)
	if err != nil || len(list) != 3 || list[0].Ordinal != 1 {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	resumed, err := sm.Resume(scope, 3) // oldest
	if err != nil || resumed != scope {
		t.Fatalf("resumed=%q err=%v", resumed, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/session -run TestListAndResume_ByScopeOrdinal -v`  
Expected: FAIL with undefined `List/Resume`.

**Step 3: Write minimal implementation**

```go
type SessionMeta struct {
	Ordinal    int
	SessionKey string
	UpdatedAt  time.Time
	MessageCnt int
	Active     bool
}
func (sm *SessionManager) List(scopeKey string) ([]SessionMeta, error) { ... }
func (sm *SessionManager) Resume(scopeKey string, index int) (string, error) { ... }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/session -run TestListAndResume_ByScopeOrdinal -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/session/manager.go pkg/session/manager_test.go
git commit -m "feat(session): add scope-local session list and resume APIs"
```

### Task 4: Add Prune/Delete API and Disk Cleanup

**Files:**
- Modify: `pkg/session/manager.go`
- Test: `pkg/session/manager_test.go`

**Step 1: Write the failing test**

```go
func TestPrune_RemovesOldestFromMemoryAndDisk(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)
	scope := "agent:main:telegram:direct:user1"
	_, _ = sm.ResolveActive(scope)
	_, _ = sm.StartNew(scope) // #2
	_, _ = sm.StartNew(scope) // #3
	if err := sm.Save(scope); err != nil { t.Fatal(err) }

	pruned, err := sm.Prune(scope, 2)
	if err != nil || len(pruned) != 1 || pruned[0] != scope {
		t.Fatalf("pruned=%v err=%v", pruned, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/session -run TestPrune_RemovesOldestFromMemoryAndDisk -v`  
Expected: FAIL with undefined `Prune/DeleteSession`.

**Step 3: Write minimal implementation**

```go
func (sm *SessionManager) DeleteSession(sessionKey string) error { ... } // map + file + index fix
func (sm *SessionManager) Prune(scopeKey string, limit int) ([]string, error) { ... } // oldest first
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/session -run TestPrune_RemovesOldestFromMemoryAndDisk -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/session/manager.go pkg/session/manager_test.go
git commit -m "feat(session): add scope backlog prune and delete-session cleanup"
```

### Task 5: Add Session Backlog Config

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/defaults.go`
- Test: `pkg/config/config_test.go`

**Step 1: Write the failing test**

```go
func TestDefaultConfig_SessionBacklogLimit(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Session.BacklogLimit != 20 {
		t.Fatalf("backlog_limit=%d, want 20", cfg.Session.BacklogLimit)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config -run TestDefaultConfig_SessionBacklogLimit -v`  
Expected: FAIL with missing field.

**Step 3: Write minimal implementation**

```go
type SessionConfig struct {
	DMScope       string              `json:"dm_scope,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
	BacklogLimit  int                 `json:"backlog_limit,omitempty"`
}
// defaults.go: BacklogLimit = 20
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/config -run TestDefaultConfig_SessionBacklogLimit -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/defaults.go pkg/config/config_test.go
git commit -m "feat(config): add session backlog limit configuration"
```

### Task 6: Wire AgentLoop to Resolve Active Session by Scope

**Files:**
- Modify: `pkg/agent/loop.go`
- Test: `pkg/agent/loop_test.go`

**Step 1: Write the failing test**

```go
func TestProcessMessage_UsesResolvedActiveSession(t *testing.T) {
	// Build loop with mock/default agent and pre-seeded session rotation in manager.
	// Assert message writes to active rotated session, not bare route session key.
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/agent -run TestProcessMessage_UsesResolvedActiveSession -v`  
Expected: FAIL before loop wiring.

**Step 3: Write minimal implementation**

```go
scopeKey := route.SessionKey
if msg.SessionKey != "" && strings.HasPrefix(msg.SessionKey, "agent:") {
	scopeKey = msg.SessionKey
}
sessionKey, err := agent.Sessions.ResolveActive(scopeKey)
if err != nil { return "", err }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/agent -run TestProcessMessage_UsesResolvedActiveSession -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/agent/loop.go pkg/agent/loop_test.go
git commit -m "refactor(agent): resolve active session key from scope before processing"
```

### Task 7: Implement `/new` and `/session list|resume` Commands

**Files:**
- Modify: `pkg/agent/loop.go`
- Test: `pkg/agent/loop_test.go`

**Step 1: Write the failing test**

```go
func TestHandleCommand_NewAndSessionCommands(t *testing.T) {
	// /new should create rotated session and prune if needed
	// /session list should return numbered entries
	// /session resume 2 should switch active session
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/agent -run TestHandleCommand_NewAndSessionCommands -v`  
Expected: FAIL with unknown command behavior.

**Step 3: Write minimal implementation**

```go
case "/new", "/reset":
  // compute scopeKey from route input
  // StartNew + Prune(backlogLimit)
case "/session":
  // subcommands list/resume
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/agent -run TestHandleCommand_NewAndSessionCommands -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/agent/loop.go pkg/agent/loop_test.go
git commit -m "feat(agent): add scope-aware new/list/resume session commands"
```

### Task 8: Docs + End-to-End Verification (Phase 1)

**Files:**
- Modify: `README.md`
- Modify: `README.zh.md`
- Modify: `config/config.example.json`

**Step 1: Write/extend failing expectation test (optional smoke)**

```go
// Optional command smoke in loop tests:
// Verify "/session list" output includes active marker and ordinals.
```

**Step 2: Run targeted verification**

Run: `go test ./pkg/session ./pkg/agent ./pkg/config -count=1`  
Expected: PASS.

**Step 3: Update docs**

```markdown
- Add commands: /new (/reset), /session list, /session resume <n>
- Explain scope-aware behavior (aligned with dm_scope)
- Document session.backlog_limit
```

**Step 4: Run full verification**

Run: `go test ./... -count=1`  
Expected: PASS.

Run: `git status --short`  
Expected: clean working tree.

**Step 5: Commit**

```bash
git add README.md README.zh.md config/config.example.json
git commit -m "docs(session): document scope-aware session rotation commands"
```

### Task 9: Phase 2 Planning Hook (No Feature Logic Yet)

**Files:**
- Modify: `pkg/session/manager.go`
- Test: `pkg/session/manager_test.go`

**Step 1: Write failing test for hook contract**

```go
func TestArchiveHook_InvokedOnPrunedSessions(t *testing.T) {
	// Set hook callback, trigger prune, verify callback receives pruned keys.
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/session -run TestArchiveHook_InvokedOnPrunedSessions -v`  
Expected: FAIL with missing hook.

**Step 3: Write minimal implementation**

```go
// Optional callback setter in SessionManager:
// SetArchiveCallback(func(scopeKey string, pruned []string))
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/session -run TestArchiveHook_InvokedOnPrunedSessions -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/session/manager.go pkg/session/manager_test.go
git commit -m "refactor(session): add archive hook for phase-2 memory extraction pipeline"
```


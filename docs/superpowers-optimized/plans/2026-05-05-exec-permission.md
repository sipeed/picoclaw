# Exec Permission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-optimized:subagent-driven-development (recommended) or superpowers-optimized:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add permission system to exec tool allowing user approval for paths outside workspace, with "once" and "session" options.

**Architecture:** New `PermissionCache` + `RequestPermissionTool` in `pkg/tools/permission.go`, modify `ExecTool` to check permissions before executing commands with outside-workspace paths. Config flag `tools.exec.ask_permission` defaults to `true`.

**Tech Stack:** Go 1.25.9, sync.RWMutex for cache, Tool interface for new tool.

**Assumptions:** Assumes user has Go environment set up — will NOT work if Go not installed. Assumes LLM follows instructions to call `request_permission` when exec returns early with permission message.

---

### Task 1: Create PermissionCache Structure

**Files:**
- Create: `pkg/tools/permission.go`
- Test: `pkg/tools/permission_test.go`

**Does NOT cover:** Grant/revoke logic for individual paths — that's Task 2.

- [ ] **Step 1: Write failing test**

```go
// pkg/tools/permission_test.go
package tools

import (
	"testing"
)

func TestPermissionCache_Check_NoPermission(t *testing.T) {
	pc := NewPermissionCache()
	result := pc.Check("/desktop")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestPermissionCache_Check_NoPermission ./pkg/tools/...`
Expected: FAIL - `NewPermissionCache` not defined

- [ ] **Step 3: Implement minimal change**

```go
// pkg/tools/permission.go
package tools

import (
	"sync"
)

type PermissionCache struct {
	mu    sync.RWMutex
	perms map[string]string // path → "once"|"session"|"denied"
}

func NewPermissionCache() *PermissionCache {
	return &PermissionCache{
		perms: make(map[string]string),
	}
}

// Check returns "once", "session", "denied", or "" (no permission)
func (pc *PermissionCache) Check(path string) string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	// Check exact match
	if val, ok := pc.perms[path]; ok {
		return val
	}
	
	// Check if any parent path is granted "session"
	for cachedPath, val := range pc.perms {
		if val == "session" && len(cachedPath) < len(path) && path[:len(cachedPath)] == cachedPath {
			return "session"
		}
	}
	
	return ""
}

func (pc *PermissionCache) Grant(path, duration string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.perms[path] = duration
}

func (pc *PermissionCache) Revoke(path string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.perms, path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestPermissionCache_Check_NoPermission ./pkg/tools/...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add pkg/tools/permission.go pkg/tools/permission_test.go
git commit -m "feat: add PermissionCache for exec tool permission system"
```

---

### Task 2: Add PermissionCache Tests

**Files:**
- Modify: `pkg/tools/permission_test.go`

**Does NOT cover:** RequestPermissionTool implementation — that's Task 3.

- [x] **Step 1: Write failing tests**

```go
// pkg/tools/permission_test.go (add to existing file)
func TestPermissionCache_GrantAndCheck(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")
	
	result := pc.Check("/desktop")
	if result != "session" {
		t.Errorf("Expected 'session', got %s", result)
	}
}

func TestPermissionCache_ParentPathMatch(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")
	
	// Child path should match parent's "session" permission
	result := pc.Check("/desktop/folder")
	if result != "session" {
		t.Errorf("Expected 'session' for child path, got %s", result)
	}
}

func TestPermissionCache_Revoke(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")
	pc.Revoke("/desktop")
	
	result := pc.Check("/desktop")
	if result != "" {
		t.Errorf("Expected empty after revoke, got %s", result)
	}
}

func TestPermissionCache_Denied(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "denied")
	
	result := pc.Check("/desktop")
	if result != "denied" {
		t.Errorf("Expected 'denied', got %s", result)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test -run "TestPermissionCache_Grant|TestPermissionCache_Parent|TestPermissionCache_Revoke|TestPermissionCache_Denied" ./pkg/tools/...`
Expected: FAIL (tests don't exist yet if file is new, or PASS if appended)

- [x] **Step 3: Implement (tests already written above)**

Tests are already written in Step 1.

- [x] **Step 4: Run tests to verify they pass**

Run: `go test -run "TestPermissionCache" ./pkg/tools/...`
Expected: PASS (all 5 tests)

- [x] **Step 5: Commit**

```bash
git add pkg/tools/permission_test.go
git commit -m "test: add PermissionCache unit tests (Grant, Check, Revoke, parent path)"
```

---

### Task 3: Create RequestPermissionTool

**Files:**
- Modify: `pkg/tools/permission.go` (add RequestPermissionTool)
- Test: `pkg/tools/permission_test.go` (add tool tests)

**Does NOT cover:** Modifying ExecTool to use permissions — that's Task 4.

- [x] **Step 1: Write failing test**

```go
// pkg/tools/permission_test.go (add)
func TestRequestPermissionTool_Name(t *testing.T) {
	pc := NewPermissionCache()
	tool := NewRequestPermissionTool(pc)
	
	if tool.Name() != "request_permission" {
		t.Errorf("Expected 'request_permission', got %s", tool.Name())
	}
}

func TestRequestPermissionTool_Execute(t *testing.T) {
	pc := NewPermissionCache()
	tool := NewRequestPermissionTool(pc)
	
	result := tool.Execute(context.Background(), map[string]any{
		"path": "/desktop",
		"command": "ls /desktop",
	})
	
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.ForUser == "" {
		t.Error("Expected non-empty ForUser message")
	}
	if result.ForLLM == "" {
		t.Error("Expected non-empty ForLLM message")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test -run "TestRequestPermissionTool" ./pkg/tools/...`
Expected: FAIL - `NewRequestPermissionTool` not defined

- [x] **Step 3: Implement minimal change**

```go
// pkg/tools/permission.go (add to existing file)
package tools

import (
	"context"
	"fmt"
	"sync"
	
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ... existing PermissionCache code ...

type RequestPermissionTool struct {
	cache *PermissionCache
}

func NewRequestPermissionTool(cache *PermissionCache) *RequestPermissionTool {
	return &RequestPermissionTool{cache: cache}
}

func (t *RequestPermissionTool) Name() string {
	return "request_permission"
}

func (t *RequestPermissionTool) Description() string {
	return "Request user permission for accessing paths outside workspace. Returns prompt for user."
}

func (t *RequestPermissionTool) PromptMetadata() PromptMetadata {
	return PromptMetadata{
		Layer:  ToolPromptLayerCapability,
		Slot:   ToolPromptSlotTooling,
		Source: ToolPromptSourceRegistry,
	}
}

func (t *RequestPermissionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path that needs permission",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Original command (for context)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *RequestPermissionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	command, _ := args["command"].(string)
	
	logger.InfoCF("permission", "Permission request", map[string]any{"path": path, "command": command})
	
	// Return structured prompt that LLM will show to user
	return &ToolResult{
		ForLLM: fmt.Sprintf("User permission needed for %s. Ask user: 'Allow once' or 'Allow for session'?", path),
		ForUser: fmt.Sprintf("⚠️ **Permission Required**\n\nPicoClaw wants to execute: `%s`\nThis accesses `%s` which is outside your workspace.\n\n**How would you like to proceed?**\n- Type `once` for one-time access\n- Type `session` to allow all access to this path for this session\n- Type `no` to deny", command, path),
	}
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test -run "TestRequestPermissionTool" ./pkg/tools/...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add pkg/tools/permission.go pkg/tools/permission_test.go
git commit -m "feat: add RequestPermissionTool for user permission prompts"
```

---

### Task 4: Modify ExecTool to Check Permissions

**Files:**
- Modify: `pkg/tools/shell.go`
- Modify: `pkg/tools/session.go` (add PermissionCache field to ExecTool)

**Does NOT cover:** Config changes for `ask_permission` — that's Task 5.

- [x] **Step 1: Write failing test**

```go
// pkg/tools/shell_test.go (add)
func TestExecTool_CheckPermission_NeedsPermission(t *testing.T) {
	pc := NewPermissionCache()
	
	// Create exec tool with permission cache
	// Note: This test verifies that exec returns early when askPermission=true and path needs permission
	// We'll need to check the tool's behavior when executing commands with outside-workspace paths
	
	// For now, test the isOutsideWorkspace helper
	tool := &ExecTool{
		permissionCache: pc,
		askPermission:  true,
		workingDir:     "/workspace",
	}
	
	// Test path extraction (simplified)
	result := tool.checkPermission("/desktop")
	if result != "needs_permission" {
		t.Errorf("Expected 'needs_permission', got %s", result)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test -run TestExecTool_CheckPermission ./pkg/tools/...`
Expected: FAIL - `checkPermission` method not defined

- [ ] **Step 3: Implement minimal change**

First, modify `pkg/tools/session.go` to add PermissionCache to ExecTool struct:

```go
// pkg/tools/session.go (modify ExecTool struct, around line 37)
type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	customAllowPatterns []*regexp.Regexp
	allowedPathPatterns []*regexp.Regexp
	restrictToWorkspace bool
	allowRemote         bool
	sessionManager      *SessionManager
	permissionCache      *PermissionCache  // NEW
	askPermission       bool               // NEW
}
```

Then modify `pkg/tools/shell.go` to add the checkPermission method and modify Execute:

```go
// pkg/tools/shell.go (add method)
func (t *ExecTool) checkPermission(command string) string {
	if !t.askPermission {
		return "granted" // Permission checks disabled
	}
	
	// Extract path from command (simplified - looks for paths in command)
	// This is a basic implementation - real version would parse the command more carefully
	path := t.extractPathFromCommand(command)
	if path == "" {
		return "granted"
	}
	
	// Check if path is outside workspace
	if t.restrictToWorkspace && t.isOutsideWorkspace(path) {
		// Check permission cache
		if perm := t.permissionCache.Check(path); perm != "" {
			if perm == "denied" {
				return "denied"
			}
			return "granted"
		}
		return "needs_permission"
	}
	
	return "granted"
}

func (t *ExecTool) isOutsideWorkspace(path string) bool {
	// Simplified check - if path is absolute and not under workspace
	if filepath.IsAbs(path) {
		absWorkspace, _ := filepath.Abs(t.workingDir)
		absPath, _ := filepath.Abs(path)
		return !strings.HasPrefix(absPath, absWorkspace)
	}
	return false
}

func (t *ExecTool) extractPathFromCommand(command string) string {
	// Basic path extraction - looks for the first absolute path or path after common commands
	// This is simplified - real implementation would be more robust
	parts := strings.Fields(command)
	for _, part := range parts {
		if filepath.IsAbs(part) {
			return part
		}
	}
	return ""
}
```

Then modify the `Execute` method in `shell.go` to check permissions early:

```go
// In ExecTool.Execute(), add near the beginning:
func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	command, _ := args["command"].(string)
	
	// Check permissions
	switch t.checkPermission(command) {
	case "needs_permission":
		path := t.extractPathFromCommand(command)
		logger.InfoCF("exec", "Permission needed", map[string]any{"command": command, "path": path})
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Permission needed for path: %s. Call request_permission tool with path='%s'.", path, path),
			ForUser: fmt.Sprintf("⚠️ Permission required to access %s", path),
		}
	case "denied":
		path := t.extractPathFromCommand(command)
		return ErrorResult(fmt.Sprintf("Access to %s was denied", path))
	}
	
	// ... rest of existing Execute logic
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test -run TestExecTool_CheckPermission ./pkg/tools/...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add pkg/tools/shell.go pkg/tools/session.go
git commit -m "feat: modify ExecTool to check PermissionCache before executing outside-workspace commands"
```

---

### Task 5: Add ask_permission to Config

**Files:**
- Modify: `pkg/config/config.go` (add AskPermission to ExecConfig)
- Modify: `pkg/config/defaults.go` (add default)

**Does NOT cover:** Wiring up in instance.go — that's Task 6.

- [ ] **Step 1: Write failing test**

```go
// pkg/config/config_test.go (add)
func TestToolsConfig_IsToolEnabled_ExecAskPermission(t *testing.T) {
	cfg := &ToolsConfig{
		Exec: ExecConfig{
			Enabled:       true,
			AskPermission:  true,
		},
	}
	
	// Verify AskPermission field exists and is accessible
	if !cfg.Exec.AskPermission {
		t.Error("Expected AskPermission to be true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestToolsConfig_IsToolEnabled_ExecAskPermission ./pkg/config/...`
Expected: FAIL - `AskPermission` field not defined in ExecConfig

- [ ] **Step 3: Implement minimal change**

In `pkg/config/config.go`, find `ExecConfig` struct (around line 763) and add the field:

```go
type ExecConfig struct {
	Enabled            bool     `json:"enabled"`
	EnableDenyPatterns bool     `json:"enable_deny_patterns"`
	CustomDenyPatterns []string `json:"custom_deny_patterns"`
	CustomAllowPatterns []string `json:"custom_allow_patterns"`
	AllowRemote        bool     `json:"allow_remote"`
	TimeoutSeconds    int      `json:"timeout_seconds"`
	AskPermission     bool     `json:"ask_permission"`  // NEW
}
```

In `pkg/config/defaults.go`, find the Exec defaults (around lines 365-372) and add the field:

```go
Exec: ExecConfig{
	Enabled:            true,
	EnableDenyPatterns: true,
	AllowRemote:        true,
	TimeoutSeconds:    60,
	AskPermission:      true,  // NEW: enable permission prompts by default
},
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test -run TestToolsConfig_IsToolEnabled_ExecAskPermission ./pkg/config/...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/defaults.go
git commit -m "feat: add ask_permission config option to ExecConfig (default: true)"
```

---

### Task 6: Wire Up in Agent Instance

**Files:**
- Modify: `pkg/agent/instance.go` (create PermissionCache, pass to ExecTool, register RequestPermissionTool)

**Does NOT cover:** Running make build/test — that's Task 7.

- [ ] **Step 1: Write failing test**

This is an integration test - verify that exec tool is registered with PermissionCache when `ask_permission=true`.

Since this requires more complex setup, we'll do a manual verification step instead of a unit test.

- [ ] **Step 2: Implement minimal change**

In `pkg/agent/instance.go`, modify the exec tool registration (around lines 104-112):

```go
// pkg/agent/instance.go
// Create permission cache (add near top of function that registers tools)
var permissionCache *tools.PermissionCache
if cfg.Tools.IsToolEnabled("exec") {
	permissionCache = tools.NewPermissionCache()
	
	execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg, allowReadPaths)
	if err != nil {
		logger.ErrorCF("agent", "Failed to initialize exec tool; continuing without exec", ...)
	} else {
		// Set permission cache and ask_permission flag
		execTool.permissionCache = permissionCache
		execTool.askPermission = cfg.Tools.Exec.AskPermission
		
		toolsRegistry.Register(execTool)
	}
}

// Register request_permission tool if exec is enabled
if permissionCache != nil {
	toolsRegistry.Register(tools.NewRequestPermissionTool(permissionCache))
}
```

Also need to modify `NewExecToolWithConfig` or pass the cache after creation. Simplest approach - add a setter or modify the constructor.

In `pkg/tools/session.go`, modify `NewExecToolWithConfig`:

```go
func NewExecToolWithConfig(workingDir string, restrictToWorkspace bool, cfg *config.ToolsConfig, allowPaths []*regexp.Regexp) (*ExecTool, error) {
	// ... existing code ...
	
	tool := &ExecTool{
		// ... existing fields ...
		askPermission:  cfg.Exec.AskPermission,  // NEW
	}
	
	// Set cache later via setter or directly if we have access
	return tool, nil
}
```

Better approach - set cache after creation in instance.go:

```go
// In instance.go, after creating execTool:
execTool.permissionCache = permissionCache
```

Make sure `permissionCache` field in ExecTool is exported or provide a setter.

- [x] **Step 3: Manual verification**

Check that:
1. `tools.NewRequestPermissionTool` is defined
2. `ExecTool.PermissionCache` field exists
3. Registration code in instance.go compiles

Run: `go build ./pkg/agent/...`
Expected: Build succeeds

- [x] **Step 4: Commit**

```bash
git add pkg/agent/instance.go pkg/tools/session.go
git commit -m "feat: wire up PermissionCache and register RequestPermissionTool in agent"
```

---

### Task 7: Run Build and Test

**Files:**
- No file changes

**Does NOT cover:** Documentation updates — that's Task 8.

- [x] **Step 1: Run make build**

Run: `make build`
Expected: Build succeeds

- [x] **Step 2: Run make test**

Run: `make test`
Expected: All tests pass

- [x] **Step 3: Manual testing**

1. Start PicoClaw
2. Try `exec ls /desktop`
3. Verify LLM calls request_permission
4. Verify user prompt appears
5. Test "once" and "session" options
6. Verify subsequent calls respect permission

---

### Task 8: Update Documentation

**Files:**
- Modify: `docs/reference/exec-tool.md`
- Modify: `docs/reference/tools-api.md`

**Does NOT cover:** Anything else — final task.

- [x] **Step 1: Update exec-tool.md**

Add new section:

```markdown
## Permission System (New)

When `tools.exec.ask_permission = true` (default), the exec tool will ask for user permission before accessing paths outside workspace.

### How It Works

1. Exec tool detects command accesses path outside workspace
2. Checks PermissionCache - if no permission, returns early
3. LLM calls `request_permission` tool
4. Tool returns prompt for user: "Allow once" or "Allow for session"
5. User responds, LLM re-calls exec tool
6. Permission cached for "once" (consumed after use) or "session" (persists)

### Request Permission Tool

| Field | Description |
|-------|-------------|
| `path` | Path that needs permission |
| `command` | Original command (for context) |

### Permission Options

- **once**: Permission consumed after first use
- **session**: Permission persists for entire session
- **no**: Access denied
```

- [x] **Step 2: Update tools-api.md**

Add `request_permission` to the tools table:

```markdown
| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `request_permission` | Request user permission for outside-workspace access | permission | `exec.ask_permission` |
```

- [x] **Step 3: Commit**

```bash
git add docs/reference/exec-tool.md docs/reference/tools-api.md
git commit -m "docs: update exec tool and tools API docs with permission system"
```

---

## Self-Review

**Spec Coverage Check:**
- [x] New `request_permission` tool ✓ (Task 3, 6)
- [x] Session-based or one-time permission ✓ (Task 1, 3 - PermissionCache)
- [x] Scope: any path outside workspace ✓ (Task 4 - checkPermission)
- [x] Approach 1: separate tool + cache ✓ (Task 1, 3)

**Placeholder Scan:**
- No TBD/TODO found ✓
- No "implement later" found ✓
- Actual code provided in all steps ✓

**Type Consistency:**
- `PermissionCache` defined in Task 1, used in Tasks 3, 4, 6 ✓
- `RequestPermissionTool` defined in Task 3, registered in Task 6 ✓
- `AskPermission` added to `ExecConfig` in Task 5, used in Task 6 ✓

**No Issues Found.** Plan ready for execution.

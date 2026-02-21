# Workspace Permission System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow tools to access paths outside the workspace after asking the user for permission, with approvals cached per-directory for the session.

**Architecture:** A `PermissionFunc` callback abstraction that tools invoke when a path-outside-workspace is detected. CLI uses stdin prompts, Telegram uses inline keyboard buttons, other channels fall back to an LLM-driven confirmation pattern. Approved directories are cached in a session-scoped `PermissionStore` shared across all tools.

**Tech Stack:** Go, telego v1.6.0 (InlineKeyboardMarkup, HandleCallbackQuery, AnswerCallbackQuery)

---

### Task 1: PermissionStore and PermissionFunc types

**Files:**
- Create: `pkg/tools/permissions.go`
- Test: `pkg/tools/permissions_test.go`

**Step 1: Write the failing test**

```go
// pkg/tools/permissions_test.go
package tools

import (
	"testing"
)

func TestPermissionStore_ApproveAndCheck(t *testing.T) {
	store := NewPermissionStore()

	// Not approved initially
	if store.IsApproved("/Volumes/Code/myproject/src/main.go") {
		t.Error("expected path to not be approved initially")
	}

	// Approve a directory
	store.Approve("/Volumes/Code/myproject")

	// Path under approved directory should be allowed
	if !store.IsApproved("/Volumes/Code/myproject/src/main.go") {
		t.Error("expected path under approved directory to be approved")
	}

	// Exact directory should be allowed
	if !store.IsApproved("/Volumes/Code/myproject") {
		t.Error("expected exact approved directory to be approved")
	}

	// Sibling directory should NOT be allowed
	if store.IsApproved("/Volumes/Code/other") {
		t.Error("expected sibling directory to not be approved")
	}

	// Parent directory should NOT be allowed
	if store.IsApproved("/Volumes/Code") {
		t.Error("expected parent directory to not be approved")
	}
}

func TestPermissionStore_MultipleApprovals(t *testing.T) {
	store := NewPermissionStore()

	store.Approve("/Volumes/Code/project-a")
	store.Approve("/Volumes/Code/project-b")

	if !store.IsApproved("/Volumes/Code/project-a/file.go") {
		t.Error("expected project-a path to be approved")
	}
	if !store.IsApproved("/Volumes/Code/project-b/file.go") {
		t.Error("expected project-b path to be approved")
	}
	if store.IsApproved("/Volumes/Code/project-c/file.go") {
		t.Error("expected project-c path to not be approved")
	}
}

func TestPermissionStore_ConcurrentAccess(t *testing.T) {
	store := NewPermissionStore()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			store.Approve("/tmp/dir")
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		store.IsApproved("/tmp/dir/file")
	}

	<-done
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/ -run TestPermissionStore -v`
Expected: FAIL — `NewPermissionStore` not defined

**Step 3: Write the implementation**

```go
// pkg/tools/permissions.go
package tools

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
)

// PermissionFunc asks the user for permission to access a directory outside the workspace.
// Returns true if approved, false if denied. Implementations should block until the user responds.
// The path argument is the directory being requested access to.
type PermissionFunc func(ctx context.Context, path string) (bool, error)

// PermissionStore tracks approved directories for a session.
// It is safe for concurrent use.
type PermissionStore struct {
	mu       sync.RWMutex
	approved map[string]struct{}
}

// NewPermissionStore creates a new empty PermissionStore.
func NewPermissionStore() *PermissionStore {
	return &PermissionStore{
		approved: make(map[string]struct{}),
	}
}

// Approve adds a directory to the approved set.
func (ps *PermissionStore) Approve(dir string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.approved[filepath.Clean(dir)] = struct{}{}
}

// IsApproved checks if a path falls under any approved directory.
func (ps *PermissionStore) IsApproved(path string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	cleanPath := filepath.Clean(path)
	for dir := range ps.approved {
		if cleanPath == dir || strings.HasPrefix(cleanPath, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools/ -run TestPermissionStore -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tools/permissions.go pkg/tools/permissions_test.go
git commit -m "feat: add PermissionStore and PermissionFunc types for workspace access control"
```

---

### Task 2: PermissibleTool interface and wire into filesystem tools

**Files:**
- Modify: `pkg/tools/permissions.go` (add interface)
- Modify: `pkg/tools/filesystem.go:12-60` (validatePath), `pkg/tools/filesystem.go:80-247` (tool structs)
- Modify: `pkg/tools/edit.go:10-23` (EditFileTool, AppendFileTool structs)
- Test: `pkg/tools/filesystem_test.go` (or existing test file)

**Step 1: Write the failing test**

```go
// In pkg/tools/permissions_test.go, add:

func TestValidatePath_WithPermission(t *testing.T) {
	workspace := t.TempDir()
	outsideDir := t.TempDir()

	// Without permission, path outside workspace is denied
	_, err := validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "file.txt"), workspace, true, nil, nil)
	if err == nil {
		t.Error("expected error for path outside workspace without permission")
	}

	// With a PermissionFunc that approves, path outside workspace is allowed
	store := NewPermissionStore()
	approveAll := func(ctx context.Context, path string) (bool, error) {
		return true, nil
	}
	result, err := validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "file.txt"), workspace, true, store, approveAll)
	if err != nil {
		t.Errorf("expected no error with approving PermissionFunc, got: %v", err)
	}
	if result != filepath.Join(outsideDir, "file.txt") {
		t.Errorf("expected resolved path %q, got %q", filepath.Join(outsideDir, "file.txt"), result)
	}

	// After approval, the store should have the directory cached
	if !store.IsApproved(outsideDir) {
		t.Error("expected directory to be cached in store after approval")
	}

	// With a PermissionFunc that denies, path outside workspace is denied
	store2 := NewPermissionStore()
	denyAll := func(ctx context.Context, path string) (bool, error) {
		return false, nil
	}
	_, err = validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "file.txt"), workspace, true, store2, denyAll)
	if err == nil {
		t.Error("expected error for path outside workspace with denying PermissionFunc")
	}

	// With nil PermissionFunc, should return descriptive error for LLM fallback
	_, err = validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "file.txt"), workspace, true, nil, nil)
	if err == nil {
		t.Error("expected error with nil PermissionFunc")
	}
	if !strings.Contains(err.Error(), "outside the workspace") {
		t.Errorf("expected descriptive error message, got: %v", err)
	}
}

func TestValidatePath_WithPermission_CachedApproval(t *testing.T) {
	workspace := t.TempDir()
	outsideDir := t.TempDir()

	store := NewPermissionStore()
	callCount := 0
	countingApprover := func(ctx context.Context, path string) (bool, error) {
		callCount++
		return true, nil
	}

	// First call should invoke PermissionFunc
	_, err := validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "a.txt"), workspace, true, store, countingApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected PermissionFunc called once, got %d", callCount)
	}

	// Second call to same directory should use cache, not call PermissionFunc again
	_, err = validatePathWithPermission(context.Background(), filepath.Join(outsideDir, "b.txt"), workspace, true, store, countingApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected PermissionFunc still called once (cached), got %d", callCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/ -run TestValidatePath_WithPermission -v`
Expected: FAIL — `validatePathWithPermission` not defined

**Step 3: Add PermissibleTool interface to permissions.go**

```go
// Add to pkg/tools/permissions.go:

// PermissibleTool is an optional interface that tools can implement
// to support permission-based access to paths outside the workspace.
type PermissibleTool interface {
	Tool
	SetPermission(store *PermissionStore, fn PermissionFunc)
}
```

**Step 4: Create `validatePathWithPermission` in filesystem.go**

Add this new function alongside the existing `validatePath`. The existing `validatePath` remains unchanged for backward compatibility.

```go
// Add to pkg/tools/filesystem.go:

// validatePathWithPermission extends validatePath with interactive permission support.
// When a path is outside the workspace and restrict is true:
//   - If the directory is already approved in store, allow access
//   - If permFn is non-nil, call it to ask the user; cache approval in store
//   - If permFn is nil, return a descriptive error (LLM-driven fallback)
func validatePathWithPermission(ctx context.Context, path, workspace string, restrict bool, store *PermissionStore, permFn PermissionFunc) (string, error) {
	if workspace == "" {
		return path, nil
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath, err = filepath.Abs(filepath.Join(absWorkspace, path))
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
	}

	if restrict && !isWithinWorkspace(absPath, absWorkspace) {
		dir := filepath.Dir(absPath)

		// Check cached approvals
		if store != nil && store.IsApproved(dir) {
			return absPath, nil
		}

		// Ask user for permission
		if permFn != nil {
			approved, err := permFn(ctx, dir)
			if err != nil {
				return "", fmt.Errorf("permission request failed: %w", err)
			}
			if !approved {
				return "", fmt.Errorf("access denied: user denied access to %s", dir)
			}
			if store != nil {
				store.Approve(dir)
			}
			return absPath, nil
		}

		// Fallback: descriptive error for LLM-driven flow
		return "", fmt.Errorf("access denied: path %s is outside the workspace. Ask the user for permission to access directory %s, then retry", absPath, dir)
	}

	// Within workspace — do existing symlink checks
	if restrict {
		workspaceReal := absWorkspace
		if resolved, err := filepath.EvalSymlinks(absWorkspace); err == nil {
			workspaceReal = resolved
		}

		if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
			if !isWithinWorkspace(resolved, workspaceReal) {
				return "", fmt.Errorf("access denied: symlink resolves outside workspace")
			}
		} else if os.IsNotExist(err) {
			if parentResolved, err := resolveExistingAncestor(filepath.Dir(absPath)); err == nil {
				if !isWithinWorkspace(parentResolved, workspaceReal) {
					return "", fmt.Errorf("access denied: symlink resolves outside workspace")
				}
			} else if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to resolve path: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	return absPath, nil
}
```

**Step 5: Add permission fields to filesystem tool structs and wire them in**

Add `permStore *PermissionStore` and `permFn PermissionFunc` fields to `ReadFileTool`, `WriteFileTool`, `ListDirTool`, `EditFileTool`, `AppendFileTool`. Implement `SetPermission` on each. Update their `Execute` methods to call `validatePathWithPermission` instead of `validatePath`.

For `ReadFileTool` (example pattern — repeat for all five tools):

```go
type ReadFileTool struct {
	workspace string
	restrict  bool
	permStore *PermissionStore
	permFn    PermissionFunc
}

func (t *ReadFileTool) SetPermission(store *PermissionStore, fn PermissionFunc) {
	t.permStore = store
	t.permFn = fn
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	resolvedPath, err := validatePathWithPermission(ctx, path, t.workspace, t.restrict, t.permStore, t.permFn)
	if err != nil {
		return ErrorResult(err.Error())
	}
	// ... rest unchanged
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./pkg/tools/ -run TestValidatePath_WithPermission -v`
Expected: PASS

**Step 7: Run all tool tests**

Run: `go test ./pkg/tools/ -v`
Expected: All PASS

**Step 8: Commit**

```bash
git add pkg/tools/permissions.go pkg/tools/permissions_test.go pkg/tools/filesystem.go pkg/tools/edit.go
git commit -m "feat: add permission-based path validation for filesystem tools"
```

---

### Task 3: Wire permission into ExecTool

**Files:**
- Modify: `pkg/tools/shell.go:19-25` (ExecTool struct), `pkg/tools/shell.go:252-306` (guardCommand)
- Test: `pkg/tools/shell_test.go`

**Step 1: Write the failing test**

```go
// Add to pkg/tools/shell_test.go:

func TestGuardCommand_WithPermission(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	// Without permission, absolute path outside workspace is blocked
	result := tool.guardCommand("ls /Volumes/Code/other", workspace)
	if result == "" {
		t.Error("expected block for path outside workspace without permission")
	}

	// Set up permission that approves
	store := NewPermissionStore()
	tool.SetPermission(store, func(ctx context.Context, path string) (bool, error) {
		return true, nil
	})

	// With permission, absolute path outside workspace is allowed
	result = tool.guardCommandWithPermission(context.Background(), "ls /Volumes/Code/other", workspace)
	if result != "" {
		t.Errorf("expected no block with permission, got: %s", result)
	}

	// Path traversal still blocked (security — no bypassing ../  check)
	result = tool.guardCommandWithPermission(context.Background(), "ls ../../etc/passwd", workspace)
	if result == "" {
		t.Error("expected path traversal to still be blocked")
	}

	// Deny patterns still blocked
	result = tool.guardCommandWithPermission(context.Background(), "rm -rf /", workspace)
	if result == "" {
		t.Error("expected deny pattern to still be blocked")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/ -run TestGuardCommand_WithPermission -v`
Expected: FAIL

**Step 3: Add permission fields and guardCommandWithPermission to ExecTool**

```go
// Modify ExecTool struct in shell.go:
type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
	permStore           *PermissionStore
	permFn              PermissionFunc
}

func (t *ExecTool) SetPermission(store *PermissionStore, fn PermissionFunc) {
	t.permStore = store
	t.permFn = fn
}
```

Add `guardCommandWithPermission` that extends `guardCommand`:
- Deny patterns and allowlist checks remain unchanged (no bypass)
- Path traversal `../` check remains unchanged (no bypass)
- For the absolute-path-outside-workspace check: instead of immediately returning a block, check the PermissionStore and call PermissionFunc

Update `Execute` to call `guardCommandWithPermission` instead of `guardCommand`.

**Step 4: Run tests**

Run: `go test ./pkg/tools/ -run TestGuardCommand -v`
Expected: All PASS (both old and new tests)

**Step 5: Commit**

```bash
git add pkg/tools/shell.go pkg/tools/shell_test.go
git commit -m "feat: add permission-based path access to exec tool"
```

---

### Task 4: Wire PermissionStore into agent instance and tool registration

**Files:**
- Modify: `pkg/agent/instance.go:37-56` (NewAgentInstance)
- Modify: `pkg/agent/loop.go:704-722` (updateToolContexts)

**Step 1: Write the failing test**

```go
// This is an integration-level check. Add to an existing agent test file or create one.
// Verify that tools registered in an agent instance implement PermissibleTool and can
// have SetPermission called on them.

// In pkg/agent/instance_test.go or similar:
func TestAgentInstance_ToolsImplementPermissibleTool(t *testing.T) {
	// Create a minimal agent instance
	defaults := &config.AgentDefaults{
		Model:              "test-model",
		Workspace:          t.TempDir(),
		RestrictToWorkspace: true,
	}
	cfg := config.DefaultConfig()
	instance := NewAgentInstance(nil, defaults, cfg, nil)

	permissibleTools := []string{"read_file", "write_file", "list_dir", "edit_file", "append_file", "exec"}
	for _, name := range permissibleTools {
		tool, ok := instance.Tools.Get(name)
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if _, ok := tool.(tools.PermissibleTool); !ok {
			t.Errorf("tool %q does not implement PermissibleTool", name)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/agent/ -run TestAgentInstance_ToolsImplementPermissibleTool -v`
Expected: FAIL

**Step 3: Create PermissionStore per agent instance and wire into tools**

In `NewAgentInstance`, after registering tools:

```go
// Wire permission system into restricted tools
permStore := tools.NewPermissionStore()
for _, name := range instance.Tools.List() {
	if tool, ok := instance.Tools.Get(name); ok {
		if pt, ok := tool.(tools.PermissibleTool); ok {
			pt.SetPermission(permStore, nil) // PermissionFunc set per-request in updateToolContexts
		}
	}
}
```

Store the `permStore` on the `AgentInstance` struct so `updateToolContexts` can use it later.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/agent/ -run TestAgentInstance_ToolsImplementPermissibleTool -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/agent/instance.go
git commit -m "feat: wire PermissionStore into agent instance tools"
```

---

### Task 5: CLI permission prompt

**Files:**
- Create: `pkg/tools/permission_cli.go`
- Test: `pkg/tools/permission_cli_test.go`
- Modify: `cmd/picoclaw/cmd_agent.go` (wire PermissionFunc into agent loop)
- Modify: `pkg/agent/loop.go` (add SetPermissionFunc method or option)

**Step 1: Write the failing test**

```go
// pkg/tools/permission_cli_test.go
package tools

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIPermissionFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
	}{
		{name: "approve_y", input: "y\n", wantOK: true},
		{name: "approve_yes", input: "yes\n", wantOK: true},
		{name: "approve_Y", input: "Y\n", wantOK: true},
		{name: "deny_n", input: "n\n", wantOK: false},
		{name: "deny_empty", input: "\n", wantOK: false},
		{name: "deny_other", input: "maybe\n", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			var output bytes.Buffer
			fn := NewCLIPermissionFunc(reader, &output)

			got, err := fn(context.Background(), "/Volumes/Code/project")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantOK {
				t.Errorf("got %v, want %v", got, tt.wantOK)
			}
			// Output should contain the path
			if !strings.Contains(output.String(), "/Volumes/Code/project") {
				t.Errorf("output should mention the path, got: %s", output.String())
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/ -run TestCLIPermissionFunc -v`
Expected: FAIL

**Step 3: Implement CLIPermissionFunc**

```go
// pkg/tools/permission_cli.go
package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// NewCLIPermissionFunc creates a PermissionFunc that prompts the user on a terminal.
// reader is the stdin source, writer is stdout for the prompt.
func NewCLIPermissionFunc(reader io.Reader, writer io.Writer) PermissionFunc {
	return func(ctx context.Context, path string) (bool, error) {
		fmt.Fprintf(writer, "\n⚠ Agent wants to access: %s\nAllow access to this directory? [y/N]: ", path)

		scanner := bufio.NewScanner(reader)
		if !scanner.Scan() {
			return false, scanner.Err()
		}

		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes", nil
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools/ -run TestCLIPermissionFunc -v`
Expected: PASS

**Step 5: Wire into the agent loop**

Add a `SetPermissionFunc` method to `AgentLoop` that stores a `PermissionFunc`. In `updateToolContexts` (loop.go:704), also set the PermissionFunc on PermissibleTools.

In `cmd_agent.go`, after creating the agent loop:

```go
agentLoop.SetPermissionFunc(tools.NewCLIPermissionFunc(os.Stdin, os.Stdout))
```

**Step 6: Run CLI manually to test**

Run: `go build -tags stdjson -o /tmp/picoclaw ./cmd/picoclaw/`
Then: `/tmp/picoclaw agent -m "list files in /tmp"`
Expected: Should prompt for permission since /tmp is outside workspace.

**Step 7: Commit**

```bash
git add pkg/tools/permission_cli.go pkg/tools/permission_cli_test.go cmd/picoclaw/cmd_agent.go pkg/agent/loop.go
git commit -m "feat: add CLI permission prompt for outside-workspace access"
```

---

### Task 6: Telegram inline keyboard permission

**Files:**
- Create: `pkg/channels/telegram_permissions.go`
- Modify: `pkg/channels/telegram.go:94-139` (Start — add HandleCallbackQuery)
- Modify: `pkg/channels/telegram.go:149-194` (Send — support structured OutboundMessage)
- Modify: `pkg/bus/types.go:13-17` (OutboundMessage — add Metadata field)
- Modify: `pkg/channels/manager.go` (pass permission callback to agent loop)

**Step 1: Extend OutboundMessage with Metadata**

```go
// pkg/bus/types.go — add Metadata to OutboundMessage:
type OutboundMessage struct {
	Channel  string            `json:"channel"`
	ChatID   string            `json:"chat_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
```

`Metadata` carries structured data like `{"type": "permission_request", "callback_id": "uuid", "path": "/Volumes/Code/project"}`.

**Step 2: Create TelegramPermissionManager**

```go
// pkg/channels/telegram_permissions.go
package channels

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// TelegramPermissionManager handles inline keyboard permission prompts.
type TelegramPermissionManager struct {
	bot     *telego.Bot
	pending sync.Map // callbackID -> chan bool
}

func NewTelegramPermissionManager(bot *telego.Bot) *TelegramPermissionManager {
	return &TelegramPermissionManager{bot: bot}
}

// AskPermission sends an inline keyboard prompt and blocks until the user responds.
func (pm *TelegramPermissionManager) AskPermission(ctx context.Context, chatID int64, path string) (bool, error) {
	callbackID := uuid.New().String()[:8]

	resultCh := make(chan bool, 1)
	pm.pending.Store(callbackID, resultCh)
	defer pm.pending.Delete(callbackID)

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			telego.InlineKeyboardButton{Text: "✅ Allow", CallbackData: "perm_allow_" + callbackID},
			telego.InlineKeyboardButton{Text: "❌ Deny", CallbackData: "perm_deny_" + callbackID},
		),
	)

	msg := tu.Message(tu.ID(chatID), fmt.Sprintf("⚠️ Agent wants to access:\n<code>%s</code>\n\nAllow access to this directory?", path))
	msg.ParseMode = telego.ModeHTML
	msg.ReplyMarkup = keyboard

	if _, err := pm.bot.SendMessage(ctx, msg); err != nil {
		return false, fmt.Errorf("sending permission prompt: %w", err)
	}

	select {
	case approved := <-resultCh:
		return approved, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// HandleCallback processes a callback query from an inline keyboard button press.
func (pm *TelegramPermissionManager) HandleCallback(ctx context.Context, query telego.CallbackQuery) error {
	data := query.Data
	var callbackID string
	var approved bool

	if len(data) > len("perm_allow_") && data[:len("perm_allow_")] == "perm_allow_" {
		callbackID = data[len("perm_allow_"):]
		approved = true
	} else if len(data) > len("perm_deny_") && data[:len("perm_deny_")] == "perm_deny_" {
		callbackID = data[len("perm_deny_"):]
		approved = false
	} else {
		return nil // Not a permission callback
	}

	if ch, ok := pm.pending.Load(callbackID); ok {
		ch.(chan bool) <- approved
	}

	label := "Denied"
	if approved {
		label = "Allowed"
	}
	_ = pm.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            label,
	})

	return nil
}

// NewTelegramPermissionFunc creates a PermissionFunc backed by Telegram inline keyboards.
func (pm *TelegramPermissionManager) NewPermissionFunc(chatID int64) func(ctx context.Context, path string) (bool, error) {
	return func(ctx context.Context, path string) (bool, error) {
		return pm.AskPermission(ctx, chatID, path)
	}
}
```

**Step 3: Register HandleCallbackQuery in Telegram Start()**

In `telegram.go` `Start()`, after the existing `bh.HandleMessage` registrations, add:

```go
// Permission callback handler
bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
	return c.permissionManager.HandleCallback(ctx, query)
}, th.CallbackDataContains("perm_"))
```

Add `permissionManager *TelegramPermissionManager` to the `TelegramChannel` struct, initialized in `NewTelegramChannel`.

**Step 4: Wire Telegram PermissionFunc into agent loop**

This is the trickiest part. The PermissionFunc for Telegram needs the chatID, which is only known at message-processing time. The channel manager needs to pass a factory function to the agent loop that creates PermissionFuncs for a given channel+chatID pair.

Add a `PermissionFuncFactory` type:

```go
// pkg/tools/permissions.go
type PermissionFuncFactory func(channel, chatID string) PermissionFunc
```

The agent loop stores this factory. In `updateToolContexts`, it creates the appropriate PermissionFunc for the current channel/chatID and calls `SetPermission` on all PermissibleTools.

The channel manager registers a factory that:
- For `"cli"` channel: returns `NewCLIPermissionFunc(os.Stdin, os.Stdout)`
- For `"telegram"` channel: returns `pm.NewPermissionFunc(chatID)` from the TelegramPermissionManager
- For other channels: returns `nil` (LLM-driven fallback)

**Step 5: Run Telegram tests**

Run: `go test ./pkg/channels/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/channels/telegram_permissions.go pkg/channels/telegram.go pkg/bus/types.go pkg/tools/permissions.go pkg/agent/loop.go
git commit -m "feat: add Telegram inline keyboard permission prompts"
```

---

### Task 7: Integration testing and cleanup

**Files:**
- Modify: `pkg/agent/loop.go` (updateToolContexts wiring)
- Run: full test suite

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: All PASS (except pre-existing cmd/picoclaw embed issue)

**Step 2: Run go vet and format**

Run: `go vet ./... && go fmt ./...`
Expected: Clean

**Step 3: Manual end-to-end test**

Build and test the CLI flow:
```bash
cp -r workspace cmd/picoclaw/workspace && go build -tags stdjson -o /tmp/picoclaw ./cmd/picoclaw/ && rm -rf cmd/picoclaw/workspace
/tmp/picoclaw agent -m "read the file /tmp/test.txt"
```
Expected: Prompts for permission, allows/denies based on user input.

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete workspace permission system with CLI and Telegram support"
```

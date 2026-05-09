# Design: Exec Tool Permission System

**Date**: 2026-05-05
**Status**: Pending User Approval
**Author**: AI Assistant

## Scope

**In-Scope**:
- Allow exec tool to work on ANY folder (not just workspace)
- Add permission system that prompts user before accessing paths outside workspace
- Permission can be "once" or "for session" (user choice)
- New dedicated `request_permission` tool

**Non-Goals**:
- Changing how exec tool executes commands (still uses same shell/pty)
- Modifying non-exec tools (read_file, write_file, etc.)
- Adding GUI dialogs (permission is via chat interface) and web.

## Architecture

### System Flow

```
User: "List directories in /desktop"
    ↓
LLM calls exec tool with command "ls /desktop"
    ↓
Exec tool detects /desktop is outside workspace
    ↓
Exec tool returns: {"permission_needed": true, "path": "/desktop", "message": "Permission required..."}
    ↓
LLM sees permission_needed = true
    ↓
LLM calls request_permission tool: {path: "/desktop", command: "ls /desktop"}
    ↓
request_permission tool prompts user: "Allow access to /desktop? [Once] [Session] [Deny]"
    ↓
User selects "Session"
    ↓
Permission cached in PermissionCache: {"/desktop": "session"}
    ↓
LLM re-calls exec tool with same command
    ↓
Exec tool checks PermissionCache, sees "/desktop" → "session"
    ↓
Exec tool proceeds with command execution
    ↓
Returns output to LLM
```

### Components

| Component | File | Purpose |
|-----------|------|---------|
| `PermissionCache` | `pkg/tools/permission.go` (new) | Stores per-session permissions |
| `RequestPermissionTool` | `pkg/tools/permission.go` (new) | Tool that prompts user for permission |
| `ExecTool` (modified) | `pkg/tools/shell.go` | Check PermissionCache before executing |
| `ToolsConfig` (modified) | `pkg/config/config.go` | Add `ask_permission` config option |
| `SessionManager` (modified) | `pkg/tools/session.go` | Store PermissionCache per session |

## Design Options

### Approach 1: New `request_permission` Tool + Session Cache (Recommended)

**Description**:
- Add new `RequestPermissionTool` that LLM calls when exec returns `permission_needed`
- Tool prompts user with options: "Allow once", "Allow for session", "Deny"
- Permission stored in `PermissionCache` (map[string]string: path → "once"|"session")
- Exec tool checks cache before executing commands with outside-workspace paths

**Pros**:
- Clean separation of concerns (permission logic separate from exec)
- Matches user's answer: "New dedicated tool"
- Flexible: supports one-time and session-based permissions
- Cache is session-based, no persistence across restarts
- "once" permission is auto-consumed after the first successful exec call using that path

**Cons**:
- LLM needs two tool calls (exec → request_permission → exec again)
- Slightly more complex flow

**Implementation**:
```go
// pkg/tools/permission.go (new file)

type PermissionCache struct {
    mu    sync.RWMutex
    perms map[string]string // path → "once"|"session"
}

func (pc *PermissionCache) Check(path string) string {
    pc.mu.RLock()
    defer pc.mu.RUnlock()
    return pc.perms[path]
}

func (pc *PermissionCache) Grant(path, duration string) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    pc.perms[path] = duration
}

type RequestPermissionTool struct {
    cache *PermissionCache
}

func (t *RequestPermissionTool) Name() string { return "request_permission" }

func (t *RequestPermissionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    path, _ := args["path"].(string)
    // This tool triggers a user prompt in the chat interface
    // The LLM will see the response and ask the user
    return &ToolResult{
        ForLLM: fmt.Sprintf("Permission needed for %s. Ask user: 'Allow once' or 'Allow for session'?", path),
        ForUser: fmt.Sprintf("⚠️ **Permission Required**\n\nPicoClaw wants to access `%s` which is outside your workspace.\n\n**Allow access?**\n- Type 'once' for one-time access\n- Type 'session' for this session\n- Type 'no' to deny", path),
    }
}
```

### Approach 2: Integrate Permission into Exec Tool

**Description**:
- Exec tool, when detecting outside-workspace access, returns early with `permission_needed` flag
- LLM asks user, then re-calls exec with a `confirmed=true` parameter
- No separate tool, but exec tool becomes more complex

**Pros**:
- Simpler flow (one tool instead of two)
- Fewer LLM calls

**Cons**:
- exec tool does two things: execute commands AND handle permissions
- LLM still needs to re-call exec after user confirms
- Less flexible for future permission types

**Rejected**: User explicitly asked for "new dedicated tool"

### Approach 3: Config-Based Whitelist Only

**Description**:
- Add `tools.exec.allowed_paths` config array
- Paths in whitelist don't need permission
- No runtime prompting

**Pros**:
- Simple implementation
- No extra LLM calls

**Cons**:
- Static - can't ask user dynamically
- Requires pre-configuration
- Doesn't match user's requirement of "asking for permission"

**Rejected**: Doesn't satisfy "ask for permission" requirement.

## Recommended Approach: Approach 1

**Rationale**:
1. Matches user's explicit answer: "New dedicated tool"
2. Clean separation of concerns
3. Supports both "once" and "session" permission durations
4. Follows existing patterns in codebase (tools are separate, registered in registry)

## Data Structures

### PermissionCache

```go
// pkg/tools/permission.go

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
        if val == "session" && strings.HasPrefix(path, cachedPath) {
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

### Modified ExecTool

```go
// In pkg/tools/shell.go

type ExecTool struct {
    // ... existing fields ...
    permissionCache *PermissionCache  // NEW
    askPermission  bool               // NEW: from config tools.exec.ask_permission
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    command, _ := args["command"].(string)
    
    // Check if command accesses paths outside workspace
    if t.askPermission && t.isOutsideWorkspace(command) {
        path := t.extractPath(command)
        
        // Check permission cache
        if perm := t.permissionCache.Check(path); perm != "" {
            if perm == "denied" {
                return ErrorResult(fmt.Sprintf("Access to %s was denied", path))
            }
            // permission granted, proceed with execution
        } else {
            // No permission - return early, LLM will call request_permission
            return &ToolResult{
                ForLLM: fmt.Sprintf("Permission needed for path: %s. Call request_permission tool.", path),
                ForUser: fmt.Sprintf("⚠️ Permission required to access %s", path),
            }
        }
    }
    
    // ... existing execution logic ...
}
```

### RequestPermissionTool

```go
// pkg/tools/permission.go

type RequestPermissionTool struct {
    cache *PermissionCache
}

func (t *RequestPermissionTool) Name() string { return "request_permission" }

func (t *RequestPermissionTool) Description() string {
    return "Request user permission for accessing paths outside workspace. Returns prompt for user."
}

func (t *RequestPermissionTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "path":    map[string]any{"type": "string", "description": "Path that needs permission"},
            "command": map[string]any{"type": "string", "description": "Original command (for context)"},
        },
        "required": []string{"path"},
    }
}

func (t *RequestPermissionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    path, _ := args["path"].(string)
    command, _ := args["command"].(string)
    
    // Return structured prompt that LLM will show to user
    return &ToolResult{
        ForLLM: fmt.Sprintf("User permission needed for %s. Ask user: 'Allow once' or 'Allow for session'?", path),
        ForUser: fmt.Sprintf("⚠️ **Permission Required**\n\nPicoClaw wants to execute: `%s`\nThis accesses `%s` which is outside your workspace.\n\n**How would you like to proceed?**\n- Type `once` for one-time access\n- Type `session` to allow all access to this path for this session\n- Type `no` to deny", command, path),
    }
}
```

### Config Changes

```go
// pkg/config/config.go

type ExecConfig struct {
    Enabled            bool     `json:"enabled"`
    EnableDenyPatterns bool     `json:"enable_deny_patterns"`
    CustomDenyPatterns []string `json:"custom_deny_patterns"`
    AllowRemote        bool     `json:"allow_remote"`
    TimeoutSeconds    int      `json:"timeout_seconds"`
    AskPermission     bool     `json:"ask_permission"`  // NEW
}

// Default in pkg/config/defaults.go
Exec: ExecConfig{
    Enabled:            true,
    EnableDenyPatterns: true,
    AllowRemote:        true,
    TimeoutSeconds:    60,
    AskPermission:      true,  // NEW: enable permission prompts by default
}
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| User denies permission | Return error, do not execute command |
| User says "once", then re-calls | Check: "once" permission is consumed after one use |
| User says "session" | Cache permission for duration of session |
| Cache full (unlikely) | Deny with error: "Too many permission entries" |
| Invalid path in permission | Ignore, treat as no permission |

## Testing Strategy

1. **Unit Tests** (`pkg/tools/permission_test.go`):
   - Test PermissionCache.Check() with exact and parent path matches
   - Test Grant/Revoke cycle
   - Test "once" permission consumed after one use

2. **Integration Tests** (`pkg/tools/shell_test.go`):
   - Test exec tool returns early when `ask_permission=true` and path outside workspace
   - Test exec proceeds when permission in cache
   - Test exec denied when permission is "denied"

3. **Manual Testing**:
   - Start PicoClaw, try `exec ls /desktop`
   - Verify LLM calls request_permission
   - Verify user prompt appears
   - Test "once" and "session" options
   - Verify subsequent calls respect permission

## Migration Notes

- **Backward Compatibility**: `ask_permission` defaults to `true`, so existing users get the new behavior
- **Config Upgrade**: No manual migration needed - new field with safe default
- **No Breaking Changes**: Exec tool still works the same way when `ask_permission=false`

## Failure Mode Check

### Failure Mode 1: LLM doesn't call request_permission after exec returns early
**Severity**: Critical
**Scenario**: Exec returns "permission needed", but LLM doesn't call request_permission tool
**Impact**: User sees confusing message, command doesn't execute
**Mitigation**: 
- Ensure exec returns clear instructions in `ForLLM` field: "Call request_permission tool with path=X"
- Add examples in tool description for LLM
- Test with multiple LLM providers

### Failure Mode 2: Permission cache never cleaned up
**Severity**: Minor
**Scenario**: Session has thousands of cached permissions
**Impact**: Memory growth (very slow)
**Mitigation**:
- PermissionCache is per-session, cleaned up when session ends
- "once" permissions could be auto-consumed
- For now, assume sessions are short-lived (no cleanup needed)

### Failure Mode 3: User says "session" but path is /desktop/subfolder
**Severity**: Minor
**Scenario**: Permission granted for /desktop, but command accesses /desktop/subfolder
**Impact**: Need to check parent paths in PermissionCache.Check()
**Mitigation**: Implemented in Check() - scans for parent paths with "session" permission

## Files to Modify

| File | Change |
|------|--------|
| `pkg/tools/permission.go` | NEW: PermissionCache + RequestPermissionTool |
| `pkg/tools/permission_test.go` | NEW: Unit tests |
| `pkg/tools/shell.go` | Modify ExecTool to check permissions |
| `pkg/tools/registry.go` | Register RequestPermissionTool |
| `pkg/config/config.go` | Add AskPermission to ExecConfig |
| `pkg/config/defaults.go` | Add default for AskPermission |
| `pkg/agent/instance.go` | Wire up PermissionCache, register tool |

## User Review Gate

Spec written to `docs/superpowers-optimized/specs/2026-05-05-exec-permission-design.md`.

Please review it and let me know if you want any changes before I write the implementation plan.

**Key Decisions to Confirm**:
1. New `request_permission` tool (as per your answer)
2. Session-based or one-time permission (user chooses at runtime)
3. Scope: any path outside workspace (as per your answer)
4. Recommended approach: Approach 1 (separate tool + cache)

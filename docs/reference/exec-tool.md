# Exec Tool - How It Works & Sandboxing

## Overview

The `exec` tool allows the AI agent to execute shell commands on the host system. It supports both synchronous execution and background sessions with PTY support.

## How Exec Works

### Registration Flow

The exec tool is registered in `pkg/agent/instance.go` (lines 104-112):

```go
if cfg.Tools.IsToolEnabled("exec") {
    execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg, allowReadPaths)
    if err != nil {
        logger.ErrorCF("agent", "Failed to initialize exec tool; continuing without exec", ...)
    } else {
        toolsRegistry.Register(execTool)
    }
}
```

**Prerequisites**: The exec tool only registers when `tools.exec.enabled = true` in config.

### LLM Invocation

The LLM calls the `exec` tool with JSON arguments:

```json
{
  "action": "run",           // run, list, poll, read, write, kill, send-keys
  "command": "ls -la",      // shell command to execute
  "workdir": "/workspace", // optional, defaults to workspace
  "background": false,       // run in background (returns sessionId)
  "pty": false              // use pseudo-terminal (Unix only)
}
```

### Execution Flow

1. **Action Routing** (`pkg/tools/shell.go`):
   - `run` → `runSync()` or `runBackground()`
   - `list` → list active sessions
   - `poll` → check background process status
   - `read` → read output from background session
   - `write` → write input to background session (PTY mode)
   - `kill` → terminate background session
   - `send-keys` → send key sequence to PTY session

2. **Synchronous Execution** (`runSync()`):
   - Command parsed and validated against deny patterns
   - `exec.CommandContext()` creates the command
   - Isolation applied via `isolation.Start(cmd)` (bubblewrap on Linux)
   - stdout/stderr captured (max 1MB output)
   - Waits for completion or timeout (default 60s)

3. **Background Execution** (`runBackground()`):
   - Starts process in background
   - Creates `ProcessSession` tracked by global `SessionManager`
   - Returns `sessionId` for later management
   - Sessions auto-cleanup 30 minutes after process exits

4. **Isolation Wrapper** (`pkg/isolation/`):
   - **Linux**: bubblewrap sandbox (namespace isolation, readonly filesystem)
   - **Windows**: Restricted tokens (limited process access)
   - **macOS**: Currently no isolation (CGO_ENABLED=1 required)

### Session Management

Background processes are tracked by `SessionManager` (singleton in `shell.go`):

```go
type ProcessSession struct {
    ID        string
    Cmd       *exec.Cmd
    Pty        *pty.Pty
    IsBackground bool
    Output    *bytes.Buffer
    // ...
}
```

- Sessions stored in global `sessionManager`
- `list` action returns all active sessions
- `poll` returns exit code and output
- Auto-cleanup after 30 minutes

## Security Measures

### Built-in Deny Patterns

The exec tool blocks 40+ dangerous command patterns (`shell.go` lines 50-98):

| Category | Blocked Patterns |
|----------|-------------------|
| **File Deletion** | `rm -rf`, `rm /f`, `rmdir /s`, `del /f` |
| **Disk Operations** | `format`, `mkfs`, `diskpart`, `dd if=` |
| **Block Devices** | Writes to `/dev/sd*`, `/dev/hd*`, `/dev/nvme*` |
| **System Control** | `shutdown`, `reboot`, `poweroff` |
| **Fork Bombs** | `:(){ :\|:& };:` (function fork bomb) |
| **Command Substitution** | Backticks, `$(...)`, `${...}` |
| **Pipe to Shell** | `| sh`, `| bash`, `curl ...\| sh`, `wget ...\| bash` |
| **Privilege Escalation** | `sudo`, `su -` |
| **Permission Changes** | `chmod 777`, `chmod -R 777`, `chown` |
| **Process Killing** | `kill -9`, `killall`, `pkill` |
| **Package Managers** | `npm install -g`, `pip install --user`, `apt/yum/dnf install/remove` |
| **Containers** | `docker run`, `docker exec`, `podman` |
| **Version Control** | `git push`, `git reset --hard`, `git force` |
| **Remote Access** | `ssh ...@`, `scp`, `rsync -e ssh` |
| **Shell Scripts** | `source *.sh`, `bash *.sh`, `eval` |

### Configuration-Based Security

| Config Key | Description | Default |
|------------|-------------|---------|
| `tools.exec.enabled` | Master switch for exec tool | `true` |
| `tools.exec.enable_deny_patterns` | Enable/disable built-in deny patterns | `true` |
| `tools.exec.custom_deny_patterns` | Additional regex patterns to block | `[]` |
| `tools.exec.custom_allow_patterns` | Exempt specific commands from deny checks | `[]` |
| `tools.exec.allow_remote` | Allow exec from external channels (Telegram, Discord, etc.) | `true` |
| `tools.exec.timeout_seconds` | Command timeout in seconds | `60` |

### Channel Restrictions

When `tools.exec.allow_remote = false`:
- **Allowed channels**: `cli`, `system`, `subagent` (defined in `pkg/constants/channels.go`)
- **Blocked channels**: `telegram`, `discord`, `slack`, `wechat`, etc.

### Workspace Restriction

When `agents.defaults.restrict_to_workspace = true`:
- Commands cannot access paths outside workspace
- Path traversal (`..`) blocked in commands
- Symlinks resolved and checked against workspace bounds

## How to Disable Exec

### Method 1: Disable via Config (Recommended)

**config.json**:
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**Environment variable**:
```bash
export PICOCLAW_TOOLS_EXEC_ENABLED=false
```

**Effect**: The exec tool is not registered. The agent cannot execute shell commands. Other tools (read_file, write_file, etc.) remain available.

### Method 2: Disable Remote Access Only

```json
{
  "tools": {
    "exec": {
      "enabled": true,
      "allow_remote": false
    }
  }
}
```

**Effect**: Exec only works from `cli`, `system`, and `subagent` channels. External chat channels cannot use exec.

### Method 3: Strict Timeout

```json
{
  "tools": {
    "exec": {
      "enabled": true,
      "timeout_seconds": 10
    }
  }
}
```

**Effect**: All commands timeout after 10 seconds (default 60s).

## How to Completely Remove Exec

To completely remove the exec tool from the codebase:

### Step 1: Delete Implementation Files
```bash
rm pkg/tools/shell.go          # Main ExecTool implementation
rm pkg/tools/shell_test.go     # Tests
rm pkg/tools/shell_process_unix.go
rm pkg/tools/shell_process_windows.go
rm pkg/tools/shell_timeout_unix_test.go
rm pkg/tools/session.go        # ProcessSession, SessionManager
rm pkg/tools/session_test.go
rm pkg/tools/spawn.go           # Spawn tool (depends on exec)
rm pkg/tools/spawn_test.go
rm pkg/tools/spawn_status.go
rm pkg/tools/spawn_status_test.go
```

### Step 2: Remove Registration Code

In `pkg/agent/instance.go`, remove lines ~104-112:
```go
// REMOVE:
if cfg.Tools.IsToolEnabled("exec") {
    execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg, allowReadPaths)
    if err != nil {
        logger.ErrorCF("agent", "Failed to initialize exec tool; continuing without exec", ...)
    } else {
        toolsRegistry.Register(execTool)
    }
}
```

### Step 3: Remove Config Structure

In `pkg/config/config.go`:
- Delete `ExecConfig` struct (around line 763-770)
- Remove `Exec ExecConfig` field from `ToolsConfig` (around line 822)
- Remove `"exec"` case from `IsToolEnabled()` function (around line 1534)

### Step 4: Remove Defaults

In `pkg/config/defaults.go`, delete the `Exec: ExecConfig{...}` block (around lines 365-372).

### Step 5: Remove Isolation (Optional)

If no other tools use isolation:
```bash
rm -rf pkg/isolation/
```

### Step 6: Update Documentation

Remove exec references from:
- `docs/reference/tools-api.md`
- `docs/reference/tools_configuration.md`

## How to Further Sandbox Exec

### A. Enable Subprocess Isolation

**Linux (bubblewrap)**:
```json
{
  "isolation": {
    "enabled": true,
    "expose_paths": [
      {"source": "/workspace", "target": "/workspace", "mode": "rw"},
      {"source": "/usr", "target": "/usr", "mode": "ro"},
      {"source": "/lib", "target": "/lib", "mode": "ro"},
      {"source": "/bin", "target": "/bin", "mode": "ro"}
    ]
  }
}
```

**Windows (restricted tokens)**:
```json
{
  "isolation": {
    "enabled": true
  }
}
```

### B. Restrict to Workspace Only

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true,
      "allow_read_outside_workspace": false
    }
  }
}
```

### C. Add Custom Deny Patterns

```json
{
  "tools": {
    "exec": {
      "custom_deny_patterns": [
        "nano", "vi", "vim", "emacs",
        "apt-get", "yum", "dnf",
        "docker", "podman"
      ]
    }
  }
}
```

### D. Block All Package Managers

```json
{
  "tools": {
    "exec": {
      "custom_deny_patterns": [
        "apt", "apt-get", "yum", "dnf", "pacman", "zypper",
        "npm", "yarn", "pnpm",
        "pip", "pip3", "conda",
        "gem", "bundle",
        "cargo", "rustup",
        "go get", "go install"
      ]
    }
  }
}
```

### E. Combined Strict Mode

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true
    }
  },
  "tools": {
    "exec": {
      "enabled": true,
      "enable_deny_patterns": true,
      "allow_remote": false,
      "timeout_seconds": 30,
      "custom_deny_patterns": [
        "apt", "yum", "dnf", "npm", "pip", "docker"
      ]
    }
  },
  "isolation": {
    "enabled": true
  }
}
```

## Key Files Reference

| File | Purpose |
|------|---------|
| `pkg/tools/shell.go` | ExecTool implementation, deny patterns, command guard |
| `pkg/tools/session.go` | ProcessSession and SessionManager |
| `pkg/tools/spawn.go` | Spawn tool (uses exec internally) |
| `pkg/tools/registry.go` | ToolRegistry - how tools are registered |
| `pkg/agent/instance.go` | Where exec tool is registered (lines 104-112) |
| `pkg/config/config.go` | ExecConfig struct (line 763), IsToolEnabled (line 1534) |
| `pkg/config/defaults.go` | Default exec settings (lines 365-372) |
| `pkg/isolation/runtime.go` | Subprocess isolation implementation |
| `pkg/constants/channels.go` | Internal channel definitions |

## Tool Result Structure

The exec tool returns `*ToolResult`:

```go
type ToolResult struct {
    ForLLM      string        // Command output for LLM processing
    ForUser     string        // Human-readable output for chat
    MediaURLs   []string      // Media attachments (if any)
    IsError     bool          // Whether execution failed
    Async       bool          // True if background process
    Err         error         // Underlying Go error
}
```

**Example success result**:
```json
{
  "ForLLM": "total 48\ndrwxr-xr-x 2 user user 4096 ...",
  "ForUser": "Command executed successfully",
  "IsError": false
}
```

**Example error result**:
```json
{
  "ForLLM": "Command blocked: matches deny pattern 'rm -rf'",
  "ForUser": "This command is not allowed for security reasons",
  "IsError": true
}

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

# Security Sandbox

PicoClaw includes a security sandbox to restrict file system access and command execution. This protects your system from accidental or malicious operations by the AI.

## Overview

The security sandbox provides:

- **Workspace restriction** - File operations limited to workspace directory
- **Command filtering** - Dangerous commands are blocked
- **Path traversal prevention** - Cannot access files outside workspace
- **Configurable patterns** - Add custom deny patterns

## Workspace Restriction

### Configuration

Enable or disable workspace restriction:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true,
      "workspace": "~/.picoclaw/workspace"
    }
  }
}
```

### Behavior When Enabled

When `restrict_to_workspace` is `true` (default):

- File read/write operations must be within workspace
- Shell commands execute from workspace directory
- Path traversal attempts (`../`) are blocked
- Absolute paths outside workspace are rejected

### Behavior When Disabled

When `restrict_to_workspace` is `false`:

- Full file system access is allowed
- Commands can run from any directory
- Use with caution in production environments

## Command Filtering

### Default Blocked Patterns

The following dangerous commands are always blocked:

| Category | Patterns |
|----------|----------|
| File deletion | `rm -rf`, `del /f`, `rmdir /s` |
| Disk operations | `format`, `mkfs`, `diskpart`, `dd if=` |
| System control | `shutdown`, `reboot`, `poweroff` |
| Privilege escalation | `sudo`, `chmod 777`, `chown` |
| Process control | `pkill`, `killall`, `kill -9` |
| Shell injection | `$(...)`, backticks, `eval`, `source` |
| Remote execution | `curl | sh`, `wget | bash` |
| Package management | `npm install -g`, `pip install --user`, `apt install` |
| Container escape | `docker run`, `docker exec` |
| Network | `ssh @` |
| Version control | `git push`, `git force` |

### Custom Deny Patterns

Add additional patterns to block:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "wget.*",
        "curl.*",
        "nc.*-l",
        "python.*-m.*http"
      ]
    }
  }
}
```

### Disable All Filtering

Disable command filtering entirely (not recommended):

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": false
    }
  }
}
```

## Path Traversal Protection

The sandbox detects and blocks path traversal attempts:

| Attempt | Blocked? |
|---------|----------|
| `../../../etc/passwd` | Yes |
| `..\..\..\windows\system32` | Yes |
| `/etc/passwd` | Yes (outside workspace) |
| `~/secret.txt` | Yes (outside workspace) |
| `workspace/file.txt` | No (relative, within workspace) |
| `/home/user/.picoclaw/workspace/file.txt` | No (absolute, within workspace) |

## Configuration Reference

### Agent Defaults

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `workspace` | string | `~/.picoclaw/workspace` | Working directory |
| `restrict_to_workspace` | bool | `true` | Enable sandbox restrictions |

### Exec Tool Configuration

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": []
    }
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable_deny_patterns` | bool | `true` | Enable command filtering |
| `custom_deny_patterns` | []string | `[]` | Additional patterns to block |

## Environment Variables

Override security settings with environment variables:

```bash
# Disable workspace restriction (not recommended)
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE="false"

# Disable command filtering (not recommended)
export PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS="false"

# Custom deny patterns
export PICOCLAW_TOOLS_EXEC_CUSTOM_DENY_PATTERNS='["wget.*", "curl.*"]'
```

## Multi-Agent Security

Each agent can have its own security settings:

```json
{
  "agents": {
    "list": [
      {
        "id": "trusted",
        "workspace": "~/.picoclaw/workspace/trusted",
        "restrict_to_workspace": false
      },
      {
        "id": "sandboxed",
        "workspace": "~/.picoclaw/workspace/sandboxed",
        "restrict_to_workspace": true
      }
    ]
  }
}
```

### Workspace Isolation

Different agents with isolated workspaces:

```
~/.picoclaw/workspace/
├── agent-a/     # Agent A's sandbox
│   ├── files/
│   └── sessions/
├── agent-b/     # Agent B's sandbox
│   ├── files/
│   └── sessions/
└── shared/      # Shared workspace (if configured)
```

## Security Best Practices

### 1. Keep Sandbox Enabled

Leave `restrict_to_workspace` enabled for production:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true
    }
  }
}
```

### 2. Use Dedicated Workspace

Create a dedicated workspace directory:

```bash
mkdir -p ~/.picoclaw/workspace
```

### 3. Review Custom Patterns

When adding custom deny patterns, test them thoroughly:

```json
{
  "tools": {
    "exec": {
      "custom_deny_patterns": [
        "\\bmysqldump\\b",
        "\\bpg_dump\\b"
      ]
    }
  }
}
```

### 4. Separate Agent Workspaces

Use different workspaces for different trust levels:

```json
{
  "agents": {
    "list": [
      {
        "id": "production",
        "workspace": "~/.picoclaw/workspace/prod",
        "restrict_to_workspace": true
      },
      {
        "id": "development",
        "workspace": "~/.picoclaw/workspace/dev",
        "restrict_to_workspace": true
      }
    ]
  }
}
```

### 5. Monitor Logs

Enable debug logging to monitor blocked operations:

```bash
picoclaw agent --debug
```

## Error Messages

When operations are blocked, users see clear error messages:

| Error | Cause |
|-------|-------|
| `Command blocked by safety guard (dangerous pattern detected)` | Command matches deny pattern |
| `Command blocked by safety guard (path traversal detected)` | Path contains `../` or `..\` |
| `Command blocked by safety guard (path outside working dir)` | Absolute path outside workspace |
| `access denied: path outside workspace` | File operation outside workspace |

## Troubleshooting

### Legitimate Commands Blocked

If safe commands are being blocked:

1. Check if command matches a deny pattern
2. Consider adding to allowlist (if configured)
3. Adjust custom deny patterns

### Cannot Access Files

If file access is denied:

1. Verify file is within workspace
2. Check workspace path configuration
3. Ensure `restrict_to_workspace` is configured correctly

### Need More Access

For trusted environments:

1. Disable workspace restriction temporarily
2. Use a separate agent with relaxed security
3. Move required files into workspace

## Related Topics

- [Multi-Agent System](multi-agent.md) - Isolated agent workspaces
- [Exec Tool](../tools/exec.md) - Command execution configuration
- [Filesystem Tool](../tools/filesystem.md) - File operations
- [Workspace Management](../workspace/README.md) - Workspace structure

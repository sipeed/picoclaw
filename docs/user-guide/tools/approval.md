# Approval Tool

The approval tool controls permissions for sensitive operations, requiring user confirmation before executing.

## Overview

When enabled, the approval tool prompts the user to confirm dangerous operations before they are executed. This adds an extra layer of safety for operations that could modify files or execute commands.

## Configuration

```json
{
  "tools": {
    "approval": {
      "enabled": true,
      "write_file": true,
      "edit_file": true,
      "append_file": true,
      "exec": true,
      "timeout_minutes": 5
    }
  }
}
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable approval functionality |
| `write_file` | bool | `true` | Require approval for file writes |
| `edit_file` | bool | `true` | Require approval for file edits |
| `append_file` | bool | `true` | Require approval for file appends |
| `exec` | bool | `true` | Require approval for command execution |
| `timeout_minutes` | int | `5` | Approval timeout in minutes |

## How It Works

1. Agent requests to perform a sensitive operation
2. Approval tool intercepts the request
3. User is prompted to approve or deny
4. If approved within timeout, operation proceeds
5. If denied or timeout, operation is cancelled

## Example Flow

```
Agent: I need to write to config.json. Requesting approval...

[Approval Request]
Operation: write_file
Path: config.json
Approve? (y/n): y

Agent: Approved. Writing to config.json...
```

## Disabling Specific Approvals

Disable approval for specific operations:

```json
{
  "tools": {
    "approval": {
      "enabled": true,
      "write_file": false,
      "exec": false
    }
  }
}
```

This allows the agent to write files and execute commands without approval, while still requiring approval for edits and appends.

## Disabling All Approvals

```json
{
  "tools": {
    "approval": {
      "enabled": false
    }
  }
}
```

**Warning**: Disabling approvals reduces safety. Use only in trusted environments.

## Environment Variables

```bash
export PICOCLAW_TOOLS_APPROVAL_ENABLED=false
export PICOCLAW_TOOLS_APPROVAL_TIMEOUT_MINUTES=10
```

## See Also

- [Exec Tool](exec.md)
- [File System Tools](filesystem.md)
- [Security Sandbox](../advanced/security-sandbox.md)

# Exec Tool

Execute shell commands from within the agent.

## Tool

### exec

Execute a shell command.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `command` | string | Yes | Shell command to execute |

**Example:**

```json
{
  "command": "ls -la"
}
```

## Security

### Workspace Restriction

When `restrict_to_workspace: true`:

- Commands execute within workspace directory
- Path arguments must be within workspace

### Deny Patterns

Dangerous commands are always blocked:

| Pattern | Reason |
|---------|--------|
| `rm -rf`, `rmdir /s` | Bulk deletion |
| `format`, `mkfs` | Disk formatting |
| `dd if=` | Disk imaging |
| `shutdown`, `reboot` | System control |
| `:(){ :\|:& };:` | Fork bomb |
| `wget`, `curl` (configurable) | Network downloads |

### Custom Deny Patterns

Add your own blocked patterns:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "apt.*",
        "yum.*",
        "pip.*"
      ]
    }
  }
}
```

## Timeout

Commands have a 60-second timeout by default. Long-running commands will be terminated.

## Usage Examples

### List Files

```
User: "Show me all Python files"

Agent uses exec:
{
  "command": "find . -name '*.py'"
}

Agent: "Found 3 Python files:
- main.py
- utils.py
- test.py"
```

### Git Operations

```
User: "What's the current git status?"

Agent uses exec:
{
  "command": "git status"
}

Agent: "On branch main, working tree clean."
```

### Process Information

```
User: "How much disk space is free?"

Agent uses exec:
{
  "command": "df -h ."
}

Agent: "You have 50GB free of 100GB total."
```

## Error Handling

```
User: "Run make build"

Agent uses exec:
{
  "command": "make build"
}

If command fails:
Agent: "The command failed with exit code 1. Error: make: *** No targets specified."
```

## Disabling Restriction

For trusted environments:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Warning**: This allows running any command the user can execute.

## See Also

- [Tools Overview](README.md)
- [Security Sandbox](../advanced/security-sandbox.md)

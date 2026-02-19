# Tools Overview

PicoClaw has built-in tools that allow the agent to interact with files, execute commands, search the web, and more.

## Available Tools

### File System Tools

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Create or overwrite files |
| `list_dir` | List directory contents |
| `edit_file` | Replace text in files |
| `append_file` | Append to files |

### Execution Tools

| Tool | Description |
|------|-------------|
| `exec` | Execute shell commands |

### Web Tools

| Tool | Description |
|------|-------------|
| `web_search` | Search the web |
| `web_fetch` | Fetch web page content |

### Communication Tools

| Tool | Description |
|------|-------------|
| `message` | Send messages to channels |
| `spawn` | Create subagents for background tasks |

### Scheduling Tools

| Tool | Description |
|------|-------------|
| `cron` | Schedule reminders and tasks |

### Hardware Tools (Linux only)

| Tool | Description |
|------|-------------|
| `i2c` | I2C bus interaction |
| `spi` | SPI bus interaction |

## Tool Security

### Workspace Restriction

When `restrict_to_workspace: true` (default):

- File tools only work within workspace
- Exec commands run in workspace directory
- Symlink escapes are blocked

### Exec Protection

Even with workspace restriction disabled, these commands are blocked:

- `rm -rf`, `del /f`, `rmdir /s`
- `format`, `mkfs`, `diskpart`
- `dd if=`
- `shutdown`, `reboot`, `poweroff`
- Fork bombs

## Tool Configuration

### Web Search

Configure search providers:

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Exec

Configure command restrictions:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": ["wget.*", "curl.*"]
    }
  }
}
```

### Cron

Configure job timeout:

```json
{
  "tools": {
    "cron": {
      "exec_timeout_minutes": 5
    }
  }
}
```

## Using Tools

The agent automatically uses tools based on your requests:

```
User: "Read the file notes.txt and summarize it"

Agent: [Uses read_file tool]
       The file contains...
```

## Tool Details

- [File System Tools](filesystem.md)
- [Exec Tool](exec.md)
- [Web Tools](web.md)
- [Messaging Tool](messaging.md)
- [Spawn Tool](spawn.md)
- [Cron Tool](cron.md)
- [Hardware Tools](hardware.md)

## See Also

- [Security Sandbox](../advanced/security-sandbox.md)
- [Configuration Reference](../../configuration/config-file.md)

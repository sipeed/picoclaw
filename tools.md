---
id: tools
title: Tools Configuration
---

# Tools Configuration

PicoClaw's tools configuration is located in `tools` field of `config.json`.

```json
{
  "tools": {
    "web": { ... },
    "mcp": { ... },
    "exec": { ... },
    "cron": { ... },
    "skills": { ... }
  }
}
```

## Tool Enable/Disable

PicoClaw supports granular control over tool availability at two levels:

1. **Tool Group Level**: Enable/disable entire tool groups (e.g., `web`, `exec`, `skills`)
2. **Individual Tool Level**: Enable/disable specific tools (e.g., `web_search`, `read_file`)

Individual tool settings override group-level settings, allowing fine-grained control.

### Default Tool Configuration

By default, PicoClaw enables these tool groups:

| Tool Group | Default | Description |
| --- | --- | --- |
| `web` | true | Web search and fetch tools |
| `exec` | true | Shell command execution |
| `cron` | true | Task scheduling |
| `skills` | true | Skill management and discovery |
| `media_cleanup` | true | Automatic media file cleanup |

Hardware tools are disabled by default (Linux only):

| Tool | Default | Description |
| --- | --- | --- |
| `i2c` | false | I2C hardware communication |
| `spi` | false | SPI hardware communication |

### Enabling/Disabling Tools

#### Disable Entire Tool Group

```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

#### Enable Individual Tool (Override Group)

```json
{
  "tools": {
    "web": {
      "enabled": true
    },
    "web_search": {
      "enabled": false
    },
    "web_fetch": {
      "enabled": true
    }
  }
}
```

In this example:
- All web tools are enabled by the group setting
- `web_search` is specifically disabled
- `web_fetch` remains enabled (individual setting not needed when group is enabled)

#### Disable All Dangerous Tools

```json
{
  "tools": {
    "exec": {
      "enabled": false
    },
    "spawn": {
      "enabled": false
    },
    "i2c": {
      "enabled": false
    },
    "spi": {
      "enabled": false
    }
  }
}
```

## Web Tools

Web tools are used for web search and fetching.

### Web Tool Group

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable all web tools |
| `proxy` | string | — | Proxy URL for all web tools (http, https, socks5) |
| `fetch_limit_bytes` | int64 | 10485760 | Maximum bytes to fetch per URL (default 10MB) |

### Brave Search

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enable Brave search |
| `api_key` | string | — | Brave Search API key |
| `max_results` | int | 5 | Maximum number of results |

Get a free API key at [brave.com/search/api](https://brave.com/search/api) (2000 free queries/month).

### DuckDuckGo

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable DuckDuckGo search |
| `max_results` | int | 5 | Maximum number of results |

DuckDuckGo is enabled by default and requires no API key.

### Perplexity

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enable Perplexity search |
| `api_key` | string | — | Perplexity API key |
| `max_results` | int | 5 | Maximum number of results |

### Tavily

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enable Tavily search |
| `api_key` | string | — | Tavily API key |
| `base_url` | string | — | Custom Tavily API base URL |
| `max_results` | int | 5 | Maximum number of results |

### GLM Search (智谱)

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enable GLM search |
| `api_key` | string | — | GLM API key |
| `base_url` | string | — | Custom GLM API base URL |
| `search_engine` | string | "search_std" | Search backend type |
| `max_results` | int | 5 | Maximum number of results |

### Web Tools Configuration Example

```json
{
  "tools": {
    "web": {
      "enabled": true,
      "proxy": "socks5://127.0.0.1:1080",
      "fetch_limit_bytes": 5242880
    },
    "web_search": {
      "enabled": false
    }
  }
}
```

## File System Tools

File system tools allow the agent to read, write, and manipulate files in the workspace.

| Tool | Default | Description |
| --- | --- | --- |
| `read_file` | true | Read file contents |
| `write_file` | true | Write content to files |
| `edit_file` | true | Edit files by replacing text |
| `append_file` | true | Append content to files |
| `list_dir` | true | List files and directories |

### File System Configuration Example

```json
{
  "tools": {
    "read_file": {
      "enabled": true
    },
    "write_file": {
      "enabled": true
    },
    "list_dir": {
      "enabled": true
    }
  }
}
```

## Exec Tool

The exec tool executes shell commands on behalf of the agent.

### Exec Tool Group

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable exec tool |

### Exec Tool Security

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enable_deny_patterns` | bool | true | Enable default dangerous command blocking |
| `custom_deny_patterns` | array | [] | Custom deny patterns (regular expressions) |
| `custom_allow_patterns` | array | [] | Custom allow patterns — matching commands bypass deny checks |

### Default Blocked Command Patterns

By default, PicoClaw blocks these dangerous commands:

- Delete commands: `rm -rf`, `del /f/q`, `rmdir /s`
- Disk operations: `format`, `mkfs`, `diskpart`, `dd if=`, writing to block devices (`/dev/sd*`, `/dev/n/nvme*`, `/dev/mmcblk*`, etc.)
- System operations: `shutdown`, `reboot`, `poweroff`
- Command substitution: `$()`, `${}`, backticks
- Pipe to shell: `| sh`, `| bash`
- Privilege escalation: `sudo`, `chmod`, `chown`
- Process control: `pkill`, `killall`, `kill -9`
- Remote operations: `curl | sh`, `wget | sh`, `ssh`
- Package management: `apt`, `yum`, `dnf`, `npm install -g`, `pip install --user`
- Containers: `docker run`, `docker exec`
- Git: `git push`, `git force`
- Other: `eval`, `source *.sh`

### Custom Allow Patterns

Use `custom_allow_patterns` to explicitly permit commands that would otherwise be blocked by deny patterns:

```json
{
  "tools": {
    "exec": {
      "enabled": true,
      "enable_deny_patterns": true,
      "custom_allow_patterns": [
        "^git push origin main$"
      ]
    }
  }
}
```

### Exec Configuration Example

```json
{
  "tools": {
    "exec": {
      "enabled": true,
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ],
      "custom_allow_patterns": []
    }
  }
}
```

## Cron Tool

The cron tool schedules periodic tasks.

### Cron Tool Group

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable cron tool |

### Cron Tool Configuration

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `exec_timeout_minutes` | int | 5 | Execution timeout in minutes (0 = no limit) |

## Skills Tool

The skills tool manages skill discovery and installation via registries like ClawHub.

### Skills Tool Group

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable skills tool group |

### Individual Skills Tools

| Tool | Default | Description |
| --- | --- | --- |
| `find_skills` | true | Search for installable skills |
| `install_skill` | true | Install a skill from registry |

### Skills Configuration

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `max_concurrent_searches` | int | 2 | Maximum concurrent search queries |

### Skills Registry (ClawHub)

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable ClawHub registry |
| `base_url` | string | `https://clawhub.ai` | ClawHub base URL |

### Skills Search Cache

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `max_size` | int | 50 | Maximum cache entries |
| `ttl_seconds` | int | 300 | Cache entry TTL in seconds |

### Skills Configuration Example

```json
{
  "tools": {
    "skills": {
      "enabled": true,
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai"
        }
      },
      "max_concurrent_searches": 2,
      "search_cache": {
        "max_size": 50,
        "ttl_seconds": 300
      }
    }
  }
}
```

## MCP (Model Context Protocol)

PicoClaw supports MCP servers for extending agent capabilities with external tools.

### MCP Configuration

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enable MCP integration |
| `servers` | object | {} | Named MCP server configurations |

Each MCP server supports two connection modes:

**stdio mode** (local process):

| Config | Type | Description |
| --- | --- | --- |
| `enabled` | bool | Enable this server |
| `command` | string | Command to run (e.g., `npx`) |
| `args` | array | Command arguments |
| `env` | object | Environment variables |
| `env_file` | string | Path to env file |

**HTTP/SSE mode** (remote server):

| Config | Type | Description |
| --- | --- | --- |
| `enabled` | bool | Enable this server |
| `type` | string | `"http"` or `"sse"` |
| `url` | string | Server URL |
| `headers` | object | HTTP headers (e.g., API keys) |

### MCP Configuration Example

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-github"],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxx"
          }
        },
        "context7": {
          "enabled": true,
          "type": "http",
          "url": "https://mcp.context7.com/mcp",
          "headers": {
            "CONTEXT7_API_KEY": "ctx7sk-xx"
          }
        }
      }
    }
  }
}
```

MCP tools are registered with naming convention `mcp_<server>_<tool>` and appear alongside built-in tools.

## Agent Tools

Tools related to agent management and message handling.

| Tool | Default | Description |
| --- | --- | --- |
| `message` | true | Send messages to user on channels |
| `spawn` | true | Spawn background subagent tasks |
| `subagent` | true | Execute subagent tasks synchronously |

## Hardware Tools (Linux Only)

Hardware tools for communicating with devices on Linux systems.

| Tool | Default | Description |
| --- | --- | --- |
| `i2c` | false | I2C bus device communication |
| `spi` | false | SPI bus device communication |

These tools are disabled by default as they only work on Linux.

## Media Cleanup

Automatic cleanup of media files from workspace.

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable media cleanup service |
| `max_age_minutes` | int | 30 | Maximum age of media files in minutes |
| `interval_minutes` | int | 5 | Cleanup interval in minutes |

## Complete Example

```json
{
  "tools": {
    "allow_read_paths": ["/tmp", "/home/user/docs"],
    "allow_write_paths": ["/home/user/workspace"],
    
    "web": {
      "enabled": true,
      "proxy": "socks5://127.0.0.1:1080",
      "duckduckgo": {
        "enabled": true,
        "max_results": 10
      }
    },
    
    "exec": {
      "enabled": true,
      "enable_deny_patterns": true,
      "custom_allow_patterns": []
    },
    
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 10
    },
    
    "skills": {
      "enabled": true,
      "max_concurrent_searches": 5
    },
    
    "mcp": {
      "enabled": false
    },
    
    "media_cleanup": {
      "enabled": true,
      "max_age_minutes": 60
    },
    
    "read_file": {
      "enabled": true
    },
    "write_file": {
      "enabled": true
    },
    "i2c": {
      "enabled": false
    }
  }
}
```

## Environment Variables

Override config options with environment variables using `PICOCLAW_TOOLS_<SECTION>_<KEY>`:

### Tool Group Controls

- `PICOCLAW_TOOLS_WEB_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_CRON_ENABLED=true`
- `PICOCLAW_TOOLS_SKILLS_ENABLED=true`
- `PICOCLAW_TOOLS_MEDIA_CLEANUP_ENABLED=true`

### Individual Tool Controls

- `PICOCLAW_TOOLS_READ_FILE_ENABLED=false`
- `PICOCLAW_TOOLS_WRITE_FILE_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_TOOL_ENABLED=false`
- `PICOCLAW_TOOLS_I2C_ENABLED=true`
- `PICOCLAW_TOOLS_SPI_ENABLED=true`

### Other Environment Variables

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_SKILLS_MAX_CONCURRENT_SEARCHES=5`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

Note: Array-type environment variables must be set via config file.

## Security Best Practices

1. **Disable dangerous tools in production**: Disable `exec` and `spawn` for public-facing deployments
2. **Use workspace restrictions**: Configure `restrict_to_workspace` to limit file access
3. **Enable deny patterns**: Keep `enable_deny_patterns` enabled for the exec tool
4. **Disable hardware tools**: Disable `i2c` and `spi` on non-Linux systems
5. **Limit web access**: Disable web tools or use a proxy for controlled access
6. **Review MCP servers**: Only enable trusted MCP servers with proper authentication

## Migration Guide

### From Older Versions

Previous versions of PicoClaw had all tools enabled by default. To migrate:

1. **No changes required**: Tools will work as before (backward compatible)
2. **Optional customization**: Add tool configuration to your config file as needed

Example minimal migration:

```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

This disables the exec tool while keeping all other tools at their default settings.

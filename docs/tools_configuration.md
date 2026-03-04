# Tools Configuration

PicoClaw's tools configuration is located in the `tools` field of `config.json`.

## Directory Structure

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

## Web Tools

Web tools are used for web search and fetching.

### Brave

| Config        | Type   | Default | Description               |
| ------------- | ------ | ------- | ------------------------- |
| `enabled`     | bool   | false   | Enable Brave search       |
| `api_key`     | string | -       | Brave Search API key      |
| `max_results` | int    | 5       | Maximum number of results |

### DuckDuckGo

| Config        | Type | Default | Description               |
| ------------- | ---- | ------- | ------------------------- |
| `enabled`     | bool | true    | Enable DuckDuckGo search  |
| `max_results` | int  | 5       | Maximum number of results |

### Perplexity

| Config        | Type   | Default | Description               |
| ------------- | ------ | ------- | ------------------------- |
| `enabled`     | bool   | false   | Enable Perplexity search  |
| `api_key`     | string | -       | Perplexity API key        |
| `max_results` | int    | 5       | Maximum number of results |

## Exec Tool

The exec tool executes shell commands using an in-process POSIX interpreter with
AST-based risk classification, environment sanitization, and file-access sandboxing.

### Configuration

| Config           | Type   | Default    | Description                                                             |
| ---------------- | ------ | ---------- | ----------------------------------------------------------------------- |
| `risk_threshold` | string | `"medium"` | Maximum allowed risk level: `"low"`, `"medium"`, `"high"`, `"critical"` |
| `risk_overrides` | object | `{}`       | Per-command risk level overrides (command name → level)                 |
| `arg_modifiers`  | object | `{}`       | Per-command argument patterns that adjust risk level                    |
| `env_allowlist`  | array  | `[]`       | Extra environment variables to expose (extends built-in defaults)       |
| `env_set`        | object | `{}`       | Explicit `VAR=value` pairs injected into every command                  |

### Risk Classification

Every command is parsed into an AST before execution. Each resolved binary is
looked up in a built-in risk table with four levels:

| Level      | Meaning                               | Examples                           |
| ---------- | ------------------------------------- | ---------------------------------- |
| `low`      | Read-only / informational             | `echo`, `cat`, `ls`, `date`        |
| `medium`   | Writes files but limited blast radius | `cp`, `mv`, `mkdir`, `tee`         |
| `high`     | System-wide side effects              | `apt`, `brew`, `docker`, `mount`   |
| `critical` | Destructive / privilege escalation    | `sudo`, `rm -rf`, `shutdown`, `dd` |

Commands with a risk level **above** `risk_threshold` are blocked before execution.

#### Argument modifiers

Some commands change risk depending on their arguments. For example, `rm` is
`medium` by default but becomes `critical` when called with `-rf`.

You can add custom argument modifiers via config. Each entry lists tokens that
must all be present (order-independent) and the resulting level:

```json
{
  "arg_modifiers": {
    "curl": [{ "args": ["--upload-file"], "level": "high" }],
    "git": [{ "args": ["push", "--force"], "level": "critical" }]
  }
}
```

The **highest matching** modifier wins (built-in and custom are merged).

### Environment Sanitization

The shell interpreter runs with a sanitized environment. Only a safe allowlist
of variables is exposed (e.g., `PATH`, `HOME`, `LANG`, `TERM`).

- **`env_allowlist`**: Extend the defaults with additional variable names.
- **`env_set`**: Inject fixed `VAR=value` pairs (overrides real env).

### File-Access Sandboxing

When `restrict_to_workspace` is enabled (the default), the interpreter's
`OpenHandler` blocks reads and writes outside the configured workspace directory.

### Cron Integration

The cron tool creates its own `ExecTool` via `NewExecToolWithConfig`, so
scheduled commands go through the same risk classifier, env sanitization, and
sandbox as agent-originated commands.

### Configuration Example

```json
{
  "tools": {
    "exec": {
      "risk_threshold": "medium",
      "risk_overrides": {
        "ffmpeg": "low",
        "terraform": "critical"
      },
      "arg_modifiers": {
        "curl": [{ "args": ["--upload-file"], "level": "high" }]
      },
      "env_allowlist": ["GOPATH", "JAVA_HOME"],
      "env_set": {
        "NODE_ENV": "production"
      }
    }
  }
}
```

## Cron Tool

The cron tool is used for scheduling periodic tasks.

| Config                 | Type | Default | Description                                    |
| ---------------------- | ---- | ------- | ---------------------------------------------- |
| `exec_timeout_minutes` | int  | 5       | Execution timeout in minutes, 0 means no limit |

## MCP Tool

The MCP tool enables integration with external Model Context Protocol servers.

### Global Config

| Config    | Type   | Default | Description                         |
| --------- | ------ | ------- | ----------------------------------- |
| `enabled` | bool   | false   | Enable MCP integration globally     |
| `servers` | object | `{}`    | Map of server name to server config |

### Per-Server Config

| Config     | Type   | Required | Description                                |
| ---------- | ------ | -------- | ------------------------------------------ |
| `enabled`  | bool   | yes      | Enable this MCP server                     |
| `type`     | string | no       | Transport type: `stdio`, `sse`, `http`     |
| `command`  | string | stdio    | Executable command for stdio transport     |
| `args`     | array  | no       | Command arguments for stdio transport      |
| `env`      | object | no       | Environment variables for stdio process    |
| `env_file` | string | no       | Path to environment file for stdio process |
| `url`      | string | sse/http | Endpoint URL for `sse`/`http` transport    |
| `headers`  | object | no       | HTTP headers for `sse`/`http` transport    |

### Transport Behavior

- If `type` is omitted, transport is auto-detected:
  - `url` is set → `sse`
  - `command` is set → `stdio`
- `http` and `sse` both use `url` + optional `headers`.
- `env` and `env_file` are only applied to `stdio` servers.

### Configuration Examples

#### 1) Stdio MCP server

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

#### 2) Remote SSE/HTTP MCP server

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "remote-mcp": {
          "enabled": true,
          "type": "sse",
          "url": "https://example.com/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_TOKEN"
          }
        }
      }
    }
  }
}
```

## Skills Tool

The skills tool configures skill discovery and installation via registries like ClawHub.

### Registries

| Config                             | Type   | Default              | Description             |
| ---------------------------------- | ------ | -------------------- | ----------------------- |
| `registries.clawhub.enabled`       | bool   | true                 | Enable ClawHub registry |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub base URL        |
| `registries.clawhub.auth_token`    | string | `""`                 | Optional Bearer token for higher rate limits |
| `registries.clawhub.search_path`   | string | `/api/v1/search`     | Search API path         |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`     | Skills API path         |
| `registries.clawhub.download_path` | string | `/api/v1/download`   | Download API path       |

### Configuration Example

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": "",
          "search_path": "/api/v1/search",
          "skills_path": "/api/v1/skills",
          "download_path": "/api/v1/download"
        }
      }
    }
  }
}
```

## Environment Variables

All configuration options can be overridden via environment variables with the format `PICOCLAW_TOOLS_<SECTION>_<KEY>`:

For example:

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

Note: Nested map-style config (for example `tools.mcp.servers.<name>.*`) is configured in `config.json` rather than environment variables.

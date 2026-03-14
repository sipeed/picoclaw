# Tools Configuration

PicoClaw's tools configuration lives under the `tools` field in `config.json`.
For agent/session/channel fields outside the tool system, see [configuration.md](configuration.md).

## Top-Level Tool Gates

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `allow_read_paths` | array or null | `null` | Regex allow-list layered on top of the workspace sandbox for read-only tools |
| `allow_write_paths` | array or null | `null` | Regex allow-list for write-capable tools |

## Directory Structure

```json
{
  "tools": {
    "web": {
      ...
    },
    "mcp": {
      ...
    },
    "exec": {
      ...
    },
    "cron": {
      ...
    },
    "skills": {
      ...
    }
  }
}
```

## Web Tools

Web tools are used for web search and fetching.

### Global Web Config

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable/disable the entire web tool family |
| `proxy` | string | `""` | Optional proxy URL for web search/fetch |
| `fetch_limit_bytes` | int | `10485760` | Maximum response size fetched by web tools before truncation/refusal |

### Brave

| Config        | Type   | Default | Description               |
|---------------|--------|---------|---------------------------|
| `enabled`     | bool   | false   | Enable Brave search       |
| `api_key`     | string | -       | Brave Search API key      |
| `max_results` | int    | 5       | Maximum number of results |

### DuckDuckGo

| Config        | Type | Default | Description               |
|---------------|------|---------|---------------------------|
| `enabled`     | bool | true    | Enable DuckDuckGo search  |
| `max_results` | int  | 5       | Maximum number of results |

### Perplexity

| Config        | Type   | Default | Description               |
|---------------|--------|---------|---------------------------|
| `enabled`     | bool   | false   | Enable Perplexity search  |
| `api_key`     | string | -       | Perplexity API key        |
| `max_results` | int    | 5       | Maximum number of results |

### Tavily

| Config        | Type   | Default | Description               |
|---------------|--------|---------|---------------------------|
| `enabled`     | bool   | false   | Enable Tavily search      |
| `api_key`     | string | -       | Tavily API key            |
| `base_url`    | string | `""`    | Optional Tavily-compatible endpoint |
| `max_results` | int    | 5       | Maximum number of results |

### SearXNG

| Config        | Type   | Default | Description                  |
|---------------|--------|---------|------------------------------|
| `enabled`     | bool   | false   | Enable SearXNG search        |
| `base_url`    | string | `""`    | Base URL of your SearXNG instance |
| `max_results` | int    | 5       | Maximum number of results    |

### GLM Search

| Config          | Type   | Default | Description |
|-----------------|--------|---------|-------------|
| `enabled`       | bool   | false   | Enable Zhipu GLM web search |
| `api_key`       | string | -       | GLM API key |
| `base_url`      | string | `https://open.bigmodel.cn/api/paas/v4/web_search` | Search endpoint |
| `search_engine` | string | `search_std` | Backend name such as `search_std`, `search_pro`, `search_pro_sogou`, `search_pro_quark` |
| `max_results`   | int    | 5       | Maximum number of results |

## Exec Tool

The exec tool is used to execute shell commands.

| Config                 | Type  | Default | Description                                |
|------------------------|-------|---------|--------------------------------------------|
| `enabled`              | bool  | true    | Enable/disable the exec tool               |
| `enable_deny_patterns` | bool  | true    | Enable default dangerous command blocking  |
| `allow_remote`         | bool  | true    | Allow exec calls from remote channels      |
| `timeout_seconds`      | int   | 60      | Per-command timeout (`0` uses the built-in default) |
| `custom_deny_patterns` | array | []      | Custom deny patterns (regular expressions) |
| `custom_allow_patterns` | array | []     | Regex allow-list checked before deny rules |

### Functionality

- **`enable_deny_patterns`**: Set to `false` to completely disable the default dangerous command blocking patterns
- **`allow_remote`**: Set to `false` to require an internal/local channel context before exec may run
- **`timeout_seconds`**: Set to `0` to fall back to the compiled default timeout
- **`custom_deny_patterns`**: Add custom deny regex patterns; commands matching these will be blocked
- **`custom_allow_patterns`**: Explicit allow regexes that are evaluated before deny rules

### Default Blocked Command Patterns

By default, PicoClaw blocks the following dangerous commands:

- Delete commands: `rm -rf`, `del /f/q`, `rmdir /s`
- Disk operations: `format`, `mkfs`, `diskpart`, `dd if=`, writing to `/dev/sd*`
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

### Configuration Example

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ]
    }
  }
}
```

## Cron Tool

The cron tool is used for scheduling periodic tasks.

| Config                 | Type | Default | Description                                    |
|------------------------|------|---------|------------------------------------------------|
| `enabled`              | bool | true    | Enable/disable the cron tool                   |
| `exec_timeout_minutes` | int  | 5       | Execution timeout in minutes, 0 means no limit |

## Skills Tool

The skills tool controls skill search, installation, and registry lookup.

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable skill-related tools |
| `max_concurrent_searches` | int | 2 | Max concurrent skill search/download lookups |
| `github.proxy` | string | `""` | Optional proxy for GitHub-backed skill downloads |
| `github.token` | string | `""` | GitHub token for higher rate limits/private repos |
| `search_cache.max_size` | int | 50 | Number of cached skill-search results |
| `search_cache.ttl_seconds` | int | 300 | Skill-search cache TTL |
| `registries.clawhub.enabled` | bool | true | Enable the default ClawHub registry |
| `registries.clawhub.base_url` | string | `https://clawhub.ai` | Registry base URL |
| `registries.clawhub.auth_token` | string | `""` | Optional registry auth token |
| `registries.clawhub.search_path` | string | `""` | Override search endpoint path |
| `registries.clawhub.skills_path` | string | `""` | Override skill metadata endpoint path |
| `registries.clawhub.download_path` | string | `""` | Override download endpoint path |
| `registries.clawhub.timeout` | int | `0` | Custom timeout override (`0` means use default) |
| `registries.clawhub.max_zip_size` | int | `0` | Custom zip size cap (`0` means use default) |
| `registries.clawhub.max_response_size` | int | `0` | Custom response size cap (`0` means use default) |

## Media Cleanup Tool

Media cleanup deletes temporary media artifacts created during channel/tool use.

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable background cleanup |
| `max_age_minutes` | int | 30 | Delete media older than this |
| `interval_minutes` | int | 5 | Cleanup sweep cadence |

## Read File Tool

| Config | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | true | Enable the read-file tool |
| `max_read_file_size` | int | 65536 | Maximum bytes returned by a single read |

## MCP Tool

The MCP tool enables integration with external Model Context Protocol servers.

### Tool Discovery (Lazy Loading)

When connecting to multiple MCP servers, exposing hundreds of tools simultaneously can exhaust the LLM's context window
and increase API costs. The **Discovery** feature solves this by keeping MCP tools *hidden* by default.

Instead of loading all tools, the LLM is provided with a lightweight search tool (using BM25 keyword matching or Regex).
When the LLM needs a specific capability, it searches the hidden library. Matching tools are then temporarily "unlocked"
and injected into the context for a configured number of turns (`ttl`).

### Global Config

| Config      | Type   | Default | Description                                  |
|-------------|--------|---------|----------------------------------------------|
| `enabled`   | bool   | false   | Enable MCP integration globally              |
| `discovery` | object | `{}`    | Configuration for Tool Discovery (see below) |
| `servers`   | object | `{}`    | Map of server name to server config          |

### Discovery Config (`discovery`)

| Config               | Type | Default | Description                                                                                                                       |
|----------------------|------|---------|-----------------------------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false   | If true, MCP tools are hidden and loaded on-demand via search. If false, all tools are loaded                                     |
| `ttl`                | int  | 5       | Number of conversational turns a discovered tool remains unlocked                                                                 |
| `max_search_results` | int  | 5       | Maximum number of tools returned per search query                                                                                 |
| `use_bm25`           | bool | true    | Enable the natural language/keyword search tool (`tool_search_tool_bm25`). **Warning**: consumes more resources than regex search |
| `use_regex`          | bool | false   | Enable the regex pattern search tool (`tool_search_tool_regex`)                                                                   |

> **Note:** If `discovery.enabled` is `true`, you MUST enable at least one search engine (`use_bm25` or `use_regex`),
> otherwise the application will fail to start.

### Per-Server Config

| Config     | Type   | Required | Description                                |
|------------|--------|----------|--------------------------------------------|
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
          "args": [
            "-y",
            "@modelcontextprotocol/server-filesystem",
            "/tmp"
          ]
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

#### 3) Massive MCP setup with Tool Discovery enabled

*In this example, the LLM will only see the `tool_search_tool_bm25`. It will search and unlock Github or Postgres tools
dynamically only when requested by the user.*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true,
        "use_regex": false
      },
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-github"
          ],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_TOKEN"
          }
        },
        "postgres": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-postgres",
            "postgresql://user:password@localhost/dbname"
          ]
        },
        "slack": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-slack"
          ],
          "env": {
            "SLACK_BOT_TOKEN": "YOUR_SLACK_BOT_TOKEN",
            "SLACK_TEAM_ID": "YOUR_SLACK_TEAM_ID"
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

| Config                             | Type   | Default              | Description                                  |
|------------------------------------|--------|----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | Enable ClawHub registry                      |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub base URL                             |
| `registries.clawhub.auth_token`    | string | `""`                 | Optional Bearer token for higher rate limits |
| `registries.clawhub.search_path`   | string | `/api/v1/search`     | Search API path                              |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`     | Skills API path                              |
| `registries.clawhub.download_path` | string | `/api/v1/download`   | Download API path                            |

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

Note: Nested map-style config (for example `tools.mcp.servers.<name>.*`) is configured in `config.json` rather than
environment variables.

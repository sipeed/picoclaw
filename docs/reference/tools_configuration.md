# Tools Configuration

PicoClaw's tools configuration is located in the `tools` field of `config.json`.

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

## Sensitive Data Filtering

Before tool results are sent to the LLM, PicoClaw can filter sensitive values (API keys, tokens, secrets) from the output. This prevents the LLM from seeing its own credentials.

See [Sensitive Data Filtering](../security/sensitive_data_filtering.md) for full documentation.

| Config | Type | Default | Description |
|--------|------|---------|-------------|
| `filter_sensitive_data` | bool | `true` | Enable/disable filtering |
| `filter_min_length` | int | `8` | Minimum content length to trigger filtering |

## Web Tools

Web tools are used for web search and fetching.

### Web Fetcher
General settings for fetching and processing webpage content.

| Config              | Type   | Default       | Description                                                                                   |
|---------------------|--------|---------------|-----------------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | Enable the webpage fetching capability.                                                       |
| `fetch_limit_bytes` | int    | 10485760      | Maximum size of the webpage payload to fetch, in bytes (default is 10MB).                     |
| `format`            | string | "plaintext"   | Output format of the fetched content. Options: `plaintext` or `markdown` (recommended).       |

### Brave

| Config        | Type     | Default | Description                                    |
|---------------|----------|---------|------------------------------------------------|
| `enabled`     | bool     | false   | Enable Brave search                            |
| `api_key`     | string   | -       | Brave Search API key                           |
| `api_keys`    | string[] | -       | Multiple API keys for rotation (takes priority over `api_key`) |
| `max_results` | int      | 5       | Maximum number of results                      |

### DuckDuckGo

| Config        | Type | Default | Description               |
|---------------|------|---------|---------------------------|
| `enabled`     | bool | true    | Enable DuckDuckGo search  |
| `max_results` | int  | 5       | Maximum number of results |

### Gemini Google Search

Gemini search uses Gemini with Google Search grounding. It returns an AI-synthesized answer with citations from Google Search.

| Config        | Type   | Default              | Description                       |
|---------------|--------|----------------------|-----------------------------------|
| `enabled`     | bool   | false                | Enable Gemini Google Search       |
| `api_key`     | string | -                    | Google Gemini API key             |
| `model`       | string | `gemini-2.5-flash`   | Gemini model used for search      |
| `max_results` | int    | 5                    | Maximum number of citations       |

```json
{
  "tools": {
    "web": {
      "gemini": {
        "enabled": true,
        "api_key": "YOUR_GEMINI_API_KEY",
        "model": "gemini-2.5-flash",
        "max_results": 5
      }
    }
  }
}
```

### Baidu Search

Baidu Search uses the [Qianfan AI Search API](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5), which is AI-powered and optimized for Chinese-language queries.

| Config        | Type   | Default                                                | Description               |
|---------------|--------|--------------------------------------------------------|---------------------------|
| `enabled`     | bool   | false                                                  | Enable Baidu Search       |
| `api_key`     | string | -                                                      | Qianfan API key           |
| `base_url`    | string | `https://qianfan.baidubce.com/v2/ai_search/web_search` | Baidu Search API URL      |
| `max_results` | int    | 5                                                      | Maximum number of results |

```json
{
  "tools": {
    "web": {
      "baidu_search": {
        "enabled": true,
        "api_key": "YOUR_BAIDU_QIANFAN_API_KEY",
        "max_results": 10
      }
    }
  }
}
```

### Perplexity

| Config        | Type     | Default | Description                                    |
|---------------|----------|---------|------------------------------------------------|
| `enabled`     | bool     | false   | Enable Perplexity search                       |
| `api_key`     | string   | -       | Perplexity API key                             |
| `api_keys`    | string[] | -       | Multiple API keys for rotation (takes priority over `api_key`) |
| `max_results` | int      | 5       | Maximum number of results                      |

### Tavily

| Config        | Type   | Default | Description               |
|---------------|--------|---------|---------------------------|
| `enabled`     | bool   | false   | Enable Tavily search      |
| `api_key`     | string | -       | Tavily API key            |
| `base_url`    | string | -       | Custom Tavily API base URL |
| `max_results` | int    | 5       | Maximum number of results |

### SearXNG

| Config        | Type   | Default                 | Description               |
|---------------|--------|-------------------------|---------------------------|
| `enabled`     | bool   | false                   | Enable SearXNG search     |
| `base_url`    | string | `http://localhost:8888` | SearXNG instance URL      |
| `max_results` | int    | 5                       | Maximum number of results |

### GLM Search

| Config          | Type   | Default                                           | Description               |
|-----------------|--------|---------------------------------------------------|---------------------------|
| `enabled`       | bool   | false                                             | Enable GLM Search         |
| `api_key`       | string | -                                                 | GLM API key               |
| `base_url`      | string | `https://open.bigmodel.cn/api/paas/v4/web_search` | GLM Search API URL        |
| `search_engine` | string | `search_std`                                      | Search engine type        |
| `max_results`   | int    | 5                                                 | Maximum number of results |

### Additional Web Settings

| Config                   | Type     | Default | Description                                                    |
|--------------------------|----------|---------|----------------------------------------------------------------|
| `prefer_native`          | bool     | true    | Prefer provider's native search over configured search engines |
| `private_host_whitelist` | string[] | `[]`    | Private/internal hosts allowed for web fetching                |

### `web_search` Tool Parameters

At runtime, the `web_search` tool accepts the following parameters:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | string | yes | Search query string |
| `count` | integer | no | Number of results to return. Default: `10`, max: `10` |
| `range` | string | no | Optional time filter: `d` (day), `w` (week), `m` (month), `y` (year) |

If `range` is omitted, PicoClaw performs an unrestricted search.

### Example `web_search` Call

```json
{
  "query": "ai agent news",
  "count": 10,
  "range": "w"
}
```

## Image Generation Tool

The `image_generate` tool creates image files through a provider that supports
image generation.

| Config | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | false | Enable the image generation tool |
| `model` | string | `gpt-image-2` | Image generation model. Values may include a provider prefix, for example `openai-codex/gpt-image-2` |

If `tools.image_generate.model` is not set, PicoClaw falls back to the legacy
`agents.defaults.image_model` setting, then to `gpt-image-2`.

```json
{
  "tools": {
    "image_generate": {
      "enabled": true,
      "model": "openai-codex/gpt-image-2"
    }
  }
}
```

## Exec Tool

The exec tool is used to execute shell commands.

| Config                 | Type  | Default | Description                                |
|------------------------|-------|---------|--------------------------------------------|
| `enabled`              | bool  | true    | Enable the exec tool                        |
| `enable_deny_patterns` | bool  | true    | Enable default dangerous command blocking  |
| `custom_deny_patterns` | array | []      | Custom deny patterns (regular expressions) |

### Disabling the Exec Tool

To completely disable the `exec` tool, set `enabled` to `false`:

**Via config file:**
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**Via environment variable:**
```bash
PICOCLAW_TOOLS_EXEC_ENABLED=false
```

> **Note:** When disabled, the agent will not be able to execute shell commands. This also affects the Cron tool's ability to run scheduled shell commands.

### Functionality

- **`enable_deny_patterns`**: Set to `false` to completely disable the default dangerous command blocking patterns
- **`custom_deny_patterns`**: Add custom deny regex patterns; commands matching these will be blocked

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

### Known Architectural Limitation

The exec guard only validates the top-level command sent to PicoClaw. It does **not** recursively inspect child
processes spawned by build tools or scripts after that command starts running.

Examples of workflows that can bypass the direct command guard once the initial command is allowed:

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

This means the guard is useful for blocking obviously dangerous direct commands, but it is **not** a full sandbox for
unreviewed build pipelines. If your threat model includes untrusted code in the workspace, use stronger isolation such
as containers, VMs, or an approval flow around build-and-run commands.

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
| `enabled`              | bool | true    | Register the agent-facing cron tool            |
| `allow_command`        | bool | true    | Allow command jobs without extra confirmation  |
| `exec_timeout_minutes` | int  | 5       | Execution timeout in minutes, 0 means no limit |

For schedule types, execution modes (`deliver`, agent turn, and command jobs), persistence, and the current command-security gates, see [Scheduled Tasks and Cron Jobs](cron.md).

## MCP Tool

The MCP tool enables integration with external Model Context Protocol servers.

If you prefer not to edit JSON manually, PicoClaw also provides an MCP configuration manager CLI:

- `picoclaw mcp add` — add or update a server (supports `--deferred` / `--no-deferred`)
- `picoclaw mcp list` — list all configured servers with status and deferred state
- `picoclaw mcp show <name>` — show full details and the tool list for one server
- `picoclaw mcp test <name>` — connectivity check for one server
- `picoclaw mcp remove <name>` — remove a server entry
- `picoclaw mcp edit` — open `config.json` in `$EDITOR` for advanced edits

These commands manage the same `tools.mcp.servers` section documented below. See [MCP Server CLI](mcp-cli.md) for command syntax, examples, and behavior details.

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
| `enabled`            | bool | false   | Global default: if `true`, all MCP tools are hidden and loaded on-demand via search; if `false`, all tools are loaded into context. Individual servers can override this with the per-server `deferred` field. |
| `ttl`                | int  | 5       | Number of conversational turns a discovered tool remains unlocked                                                                 |
| `max_search_results` | int  | 5       | Maximum number of tools returned per search query                                                                                 |
| `use_bm25`           | bool | true    | Enable the natural language/keyword search tool (`tool_search_tool_bm25`). **Warning**: consumes more resources than regex search |
| `use_regex`          | bool | false   | Enable the regex pattern search tool (`tool_search_tool_regex`)                                                                   |

> **Note:** If `discovery.enabled` is `true`, you MUST enable at least one search engine (`use_bm25` or `use_regex`),
> otherwise the application will fail to start.

### Per-Server Config

| Config     | Type    | Required | Description                                                                                                                                                     |
|------------|---------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `enabled`  | bool    | yes      | Enable this MCP server                                                                                                                                          |
| `deferred` | bool    | no       | Override deferred mode for this server only. `true` = tools are hidden and discoverable via search; `false` = tools are always visible in context. When omitted, the global `discovery.enabled` value applies. |
| `type`     | string  | no       | Transport type: `stdio`, `sse`, `http`                                                                                                                          |
| `command`  | string  | stdio    | Executable command for stdio transport                                                                                                                          |
| `args`     | array   | no       | Command arguments for stdio transport                                                                                                                           |
| `env`      | object  | no       | Environment variables for stdio process                                                                                                                         |
| `env_file` | string  | no       | Path to environment file for stdio process                                                                                                                      |
| `url`      | string  | sse/http | Endpoint URL for `sse`/`http` transport                                                                                                                         |
| `headers`  | object  | no       | HTTP headers for `sse`/`http` transport                                                                                                                         |

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
          "type": "slack",
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

#### 4) Mixed setup: per-server deferred override

*Discovery is enabled globally, but `filesystem` is pinned as always-visible while `context7` follows the global
default (deferred). `aws` explicitly opts in to deferred mode even though it is the same as the global default.*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true
      },
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
          "deferred": false
        },
        "context7": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@upstash/context7-mcp"]
        },
        "aws": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "aws-mcp-server"],
          "deferred": true
        }
      }
    }
  }
}
```

> **Tip:** `deferred` on a per-server basis is independent of `discovery.enabled`. You can keep
> `discovery.enabled: false` globally (all tools visible by default) and still mark individual
> high-volume servers as `"deferred": true` to avoid polluting the context with their tools.

## Skills Tool

The skills tool configures skill discovery and installation via registries like ClawHub and GitHub.

### Registries

| Config                             | Type   | Default              | Description                                  |
|------------------------------------|--------|----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | Enable ClawHub registry                      |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub base URL                             |
| `registries.clawhub.auth_token`    | string | `""`                 | Optional Bearer token for higher rate limits |
| `registries.clawhub.search_path`   | string | `""`                 | Search API path                              |
| `registries.clawhub.skills_path`   | string | `""`                 | Skills API path                              |
| `registries.clawhub.download_path` | string | `""`                 | Download API path                            |
| `registries.clawhub.timeout`       | int    | 0                    | Request timeout in seconds (0 = default)     |
| `registries.clawhub.max_zip_size`  | int    | 0                    | Max skill zip size in bytes (0 = default)    |
| `registries.clawhub.max_response_size` | int | 0                   | Max API response size in bytes (0 = default) |
| `registries.github.enabled`        | bool   | true                 | Enable GitHub installs via registry config    |
| `registries.github.base_url`       | string | `https://github.com` | GitHub or GitHub Enterprise base URL          |
| `registries.github.auth_token`     | string | `""`                | GitHub personal access token                  |
| `registries.github.proxy`          | string | `""`                | HTTP proxy for GitHub API requests            |

### Legacy GitHub Config

`github.*` is deprecated. Use `registries.github.*` instead. The legacy fields are still supported for compatibility and will be removed later.

| Config             | Type   | Default              | Description                    |
|--------------------|--------|----------------------|--------------------------------|
| `github.base_url`  | string | `https://github.com` | Deprecated GitHub base URL     |
| `github.proxy`     | string | `""`                | Deprecated GitHub proxy        |
| `github.token`     | string | `""`                | Deprecated GitHub token        |

### Search Settings

| Config                    | Type | Default | Description                                |
|---------------------------|------|---------|--------------------------------------------|
| `max_concurrent_searches` | int  | 2       | Max concurrent skill search requests       |
| `search_cache.max_size`   | int  | 50      | Max cached search results                  |
| `search_cache.ttl_seconds`| int  | 300     | Cache TTL in seconds                       |

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
	          "search_path": "",
	          "skills_path": "",
	          "download_path": "",
	          "timeout": 0,
	          "max_zip_size": 0,
	          "max_response_size": 0
        },
        "github": {
          "enabled": true,
          "base_url": "https://github.com",
	          "auth_token": "",
	          "proxy": ""
        }
      },
      "github": {
	        "base_url": "https://github.com",
        "proxy": "",
        "token": ""
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

## Environment Variables

All configuration options can be overridden via environment variables with the format `PICOCLAW_TOOLS_<SECTION>_<KEY>`:

For example:

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`
- `PICOCLAW_TOOLS_MCP_MAX_INLINE_TEXT_CHARS=16384`

Note: Nested map-style config (for example `tools.mcp.servers.<name>.*`) is configured in `config.json` rather than
environment variables.

For MCP tools, `tools.mcp.max_inline_text_chars` controls how much text result is kept inline in model context. The threshold is counted in Unicode characters (Go runes), not bytes. For example, `16384` means up to 16,384 characters inline, which may occupy more than 16 KB for multibyte text such as CJK. Above this threshold, PicoClaw saves the MCP text result as a local artifact in the agent workspace and gives the model a short note plus a structured `[file:...]` artifact path instead of injecting the full payload into context.

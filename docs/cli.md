# PicoClaw CLI Reference

## Install

```bash
go install github.com/nuestra-ai/picoclaw/cmd/picoclaw@latest
```

## Global Config

Create `~/.picoclaw/config.json` (or set `PICOCLAW_CONFIG` env var):

```jsonc
{
  "model_list": [
    {
      "model_name": "main",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-...",
      "api_base": "https://api.anthropic.com/v1"
    }
  ],
  "agents": {
    "defaults": {
      "workspace_root": "/data/workspaces",
      "model_name": "main",
      "max_tokens": 4096,
      "max_tool_iterations": 20
    }
  }
}
```

The global config is the base. Per-workspace configs overlay it (see [Workspace Config](#workspace-config)).

`workspace_root` defines the security boundary for all workspace and config-dir paths. When set, `--workspace` and `--config-dir` must be relative subdirectories within it. See [Path Security](#path-security) for details.

---

## Subcommands

| Command | Alias | Description |
|---------|-------|-------------|
| `picoclaw agent` | | Interact with the agent (interactive REPL or one-shot) |
| `picoclaw gateway` | `g` | Start the HTTP gateway server |
| `picoclaw auth` | | Manage authentication (login, logout, status, models) |
| `picoclaw cron` | `c` | Manage scheduled tasks |
| `picoclaw skills` | | Manage skills (list, install, remove, search) |
| `picoclaw status` | | Show current status |
| `picoclaw migrate` | | Migrate config/workspace from another installation |
| `picoclaw onboard` | | Interactive first-run setup |
| `picoclaw version` | | Print version |

---

## picoclaw agent

Interact with the agent directly. Without `-m`, starts an interactive REPL. With `-m`, sends a single message and exits.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--message` | `-m` | | Single message (non-interactive). Omit for interactive REPL. |
| `--session` | `-s` | `""` (→ `agent:main:cli:default`) | Session key for conversation isolation (e.g. `stackId:conversationId`). |
| `--model` | | | Override model name from config. |
| `--workspace` | | | Agent workspace directory, relative to `workspace_root`. Resolved to an absolute path before use. |
| `--config-dir` | | | Config directory (relative to `workspace_root`) containing `config.json` and bootstrap files (`AGENTS.md`, `IDENTITY.md`, `SOUL.md`, `USER.md`). |
| `--tools` | | | Comma-separated tool allowlist (e.g. `read_file,web_fetch`). Only these tools are enabled. |
| `--skills` | | | Comma-separated skill filter (e.g. `summarize,translate`). Only these skills are loaded. |
| `--debug` | `-d` | `false` | Enable debug logging. |

### Config Precedence (highest wins)

```
CLI flags (--model, --tools, --skills, --workspace)
  > config-dir/config.json
    > ~/.picoclaw/config.json
      > defaults
```

### Session Key Formatting

The `--session` value is automatically prefixed:

| Input | Resulting key |
|-------|--------------|
| _(empty)_ | `agent:main:cli:default` |
| `s1:c1` | `agent:main:cli:s1:c1` |
| `my-project` | `agent:main:cli:my-project` |
| `agent:custom:key` | `agent:custom:key` (used as-is) |

Session files are stored at `{workspace}/sessions/{sanitized_key}.json`.

### Examples

```bash
# Interactive mode with default session
picoclaw agent

# One-shot message
picoclaw agent -m "Hello, world"

# With workspace isolation (paths relative to workspace_root)
picoclaw agent -m "Summarize the report" \
  -s tenant1:conv42 \
  --workspace tenant1/conv42 \
  --config-dir tenant1/config

# Restricted tools, custom model
picoclaw agent -m "Search the web for recent news" \
  --tools web,web_fetch \
  --model gpt-5.2

# Debug mode to see session key, model, and iteration details
picoclaw agent -d -m "Hello" -s test
```

> **Note**: When `workspace_root` is set in the global config, `--workspace` and `--config-dir` must be relative paths (e.g. `tenant1/conv42`, not `/data/workspaces/tenant1/conv42`). When `workspace_root` is not set, paths fall back to `filepath.Abs` resolution for backward compatibility.

---

## picoclaw gateway

Start the HTTP gateway server. Channels (Telegram, Discord, MagicForm, etc.) receive messages via webhooks and respond asynchronously.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--debug` | `-d` | `false` | Enable debug logging. |

```bash
picoclaw gateway
# or with debug logging:
picoclaw gateway -d
```

Listens on `{gateway.host}:{gateway.port}` from the config (default `127.0.0.1:18790`).

---

## picoclaw auth

Manage provider authentication.

### picoclaw auth login

```bash
picoclaw auth login -p <provider>
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--provider` | `-p` | _(required)_ | Provider: `openai`, `anthropic`, `google-antigravity` |
| `--device-code` | | `false` | Use device code flow for headless environments (OpenAI) |
| `--setup-token` | | `false` | Use setup-token flow for Anthropic (from `claude setup-token`) |

**Anthropic login** prompts to choose between:
1. **Setup token** (recommended) — paste a token from `claude setup-token`
2. **API key** — paste an `sk-ant-...` key from console.anthropic.com

Use `--setup-token` to skip the prompt and go directly to option 1.

**OpenAI login** uses browser-based OAuth by default, or `--device-code` for headless environments.

**Google Antigravity login** uses browser-based OAuth with PKCE. Also fetches user email and Cloud Code Assist project ID.

### picoclaw auth logout

```bash
picoclaw auth logout [-p <provider>]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--provider` | `-p` | `""` (all) | Provider to logout from. Empty = logout from all. |

### picoclaw auth status

```bash
picoclaw auth status
```

Shows all authenticated providers with method, status, account info, and expiry. For Anthropic OAuth credentials, also displays 5-hour and 7-day usage percentages.

### picoclaw auth models

```bash
picoclaw auth models
```

Lists available models for Google Antigravity (requires prior login). Shows model ID, display name, and quota status.

---

## picoclaw cron

Manage scheduled tasks.

### picoclaw cron list

```bash
picoclaw cron list
```

### picoclaw cron add

```bash
picoclaw cron add -n <name> -m <message> (--every <seconds> | --cron <expr>)
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | _(required)_ | Job name |
| `--message` | `-m` | _(required)_ | Message for the agent |
| `--every` | `-e` | | Run every N seconds |
| `--cron` | `-c` | | Cron expression (e.g. `0 9 * * *`) |
| `--deliver` | `-d` | `false` | Deliver response to a channel |
| `--channel` | | | Channel for delivery |
| `--to` | | | Recipient for delivery |

`--every` and `--cron` are mutually exclusive; one is required.

### picoclaw cron remove / enable / disable

```bash
picoclaw cron remove <job-id>
picoclaw cron enable <job-id>
picoclaw cron disable <job-id>
```

---

## picoclaw skills

Manage skills (install, remove, list, search).

```bash
picoclaw skills list              # List installed skills
picoclaw skills list-builtin      # List built-in skills
picoclaw skills install <url>     # Install from URL or local path
picoclaw skills install-builtin   # Install built-in skills to workspace
picoclaw skills remove <name>     # Remove an installed skill
picoclaw skills search <query>    # Search skill registries
picoclaw skills show <name>       # Show skill details
```

The `install` command also supports `--registry <name> <slug>` to install from a named registry.

---

## picoclaw migrate

Migrate config and workspace from another installation (e.g. OpenClaw).

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | `openclaw` | Source to migrate from |
| `--dry-run` | `false` | Preview changes without applying |
| `--refresh` | `false` | Re-run migration (overwrite existing) |
| `--config-only` | `false` | Migrate config only |
| `--workspace-only` | `false` | Migrate workspace only |
| `--force` | `false` | Overwrite existing files |
| `--source-home` | | Custom source home directory |
| `--target-home` | | Custom target home directory |

---

## Workspace Config

A workspace-level `config.json` (placed in the config directory) overlays the global config. Only these fields are honored:

| Field | Behavior | Required? |
|-------|----------|-----------|
| `model_list` | Replaces global model_list | Yes, if using a different API key |
| `agents.defaults` | Merges non-zero fields (model_name, max_tokens, temperature, etc.) | No |
| `agents.list` | Replaces global agents list | No |
| `tools` | Merges only keys present in the file (unmentioned tools are not affected) | No |
| `session` | Merges non-zero fields (dm_scope, identity_links) | No |
| `bindings` | Replaces global bindings | No |

**Not honored** (infrastructure-level): `gateway`, `heartbeat`, `devices`, `providers`, `channels`.

**Protected fields**: `workspace_root` cannot be overridden by a workspace config overlay — the boundary is always set by the base (global) config. If a workspace overlay sets a `workspace` value, it is validated against the base config's `workspace_root`; traversal attempts (e.g. `../../escape`) cause a hard error.

Example `config.json`:

```json
{
  "model_list": [
    {
      "model_name": "main",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-tenant-key",
      "api_base": "https://api.anthropic.com/v1"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "main",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "tools": {
    "exec": { "enabled": false }
  }
}
```

---

## Directory Layout

PicoClaw uses the following workspace directory structure for per-tenant/per-conversation isolation:

```
{workspace_root}/
  {stackId}/
    config/                    # configDir -- shared per-stack
      config.json              # API key, model, agent settings
      AGENTS.md                # Agent instructions (optional)
      IDENTITY.md              # Agent identity (optional)
      SOUL.md                  # Agent personality (optional)
      USER.md                  # User context (optional)
    {conversationId}/          # workspace -- per-conversation
      sessions/                # Conversation history (managed by PicoClaw)
      memory/                  # Persistent agent memory (managed by PicoClaw)
      skills/                  # Workspace-local skills (optional)
```

The config directory contains shared settings (API keys, bootstrap files) for all conversations in a stack. Each conversation gets its own workspace directory with isolated sessions and memory.

---

## Tool Names Reference

For `--tools` (CLI) or `allowedTools` (webhook API):

| Tool name | Description |
|-----------|-------------|
| `read_file` | Read files from workspace |
| `write_file` | Write files to workspace |
| `edit_file` | Edit files in workspace |
| `append_file` | Append to files in workspace |
| `list_dir` | List directory contents |
| `exec` | Execute shell commands |
| `spawn` | Spawn background processes |
| `cron` | Schedule recurring tasks |
| `web` | Web search (DuckDuckGo, Brave, etc.) |
| `web_fetch` | Fetch and parse web pages |
| `skills` | Run installed skills |
| `find_skills` | Search skill registries |
| `install_skill` | Install skills from registry |
| `subagent` | Spawn sub-agents |
| `message` | Send messages to channels |
| `mcp` | Model Context Protocol tools |
| `i2c` | I2C hardware bus (Linux only) |
| `spi` | SPI hardware bus (Linux only) |

---

## Path Security

When `workspace_root` is set in `agents.defaults`, all workspace and config-dir paths are validated as relative subdirectories of that root. This applies uniformly across the CLI (`--workspace`, `--config-dir`), the gateway (webhook `workspace`/`configDir` fields), and workspace config overlays.

**Validation rules** (when `workspace_root` is set):

- **Absolute paths are rejected** — use `tenant1/conv42`, not `/data/workspaces/tenant1/conv42`.
- **Directory traversal is rejected** — `../escape`, `a/../../etc`, `foo/..` all fail.
- **Empty string and bare `.` are rejected** — the path must pick a subdirectory, not root itself.
- **Relative paths are joined to `workspace_root`** — `tenant1/conv42` becomes `/data/workspaces/tenant1/conv42`.
- **A post-join boundary check** confirms the resolved path stays within `workspace_root`.

**Without `workspace_root`**: paths fall back to `filepath.Abs` resolution for backward compatibility, but traversal (`..`) is still rejected.

**Enforcement points**:

1. **CLI** — `--workspace` and `--config-dir` are validated before any workspace config overlay is loaded.
2. **Gateway** — webhook `workspace` and `configDir` fields are validated by each channel handler (e.g. MagicForm).
3. **Agent loop** — defense-in-depth re-validation of metadata-driven workspace overrides.
4. **Config overlay** — `workspace_root` cannot be overridden; `workspace` values from overlays are validated against the base config boundary.

---

## Troubleshooting

**`--workspace` or `--config-dir` rejected**
- When `workspace_root` is set, paths must be relative subdirectories (e.g. `s1/c1`). Absolute paths, `..` traversal, and empty/`.` paths are rejected.
- If `workspace_root` is not set and you get a traversal error, the path contains `..` which is always blocked.

**Workspace config overlay rejected**
- A workspace `config.json` that sets `agents.defaults.workspace` to a path escaping `workspace_root` will cause a hard error during config merge. Fix the overlay's `workspace` value.

**Workspace config ignored**
- Verify `config.json` exists in the `--config-dir` path.
- Check that the JSON is valid (`picoclaw agent -d` shows parse errors).
- Only allowed fields are merged. `gateway`, `heartbeat`, `devices`, `providers` are ignored.

**Tools unexpectedly disabled**
- Workspace `config.json` only affects tools explicitly mentioned. If you set `{"tools": {"exec": {"enabled": false}}}`, only `exec` is disabled; all other tools keep their global config values.

**Session not persisting**
- Ensure the same `--workspace` path is used for the same conversation.
- Sessions are stored at `{workspace}/sessions/`. Different workspace paths = different sessions.

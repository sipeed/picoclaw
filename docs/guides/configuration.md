# ⚙️ Configuration Guide

> Back to [README](../README.md)

## ⚙️ Configuration

Config file: `~/.picoclaw/config.json`

> **Security Configuration:** For storing API keys, tokens, and other sensitive data, see the [Security Configuration Guide](../security/security_configuration.md).

### Environment Variables

You can override default paths using environment variables. This is useful for portable installations, containerized deployments, or running picoclaw as a system service. These variables are independent and control different paths.

| Variable          | Description                                                                                                                             | Default Path              |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Overrides the path to the configuration file. This directly tells picoclaw which `config.json` to load, ignoring all other locations. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Overrides the root directory for picoclaw data. This changes the default location of the `workspace` and other data directories.          | `~/.picoclaw`             |

**Examples:**

```bash
# Run picoclaw using a specific config file
# The workspace path will be read from within that config file
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Run picoclaw with all its data stored in /opt/picoclaw
# Config will be loaded from the default ~/.picoclaw/config.json
# Workspace will be created at /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Use both for a fully customized setup
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Gateway Log Level

`gateway.log_level` controls Gateway log verbosity and is configurable in `config.json`.

```json
{
  "gateway": {
    "log_level": "warn"
  }
}
```

When omitted, the default is `warn`. Supported values: `debug`, `info`, `warn`, `error`, `fatal`.

You can also override this with the environment variable `PICOCLAW_LOG_LEVEL`.

### Workspace Layout

PicoClaw stores data in your configured workspace (default: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Conversation sessions and history
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state (last channel, etc.)
├── cron/             # Scheduled jobs database
├── skills/           # Custom skills
├── AGENT.md          # Agent behavior guide
├── HEARTBEAT.md      # Periodic task prompts (checked every 30 min)
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent soul
└── USER.md           # User preferences
```

> **Note:** Changes to `AGENT.md`, `SOUL.md`, `USER.md` and `memory/MEMORY.md` are automatically detected at runtime via file modification time (mtime) tracking. You do **not** need to restart the gateway after editing these files — the agent picks up the new content on the next request.

### Web launcher dashboard

**picoclaw-launcher** serves a browser UI that requires password sign-in first. On first run, open `/launcher-setup` to create the dashboard password. Later manual sign-ins use `/launcher-login`.

- **Config file**: Same directory as `config.json` (or the file pointed to by `PICOCLAW_CONFIG`). The launcher-specific file is `launcher-config.json`.
- **Password storage**: On supported platforms, the password is stored as a bcrypt hash in `launcher-auth.db`. On platforms where the SQLite password store is unavailable, the bcrypt hash is stored in `launcher-config.json`.
- **Legacy migration**: Older `launcher_token` values are migrated once into password login and removed from saved launcher config.
- **Local auto-login**: When the launcher auto-opens a local browser after startup, it uses a one-shot loopback-only bootstrap endpoint to set the session cookie automatically.
- **Unsupported auth paths**: URL token login (`?token=...`), `PICOCLAW_LAUNCHER_TOKEN`, and `Authorization: Bearer` dashboard auth are no longer supported.
- **Sign-out**: Use **`POST /api/auth/logout`** with **`Content-Type: application/json`** (body may be `{}`). Do not rely on a GET URL for logout (CSRF-safe pattern).
- **Brute-force**: **`POST /api/auth/login`** is **rate-limited per client IP per minute** (HTTP 429 when exceeded).
- **Session lifetime**: The HttpOnly session cookie lasts about **31 days** by default, but sessions are invalidated when the launcher process restarts.

### Skill Sources

By default, skills are loaded from:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<binary-embedded-path>/skills` (builtin, set at build time)

For advanced/test setups, you can override the builtin skills root with:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Using Skills From Chat Channels

Once skills are installed, you can inspect and force them directly from a chat channel:

- `/list skills` shows the installed skill names available to the current agent.
- `/use <skill> <message>` forces a specific skill for a single request.
- `/use <skill>` arms that skill for your next message in the same chat session.
- `/use clear` cancels a pending skill override created by `/use <skill>`.
- `/btw <question>` asks an immediate side question without changing the current session history. `/btw` is handled as a no-tool query and does not enter the normal tool-execution flow.

Examples:

```text
/list skills
/use git explain how to squash the last 3 commits
/btw remind me what we already decided about the deploy plan
/use italiapersonalfinance
dammi le ultime news
```

### Unified Command Execution Policy

- Generic slash commands are executed through a single path in `pkg/agent/loop.go` via `commands.Executor`.
- Channel adapters no longer consume generic commands locally; they forward inbound text to the bus/agent path. Telegram still auto-registers supported commands such as `/start`, `/help`, `/show`, `/list`, `/use`, and `/btw` at startup.
- Unknown slash command (for example `/foo`) passes through to normal LLM processing.
- Registered but unsupported command on the current channel (for example `/show` on WhatsApp) returns an explicit user-facing error and stops further processing.

### Session Isolation

Session scope controls how much memory is shared between chats, users, threads, and spaces.

- Use `session.dimensions` for the global default.
- Use `session_dimensions` on a dispatch rule for one routed exception.

For step-by-step recipes and isolation patterns, see the [Session Guide](session-guide.md).

### Routing

Routing is configured through `agents.dispatch.rules`.

Each rule matches against the normalized inbound context produced by channels.
Rules are evaluated from top to bottom. The first matching rule wins. If no
rule matches, PicoClaw falls back to the configured default agent.

Supported match fields:

* `channel`
* `account`
* `space`
* `chat`
* `topic`
* `sender`
* `mentioned`

Match values use the same scope vocabulary as the session system:

* `space`: `workspace:t001`, `guild:123456`
* `chat`: `direct:user123`, `group:-100123`, `channel:c123`
* `topic`: `topic:42`
* `sender`: a normalized sender identifier for the platform

Rules may optionally override the global `session.dimensions` value through
`session_dimensions`. This allows routing and session allocation to stay aligned
without reintroducing the old `bindings` or `dm_scope` formats.

Example:

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" },
      { "id": "sales" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "vip in support group",
          "agent": "sales",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890",
            "sender": "12345"
          },
          "session_dimensions": ["chat", "sender"]
        },
        {
          "name": "telegram support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

In the example above, the VIP rule must appear before the broader group rule.
Because routing is strictly ordered, more specific rules should be placed
earlier and broader fallback rules later.

For more complete routing and model-tier examples, see the [Routing Guide](routing-guide.md).

### 🔒 Security Sandbox

PicoClaw runs in a sandboxed environment by default. The agent can only access files and execute commands within the configured workspace.

#### Default Configuration

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

| Option                  | Default                 | Description                               |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Working directory for the agent           |
| `restrict_to_workspace` | `true`                  | Restrict file/command access to workspace |

#### Protected Tools

When `restrict_to_workspace: true`, the following tools are sandboxed:

| Tool          | Function         | Restriction                            |
| ------------- | ---------------- | -------------------------------------- |
| `read_file`   | Read files       | Only files within workspace            |
| `write_file`  | Write files      | Only files within workspace            |
| `list_dir`    | List directories | Only directories within workspace      |
| `edit_file`   | Edit files       | Only files within workspace            |
| `append_file` | Append to files  | Only files within workspace            |
| `exec`        | Execute commands | Command paths must be within workspace |

#### Additional Exec Protection

Even with `restrict_to_workspace: false`, the `exec` tool blocks these dangerous commands:

* `rm -rf`, `del /f`, `rmdir /s` — Bulk deletion
* `format`, `mkfs`, `diskpart` — Disk formatting
* `dd if=` — Disk imaging
* Writing to `/dev/sd[a-z]` — Direct disk writes
* `shutdown`, `reboot`, `poweroff` — System shutdown
* Fork bomb `:(){ :|:& };:`

### File Access Control

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.allow_read_paths` | string[] | `[]` | Additional paths allowed for reading outside workspace |
| `tools.allow_write_paths` | string[] | `[]` | Additional paths allowed for writing outside workspace |

### Read File Mode

`read_file` has two mutually exclusive implementations selected by config. PicoClaw registers exactly one of them at startup:

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.read_file.enabled` | bool | `true` | Enables the `read_file` tool |
| `tools.read_file.mode` | string | `bytes` | Selects the `read_file` implementation: `bytes` or `lines` |
| `tools.read_file.max_read_file_size` | int | `65536` | Maximum bytes returned by `read_file` |

#### Mode: `bytes`

Optimized for arbitrary files and binary-safe pagination.

Parameters:

* `path` (required): File path
* `offset` (optional): Starting byte offset, default `0`
* `length` (optional): Maximum number of bytes to read, default `max_read_file_size`

Use `bytes` when:

* You may read binary files
* You want deterministic byte-range pagination

#### Mode: `lines`

Text-oriented behavior, optimized for source files, markdown, logs, and configs. The tool reads sequentially by line and stops when the configured byte budget is reached.

Parameters:

* `path` (required): File path
* `start_line` (optional): Starting line number, 1-indexed and inclusive, default `1`
* `max_lines` (optional): Maximum number of lines to read, default = all remaining lines until EOF or byte budget

Behavior notes:

* Binary-looking files are rejected with guidance to switch `read_file` to `mode = bytes`
* Extremely long single lines are truncated rather than skipped

Use `mode = lines` when:

* The agent mostly reads text files
* You want line-based pagination in prompts and tool calls
* You want cleaner chunks for code review, logs, and documentation

#### Example

```json
{
  "tools": {
    "read_file": {
      "enabled": true,
       "mode": "lines",
      "max_read_file_size": 65536
    }
  }
}
```

### Exec Security

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Allow exec tool from remote channels (Telegram/Discord etc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Enable dangerous command interception |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Custom regex patterns to block |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Custom regex patterns to allow |

> **Security Note:** Symlink protection is enabled by default — all file paths are resolved through `filepath.EvalSymlinks` before whitelist matching, preventing symlink escape attacks.

#### Known Limitation: Child Processes From Build Tools

The exec safety guard only inspects the command line PicoClaw launches directly. It does not recursively inspect child
processes spawned by allowed developer tools such as `make`, `go run`, `cargo`, `npm run`, or custom build scripts.

That means a top-level command can still compile or launch other binaries after it passes the initial guard check. In
practice, treat build scripts, Makefiles, package scripts, and generated binaries as executable code that needs the same
level of review as a direct shell command.

For higher-risk environments:

* Review build scripts before execution.
* Prefer approval/manual review for compile-and-run workflows.
* Run PicoClaw inside a container or VM if you need stronger isolation than the built-in guard provides.

#### Error Examples

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Disabling Restrictions (Security Risk)

If you need the agent to access paths outside the workspace:

**Method 1: Config file**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Method 2: Environment variable**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Warning**: Disabling this restriction allows the agent to access any path on your system. Use with caution in controlled environments only.

#### Security Boundary Consistency

The `restrict_to_workspace` setting applies consistently across all execution paths:

| Execution Path   | Security Boundary            |
| ---------------- | ---------------------------- |
| Main Agent       | `restrict_to_workspace` ✅   |
| Subagent / Spawn | Inherits same restriction ✅ |
| Heartbeat tasks  | Inherits same restriction ✅ |

All paths share the same workspace restriction — there's no way to bypass the security boundary through subagents or scheduled tasks.

### Heartbeat (Periodic Tasks)

PicoClaw can perform periodic tasks automatically. Create a `HEARTBEAT.md` file in your workspace:

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast
```

The agent will read this file every 30 minutes (configurable) and execute any tasks using available tools.

#### Async Tasks with Spawn

For long-running tasks (web search, API calls), use the `spawn` tool to create a **subagent**:

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)

- Report current time

## Long Tasks (use spawn for async)

- Search the web for AI news and summarize
- Check email and report important messages
```

**Key behaviors:**

| Feature                 | Description                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Creates async subagent, doesn't block heartbeat           |
| **Independent context** | Subagent has its own context, no session history          |
| **message tool**        | Subagent communicates with user directly via message tool |
| **Non-blocking**        | After spawning, heartbeat continues to next task          |

#### How Subagent Communication Works

```
Heartbeat triggers
    ↓
Agent reads HEARTBEAT.md
    ↓
For long task: spawn subagent
    ↓                           ↓
Continue to next task      Subagent works independently
    ↓                           ↓
All tasks done            Subagent uses "message" tool
    ↓                           ↓
Respond HEARTBEAT_OK      User receives result directly
```

The subagent has access to tools (message, web_search, etc.) and can communicate with the user independently without going through the main agent.

**Configuration:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `enabled`  | `true`  | Enable/disable heartbeat           |
| `interval` | `30`    | Check interval in minutes (min: 5) |

**Environment variables:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` to disable
* `PICOCLAW_HEARTBEAT_INTERVAL=60` to change interval

### Providers

> [!NOTE]
> Groq provides free voice transcription via Whisper. If configured, audio messages from any channel will be automatically transcribed at the agent level.

| Provider     | Purpose                                 | Get API Key                                                  |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM (Gemini direct)                     | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM (Zhipu direct)                      | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM (Volcengine direct)                 | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM (recommended, access to all models) | [openrouter.ai](https://openrouter.ai)                       |
| `anthropic`  | LLM (Claude direct)                     | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM (GPT direct)                        | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM (DeepSeek direct)                   | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM (Qwen direct)                       | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **Voice transcription** (Whisper) | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM (Cerebras direct)                   | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM (Vivgrid direct)                    | [vivgrid.com](https://vivgrid.com)                           |

### Model Configuration (model_list)

> **What's New?** PicoClaw now uses a **model-centric** configuration approach. Simply specify `vendor/model` format (e.g., `zhipu/glm-4.7`) to add new providers — **zero code changes required!**

This design also enables **multi-agent support** with flexible provider selection:

- **Different agents, different providers**: Each agent can use its own LLM provider
- **Model fallbacks**: Configure primary and fallback models for resilience
- **Load balancing**: Distribute requests across multiple endpoints or keys
- **Centralized configuration**: Manage all providers in one place
- **Model enable/disable**: Use the `enabled` field to temporarily disable a model without removing its configuration

#### 🔒 Security Configuration (Recommended)

PicoClaw supports separating sensitive data (API keys, tokens, secrets) from your main configuration by storing them in a `.security.yml` file.

**Key Benefits:**
- **Security**: Sensitive data is never in your main config file
- **Easy sharing**: Share config.json without exposing API keys
- **Version control**: Add `.security.yml` to `.gitignore`
- **Flexible deployment**: Different environments can use different security files

**Quick Setup:**

1. Create `~/.picoclaw/.security.yml` with your API keys:
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-your-actual-openai-key"
  claude-sonnet-4.6:
    api_keys:
      - "sk-ant-your-actual-anthropic-key"
channels:
  telegram:
    token: "your-telegram-bot-token"
web:
  brave:
    api_keys:
      - "BSAyour-brave-api-key"
  glm_search:
    api_key: "your-glm-search-api-key"
```

2. Set proper permissions:
```bash
chmod 600 ~/.picoclaw/.security.yml
```

3. Remove sensitive fields from `config.json` (recommended):
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4"
      // api_key loaded from .security.yml
    }
  ],
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      // token loaded from .security.yml
    }
  }
}
```

**How it works:**
- Values from `.security.yml` are automatically mapped to config fields
- No special syntax needed — just omit sensitive fields from config.json
- If a field exists in both files, `.security.yml` value takes precedence
- You can mix direct values in config.json with security values

For complete documentation, see [`../security/security_configuration.md`](../security/security_configuration.md).

#### All Supported Vendors

| Vendor                  | `model` Prefix    | Default API Base                                    | Protocol  | API Key                                                          |
| ----------------------- | ----------------- | --------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| **OpenAI**              | `openai/`         | `https://api.openai.com/v1`                         | OpenAI    | [Get Key](https://platform.openai.com)                           |
| **Anthropic**           | `anthropic/`      | `https://api.anthropic.com/v1`                      | Anthropic | [Get Key](https://console.anthropic.com)                         |
| **智谱 AI (GLM)**       | `zhipu/`          | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Get Key](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**            | `deepseek/`       | `https://api.deepseek.com/v1`                       | OpenAI    | [Get Key](https://platform.deepseek.com)                         |
| **Google Gemini**       | `gemini/`         | `https://generativelanguage.googleapis.com/v1beta`  | Gemini    | [Get Key](https://aistudio.google.com/api-keys)                  |
| **Groq**                | `groq/`           | `https://api.groq.com/openai/v1`                    | OpenAI    | [Get Key](https://console.groq.com)                              |
| **Moonshot**            | `moonshot/`       | `https://api.moonshot.cn/v1`                        | OpenAI    | [Get Key](https://platform.moonshot.cn)                          |
| **通义千问 (Qwen)**     | `qwen/`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Get Key](https://dashscope.console.aliyun.com)                  |
| **NVIDIA**              | `nvidia/`         | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [Get Key](https://build.nvidia.com)                              |
| **Ollama**              | `ollama/`         | `http://localhost:11434/v1`                         | OpenAI    | Local (no key needed)                                            |
| **LM Studio**           | `lmstudio/`       | `http://localhost:1234/v1`                          | OpenAI    | Optional (local default: no key)                                 |
| **OpenRouter**          | `openrouter/`     | `https://openrouter.ai/api/v1`                      | OpenAI    | [Get Key](https://openrouter.ai/keys)                            |
| **LiteLLM Proxy**       | `litellm/`        | `http://localhost:4000/v1`                          | OpenAI    | Your LiteLLM proxy key                                           |
| **VLLM**                | `vllm/`           | `http://localhost:8000/v1`                          | OpenAI    | Local                                                            |
| **Cerebras**            | `cerebras/`       | `https://api.cerebras.ai/v1`                        | OpenAI    | [Get Key](https://cerebras.ai)                                   |
| **VolcEngine (Doubao)** | `volcengine/`     | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Get Key](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **神算云**              | `shengsuanyun/`   | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | —                                                                |
| **BytePlus**            | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [Get Key](https://www.byteplus.com)                              |
| **Vivgrid**             | `vivgrid/`        | `https://api.vivgrid.com/v1`                        | OpenAI    | [Get Key](https://vivgrid.com)                                   |
| **LongCat**             | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [Get Key](https://longcat.chat/platform)                         |
| **ModelScope (魔搭)**   | `modelscope/`     | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [Get Token](https://modelscope.cn/my/tokens)                     |
| **Antigravity**         | `antigravity/`    | Google Cloud                                        | Custom    | OAuth only                                                       |
| **GitHub Copilot**      | `github-copilot/` | `localhost:4321`                                    | gRPC      | —                                                                |

#### Basic Configuration

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_keys": ["sk-your-api-key"]
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_keys": ["sk-your-openai-key"]
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_keys": ["sk-ant-your-key"]
    },
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_keys": ["your-zhipu-key"]
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt-5.4"
    }
  }
}
```

> **Security Note**: You can remove `api_keys` fields from your config and store them in `.security.yml` instead. See [Security Configuration](#-security-configuration-recommended) above for details.
>
> **Note**: The `enabled` field can be set to `false` to disable a model entry without removing it. When omitted, it defaults to `true` during migration for models that have API keys.

#### Vendor-Specific Examples

> **Tip**: You can omit `api_key` fields and store them in `.security.yml` for better security. See [Security Configuration](#-security-configuration-recommended).

<details>
<summary><b>OpenAI</b></summary>

```json
{
  "model_name": "gpt-5.4",
  "model": "openai/gpt-5.4"
  // api_key: set in .security.yml
}
```

</details>

<details>
<summary><b>VolcEngine (Doubao)</b></summary>

```json
{
  "model_name": "ark-code-latest",
  "model": "volcengine/ark-code-latest"
  // api_key: set in .security.yml
}
```

</details>

<details>
<summary><b>智谱 AI (GLM)</b></summary>

```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7"
  // api_key: set in .security.yml
}
```

</details>

<details>
<summary><b>DeepSeek</b></summary>

```json
{
  "model_name": "deepseek-chat",
  "model": "deepseek/deepseek-chat"
  // api_key: set in .security.yml
}
```

</details>

<details>
<summary><b>Anthropic</b></summary>

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6"
  // api_key: set in .security.yml
}
```

> Run `picoclaw auth login --provider anthropic` to paste your API token.

For direct Anthropic API access or custom endpoints that only support Anthropic's native message format:

```json
{
  "model_name": "claude-opus-4-6",
  "model": "anthropic-messages/claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> Use `anthropic-messages` when the endpoint requires Anthropic's native `/v1/messages` format instead of OpenAI-compatible `/v1/chat/completions`.

</details>

<details>
<summary><b>Ollama (local)</b></summary>

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

</details>

<details>
<summary><b>LM Studio (local)</b></summary>

```json
{
  "model_name": "lmstudio-local",
  "model": "lmstudio/openai/gpt-oss-20b"
}
```

`api_base` defaults to `http://localhost:1234/v1`. API key is optional unless your LM Studio server enables authentication.<br/>
PicoClaw sends OpenAI-compatible requests to LM Studio, and strips the `lmstudio/` prefix before sending requests, so `lmstudio/openai/gpt-oss-20b` sends `openai/gpt-oss-20b` to the LM Studio server.

</details>

<details>
<summary><b>Custom Proxy / LiteLLM</b></summary>

```json
{
  "model_name": "my-custom-model",
  "model": "openai/custom-model",
  "api_base": "https://my-proxy.com/v1"
  // api_key: set in .security.yml
}
```

PicoClaw strips only the outer `litellm/` prefix before sending the request, so `litellm/lite-gpt4` sends `lite-gpt4`, while `litellm/openai/gpt-4o` sends `openai/gpt-4o`.

</details>

#### Load Balancing

Configure multiple endpoints for the same model name — PicoClaw will automatically round-robin between them:

**Option 1: Multiple API Keys in .security.yml (Recommended)**

```yaml
# .security.yml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-key-1"
      - "sk-proj-key-2"
```

```json
// config.json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1"
      // api_keys loaded from .security.yml
    }
  ]
}
```

**Option 2: Multiple Model Entries**

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_keys": ["sk-key1"]
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_keys": ["sk-key2"]
    }
  ]
}
```

#### Migration from Legacy `providers` Config

The old `providers` configuration is **deprecated** and has been removed in V2. Existing V0/V1 configs are auto-migrated. See [docs/migration/model-list-migration.md](../migration/model-list-migration.md) for the full guide.

### Provider Architecture

PicoClaw routes providers by protocol family:

- **OpenAI-compatible**: OpenRouter, Groq, Zhipu, vLLM-style endpoints, and most others.
- **Gemini native**: Google Gemini via the native `models/*:generateContent` and `models/*:streamGenerateContent` endpoints.
- **Anthropic**: Claude-native API behavior.
- **Codex/OAuth**: OpenAI OAuth/token authentication route.

This keeps the runtime lightweight while making new OpenAI-compatible backends mostly a config operation (`api_base` + `api_keys`).

<details>
<summary><b>Zhipu (legacy providers format)</b></summary>

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "max_parallel_turns": 1
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

> **Note**: The `providers` format is deprecated. Use the new `model_list` format with `.security.yml` for better security.
>
> **`max_parallel_turns`**: Controls concurrent processing of messages from different sessions. `1` (default) = sequential; `>1` = parallel. Messages from the same session are always serialized. See [Steering docs](../architecture/steering.md) for details.

</details>

<details>
<summary><b>Full config example</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "session": {
    "dm_scope": "per-channel-peer",
    "backlog_limit": 20
  },
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      // token: set in .security.yml
      "allow_from": ["123456789"]
    }
  },
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

> **Note**: Sensitive fields (`api_key`, `token`, etc.) can be omitted and stored in `.security.yml` for better security.

</details>

### Scheduled Tasks / Reminders

PicoClaw supports cron-style scheduled tasks via the `cron` tool. The agent can set, list, and cancel reminders or recurring jobs that trigger at specified times.

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5
    }
  }
}
```

Scheduled tasks persist across restarts and are stored in `~/.picoclaw/workspace/cron/`.

### Advanced Topics

| Topic | Description |
| ----- | ----------- |
| [Security Configuration](../security/security_configuration.md) | Store API keys and secrets in separate `.security.yml` file |
| [Sensitive Data Filtering](../security/sensitive_data_filtering.md) | Filter API keys and tokens from tool results before sending to LLM |
| [Hook System](../architecture/hooks/README.md) | Event-driven hooks: observers, interceptors, approval hooks |
| [Steering](../architecture/steering.md) | Inject messages into a running agent loop between tool calls |
| [SubTurn](../architecture/subturn.md) | Subagent coordination, concurrency control, lifecycle |
| [Context Management](../architecture/agent-refactor/context.md) | Context boundary detection, proactive budget check, compression |

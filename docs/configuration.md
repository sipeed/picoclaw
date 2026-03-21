# ⚙️ Configuration Guide

> Back to [README](../README.md)

## ⚙️ Configuration

Config file: `~/.piconomous/config.json`

### Environment Variables

You can override default paths using environment variables. This is useful for portable installations, containerized deployments, or running piconomous as a system service. These variables are independent and control different paths.

| Variable          | Description                                                                                                                             | Default Path              |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICONOMOUS_CONFIG` | Overrides the path to the configuration file. This directly tells piconomous which `config.json` to load, ignoring all other locations. | `~/.piconomous/config.json` |
| `PICONOMOUS_HOME`   | Overrides the root directory for piconomous data. This changes the default location of the `workspace` and other data directories.          | `~/.piconomous`             |

**Examples:**

```bash
# Run piconomous using a specific config file
# The workspace path will be read from within that config file
PICONOMOUS_CONFIG=/etc/piconomous/production.json piconomous gateway

# Run piconomous with all its data stored in /opt/piconomous
# Config will be loaded from the default ~/.piconomous/config.json
# Workspace will be created at /opt/piconomous/workspace
PICONOMOUS_HOME=/opt/piconomous piconomous agent

# Use both for a fully customized setup
PICONOMOUS_HOME=/srv/piconomous PICONOMOUS_CONFIG=/srv/piconomous/main.json piconomous gateway
```

### Workspace Layout

Piconomous stores data in your configured workspace (default: `~/.piconomous/workspace`):

```
~/.piconomous/workspace/
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

### Skill Sources

By default, skills are loaded from:

1. `~/.piconomous/workspace/skills` (workspace)
2. `~/.piconomous/skills` (global)
3. `<binary-embedded-path>/skills` (builtin, set at build time)

For advanced/test setups, you can override the builtin skills root with:

```bash
export PICONOMOUS_BUILTIN_SKILLS=/path/to/skills
```

### Unified Command Execution Policy

- Generic slash commands are executed through a single path in `pkg/agent/loop.go` via `commands.Executor`.
- Channel adapters no longer consume generic commands locally; they forward inbound text to the bus/agent path. Telegram still auto-registers supported commands at startup.
- Unknown slash command (for example `/foo`) passes through to normal LLM processing.
- Registered but unsupported command on the current channel (for example `/show` on WhatsApp) returns an explicit user-facing error and stops further processing.

### Agent Bindings (Route messages to specific agents)

Use `bindings` in `config.json` to route incoming messages to different agents by channel/account/context.

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.piconomous/workspace",
      "model_name": "gpt-4o-mini"
    },
    "list": [
      { "id": "main", "default": true, "name": "Main Assistant" },
      { "id": "support", "name": "Support Assistant" },
      { "id": "sales", "name": "Sales Assistant" }
    ]
  },
  "bindings": [
    {
      "agent_id": "support",
      "match": {
        "channel": "telegram",
        "account_id": "*",
        "peer": { "kind": "direct", "id": "user123" }
      }
    },
    {
      "agent_id": "sales",
      "match": {
        "channel": "discord",
        "account_id": "my-discord-bot",
        "guild_id": "987654321"
      }
    }
  ]
}
```

#### `bindings` fields

| Field | Required | Description |
|-------|----------|-------------|
| `agent_id` | Yes | Target agent id in `agents.list` |
| `match.channel` | Yes | Channel name (e.g. `telegram`, `discord`) |
| `match.account_id` | No | Channel account filter. Use `"*"` for all accounts of that channel. If omitted, only default account is matched |
| `match.peer.kind` + `match.peer.id` | No | Exact peer match (e.g. direct chat / topic / group id) |
| `match.guild_id` | No | Guild/server-level match |
| `match.team_id` | No | Team/workspace-level match |

#### Matching priority

When multiple bindings exist, Piconomous resolves in this order:

1. `peer`
2. `parent_peer` (for thread/topic parent contexts)
3. `guild_id`
4. `team_id`
5. `account_id` (non-wildcard)
6. channel wildcard (`account_id: "*"`)
7. default agent

If a binding points to a missing `agent_id`, Piconomous falls back to the default agent.

#### How matching works (step-by-step)

1. Piconomous first filters bindings by `match.channel` (must equal current channel).
2. It then filters by `match.account_id`:
   - omitted: match only the channel's default account
   - `"*"`: match all accounts on this channel
   - explicit value: exact account id match (case-insensitive)
3. From the remaining candidates, it applies the priority chain above and stops at the first hit.

In other words: **channel + account form the candidate set; peer/guild/team then decide final winner**.

#### Common recipes

**1) Route one specific DM user to a specialist agent**

```json
{
  "agent_id": "support",
  "match": {
    "channel": "telegram",
    "account_id": "*",
    "peer": { "kind": "direct", "id": "user123" }
  }
}
```

**2) Route one Discord server (guild) to a dedicated agent**

```json
{
  "agent_id": "sales",
  "match": {
    "channel": "discord",
    "account_id": "my-discord-bot",
    "guild_id": "987654321"
  }
}
```

**3) Route all remaining traffic of a channel to a fallback agent**

```json
{
  "agent_id": "main",
  "match": {
    "channel": "discord",
    "account_id": "*"
  }
}
```

#### Authoring guidelines (important)

- Keep exactly one clear default agent in `agents.list` (`"default": true`).
- Put specific rules (`peer`, `guild_id`, `team_id`) and broad rules (`account_id: "*"` only) together safely; priority already guarantees specific rules win.
- Avoid duplicate rules with the same specificity and match values. If duplicates exist, the first matching entry in the config array wins.
- Ensure every `agent_id` exists in `agents.list`; unknown IDs silently fall back to default.

#### Troubleshooting checklist

- **Rule not taking effect?** Check `match.channel` spelling first (must be exact).
- **Expected account-specific routing but still using default?** Verify `match.account_id` equals actual runtime account id.
- **Wildcard catches too much traffic?** Add more specific `peer/guild/team` rules for critical paths.
- **Unexpected default fallback?** Confirm `agent_id` exists and is not misspelled.

### 🔒 Security Sandbox

Piconomous runs in a sandboxed environment by default. The agent can only access files and execute commands within the configured workspace.

#### Default Configuration

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.piconomous/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Option                  | Default                 | Description                               |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.piconomous/workspace` | Working directory for the agent           |
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

### Exec Security

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Allow exec tool from remote channels (Telegram/Discord etc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Enable dangerous command interception |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Custom regex patterns to block |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Custom regex patterns to allow |

> **Security Note:** Symlink protection is enabled by default — all file paths are resolved through `filepath.EvalSymlinks` before whitelist matching, preventing symlink escape attacks.

#### Known Limitation: Child Processes From Build Tools

The exec safety guard only inspects the command line Piconomous launches directly. It does not recursively inspect child
processes spawned by allowed developer tools such as `make`, `go run`, `cargo`, `npm run`, or custom build scripts.

That means a top-level command can still compile or launch other binaries after it passes the initial guard check. In
practice, treat build scripts, Makefiles, package scripts, and generated binaries as executable code that needs the same
level of review as a direct shell command.

For higher-risk environments:

* Review build scripts before execution.
* Prefer approval/manual review for compile-and-run workflows.
* Run Piconomous inside a container or VM if you need stronger isolation than the built-in guard provides.

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
export PICONOMOUS_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
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

Piconomous can perform periodic tasks automatically. Create a `HEARTBEAT.md` file in your workspace:

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

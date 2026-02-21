# CLI Reference

Complete reference for all PicoClaw command-line commands.

## Global Commands

### picoclaw onboard

Initialize configuration and workspace.

```bash
picoclaw onboard
```

This command:
- Creates `~/.picoclaw/config.json` with default settings
- Creates `~/.picoclaw/workspace/` directory
- Copies default workspace templates (AGENT.md, IDENTITY.md, etc.)

### picoclaw agent

Interact with the agent directly.

```bash
# Send a single message
picoclaw agent -m "What is 2+2?"

# Interactive mode
picoclaw agent

# With debug logging
picoclaw agent --debug -m "Hello"

# With specific session
picoclaw agent -s "my-session" -m "Continue our conversation"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--message` | `-m` | Send a single message and exit |
| `--session` | `-s` | Session key (default: `cli:default`) |
| `--debug` | `-d` | Enable debug logging |

**Interactive Mode Commands:**

| Command | Description |
|---------|-------------|
| `exit` | Exit interactive mode |
| `quit` | Exit interactive mode |
| `Ctrl+C` | Exit interactive mode |

### picoclaw gateway

Start the gateway for all enabled channels.

```bash
# Start gateway
picoclaw gateway

# With debug logging
picoclaw gateway --debug
```

| Flag | Short | Description |
|------|-------|-------------|
| `--debug` | `-d` | Enable debug logging |

The gateway:
- Starts all enabled channels (Telegram, Discord, etc.)
- Starts the heartbeat service
- Starts the cron service
- Starts health endpoints at `http://host:port/health` and `/ready`

### picoclaw status

Show system status.

```bash
picoclaw status
```

Displays:
- Version and build info
- Config file location
- Workspace location
- Configured API keys status
- OAuth authentication status

### picoclaw version

Show version information.

```bash
picoclaw version
# or
picoclaw -v
picoclaw --version
```

## Authentication Commands

### picoclaw auth login

Login via OAuth or paste token.

```bash
# Login with OAuth (opens browser)
picoclaw auth login --provider openai

# Login with device code (headless)
picoclaw auth login --provider openai --device-code

# Login with token paste
picoclaw auth login --provider anthropic
```

| Flag | Short | Description |
|------|-------|-------------|
| `--provider` | `-p` | Provider name (openai, anthropic) |
| `--device-code` | | Use device code flow for headless environments |

### picoclaw auth logout

Remove stored credentials.

```bash
# Logout from specific provider
picoclaw auth logout --provider openai

# Logout from all providers
picoclaw auth logout
```

### picoclaw auth status

Show current authentication status.

```bash
picoclaw auth status
```

## Cron Commands

### picoclaw cron list

List all scheduled jobs.

```bash
picoclaw cron list
```

### picoclaw cron add

Add a new scheduled job.

```bash
# Every 2 hours (7200 seconds)
picoclaw cron add -n "Check weather" -m "Check weather in Beijing" -e 7200

# With cron expression (daily at 9am)
picoclaw cron add -n "Morning report" -m "Give me a morning report" -c "0 9 * * *"

# Deliver response to channel
picoclaw cron add -n "Reminder" -m "Time for a break!" -e 3600 -d --to "123456789" --channel "telegram"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--name` | `-n` | Job name (required) |
| `--message` | `-m` | Message for agent (required) |
| `--every` | `-e` | Run every N seconds |
| `--cron` | `-c` | Cron expression |
| `--deliver` | `-d` | Deliver response to channel |
| `--to` | | Recipient for delivery |
| `--channel` | | Channel for delivery |

### picoclaw cron remove

Remove a scheduled job.

```bash
picoclaw cron remove <job_id>
```

### picoclaw cron enable

Enable a disabled job.

```bash
picoclaw cron enable <job_id>
```

### picoclaw cron disable

Disable a job without removing it.

```bash
picoclaw cron disable <job_id>
```

## Skills Commands

### picoclaw skills list

List installed skills.

```bash
picoclaw skills list
```

### picoclaw skills install

Install a skill from GitHub.

```bash
picoclaw skills install sipeed/picoclaw-skills/weather
```

### picoclaw skills remove

Remove an installed skill.

```bash
picoclaw skills remove weather
```

### picoclaw skills install-builtin

Install all builtin skills to workspace.

```bash
picoclaw skills install-builtin
```

### picoclaw skills list-builtin

List available builtin skills.

```bash
picoclaw skills list-builtin
```

### picoclaw skills search

Search for available skills.

```bash
picoclaw skills search
```

### picoclaw skills show

Show skill details.

```bash
picoclaw skills show weather
```

## Migration Commands

### picoclaw migrate

Migrate from OpenClaw to PicoClaw.

```bash
# Detect and migrate
picoclaw migrate

# Preview changes
picoclaw migrate --dry-run

# Re-sync workspace files
picoclaw migrate --refresh

# Migrate without confirmation
picoclaw migrate --force

# Only migrate config
picoclaw migrate --config-only

# Only migrate workspace
picoclaw migrate --workspace-only

# Override directories
picoclaw migrate --openclaw-home /path/to/openclaw --picoclaw-home /path/to/picoclaw
```

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be migrated without making changes |
| `--refresh` | Re-sync workspace files from OpenClaw |
| `--config-only` | Only migrate config, skip workspace |
| `--workspace-only` | Only migrate workspace files, skip config |
| `--force` | Skip confirmation prompts |
| `--openclaw-home` | Override OpenClaw home directory |
| `--picoclaw-home` | Override PicoClaw home directory |

## Command Summary

| Command | Description |
|---------|-------------|
| `onboard` | Initialize configuration and workspace |
| `agent` | Interact with agent directly |
| `gateway` | Start gateway for all channels |
| `status` | Show system status |
| `version` | Show version information |
| `auth login` | Login via OAuth or token |
| `auth logout` | Remove stored credentials |
| `auth status` | Show authentication status |
| `cron list` | List scheduled jobs |
| `cron add` | Add a scheduled job |
| `cron remove` | Remove a job |
| `cron enable` | Enable a job |
| `cron disable` | Disable a job |
| `skills list` | List installed skills |
| `skills install` | Install a skill |
| `skills remove` | Remove a skill |
| `skills install-builtin` | Install builtin skills |
| `skills list-builtin` | List builtin skills |
| `skills search` | Search for skills |
| `skills show` | Show skill details |
| `migrate` | Migrate from OpenClaw |

## See Also

- [Configuration Reference](../configuration/config-file.md) - Complete configuration options
- [Quick Start](../getting-started/quick-start.md) - Get started quickly
- [Troubleshooting](../operations/troubleshooting.md) - Common issues and solutions

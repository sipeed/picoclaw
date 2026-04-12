# Scheduled Tasks and Cron Jobs

> Back to [README](../README.md)

PicoClaw stores scheduled jobs in the current workspace and can run them either as reminders, full agent turns, or shell commands.

## Schedule Types

PicoClaw currently uses three schedule forms in the cron tool:

- `at_seconds`: one-time job, relative to now. After it runs, the job is removed from the store.
- `every_seconds`: recurring interval, in seconds.
- `cron_expr`: recurring cron expression such as `0 9 * * *`.

The CLI command `picoclaw cron add` currently supports recurring jobs only:

- `--every <seconds>`
- `--cron '<expr>'`

There is no CLI flag for a one-time `at` job today.

Examples:

```bash
picoclaw cron add --name "Daily summary" --message "Summarize today's logs" --cron "0 18 * * *"
picoclaw cron add --name "Ping" --message "heartbeat" --every 300 --deliver
```

## Execution Modes

Jobs are stored with a message payload and can execute in three stable user-facing modes:

### `deliver: false`

This is the default for the cron tool.

When the job fires, PicoClaw sends the saved message back through the agent loop as a new agent turn. Use this for scheduled work that may need reasoning, tools, or a generated reply.

Because the saved `message` is replayed as a new user-style input, write it from the user's perspective or as direct instructions to the agent. Prefer wording such as `check the repo every hour and tell me if there is a new release` over third-person wording such as `check the repo and notify the user`.

### `deliver: true`

When the job fires, PicoClaw publishes the saved message directly to the target channel and recipient without agent processing.

The CLI `picoclaw cron add --deliver` flag uses this mode.

### `command`

When a cron-tool job includes `command`, PicoClaw runs that shell command through the `exec` tool and publishes the command output back to the channel.

For command jobs, `deliver` is forced to `false` when the job is created. The saved `message` becomes descriptive text only; the scheduled action is the shell command.

The current CLI `picoclaw cron add` command does not expose a `command` flag.

## Writing Job Messages

For normal cron jobs without `command`, the saved `payload.message` becomes the next input sent to the agent when the job fires. In practice, that means the job message should read like something the user would say to the agent.

Recommended style:

- Use first-person or direct-address wording such as `tell me`, `remind me`, `reply in Chinese`, `do not reply if nothing changed`
- Be explicit about the quiet case if needed, for example `If there is no update, do not reply`
- Avoid third-person wording such as `notify the user`, because the model may continue replying in third person

Examples:

```text
Good: Check gdsfactory/gdsfactory every hour. If there is a new release, tell me in Chinese and summarize the changes. If nothing changed, do not reply.
Bad: Check gdsfactory/gdsfactory every hour. If there is a new release, notify the user and summarize the changes.
```

## Config and Security Gates

### `tools.cron`

`tools.cron.enabled` controls whether the agent-facing `cron` tool is registered. Default: `true`.

If you disable `tools.cron`, users can no longer create or manage jobs through the agent tool. The gateway still starts `CronService`, but it does not install the job execution callback. As a result, due jobs do not actually run; one-time jobs may be deleted and recurring jobs may be rescheduled without executing their payload. The CLI still uses the same job store.

`tools.cron.exec_timeout_minutes` sets the timeout used for scheduled command execution. Default: `5`. Set `0` for no timeout.

### `tools.exec`

Scheduled command jobs depend on `tools.exec.enabled`. Default: `true`.

If `tools.exec.enabled` is `false`:

- new command jobs are rejected by the cron tool
- existing command jobs publish a `command execution is disabled` error when they fire

`tools.exec.allow_remote` is still enforced by the exec tool, but cron command scheduling already requires an internal channel when the job is created. In practice, reminder jobs can be scheduled from remote channels, while scheduled command jobs are limited to internal channels.

### `allow_command`

`tools.cron.allow_command` defaults to `true`.

This is not a hard disable switch. If you set `allow_command` to `false`, PicoClaw still allows a command job when the caller explicitly passes `command_confirm: true`.

Command jobs also require an internal channel. Non-command reminders do not have that restriction.

Example:

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5,
      "allow_command": true
    },
    "exec": {
      "enabled": true
    }
  }
}
```

## Persistence and Location

Cron jobs are stored in:

```text
<workspace>/cron/jobs.json
```

By default, the workspace is:

```text
~/.picoclaw/workspace
```

If `PICOCLAW_HOME` is set, the default workspace becomes:

```text
$PICOCLAW_HOME/workspace
```

Both the gateway and `picoclaw cron` CLI subcommands use the same `cron/jobs.json` file.

Notes:

- one-time `at_seconds` jobs are deleted after they run
- recurring jobs stay in the store until removed
- disabled jobs stay in the store and still appear in `picoclaw cron list`

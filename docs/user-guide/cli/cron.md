# picoclaw cron

Manage scheduled tasks and reminders.

## Usage

```bash
# List jobs
picoclaw cron list

# Add job
picoclaw cron add -n "name" -m "message" (-e <seconds> | -c <cron>)

# Remove job
picoclaw cron remove <job_id>

# Enable/disable
picoclaw cron enable <job_id>
picoclaw cron disable <job_id>
```

## Subcommands

### cron list

List all scheduled jobs.

```bash
picoclaw cron list
```

Output:
```
Scheduled Jobs:
----------------
  Weather Check (abc123)
    Schedule: every 3600s
    Status: enabled
    Next run: 2025-02-20 15:00

  Morning Report (def456)
    Schedule: 0 9 * * *
    Status: enabled
    Next run: 2025-02-21 09:00
```

### cron add

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
| `--message` | `-m` | Message for the agent (required) |
| `--every` | `-e` | Run every N seconds |
| `--cron` | `-c` | Cron expression |
| `--deliver` | `-d` | Deliver response to channel |
| `--to` | | Recipient ID for delivery |
| `--channel` | | Channel name for delivery |

**Schedule Types:**

| Type | Flag | Example |
|------|------|---------|
| Interval | `-e 3600` | Every hour |
| Cron | `-c "0 9 * * *"` | Daily at 9am |

### cron remove

Remove a scheduled job.

```bash
picoclaw cron remove <job_id>
```

### cron enable / cron disable

Enable or disable a job without removing it.

```bash
picoclaw cron enable <job_id>
picoclaw cron disable <job_id>
```

## Cron Expressions

Standard 5-field cron format:

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6, Sunday = 0)
│ │ │ │ │
* * * * *
```

**Examples:**

| Expression | Meaning |
|------------|---------|
| `0 9 * * *` | Every day at 9:00 AM |
| `0 */2 * * *` | Every 2 hours |
| `30 8 * * 1-5` | Weekdays at 8:30 AM |
| `0 0 * * 0` | Every Sunday at midnight |
| `0 9 1 * *` | First day of month at 9:00 AM |

## Storage

Jobs are stored in:
```
~/.picoclaw/workspace/cron/jobs.json
```

## Configuration

Configure timeout in `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "cron": {
      "exec_timeout_minutes": 5
    }
  }
}
```

## Job Delivery

When `--deliver` is set, the agent's response is sent to the specified channel:

```bash
# Send reminder to Telegram user
picoclaw cron add -n "Standup" -m "Time for standup!" \
  -c "0 10 * * 1-5" -d --to "123456789" --channel "telegram"
```

## Examples

```bash
# Daily weather report
picoclaw cron add -n "Weather" -m "What's the weather today?" -c "0 7 * * *"

# Hourly reminder
picoclaw cron add -n "Hydrate" -m "Drink water!" -e 3600

# Weekly summary
picoclaw cron add -n "Weekly" -m "Summarize what I did this week" -c "0 17 * * 5"

# List all jobs
picoclaw cron list

# Disable a job
picoclaw cron disable abc123

# Remove a job
picoclaw cron remove abc123
```

## See Also

- [Heartbeat Tasks](../workspace/heartbeat-tasks.md)
- [Cron Tool](../tools/cron.md)
- [CLI Reference](../cli-reference.md)

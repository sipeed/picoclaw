# Cron Tool

Schedule reminders and recurring tasks.

## Tool

### cron

Schedule, list, or manage scheduled jobs.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `action` | string | Yes | Action: `add`, `list`, `remove`, `enable`, `disable` |
| `name` | string | Conditional | Job name (for add) |
| `message` | string | Conditional | Message for agent (for add) |
| `every_seconds` | int | Conditional | Run every N seconds |
| `at_seconds` | int | Conditional | Run once after N seconds |
| `cron_expr` | string | Conditional | Cron expression |
| `job_id` | string | Conditional | Job ID (for remove/enable/disable) |

**Examples:**

```json
// Add recurring job
{
  "action": "add",
  "name": "Hourly check",
  "message": "Check system status",
  "every_seconds": 3600
}

// Add one-time reminder
{
  "action": "add",
  "name": "Meeting reminder",
  "message": "Time for your meeting!",
  "at_seconds": 300
}

// List all jobs
{
  "action": "list"
}

// Remove a job
{
  "action": "remove",
  "job_id": "abc123"
}
```

## CLI vs Tool

### CLI Commands

Use CLI for administrative tasks:

```bash
picoclaw cron list
picoclaw cron add -n "name" -m "message" -e 3600
picoclaw cron remove <job_id>
```

### Tool Usage

The agent uses the cron tool when you ask:

```
User: "Remind me to take a break in 30 minutes"

Agent uses cron tool:
{
  "action": "add",
  "name": "Break reminder",
  "message": "Time to take a break!",
  "at_seconds": 1800
}

Agent: "I'll remind you to take a break in 30 minutes."
```

## Schedule Types

### One-Time (at_seconds)

Run once after N seconds:

```json
{
  "action": "add",
  "name": "Reminder",
  "message": "Meeting in 10 minutes!",
  "at_seconds": 600
}
```

### Recurring (every_seconds)

Run every N seconds:

```json
{
  "action": "add",
  "name": "Hourly check",
  "message": "Check the time",
  "every_seconds": 3600
}
```

### Cron Expression

Use standard cron format:

```json
{
  "action": "add",
  "name": "Daily standup",
  "message": "Time for standup!",
  "cron_expr": "0 9 * * 1-5"
}
```

## Cron Expressions

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6)
│ │ │ │ │
* * * * *
```

| Expression | Meaning |
|------------|---------|
| `0 9 * * *` | Daily at 9:00 AM |
| `0 */2 * * *` | Every 2 hours |
| `30 8 * * 1-5` | Weekdays at 8:30 AM |
| `0 0 * * 0` | Every Sunday midnight |

## Configuration

### Timeout

Set max execution time for jobs:

```json
{
  "tools": {
    "cron": {
      "exec_timeout_minutes": 5
    }
  }
}
```

## Storage

Jobs are stored in:

```
~/.picoclaw/workspace/cron/jobs.json
```

## Examples

```
User: "Remind me every hour to drink water"

Agent uses cron tool:
{
  "action": "add",
  "name": "Hydration reminder",
  "message": "Time to drink some water!",
  "every_seconds": 3600
}
```

```
User: "Set a reminder for my 3 PM meeting"

Agent uses cron tool:
{
  "action": "add",
  "name": "Meeting reminder",
  "message": "Your meeting starts in 10 minutes!",
  "at_seconds": <calculated seconds to 2:50 PM>
}
```

## See Also

- [CLI Cron Commands](../cli/cron.md)
- [Heartbeat Tasks](../workspace/heartbeat-tasks.md)

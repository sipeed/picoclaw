# Scheduled Tasks Tutorial

This tutorial covers setting up automated tasks with cron jobs and heartbeat features.

## Prerequisites

- 15 minutes
- PicoClaw installed and configured
- A working LLM provider

## Overview

PicoClaw offers two ways to schedule automated tasks:

| Feature | Purpose | Trigger |
|---------|---------|---------|
| **Cron Jobs** | Specific scheduled tasks | Time-based (cron syntax) |
| **Heartbeat** | Periodic autonomous actions | Interval-based |

## Part 1: Cron Jobs

### Understanding Cron

Cron jobs run at specific times using cron syntax:

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday = 0)
│ │ │ │ │
* * * * *
```

### Create a Cron Job

#### Using CLI

```bash
picoclaw cron add "0 9 * * *" "Send me a daily summary"
```

#### Using Configuration

Edit `~/.picoclaw/config.json`:

```json
{
  "cron": {
    "enabled": true,
    "jobs": [
      {
        "name": "morning_summary",
        "schedule": "0 9 * * *",
        "prompt": "Generate a morning summary including the current date, weather, and any important notes from the memory file"
      },
      {
        "name": "hourly_check",
        "schedule": "0 * * * *",
        "prompt": "Check system status and report any issues"
      }
    ]
  }
}
```

### Cron Schedule Examples

| Schedule | Meaning |
|----------|---------|
| `0 9 * * *` | Every day at 9:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 */2 * * *` | Every 2 hours |
| `0 9 * * 1` | Every Monday at 9:00 AM |
| `0 9 1 * *` | First day of every month |
| `0 0 * * *` | Every day at midnight |

### Managing Cron Jobs

#### List Jobs

```bash
picoclaw cron list
```

Output:

```
Name              Schedule        Next Run                Prompt
---------------------------------------------------------------------------
morning_summary   0 9 * * *       2024-01-16 09:00:00     Generate a morning...
hourly_check      0 * * * *       2024-01-15 11:00:00     Check system status...
```

#### Remove Job

```bash
picoclaw cron remove morning_summary
```

#### Enable/Disable

```bash
picoclaw cron enable morning_summary
picoclaw cron disable morning_summary
```

### Cron with Output Channels

Send cron output to a channel:

```json
{
  "cron": {
    "enabled": true,
    "jobs": [
      {
        "name": "daily_report",
        "schedule": "0 18 * * *",
        "prompt": "Generate a daily activity report",
        "output": {
          "channel": "telegram",
          "chat_id": "123456789"
        }
      }
    ]
  }
}
```

## Part 2: Heartbeat Tasks

### Understanding Heartbeat

Heartbeat tasks run at regular intervals, allowing the agent to act autonomously. They're defined in `HEARTBEAT.md`.

### Create HEARTBEAT.md

```bash
nano ~/.picoclaw/workspace/HEARTBEAT.md
```

```markdown
# Heartbeat Tasks

You are running in periodic mode. Every interval, perform these checks:

## System Health
- Check available disk space
- Check memory usage
- Report any anomalies

## Data Collection
- Collect sensor data (if hardware is connected)
- Update any tracking logs

## Notifications
- If any issues are found, prepare a notification
- Summarize findings concisely

## Memory Update
- Update MEMORY.md with important observations
```

### Configure Heartbeat

Edit `~/.picoclaw/config.json`:

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": "5m",
    "max_runs": 0
  }
}
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable heartbeat tasks |
| `interval` | `5m` | Time between runs |
| `max_runs` | `0` | Maximum runs (0 = unlimited) |

### Interval Examples

| Interval | Meaning |
|----------|---------|
| `30s` | Every 30 seconds |
| `5m` | Every 5 minutes |
| `1h` | Every hour |
| `24h` | Every day |

### Running Heartbeat

```bash
# Start with heartbeat enabled
picoclaw agent --heartbeat

# Or with gateway
picoclaw gateway
```

### Environment Variable

```bash
export PICOCLAW_HEARTBEAT_ENABLED=true
export PICOCLAW_HEARTBEAT_INTERVAL=10m

picoclaw gateway
```

## Part 3: Practical Examples

### Example 1: Daily Backup Reminder

**Cron Job:**

```json
{
  "cron": {
    "jobs": [
      {
        "name": "backup_reminder",
        "schedule": "0 20 * * *",
        "prompt": "Check when the last backup was made (look in workspace/logs) and remind the user if it's been more than 24 hours"
      }
    ]
  }
}
```

### Example 2: System Monitoring

**Heartbeat (HEARTBEAT.md):**

```markdown
# System Monitor

Every 5 minutes, check:
1. CPU usage (use: top -bn1 | head -5)
2. Memory usage (use: free -h)
3. Disk space (use: df -h)

If any resource is above 80%, log a warning to ~/workspace/alerts.log
```

### Example 3: News Digest

**Cron Job:**

```json
{
  "cron": {
    "jobs": [
      {
        "name": "news_digest",
        "schedule": "0 8 * * *",
        "prompt": "Search for tech news from the last 24 hours and create a brief summary of the top 5 stories",
        "output": {
          "channel": "telegram",
          "chat_id": "123456789"
        }
      }
    ]
  }
}
```

### Example 4: Website Monitor

**Heartbeat:**

```markdown
# Website Monitor

Every 10 minutes:
1. Fetch https://example.com/health
2. If response is not 200, log to ~/workspace/outages.log
3. If this is a new outage, create an alert message
```

### Example 5: IoT Sensor Logger

**Heartbeat:**

```markdown
# Sensor Logger

Every minute:
1. Read temperature from I2C sensor at address 0x76
2. Read humidity
3. Append to ~/workspace/sensors.csv with timestamp
4. Alert if temperature > 30 or humidity > 80
```

## Part 4: Viewing Task History

### Cron Logs

```bash
picoclaw cron logs
```

Output:

```
Time                    Job              Status   Result
----------------------------------------------------------------
2024-01-15 09:00:00    morning_summary   success  Generated summary...
2024-01-15 10:00:00    hourly_check      success  System OK
2024-01-15 11:00:00    hourly_check      success  System OK
```

### Heartbeat Logs

Check the workspace logs:

```bash
cat ~/.picoclaw/workspace/heartbeat.log
```

## Part 5: Best Practices

### Use Appropriate Intervals

| Task Type | Recommended Interval |
|-----------|---------------------|
| Critical monitoring | 1-5 minutes |
| Regular checks | 5-15 minutes |
| Daily tasks | Once per day |
| Weekly summaries | Once per week |

### Error Handling

Include error handling in your prompts:

```markdown
When performing tasks:
- If an operation fails, log the error
- Try up to 3 times before giving up
- Report failures clearly
```

### Resource Management

Avoid resource-intensive tasks:

```markdown
Guidelines:
- Keep operations quick (< 30 seconds)
- Don't create large files
- Clean up temporary data
- Use efficient commands
```

### Idempotency

Make tasks idempotent (safe to run multiple times):

```markdown
Before performing actions:
- Check if action is needed
- Don't duplicate work
- Update timestamps, don't create new entries
```

## Troubleshooting

### Cron Jobs Not Running

1. Check cron is enabled:
   ```json
   {"cron": {"enabled": true}}
   ```

2. Verify schedule syntax:
   ```bash
   # Test cron expression
   crontab -e
   # Add: * * * * * echo "test" >> /tmp/cron-test
   ```

3. Check gateway is running

### Heartbeat Not Working

1. Check heartbeat is enabled:
   ```bash
   picoclaw agent --debug --heartbeat
   ```

2. Verify HEARTBEAT.md exists:
   ```bash
   cat ~/.picoclaw/workspace/HEARTBEAT.md
   ```

3. Check interval is reasonable

### Tasks Taking Too Long

1. Reduce task complexity
2. Increase interval
3. Use simpler prompts

## Next Steps

- [Multi-Agent Tutorial](multi-agent-setup.md) - Run specialized agents
- [Cron Tool Docs](../user-guide/tools/cron.md) - Full cron documentation
- [Heartbeat Docs](../user-guide/workspace/heartbeat-tasks.md) - Heartbeat details

## Summary

You learned:
- How to create and manage cron jobs
- How to configure heartbeat tasks
- Practical scheduling examples
- Best practices for automated tasks

You can now automate your PicoClaw assistant!

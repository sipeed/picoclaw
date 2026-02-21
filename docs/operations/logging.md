# Log Management

This guide covers log configuration, collection, and analysis for PicoClaw.

## Overview

PicoClaw provides structured logging with multiple output formats and levels. Proper log management helps with:

- Debugging issues
- Security auditing
- Performance analysis
- Compliance requirements

## Log Levels

| Level | Description |
|-------|-------------|
| `debug` | Detailed debugging information |
| `info` | General operational messages |
| `warn` | Warning conditions |
| `error` | Error conditions |

## Configuration

### Basic Configuration

```json
{
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  }
}
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `level` | `info` | Minimum log level |
| `format` | `json` | Output format (`json`, `text`) |
| `output` | `stdout` | Output destination (`stdout`, `stderr`, file path) |
| `max_size` | `100` | Max log file size in MB (for file output) |
| `max_backups` | `3` | Max number of old log files |
| `max_age` | `28` | Max days to retain logs |
| `compress` | `true` | Compress rotated logs |

### Environment Variables

```bash
export PICOCLAW_LOGGING_LEVEL=debug
export PICOCLAW_LOGGING_FORMAT=json
export PICOCLAW_LOGGING_OUTPUT=/var/log/picoclaw/app.log
```

## Log Formats

### JSON Format (Recommended)

```json
{
  "time": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "component": "agent",
  "message": "Processing message",
  "session_id": "abc123",
  "duration_ms": 1234,
  "model": "claude-opus-4-5"
}
```

### Text Format

```
2024-01-15T10:30:00.123Z INFO agent Processing message session_id=abc123 duration_ms=1234
```

## Log Components

PicoClaw organizes logs by component:

| Component | Description |
|-----------|-------------|
| `main` | Application startup/shutdown |
| `config` | Configuration loading |
| `agent` | Agent loop and message processing |
| `provider` | LLM provider interactions |
| `channel` | Chat platform integrations |
| `tool` | Tool execution |
| `session` | Session management |
| `bus` | Message bus activity |
| `cron` | Scheduled tasks |

## Command Line Options

### Debug Mode

Enable verbose logging:

```bash
picoclaw agent --debug -m "Hello"
picoclaw gateway --debug
```

### Verbose Output

```bash
picoclaw agent -v -m "Hello"
picoclaw gateway -vv
```

| Flag | Level |
|------|-------|
| (none) | info |
| `-v` | info (verbose) |
| `-vv` | debug |
| `--debug` | debug (with stack traces) |

## File Logging

### Configuration

```json
{
  "logging": {
    "level": "info",
    "format": "json",
    "output": "/var/log/picoclaw/picoclaw.log",
    "max_size": 100,
    "max_backups": 5,
    "max_age": 30,
    "compress": true
  }
}
```

### Directory Setup

```bash
# Create log directory
sudo mkdir -p /var/log/picoclaw
sudo chown $USER:$USER /var/log/picoclaw
```

### Log Rotation

PicoClaw handles log rotation automatically when configured for file output. Alternatively, use logrotate:

```bash
# /etc/logrotate.d/picoclaw
/var/log/picoclaw/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 0640 $USER $USER
    sharedscripts
    postrotate
        systemctl reload picoclaw > /dev/null 2>&1 || true
    endscript
}
```

## Log Collection

### Fluentd/Fluent Bit

```yaml
# fluent-bit.conf
[INPUT]
    Name tail
    Path /var/log/picoclaw/picoclaw.log
    Tag picoclaw

[FILTER]
    Name parser
    Match picoclaw*
    Key_Name log
    Parser json

[OUTPUT]
    Name elasticsearch
    Match picoclaw*
    Host elasticsearch
    Port 9200
```

### Logstash

```ruby
# logstash.conf
input {
  file {
    path => "/var/log/picoclaw/picoclaw.log"
    codec => json
    type => "picoclaw"
  }
}

filter {
  if [type] == "picoclaw" {
    date {
      match => [ "time", "ISO8601" ]
    }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "picoclaw-%{+YYYY.MM.dd}"
  }
}
```

### Loki/Promtail

```yaml
# promtail.yml
scrape_configs:
  - job_name: picoclaw
    static_configs:
      - targets:
          - localhost
        labels:
          job: picoclaw
          __path__: /var/log/picoclaw/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            component: component
            message: message
      - labels:
          level:
          component:
```

## Systemd Journal

When running as a systemd service, logs go to journald:

```bash
# View logs
journalctl -u picoclaw

# Follow logs
journalctl -u picoclaw -f

# Filter by level
journalctl -u picoclaw -p err

# Filter by time
journalctl -u picoclaw --since "1 hour ago"
```

### Journald Configuration

```bash
# /etc/systemd/journald.conf
[Journal]
Storage=persistent
Compress=yes
MaxRetentionSec=1month
```

## Docker Logging

### Default Configuration

```bash
docker run picoclaw:latest
```

Logs go to Docker's logging driver (json-file by default).

### Docker Compose

```yaml
services:
  picoclaw:
    image: picoclaw:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Send to External System

```yaml
services:
  picoclaw:
    image: picoclaw:latest
    logging:
      driver: "syslog"
      options:
        syslog-address: "tcp://logserver:514"
        syslog-facility: "daemon"
        tag: "picoclaw"
```

## Log Analysis

### Common Queries

#### Error Count

```bash
grep -c '"level":"error"' /var/log/picoclaw/picoclaw.log
```

#### Recent Errors

```bash
grep '"level":"error"' /var/log/picoclaw/picoclaw.log | tail -20
```

#### Provider Errors

```bash
grep '"component":"provider"' /var/log/picoclaw/picoclaw.log | grep '"level":"error"'
```

#### Slow Requests

```bash
jq 'select(.duration_ms > 5000)' /var/log/picoclaw/picoclaw.log
```

### Using jq

```bash
# Parse and pretty print
cat picoclaw.log | jq '.'

# Filter by level
cat picoclaw.log | jq 'select(.level == "error")'

# Filter by component
cat picoclaw.log | jq 'select(.component == "agent")'

# Extract specific fields
cat picoclaw.log | jq '{time, level, message}'

# Count by level
cat picoclaw.log | jq -r '.level' | sort | uniq -c
```

## Security and Privacy

### Redacting Sensitive Data

PicoClaw automatically redacts:
- API keys (masked as `***`)
- Authentication tokens
- Passwords

### Manual Redaction

```bash
# Redact API keys before sharing logs
sed -E 's/sk-[a-zA-Z0-9_-]+/sk-***/g' picoclaw.log > redacted.log
```

### Log Access Control

```bash
# Restrict log file access
chmod 640 /var/log/picoclaw/picoclaw.log
chown root:picoclaw /var/log/picoclaw/picoclaw.log
```

## Debugging with Logs

### Enable Debug Mode

```json
{
  "logging": {
    "level": "debug"
  }
}
```

### Trace a Session

```bash
# Find all logs for a session
grep '"session_id":"abc123"' /var/log/picoclaw/picoclaw.log
```

### Analyze Tool Calls

```bash
# All tool executions
grep '"component":"tool"' /var/log/picoclaw/picoclaw.log | jq '.'
```

### Provider Issues

```bash
# Provider connection issues
grep '"component":"provider"' /var/log/picoclaw/picoclaw.log | grep -E '(error|warn)'
```

## Best Practices

1. **Use JSON format** - Easier to parse and analyze
2. **Set appropriate levels** - Use `info` for production, `debug` for troubleshooting
3. **Configure rotation** - Prevent disk space issues
4. **Centralize logs** - Use a log aggregation system
5. **Monitor error rates** - Set up alerts for error spikes
6. **Secure logs** - Restrict access and redact sensitive data
7. **Archive old logs** - Keep historical data for analysis

## See Also

- [Monitoring Guide](monitoring.md)
- [Troubleshooting](troubleshooting.md)
- [Health Endpoints](health-endpoints.md)
- [Security Configuration](../deployment/security.md)

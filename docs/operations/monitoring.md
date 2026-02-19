# Monitoring Setup Guide

This guide covers how to monitor PicoClaw in production environments.

## Overview

Effective monitoring helps you:
- Track system health and performance
- Detect issues before they impact users
- Understand usage patterns
- Plan capacity

## Monitoring Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PicoClaw      │───>│   Prometheus    │───>│    Grafana      │
│   Gateway       │    │   (Metrics)     │    │   (Dashboard)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         │
         v
┌─────────────────┐    ┌─────────────────┐
│   Health        │───>│   Alerting      │
│   Endpoints     │    │   System        │
└─────────────────┘    └─────────────────┘
```

## Metrics Collection

### Prometheus Integration

PicoClaw exposes metrics in Prometheus format at `/metrics`:

```bash
curl http://localhost:18790/metrics
```

**Available Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `picoclaw_messages_total` | Counter | Total messages processed |
| `picoclaw_messages_active` | Gauge | Currently active conversations |
| `picoclaw_tool_calls_total` | Counter | Tool invocations by type |
| `picoclaw_provider_requests_total` | Counter | Provider API requests |
| `picoclaw_provider_errors_total` | Counter | Provider API errors |
| `picoclaw_session_count` | Gauge | Active sessions |
| `picoclaw_response_time_seconds` | Histogram | Response latency |

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'picoclaw'
    static_configs:
      - targets: ['localhost:18790']
    scrape_interval: 15s
    metrics_path: /metrics
```

### Enable Metrics

Add to your configuration:

```json
{
  "gateway": {
    "metrics": {
      "enabled": true,
      "path": "/metrics"
    }
  }
}
```

## Grafana Dashboards

### Sample Dashboard Configuration

Import this dashboard configuration:

```json
{
  "dashboard": {
    "title": "PicoClaw Monitoring",
    "panels": [
      {
        "title": "Messages per Minute",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(picoclaw_messages_total[1m])",
            "legendFormat": "Messages/min"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(picoclaw_response_time_seconds_bucket[5m]))",
            "legendFormat": "p95 Latency"
          }
        ]
      },
      {
        "title": "Provider Errors",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(picoclaw_provider_errors_total[5m])",
            "legendFormat": "{{provider}}"
          }
        ]
      },
      {
        "title": "Active Sessions",
        "type": "stat",
        "targets": [
          {
            "expr": "picoclaw_session_count",
            "legendFormat": "Sessions"
          }
        ]
      }
    ]
  }
}
```

## Alerting

### Prometheus Alert Rules

```yaml
# alerts.yml
groups:
  - name: picoclaw
    rules:
      - alert: PicoClawDown
        expr: up{job="picoclaw"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "PicoClaw is down"
          description: "PicoClaw instance has been down for more than 1 minute."

      - alert: HighErrorRate
        expr: rate(picoclaw_provider_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High provider error rate"
          description: "Provider {{ $labels.provider }} has high error rate."

      - alert: SlowResponses
        expr: histogram_quantile(0.95, rate(picoclaw_response_time_seconds_bucket[5m])) > 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow response times"
          description: "P95 response time exceeds 30 seconds."

      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes{job="picoclaw"} > 52428800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "PicoClaw is using more than 50MB of memory."
```

### Alertmanager Configuration

```yaml
# alertmanager.yml
route:
  receiver: 'team-notifications'
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'team-notifications'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/xxx'
        channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ .CommonAnnotations.description }}'
```

## Health Checks

### HTTP Health Monitoring

Use external services to monitor health endpoints:

```bash
# Simple health check
curl -f http://localhost:18790/health || exit 1

# Detailed health check with jq
curl -s http://localhost:18790/health | jq -r '.status' | grep -q "healthy" || exit 1
```

### Cron-based Monitoring

```bash
# /etc/cron.d/picoclaw-monitor
*/5 * * * * root curl -f http://localhost:18790/health || /usr/local/bin/alert-script.sh
```

## Log-based Monitoring

### Structured Logging

PicoClaw outputs structured logs that can be parsed:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "agent",
  "message": "Processing message",
  "session_id": "abc123",
  "duration_ms": 1234
}
```

### Log Aggregation

For Loki/Grafana stack:

```yaml
# loki/promtail.yml
clients:
  - url: http://localhost:3100/loki/api/v1/push

scrape_configs:
  - job_name: picoclaw
    static_configs:
      - targets:
          - localhost
        labels:
          job: picoclaw
          __path__: /var/log/picoclaw/*.log
```

## System Metrics

### Process Monitoring

Monitor PicoClaw process directly:

```bash
# CPU and memory usage
ps aux | grep picoclaw

# Open file descriptors
lsof -p $(pgrep picoclaw) | wc -l

# Network connections
netstat -an | grep 18790
```

### Resource Limits

Set appropriate limits:

```bash
# Systemd service with limits
[Service]
ExecStart=/usr/local/bin/picoclaw gateway
LimitNOFILE=65535
MemoryMax=100M
CPUQuota=50%
```

## Monitoring Checklist

### Basic Monitoring

- [ ] Health endpoint checks (every 30 seconds)
- [ ] Process uptime monitoring
- [ ] Log error rate monitoring
- [ ] API response time tracking

### Advanced Monitoring

- [ ] Provider latency and error rates
- [ ] Message throughput metrics
- [ ] Session count trends
- [ ] Memory usage over time
- [ ] Tool call success rates

### Alerting Setup

- [ ] Service down alerts
- [ ] High error rate alerts
- [ ] Slow response alerts
- [ ] Resource usage alerts

## Third-party Integrations

### Datadog

```yaml
# datadog.yaml
instances:
  - url: http://localhost:18790/health
    name: PicoClaw
    tags:
      - env:production
      - service:picoclaw
    collect_response_time: true
```

### New Relic

```yaml
# newrelic-infra.yml
integrations:
  - name: nri-flex
    config:
      name: picoclawHealth
      apis:
        - name: health
          url: http://localhost:18790/health
```

### Uptime Kuma

Add PicoClaw as a monitored service:
1. Create new monitor
2. URL: `http://your-server:18790/health`
3. Expected status: 200

## See Also

- [Health Endpoints](health-endpoints.md)
- [Logging Guide](logging.md)
- [Troubleshooting](troubleshooting.md)
- [Docker Deployment](../deployment/docker.md)

# Health Check Endpoints

PicoClaw provides health check endpoints for monitoring system status when running in gateway mode.

## Overview

Health endpoints allow you to monitor PicoClaw's operational status, which is essential for:

- Load balancer health checks
- Container orchestration (Kubernetes, Docker Swarm)
- Monitoring systems (Prometheus, Nagios)
- Automated alerting

## Endpoints

### Health Check

**Endpoint:** `GET /health`

Returns the overall health status of the PicoClaw gateway.

```bash
curl http://localhost:18790/health
```

**Response (Healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h30m15s"
}
```

**Response (Unhealthy):**
```json
{
  "status": "unhealthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "issues": [
    "provider_connection_failed",
    "redis_unavailable"
  ]
}
```

### Readiness Check

**Endpoint:** `GET /ready`

Returns whether PicoClaw is ready to accept requests.

```bash
curl http://localhost:18790/ready
```

**Response (Ready):**
```json
{
  "ready": true,
  "checks": {
    "config_loaded": true,
    "providers_initialized": true,
    "channels_connected": true
  }
}
```

**Response (Not Ready):**
```json
{
  "ready": false,
  "checks": {
    "config_loaded": true,
    "providers_initialized": false,
    "channels_connected": false
  }
}
```

### Liveness Check

**Endpoint:** `GET /live`

Simple endpoint to verify the process is running.

```bash
curl http://localhost:18790/live
```

**Response:**
```json
{
  "alive": true
}
```

## Configuration

Configure health endpoints in your config file:

```json
{
  "gateway": {
    "port": 18790,
    "health_check": {
      "enabled": true,
      "path": "/health",
      "port": 18791
    }
  }
}
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable health check endpoints |
| `path` | `/health` | Base path for health checks |
| `port` | Same as gateway | Separate port for health checks |

## Usage Examples

### Kubernetes Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 18790
  initialDelaySeconds: 10
  periodSeconds: 30
```

### Kubernetes Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 18790
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Docker Compose Health Check

```yaml
services:
  picoclaw:
    image: picoclaw:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:18790/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Prometheus Blackbox Exporter

```yaml
modules:
  http_2xx:
    prober: http
    http:
      valid_status_codes: [200]
      preferred_ip_protocol: ip4
```

```yaml
scrape_configs:
  - job_name: 'picoclaw_health'
    metrics_path: /probe
    params:
      module: [http_2xx]
    targets:
      - http://localhost:18790/health
```

### Load Balancer (nginx)

```nginx
upstream picoclaw {
    server 127.0.0.1:18790;
}

server {
    location /health {
        proxy_pass http://picoclaw/health;
        access_log off;
    }
}
```

### HAProxy

```
backend picoclaw
    option httpchk GET /health
    http-check expect status 200
    server picoclaw1 127.0.0.1:18790 check
```

## Status Codes

| Code | Meaning |
|------|---------|
| 200 | Healthy/Ready/Alive |
| 503 | Unhealthy/Not Ready |

## Detailed Status

For more detailed diagnostics, use the CLI status command:

```bash
picoclaw status
```

This provides comprehensive information about:
- All configured providers and their status
- Active channel connections
- Session statistics
- System resources

## Monitoring Integration

### Datadog

```yaml
# datadog.yaml
instances:
  - url: http://localhost:18790/health
    name: PicoClaw
    tags:
      - service:picoclaw
```

### New Relic

```yaml
# newrelic.yml
synthetics:
  - name: PicoClaw Health
    url: http://localhost:18790/health
    verify_ssl: false
```

### Grafana + Prometheus

Use the blackbox exporter with Grafana dashboards for visual monitoring.

## Troubleshooting

### Endpoints not responding

1. Check gateway is running:
   ```bash
   picoclaw gateway --debug
   ```

2. Verify port is accessible:
   ```bash
   curl -v http://localhost:18790/health
   ```

3. Check firewall rules

### Returns 503 status

1. Check provider configuration
2. Verify API keys are valid
3. Check network connectivity to providers

## See Also

- [Monitoring Guide](monitoring.md)
- [Troubleshooting](troubleshooting.md)
- [Docker Deployment](../deployment/docker.md)

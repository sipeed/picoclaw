# Production Security Checklist

This guide provides a comprehensive security checklist for deploying PicoClaw in production environments.

## API Key Security

### Storage

- [ ] **Never commit API keys to version control**
- [ ] Store API keys in environment variables or secure secret management
- [ ] Use separate API keys for development and production
- [ ] Rotate API keys regularly (every 90 days recommended)

### File Permissions

```bash
# Set restrictive permissions on config file
chmod 600 ~/.picoclaw/config.json

# For system-wide installation
sudo chmod 600 /etc/picoclaw/config.json
sudo chown picoclaw:picoclaw /etc/picoclaw/config.json
```

### Environment Variables

```bash
# Use environment variables instead of config file for secrets
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"

# Or use a secrets file with restricted access
cat > /etc/picoclaw/secrets.env << 'EOF'
PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-xxx
EOF
chmod 600 /etc/picoclaw/secrets.env
chown picoclaw:picoclaw /etc/picoclaw/secrets.env
```

### Docker Secrets

When using Docker Swarm or Kubernetes:

```yaml
# Docker Compose with secrets
services:
  picoclaw:
    secrets:
      - openrouter_key
    environment:
      - PICOCLAW_PROVIDERS_OPENROUTER_API_KEY_FILE=/run/secrets/openrouter_key

secrets:
  openrouter_key:
    external: true
```

## Network Security

### Firewall Configuration

```bash
# UFW (Ubuntu/Debian)
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow from 10.0.0.0/8 to any port 18790  # Internal only
sudo ufw enable

# iptables
iptables -A INPUT -p tcp --dport 18790 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 18790 -j DROP
```

### Reverse Proxy

Use a reverse proxy for TLS termination:

**Nginx:**
```nginx
server {
    listen 443 ssl http2;
    server_name picoclaw.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";

    location / {
        proxy_pass http://127.0.0.1:18790;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Caddy (automatic HTTPS):**
```caddyfile
picoclaw.example.com {
    reverse_proxy localhost:18790
}
```

### Bind to Localhost

By default, bind to localhost unless you need external access:

```json
{
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  }
}
```

## Access Control

### Channel Allowlists

Restrict who can interact with your bot:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": ["123456789", "987654321"]
    },
    "discord": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": ["123456789012345678"]
    }
  }
}
```

### Multi-Factor Authentication

For admin access, consider:
- SSH key authentication only
- VPN requirement for administrative access
- Separate admin network/VLAN

## Workspace Security

### Restrict to Workspace

Always enable workspace restrictions in production:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true,
      "workspace": "/var/lib/picoclaw/workspace"
    }
  }
}
```

### Dangerous Command Blocking

Enable and configure command deny patterns:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "rm -rf",
        "dd if=",
        "mkfs",
        "shutdown",
        "reboot",
        "init 0",
        "> /dev/sd",
        "chmod 777",
        "wget.*\\|.*sh",
        "curl.*\\|.*sh"
      ]
    }
  }
}
```

### Workspace Isolation

For multi-tenant setups, use separate workspaces:

```json
{
  "agents": {
    "list": [
      {
        "id": "agent-a",
        "workspace": "/var/lib/picoclaw/agent-a",
        "restrict_to_workspace": true
      },
      {
        "id": "agent-b",
        "workspace": "/var/lib/picoclaw/agent-b",
        "restrict_to_workspace": true
      }
    ]
  }
}
```

## System Hardening

### Dedicated User

Run PicoClaw as a dedicated non-root user:

```bash
# Create system user
sudo useradd -r -s /bin/false -d /var/lib/picoclaw picoclaw

# Set ownership
sudo chown -R picoclaw:picoclaw /var/lib/picoclaw
```

### Systemd Sandbox

Use systemd security features:

```ini
[Service]
# Basic restrictions
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true

# Advanced restrictions
ProtectKernelTunables=true
ProtectControlGroups=true
ProtectKernelModules=true
ProtectHostname=true
ProtectClock=true

# Network restrictions (if no external network needed)
# RestrictAddressFamilies=AF_UNIX

# System call filtering
SystemCallFilter=@system-service
SystemCallArchitectures=native

# Memory protection
MemoryDenyWriteExecute=true

# Allow workspace access
ReadWritePaths=/var/lib/picoclaw
ReadOnlyPaths=/etc/picoclaw
```

### Docker Security

```yaml
services:
  picoclaw:
    # Run as non-root (default in official image)
    user: "1000:1000"

    # Read-only root filesystem
    read_only: true

    # Drop all capabilities
    cap_drop:
      - ALL

    # Security options
    security_opt:
      - no-new-privileges:true

    # Resource limits
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '1.0'

    # Temporary filesystem for writable paths
    tmpfs:
      - /tmp:size=10M,mode=1777
```

## Monitoring and Logging

### Log Configuration

Enable comprehensive logging:

```bash
# View logs
journalctl -u picoclaw -f

# For Docker
docker logs -f picoclaw-gateway
```

### Audit Trail

Log all tool executions:

```bash
# Debug mode shows all tool calls
picoclaw gateway --debug
```

### Monitoring Setup

Monitor key metrics:

```bash
# Process monitoring
[ -f /var/run/picoclaw.pid ] && ps -p $(cat /var/run/picoclaw.pid) > /dev/null

# Health check endpoint
curl -sf http://localhost:18790/health || echo "PicoClaw unhealthy"

# Memory usage
ps -o rss= -p $(pgrep -f "picoclaw gateway") | awk '{print $1/1024 " MB"}'
```

### Log Rotation

Configure log rotation for long-running deployments:

```bash
# /etc/logrotate.d/picoclaw
/var/log/picoclaw/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 picoclaw picoclaw
}
```

## Updates and Maintenance

### Regular Updates

- [ ] Update PicoClaw regularly for security patches
- [ ] Subscribe to release notifications
- [ ] Test updates in staging before production

```bash
# Check current version
picoclaw version

# Update binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64
chmod +x picoclaw-linux-amd64
sudo mv picoclaw-linux-amd64 /usr/local/bin/picoclaw
```

### Dependency Updates

For source installations:

```bash
# Update Go dependencies
make update-deps
make build
```

### Backup Strategy

```bash
# Backup configuration and workspace
tar czf picoclaw-backup-$(date +%Y%m%d).tar.gz \
  /etc/picoclaw \
  /var/lib/picoclaw

# Store backups securely (encrypted)
gpg -c picoclaw-backup-$(date +%Y%m%d).tar.gz
```

## Security Checklist Summary

### Before Deployment

- [ ] API keys stored securely (not in config committed to git)
- [ ] Config file has restrictive permissions (600)
- [ ] Running as non-root user
- [ ] Workspace restriction enabled
- [ ] Command deny patterns enabled
- [ ] Channel allowlists configured
- [ ] Firewall configured
- [ ] HTTPS enabled (if external access needed)

### After Deployment

- [ ] Health check endpoint accessible
- [ ] Logs being captured
- [ ] Monitoring configured
- [ ] Backup strategy in place
- [ ] Update schedule established

### Regular Maintenance

- [ ] Rotate API keys (every 90 days)
- [ ] Review access logs
- [ ] Apply security updates
- [ ] Test backup restoration
- [ ] Audit user access

## Incident Response

### If API Key is Compromised

1. Immediately revoke the compromised key at the provider
2. Generate a new key
3. Update configuration
4. Restart PicoClaw
5. Review logs for unauthorized usage

### If System is Compromised

1. Isolate the system from network
2. Preserve logs for investigation
3. Revoke all API keys
4. Rebuild from clean backup
5. Update all credentials
6. Review and patch vulnerability

## Additional Resources

- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [OWASP Security Guidelines](https://owasp.org/www-project-web-security-testing-guide/)
- [Linux Hardening Guide](https://madaidans-insecurities.github.io/guides/linux-hardening.html)

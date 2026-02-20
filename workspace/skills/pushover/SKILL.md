---
name: pushover
description: Send notifications via Pushover.
metadata: {"picoclaw":{"emoji":"🔔","requires":{"config":["channels.pushover"]}}}
---

# Pushover

Pushover is a notification service. Use the pushover tool to send push notifications to your devices.

## Configuration

```json
{
  "channels": {
    "pushover": {
      "enabled": true,
      "app_token": "your-app-token",
      "user_key": "your-user-key"
    }
  }
}
```

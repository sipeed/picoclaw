---
name: pushover
description: Send push notifications to your phone via Pushover.
metadata: {"picoclaw":{"emoji":"📱","requires":{"config":["channels.pushover"]}}}
---

# Pushover

Send push notifications to your iPhone/Android via Pushover.

## Usage

Use the `pushover` tool to send notifications:

```json
{
  "message": "Your notification message here"
}
```

## When to Use

- Send heartbeat status notifications
- Alert yourself of important events
- Notify when long-running tasks complete

## Setup

Configure in `config.json`:

```json
{
  "channels": {
    "pushover": {
      "enabled": true,
      "app_token": "YOUR_APP_TOKEN",
      "user_key": "YOUR_USER_KEY"
    }
  }
}
```

Get tokens from https://pushover.net/

> Back to [README](../../../README.md)

# WeCom

PicoClaw exposes WeCom as a single `channels.wecom` channel built on the official WeCom AI Bot WebSocket API.
This replaces the legacy `wecom`, `wecom_app`, and `wecom_aibot` split with one unified configuration model.

> No public webhook callback URL is required. PicoClaw opens an outbound WebSocket connection to WeCom.

## What This Channel Supports

- Direct chat and group chat delivery
- Channel-side streaming replies over WeCom's AI Bot protocol
- Incoming text, voice, image, file, video, and mixed messages
- Outbound text and media replies (`image`, `file`, `voice`, `video`)
- QR-based onboarding via Web UI or CLI
- Shared allowlist and `reasoning_channel_id` routing

---

## Quick Start

### Option 1: Web UI QR Binding (Recommended)

Open the Web UI, navigate to **Channels → WeCom**, and click the QR binding button. Scan the QR code with WeCom and confirm in the app — credentials are saved automatically.

<p align="center">
<img src="../../../assets/wecom-qr-binding.jpg" alt="WeCom QR binding in Web UI" width="600">
</p>

### Option 2: CLI QR Login

Run:

```bash
picoclaw auth wecom
```

The command:
1. Requests a QR code from WeCom and prints it in the terminal
2. Also prints a **QR Code Link** you can open in a browser if the terminal QR is hard to scan
3. Polls for confirmation — after scanning, you must also **confirm the login inside the WeCom app**
4. On success, writes `bot_id` and `secret` into `channels.wecom` and saves the config

The default timeout is **5 minutes**. Use `--timeout` to extend it:

```bash
picoclaw auth wecom --timeout 10m
```

> ⚠️ Scanning the QR code is not enough — you must also tap **Confirm** inside the WeCom app, otherwise the command will time out.

### Option 3: Configure Manually

If you already have a `bot_id` and `secret` from the WeCom AI Bot platform, configure directly:

```json
{
  "channel_list": {
    "wecom": {
      "enabled": true,
      "type": "wecom",
      "bot_id": "YOUR_BOT_ID",
      "secret": "YOUR_SECRET",
      "websocket_url": "wss://openws.work.weixin.qq.com",
      "send_thinking_message": true,
      "allow_from": [],
      "reasoning_channel_id": ""
    }
  }
}
```

---

## Configuration

| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | `false` | Enable the WeCom channel. |
| `bot_id` | string | — | WeCom AI Bot identifier. Required when enabled. |
| `secret` | string | — | WeCom AI Bot secret. Stored encrypted in `.security.yml`. Required when enabled. |
| `websocket_url` | string | `wss://openws.work.weixin.qq.com` | WeCom WebSocket endpoint. |
| `send_thinking_message` | bool | `true` | Send a `Processing...` message before the streamed reply begins. |
| `allow_from` | array | `[]` | Sender allowlist. Empty means allow all senders. |
| `reasoning_channel_id` | string | `""` | Optional chat ID to route reasoning/thinking output to a separate conversation. |

### Environment Variables

All fields can be overridden via environment variables with the prefix `PICOCLAW_CHANNELS_WECOM_`:

| Environment Variable | Corresponding Field |
| -------------------- | ------------------- |
| `PICOCLAW_CHANNELS_WECOM_ENABLED` | `enabled` |
| `PICOCLAW_CHANNELS_WECOM_BOT_ID` | `bot_id` |
| `PICOCLAW_CHANNELS_WECOM_SECRET` | `secret` |
| `PICOCLAW_CHANNELS_WECOM_WEBSOCKET_URL` | `websocket_url` |
| `PICOCLAW_CHANNELS_WECOM_SEND_THINKING_MESSAGE` | `send_thinking_message` |
| `PICOCLAW_CHANNELS_WECOM_ALLOW_FROM` | `allow_from` |
| `PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID` | `reasoning_channel_id` |

---

## Runtime Behavior

- PicoClaw maintains an active WeCom turn so streaming replies can continue on the same stream when possible.
- Streaming replies have a maximum duration of **5.5 minutes** and a minimum send interval of **500ms**.
- If streaming is no longer available, replies fall back to active push delivery.
- Chat route associations expire after **30 minutes** of inactivity.
- Incoming media is downloaded into the local media store before being passed to the agent.
- Outbound media is uploaded to WeCom as a temporary file and then sent as a media message.
- Duplicate messages are detected and suppressed (ring buffer of last 1000 message IDs).

---

## Migration from Legacy WeCom Config

| Previous config | Migration |
| --------------- | --------- |
| `channels.wecom` (webhook bot) | Replace with `channels.wecom` using `bot_id` + `secret`. |
| `channels.wecom_app` | Remove. Use `channels.wecom` instead. |
| `channels.wecom_aibot` | Move `bot_id` and `secret` to `channels.wecom`. |
| `token`, `encoding_aes_key`, `webhook_url`, `webhook_path` | No longer used. Remove from config. |
| `corp_id`, `corp_secret`, `agent_id` | No longer used. Remove from config. |
| `welcome_message`, `processing_message`, `max_steps` | No longer part of the WeCom channel config. |

---

## Troubleshooting

### QR binding times out

- After scanning the QR code, you must also **confirm the login inside the WeCom app**. Scanning alone is not enough.
- Re-run with a larger `--timeout`: `picoclaw auth wecom --timeout 10m`
- If the QR code in the terminal is hard to scan, use the **QR Code Link** printed below it to open in a browser.

### QR code expired

- The QR code has a limited validity. Re-run `picoclaw auth wecom` to get a fresh one.

### WebSocket connection fails

- Verify `bot_id` and `secret` are correct.
- Confirm the host can reach `wss://openws.work.weixin.qq.com` (outbound WebSocket, no inbound port needed).

### Replies do not arrive

- Check whether `allow_from` is blocking the sender.
- Check that `channels.wecom.bot_id` and `channels.wecom.secret` are set and non-empty.

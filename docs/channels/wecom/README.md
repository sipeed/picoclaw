> Back to [README](../../../README.md)

# WeCom

PicoClaw now exposes WeCom as a single `channels.wecom` channel built on the official WeCom AI Bot WebSocket API.
This replaces the legacy `wecom`, `wecom_app`, and `wecom_aibot` split with one configuration model.

## What This Channel Supports

- Direct chat and group chat delivery
- Channel-side streaming replies over WeCom's AI Bot protocol
- Incoming text, voice, image, file, video, and mixed messages
- Outbound text and media replies (`image`, `file`, `voice`, `video`)
- QR-based CLI onboarding with `picoclaw auth wecom`
- Shared allowlist and `reasoning_channel_id` routing

> No public webhook callback URL is required for this channel. PicoClaw opens an outbound WebSocket connection to WeCom.

## Quick Start

### Option 1: QR Login From CLI

Run:

```bash
picoclaw auth wecom
```

The command prints a QR code in the terminal, waits for confirmation in WeCom, and then writes the resulting
`bot_id` and `secret` into `channels.wecom`.

Use `--timeout` if you want to wait longer:

```bash
picoclaw auth wecom --timeout 10m
```

### Option 2: Configure Manually

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
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

## Configuration

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `enabled` | bool | No | Enables the WeCom channel. |
| `bot_id` | string | Yes | WeCom AI Bot identifier. Required when the channel is enabled. |
| `secret` | string | Yes | WeCom AI Bot secret. Required when the channel is enabled. |
| `websocket_url` | string | No | WebSocket endpoint. Defaults to `wss://openws.work.weixin.qq.com`. |
| `send_thinking_message` | bool | No | Sends an initial `Processing...` chunk before the final streamed reply. Defaults to `true`. |
| `allow_from` | array | No | Sender allowlist. Empty means allow all senders. |
| `reasoning_channel_id` | string | No | Optional destination for reasoning/thinking output. |

## Runtime Behavior

- PicoClaw keeps the active WeCom turn so normal replies can continue the same stream when possible.
- If streaming is no longer available, replies fall back to active push delivery to the resolved chat route.
- Incoming media is downloaded into the media store before being handed to the agent.
- Outbound media is uploaded to WeCom in temporary chunks and then sent as a regular media message.

## Migration Notes

This branch removes the old multi-channel WeCom model.

| Previous config | Now |
| --------------- | --- |
| `channels.wecom` webhook bot | Replace with `channels.wecom` using `bot_id` + `secret`. |
| `channels.wecom_app` | Remove it and use `channels.wecom`. |
| `channels.wecom_aibot` | Move the config to `channels.wecom`. |
| `token`, `encoding_aes_key`, `webhook_url`, `webhook_path` | No longer used by the WeCom channel. |
| `corp_id`, `corp_secret`, `agent_id` | No longer used by the WeCom channel. |
| `welcome_message`, `processing_message`, `max_steps` under WeCom | No longer part of the WeCom channel config. |

## Troubleshooting

### `picoclaw auth wecom` times out

- Re-run with a larger `--timeout`.
- Make sure the QR code was confirmed inside WeCom, not only scanned.

### WebSocket connection fails

- Verify `bot_id` and `secret`.
- Confirm the host can reach `wss://openws.work.weixin.qq.com`.

### Replies do not arrive

- Check whether `allow_from` blocks the sender.
- Check launcher or startup validation for missing `channels.wecom.bot_id` / `channels.wecom.secret`.


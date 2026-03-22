# 💬 Weixin (WeChat Personal) Channel

PicoClaw supports connecting to your personal WeChat account using the official Tencent iLink API.

## 🚀 Quick Onboarding

The easiest way to set up the Weixin channel is using the interactive onboarding command:

```bash
picoclaw onboard weixin
```

This command will:
1. Request a QR code from the iLink API and display it in your terminal.
2. Wait for you to scan the QR code with your WeChat mobile app.
3. Upon approval, automatically save the generated access token to your `~/.picoclaw/config.json`.

After onboarding, you can start the gateway:

```bash
picoclaw gateway
```

---

## ⚙️ Configuration

You can also manually configure the filter rules in `config.json` under the `channels.weixin` section.

```json
{
  "channels": {
    "weixin": {
      "enabled": true,
      "token": "YOUR_WEIXIN_TOKEN",
      "allow_from": [
        "user_id_1",
        "user_id_2"
      ],
      "proxy": ""
    }
  }
}
```

### Configuration Fields

| Field | Description |
|---|---|
| `enabled` | Set to `true` to enable the channel at startup. |
| `token` | The authentication token obtained via QR login. |
| `allow_from` | (Optional) List of WeChat User IDs permitted to interact with the bot. If empty, anyone who can send messages to the connected account can trigger the bot. |
| `proxy` | (Optional) HTTP proxy address (e.g. `http://localhost:7890`) for environments where connection to `ilinkai.weixin.qq.com` is restricted. |

## ⚠️ Important Notes

- **One Account Only**: The iLink token binds to a single session. Starting a new interaction generally invalidates older tokens if another device authorizes.
- **Message Rate Limits**: To avoid getting your account restricted by WeChat anti-spam systems, avoid loop triggers or high-frequency broadcasts.

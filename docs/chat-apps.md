# 💬 Chat Apps Configuration

> Back to [README](../README.md)

## 💬 Chat Apps

Talk to your picoclaw through Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot, MaixCam, or Pico (native protocol)

> **Note**: All webhook-based channels (LINE, WeCom, etc.) are served on a single shared Gateway HTTP server (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`). There are no per-channel ports to configure. Note: Feishu uses WebSocket/SDK mode and does not use the shared HTTP webhook server.

| Channel              | Difficulty         | Description                                           | Documentation                                                                                                    |
| -------------------- | ------------------ | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ Easy            | Recommended, voice-to-text, long polling (no public IP needed) | [Docs](channels/telegram/README.md)                                                                  |
| **Discord**          | ⭐ Easy            | Socket Mode, group/DM support, rich bot ecosystem     | [Docs](channels/discord/README.md)                                                                           |
| **WhatsApp**         | ⭐ Easy            | Native (QR scan) or Bridge URL                        | [Docs](#whatsapp)                                                                                                |
| **Weixin**           | ⭐ Easy            | Native QR scan (Tencent iLink API)                    | [Docs](#weixin)                                                                            |
| **Slack**            | ⭐ Easy            | **Socket Mode** (no public IP needed), enterprise     | [Docs](channels/slack/README.md)                                                                             |
| **Matrix**           | ⭐⭐ Medium        | Federated protocol, self-hosting supported            | [Docs](channels/matrix/README.md)                                                                            |
| **QQ**               | ⭐⭐ Medium        | Official bot API, Chinese community                   | [Docs](channels/qq/README.md)                                                                                |
| **DingTalk**         | ⭐⭐ Medium        | Stream mode (no public IP needed), enterprise         | [Docs](channels/dingtalk/README.md)                                                                          |
| **LINE**             | ⭐⭐⭐ Advanced    | HTTPS Webhook required                                | [Docs](channels/line/README.md)                                                                              |
| **WeCom (企业微信)** | ⭐⭐⭐ Advanced    | Group Bot (Webhook), custom App (API), AI Bot         | [Bot](channels/wecom/wecom_bot/README.md) / [App](channels/wecom/wecom_app/README.md) / [AI Bot](channels/wecom/wecom_aibot/README.md) |
| **Feishu (飞书)**    | ⭐⭐⭐ Advanced    | Enterprise collaboration, feature-rich                | [Docs](channels/feishu/README.md)                                                                            |
| **IRC**              | ⭐⭐ Medium        | Server + TLS configuration                            | [Docs](#irc)                                                                                                     |
| **OneBot**           | ⭐⭐ Medium        | NapCat/Go-CQHTTP compatible, community ecosystem      | [Docs](channels/onebot/README.md)                                                                            |
| **MaixCam**          | ⭐ Easy            | Hardware integration channel for Sipeed AI cameras    | [Docs](channels/maixcam/README.md)                                                                           |
| **Pico**             | ⭐ Easy            | Native PicoClaw protocol channel                      |                                                                                                                  |

<a id="telegram"></a>
<details>
<summary><b>Telegram</b> (Recommended)</summary>

**1. Create a bot**

* Open Telegram, search `@BotFather`
* Send `/newbot`, follow prompts
* Copy the token

**2. Configure**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": false
    }
  }
}
```

> Get your user ID from `@userinfobot` on Telegram.

**3. Run**

```bash
picoclaw gateway
```

**4. Telegram command menu (auto-registered at startup)**

PicoClaw now keeps command definitions in one shared registry. On startup, Telegram will automatically register supported bot commands (for example `/start`, `/help`, `/show`, `/list`, `/use`) so command menu and runtime behavior stay in sync.
Telegram command menu registration remains channel-local discovery UX; generic command execution is handled centrally in the agent loop via the commands executor.

If command registration fails (network/API transient errors), the channel still starts and PicoClaw retries registration in the background.

You can also manage installed skills directly from Telegram:

- `/list skills`
- `/use <skill> <message>`
- `/use <skill>` and then send the actual request in the next message
- `/use clear`

**4. Advanced Formatting**
You can set use_markdown_v2: true to enable enhanced formatting options. This allows the bot to utilize the full range of Telegram MarkdownV2 features, including nested styles, spoilers, and custom fixed-width blocks.

</details>

<a id="discord"></a>
<details>
<summary><b>Discord</b></summary>

**1. Create a bot**

* Go to <https://discord.com/developers/applications>
* Create an application → Bot → Add Bot
* Copy the bot token

**2. Enable intents**

* In the Bot settings, enable **MESSAGE CONTENT INTENT**
* (Optional) Enable **SERVER MEMBERS INTENT** if you plan to use allow lists based on member data

**3. Get your User ID**
* Discord Settings → Advanced → enable **Developer Mode**
* Right-click your avatar → **Copy User ID**

**4. Configure**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Invite the bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Open the generated invite URL and add the bot to your server

**Optional: Group trigger mode**

By default the bot responds to all messages in a server channel. To restrict responses to @-mentions only, add:

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

You can also trigger by keyword prefixes (e.g. `!bot`):

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Run**

```bash
picoclaw gateway
```

</details>

<a id="whatsapp"></a>
<details>
<summary><b>WhatsApp</b> (native via whatsmeow)</summary>

PicoClaw can connect to WhatsApp in two ways:

- **Native (recommended):** In-process using [whatsmeow](https://github.com/tulir/whatsmeow). No separate bridge. Set `"use_native": true` and leave `bridge_url` empty. On first run, scan the QR code with WhatsApp (Linked Devices). Session is stored under your workspace (e.g. `workspace/whatsapp/`). The native channel is **optional** to keep the default binary small; build with `-tags whatsapp_native` (e.g. `make build-whatsapp-native` or `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Connect to an external WebSocket bridge. Set `bridge_url` (e.g. `ws://localhost:3001`) and keep `use_native` false.

**Configure (native)**

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

If `session_store_path` is empty, the session is stored in `<workspace>/whatsapp/`. Run `picoclaw gateway`; on first run, scan the QR code printed in the terminal with WhatsApp → Linked Devices.

</details>

<a id="weixin"></a>
<details>
<summary><b>Weixin</b> (WeChat Personal)</summary>

PicoClaw supports connecting to your personal WeChat account using the official Tencent iLink API.

**1. Login**

Run the interactive QR login flow:
```bash
picoclaw onboard weixin
```
Scan the printed QR code with your WeChat mobile app. On success, the token is saved to your config.

**2. Configure**

(Optional) Update `allow_from` with your WeChat User ID to restrict who can message the bot:
```json
{
  "channels": {
    "weixin": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**3. Run**
```bash
picoclaw gateway
```

</details>

<a id="qq"></a>
<details>
<summary><b>QQ</b></summary>

**Quick setup (recommended)**

QQ Open Platform provides a one-click setup page for OpenClaw-compatible bots:

1. Open [QQ Bot Quick Start](https://q.qq.com/qqbot/openclaw/index.html) and scan the QR code to log in
2. A bot is created automatically — copy the **App ID** and **App Secret**
3. Configure PicoClaw:

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

4. Run `picoclaw gateway` and open QQ to chat with your bot

> The App Secret is only shown once. Save it immediately — viewing it again will force a reset.
>
> Bots created via the quick setup page are initially for the creator only and do not support group chats. To enable group access, configure sandbox mode on the [QQ Open Platform](https://q.qq.com/).

**Manual setup**

If you prefer to create the bot manually:

* Log in at [QQ Open Platform](https://q.qq.com/) to register as a developer
* Create a QQ bot — customize its avatar and name
* Copy the **App ID** and **App Secret** from the bot settings
* Configure as shown above and run `picoclaw gateway`

</details>

<a id="dingtalk"></a>
<details>
<summary><b>DingTalk</b></summary>

**1. Create a bot**

* Go to [Open Platform](https://open.dingtalk.com/)
* Create an internal app
* Copy Client ID and Client Secret

**2. Configure**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Set `allow_from` to empty to allow all users, or specify DingTalk user IDs to restrict access.

**3. Run**

```bash
picoclaw gateway
```
</details>

<a id="matrix"></a>
<details>
<summary><b>Matrix</b></summary>

**1. Prepare bot account**

* Use your preferred homeserver (e.g. `https://matrix.org` or self-hosted)
* Create a bot user and obtain its access token

**2. Configure**

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. Run**

```bash
picoclaw gateway
```

For full options (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), see [Matrix Channel Configuration Guide](channels/matrix/README.md).

</details>

<a id="line"></a>
<details>
<summary><b>LINE</b></summary>

**1. Create a LINE Official Account**

- Go to [LINE Developers Console](https://developers.line.biz/)
- Create a provider → Create a Messaging API channel
- Copy **Channel Secret** and **Channel Access Token**

**2. Configure**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

> LINE webhook is served on the shared Gateway server (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`).

**3. Set up Webhook URL**

LINE requires HTTPS for webhooks. Use a reverse proxy or tunnel:

```bash
# Example with ngrok (gateway default port is 18790)
ngrok http 18790
```

Then set the Webhook URL in LINE Developers Console to `https://your-domain/webhook/line` and enable **Use webhook**.

**4. Run**

```bash
picoclaw gateway
```

> In group chats, the bot responds only when @mentioned. Replies quote the original message.

</details>

<a id="wecom"></a>
<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw supports three types of WeCom integration:

**Option 1: WeCom Bot (Bot)** - Easier setup, supports group chats
**Option 2: WeCom App (Custom App)** - More features, proactive messaging, private chat only
**Option 3: WeCom AI Bot (AI Bot)** - Official AI Bot, streaming replies, supports group & private chat

See [WeCom AI Bot Configuration Guide](channels/wecom/wecom_aibot/README.md) for detailed setup instructions.

**Quick Setup - WeCom Bot:**

**1. Create a bot**

* Go to WeCom Admin Console → Group Chat → Add Group Bot
* Copy the webhook URL (format: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Configure**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

> WeCom webhook is served on the shared Gateway server (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`).

**Quick Setup - WeCom App:**

**1. Create an app**

* Go to WeCom Admin Console → App Management → Create App
* Copy **AgentId** and **Secret**
* Go to "My Company" page, copy **CorpID**

**2. Configure receive message**

* In App details, click "Receive Message" → "Set API"
* Set URL to `http://your-server:18790/webhook/wecom-app`
* Generate **Token** and **EncodingAESKey**

**3. Configure**

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": []
    }
  }
}
```

**4. Run**

```bash
picoclaw gateway
```

> **Note**: WeCom webhook callbacks are served on the Gateway port (default 18790). Use a reverse proxy for HTTPS.

**Quick Setup - WeCom AI Bot:**

**1. Create an AI Bot**

* Go to WeCom Admin Console → App Management → AI Bot
* In the AI Bot settings, configure callback URL: `http://your-server:18790/webhook/wecom-aibot`
* Copy **Token** and click "Random Generate" for **EncodingAESKey**

**2. Configure**

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "Hello! How can I help you?",
      "processing_message": "⏳ Processing, please wait. The results will be sent shortly."
    }
  }
}
```

**3. Run**

```bash
picoclaw gateway
```

> **Note**: WeCom AI Bot uses streaming pull protocol — no reply timeout concerns. Long tasks (>30 seconds) automatically switch to `response_url` push delivery.

</details>

<a id="feishu"></a>
<details>
<summary><b>Feishu (Lark)</b></summary>

PicoClaw connects to Feishu via WebSocket/SDK mode — no public webhook URL or callback server needed.

**1. Create an app**

* Go to [Feishu Open Platform](https://open.feishu.cn/) and create an application
* In the app settings, enable the **Bot** capability
* Create a version and publish the app (the app must be published to take effect)
* Copy the **App ID** (starts with `cli_`) and **App Secret**

**2. Configure**

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

Optional fields: `encrypt_key` and `verification_token` for event encryption (recommended for production).

**3. Run and chat**

```bash
picoclaw gateway
```

Open Feishu, search for your bot name, and start chatting. You can also add the bot to a group — use `group_trigger.mention_only: true` to only respond when @mentioned.

For full options, see [Feishu Channel Configuration Guide](channels/feishu/README.md).

</details>

<a id="slack"></a>
<details>
<summary><b>Slack</b></summary>

**1. Create a Slack app**

* Go to [Slack API](https://api.slack.com/apps) and create a new app
* Under **OAuth & Permissions**, add bot scopes: `chat:write`, `app_mentions:read`, `im:history`, `im:read`, `im:write`
* Install the app to your workspace
* Copy the **Bot Token** (`xoxb-...`) and **App-Level Token** (`xapp-...`, enable Socket Mode to get this)

**2. Configure**

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
      "allow_from": []
    }
  }
}
```

**3. Run**

```bash
picoclaw gateway
```

</details>

<a id="irc"></a>
<details>
<summary><b>IRC</b></summary>

**1. Configure**

```json
{
  "channels": {
    "irc": {
      "enabled": true,
      "server": "irc.libera.chat:6697",
      "tls": true,
      "nick": "picoclaw-bot",
      "channels": ["#your-channel"],
      "password": "",
      "allow_from": []
    }
  }
}
```

Optional: `nickserv_password` for NickServ authentication, `sasl_user`/`sasl_password` for SASL auth.

**2. Run**

```bash
picoclaw gateway
```

The bot will connect to the IRC server and join the specified channels.

</details>

<a id="onebot"></a>
<details>
<summary><b>OneBot (QQ via OneBot protocol)</b></summary>

OneBot is an open protocol for QQ bots. PicoClaw connects to any OneBot v11 compatible implementation (e.g., [Lagrange](https://github.com/LagrangeDev/Lagrange.Core), [NapCat](https://github.com/NapNeko/NapCatQQ)) via WebSocket.

**1. Set up a OneBot implementation**

Install and run a OneBot v11 compatible QQ bot framework. Enable its WebSocket server.

**2. Configure**

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `ws_url` | WebSocket URL of the OneBot implementation |
| `access_token` | Access token for authentication (if configured in OneBot) |
| `reconnect_interval` | Reconnect interval in seconds (default: 5) |

**3. Run**

```bash
picoclaw gateway
```

</details>

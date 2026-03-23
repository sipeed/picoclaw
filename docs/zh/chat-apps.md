# 💬 聊天应用配置

> 返回 [README](../../README.zh.md)

## 💬 聊天应用集成 (Chat Apps)

PicoClaw 支持多种聊天平台，使您的 Agent 能够连接到任何地方。

> **注意**: 所有 Webhook 类渠道（LINE、WeCom 等）均挂载在同一个 Gateway HTTP 服务器上（`gateway.host`:`gateway.port`，默认 `127.0.0.1:18790`），无需为每个渠道单独配置端口。注意：飞书（Feishu）使用 WebSocket/SDK 模式，不通过该共享 HTTP webhook 服务器接收消息。

### 核心渠道

| 渠道                 | 设置难度    | 特性说明                                  | 文档链接                                                                                                        |
| -------------------- | ----------- | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ 简单     | 推荐，支持语音转文字，长轮询无需公网      | [查看文档](../channels/telegram/README.zh.md)                                                                 |
| **Discord**          | ⭐ 简单     | Socket Mode，支持群组/私信，Bot 生态成熟  | [查看文档](../channels/discord/README.zh.md)                                                                  |
| **WhatsApp**         | ⭐ 简单     | 原生 (QR 扫码) 或 Bridge URL              | [查看文档](#whatsapp)                                                                 |
| **微信 (Weixin)**    | ⭐ 简单     | 原生扫码（腾讯 iLink API）                | [查看文档](#weixin)                                                                   |
| **Slack**            | ⭐ 简单     | **Socket Mode** (无需公网 IP)，企业级支持 | [查看文档](../channels/slack/README.zh.md)                                                                    |
| **Matrix**           | ⭐⭐ 中等   | 联邦协议，支持自建 homeserver 与公开服务器 | [查看文档](../channels/matrix/README.zh.md)                                                                  |
| **QQ**               | ⭐⭐ 中等   | 官方机器人 API，适合国内社群              | [查看文档](../channels/qq/README.zh.md)                                                                       |
| **钉钉 (DingTalk)**  | ⭐⭐ 中等   | Stream 模式无需公网，企业办公首选         | [查看文档](../channels/dingtalk/README.zh.md)                                                                 |
| **LINE**             | ⭐⭐⭐ 较难 | 需要 HTTPS Webhook                        | [查看文档](../channels/line/README.zh.md)                                                                     |
| **企业微信 (WeCom)** | ⭐⭐⭐ 较难 | 支持群机器人(Webhook)、自建应用(API)和智能机器人(AI Bot) | [Bot 文档](../channels/wecom/wecom_bot/README.zh.md) / [App 文档](../channels/wecom/wecom_app/README.zh.md) / [AI Bot 文档](../channels/wecom/wecom_aibot/README.zh.md) |
| **飞书 (Feishu)**    | ⭐⭐⭐ 较难 | 企业级协作，功能丰富                      | [查看文档](../channels/feishu/README.zh.md)                                                                   |
| **IRC**              | ⭐⭐ 中等   | 服务器 + TLS 配置                         | [查看文档](#irc) |
| **OneBot**           | ⭐⭐ 中等   | 兼容 NapCat/Go-CQHTTP，社区生态丰富       | [查看文档](../channels/onebot/README.zh.md)                                                                   |
| **MaixCam**          | ⭐ 简单     | 专为 AI 摄像头设计的硬件集成通道          | [查看文档](../channels/maixcam/README.zh.md)                                                                  |
| **Pico**             | ⭐ 简单     | PicoClaw 原生协议通道                     |                                                                                                               |

---

<a id="telegram"></a>
<details>
<summary><b>Telegram</b>（推荐）</summary>

**1. 创建 Bot**

* 打开 Telegram，搜索 `@BotFather`
* 发送 `/newbot`，按提示操作
* 复制 Token

**2. 配置**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

> 通过 Telegram 上的 `@userinfobot` 获取你的 User ID。

**3. 运行**

```bash
picoclaw gateway
```

**4. Telegram 命令菜单（启动时自动注册）**

PicoClaw 使用统一的命令定义来源。启动时会自动将 Telegram 支持的命令（例如 `/start`、`/help`、`/show`、`/list`）注册到 Bot 命令菜单，确保菜单展示与实际行为一致。
Telegram 侧保留的是命令菜单注册能力；通用命令的实际执行统一走 Agent Loop 中的 commands executor。

如果注册因网络或 API 短暂异常失败，不会阻塞 channel 启动；系统会在后台自动重试。

</details>

<a id="discord"></a>
<details>
<summary><b>Discord</b></summary>

**1. 创建 Bot**

* 前往 <https://discord.com/developers/applications>
* 创建应用 → Bot → 添加 Bot
* 复制 Bot Token

**2. 启用 Intents**

* 在 Bot 设置中启用 **MESSAGE CONTENT INTENT**
* （可选）启用 **SERVER MEMBERS INTENT**（如需基于成员数据的白名单）

**3. 获取 User ID**

* Discord 设置 → 高级 → 启用 **开发者模式**
* 右键点击头像 → **复制用户 ID**

**4. 配置**

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

**5. 邀请 Bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* 打开生成的邀请链接，将 Bot 添加到服务器

**可选：群组触发模式**

默认情况下 Bot 会回复服务器频道中的所有消息。如需仅在 @提及时回复：

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

也可通过关键词前缀触发（如 `!bot`）：

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. 运行**

```bash
picoclaw gateway
```

</details>

<a id="whatsapp"></a>
<details>
<summary><b>WhatsApp</b>（原生 whatsmeow）</summary>

PicoClaw 支持两种 WhatsApp 连接方式：

- **原生（推荐）：** 进程内使用 [whatsmeow](https://github.com/tulir/whatsmeow)，无需独立 Bridge。设置 `"use_native": true` 并留空 `bridge_url`。首次运行时用 WhatsApp 扫描 QR 码（关联设备）。会话存储在工作区下（如 `workspace/whatsapp/`）。原生渠道为**可选**构建，使用 `-tags whatsapp_native` 编译（如 `make build-whatsapp-native` 或 `go build -tags whatsapp_native ./cmd/...`）。
- **Bridge：** 连接外部 WebSocket Bridge。设置 `bridge_url`（如 `ws://localhost:3001`），保持 `use_native` 为 false。

**配置（原生）**

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

如果 `session_store_path` 为空，会话存储在 `<workspace>/whatsapp/`。运行 `picoclaw gateway`；首次运行时在终端扫描 QR 码（WhatsApp → 关联设备）。

</details>

<a id="weixin"></a>
<details>
<summary><b>微信 (Weixin)</b></summary>

PicoClaw 通过腾讯 iLink 官方 API 支持连接微信个人号。

**1. 登录**

运行交互式扫码登录流程：
```bash
picoclaw onboard weixin
```
用微信手机端扫描打印出的二维码。登录成功后，token 会自动保存到配置文件。

**2. 配置**

（可选）在 `allow_from` 中填入你的微信用户 ID，限制可以与机器人对话的用户：
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

**3. 运行**
```bash
picoclaw gateway
```

</details>

<a id="matrix"></a>
<details>
<summary><b>Matrix</b></summary>

**1. 准备 Bot 账号**

* 使用你的 homeserver（如 `https://matrix.org` 或自建）
* 创建 Bot 用户并获取 access token

**2. 配置**

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

**3. 运行**

```bash
picoclaw gateway
```

完整选项（`device_id`、`join_on_invite`、`group_trigger`、`placeholder`、`reasoning_channel_id`）请参考 [Matrix 渠道配置指南](../channels/matrix/README.md)。

</details>

<a id="qq"></a>
<details>
<summary><b>QQ</b></summary>

**快速设置（推荐）**

QQ 开放平台提供了一键创建 OpenClaw 兼容机器人的页面：

1. 打开 [QQ 机器人快速创建](https://q.qq.com/qqbot/openclaw/index.html)，扫码登录
2. 机器人自动创建 — 复制 **App ID** 和 **App Secret**
3. 配置 PicoClaw：

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

4. 运行 `picoclaw gateway`，打开 QQ 与机器人聊天

> App Secret 仅显示一次，请立即保存 — 再次查看将强制重置。
>
> 通过快速创建页面创建的机器人初始仅限创建者使用，不支持群聊。如需启用群聊访问，请在 [QQ 开放平台](https://q.qq.com/) 配置沙箱模式。

**手动设置**

如果你更喜欢手动创建机器人：

* 登录 [QQ 开放平台](https://q.qq.com/) 注册成为开发者
* 创建 QQ 机器人 — 自定义头像和名称
* 从机器人设置中复制 **App ID** 和 **App Secret**
* 按上述方式配置并运行 `picoclaw gateway`

</details>

<a id="slack"></a>
<details>
<summary><b>Slack</b></summary>

**1. 创建 Slack App**

* 前往 [Slack API](https://api.slack.com/apps) 创建新应用
* 在 **OAuth & Permissions** 中添加 Bot 权限范围：`chat:write`、`app_mentions:read`、`im:history`、`im:read`、`im:write`
* 将应用安装到你的工作区
* 复制 **Bot Token**（`xoxb-...`）和 **App-Level Token**（`xapp-...`，启用 Socket Mode 后获取）

**2. 配置**

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

**3. 运行**

```bash
picoclaw gateway
```

</details>

<a id="irc"></a>
<details>
<summary><b>IRC</b></summary>

**1. 配置**

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

可选：`nickserv_password` 用于 NickServ 认证，`sasl_user`/`sasl_password` 用于 SASL 认证。

**2. 运行**

```bash
picoclaw gateway
```

Bot 将连接到 IRC 服务器并加入指定的频道。

</details>

<a id="dingtalk"></a>
<details>
<summary><b>钉钉 (DingTalk)</b></summary>

**1. 创建 Bot**

* 前往 [开放平台](https://open.dingtalk.com/)
* 创建内部应用
* 复制 Client ID 和 Client Secret

**2. 配置**

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

> `allow_from` 留空表示允许所有用户，或指定钉钉用户 ID 限制访问。

**3. 运行**

```bash
picoclaw gateway
```

</details>

<a id="line"></a>
<details>
<summary><b>LINE</b></summary>

**1. 创建 LINE Official Account**

- 前往 [LINE Developers Console](https://developers.line.biz/)
- 创建 Provider → 创建 Messaging API Channel
- 复制 **Channel Secret** 和 **Channel Access Token**

**2. 配置**

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

> LINE Webhook 挂载在共享 Gateway 服务器上（`gateway.host`:`gateway.port`，默认 `127.0.0.1:18790`）。

**3. 设置 Webhook URL**

LINE 要求 HTTPS Webhook。使用反向代理或隧道：

```bash
# 示例：使用 ngrok（Gateway 默认端口 18790）
ngrok http 18790
```

然后在 LINE Developers Console 中将 Webhook URL 设置为 `https://your-domain/webhook/line` 并启用 **Use webhook**。

**4. 运行**

```bash
picoclaw gateway
```

> 在群聊中，Bot 仅在被 @提及时回复。回复会引用原始消息。

</details>

<a id="feishu"></a>
<details>
<summary><b>飞书 (Feishu)</b></summary>

PicoClaw 通过 WebSocket/SDK 模式连接飞书 — 无需公网 Webhook URL 或回调服务器。

**1. 创建应用**

* 前往 [飞书开放平台](https://open.feishu.cn/) 创建应用
* 在应用设置中启用 **机器人** 能力
* 创建版本并发布应用（应用必须发布后才能生效）
* 复制 **App ID**（以 `cli_` 开头）和 **App Secret**

**2. 配置**

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

可选：`encrypt_key` 和 `verification_token` 用于事件加密（生产环境推荐）。

**3. 运行并聊天**

```bash
picoclaw gateway
```

打开飞书，搜索你的机器人名称即可开始聊天。也可以将机器人添加到群组 — 使用 `group_trigger.mention_only: true` 设置为仅在 @提及时回复。

完整选项请参考 [飞书渠道配置指南](../channels/feishu/README.zh.md)。

</details>

<a id="wecom"></a>
<details>
<summary><b>企业微信 (WeCom)</b></summary>

PicoClaw 支持三种企业微信集成方式：

**方式 1: 群机器人 (Bot)** — 设置简单，支持群聊
**方式 2: 自建应用 (App)** — 功能更多，支持主动推送，仅私聊
**方式 3: 智能机器人 (AI Bot)** — 官方 AI Bot，流式回复，支持群聊和私聊

详细设置请参考 [企业微信 AI Bot 配置指南](../channels/wecom/wecom_aibot/README.zh.md)。

**快速设置 — 群机器人：**

**1. 创建 Bot**

* 企业微信管理后台 → 群聊 → 添加群机器人
* 复制 Webhook URL（格式：`https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`）

**2. 配置**

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

> WeCom Webhook 挂载在共享 Gateway 服务器上（`gateway.host`:`gateway.port`，默认 `127.0.0.1:18790`）。

**快速设置 — 自建应用：**

**1. 创建应用**

* 企业微信管理后台 → 应用管理 → 创建应用
* 复制 **AgentId** 和 **Secret**
* 前往"我的企业"页面，复制 **CorpID**

**2. 配置接收消息**

* 在应用详情中，点击"接收消息" → "设置 API"
* 设置 URL 为 `http://your-server:18790/webhook/wecom-app`
* 生成 **Token** 和 **EncodingAESKey**

**3. 配置**

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

**4. 运行**

```bash
picoclaw gateway
```

> **注意**: WeCom Webhook 回调挂载在 Gateway 端口（默认 18790）。使用反向代理配置 HTTPS。

**快速设置 — 智能机器人 (AI Bot)：**

**1. 创建 AI Bot**

* 企业微信管理后台 → 应用管理 → AI Bot
* 在 AI Bot 设置中配置回调 URL：`http://your-server:18790/webhook/wecom-aibot`
* 复制 **Token** 并点击"随机生成" **EncodingAESKey**

**2. 配置**

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮你的？",
      "processing_message": "⏳ Processing, please wait. The results will be sent shortly."
    }
  }
}
```

**3. 运行**

```bash
picoclaw gateway
```

> **注意**: 企业微信 AI Bot 使用流式拉取协议，无回复超时问题。长任务（>30 秒）会自动切换到 `response_url` 推送投递。

</details>

<a id="onebot"></a>
<details>
<summary><b>OneBot（通过 OneBot 协议连接 QQ）</b></summary>

OneBot 是 QQ 机器人的开放协议。PicoClaw 通过 WebSocket 连接任何 OneBot v11 兼容实现（如 [Lagrange](https://github.com/LagrangeDev/Lagrange.Core)、[NapCat](https://github.com/NapNeko/NapCatQQ)）。

**1. 设置 OneBot 实现**

安装并运行 OneBot v11 兼容的 QQ 机器人框架，启用其 WebSocket 服务器。

**2. 配置**

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

| 字段 | 说明 |
|------|------|
| `ws_url` | OneBot 实现的 WebSocket URL |
| `access_token` | 认证用的访问令牌（如果在 OneBot 中配置了的话） |
| `reconnect_interval` | 重连间隔（秒）（默认：5） |

**3. 运行**

```bash
picoclaw gateway
```

</details>

<a id="maixcam"></a>
<details>
<summary><b>MaixCam</b></summary>

专为 Sipeed AI 摄像头硬件设计的集成通道。

```json
{
  "channels": {
    "maixcam": {
      "enabled": true
    }
  }
}
```

```bash
picoclaw gateway
```

</details>

# 💬 聊天應用程式設定

> 回到 [README](../project/README.zh-tw.md)

## 💬 聊天應用程式

可以使用 Telegram、Discord、WhatsApp、Matrix、QQ、DingTalk、LINE、WeCom、Feishu、Slack、IRC、OneBot、MQTT、MaixCam 或 Pico（原生通訊協定）與 PicoClaw 對話。

> **注意：** 仰賴 HTTP 回呼的頻道會共用同一個閘道 HTTP 伺服器（`gateway.host`:`gateway.port`，預設為 `127.0.0.1:18790`）。Feishu、DingTalk 及 WeCom 等使用 Socket／串流的頻道，不會仰賴共用 Webhook 伺服器接收訊息。

| 頻道                 | 難易度             | 說明                                                  | 文件                                                                                                             |
| -------------------- | ------------------ | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ 簡單            | 建議使用、語音轉文字、長輪詢（不需要公開 IP）        | [文件](../channels/telegram/README.zh-tw.md)                                                                     |
| **Discord**          | ⭐ 簡單            | Socket Mode、群組／私人訊息、完整的 Bot 生態系       | [文件](../channels/discord/README.zh-tw.md)                                                                      |
| **WhatsApp**         | ⭐ 簡單            | 原生模式（掃描 QR Code）或 Bridge URL                 | [文件](#whatsapp)                                                                                                |
| **Weixin**           | ⭐ 簡單            | 原生掃描 QR Code（騰訊 iLink API）                    | [文件](#weixin)                                                                                                  |
| **Slack**            | ⭐ 簡單            | **Socket Mode**（不需要公開 IP）、企業用途            | [文件](../channels/slack/README.zh-tw.md)                                                                        |
| **Matrix**           | ⭐⭐ 中等          | 聯邦式通訊協定、支援自行架設                          | [文件](../channels/matrix/README.zh-tw.md)                                                                       |
| **QQ**               | ⭐⭐ 中等          | 官方 Bot API、面向中國社群                            | [文件](../channels/qq/README.zh-tw.md)                                                                           |
| **DingTalk**         | ⭐⭐ 中等          | 串流模式（不需要公開 IP）、企業用途                   | [文件](../channels/dingtalk/README.zh-tw.md)                                                                     |
| **LINE**             | ⭐⭐⭐ 進階        | 必須使用 HTTPS Webhook                                | [文件](../channels/line/README.zh-tw.md)                                                                         |
| **WeCom（企業微信）** | ⭐⭐⭐ 進階       | 使用官方 AI Bot WebSocket、串流與媒體功能             | [文件](../channels/wecom/README.zh-tw.md)                                                                        |
| **Feishu（飛書）**   | ⭐⭐⭐ 進階        | 企業協作、功能完整                                    | [文件](../channels/feishu/README.zh-tw.md)                                                                       |
| **IRC**              | ⭐⭐ 中等          | 伺服器與 TLS 設定                                     | [文件](#irc)                                                                                                     |
| **OneBot**           | ⭐⭐ 中等          | 與 NapCat／Go-CQHTTP 相容、社群生態系完整             | [文件](../channels/onebot/README.zh-tw.md)                                                                       |
| **MQTT**             | ⭐ 簡單            | 任何 MQTT 使用者端皆可使用 Broker 發布／訂閱訊息     | [文件](../channels/mqtt/README.zh-tw.md)                                                                         |
| **MaixCam**          | ⭐ 簡單            | Sipeed AI 攝影機的硬體整合頻道                        | [文件](../channels/maixcam/README.zh-tw.md)                                                                      |
| **Pico**             | ⭐ 簡單            | PicoClaw 原生通訊協定頻道                             |                                                                                                                  |

<a id="telegram"></a>
<details>
<summary><b>Telegram</b>（建議）</summary>

**1. 建立 Bot**

* 開啟 Telegram 並搜尋 `@BotFather`
* 傳送 `/newbot` 並依照提示操作
* 複製 Token

**2. 設定**

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": false
    }
  }
}
```

> 可以使用 Telegram 中的 `@userinfobot` 取得使用者 ID。

**3. 執行**

```bash
picoclaw gateway
```

**4. Telegram 命令選單（啟動時自動註冊）**

PicoClaw 現在會將命令定義集中儲存在共用登錄表中。啟動時，Telegram 會自動註冊支援的 Bot 命令（例如 `/start`、`/help`、`/show`、`/list`、`/use`、`/btw`），讓命令選單與執行階段行為保持同步。
Telegram 命令選單註冊仍屬於頻道內的探索體驗；一般命令則由代理人迴圈中的 commands 執行器集中處理。

如果命令註冊因暫時的網路／API 錯誤而失敗，頻道仍會啟動，PicoClaw 也會在背景重試註冊。

也可以直接從 Telegram 檢視技能及 MCP 伺服器：

- `/list skills`
- `/list mcp`
- `/show mcp <server>`
- `/use <skill> <message>`
- `/use <skill>`，接著在下一則訊息傳送實際要求
- `/use clear`
- `/btw <question>`：立即提出不會變更作用中工作階段記錄的附帶問題；`/btw` 會視為不使用工具的查詢，不會進入一般工具執行流程

**4. 進階格式**
將 `use_markdown_v2` 設為 `true`，即可啟用更豐富的格式功能。Bot 因此能完整使用 Telegram MarkdownV2 功能，包括巢狀樣式、隱藏文字及自訂固定寬度區塊。

</details>

<a id="discord"></a>
<details>
<summary><b>Discord</b></summary>

**1. 建立 Bot**

* 前往 <https://discord.com/developers/applications>
* 建立應用程式 → [Bot] → [新增 Bot]
* 複製 Bot Token

**2. 啟用 Intents**

* 在 Bot 設定中啟用 **[訊息內容意圖]**
* （選用）如果要依成員資料使用允許清單，請啟用 **[伺服器成員意圖]**

**3. 取得使用者 ID**
* 前往 Discord [設定] → [進階] → 啟用 **[開發者模式]**
* 在大頭貼上按一下滑鼠右鍵 → **[複製使用者 ID]**

**4. 設定**

```json
{
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**5. 邀請 Bot**

* [OAuth2] → [URL 產生器]
* [範圍]：`bot`
* [Bot 權限]：[傳送訊息]、[讀取訊息記錄]
* 開啟所建立的邀請 URL，將 Bot 加入伺服器

**選用：群組觸發模式**

Bot 預設會回應伺服器頻道中的所有訊息。若只要回應 @提及，請加入：

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

也可以使用關鍵字前置字串觸發（例如 `!bot`）：

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. 執行**

```bash
picoclaw gateway
```

</details>

<a id="whatsapp"></a>
<details>
<summary><b>WhatsApp</b>（使用 whatsmeow 的原生模式）</summary>

PicoClaw 可用兩種方式連線至 WhatsApp：

- **原生模式（建議）：** 在處理程序內使用 [whatsmeow](https://github.com/tulir/whatsmeow)，不需要獨立的 Bridge。請將 `"use_native"` 設為 `true`，並將 `bridge_url` 留空。首次執行時，請使用 WhatsApp（已連結的裝置）掃描 QR Code。工作階段會儲存在工作區中（例如 `workspace/whatsapp/`）。原生頻道屬於**選用功能**，可避免增加預設二進位檔的大小；請使用 `-tags whatsapp_native` 建置（例如 `make build-whatsapp-native` 或 `go build -tags whatsapp_native ./cmd/...`）。
- **Bridge：** 連線至外部 WebSocket Bridge。請設定 `bridge_url`（例如 `ws://localhost:3001`），並將 `use_native` 維持為 false。

**設定（原生模式）**

```json
{
  "channel_list": {
    "whatsapp": {
      "enabled": true,
      "type": "whatsapp",
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

如果 `session_store_path` 留空，工作階段會儲存在 `<workspace>/whatsapp/`。執行 `picoclaw gateway`；首次執行時，請在 WhatsApp → [已連結的裝置] 中掃描終端機顯示的 QR Code。

</details>

<a id="weixin"></a>
<details>
<summary><b>Weixin</b>（WeChat 個人帳戶）</summary>

PicoClaw 支援使用騰訊官方 iLink API 連線至微信個人帳戶。

**1. 登入**

執行互動式 QR Code 登入流程：
```bash
picoclaw auth weixin
```
使用微信行動應用程式掃描畫面上的 QR Code。成功後，系統會將 Token 儲存至設定中。

**2. 設定**

（選用）在 `allow_from` 中加入微信使用者 ID，限制可傳送訊息給 Bot 的對象：
```json
{
  "channel_list": {
    "weixin": {
      "enabled": true,
      "type": "weixin",
      "token": "YOUR_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**3. 執行**
```bash
picoclaw gateway
```

</details>

<a id="qq"></a>
<details>
<summary><b>QQ</b></summary>

**快速設定（建議）**

QQ 開放平台為 OpenClaw 相容 Bot 提供按一下即可完成設定的頁面：

1. 開啟 [QQ Bot 快速開始頁面](https://q.qq.com/qqbot/openclaw/index.html)，並掃描 QR Code 登入
2. 系統會自動建立 Bot，請複製 **App ID** 與 **App Secret**
3. 設定 PicoClaw：

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

4. 執行 `picoclaw gateway`，再開啟 QQ 與 Bot 對話

> App Secret 只會顯示一次，請立即妥善儲存；再次檢視會強制重設 App Secret。
>
> 從快速設定頁面建立的 Bot 起初僅供建立者本人使用，不支援群組聊天。如需開放群組使用，請在 [QQ 開放平台](https://q.qq.com/) 設定沙盒模式。

**手動設定**

如果想手動建立 Bot：

* 登入 [QQ 開放平台](https://q.qq.com/) 並註冊為開發人員
* 建立 QQ Bot 並自訂大頭貼與名稱
* 從 Bot 設定複製 **App ID** 與 **App Secret**
* 依照上方範例完成設定，再執行 `picoclaw gateway`

</details>

<a id="dingtalk"></a>
<details>
<summary><b>DingTalk</b></summary>

**1. 建立 Bot**

* 前往 [開放平台](https://open.dingtalk.com/)
* 建立內部應用程式
* 複製 Client ID 及 Client Secret

**2. 設定**

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> 將 `allow_from` 留空即可允許所有使用者，也可以指定 DingTalk 使用者 ID 來限制存取。

**3. 執行**

```bash
picoclaw gateway
```
</details>

<a id="matrix"></a>
<details>
<summary><b>Matrix</b></summary>

**1. 準備 Bot 帳戶**

* 使用慣用的 homeserver（例如 `https://matrix.org` 或自行架設的伺服器）
* 建立 Bot 使用者並取得其 Access Token

**2. 設定**

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. 執行**

```bash
picoclaw gateway
```

完整選項（`device_id`、`join_on_invite`、`group_trigger`、`placeholder`、`reasoning_channel_id`）請參閱 [Matrix 頻道設定指南](../channels/matrix/README.zh-tw.md)。

</details>

<a id="line"></a>
<details>
<summary><b>LINE</b></summary>

**1. 建立 LINE 官方帳號**

- 前往 [LINE Developers Console](https://developers.line.biz/)
- 建立 Provider → 建立 Messaging API 頻道
- 複製 **Channel Secret** 與 **Channel Access Token**

**2. 設定**

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

> LINE Webhook 由共用的閘道伺服器提供（`gateway.host`:`gateway.port`，預設為 `127.0.0.1:18790`）。

**3. 設定 Webhook URL**

LINE 要求 Webhook 使用 HTTPS。請使用反向 Proxy 或 Tunnel：

```bash
# Example with ngrok (gateway default port is 18790)
ngrok http 18790
```

接著在 LINE Developers Console 將 Webhook URL 設為 `https://your-domain/webhook/line`，並啟用 **[使用 Webhook]**。

**4. 執行**

```bash
picoclaw gateway
```

> 在群組聊天中，Bot 只會在收到 @提及時回應，回覆也會引用原始訊息。

</details>

<a id="wecom"></a>
<details>
<summary><b>WeCom（企業微信）</b></summary>

PicoClaw 現在使用 WebSocket，將 WeCom 提供為單一 AI Bot 頻道。
不需要公開的 Webhook 回呼 URL。

完整設定參考與移轉注意事項，請參閱 [WeCom 設定指南](../channels/wecom/README.zh-tw.md)。

**快速設定（建議）**

**1. 驗證**

```bash
picoclaw auth wecom
```

此命令會顯示 QR Code、等候在 WeCom 中核准，並將 `bot_id` 與 `secret` 寫入 `channels.wecom`。

**2. 視需要手動設定**

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

**3. 執行**

```bash
picoclaw gateway
```

> 在此分支中，統一的 `channels.wecom` 設定會取代舊版 `wecom_app` 及 `wecom_aibot` 設定。

</details>

<a id="feishu"></a>
<details>
<summary><b>Feishu（Lark）</b></summary>

PicoClaw 使用 WebSocket／SDK 模式連線至 Feishu，不需要公開的 Webhook URL 或回呼伺服器。

**1. 建立應用程式**

* 前往 [Feishu Open Platform](https://open.feishu.cn/) 並建立應用程式
* 在應用程式設定中啟用 **[Bot]** 功能
* 建立版本後發布應用程式（必須發布才會生效）
* 複製 **App ID**（以 `cli_` 開頭）及 **App Secret**

**2. 設定**

```json
{
  "channel_list": {
    "feishu": {
      "enabled": true,
      "type": "feishu",
      "app_id": "cli_xxx",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

選用欄位：用於事件加密的 `encrypt_key` 及 `verification_token`（正式環境建議使用）。

**3. 執行並開始對話**

```bash
picoclaw gateway
```

開啟 Feishu、搜尋 Bot 名稱，然後開始對話。也可以將 Bot 加入群組；設定 `group_trigger.mention_only: true`，即可只在收到 @提及時回應。

完整選項請參閱 [Feishu 頻道設定指南](../channels/feishu/README.zh-tw.md)。

</details>

<a id="slack"></a>
<details>
<summary><b>Slack</b></summary>

**1. 建立 Slack 應用程式**

* 前往 [Slack API](https://api.slack.com/apps) 並建立新的應用程式
* 在 **[OAuth 與權限]** 下加入 Bot Scopes：`chat:write`、`app_mentions:read`、`im:history`、`im:read`、`im:write`
* 將應用程式安裝至工作區
* 複製 **Bot Token**（`xoxb-...`）及 **App-Level Token**（`xapp-...`；啟用 Socket Mode 後即可取得）

**2. 設定**

```json
{
  "channel_list": {
    "slack": {
      "enabled": true,
      "type": "slack",
      "bot_token": "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
      "allow_from": []
    }
  }
}
```

**3. 執行**

```bash
picoclaw gateway
```

</details>

<a id="irc"></a>
<details>
<summary><b>IRC</b></summary>

**1. 設定**

```json
{
  "channel_list": {
    "irc": {
      "enabled": true,
      "type": "irc",
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

選用：用於 NickServ 驗證的 `nickserv_password`，以及用於 SASL 驗證的 `sasl_user`／`sasl_password`。

**2. 執行**

```bash
picoclaw gateway
```

Bot 會連線至 IRC 伺服器，並加入指定頻道。

</details>

<a id="onebot"></a>
<details>
<summary><b>OneBot（使用 OneBot 通訊協定連線至 QQ）</b></summary>

OneBot 是 QQ Bot 的開放通訊協定。PicoClaw 會使用 WebSocket 連線至任何與 OneBot v11 相容的實作（例如 [Lagrange](https://github.com/LagrangeDev/Lagrange.Core)、[NapCat](https://github.com/NapNeko/NapCatQQ)）。

**1. 設定 OneBot 實作**

安裝並執行與 OneBot v11 相容的 QQ Bot 框架，再啟用其 WebSocket 伺服器。

**2. 設定**

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://127.0.0.1:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| 欄位 | 說明 |
|------|------|
| `ws_url` | OneBot 實作的 WebSocket URL |
| `access_token` | 用於驗證的 Access Token（若已在 OneBot 中設定） |
| `reconnect_interval` | 重新連線間隔秒數（預設：5） |

**3. 執行**

```bash
picoclaw gateway
```

</details>

<a id="mqtt"></a>
<details>
<summary><b>MQTT</b></summary>

任何 MQTT 使用者端都能使用 Broker 與 PicoClaw 通訊。裝置或服務會將要求發布至 Broker；PicoClaw 訂閱並處理要求，再發布回應。

**1. 設定**

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "ssl://your-broker:8883",
        "agent_id": "assistant",
        "topic_prefix": "/picoclaw",
        "keep_alive": 60,
        "qos": 0
      }
    }
  }
}
```

使用者名稱與密碼會放在 `~/.picoclaw/.security.yml`：

```yaml
channel_list:
  mqtt:
    settings:
      username: your_username
      password: your_password
```

**Topic 格式**

```
{prefix}/{agent_id}/{client_id}/request    # Client → PicoClaw
{prefix}/{agent_id}/{client_id}/response   # PicoClaw → Client
```

`client_id` 由使用者端應用程式設定，用來識別不同裝置或工作階段。

**2. 執行**

```bash
picoclaw gateway
```

**3. 測試**

```bash
# Send a message
mosquitto_pub -t "/picoclaw/assistant/device1/request" \
  -m '{"text": "Hello"}'

# Subscribe to responses
mosquitto_sub -t "/picoclaw/assistant/device1/response"
```

完整設定選項請參閱 [MQTT 頻道文件](../channels/mqtt/README.zh-tw.md)。

</details>

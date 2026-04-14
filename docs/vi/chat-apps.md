# 💬 Cấu Hình Ứng Dụng Chat

> Quay lại [README](../../README.vi.md)

## 💬 Ứng Dụng Chat

Trò chuyện với picoclaw của bạn qua Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot hoặc MaixCam

> **Lưu ý**: Tất cả các kênh dựa trên webhook (LINE, WeCom, v.v.) được phục vụ trên một máy chủ HTTP Gateway chung (`gateway.host`:`gateway.port`, mặc định `127.0.0.1:18790`). Không có port riêng cho từng kênh. Lưu ý: Feishu sử dụng chế độ WebSocket/SDK và không sử dụng máy chủ HTTP webhook chung.

| Kênh                 | Độ khó             | Mô tả                                                 | Tài liệu                                                                                                         |
| -------------------- | ------------------ | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ Dễ              | Khuyến nghị, chuyển giọng nói thành văn bản, long polling (không cần IP công khai) | [Tài liệu](../channels/telegram/README.vi.md)                                              |
| **Discord**          | ⭐ Dễ              | Socket Mode, hỗ trợ nhóm/DM, hệ sinh thái bot phong phú | [Tài liệu](../channels/discord/README.vi.md)                                                                  |
| **WhatsApp**         | ⭐ Dễ              | Bản địa (quét QR) hoặc Bridge URL                     | [Tài liệu](#whatsapp)                                                                                            |
| **Weixin**           | ⭐ Dễ              | Quét QR gốc (API Tencent iLink)                       | [Tài liệu](#weixin)                                                                                              |
| **Slack**            | ⭐ Dễ              | **Socket Mode** (không cần IP công khai), doanh nghiệp | [Tài liệu](../channels/slack/README.vi.md)                                                                      |
| **Matrix**           | ⭐⭐ Trung bình    | Giao thức liên kết, hỗ trợ tự lưu trữ                | [Tài liệu](../channels/matrix/README.vi.md)                                                                     |
| **QQ**               | ⭐⭐ Trung bình    | API bot chính thức, cộng đồng Trung Quốc              | [Tài liệu](../channels/qq/README.vi.md)                                                                         |
| **DingTalk**         | ⭐⭐ Trung bình    | Chế độ Stream (không cần IP công khai), doanh nghiệp  | [Tài liệu](../channels/dingtalk/README.vi.md)                                                                   |
| **LINE**             | ⭐⭐⭐ Nâng cao    | Yêu cầu HTTPS Webhook                                 | [Tài liệu](../channels/line/README.vi.md)                                                                       |
| **WeCom (企业微信)** | ⭐⭐⭐ Nâng cao    | Bot nhóm (Webhook), ứng dụng tùy chỉnh (API), AI Bot | [Bot](../channels/wecom/wecom_bot/README.vi.md) / [App](../channels/wecom/wecom_app/README.vi.md) / [AI Bot](../channels/wecom/wecom_aibot/README.vi.md) |
| **Feishu (飞书)**    | ⭐⭐⭐ Nâng cao    | Cộng tác doanh nghiệp, nhiều tính năng                | [Tài liệu](../channels/feishu/README.vi.md)                                                                     |
| **IRC**              | ⭐⭐ Trung bình    | Máy chủ + cấu hình TLS                                | [Tài liệu](#irc) |
| **OneBot**           | ⭐⭐ Trung bình    | Tương thích NapCat/Go-CQHTTP, hệ sinh thái cộng đồng  | [Tài liệu](../channels/onebot/README.vi.md)                                                                     |
| **MaixCam**          | ⭐ Dễ              | Kênh tích hợp phần cứng cho camera AI Sipeed          | [Tài liệu](../channels/maixcam/README.vi.md)                                                                    |
| **Pico**             | ⭐ Dễ              | Kênh giao thức bản địa PicoClaw                       |                                                                                                                  |

<a id="telegram"></a>
<details>
<summary><b>Telegram</b> (Khuyến nghị)</summary>

**1. Tạo bot**

* Mở Telegram, tìm `@BotFather`
* Gửi `/newbot`, làm theo hướng dẫn
* Sao chép token

**2. Cấu hình**

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

> Lấy user ID của bạn từ `@userinfobot` trên Telegram.

**3. Chạy**

```bash
picoclaw gateway
```

**4. Menu lệnh Telegram (tự động đăng ký khi khởi động)**

PicoClaw hiện lưu trữ định nghĩa lệnh trong một registry chung. Khi khởi động, Telegram sẽ tự động đăng ký các lệnh bot được hỗ trợ (ví dụ `/start`, `/help`, `/show`, `/list`) để menu lệnh và hành vi runtime luôn đồng bộ.
Đăng ký menu lệnh Telegram vẫn là UX khám phá cục bộ của kênh; thực thi lệnh chung được xử lý tập trung trong vòng lặp agent qua commands executor.

Nếu đăng ký lệnh thất bại (lỗi tạm thời mạng/API), kênh vẫn khởi động và PicoClaw thử lại đăng ký trong nền.

</details>

<a id="discord"></a>
<details>
<summary><b>Discord</b></summary>

**1. Tạo bot**

* Truy cập <https://discord.com/developers/applications>
* Tạo ứng dụng → Bot → Add Bot
* Sao chép bot token

**2. Bật intents**

* Trong cài đặt Bot, bật **MESSAGE CONTENT INTENT**
* (Tùy chọn) Bật **SERVER MEMBERS INTENT** nếu bạn muốn sử dụng danh sách cho phép dựa trên dữ liệu thành viên

**3. Lấy User ID**
* Cài đặt Discord → Nâng cao → bật **Developer Mode**
* Nhấp chuột phải vào avatar → **Copy User ID**

**4. Cấu hình**

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

**5. Mời bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Mở URL mời được tạo và thêm bot vào server của bạn

**Tùy chọn: Chế độ kích hoạt nhóm**

Mặc định bot phản hồi tất cả tin nhắn trong kênh server. Để giới hạn phản hồi chỉ khi @mention, thêm:

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Bạn cũng có thể kích hoạt bằng tiền tố từ khóa (ví dụ: `!bot`):

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Chạy**

```bash
picoclaw gateway
```

</details>

<a id="whatsapp"></a>
<details>
<summary><b>WhatsApp</b> (native qua whatsmeow)</summary>

PicoClaw có thể kết nối WhatsApp theo hai cách:

- **Native (khuyến nghị):** In-process sử dụng [whatsmeow](https://github.com/tulir/whatsmeow). Không cần bridge riêng. Đặt `"use_native": true` và để trống `bridge_url`. Lần chạy đầu tiên, quét mã QR bằng WhatsApp (Thiết bị liên kết). Phiên được lưu trong workspace (ví dụ: `workspace/whatsapp/`). Kênh native là **tùy chọn** để giữ binary mặc định nhỏ; build với `-tags whatsapp_native` (ví dụ: `make build-whatsapp-native` hoặc `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Kết nối đến bridge WebSocket bên ngoài. Đặt `bridge_url` (ví dụ: `ws://localhost:3001`) và giữ `use_native` là false.

**Cấu hình (native)**

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

Nếu `session_store_path` trống, phiên được lưu tại `<workspace>/whatsapp/`. Chạy `picoclaw gateway`; lần chạy đầu tiên, quét mã QR hiển thị trong terminal bằng WhatsApp → Thiết bị liên kết.

</details>

<a id="weixin"></a>
<details>
<summary><b>Weixin</b> (WeChat Cá nhân)</summary>

PicoClaw hỗ trợ kết nối với tài khoản WeChat cá nhân của bạn thông qua API chính thức Tencent iLink.

**1. Đăng nhập**

Chạy luồng đăng nhập QR tương tác:
```bash
picoclaw auth weixin
```
Quét mã QR được in ra bằng ứng dụng WeChat trên điện thoại. Sau khi đăng nhập thành công, token sẽ được lưu vào cấu hình.

**2. Cấu hình**

(Tùy chọn) Thêm ID người dùng WeChat vào `allow_from` để giới hạn ai có thể nhắn tin với bot:
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

**3. Chạy**
```bash
picoclaw gateway
```

</details>

<a id="qq"></a>
<details>
<summary><b>QQ</b></summary>

**Thiết lập nhanh (khuyến nghị)**

QQ Open Platform cung cấp trang thiết lập một chạm cho bot tương thích OpenClaw:

1. Mở [QQ Bot Quick Start](https://q.qq.com/qqbot/openclaw/index.html) và quét mã QR để đăng nhập
2. Bot được tạo tự động — sao chép **App ID** và **App Secret**
3. Cấu hình PicoClaw:

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

4. Chạy `picoclaw gateway` và mở QQ để trò chuyện với bot của bạn

> App Secret chỉ hiển thị một lần. Lưu ngay lập tức — xem lại sẽ buộc phải đặt lại.
>
> Bot được tạo qua trang thiết lập nhanh ban đầu chỉ dành cho người tạo và không hỗ trợ chat nhóm. Để bật quyền truy cập nhóm, cấu hình chế độ sandbox trên [QQ Open Platform](https://q.qq.com/).

**Thiết lập thủ công**

Nếu bạn muốn tạo bot thủ công:

* Đăng nhập tại [QQ Open Platform](https://q.qq.com/) để đăng ký làm nhà phát triển
* Tạo bot QQ — tùy chỉnh avatar và tên
* Sao chép **App ID** và **App Secret** từ cài đặt bot
* Cấu hình như trên và chạy `picoclaw gateway`

</details>

<a id="dingtalk"></a>
<details>
<summary><b>DingTalk</b></summary>

**1. Tạo bot**

* Truy cập [Open Platform](https://open.dingtalk.com/)
* Tạo ứng dụng nội bộ
* Sao chép Client ID và Client Secret

**2. Cấu hình**

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

> Đặt `allow_from` trống để cho phép tất cả người dùng, hoặc chỉ định DingTalk user ID để giới hạn truy cập.

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<a id="maixcam"></a>
<details>
<summary><b>MaixCam</b></summary>

Kênh tích hợp được thiết kế đặc biệt cho phần cứng camera AI Sipeed.

```json
{
  "channel_list": {
    "maixcam": {
      "enabled": true,
      "type": "maixcam"
    }
  }
}
```

```bash
picoclaw gateway
```

</details>


<a id="matrix"></a>
<details>
<summary><b>Matrix</b></summary>

**1. Chuẩn bị tài khoản bot**

* Sử dụng homeserver ưa thích (ví dụ: `https://matrix.org` hoặc tự host)
* Tạo user bot và lấy access token

**2. Cấu hình**

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

**3. Chạy**

```bash
picoclaw gateway
```

Để xem đầy đủ các tùy chọn (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), xem [Hướng Dẫn Cấu Hình Kênh Matrix](../channels/matrix/README.md).

</details>

<a id="line"></a>
<details>
<summary><b>LINE</b></summary>

**1. Tạo Tài Khoản LINE Official**

- Truy cập [LINE Developers Console](https://developers.line.biz/)
- Tạo provider → Tạo kênh Messaging API
- Sao chép **Channel Secret** và **Channel Access Token**

**2. Cấu hình**

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

> Webhook LINE được phục vụ trên máy chủ Gateway chung (`gateway.host`:`gateway.port`, mặc định `127.0.0.1:18790`).

**3. Thiết lập Webhook URL**

LINE yêu cầu HTTPS cho webhook. Sử dụng reverse proxy hoặc tunnel:

```bash
# Ví dụ với ngrok (port mặc định gateway là 18790)
ngrok http 18790
```

Sau đó đặt Webhook URL trong LINE Developers Console thành `https://your-domain/webhook/line` và bật **Use webhook**.

**4. Chạy**

```bash
picoclaw gateway
```

> Trong chat nhóm, bot chỉ phản hồi khi được @mention. Phản hồi trích dẫn tin nhắn gốc.

</details>

<a id="wecom"></a>
<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw hỗ trợ ba loại tích hợp WeCom:

**Tùy chọn 1: WeCom Bot (Bot)** - Thiết lập dễ hơn, hỗ trợ chat nhóm
**Tùy chọn 2: WeCom App (App Tùy chỉnh)** - Nhiều tính năng hơn, nhắn tin chủ động, chỉ chat riêng
**Tùy chọn 3: WeCom AI Bot (AI Bot)** - AI Bot chính thức, phản hồi streaming, hỗ trợ chat nhóm & riêng

Xem [Hướng Dẫn Cấu Hình WeCom AI Bot](../channels/wecom/wecom_aibot/README.vi.md) để biết hướng dẫn thiết lập chi tiết.

**Thiết Lập Nhanh - WeCom Bot:**

**1. Tạo bot**

* Truy cập Console Quản Trị WeCom → Chat Nhóm → Thêm Bot Nhóm
* Sao chép URL webhook (định dạng: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Cấu hình**

```json
{
  "channel_list": {
    "wecom": {
      "enabled": true,
      "type": "wecom",
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

> Webhook WeCom được phục vụ trên máy chủ Gateway chung (`gateway.host`:`gateway.port`, mặc định `127.0.0.1:18790`).

**Thiết Lập Nhanh - WeCom App:**

**1. Tạo ứng dụng**

* Truy cập Console Quản Trị WeCom → Quản Lý App → Tạo App
* Sao chép **AgentId** và **Secret**
* Truy cập trang "Công Ty Của Tôi", sao chép **CorpID**

**2. Cấu hình nhận tin nhắn**

* Trong chi tiết App, nhấp "Nhận Tin Nhắn" → "Cấu Hình API"
* Đặt URL thành `http://your-server:18790/webhook/wecom-app`
* Tạo **Token** và **EncodingAESKey**

**3. Cấu hình**

```json
{
  "channel_list": {
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

**4. Chạy**

```bash
picoclaw gateway
```

> **Lưu ý**: Callback webhook WeCom được phục vụ trên port Gateway (mặc định 18790). Sử dụng reverse proxy cho HTTPS.

**Thiết Lập Nhanh - WeCom AI Bot:**

**1. Tạo AI Bot**

* Truy cập Console Quản Trị WeCom → Quản Lý App → AI Bot
* Trong cài đặt AI Bot, cấu hình callback URL: `http://your-server:18790/webhook/wecom-aibot`
* Sao chép **Token** và nhấp "Tạo Ngẫu Nhiên" cho **EncodingAESKey**

**2. Cấu hình**

```json
{
  "channel_list": {
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

**3. Chạy**

```bash
picoclaw gateway
```

> **Lưu ý**: WeCom AI Bot sử dụng giao thức streaming pull — không lo timeout phản hồi. Tác vụ dài (>30 giây) tự động chuyển sang gửi qua `response_url` push.

</details>

<a id="feishu"></a>
<details>
<summary><b>Feishu (Lark)</b></summary>

PicoClaw kết nối với Feishu qua chế độ WebSocket/SDK — không cần URL webhook công khai hay máy chủ callback.

**1. Tạo ứng dụng**

* Truy cập [Feishu Open Platform](https://open.feishu.cn/) và tạo ứng dụng
* Trong cài đặt ứng dụng, bật khả năng **Bot**
* Tạo phiên bản và xuất bản ứng dụng (ứng dụng phải được xuất bản mới có hiệu lực)
* Sao chép **App ID** (bắt đầu bằng `cli_`) và **App Secret**

**2. Cấu hình**

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

Tùy chọn: `encrypt_key` và `verification_token` để mã hóa sự kiện (khuyến nghị cho môi trường production).

**3. Chạy và trò chuyện**

```bash
picoclaw gateway
```

Mở Feishu, tìm tên bot của bạn và bắt đầu trò chuyện. Bạn cũng có thể thêm bot vào nhóm — sử dụng `group_trigger.mention_only: true` để chỉ phản hồi khi được @mention.

Để xem đầy đủ các tùy chọn, xem [Hướng Dẫn Cấu Hình Kênh Feishu](../channels/feishu/README.vi.md).

</details>

<a id="slack"></a>
<details>
<summary><b>Slack</b></summary>

**1. Tạo ứng dụng Slack**

* Truy cập [Slack API](https://api.slack.com/apps) và tạo ứng dụng mới
* Trong **OAuth & Permissions**, thêm các scope bot: `chat:write`, `app_mentions:read`, `im:history`, `im:read`, `im:write`
* Cài đặt ứng dụng vào workspace của bạn
* Sao chép **Bot Token** (`xoxb-...`) và **App-Level Token** (`xapp-...`, bật Socket Mode để lấy token này)

**2. Cấu hình**

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

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<a id="irc"></a>
<details>
<summary><b>IRC</b></summary>

**1. Cấu hình**

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

Tùy chọn: `nickserv_password` để xác thực NickServ, `sasl_user`/`sasl_password` để xác thực SASL.

**2. Chạy**

```bash
picoclaw gateway
```

Bot sẽ kết nối đến máy chủ IRC và tham gia các kênh đã chỉ định.

</details>

<a id="onebot"></a>
<details>
<summary><b>OneBot (QQ qua giao thức OneBot)</b></summary>

OneBot là giao thức mở cho bot QQ. PicoClaw kết nối với bất kỳ triển khai tương thích OneBot v11 nào (ví dụ: [Lagrange](https://github.com/LagrangeDev/Lagrange.Core), [NapCat](https://github.com/NapNeko/NapCatQQ)) qua WebSocket.

**1. Thiết lập triển khai OneBot**

Cài đặt và chạy framework bot QQ tương thích OneBot v11. Bật máy chủ WebSocket của nó.

**2. Cấu hình**

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

| Trường | Mô tả |
|--------|-------|
| `ws_url` | URL WebSocket của triển khai OneBot |
| `access_token` | Token truy cập để xác thực (nếu đã cấu hình trong OneBot) |
| `reconnect_interval` | Khoảng thời gian kết nối lại tính bằng giây (mặc định: 5) |

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>MaixCam</b></summary>

Kênh tích hợp được thiết kế đặc biệt cho phần cứng camera AI Sipeed.

```json
{
  "channel_list": {
    "maixcam": {
      "enabled": true,
      "type": "maixcam"
    }
  }
}
```

```bash
picoclaw gateway
```

</details>

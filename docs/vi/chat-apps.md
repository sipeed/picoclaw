# 💬 Cấu Hình Ứng Dụng Chat

> Quay lại [README](../../README.vi.md)

## 💬 Ứng Dụng Chat

Trò chuyện với picoclaw của bạn qua Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot hoặc MaixCam

> **Lưu ý**: Tất cả các kênh dựa trên webhook (LINE, WeCom, v.v.) được phục vụ trên một máy chủ HTTP Gateway chung (`gateway.host`:`gateway.port`, mặc định `127.0.0.1:18790`). Không có port riêng cho từng kênh. Lưu ý: Feishu sử dụng chế độ WebSocket/SDK và không sử dụng máy chủ HTTP webhook chung.

| Channel      | Setup                              |
| ------------ | ---------------------------------- |
| **Telegram** | Easy (just a token)                |
| **Discord**  | Easy (bot token + intents)         |
| **WhatsApp** | Easy (native: QR scan; or bridge URL) |
| **Matrix**   | Medium (homeserver + bot access token) |
| **QQ**       | Easy (AppID + AppSecret)           |
| **DingTalk** | Medium (app credentials)           |
| **LINE**     | Medium (credentials + webhook URL) |
| **WeCom AI Bot** | Medium (Token + AES key)       |
| **Feishu**   | Medium (App ID + Secret, WebSocket mode) |
| **Slack**    | Medium (Bot token + App token) |
| **IRC**      | Medium (server + TLS config)   |
| **OneBot**   | Medium (QQ via OneBot protocol) |
| **MaixCam**  | Easy (Sipeed hardware integration) |
| **Pico**     | Native PicoClaw protocol           |

<details>
<summary><b>Telegram</b> (Khuyến nghị)</summary>

**1. Tạo bot**

* Mở Telegram, tìm `@BotFather`
* Gửi `/newbot`, làm theo hướng dẫn
* Sao chép token

**2. Cấu hình**

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
  "channels": {
    "discord": {
      "enabled": true,
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
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Bạn cũng có thể kích hoạt bằng tiền tố từ khóa (ví dụ: `!bot`):

```json
{
  "channels": {
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

<details>
<summary><b>WhatsApp</b> (native qua whatsmeow)</summary>

PicoClaw có thể kết nối WhatsApp theo hai cách:

- **Native (khuyến nghị):** In-process sử dụng [whatsmeow](https://github.com/tulir/whatsmeow). Không cần bridge riêng. Đặt `"use_native": true` và để trống `bridge_url`. Lần chạy đầu tiên, quét mã QR bằng WhatsApp (Thiết bị liên kết). Phiên được lưu trong workspace (ví dụ: `workspace/whatsapp/`). Kênh native là **tùy chọn** để giữ binary mặc định nhỏ; build với `-tags whatsapp_native` (ví dụ: `make build-whatsapp-native` hoặc `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Kết nối đến bridge WebSocket bên ngoài. Đặt `bridge_url` (ví dụ: `ws://localhost:3001`) và giữ `use_native` là false.

**Cấu hình (native)**

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

Nếu `session_store_path` trống, phiên được lưu tại `<workspace>/whatsapp/`. Chạy `picoclaw gateway`; lần chạy đầu tiên, quét mã QR hiển thị trong terminal bằng WhatsApp → Thiết bị liên kết.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Tạo bot**

- Truy cập [QQ Open Platform](https://q.qq.com/#)
- Tạo ứng dụng → Lấy **AppID** và **AppSecret**

**2. Cấu hình**

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

> Đặt `allow_from` trống để cho phép tất cả người dùng, hoặc chỉ định số QQ để giới hạn truy cập.

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Tạo bot**

* Truy cập [Open Platform](https://open.dingtalk.com/)
* Tạo ứng dụng nội bộ
* Sao chép Client ID và Client Secret

**2. Cấu hình**

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

> Đặt `allow_from` trống để cho phép tất cả người dùng, hoặc chỉ định DingTalk user ID để giới hạn truy cập.

**3. Chạy**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Chuẩn bị tài khoản bot**

* Sử dụng homeserver ưa thích (ví dụ: `https://matrix.org` hoặc tự host)
* Tạo user bot và lấy access token

**2. Cấu hình**

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

**3. Chạy**

```bash
picoclaw gateway
```

Để xem đầy đủ các tùy chọn (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), xem [Hướng Dẫn Cấu Hình Kênh Matrix](docs/channels/matrix/README.md).

</details>

<details>
<summary><b>LINE</b></summary>

**1. Tạo Tài Khoản LINE Official**

- Truy cập [LINE Developers Console](https://developers.line.biz/)
- Tạo provider → Tạo kênh Messaging API
- Sao chép **Channel Secret** và **Channel Access Token**

**2. Cấu hình**

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

<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw hỗ trợ ba loại tích hợp WeCom:

**Tùy chọn 1: WeCom Bot (Bot)** - Thiết lập dễ hơn, hỗ trợ chat nhóm
**Tùy chọn 2: WeCom App (App Tùy chỉnh)** - Nhiều tính năng hơn, nhắn tin chủ động, chỉ chat riêng
**Tùy chọn 3: WeCom AI Bot (AI Bot)** - AI Bot chính thức, phản hồi streaming, hỗ trợ chat nhóm & riêng

Xem [Hướng Dẫn Cấu Hình WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) để biết hướng dẫn thiết lập chi tiết.

**Thiết Lập Nhanh - WeCom Bot:**

**1. Tạo bot**

* Truy cập Console Quản Trị WeCom → Chat Nhóm → Thêm Bot Nhóm
* Sao chép URL webhook (định dạng: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Cấu hình**

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

**4. Chạy**

```bash
picoclaw gateway
```

> **Lưu ý**: Callback webhook WeCom được phục vụ trên port Gateway (mặc định 18790). Sử dụng reverse proxy cho HTTPS.

**Thiết Lập Nhanh - WeCom AI Bot:**

**1. Tạo AI Bot**

* Truy cập Console Quản Trị WeCom → Quản Lý App → AI Bot
* Trong cài đặt AI Bot, cấu hình callback URL: `http://your-server:18791/webhook/wecom-aibot`
* Sao chép **Token** và nhấp "Tạo Ngẫu Nhiên" cho **EncodingAESKey**

**2. Cấu hình**

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "Hello! How can I help you?"
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

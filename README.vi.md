<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Trợ lý AI Siêu Nhẹ viết bằng Go</h1>

<h3>Phần cứng $10 · RAM 10MB · Khởi động 1 giây · Nào, xuất phát!</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | **Tiếng Việt** | [Français](README.fr.md) | [English](README.md)
</div>

---

🦐 **PicoClaw** là trợ lý AI cá nhân siêu nhẹ, lấy cảm hứng từ [nanobot](https://github.com/HKUDS/nanobot), được viết lại hoàn toàn bằng **Go** thông qua quá trình "tự khởi tạo" (self-bootstrapping) — nơi chính AI Agent đã tự dẫn dắt toàn bộ quá trình chuyển đổi kiến trúc và tối ưu hóa mã nguồn.

⚡️ **Cực kỳ nhẹ:** Chạy trên phần cứng chỉ **$10** với RAM **<10MB**. Tiết kiệm 99% bộ nhớ so với OpenClaw và rẻ hơn 98% so với Mac mini!

<table align="center">
<tr align="center">
<td align="center" valign="top">
<p align="center">
<img src="assets/picoclaw_mem.gif" width="360" height="240">
</p>
</td>
<td align="center" valign="top">
<p align="center">
<img src="assets/licheervnano.png" width="400" height="240">
</p>
</td>
</tr>
</table>

> [!CAUTION]
> **🚨 TUYÊN BỐ BẢO MẬT & KÊNH CHÍNH THỨC**
>
> * **KHÔNG CÓ CRYPTO:** PicoClaw **KHÔNG** có bất kỳ token/coin chính thức nào. Mọi thông tin trên `pump.fun` hoặc các sàn giao dịch khác đều là **LỪA ĐẢO**.
> * **DOMAIN CHÍNH THỨC:** Website chính thức **DUY NHẤT** là **[picoclaw.io](https://picoclaw.io)**, website công ty là **[sipeed.com](https://sipeed.com)**.
> * **Cảnh báo:** Nhiều tên miền `.ai/.org/.com/.net/...` đã bị bên thứ ba đăng ký, không phải của chúng tôi.
> * **Cảnh báo:** PicoClaw đang trong giai đoạn phát triển sớm và có thể còn các vấn đề bảo mật mạng chưa được giải quyết. Không nên triển khai lên môi trường production trước phiên bản v1.0.
> * **Lưu ý:** PicoClaw gần đây đã merge nhiều PR, dẫn đến bộ nhớ sử dụng có thể lớn hơn (10–20MB) ở các phiên bản mới nhất. Chúng tôi sẽ ưu tiên tối ưu tài nguyên khi bộ tính năng đã ổn định.


## 📢 Tin tức

2026-02-16 🎉 PicoClaw đạt 12K stars chỉ trong một tuần! Cảm ơn tất cả mọi người! PicoClaw đang phát triển nhanh hơn chúng tôi tưởng tượng. Do số lượng PR tăng cao, chúng tôi cấp thiết cần maintainer từ cộng đồng. Các vai trò tình nguyện viên và roadmap đã được công bố [tại đây](docs/ROADMAP.md) — rất mong đón nhận sự tham gia của bạn!

2026-02-13 🎉 PicoClaw đạt 5000 stars trong 4 ngày! Cảm ơn cộng đồng! Chúng tôi đang hoàn thiện **Lộ trình dự án (Roadmap)** và thiết lập **Nhóm phát triển** để đẩy nhanh tốc độ phát triển PicoClaw.
🚀 **Kêu gọi hành động:** Vui lòng gửi yêu cầu tính năng tại GitHub Discussions. Chúng tôi sẽ xem xét và ưu tiên trong cuộc họp hàng tuần.

2026-02-09 🎉 PicoClaw chính thức ra mắt! Được xây dựng trong 1 ngày để mang AI Agent đến phần cứng $10 với RAM <10MB. 🦐 PicoClaw, Lên Đường!

## ✨ Tính năng nổi bật

🪶 **Siêu nhẹ**: Bộ nhớ sử dụng <10MB — nhỏ hơn 99% so với Clawdbot (chức năng cốt lõi).

💰 **Chi phí tối thiểu**: Đủ hiệu quả để chạy trên phần cứng $10 — rẻ hơn 98% so với Mac mini.

⚡️ **Khởi động siêu nhanh**: Nhanh gấp 400 lần, khởi động trong 1 giây ngay cả trên CPU đơn nhân 0.6GHz.

🌍 **Di động thực sự**: Một file binary duy nhất chạy trên RISC-V, ARM, MIPS và x86. Một click là chạy!

🤖 **AI tự xây dựng**: Triển khai Go-native tự động — 95% mã nguồn cốt lõi được Agent tạo ra, với sự tinh chỉnh của con người.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Ngôn ngữ**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB**                                |
| **Thời gian khởi động**</br>(CPU 0.8GHz) | >500s         | >30s                     | **<1s**                                   |
| **Chi phí**                   | Mac Mini $599 | Hầu hết SBC Linux ~$50  | **Mọi bo mạch Linux**</br>**Chỉ từ $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demo

### 🛠️ Quy trình trợ lý tiêu chuẩn

<table align="center">
<tr align="center">
<th><p align="center">🧩 Lập trình Full-Stack</p></th>
<th><p align="center">🗂️ Quản lý Nhật ký & Kế hoạch</p></th>
<th><p align="center">🔎 Tìm kiếm Web & Học hỏi</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Phát triển • Triển khai • Mở rộng</td>
<td align="center">Lên lịch • Tự động hóa • Ghi nhớ</td>
<td align="center">Khám phá • Phân tích • Xu hướng</td>
</tr>
</table>

### 🐜 Triển khai sáng tạo trên phần cứng tối thiểu

PicoClaw có thể triển khai trên hầu hết mọi thiết bị Linux!

* $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) phiên bản E (Ethernet) hoặc W (WiFi6), dùng làm Trợ lý Gia đình tối giản.
* $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), hoặc $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), dùng cho quản trị Server tự động.
* $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) hoặc $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), dùng cho Giám sát thông minh.

https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4

🌟 Nhiều hình thức triển khai hơn đang chờ bạn khám phá!

## 📦 Cài đặt

### Cài đặt bằng binary biên dịch sẵn

Tải file binary cho nền tảng của bạn từ [trang Release](https://github.com/sipeed/picoclaw/releases).

### Cài đặt từ mã nguồn (có tính năng mới nhất, khuyên dùng cho phát triển)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build (không cần cài đặt)
make build

# Build cho nhiều nền tảng
make build-all

# Build và cài đặt
make install
```

## 🐳 Docker Compose

Bạn cũng có thể chạy PicoClaw bằng Docker Compose mà không cần cài đặt gì trên máy.

```bash
# 1. Clone repo
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Lần chạy đầu tiên — tự tạo docker/data/config.json rồi dừng lại
docker compose -f docker/docker-compose.yml --profile gateway up
# Container hiển thị "First-run setup complete." rồi tự dừng.

# 3. Thiết lập API Key
vim docker/data/config.json   # API key của provider, bot token, v.v.

# 4. Khởi động
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Người dùng Docker**: Theo mặc định, Gateway lắng nghe trên `127.0.0.1`, không thể truy cập từ máy chủ. Nếu bạn cần truy cập các endpoint kiểm tra sức khỏe hoặc mở cổng, hãy đặt `PICOCLAW_GATEWAY_HOST=0.0.0.0` trong môi trường của bạn hoặc cập nhật `config.json`.

```bash
# 5. Xem logs
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Dừng
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Chế độ Agent (chạy một lần)

```bash
# Đặt câu hỏi
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "2+2 bằng mấy?"

# Chế độ tương tác
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Cập nhật

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Bắt đầu nhanh

> [!TIP]
> Thiết lập API key trong `~/.picoclaw/config.json`. Lấy API key: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). Tìm kiếm web là **tùy chọn** — lấy [Tavily API](https://tavily.com) miễn phí (1000 truy vấn/tháng) hoặc [Brave Search API](https://brave.com/search/api) (2000 truy vấn/tháng).

**1. Khởi tạo**

```bash
picoclaw onboard
```

**2. Cấu hình** (`~/.picoclaw/config.json`)

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key",
      "api_base":"https://ark.cn-beijing.volces.com/api/coding/v3"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-openai-key",
      "request_timeout": 300,
      "api_base": "https://api.openai.com/v1"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gpt4"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TELEGRAM_BOT_TOKEN",
      "allow_from": []
    }
  }
}
```

> **Mới**: Định dạng cấu hình `model_list` cho phép thêm nhà cung cấp mà không cần thay đổi mã nguồn. Xem [Cấu hình Mô hình](#cấu-hình-mô-hình-model_list) để biết chi tiết.
> `request_timeout` là tùy chọn và dùng đơn vị giây. Nếu bỏ qua hoặc đặt `<= 0`, PicoClaw sẽ dùng timeout mặc định (120s).

**3. Lấy API Key**

* **Nhà cung cấp LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Tìm kiếm Web** (tùy chọn): [Brave Search](https://brave.com/search/api) — Có gói miễn phí (2000 truy vấn/tháng)

> **Lưu ý**: Xem `config.example.json` để có mẫu cấu hình đầy đủ.

**4. Trò chuyện**

```bash
picoclaw agent -m "Xin chào, bạn là ai?"
```

Vậy là xong! Bạn đã có một trợ lý AI hoạt động chỉ trong 2 phút.

---

## 💬 Tích hợp ứng dụng Chat

Trò chuyện với PicoClaw qua Telegram, Discord, DingTalk, LINE hoặc WeCom.

| Kênh | Mức độ thiết lập |
| --- | --- |
| **Telegram** | Dễ (chỉ cần token) |
| **Discord** | Dễ (bot token + intents) |
| **QQ** | Dễ (AppID + AppSecret) |
| **DingTalk** | Trung bình (app credentials) |
| **LINE** | Trung bình (credentials + webhook URL) |
| **WeCom AI Bot** | Trung bình (Token + khóa AES) |

<details>
<summary><b>Telegram</b> (Khuyên dùng)</summary>

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

> Lấy User ID từ `@userinfobot` trên Telegram.

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Discord</b></summary>

**1. Tạo bot**

* Truy cập <https://discord.com/developers/applications>
* Create an application → Bot → Add Bot
* Sao chép bot token

**2. Bật Intents**

* Trong phần Bot settings, bật **MESSAGE CONTENT INTENT**
* (Tùy chọn) Bật **SERVER MEMBERS INTENT** nếu muốn dùng danh sách cho phép theo thông tin thành viên

**3. Lấy User ID**

* Discord Settings → Advanced → bật **Developer Mode**
* Click chuột phải vào avatar → **Copy User ID**

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

**5. Mời bot vào server**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Mở URL mời được tạo và thêm bot vào server của bạn

**6. Chạy**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>QQ</b></summary>

**1. Tạo bot**

* Truy cập [QQ Open Platform](https://q.qq.com/#)
* Tạo ứng dụng → Lấy **AppID** và **AppSecret**

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

> Để `allow_from` trống để cho phép tất cả người dùng, hoặc chỉ định số QQ để giới hạn quyền truy cập.

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

> Để `allow_from` trống để cho phép tất cả người dùng, hoặc chỉ định ID để giới hạn quyền truy cập.

**3. Chạy**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. Tạo tài khoản LINE Official**

- Truy cập [LINE Developers Console](https://developers.line.biz/)
- Tạo provider → Tạo Messaging API channel
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

**3. Thiết lập Webhook URL**

LINE yêu cầu HTTPS cho webhook. Sử dụng reverse proxy hoặc tunnel:

```bash
# Ví dụ với ngrok
ngrok http 18790
```

Sau đó cài đặt Webhook URL trong LINE Developers Console thành `https://your-domain/webhook/line` và bật **Use webhook**.

**4. Chạy**

```bash
picoclaw gateway
```

> Trong nhóm chat, bot chỉ phản hồi khi được @mention. Các câu trả lời sẽ trích dẫn tin nhắn gốc.

> **Docker Compose**: Nếu bạn cần mở port webhook cục bộ, hãy thêm một rule chuyển tiếp từ port Gateway (mặc định 18790) tới host. Lưu ý: LINE webhook được phục vụ bởi Gateway HTTP chung (mặc định 127.0.0.1:18790).

</details>

<details>
<summary><b>WeCom (WeChat Work)</b></summary>

PicoClaw hỗ trợ ba loại tích hợp WeCom:

**Tùy chọn 1: WeCom Bot (Robot)** - Thiết lập dễ dàng hơn, hỗ trợ chat nhóm
**Tùy chọn 2: WeCom App (Ứng dụng Tùy chỉnh)** - Nhiều tính năng hơn, nhắn tin chủ động, chỉ chat riêng tư
**Tùy chọn 3: WeCom AI Bot (Bot Thông Minh)** - Bot AI chính thức, phản hồi streaming, hỗ trợ nhóm và riêng tư

Xem [Hướng dẫn Cấu hình WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) để biết hướng dẫn chi tiết.

**Thiết lập Nhanh - WeCom Bot:**

**1. Tạo bot**

* Truy cập Bảng điều khiển Quản trị WeCom → Chat Nhóm → Thêm Bot Nhóm
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

> **Lưu ý:** Các endpoint webhook của WeCom Bot được phục vụ bởi máy chủ Gateway HTTP dùng chung (mặc định 127.0.0.1:18790). Nếu bạn cần truy cập từ bên ngoài, hãy cấu hình reverse proxy hoặc mở cổng Gateway tương ứng.

**Thiết lập Nhanh - WeCom App:**

**1. Tạo ứng dụng**

* Truy cập Bảng điều khiển Quản trị WeCom → Quản lý Ứng dụng → Tạo Ứng dụng
* Sao chép **AgentId** và **Secret**
* Truy cập trang "Công ty của tôi", sao chép **CorpID**

**2. Cấu hình nhận tin nhắn**

* Trong chi tiết ứng dụng, nhấp vào "Nhận Tin nhắn" → "Thiết lập API"
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

> **Lưu ý**: WeCom App callback webhook được phục vụ bởi Gateway HTTP chung (mặc định 127.0.0.1:18790). Sử dụng proxy ngược để cung cấp HTTPS trong môi trường production nếu cần.

**Thiết lập Nhanh - WeCom AI Bot:**

**1. Tạo AI Bot**

* Truy cập Bảng điều khiển Quản trị WeCom → Quản lý Ứng dụng → AI Bot
* Cấu hình URL callback: `http://your-server:18791/webhook/wecom-aibot`
* Sao chép **Token** và tạo **EncodingAESKey**

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
      "welcome_message": "Xin chào! Tôi có thể giúp gì cho bạn?"
    }
  }
}
```

**3. Chạy**

```bash
picoclaw gateway
```

> **Lưu ý**: WeCom AI Bot sử dụng giao thức pull streaming — không lo timeout phản hồi. Tác vụ dài (>5,5 phút) tự động chuyển sang gửi qua `response_url`.

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Tham gia Mạng xã hội Agent

Kết nối PicoClaw với Mạng xã hội Agent chỉ bằng cách gửi một tin nhắn qua CLI hoặc bất kỳ ứng dụng Chat nào đã tích hợp.

**Đọc `https://clawdchat.ai/skill.md` và làm theo hướng dẫn để tham gia [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Cấu hình chi tiết

File cấu hình: `~/.picoclaw/config.json`

### Biến môi trường

Bạn có thể ghi đè các đường dẫn mặc định bằng cách sử dụng các biến môi trường. Điều này hữu ích cho việc cài đặt di động, triển khai container hóa hoặc chạy picoclaw như một dịch vụ hệ thống. Các biến này độc lập và kiểm soát các đường dẫn khác nhau.

| Biến              | Mô tả                                                                                                                             | Đường dẫn mặc định        |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Ghi đè đường dẫn đến file cấu hình. Điều này trực tiếp yêu cầu picoclaw tải file `config.json` nào, bỏ qua tất cả các vị trí khác. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Ghi đè thư mục gốc cho dữ liệu picoclaw. Điều này thay đổi vị trí mặc định của `workspace` và các thư mục dữ liệu khác.          | `~/.picoclaw`             |

**Ví dụ:**

```bash
# Chạy picoclaw bằng một file cấu hình cụ thể
# Đường dẫn workspace sẽ được đọc từ trong file cấu hình đó
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Chạy picoclaw với tất cả dữ liệu được lưu trữ trong /opt/picoclaw
# Cấu hình sẽ được tải từ ~/.picoclaw/config.json mặc định
# Workspace sẽ được tạo tại /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Sử dụng cả hai để có thiết lập tùy chỉnh hoàn toàn
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Cấu trúc Workspace

PicoClaw lưu trữ dữ liệu trong workspace đã cấu hình (mặc định: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Phiên hội thoại và lịch sử
├── memory/           # Bộ nhớ dài hạn (MEMORY.md)
├── state/            # Trạng thái lưu trữ (kênh cuối cùng, v.v.)
├── cron/             # Cơ sở dữ liệu tác vụ định kỳ
├── skills/           # Kỹ năng tùy chỉnh
├── AGENTS.md         # Hướng dẫn hành vi Agent
├── HEARTBEAT.md      # Prompt tác vụ định kỳ (kiểm tra mỗi 30 phút)
├── IDENTITY.md       # Danh tính Agent
├── SOUL.md           # Tâm hồn/Tính cách Agent
└── USER.md           # Tùy chọn người dùng
```

### 🔒 Hộp cát bảo mật (Security Sandbox)

PicoClaw chạy trong môi trường sandbox theo mặc định. Agent chỉ có thể truy cập file và thực thi lệnh trong phạm vi workspace.

#### Cấu hình mặc định

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Tùy chọn | Mặc định | Mô tả |
|----------|---------|-------|
| `workspace` | `~/.picoclaw/workspace` | Thư mục làm việc của agent |
| `restrict_to_workspace` | `true` | Giới hạn truy cập file/lệnh trong workspace |

#### Công cụ được bảo vệ

Khi `restrict_to_workspace: true`, các công cụ sau bị giới hạn trong sandbox:

| Công cụ | Chức năng | Giới hạn |
|---------|----------|---------|
| `read_file` | Đọc file | Chỉ file trong workspace |
| `write_file` | Ghi file | Chỉ file trong workspace |
| `list_dir` | Liệt kê thư mục | Chỉ thư mục trong workspace |
| `edit_file` | Sửa file | Chỉ file trong workspace |
| `append_file` | Thêm vào file | Chỉ file trong workspace |
| `exec` | Thực thi lệnh | Đường dẫn lệnh phải trong workspace |

#### Bảo vệ bổ sung cho Exec

Ngay cả khi `restrict_to_workspace: false`, công cụ `exec` vẫn chặn các lệnh nguy hiểm sau:

* `rm -rf`, `del /f`, `rmdir /s` — Xóa hàng loạt
* `format`, `mkfs`, `diskpart` — Định dạng ổ đĩa
* `dd if=` — Tạo ảnh đĩa
* Ghi vào `/dev/sd[a-z]` — Ghi trực tiếp lên đĩa
* `shutdown`, `reboot`, `poweroff` — Tắt/khởi động lại hệ thống
* Fork bomb `:(){ :|:& };:`

#### Ví dụ lỗi

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Tắt giới hạn (Rủi ro bảo mật)

Nếu bạn cần agent truy cập đường dẫn ngoài workspace:

**Cách 1: File cấu hình**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Cách 2: Biến môi trường**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Cảnh báo**: Tắt giới hạn này cho phép agent truy cập mọi đường dẫn trên hệ thống. Chỉ sử dụng cẩn thận trong môi trường được kiểm soát.

#### Tính nhất quán của ranh giới bảo mật

Cài đặt `restrict_to_workspace` áp dụng nhất quán trên mọi đường thực thi:

| Đường thực thi | Ranh giới bảo mật |
|----------------|-------------------|
| Agent chính | `restrict_to_workspace` ✅ |
| Subagent / Spawn | Kế thừa cùng giới hạn ✅ |
| Tác vụ Heartbeat | Kế thừa cùng giới hạn ✅ |

Tất cả đường thực thi chia sẻ cùng giới hạn workspace — không có cách nào vượt qua ranh giới bảo mật thông qua subagent hoặc tác vụ định kỳ.

### Heartbeat (Tác vụ định kỳ)

PicoClaw có thể tự động thực hiện các tác vụ định kỳ. Tạo file `HEARTBEAT.md` trong workspace:

```markdown
# Tác vụ định kỳ

- Kiểm tra email xem có tin nhắn quan trọng không
- Xem lại lịch cho các sự kiện sắp tới
- Kiểm tra dự báo thời tiết
```

Agent sẽ đọc file này mỗi 30 phút (có thể cấu hình) và thực hiện các tác vụ bằng công cụ có sẵn.

#### Tác vụ bất đồng bộ với Spawn

Đối với các tác vụ chạy lâu (tìm kiếm web, gọi API), sử dụng công cụ `spawn` để tạo **subagent**:

```markdown
# Tác vụ định kỳ

## Tác vụ nhanh (trả lời trực tiếp)
- Báo cáo thời gian hiện tại

## Tác vụ lâu (dùng spawn cho async)
- Tìm kiếm tin tức AI trên web và tóm tắt
- Kiểm tra email và báo cáo tin nhắn quan trọng
```

**Hành vi chính:**

| Tính năng | Mô tả |
|-----------|-------|
| **spawn** | Tạo subagent bất đồng bộ, không chặn heartbeat |
| **Context độc lập** | Subagent có context riêng, không có lịch sử phiên |
| **message tool** | Subagent giao tiếp trực tiếp với người dùng qua công cụ message |
| **Không chặn** | Sau khi spawn, heartbeat tiếp tục tác vụ tiếp theo |

#### Cách Subagent giao tiếp

```
Heartbeat kích hoạt
    ↓
Agent đọc HEARTBEAT.md
    ↓
Tác vụ lâu: spawn subagent
    ↓                           ↓
Tiếp tục tác vụ tiếp theo   Subagent làm việc độc lập
    ↓                           ↓
Tất cả tác vụ hoàn thành    Subagent dùng công cụ "message"
    ↓                           ↓
Phản hồi HEARTBEAT_OK       Người dùng nhận kết quả trực tiếp
```

Subagent có quyền truy cập các công cụ (message, web_search, v.v.) và có thể giao tiếp với người dùng một cách độc lập mà không cần thông qua agent chính.

**Cấu hình:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Tùy chọn | Mặc định | Mô tả |
|----------|---------|-------|
| `enabled` | `true` | Bật/tắt heartbeat |
| `interval` | `30` | Khoảng thời gian kiểm tra (phút, tối thiểu: 5) |

**Biến môi trường:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` để tắt
* `PICOCLAW_HEARTBEAT_INTERVAL=60` để thay đổi khoảng thời gian

### Nhà cung cấp (Providers)

> [!NOTE]
> Groq cung cấp dịch vụ chuyển giọng nói thành văn bản miễn phí qua Whisper. Nếu đã cấu hình Groq, tin nhắn âm thanh từ bất kỳ kênh nào sẽ được tự động chuyển thành văn bản ở cấp độ agent.

| Nhà cung cấp | Mục đích | Lấy API Key |
| --- | --- | --- |
| `gemini` | LLM (Gemini trực tiếp) | [aistudio.google.com](https://aistudio.google.com) |
| `zhipu` | LLM (Zhipu trực tiếp) | [bigmodel.cn](bigmodel.cn) |
| `volcengine`             | LLM(Volcengine trực tiếp)                   | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw)           |
| `openrouter` (Đang thử nghiệm) | LLM (khuyên dùng, truy cập mọi model) | [openrouter.ai](https://openrouter.ai) |
| `anthropic` (Đang thử nghiệm) | LLM (Claude trực tiếp) | [console.anthropic.com](https://console.anthropic.com) |
| `openai` (Đang thử nghiệm) | LLM (GPT trực tiếp) | [platform.openai.com](https://platform.openai.com) |
| `deepseek` (Đang thử nghiệm) | LLM (DeepSeek trực tiếp) | [platform.deepseek.com](https://platform.deepseek.com) |
| `groq` | LLM + **Chuyển giọng nói** (Whisper) | [console.groq.com](https://console.groq.com) |
| `qwen` | LLM (Qwen trực tiếp) | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `cerebras` | LLM (Cerebras trực tiếp) | [cerebras.ai](https://cerebras.ai) |

<details>
<summary><b>Cấu hình Zhipu</b></summary>

**1. Lấy API key**

* Lấy [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Cấu hình**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Chạy**

```bash
picoclaw agent -m "Xin chào"
```

</details>

<details>
<summary><b>Ví dụ cấu hình đầy đủ</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "BSA...",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

</details>

### Cấu hình Mô hình (model_list)

> **Tính năng mới!** PicoClaw hiện sử dụng phương pháp cấu hình **đặt mô hình vào trung tâm**. Chỉ cần chỉ định dạng `nhà cung cấp/mô hình` (ví dụ: `zhipu/glm-4.7`) để thêm nhà cung cấp mới—**không cần thay đổi mã!**

Thiết kế này cũng cho phép **hỗ trợ đa tác nhân** với lựa chọn nhà cung cấp linh hoạt:

- **Tác nhân khác nhau, nhà cung cấp khác nhau** : Mỗi tác nhân có thể sử dụng nhà cung cấp LLM riêng
- **Mô hình dự phòng** : Cấu hình mô hình chính và dự phòng để tăng độ tin cậy
- **Cân bằng tải** : Phân phối yêu cầu trên nhiều endpoint khác nhau
- **Cấu hình tập trung** : Quản lý tất cả nhà cung cấp ở một nơi

#### 📋 Tất cả Nhà cung cấp được Hỗ trợ

| Nhà cung cấp | Prefix `model` | API Base Mặc định | Giao thức | Khóa API |
|-------------|----------------|-------------------|-----------|----------|
| **OpenAI** | `openai/` | `https://api.openai.com/v1` | OpenAI | [Lấy Khóa](https://platform.openai.com) |
| **Anthropic** | `anthropic/` | `https://api.anthropic.com/v1` | Anthropic | [Lấy Khóa](https://console.anthropic.com) |
| **Zhipu AI (GLM)** | `zhipu/` | `https://open.bigmodel.cn/api/paas/v4` | OpenAI | [Lấy Khóa](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek** | `deepseek/` | `https://api.deepseek.com/v1` | OpenAI | [Lấy Khóa](https://platform.deepseek.com) |
| **Google Gemini** | `gemini/` | `https://generativelanguage.googleapis.com/v1beta` | OpenAI | [Lấy Khóa](https://aistudio.google.com/api-keys) |
| **Groq** | `groq/` | `https://api.groq.com/openai/v1` | OpenAI | [Lấy Khóa](https://console.groq.com) |
| **Moonshot** | `moonshot/` | `https://api.moonshot.cn/v1` | OpenAI | [Lấy Khóa](https://platform.moonshot.cn) |
| **Qwen (Alibaba)** | `qwen/` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI | [Lấy Khóa](https://dashscope.console.aliyun.com) |
| **NVIDIA** | `nvidia/` | `https://integrate.api.nvidia.com/v1` | OpenAI | [Lấy Khóa](https://build.nvidia.com) |
| **Ollama** | `ollama/` | `http://localhost:11434/v1` | OpenAI | Local (không cần khóa) |
| **OpenRouter** | `openrouter/` | `https://openrouter.ai/api/v1` | OpenAI | [Lấy Khóa](https://openrouter.ai/keys) |
| **VLLM** | `vllm/` | `http://localhost:8000/v1` | OpenAI | Local |
| **Cerebras** | `cerebras/` | `https://api.cerebras.ai/v1` | OpenAI | [Lấy Khóa](https://cerebras.ai) |
| **VolcEngine (Doubao)** | `volcengine/` | `https://ark.cn-beijing.volces.com/api/v3` | OpenAI | [Lấy Khóa](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **ShengsuanYun** | `shengsuanyun/` | `https://router.shengsuanyun.com/api/v1` | OpenAI | - |
| **BytePlus**        | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [Lấy Khóa](https://www.byteplus.com)                      |
| **LongCat**         | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [Lấy Key](https://longcat.chat/platform)                        |
| **ModelScope (魔搭)**| `modelscope/`    | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [Lấy Token](https://modelscope.cn/my/tokens)                    |
| **Azure OpenAI**    | `azure/`          | `https://{resource}.openai.azure.com`               | Azure     | [Lấy Khóa](https://portal.azure.com)                             |
| **Antigravity** | `antigravity/` | Google Cloud | Tùy chỉnh | Chỉ OAuth |
| **GitHub Copilot** | `github-copilot/` | `localhost:4321` | gRPC | - |

#### Cấu hình Cơ bản

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-openai-key"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-your-key"
    },
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-zhipu-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt-5.4"
    }
  }
}
```

#### Ví dụ theo Nhà cung cấp

**OpenAI**
```json
{
  "model_name": "gpt-5.4",
  "model": "openai/gpt-5.4",
  "api_key": "sk-..."
}
```

**VolcEngine (Doubao)**
```json
{
  "model_name": "ark-code-latest",
  "model": "volcengine/ark-code-latest",
  "api_key": "sk-..."
}
```

**Zhipu AI (GLM)**
```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7",
  "api_key": "your-key"
}
```

**Anthropic (với OAuth)**
```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "auth_method": "oauth"
}
```
> Chạy `picoclaw auth login --provider anthropic` để thiết lập thông tin xác thực OAuth.

**Proxy/API tùy chỉnh**
```json
{
  "model_name": "my-custom-model",
  "model": "openai/custom-model",
  "api_base": "https://my-proxy.com/v1",
  "api_key": "sk-...",
  "request_timeout": 300
}
```

#### Cân bằng Tải tải

Định cấu hình nhiều endpoint cho cùng một tên mô hình—PicoClaw sẽ tự động phân phối round-robin giữa chúng:

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_key": "sk-key1"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_key": "sk-key2"
    }
  ]
}
```

#### Chuyển đổi từ Cấu hình `providers` Cũ

Cấu hình `providers` cũ đã **ngừng sử dụng** nhưng vẫn được hỗ trợ để tương thích ngược.

**Cấu hình Cũ (đã ngừng sử dụng):**
```json
{
  "providers": {
    "zhipu": {
      "api_key": "your-key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
  "agents": {
    "defaults": {
      "provider": "zhipu",
      "model": "glm-4.7"
    }
  }
}
```

**Cấu hình Mới (khuyến nghị):**
```json
{
  "model_list": [
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  }
}
```

Xem hướng dẫn chuyển đổi chi tiết tại [docs/migration/model-list-migration.md](docs/migration/model-list-migration.md).

## Tham chiếu CLI

| Lệnh | Mô tả |
| --- | --- |
| `picoclaw onboard` | Khởi tạo cấu hình & workspace |
| `picoclaw agent -m "..."` | Trò chuyện với agent |
| `picoclaw agent` | Chế độ chat tương tác |
| `picoclaw gateway` | Khởi động gateway (cho bot chat) |
| `picoclaw status` | Hiển thị trạng thái |
| `picoclaw cron list` | Liệt kê tất cả tác vụ định kỳ |
| `picoclaw cron add ...` | Thêm tác vụ định kỳ |

### Tác vụ định kỳ / Nhắc nhở

PicoClaw hỗ trợ nhắc nhở theo lịch và tác vụ lặp lại thông qua công cụ `cron`:

* **Nhắc nhở một lần**: "Remind me in 10 minutes" (Nhắc tôi sau 10 phút) → kích hoạt một lần sau 10 phút
* **Tác vụ lặp lại**: "Remind me every 2 hours" (Nhắc tôi mỗi 2 giờ) → kích hoạt mỗi 2 giờ
* **Biểu thức Cron**: "Remind me at 9am daily" (Nhắc tôi lúc 9 giờ sáng mỗi ngày) → sử dụng biểu thức cron

Các tác vụ được lưu trong `~/.picoclaw/workspace/cron/` và được xử lý tự động.

## 🤝 Đóng góp & Lộ trình

Chào đón mọi PR! Mã nguồn được thiết kế nhỏ gọn và dễ đọc. 🤗

Lộ trình sắp được công bố...

Nhóm phát triển đang được xây dựng. Điều kiện tham gia: Ít nhất 1 PR đã được merge.

Nhóm người dùng:

Discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Xử lý sự cố

### Tìm kiếm web hiện "API 配置问题"

Điều này là bình thường nếu bạn chưa cấu hình API key cho tìm kiếm. PicoClaw sẽ cung cấp các liên kết hữu ích để tìm kiếm thủ công.

Để bật tìm kiếm web:

1. **Tùy chọn 1 (Khuyên dùng)**: Lấy API key miễn phí tại [https://brave.com/search/api](https://brave.com/search/api) (2000 truy vấn miễn phí/tháng) để có kết quả tốt nhất.
2. **Tùy chọn 2 (Không cần thẻ tín dụng)**: Nếu không có key, hệ thống tự động chuyển sang dùng **DuckDuckGo** (không cần key).

Thêm key vào `~/.picoclaw/config.json` nếu dùng Brave:

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Gặp lỗi lọc nội dung (Content Filtering)

Một số nhà cung cấp (như Zhipu) có bộ lọc nội dung nghiêm ngặt. Thử diễn đạt lại câu hỏi hoặc sử dụng model khác.

### Telegram bot báo "Conflict: terminated by other getUpdates"

Điều này xảy ra khi có một instance bot khác đang chạy. Đảm bảo chỉ có một tiến trình `picoclaw gateway` chạy tại một thời điểm.

---

## 📝 So sánh API Key

| Dịch vụ | Gói miễn phí | Trường hợp sử dụng |
| --- | --- | --- |
| **OpenRouter** | 200K tokens/tháng | Đa model (Claude, GPT-4, v.v.) |
| **Volcengine CodingPlan** | ¥9.9/tháng đầu | Tốt nhất cho người dùng Trung Quốc, nhiều mô hình SOTA (Doubao, DeepSeek, v.v.) |
| **Zhipu** | 200K tokens/tháng | Phù hợp cho người dùng Trung Quốc |
| **Brave Search** | 2000 truy vấn/tháng | Chức năng tìm kiếm web |
| **Groq** | Có gói miễn phí | Suy luận siêu nhanh (Llama, Mixtral) |
| **ModelScope** | 2000 yêu cầu/ngày | Suy luận miễn phí (Qwen, GLM, DeepSeek, v.v.) |

---

<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw Meme" width="512">
</div>

# Zalo OA Channel — Hướng dẫn tích hợp

Tích hợp Zalo Official Account (OA) vào PicoClaw, cho phép bot AI nhận và trả lời tin nhắn từ người dùng Zalo.

## Tính năng

### OA Messaging (chính)
- Nhận tin nhắn text từ user qua webhook (`user_send_text`)
- Nhận hình ảnh từ user (`user_send_image`)
- Gửi tin nhắn text reply qua OA Send Message API
- Gửi hình ảnh reply qua OA Send Message API
- Tự động refresh Access Token mỗi 80 phút

### OAuth 2.0 PKCE
- Hàm `GeneratePKCE()` tạo code_verifier + code_challenge
- Exchange authorization code lấy access_token + refresh_token
- Tool `cmd/zalo-auth/main.go` tự động hoá toàn bộ flow

### Social API (bổ sung)
- Lấy profile user (`GetUserProfile`)
- Lấy danh sách bạn bè (`GetFriendList`)
- Lấy profile qua Social API (`GetSocialUserProfile`)

## Yêu cầu quan trọng

### OA phải được Zalo duyệt

> **Đây là blocker lớn nhất.** OA mới tạo chưa được duyệt sẽ KHÔNG thể lấy Access Token qua OAuth.
> Thời gian duyệt trung bình: **1–3 ngày làm việc**.

Quy trình:
1. Tạo OA tại https://oa.zalo.me
2. Điền đầy đủ thông tin OA (tên, mô tả, ảnh đại diện, ảnh bìa)
3. Gửi yêu cầu duyệt OA
4. Chờ Zalo duyệt (1–3 ngày)
5. Sau khi duyệt → tạo App tại https://developers.zalo.me
6. Liên kết App với OA
7. Lấy Access Token

### Tạo App trên Zalo Developers

1. Vào https://developers.zalo.me → **Tạo ứng dụng mới**
2. Chọn loại: **Zalo Official Account API**
3. Lưu lại:
   - **App ID**: ID của app (vd: `YOUR_APP_ID`)
   - **App Secret**: secret key (vd: `YOUR_APP_SECRET`)
   - **OA Secret Key**: dùng để verify webhook signature (vd: `YOUR_OA_SECRET_KEY`)
4. Mục **Webhook** → điền URL: `https://your-domain.com/webhook/zalo`
5. Mục **Scopes** → bật: `send_message`, `manage_oa`

## Lấy credentials

### OA ID
Vào https://oa.zalo.me → **Cài đặt** → **Thông tin OA** → OA ID

### App ID + App Secret
Vào https://developers.zalo.me → App → **Thông tin ứng dụng**

### OA Secret Key
Vào https://developers.zalo.me → App → **Webhook** → OA Secret Key

### Access Token + Refresh Token

**Cách 1: Dùng tool `cmd/zalo-auth`** (khuyến nghị)

```bash
cd ~/picoclaw

# Thêm nginx rule cho callback (nếu chưa có)
# location /auth/zalo/callback → proxy_pass http://127.0.0.1:9999

go run -tags stdjson ./cmd/zalo-auth/
# Mở URL hiện ra trong browser → Authorize → Token tự động lưu vào config.json
```

**Cách 2: Dùng script `scripts/zalo-get-token.sh`**

```bash
bash scripts/zalo-get-token.sh
```

**Cách 3: Thủ công qua Zalo Developer Console**

1. Vào https://developers.zalo.me/app/{APP_ID}/oa → mục **Tools** hoặc **Access Token**
2. Click **"Cấp quyền"** → Authorize
3. Copy Access Token + Refresh Token

> **Lưu ý:** Access Token hết hạn sau **24 giờ**. PicoClaw tự động refresh mỗi 80 phút
> nếu có Refresh Token hợp lệ. Refresh Token hết hạn sau **3 tháng**.

## Cấu hình config.json

Thêm vào `channels` trong `docker/data/config.json`:

```json
{
  "channels": {
    "zalo": {
      "enabled": true,
      "oa_id": "YOUR_OA_ID",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "oa_secret_key": "YOUR_OA_SECRET_KEY",
      "access_token": "TOKEN_SAU_KHI_OAUTH",
      "refresh_token": "REFRESH_TOKEN_SAU_KHI_OAUTH",
      "webhook_path": "/webhook/zalo",
      "allow_from": [],
      "oauth_redirect_uri": "https://db2.moonlight.pro.vn/auth/zalo/callback"
    }
  }
}
```

| Field | Bắt buộc | Mô tả |
|-------|----------|-------|
| `enabled` | Có | Bật/tắt channel |
| `oa_id` | Không | OA ID, dùng cho tham chiếu |
| `app_id` | Có | App ID từ Zalo Developers |
| `app_secret` | Có | App Secret từ Zalo Developers |
| `oa_secret_key` | Không | OA Secret Key dùng verify webhook signature |
| `access_token` | Có* | OA Access Token (cần để gửi tin nhắn) |
| `refresh_token` | Không | Refresh Token để tự động renew access_token |
| `webhook_path` | Không | Mặc định `/webhook/zalo` |
| `allow_from` | Không | Danh sách Zalo user ID được phép. Rỗng = tất cả |
| `oauth_redirect_uri` | Không | Redirect URI cho OAuth flow |

> *Nếu `access_token` trống, channel vẫn khởi tạo (webhook nhận event) nhưng không gửi được reply.

## Setup Webhook trên Zalo Developers

### 1. Chuẩn bị server

```bash
# Nginx reverse proxy
server {
    listen 443 ssl;
    server_name db2.moonlight.pro.vn;
    ssl_certificate /etc/letsencrypt/live/db2.moonlight.pro.vn/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/db2.moonlight.pro.vn/privkey.pem;

    location /webhook/ {
        proxy_pass http://127.0.0.1:18790;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Verify trên Zalo Developers

1. Vào https://developers.zalo.me → App → **Webhook**
2. Điền URL: `https://db2.moonlight.pro.vn/webhook/zalo`
3. Click **Verify** — Zalo gửi POST tới URL, expect HTTP 200
4. Subscribe events: `user_send_text`, `user_send_image`

### 3. Test webhook

```bash
# Test challenge (GET)
curl "https://db2.moonlight.pro.vn/webhook/zalo?challenge=test123"
# Expected: test123

# Test event (POST)
curl -X POST https://db2.moonlight.pro.vn/webhook/zalo \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "user_send_text",
    "sender": {"id": "test_user_001", "display_name": "Test User"},
    "recipient": {"id": "YOUR_OA_ID"},
    "message": {"msg_id": "msg001", "text": "xin chào"},
    "timestamp": 1711234567
  }'
# Expected: HTTP 200 (gateway log sẽ hiện "Processing message from zalo")
```

## Cấu trúc file

```
pkg/channels/zalo/
├── zalo.go      # ZaloChannel: Start, Stop, Send, ServeHTTP, handleEvent, verifySignature
├── api.go       # ZaloAPI: SendTextMessage, SendImageMessage, RefreshAccessToken, OAuth helpers
├── init.go      # RegisterFactory("zalo", ...) — đăng ký channel vào registry
├── oauth.go     # GeneratePKCE() — tạo code_verifier + code_challenge cho OAuth PKCE

cmd/zalo-auth/
└── main.go      # Tool OAuth tự động: tạo PKCE → in URL → lắng nghe callback → exchange token → update config

scripts/
├── zalo-get-token.sh   # Shell script lấy token (không cần Go build)
└── zalo-oauth.sh       # Script OAuth đơn giản (manual flow)
```

### Các file đã patch trong codebase

| File | Thay đổi |
|------|----------|
| `pkg/config/config.go` | Thêm `ZaloConfig` struct với `OASecretKey` field |
| `pkg/channels/manager.go` | Thêm `initChannel("zalo", "Zalo")` trong `initChannels()` |
| `pkg/gateway/gateway.go` | Thêm `_ "github.com/sipeed/picoclaw/pkg/channels/zalo"` import |

## Troubleshooting

### Lỗi `-216: Access token is invalid`

Access token trống hoặc hết hạn.

```bash
# Kiểm tra token trong config
sudo python3 -c "
import json
with open('docker/data/config.json') as f:
    print(json.load(f)['channels']['zalo']['access_token'][:20] + '...')
"

# Lấy token mới
go run -tags stdjson ./cmd/zalo-auth/
```

### Lỗi `-14068: OA has not been granted the required permission`

OA chưa được duyệt hoặc chưa cấp quyền `send_message`.

Giải pháp:
1. Kiểm tra OA đã được duyệt tại https://oa.zalo.me
2. Vào Zalo Developers → App → Scopes → bật `send_message`
3. Re-authorize để lấy token mới với scope đúng

### Webhook trả 403 (Forbidden)

Signature verification thất bại.

```bash
# Kiểm tra oa_secret_key đã cấu hình đúng chưa
grep oa_secret_key docker/data/config.json
```

### Webhook trả 400 (Bad Request)

Body request không đúng JSON format. Kiểm tra gateway log:

```bash
docker compose -f docker/docker-compose.yml --profile gateway logs --tail=50 | grep -i zalo
```

### Webhook trả 502 (Bad Gateway)

Nginx không kết nối được gateway.

```bash
# Kiểm tra gateway đang chạy
docker ps | grep picoclaw-gateway

# Kiểm tra port binding
docker port picoclaw-gateway

# Kiểm tra gateway log
docker compose -f docker/docker-compose.yml --profile gateway logs --tail=20
```

### Domain verify thất bại trên Zalo Developers

1. Kiểm tra DNS: `host db2.moonlight.pro.vn`
2. Kiểm tra HTTPS: `curl -I https://db2.moonlight.pro.vn/webhook/zalo`
3. Kiểm tra SSL cert hợp lệ (không phải self-signed): `openssl s_client -connect db2.moonlight.pro.vn:443 -servername db2.moonlight.pro.vn </dev/null 2>/dev/null | openssl x509 -noout -issuer`
4. Nếu dùng Cloudflare proxy: SSL mode phải là **Full** hoặc **Full (strict)**

### Token refresh thất bại liên tục

Refresh token hết hạn (sau 3 tháng). Cần re-authorize:

```bash
go run -tags stdjson ./cmd/zalo-auth/
```

### Channel không khởi tạo (không có trong log)

Kiểm tra `manager.go` — block `initChannel("zalo")` phải nằm NGOÀI block `if` của channel khác:

```go
// Đúng:
if channels.IRC.Enabled && channels.IRC.Server != "" {
    m.initChannel("irc", "IRC")
}

if channels.Zalo.Enabled && channels.Zalo.AppID != "" {
    m.initChannel("zalo", "Zalo")
}

// Sai (Zalo nằm trong block IRC):
if channels.IRC.Enabled && channels.IRC.Server != "" {
    m.initChannel("irc", "IRC")
    if channels.Zalo.Enabled && channels.Zalo.AppID != "" {
        m.initChannel("zalo", "Zalo")
    }
}
```

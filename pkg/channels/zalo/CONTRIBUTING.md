# Hướng dẫn phát triển Channel mới — Lấy Zalo làm mẫu

Tài liệu này hướng dẫn cách thêm một channel messaging mới vào PicoClaw,
sử dụng Zalo channel làm ví dụ cụ thể.

## Tổng quan kiến trúc

```
User nhắn tin → Platform webhook → Channel.ServeHTTP()
  → BaseChannel.HandleMessage() → MessageBus.Inbound
  → Agent Loop → LLM → MessageBus.Outbound
  → Manager → Channel.Send() → Platform API → User nhận reply
```

Mỗi channel là một sub-package trong `pkg/channels/`, đăng ký vào registry qua `init()`.

## Bước 1: Tạo sub-package

```
pkg/channels/myplatform/
├── myplatform.go   # Struct chính, implement Channel interface
├── api.go          # HTTP client gọi API của platform
├── init.go         # Đăng ký factory vào registry
└── oauth.go        # (nếu cần) OAuth helpers
```

## Bước 2: Implement Channel interface

### Interface bắt buộc

```go
// pkg/channels/base.go
type Channel interface {
    Name() string                                          // Tên channel (vd: "zalo")
    Start(ctx context.Context) error                       // Khởi tạo, connect
    Stop(ctx context.Context) error                        // Graceful shutdown
    Send(ctx context.Context, msg bus.OutboundMessage) error // Gửi tin nhắn
    IsRunning() bool                                       // Channel đang chạy?
    IsAllowed(senderID string) bool                        // Check allow-list
    IsAllowedSender(sender bus.SenderInfo) bool            // Check allow-list (structured)
    ReasoningChannelID() string                            // Channel ID cho reasoning output
}
```

### Interface tuỳ chọn (opt-in)

```go
// Webhook-based channel (Zalo, LINE, Telegram webhook, ...)
type WebhookHandler interface {
    WebhookPath() string                    // Vd: "/webhook/zalo"
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Hiển thị "đang gõ..."
type TypingCapable interface {
    StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// Sửa tin nhắn đã gửi (dùng cho placeholder "Thinking..." → reply thật)
type MessageEditor interface {
    EditMessage(ctx context.Context, chatID, messageID, content string) error
}

// Gửi placeholder "Thinking... 💭"
type PlaceholderCapable interface {
    SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

// Gửi file/media
type MediaSender interface {
    SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error
}
```

### Ví dụ: Zalo channel struct

```go
type ZaloChannel struct {
    *channels.BaseChannel        // Embed pointer — cung cấp Name(), IsRunning(), HandleMessage(), ...
    config config.ZaloConfig
    api    *ZaloAPI
    mu     sync.Mutex
    ctx    context.Context
    cancel context.CancelFunc
}
```

**Quan trọng:**
- Embed `*channels.BaseChannel` (pointer, không phải value)
- Dùng `channels.NewBaseChannel()` để tạo
- `BaseChannel` đã implement: `Name()`, `IsRunning()`, `IsAllowed()`, `IsAllowedSender()`,
  `ReasoningChannelID()`, `HandleMessage()`, `MaxMessageLength()`

### Ví dụ: Constructor

```go
func NewZaloChannel(cfg config.ZaloConfig, messageBus *bus.MessageBus) (*ZaloChannel, error) {
    // Validate config
    if cfg.AppID == "" || cfg.AppSecret == "" {
        return nil, fmt.Errorf("zalo: app_id and app_secret are required")
    }

    // Tạo BaseChannel
    base := channels.NewBaseChannel(channelName, cfg, messageBus, cfg.AllowFrom,
        // Tuỳ chọn:
        // channels.WithMaxMessageLength(2000),
        // channels.WithGroupTrigger(cfg.GroupTrigger),
        // channels.WithReasoningChannelID(cfg.ReasoningChannelID),
    )

    return &ZaloChannel{
        BaseChannel: base,
        config:      cfg,
        api:         NewZaloAPI(cfg.AppID, cfg.AppSecret, cfg.AccessToken, cfg.RefreshToken),
    }, nil
}
```

### Ví dụ: Start / Stop lifecycle

```go
func (z *ZaloChannel) Start(ctx context.Context) error {
    z.ctx, z.cancel = context.WithCancel(ctx)
    z.SetRunning(true)                        // BẮT BUỘC sau khi start thành công
    logger.InfoC("zalo", "Zalo channel started")
    return nil
}

func (z *ZaloChannel) Stop(_ context.Context) error {
    z.SetRunning(false)                       // BẮT BUỘC đầu tiên khi stop
    if z.cancel != nil {
        z.cancel()
    }
    return nil
}
```

### Ví dụ: Send

```go
func (z *ZaloChannel) Send(_ context.Context, msg bus.OutboundMessage) error {
    if !z.IsRunning() {
        return channels.ErrNotRunning          // Manager sẽ KHÔNG retry
    }
    return z.api.SendTextMessage(msg.ChatID, msg.Content)
}
```

### Ví dụ: Webhook handler

```go
func (z *ZaloChannel) WebhookPath() string { return "/webhook/zalo" }

func (z *ZaloChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Đọc body
    body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
    defer r.Body.Close()

    // Trả 200 TRƯỚC — nhiều platform yêu cầu response nhanh
    w.WriteHeader(http.StatusOK)

    // Parse event
    var evt WebhookEvent
    json.Unmarshal(body, &evt)

    // Đẩy vào pipeline qua BaseChannel.HandleMessage()
    peer := bus.Peer{Kind: "direct", ID: evt.Sender.ID}
    z.HandleMessage(z.ctx, peer, evt.Message.MsgID, evt.Sender.ID, evt.Sender.ID,
        evt.Message.Text, nil, map[string]string{"platform": "zalo"})
}
```

## Bước 3: Đăng ký factory (init.go)

```go
package zalo

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
)

func init() {
    channels.RegisterFactory(channelName, func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewZaloChannel(cfg.Channels.Zalo, b)
    })
}
```

**Lưu ý:** Factory signature phải đúng:
```go
func(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error)
```

## Bước 4: Thêm config struct

Trong `pkg/config/config.go`:

```go
type ZaloConfig struct {
    Enabled     bool                `json:"enabled"     env:"PICOCLAW_CHANNELS_ZALO_ENABLED"`
    AppID       string              `json:"app_id"      env:"PICOCLAW_CHANNELS_ZALO_APP_ID"`
    AppSecret   string              `json:"app_secret"  env:"PICOCLAW_CHANNELS_ZALO_APP_SECRET"`
    AccessToken string              `json:"access_token" env:"PICOCLAW_CHANNELS_ZALO_ACCESS_TOKEN"`
    WebhookPath string              `json:"webhook_path" env:"PICOCLAW_CHANNELS_ZALO_WEBHOOK_PATH"`
    AllowFrom   FlexibleStringSlice `json:"allow_from"  env:"PICOCLAW_CHANNELS_ZALO_ALLOW_FROM"`
}
```

Và thêm vào `ChannelsConfig`:
```go
type ChannelsConfig struct {
    // ... existing channels ...
    Zalo ZaloConfig `json:"zalo"`
}
```

**Chú ý:** `AllowFrom` dùng `FlexibleStringSlice` để chấp nhận cả string và number trong JSON.

## Bước 5: Thêm vào manager

Trong `pkg/channels/manager.go`, function `initChannels()`:

```go
if channels.Zalo.Enabled && channels.Zalo.AppID != "" {
    m.initChannel("zalo", "Zalo")
}
```

**Cẩn thận:** Block `if` phải đóng đúng — không lồng vào block của channel khác.

## Bước 6: Import trong gateway

Trong `pkg/gateway/gateway.go`, thêm blank import:

```go
import (
    _ "github.com/sipeed/picoclaw/pkg/channels/zalo"
)
```

Blank import `_` trigger `init()` → đăng ký factory vào registry.

## Bước 7: Test với curl

### Test webhook nhận message

```bash
curl -X POST http://localhost:18790/webhook/zalo \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "user_send_text",
    "sender": {"id": "user123", "display_name": "Test User"},
    "recipient": {"id": "oa_id"},
    "message": {"msg_id": "msg001", "text": "hello"},
    "timestamp": 1711234567
  }'
```

### Verify pipeline hoạt động

Chạy gateway ở debug mode:
```bash
docker compose -f docker/docker-compose.yml --profile gateway run --rm \
  -e PICOCLAW_GATEWAY_HOST=0.0.0.0 picoclaw-gateway gateway -d
```

Khi gửi curl test, sẽ thấy:
```
Processing message from zalo:user123: hello
Routed message agent_id=main channel=zalo
LLM response content_chars=XX
Published outbound response channel=zalo chat_id=user123
```

Nếu access_token trống, sẽ thấy lỗi cuối:
```
Send failed error="zalo oa error -216: Access token is invalid"
```

Đây là bình thường — pipeline hoạt động, chỉ thiếu token.

### Test webhook verification (GET)

```bash
curl "http://localhost:18790/webhook/zalo?challenge=test123"
# Expected: test123
```

## Checklist thêm channel mới

- [ ] Tạo sub-package `pkg/channels/<name>/`
- [ ] Implement `Channel` interface (embed `*BaseChannel`)
- [ ] `Start()`: gọi `SetRunning(true)` sau khi thành công
- [ ] `Stop()`: gọi `SetRunning(false)` đầu tiên
- [ ] `Send()`: check `IsRunning()`, return `ErrNotRunning` nếu chưa start
- [ ] Webhook: trả HTTP 200 trước khi xử lý, dùng `HandleMessage()` để đẩy vào bus
- [ ] `init.go`: `RegisterFactory()` với đúng signature
- [ ] Config struct trong `pkg/config/config.go`
- [ ] Thêm vào `ChannelsConfig` struct
- [ ] Thêm `initChannel()` trong `manager.go`
- [ ] Blank import `_` trong `gateway.go`
- [ ] Test build: `CGO_ENABLED=0 go build -tags stdjson ./...`
- [ ] Test curl webhook
- [ ] Thêm vào `docker/data/config.json` mẫu

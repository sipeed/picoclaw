# 🔄 Tác Vụ Bất Đồng Bộ và Spawn

> Quay lại [README](../project/README.vi.md)

## Tác Vụ Nhanh (phản hồi trực tiếp)

- Báo cáo thời gian hiện tại

## Tác Vụ Dài (sử dụng spawn cho bất đồng bộ)

- Tìm kiếm web tin tức AI và tóm tắt
- Kiểm tra email và báo cáo tin nhắn quan trọng
```

**Hành vi chính:**

| Feature                 | Description                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Creates async subagent, doesn't block heartbeat           |
| **Independent context** | Subagent has its own context, no session history          |
| **message tool**        | Subagent communicates with user directly via message tool |
| **Non-blocking**        | After spawning, heartbeat continues to next task          |

#### Cách Giao Tiếp Subagent Hoạt Động

```
Heartbeat được kích hoạt
    ↓
Agent đọc HEARTBEAT.md
    ↓
Cho tác vụ dài: spawn subagent
    ↓                           ↓
Tiếp tục tác vụ tiếp theo    Subagent làm việc độc lập
    ↓                           ↓
Tất cả tác vụ hoàn thành     Subagent sử dụng công cụ "message"
    ↓                           ↓
Phản hồi HEARTBEAT_OK        Người dùng nhận kết quả trực tiếp
```

Subagent có quyền truy cập công cụ (message, web_search, v.v.) và có thể giao tiếp với người dùng độc lập mà không cần qua agent chính.

**Cấu hình:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `enabled`  | `true`  | Enable/disable heartbeat           |
| `interval` | `30`    | Check interval in minutes (min: 5) |

**Biến môi trường:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` để tắt
* `PICOCLAW_HEARTBEAT_INTERVAL=60` để thay đổi khoảng thời gian

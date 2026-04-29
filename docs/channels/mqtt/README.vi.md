# 📡 Kênh MQTT

PicoClaw hỗ trợ bất kỳ client MQTT nào làm kênh nhắn tin. Thiết bị hoặc dịch vụ publish yêu cầu lên broker; PicoClaw subscribe, xử lý và publish phản hồi trở lại.

## 🚀 Bắt đầu nhanh

**1. Thêm kênh vào `~/.picoclaw/config.json`:**

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "tcp://localhost:1883",
        "agent_id": "assistant"
      }
    }
  }
}
```

**2. Khởi động gateway:**

```bash
picoclaw gateway
```

**3. Gửi tin nhắn từ bất kỳ client MQTT nào:**

```bash
mosquitto_pub -t "/picoclaw/assistant/device1/request" \
  -m '{"text": "CPU đang dùng bao nhiêu phần trăm?"}'
```

**4. Subscribe để nhận phản hồi:**

```bash
mosquitto_sub -t "/picoclaw/assistant/device1/response"
```

---

## 📨 Cấu trúc topic

```
{prefix}/{agent_id}/{client_id}/request    # Client → PicoClaw
{prefix}/{agent_id}/{client_id}/response   # PicoClaw → Client
```

| Phân đoạn | Mô tả |
|-----------|-------|
| `prefix` | Tiền tố topic, cấu hình phía server. Mặc định: `/picoclaw` |
| `agent_id` | Định danh instance PicoClaw, đặt trong trường `agent_id` |
| `client_id` | Định danh phiên do client xác định — dùng ID ổn định cho mỗi thiết bị để duy trì ngữ cảnh hội thoại |

### Payload tin nhắn (JSON)

```json
{ "text": "nội dung tin nhắn" }
```

---

## ⚙️ Cấu hình

### config.json

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
        "client_id": "",
        "keep_alive": 60,
        "qos": 0
      }
    }
  }
}
```

### .security.yml (thông tin xác thực)

Tên người dùng và mật khẩu được lưu trong `~/.picoclaw/.security.yml`, không phải trong `config.json`:

```yaml
channel_list:
  mqtt:
    settings:
      username: ten_nguoi_dung
      password: mat_khau
```

### Các trường cấu hình

| Trường | Vị trí | Bắt buộc | Mặc định | Mô tả |
|--------|--------|----------|----------|-------|
| `broker` | `settings` | Có | — | URL của MQTT broker, ví dụ `tcp://host:1883`, `ssl://host:8883` |
| `agent_id` | `settings` | Có | — | Định danh agent, dùng làm một phần của đường dẫn topic |
| `topic_prefix` | `settings` | Không | `/picoclaw` | Tiền tố không gian tên topic |
| `username` | `.security.yml` | Không | — | Tên người dùng xác thực với broker |
| `password` | `.security.yml` | Không | — | Mật khẩu xác thực với broker |
| `client_id` | `settings` | Không | tự động tạo | Client ID paho gửi đến broker. Tự động tạo dạng `picoclaw-mqtt-{agent_id}-{8 hex}` nếu không đặt; cố định trong suốt vòng đời tiến trình, tái sử dụng khi kết nối lại |
| `keep_alive` | `settings` | Không | `60` | Khoảng thời gian keepalive MQTT (giây) |
| `qos` | `settings` | Không | `0` | Mức QoS cho publish và subscribe: `0`, `1` hoặc `2` |

### Biến môi trường

| Biến | Trường |
|------|--------|
| `PICOCLAW_CHANNELS_MQTT_BROKER` | `broker` |
| `PICOCLAW_CHANNELS_MQTT_AGENT_ID` | `agent_id` |
| `PICOCLAW_CHANNELS_MQTT_TOPIC_PREFIX` | `topic_prefix` |
| `PICOCLAW_CHANNELS_MQTT_USERNAME` | `username` |
| `PICOCLAW_CHANNELS_MQTT_PASSWORD` | `password` |
| `PICOCLAW_CHANNELS_MQTT_CLIENT_ID` | `client_id` |
| `PICOCLAW_CHANNELS_MQTT_KEEP_ALIVE` | `keep_alive` |
| `PICOCLAW_CHANNELS_MQTT_QOS` | `qos` |

---

## 🔄 Kết nối lại

PicoClaw tự động kết nối lại với broker nếu mất kết nối, với khoảng thời gian thử lại 5 giây. Sau khi kết nối lại, subscription được tái thiết lập tự động. Client ID phía broker giữ nguyên qua các lần kết nối lại, giúp broker nhận diện chính xác cùng một phiên.

---

## ⚠️ Lưu ý

- **TLS**: Hỗ trợ SSL/TLS (URL broker dùng `ssl://`). Mặc định bỏ qua xác minh chứng chỉ.
- **Phản hồi streaming**: Phản hồi streaming gửi nhiều tin nhắn đến topic response; ghép nối chúng theo thứ tự để có phản hồi đầy đủ.
- **client_id và ID phiên**: `client_id` trong đường dẫn topic được đặt bởi ứng dụng client của bạn và xác định phiên hội thoại. Nó khác với client ID paho mà PicoClaw dùng để kết nối broker.
- **Nhiều instance**: Nếu nhiều instance PicoClaw dùng cùng `agent_id` trên cùng broker, hãy đặt `client_id` riêng biệt cho từng instance để tránh xung đột ở tầng broker.

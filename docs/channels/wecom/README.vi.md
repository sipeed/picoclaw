> Quay lại [README](../../project/README.vi.md)

# WeCom

PicoClaw cung cấp WeCom dưới dạng một kênh duy nhất `channels.wecom`, được xây dựng trên API WebSocket chính thức của WeCom AI Bot.
Điều này thay thế việc phân tách cũ `wecom`, `wecom_app` và `wecom_aibot` bằng một mô hình cấu hình thống nhất.

> Không cần URL callback webhook công khai. PicoClaw thiết lập kết nối WebSocket đi ra tới WeCom.

## Tính năng được hỗ trợ

- Chat trực tiếp và chat nhóm
- Phản hồi streaming qua giao thức WeCom AI Bot
- Nhận tin nhắn văn bản, giọng nói, hình ảnh, tệp, video và tin nhắn hỗn hợp
- Gửi phản hồi văn bản và phương tiện (`image`, `file`, `voice`, `video`)
- Đăng ký qua mã QR bằng Web UI hoặc CLI
- Danh sách cho phép chung và định tuyến `reasoning_channel_id`

---

## Bắt đầu nhanh

### Tùy chọn 1: Liên kết QR qua Web UI (Khuyến nghị)

Mở Web UI, điều hướng đến **Channels → WeCom** và nhấp vào nút liên kết QR. Quét mã QR bằng WeCom và xác nhận trong ứng dụng — thông tin đăng nhập được lưu tự động.

<p align="center">
<img src="../../../assets/wecom-qr-binding.jpg" alt="Liên kết QR WeCom trong Web UI" width="600">
</p>

### Tùy chọn 2: Đăng nhập QR qua CLI

Chạy:

```bash
picoclaw auth wecom
```

Lệnh thực hiện:
1. Yêu cầu mã QR từ WeCom và hiển thị trong terminal
2. Đồng thời in ra một **Liên kết mã QR** mà bạn có thể mở trong trình duyệt nếu mã QR trên terminal khó quét
3. Chờ xác nhận — sau khi quét, bạn cũng phải **xác nhận đăng nhập trong ứng dụng WeCom**
4. Khi thành công, ghi `bot_id` và `secret` vào `channels.wecom` và lưu cấu hình

Thời gian chờ mặc định là **5 phút**. Sử dụng `--timeout` để kéo dài:

```bash
picoclaw auth wecom --timeout 10m
```

> ⚠️ Quét mã QR là chưa đủ — bạn cũng phải nhấn **Xác nhận** trong ứng dụng WeCom, nếu không lệnh sẽ hết thời gian chờ.

### Tùy chọn 3: Cấu hình thủ công

Nếu bạn đã có `bot_id` và `secret` từ nền tảng WeCom AI Bot, hãy cấu hình trực tiếp:

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

---

## Cấu hình

| Trường | Kiểu | Mặc định | Mô tả |
| ------ | ---- | -------- | ----- |
| `enabled` | bool | `false` | Kích hoạt kênh WeCom. |
| `bot_id` | string | — | Mã định danh WeCom AI Bot. Bắt buộc khi được kích hoạt. |
| `secret` | string | — | Secret của WeCom AI Bot. Được lưu mã hóa trong `.security.yml`. Bắt buộc khi được kích hoạt. |
| `websocket_url` | string | `wss://openws.work.weixin.qq.com` | Điểm cuối WebSocket của WeCom. |
| `send_thinking_message` | bool | `true` | Gửi tin nhắn `Processing...` trước khi phản hồi streaming bắt đầu. |
| `allow_from` | array | `[]` | Danh sách cho phép người gửi. Để trống nghĩa là cho phép tất cả. |
| `reasoning_channel_id` | string | `""` | ID chat tùy chọn để định tuyến đầu ra suy luận đến một cuộc hội thoại riêng. |

### Biến môi trường

Tất cả các trường có thể được ghi đè bằng biến môi trường với tiền tố `PICOCLAW_CHANNELS_WECOM_`:

| Biến môi trường | Trường tương ứng |
| ---------------- | ---------------- |
| `PICOCLAW_CHANNELS_WECOM_ENABLED` | `enabled` |
| `PICOCLAW_CHANNELS_WECOM_BOT_ID` | `bot_id` |
| `PICOCLAW_CHANNELS_WECOM_SECRET` | `secret` |
| `PICOCLAW_CHANNELS_WECOM_WEBSOCKET_URL` | `websocket_url` |
| `PICOCLAW_CHANNELS_WECOM_SEND_THINKING_MESSAGE` | `send_thinking_message` |
| `PICOCLAW_CHANNELS_WECOM_ALLOW_FROM` | `allow_from` |
| `PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID` | `reasoning_channel_id` |

---

## Hành vi khi chạy

- PicoClaw duy trì một lượt WeCom đang hoạt động để phản hồi streaming có thể tiếp tục trên cùng một luồng khi có thể.
- Phản hồi streaming có thời lượng tối đa **5,5 phút** và khoảng cách gửi tối thiểu **500ms**.
- Nếu streaming không còn khả dụng, phản hồi sẽ chuyển sang gửi push chủ động.
- Các liên kết tuyến chat hết hạn sau **30 phút** không hoạt động.
- Phương tiện nhận được sẽ được tải xuống bộ lưu trữ phương tiện cục bộ trước khi chuyển cho agent.
- Phương tiện gửi đi được tải lên WeCom dưới dạng tệp tạm thời, sau đó gửi dưới dạng tin nhắn phương tiện.
- Tin nhắn trùng lặp được phát hiện và loại bỏ (bộ đệm vòng của 1000 ID tin nhắn gần nhất).

---

## Di chuyển từ cấu hình WeCom cũ

| Cấu hình trước đây | Di chuyển |
| ------------------- | --------- |
| `channels.wecom` (bot webhook) | Thay thế bằng `channels.wecom` sử dụng `bot_id` + `secret`. |
| `channels.wecom_app` | Xóa. Sử dụng `channels.wecom` thay thế. |
| `channels.wecom_aibot` | Di chuyển `bot_id` và `secret` sang `channels.wecom`. |
| `token`, `encoding_aes_key`, `webhook_url`, `webhook_path` | Không còn sử dụng. Xóa khỏi cấu hình. |
| `corp_id`, `corp_secret`, `agent_id` | Không còn sử dụng. Xóa khỏi cấu hình. |
| `welcome_message`, `processing_message`, `max_steps` | Không còn là một phần của cấu hình kênh WeCom. |

---

## Khắc phục sự cố

### Liên kết QR hết thời gian chờ

- Sau khi quét mã QR, bạn cũng phải **xác nhận đăng nhập trong ứng dụng WeCom**. Chỉ quét là chưa đủ.
- Chạy lại với `--timeout` lớn hơn: `picoclaw auth wecom --timeout 10m`
- Nếu mã QR trên terminal khó quét, hãy sử dụng **Liên kết mã QR** được in bên dưới để mở trong trình duyệt.

### Mã QR đã hết hạn

- Mã QR có thời hạn hiệu lực giới hạn. Chạy lại `picoclaw auth wecom` để lấy mã mới.

### Kết nối WebSocket thất bại

- Kiểm tra xem `bot_id` và `secret` có chính xác không.
- Xác nhận máy chủ có thể kết nối đến `wss://openws.work.weixin.qq.com` (WebSocket đi ra, không cần cổng đến).

### Phản hồi không đến

- Kiểm tra xem `allow_from` có đang chặn người gửi không.
- Kiểm tra rằng `channels.wecom.bot_id` và `channels.wecom.secret` đã được thiết lập và không trống.

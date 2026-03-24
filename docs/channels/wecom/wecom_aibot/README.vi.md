> Quay lại [README](../../../../README.vi.md)

# WeCom AI Bot

WeCom AI Bot là phương thức tích hợp hội thoại AI chính thức do WeCom cung cấp. Hỗ trợ cả chat riêng tư và chat nhóm, tích hợp giao thức phản hồi streaming, và hỗ trợ chủ động đẩy phản hồi cuối cùng qua `response_url` sau khi hết thời gian chờ.

## So sánh với các kênh WeCom khác

| Tính năng | WeCom Bot | WeCom App | **WeCom AI Bot** |
|-----------|-----------|-----------|-----------------|
| Chat riêng tư | ✅ | ✅ | ✅ |
| Chat nhóm | ✅ | ❌ | ✅ |
| Đầu ra streaming | ❌ | ❌ | ✅ |
| Đẩy chủ động khi timeout | ❌ | ✅ | ✅ |
| Độ phức tạp cấu hình | Thấp | Cao | Trung bình |

## Cấu hình

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

| Trường | Kiểu | Bắt buộc | Mô tả |
| ---------------- | ------ | --------- | -------------------------------------------------- |
| token | string | Có | Token xác minh callback, cấu hình trên trang quản lý AI Bot |
| encoding_aes_key | string | Có | Khóa AES 43 ký tự, được tạo ngẫu nhiên trên trang quản lý AI Bot |
| webhook_path | string | Không | Đường dẫn webhook (mặc định: /webhook/wecom-aibot) |
| allow_from | array | Không | Danh sách cho phép ID người dùng; mảng rỗng cho phép tất cả người dùng |
| welcome_message | string | Không | Tin nhắn chào mừng gửi khi người dùng mở chat; để trống để tắt |
| reply_timeout | int | Không | Thời gian chờ phản hồi tính bằng giây (mặc định: 5) |
| max_steps | int | Không | Số bước thực thi tối đa của agent (mặc định: 10) |

## Hướng dẫn thiết lập

1. Đăng nhập vào [Bảng điều khiển quản trị WeCom](https://work.weixin.qq.com/wework_admin)
2. Vào "Quản lý ứng dụng" → "AI Bot", sau đó tạo hoặc chọn một AI Bot
3. Trên trang cấu hình AI Bot, điền thông tin "Nhận tin nhắn":
   - **URL**: `http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token**: Tạo ngẫu nhiên hoặc tùy chỉnh
   - **EncodingAESKey**: Nhấp "Tạo ngẫu nhiên" để lấy khóa 43 ký tự
4. Nhập Token và EncodingAESKey vào file cấu hình PicoClaw, khởi động dịch vụ rồi quay lại bảng điều khiển quản trị để lưu (WeCom sẽ gửi yêu cầu xác minh)

> [!TIP]
> Máy chủ cần có thể truy cập được từ các máy chủ WeCom. Nếu bạn đang ở mạng nội bộ hoặc phát triển cục bộ, hãy sử dụng [ngrok](https://ngrok.com) hoặc frp để tạo tunnel.

## Giao thức phản hồi streaming

WeCom AI Bot sử dụng giao thức "pull streaming", khác với phản hồi một lần của webhook thông thường:

```
Người dùng gửi tin nhắn
  │
  ▼
PicoClaw trả về ngay {finish: false} (Agent bắt đầu xử lý)
  │
  ▼
WeCom pull khoảng mỗi 1 giây với {msgtype: "stream", stream: {id: "..."}}
  │
  ├─ Agent chưa xong → trả về {finish: false} (tiếp tục chờ)
  │
  └─ Agent xong → trả về {finish: true, content: "nội dung phản hồi"}
```

**Xử lý timeout** (tác vụ vượt quá 30 giây):

Nếu thời gian xử lý của agent vượt quá khoảng 30 giây (cửa sổ polling tối đa của WeCom là 6 phút), PicoClaw sẽ:

1. Đóng stream ngay lập tức và hiển thị cho người dùng: "⏳ 正在处理中，请稍候，结果将稍后发送。"
2. Agent tiếp tục chạy ở nền
3. Sau khi agent hoàn thành, phản hồi cuối cùng được chủ động đẩy đến người dùng qua `response_url` có trong tin nhắn

> `response_url` do WeCom cấp, có hiệu lực 1 giờ, chỉ dùng được một lần, không cần mã hóa — chỉ cần POST trực tiếp nội dung tin nhắn markdown.

## Tin nhắn chào mừng

Khi `welcome_message` được cấu hình, PicoClaw sẽ tự động phản hồi bằng tin nhắn đó khi người dùng mở cửa sổ chat với AI Bot (sự kiện `enter_chat`). Để trống để bỏ qua im lặng.

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## Câu hỏi thường gặp

### Xác minh URL callback thất bại

- Xác nhận tường lửa máy chủ đã mở cổng tương ứng (mặc định 18790)
- Xác nhận `token` và `encoding_aes_key` được điền đúng
- Kiểm tra log PicoClaw xem có nhận được yêu cầu GET từ WeCom không

### Tin nhắn không nhận được phản hồi

- Kiểm tra xem `allow_from` có vô tình hạn chế người gửi không
- Tìm `context canceled` hoặc lỗi agent trong log
- Xác nhận cấu hình agent (ví dụ: `model_name`) là đúng

### Không nhận được push cuối cùng cho tác vụ dài

- Xác nhận callback tin nhắn có chứa `response_url` (chỉ hỗ trợ bởi WeCom AI Bot phiên bản mới)
- Xác nhận máy chủ có thể thực hiện yêu cầu ra ngoài (cần POST đến `response_url`)
- Kiểm tra log với từ khóa `response_url mode` và `Sending reply via response_url`

## Tài liệu tham khảo

- [Tài liệu tích hợp WeCom AI Bot](https://developer.work.weixin.qq.com/document/path/100719)
- [Mô tả giao thức phản hồi streaming](https://developer.work.weixin.qq.com/document/path/100719)
- [Phản hồi chủ động qua response_url](https://developer.work.weixin.qq.com/document/path/101138)

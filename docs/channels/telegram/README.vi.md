> Quay lại [README](../../project/README.vi.md)

# Telegram

Kênh Telegram sử dụng long polling qua Telegram Bot API để giao tiếp dựa trên bot. Hỗ trợ tin nhắn văn bản, tệp đính kèm đa phương tiện (ảnh, giọng nói, âm thanh, tài liệu), chuyển giọng nói thành văn bản qua Groq Whisper và xử lý lệnh tích hợp sẵn.

## Cấu hình

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "proxy": "",
        "use_markdown_v2": false,
        "business_mode": false,
        "business_owner": "123456789",
        "business_commands_enable": false,
        "guest_mode": false
      }
    }
  }
}
```

| Trường         | Kiểu   | Bắt buộc | Mô tả                                                                    |
| -------------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled        | bool   | Có       | Có bật kênh Telegram hay không                                           |
| allow_from     | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả        |
| settings.proxy | string | Không    | URL proxy để kết nối với Telegram API (ví dụ: http://127.0.0.1:7890)    |
| settings.use_markdown_v2 | bool | Không | Bật định dạng Telegram MarkdownV2                                        |
| settings.business_mode | bool | Không | Bật xử lý tin nhắn Telegram Business                                     |
| settings.business_owner | string | Không | ID người dùng Telegram của chủ tài khoản Business cần bỏ qua             |
| settings.business_commands_enable | bool | Không | Cho phép lệnh bot trong chat Telegram Business                          |
| settings.guest_mode | bool | Không | Bật xử lý và trả lời tin nhắn Telegram Guest                             |

## Hướng dẫn thiết lập

1. Tìm kiếm `@BotFather` trong Telegram
2. Gửi lệnh `/newbot` và làm theo hướng dẫn để tạo bot mới
3. Lấy Token API HTTP
4. Điền Token vào file cấu hình
5. (Tùy chọn) Cấu hình `allow_from` để giới hạn ID người dùng được phép tương tác (có thể lấy ID qua `@userinfobot`)

## Chế độ Telegram Business

Đặt `settings.business_mode: true` để nhận và trả lời tin nhắn Telegram Business từ các tài khoản doanh nghiệp đã kết nối. Phản hồi dùng `business_connection_id` nhận được, và PicoClaw đánh dấu tin nhắn Business là đã đọc khi bot có quyền `can_read_messages`.

Đặt `settings.business_owner` thành ID người dùng Telegram của chủ tài khoản Business. Tin nhắn Business từ người dùng này sẽ bị bỏ qua để tránh bot trả lời các tin nhắn bạn gửi thủ công từ tài khoản đã kết nối.

Theo mặc định, lệnh bot trong chat Business bị bỏ qua. Đặt `settings.business_commands_enable: true` để xử lý `/new`, `/help`, `/show`, `/list` và `/use`.

Ví dụ :

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "business_mode": true,
        "business_owner": "123456789",
        "business_commands_enable": true
      }
    }
  }
}
```

## Chế độ Telegram Guest

Đặt `settings.guest_mode: true` để nhận cập nhật `guest_message` và trả lời bằng phương thức Telegram `answerGuestQuery`. Tin nhắn Guest đến từ chat mà bot không phải là thành viên, nên PicoClaw tách phiên bằng `guest_query_id` nhận được.

Khi `settings.guest_mode` là `false`, cập nhật Guest không được yêu cầu và mọi tin nhắn Guest được giải mã sẽ bị bỏ qua. Các chỉ báo gõ và placeholder được bỏ qua cho phản hồi Guest vì Telegram yêu cầu một phản hồi `answerGuestQuery` duy nhất.

Ví dụ :

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "guest_mode": true
      }
    }
  }
}
```

## Định dạng nâng cao

Bạn có thể đặt `use_markdown_v2: true` để bật các tùy chọn định dạng nâng cao. Điều này cho phép bot sử dụng toàn bộ các tính năng của Telegram MarkdownV2, bao gồm các kiểu lồng nhau, spoiler và các khối chiều rộng cố định tùy chỉnh.

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["YOUR_USER_ID"],
      "settings": {
        "use_markdown_v2": true
      }
    }
  }
}
```

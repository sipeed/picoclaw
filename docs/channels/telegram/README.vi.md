> Quay lại [README](../../../README.vi.md)

# Telegram

Kênh Telegram sử dụng long polling qua Telegram Bot API để giao tiếp dựa trên bot. Hỗ trợ tin nhắn văn bản, tệp đính kèm đa phương tiện (ảnh, giọng nói, âm thanh, tài liệu), chuyển giọng nói thành văn bản qua Groq Whisper và xử lý lệnh tích hợp sẵn.

## Cấu hình

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": "",
      "use_markdown_v2": false
    }
  }
}
```

| Trường         | Kiểu   | Bắt buộc | Mô tả                                                                    |
| -------------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled        | bool   | Có       | Có bật kênh Telegram hay không                                           |
| token          | string | Có       | Token API Bot Telegram                                                   |
| allow_from     | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả        |
| proxy          | string | Không    | URL proxy để kết nối với Telegram API (ví dụ: http://127.0.0.1:7890)    |
| use_markdown_v2 | bool   | Không    | Bật định dạng Telegram MarkdownV2                                        |

## Hướng dẫn thiết lập

1. Tìm kiếm `@BotFather` trong Telegram
2. Gửi lệnh `/newbot` và làm theo hướng dẫn để tạo bot mới
3. Lấy Token API HTTP
4. Điền Token vào file cấu hình
5. (Tùy chọn) Cấu hình `allow_from` để giới hạn ID người dùng được phép tương tác (có thể lấy ID qua `@userinfobot`)

## Định dạng nâng cao

Bạn có thể đặt `use_markdown_v2: true` để bật các tùy chọn định dạng nâng cao. Điều này cho phép bot sử dụng toàn bộ các tính năng của Telegram MarkdownV2, bao gồm các kiểu lồng nhau, spoiler và các khối chiều rộng cố định tùy chỉnh.

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": true
    }
  }
}
```

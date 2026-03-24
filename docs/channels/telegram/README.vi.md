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
      "proxy": ""
    }
  }
}
```

| Trường     | Kiểu   | Bắt buộc | Mô tả                                                                    |
| ---------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled    | bool   | Có       | Có bật kênh Telegram hay không                                           |
| token      | string | Có       | Token API Bot Telegram                                                   |
| allow_from | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả        |
| proxy      | string | Không    | URL proxy để kết nối với Telegram API (ví dụ: http://127.0.0.1:7890)    |

## Hướng dẫn thiết lập

1. Tìm kiếm `@BotFather` trong Telegram
2. Gửi lệnh `/newbot` và làm theo hướng dẫn để tạo bot mới
3. Lấy Token API HTTP
4. Điền Token vào file cấu hình
5. (Tùy chọn) Cấu hình `allow_from` để giới hạn ID người dùng được phép tương tác (có thể lấy ID qua `@userinfobot`)

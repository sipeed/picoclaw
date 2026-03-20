> Quay lại [README](../../../README.vi.md)

# Slack

Slack là nền tảng nhắn tin tức thì hàng đầu dành cho doanh nghiệp. PicoClaw sử dụng Socket Mode của Slack để giao tiếp hai chiều theo thời gian thực, không cần cấu hình endpoint webhook công khai.

## Cấu hình

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| Trường     | Kiểu   | Bắt buộc | Mô tả                                                                        |
| ---------- | ------ | -------- | ---------------------------------------------------------------------------- |
| enabled    | bool   | Có       | Có bật kênh Slack hay không                                                  |
| bot_token  | string | Có       | Bot User OAuth Token của Slack bot (bắt đầu bằng xoxb-)                      |
| app_token  | string | Có       | App Level Token Socket Mode của ứng dụng Slack (bắt đầu bằng xapp-)          |
| allow_from | array  | Không    | Danh sách trắng ID người dùng; để trống cho phép tất cả                      |

## Quy trình thiết lập

1. Truy cập [Slack API](https://api.slack.com/) và tạo một ứng dụng Slack mới
2. Bật Socket Mode và lấy App Level Token
3. Thêm Bot Token Scopes (ví dụ: `chat:write`, `im:history`, v.v.)
4. Cài đặt ứng dụng vào workspace và lấy Bot User OAuth Token
5. Điền Bot Token và App Token vào file cấu hình

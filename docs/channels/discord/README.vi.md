> Quay lại [README](../../../README.vi.md)

# Discord

Discord là ứng dụng chat thoại, video và văn bản miễn phí được thiết kế cho cộng đồng. PicoClaw kết nối với máy chủ Discord qua Discord Bot API, hỗ trợ nhận và gửi tin nhắn.

## Cấu hình

```json
{
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "group_trigger": {
        "mention_only": false
      }
    }
  }
}
```

| Trường        | Kiểu   | Bắt buộc | Mô tả                                                                       |
| ------------- | ------ | -------- | --------------------------------------------------------------------------- |
| enabled       | bool   | Có       | Có bật kênh Discord hay không                                               |
| token         | string | Có       | Token Bot Discord                                                           |
| allow_from    | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả           |
| group_trigger | object | Không    | Cài đặt kích hoạt nhóm (ví dụ: { "mention_only": false })                  |

## Hướng dẫn thiết lập

1. Truy cập [Discord Developer Portal](https://discord.com/developers/applications) và tạo ứng dụng mới
2. Bật các Intents:
   - Message Content Intent
   - Server Members Intent
3. Lấy Bot Token
4. Điền Bot Token vào file cấu hình
5. Mời bot vào máy chủ và cấp các quyền cần thiết (ví dụ: gửi tin nhắn, đọc lịch sử tin nhắn)

> Quay lại [README](../../project/README.vi.md)

# Hướng dẫn Cấu hình Kênh Matrix

## 1. Cấu hình Mẫu

Thêm vào `config.json`:

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": "Thinking..."
      },
      "reasoning_channel_id": "",
      "message_format": "richtext"
    }
  }
}
```

## 2. Tham chiếu Trường

| Trường               | Kiểu     | Bắt buộc | Mô tả |
|----------------------|----------|----------|-------|
| enabled              | bool     | Có       | Bật hoặc tắt kênh Matrix |
| homeserver           | string   | Có       | URL homeserver Matrix (ví dụ `https://matrix.org`) |
| user_id              | string   | Có       | ID người dùng Matrix của bot (ví dụ `@bot:matrix.org`) |
| access_token         | string   | Có       | Token truy cập của bot |
| device_id            | string   | Không    | ID thiết bị Matrix tùy chọn |
| join_on_invite       | bool     | Không    | Tự động tham gia phòng được mời |
| allow_from           | []string | Không    | Danh sách trắng người dùng (ID Matrix) |
| group_trigger        | object   | Không    | Chiến lược kích hoạt nhóm (`mention_only` / `prefixes`) |
| placeholder          | object   | Không    | Cấu hình tin nhắn giữ chỗ |
| reasoning_channel_id | string   | Không    | Kênh đích cho đầu ra suy luận |
| message_format       | string   | Không    | Định dạng đầu ra: `"richtext"` (mặc định) render markdown thành HTML; `"plain"` chỉ gửi văn bản thuần |

## 3. Tính năng Hiện tại

- Gửi/nhận tin nhắn văn bản với render markdown (đậm, nghiêng, tiêu đề, khối code, v.v.)
- Định dạng tin nhắn có thể cấu hình (`richtext` / `plain`)
- Tải xuống hình ảnh/âm thanh/video/tệp đến (MediaStore trước, fallback đường dẫn cục bộ)
- Chuẩn hóa âm thanh đến vào luồng phiên âm hiện có (`[audio: ...]`)
- Tải lên và gửi hình ảnh/âm thanh/video/tệp đi
- Quy tắc kích hoạt nhóm (bao gồm chế độ chỉ đề cập)
- Trạng thái đang gõ (`m.typing`)
- Tin nhắn giữ chỗ + thay thế phản hồi cuối cùng
- Tự động tham gia phòng được mời (có thể tắt)

## 4. TODO

- Cải thiện metadata phương tiện phong phú (ví dụ kích thước và hình thu nhỏ hình ảnh/video)

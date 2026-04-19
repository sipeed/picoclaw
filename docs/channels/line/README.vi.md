> Quay lại [README](../../project/README.vi.md)

# Line

PicoClaw hỗ trợ LINE thông qua LINE Messaging API kết hợp với webhook callback.

## Cấu hình

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Trường               | Kiểu   | Bắt buộc | Mô tả                                                                  |
| -------------------- | ------ | -------- | ---------------------------------------------------------------------- |
| enabled              | bool   | Có       | Có bật kênh LINE hay không                                             |
| channel_secret       | string | Có       | Channel Secret của LINE Messaging API                                  |
| channel_access_token | string | Có       | Channel Access Token của LINE Messaging API                            |
| webhook_path         | string | Không    | Đường dẫn webhook (mặc định: /webhook/line)                            |
| allow_from           | array  | Không    | Danh sách trắng ID người dùng; để trống cho phép tất cả                |

## Quy trình thiết lập

1. Truy cập [LINE Developers Console](https://developers.line.biz/console/) và tạo một nhà cung cấp dịch vụ cùng một kênh Messaging API
2. Lấy Channel Secret và Channel Access Token
3. Cấu hình webhook:
   - LINE yêu cầu webhook phải sử dụng HTTPS, vì vậy bạn cần triển khai máy chủ hỗ trợ HTTPS hoặc dùng công cụ reverse proxy như ngrok để expose máy chủ cục bộ ra internet
   - PicoClaw sử dụng máy chủ HTTP Gateway dùng chung để nhận webhook callback cho tất cả các kênh, mặc định lắng nghe tại 127.0.0.1:18790
   - Đặt Webhook URL thành `https://your-domain.com/webhook/line`, sau đó reverse proxy tên miền bên ngoài về Gateway cục bộ (cổng mặc định 18790)
   - Bật webhook và xác minh URL
4. Điền Channel Secret và Channel Access Token vào file cấu hình

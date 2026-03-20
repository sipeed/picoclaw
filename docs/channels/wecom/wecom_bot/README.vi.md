> Quay lại [README](../../../../README.vi.md)

# WeCom Bot

WeCom Bot là phương thức tích hợp nhanh do WeCom cung cấp, có thể nhận tin nhắn qua URL Webhook.

## Cấu hình

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Trường | Kiểu | Bắt buộc | Mô tả |
| ---------------- | ------ | --------- | -------------------------------------------- |
| token | string | Có | Token xác minh chữ ký |
| encoding_aes_key | string | Có | Khóa AES 43 ký tự dùng để giải mã |
| webhook_url | string | Có | URL webhook của bot nhóm WeCom dùng để gửi phản hồi |
| webhook_path | string | Không | Đường dẫn endpoint webhook (mặc định: /webhook/wecom) |
| allow_from | array | Không | Danh sách cho phép ID người dùng (rỗng = cho phép tất cả) |
| reply_timeout | int | Không | Thời gian chờ phản hồi tính bằng giây (mặc định: 5) |

## Hướng dẫn thiết lập

1. Thêm bot vào một nhóm WeCom
2. Lấy URL Webhook
3. (Để nhận tin nhắn) Cấu hình địa chỉ API nhận tin nhắn (URL callback), Token và EncodingAESKey trên trang cấu hình bot
4. Nhập thông tin liên quan vào file cấu hình

   Lưu ý: PicoClaw hiện sử dụng máy chủ HTTP Gateway dùng chung để nhận callback webhook cho tất cả các kênh. Địa chỉ lắng nghe mặc định là 127.0.0.1:18790. Để nhận callback từ internet công cộng, hãy cấu hình reverse proxy từ tên miền bên ngoài của bạn đến Gateway (cổng mặc định 18790).

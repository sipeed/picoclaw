> Quay lại [README](../../../../README.vi.md)

# Ứng dụng nội bộ WeCom

Ứng dụng nội bộ WeCom là ứng dụng được doanh nghiệp tạo ra trong WeCom, chủ yếu dùng cho mục đích nội bộ. Thông qua ứng dụng nội bộ WeCom, doanh nghiệp có thể thực hiện giao tiếp và cộng tác hiệu quả với nhân viên, nâng cao hiệu suất làm việc.

## Cấu hình

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Trường | Kiểu | Bắt buộc | Mô tả |
| ---------------- | ------ | --------- | ---------------------------------------- |
| corp_id | string | Có | ID doanh nghiệp |
| corp_secret | string | Có | Secret của ứng dụng |
| agent_id | int | Có | ID agent của ứng dụng |
| token | string | Có | Token xác minh callback |
| encoding_aes_key | string | Có | Khóa AES 43 ký tự |
| webhook_path | string | Không | Đường dẫn webhook (mặc định: /webhook/wecom-app) |
| allow_from | array | Không | Danh sách cho phép ID người dùng |
| reply_timeout | int | Không | Thời gian chờ phản hồi tính bằng giây |

## Hướng dẫn thiết lập

1. Đăng nhập vào [Bảng điều khiển quản trị WeCom](https://work.weixin.qq.com/)
2. Vào "Quản lý ứng dụng" -> "Tạo ứng dụng"
3. Lấy ID doanh nghiệp (CorpID) và Secret của ứng dụng
4. Cấu hình "Nhận tin nhắn" trong cài đặt ứng dụng để lấy Token và EncodingAESKey
5. Đặt URL callback thành `http://<your-server-ip>:<port>/webhook/wecom-app`
6. Nhập CorpID, Secret, AgentID và các thông tin khác vào file cấu hình

   Lưu ý: PicoClaw hiện sử dụng máy chủ HTTP Gateway dùng chung để nhận callback webhook cho tất cả các kênh. Địa chỉ lắng nghe mặc định là 127.0.0.1:18790. Để nhận callback từ internet công cộng, hãy cấu hình reverse proxy từ tên miền bên ngoài của bạn đến Gateway (cổng mặc định 18790).

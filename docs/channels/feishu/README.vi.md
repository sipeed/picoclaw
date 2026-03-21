> Quay lại [README](../../../README.vi.md)

# Feishu

Feishu (tên quốc tế: Lark) là nền tảng cộng tác doanh nghiệp của ByteDance. Hỗ trợ cả thị trường Trung Quốc và toàn cầu thông qua kết nối WebSocket theo hướng sự kiện.

## Cấu hình

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| Trường                | Kiểu   | Bắt buộc | Mô tả                                                                    |
| --------------------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled               | bool   | Có       | Có bật kênh Feishu hay không                                             |
| app_id                | string | Có       | App ID của ứng dụng Feishu (bắt đầu bằng `cli_`)                        |
| app_secret            | string | Có       | App Secret của ứng dụng Feishu                                           |
| encrypt_key           | string | Không    | Khóa mã hóa cho callback sự kiện                                         |
| verification_token    | string | Không    | Token dùng để xác minh sự kiện Webhook                                   |
| allow_from            | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả        |
| random_reaction_emoji | array  | Không    | Danh sách emoji phản ứng ngẫu nhiên; để trống dùng "Pin" mặc định       |

## Hướng dẫn thiết lập

1. Truy cập [Nền tảng Mở Feishu](https://open.feishu.cn/) và tạo ứng dụng
2. Bật khả năng **Bot** trong cài đặt ứng dụng
3. Tạo phiên bản và xuất bản ứng dụng (cấu hình có hiệu lực sau khi xuất bản)
4. Lấy **App ID** (bắt đầu bằng `cli_`) và **App Secret**
5. Điền App ID và App Secret vào file cấu hình PicoClaw
6. Chạy `picoclaw gateway` để khởi động dịch vụ
7. Tìm kiếm tên bot trong Feishu và bắt đầu trò chuyện

> PicoClaw kết nối với Feishu bằng chế độ WebSocket/SDK — không cần cấu hình địa chỉ callback công khai hay Webhook URL.
>
> `encrypt_key` và `verification_token` là tùy chọn; nên bật mã hóa sự kiện trong môi trường sản xuất.
>
> Tham khảo emoji tùy chỉnh: [Danh sách Emoji Feishu](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)

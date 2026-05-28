> Quay lại [README](../../project/README.vi.md)

# QQ

PicoClaw hỗ trợ QQ thông qua API Bot chính thức của Nền tảng Mở QQ.

## Cấu hình

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| Trường     | Kiểu   | Bắt buộc | Mô tả                                                                    |
| ---------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled    | bool   | Có       | Có bật kênh QQ hay không                                                 |
| app_id     | string | Có       | App ID của ứng dụng bot QQ                                               |
| app_secret | string | Có       | App Secret của ứng dụng bot QQ                                           |
| allow_from | array  | Không    | Danh sách trắng ID người dùng; để trống nghĩa là cho phép tất cả        |

## Hướng dẫn thiết lập

### Thiết lập nhanh (Khuyến nghị)

Nền tảng Mở QQ cung cấp lối vào tạo bot một chạm:

1. Mở [QQ Bot Quick Create](https://q.qq.com/qqbot/openclaw/index.html) và đăng nhập bằng cách quét mã QR
2. Hệ thống tự động tạo bot — sao chép **App ID** và **App Secret**
3. Điền thông tin xác thực vào file cấu hình PicoClaw
4. Chạy `picoclaw gateway` để khởi động dịch vụ
5. Mở QQ và bắt đầu trò chuyện với bot

> App Secret chỉ hiển thị một lần — hãy lưu lại ngay. Xem lại sẽ buộc phải đặt lại.
>
> Bot được tạo qua lối vào nhanh chỉ dành cho người tạo sử dụng cá nhân và chưa hỗ trợ chat nhóm. Để hỗ trợ chat nhóm, hãy cấu hình chế độ sandbox trên [Nền tảng Mở QQ](https://q.qq.com/).

### Tạo thủ công

1. Đăng nhập vào [Nền tảng Mở QQ](https://q.qq.com/) bằng tài khoản QQ và đăng ký tài khoản nhà phát triển
2. Tạo bot QQ, tùy chỉnh ảnh đại diện và tên
3. Lấy **App ID** và **App Secret** trong cài đặt bot
4. Điền thông tin xác thực vào file cấu hình PicoClaw
5. Chạy `picoclaw gateway` để khởi động dịch vụ
6. Tìm kiếm bot của bạn trong QQ và bắt đầu trò chuyện

> Trong giai đoạn phát triển, nên bật chế độ sandbox và thêm người dùng, nhóm thử nghiệm vào sandbox để gỡ lỗi.

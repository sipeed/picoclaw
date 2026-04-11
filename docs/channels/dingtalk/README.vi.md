> Quay lại [README](../../../README.vi.md)

# DingTalk

DingTalk là nền tảng giao tiếp doanh nghiệp của Alibaba, được sử dụng rộng rãi trong môi trường làm việc tại Trung Quốc. Nền tảng này sử dụng SDK streaming để duy trì kết nối liên tục.

## Cấu hình

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| Trường        | Kiểu   | Bắt buộc | Mô tả                                                            |
| ------------- | ------ | -------- | ---------------------------------------------------------------- |
| enabled       | bool   | Có       | Có bật kênh DingTalk hay không                                   |
| client_id     | string | Có       | Client ID của ứng dụng DingTalk                                  |
| client_secret | string | Có       | Client Secret của ứng dụng DingTalk                              |
| allow_from    | array  | Không    | Danh sách trắng ID người dùng; để trống cho phép tất cả          |

## Quy trình thiết lập

1. Truy cập [Nền tảng mở DingTalk](https://open.dingtalk.com/)
2. Tạo một ứng dụng nội bộ doanh nghiệp
3. Lấy Client ID và Client Secret từ cài đặt ứng dụng
4. Cấu hình OAuth và đăng ký sự kiện (nếu cần)
5. Điền Client ID và Client Secret vào file cấu hình

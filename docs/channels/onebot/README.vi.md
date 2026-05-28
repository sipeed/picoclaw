> Quay lại [README](../../project/README.vi.md)

# OneBot

OneBot là tiêu chuẩn giao thức mở dành cho bot QQ, cung cấp giao diện thống nhất cho nhiều triển khai bot QQ khác nhau (ví dụ: go-cqhttp, Mirai). Nó sử dụng WebSocket để giao tiếp.

## Cấu hình

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Trường       | Kiểu   | Bắt buộc | Mô tả                                                                |
| ------------ | ------ | -------- | -------------------------------------------------------------------- |
| enabled      | bool   | Có       | Có bật kênh OneBot hay không                                         |
| ws_url       | string | Có       | URL WebSocket của máy chủ OneBot                                     |
| access_token | string | Không    | Token truy cập để kết nối với máy chủ OneBot                         |
| allow_from   | array  | Không    | Danh sách trắng ID người dùng; để trống cho phép tất cả              |

## Quy trình thiết lập

1. Triển khai một bản triển khai tương thích OneBot (ví dụ: napcat)
2. Cấu hình bản triển khai OneBot để bật dịch vụ WebSocket và đặt token truy cập (nếu cần)
3. Điền URL WebSocket và token truy cập vào file cấu hình

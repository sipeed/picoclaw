> Quay lại [README](../../../README.vi.md)

# MaixCam

MaixCam là kênh chuyên dụng để kết nối với các thiết bị camera AI Sipeed MaixCAM và MaixCAM2. Sử dụng TCP socket để giao tiếp hai chiều và hỗ trợ các kịch bản triển khai AI tại biên.

## Cấu hình

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| Trường     | Kiểu   | Bắt buộc | Mô tả                                                                    |
| ---------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled    | bool   | Có       | Có bật kênh MaixCam hay không                                            |
| host       | string | Có       | Địa chỉ lắng nghe của máy chủ TCP                                        |
| port       | int    | Có       | Cổng lắng nghe của máy chủ TCP                                           |
| allow_from | array  | Không    | Danh sách trắng ID thiết bị; để trống nghĩa là cho phép tất cả thiết bị  |

## Trường hợp sử dụng

Kênh MaixCam cho phép PicoClaw hoạt động như backend AI cho các thiết bị biên:

- **Giám sát thông minh**: MaixCAM gửi khung hình ảnh; PicoClaw phân tích bằng mô hình thị giác
- **Điều khiển IoT**: Thiết bị gửi dữ liệu cảm biến; PicoClaw điều phối phản hồi
- **AI ngoại tuyến**: Triển khai PicoClaw trên mạng nội bộ để suy luận độ trễ thấp

# Gỡ lỗi PicoClaw

> Quay lại [README](../project/README.vi.md)

PicoClaw thực hiện nhiều tương tác phức tạp ở hậu trường cho mỗi yêu cầu nhận được — từ định tuyến tin nhắn và đánh giá độ phức tạp, đến thực thi công cụ và thích ứng với lỗi mô hình. Khả năng xem chính xác những gì đang xảy ra là rất quan trọng, không chỉ để khắc phục các sự cố tiềm ẩn, mà còn để thực sự hiểu cách agent hoạt động.

## Khởi động PicoClaw ở chế độ gỡ lỗi

Để nhận thông tin chi tiết về những gì agent đang thực hiện (yêu cầu LLM, lệnh gọi công cụ, định tuyến tin nhắn), bạn có thể khởi động gateway PicoClaw với cờ gỡ lỗi:

```bash
picoclaw gateway --debug
# or
picoclaw gateway -d
```

Ở chế độ này, hệ thống sẽ định dạng log chi tiết và hiển thị bản xem trước của prompt hệ thống và kết quả thực thi công cụ.

## Tắt cắt ngắn log (log đầy đủ)

Theo mặc định, PicoClaw cắt ngắn các chuỗi rất dài (như *Prompt Hệ thống* hoặc kết quả JSON lớn) trong log gỡ lỗi để giữ cho console dễ đọc.

Nếu bạn cần kiểm tra đầu ra đầy đủ của một lệnh hoặc payload chính xác được gửi đến mô hình LLM, bạn có thể sử dụng cờ `--no-truncate`.

**Lưu ý:** Cờ này *chỉ* hoạt động khi kết hợp với chế độ `--debug`.

```bash
picoclaw gateway --debug --no-truncate

```

Khi cờ này được kích hoạt, chức năng cắt ngắn toàn cục sẽ bị vô hiệu hóa. Điều này cực kỳ hữu ích để:

* Xác minh cú pháp chính xác của các tin nhắn được gửi đến nhà cung cấp.
* Đọc đầu ra đầy đủ của các công cụ như `exec`, `web_fetch` hoặc `read_file`.
* Gỡ lỗi lịch sử phiên được lưu trong bộ nhớ.

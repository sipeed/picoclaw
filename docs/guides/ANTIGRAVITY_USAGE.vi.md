> Quay lại [README](../project/README.vi.md)

# Sử dụng nhà cung cấp Antigravity trong PicoClaw

Hướng dẫn này giải thích cách thiết lập và sử dụng nhà cung cấp **Antigravity** (Google Cloud Code Assist) trong PicoClaw.

## Điều kiện tiên quyết

1.  Một tài khoản Google.
2.  Đã kích hoạt Google Cloud Code Assist (thường có sẵn thông qua quy trình giới thiệu "Gemini for Google Cloud").

## 1. Xác thực

Để xác thực với Antigravity, chạy lệnh sau:

```bash
picoclaw auth login --provider antigravity
```

### Xác thực thủ công (Headless/VPS)
Nếu bạn đang chạy trên máy chủ (Coolify/Docker) và không thể truy cập `localhost`, hãy làm theo các bước sau:
1.  Chạy lệnh ở trên.
2.  Sao chép URL được cung cấp và mở nó trong trình duyệt cục bộ của bạn.
3.  Hoàn tất đăng nhập.
4.  Trình duyệt của bạn sẽ chuyển hướng đến URL `localhost:51121` (trang sẽ không tải được).
5.  **Sao chép URL cuối cùng đó** từ thanh địa chỉ trình duyệt.
6.  **Dán nó vào terminal** nơi PicoClaw đang chờ.

PicoClaw sẽ tự động trích xuất mã ủy quyền và hoàn tất quy trình.

## 2. Quản lý mô hình

### Liệt kê các mô hình khả dụng
Để xem dự án của bạn có quyền truy cập vào những mô hình nào và kiểm tra hạn mức của chúng:

```bash
picoclaw auth models
```

### Chuyển đổi mô hình
Bạn có thể thay đổi mô hình mặc định trong `~/.picoclaw/config.json` hoặc ghi đè qua CLI:

```bash
# Ghi đè cho một lệnh duy nhất
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

## 3. Sử dụng thực tế (Coolify/Docker)

Nếu bạn đang triển khai qua Coolify hoặc Docker, hãy làm theo các bước sau để kiểm tra:

1.  **Biến môi trường**:
    *   `PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash`
2.  **Lưu trữ xác thực**:
    Nếu bạn đã đăng nhập cục bộ, bạn có thể sao chép thông tin xác thực lên máy chủ:
    ```bash
    scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
    ```
    *Hoặc*, chạy lệnh `auth login` một lần trên máy chủ nếu bạn có quyền truy cập terminal.

## 4. Khắc phục sự cố

*   **Phản hồi trống**: Nếu một mô hình trả về phản hồi trống, nó có thể bị hạn chế cho dự án của bạn. Hãy thử `gemini-3-flash` hoặc `claude-opus-4-6-thinking`.
*   **429 Giới hạn tốc độ**: Antigravity có hạn mức nghiêm ngặt. PicoClaw sẽ hiển thị "thời gian đặt lại" trong thông báo lỗi nếu bạn đạt đến giới hạn.
*   **404 Không tìm thấy**: Đảm bảo bạn đang sử dụng ID mô hình từ danh sách `picoclaw auth models`. Sử dụng ID ngắn (ví dụ: `gemini-3-flash`) thay vì đường dẫn đầy đủ.

## 5. Tóm tắt các mô hình hoạt động tốt

Dựa trên kiểm tra, các mô hình sau đáng tin cậy nhất:
*   `gemini-3-flash` (Nhanh, khả dụng cao)
*   `gemini-2.5-flash-lite` (Nhẹ)
*   `claude-opus-4-6-thinking` (Mạnh mẽ, bao gồm khả năng suy luận)

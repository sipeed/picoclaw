> Quay lại [README](../project/README.vi.md)

# Mã hóa Thông tin Xác thực

PicoClaw hỗ trợ mã hóa các giá trị `api_key` trong các mục cấu hình `model_list`.
Các khóa đã mã hóa được lưu trữ dưới dạng chuỗi `enc://<base64>` và được giải mã tự động khi khởi động.

---

## Bắt đầu Nhanh

**1. Đặt cụm mật khẩu**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. Mã hóa khóa API**

Chạy `picoclaw onboard` — nó yêu cầu nhập cụm mật khẩu và tạo khóa SSH,
sau đó tự động mã hóa lại tất cả các mục `api_key` dạng văn bản thuần trong cấu hình
ở lần gọi `SaveConfig` tiếp theo. Giá trị `enc://` kết quả sẽ có dạng:

```
enc://AAAA...base64...
```

**3. Dán kết quả vào cấu hình**

```json
{
  "model_list": [
    {
      "model_name": "gpt-4o",
      "model": "openai/gpt-4o",
      "api_keys": ["enc://AAAA...base64..."],
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

---

## Các Định dạng `api_key` được Hỗ trợ

| Định dạng | Ví dụ | Hành vi |
|-----------|-------|---------|
| Văn bản thuần | `sk-abc123` | Sử dụng nguyên trạng |
| Tham chiếu tệp | `file://openai.key` | Nội dung được đọc từ cùng thư mục với tệp cấu hình |
| Đã mã hóa | `enc://<base64>` | Giải mã khi khởi động bằng `PICOCLAW_KEY_PASSPHRASE` |
| Trống | `""` | Truyền qua không thay đổi (dùng với `auth_method: oauth`) |

---

## Thiết kế Mật mã

### Dẫn xuất Khóa

Mã hóa sử dụng **HKDF-SHA256** với khóa riêng SSH làm yếu tố thứ hai.

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### Mã hóa

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### Định dạng Truyền tải

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| Trường | Kích thước | Mô tả |
|--------|-----------|-------|
| `salt` | 16 byte | Ngẫu nhiên mỗi lần mã hóa; đưa vào HKDF |
| `nonce` | 12 byte | Ngẫu nhiên mỗi lần mã hóa; IV của AES-GCM |
| `ciphertext` | thay đổi | Bản mã AES-256-GCM + thẻ xác thực 16 byte |

Thẻ xác thực GCM được tự động nối vào bản mã. Bất kỳ sự giả mạo nào đều khiến giải mã thất bại với lỗi thay vì trả về văn bản thuần bị hỏng.

### Hiệu suất

| Thao tác | Thời gian (ARM Cortex-A) |
|----------|--------------------------|
| Dẫn xuất khóa (HKDF) | < 1 ms |
| Giải mã AES-256-GCM | < 1 ms |
| **Tổng chi phí khởi động** | **< 2 ms mỗi khóa** |

---

## Bảo mật Hai Yếu tố với Khóa SSH

Khi khóa riêng SSH được cung cấp, việc phá vỡ mã hóa yêu cầu **cả hai**:

1. **Cụm mật khẩu** (`PICOCLAW_KEY_PASSPHRASE`)
2. **Tệp khóa riêng SSH**

Điều này có nghĩa là chỉ rò rỉ tệp cấu hình không đủ để khôi phục khóa API, ngay cả khi cụm mật khẩu yếu. Khóa SSH đóng góp 256 bit entropy (Ed25519) bất kể độ mạnh của cụm mật khẩu.

### Mô hình Mối đe dọa

| Kẻ tấn công có | Có thể giải mã? |
|----------------|-----------------|
| Chỉ tệp cấu hình | Không — cần cụm mật khẩu + khóa SSH |
| Chỉ khóa SSH | Không — cần cụm mật khẩu |
| Chỉ cụm mật khẩu | Không — cần khóa SSH |
| Tệp cấu hình + khóa SSH + cụm mật khẩu | Có — xâm phạm hoàn toàn |

---

## Biến Môi trường

| Biến | Bắt buộc | Mô tả |
|------|----------|-------|
| `PICOCLAW_KEY_PASSPHRASE` | Có (cho `enc://`) | Cụm mật khẩu dùng để dẫn xuất khóa |
| `PICOCLAW_SSH_KEY_PATH` | Không | Đường dẫn đến khóa riêng SSH. Nếu không đặt, tự động phát hiện từ `~/.ssh/picoclaw_ed25519.key` |

### Tự động Phát hiện Khóa SSH

Nếu `PICOCLAW_SSH_KEY_PATH` không được đặt, PicoClaw tìm khóa chuyên dụng:

```
~/.ssh/picoclaw_ed25519.key
```

Tệp chuyên dụng này tránh xung đột với các khóa SSH hiện có của người dùng.
Chạy `picoclaw onboard` để tạo tự động.

`os.UserHomeDir()` được sử dụng để phân giải thư mục home đa nền tảng (đọc `USERPROFILE` trên Windows, `HOME` trên Unix/macOS).

> **Lưu ý:** Tệp khóa SSH là bắt buộc cho mã hóa thông tin xác thực. Nếu không tìm thấy khóa và `PICOCLAW_SSH_KEY_PATH` không được đặt, mã hóa/giải mã sẽ thất bại. Chạy `picoclaw onboard` để tạo khóa tự động.

---

## Di chuyển

Vì tài liệu bí mật duy nhất là `PICOCLAW_KEY_PASSPHRASE` và tệp khóa riêng SSH, việc di chuyển rất đơn giản:

1. Sao chép tệp cấu hình sang máy mới.
2. Đặt `PICOCLAW_KEY_PASSPHRASE` với cùng giá trị.
3. Sao chép tệp khóa riêng SSH đến cùng đường dẫn (hoặc đặt `PICOCLAW_SSH_KEY_PATH` đến vị trí mới).

Không cần mã hóa lại.

---

## Lưu ý về Bảo mật

- **Cả cụm mật khẩu và khóa SSH đều bắt buộc.** Khóa SSH đóng vai trò yếu tố thứ hai — không có nó, mã hóa/giải mã sẽ thất bại. Chạy `picoclaw onboard` để tạo khóa nếu chưa tồn tại.
- **Khóa SSH chỉ đọc khi chạy.** PicoClaw không bao giờ ghi hoặc sửa đổi tệp khóa SSH.
- **Khóa văn bản thuần vẫn được hỗ trợ.** Các cấu hình hiện có không dùng `enc://` không bị ảnh hưởng.
- **Định dạng `enc://` được quản lý phiên bản** thông qua trường `info` của HKDF (`picoclaw-credential-v1`), cho phép nâng cấp thuật toán trong tương lai mà không làm hỏng các giá trị đã mã hóa hiện có.

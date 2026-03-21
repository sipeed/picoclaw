# ⚙️ Hướng Dẫn Cấu Hình

> Quay lại [README](../../README.vi.md)

## ⚙️ Cấu Hình

File cấu hình: `~/.piconomous/config.json`

### Biến Môi Trường

Bạn có thể ghi đè các đường dẫn mặc định bằng biến môi trường. Điều này hữu ích cho cài đặt portable, triển khai container, hoặc chạy piconomous như dịch vụ hệ thống. Các biến này độc lập và kiểm soát các đường dẫn khác nhau.

| Biến              | Mô tả                                                                                                                             | Đường Dẫn Mặc Định       |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICONOMOUS_CONFIG` | Ghi đè đường dẫn đến file cấu hình. Chỉ định trực tiếp cho piconomous file `config.json` nào cần tải, bỏ qua tất cả vị trí khác. | `~/.piconomous/config.json` |
| `PICONOMOUS_HOME`   | Ghi đè thư mục gốc cho dữ liệu piconomous. Thay đổi vị trí mặc định của `workspace` và các thư mục dữ liệu khác.          | `~/.piconomous`             |

**Ví dụ:**

```bash
# Chạy piconomous với file cấu hình cụ thể
# Đường dẫn workspace sẽ được đọc từ trong file cấu hình đó
PICONOMOUS_CONFIG=/etc/piconomous/production.json piconomous gateway

# Chạy piconomous với tất cả dữ liệu lưu tại /opt/piconomous
# Cấu hình sẽ được tải từ mặc định ~/.piconomous/config.json
# Workspace sẽ được tạo tại /opt/piconomous/workspace
PICONOMOUS_HOME=/opt/piconomous piconomous agent

# Sử dụng cả hai cho thiết lập tùy chỉnh hoàn toàn
PICONOMOUS_HOME=/srv/piconomous PICONOMOUS_CONFIG=/srv/piconomous/main.json piconomous gateway
```

### Bố Cục Workspace

Piconomous lưu trữ dữ liệu trong workspace đã cấu hình (mặc định: `~/.piconomous/workspace`):

```
~/.piconomous/workspace/
├── sessions/          # Phiên hội thoại và lịch sử
├── memory/           # Bộ nhớ dài hạn (MEMORY.md)
├── state/            # Trạng thái bền vững (kênh cuối, v.v.)
├── cron/             # Cơ sở dữ liệu tác vụ lên lịch
├── skills/           # Skill tùy chỉnh
├── AGENT.md          # Hướng dẫn hành vi agent
├── HEARTBEAT.md      # Prompt tác vụ định kỳ (kiểm tra mỗi 30 phút)
├── IDENTITY.md       # Danh tính agent
├── SOUL.md           # Linh hồn agent
└── USER.md           # Tùy chọn người dùng
```

> **Lưu ý:** Các thay đổi đối với `AGENT.md`, `SOUL.md`, `USER.md` và `memory/MEMORY.md` được tự động phát hiện trong thời gian chạy thông qua theo dõi thời gian sửa đổi file (mtime). **Không cần khởi động lại gateway** sau khi chỉnh sửa các file này — agent sẽ tải nội dung mới vào yêu cầu tiếp theo.

### Nguồn Skill

Mặc định, skill được tải từ:

1. `~/.piconomous/workspace/skills` (workspace)
2. `~/.piconomous/skills` (global)
3. `<đường-dẫn-nhúng-khi-build>/skills` (tích hợp)

Cho thiết lập nâng cao/test, bạn có thể ghi đè thư mục gốc skill builtin với:

```bash
export PICONOMOUS_BUILTIN_SKILLS=/path/to/skills
```

### Chính Sách Thực Thi Lệnh Thống Nhất

- Lệnh slash chung được thực thi qua một đường dẫn duy nhất trong `pkg/agent/loop.go` qua `commands.Executor`.
- Adapter kênh không còn xử lý lệnh chung cục bộ; chúng chuyển tiếp văn bản đầu vào đến đường dẫn bus/agent. Telegram vẫn tự động đăng ký lệnh được hỗ trợ khi khởi động.
- Lệnh slash không xác định (ví dụ `/foo`) được chuyển sang xử lý LLM bình thường.
- Lệnh đã đăng ký nhưng không được hỗ trợ trên kênh hiện tại (ví dụ `/show` trên WhatsApp) trả về lỗi rõ ràng cho người dùng và dừng xử lý tiếp.

### 🔒 Sandbox Bảo Mật

Piconomous chạy trong môi trường sandbox mặc định. Agent chỉ có thể truy cập file và thực thi lệnh trong workspace đã cấu hình.

#### Cấu Hình Mặc Định

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.piconomous/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Tùy chọn                | Mặc định                | Mô tả                                    |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.piconomous/workspace` | Thư mục làm việc của agent               |
| `restrict_to_workspace` | `true`                  | Giới hạn truy cập file/lệnh trong workspace |

#### Công Cụ Được Bảo Vệ

Khi `restrict_to_workspace: true`, các công cụ sau được sandbox:

| Công cụ       | Chức năng        | Giới hạn                               |
| ------------- | ---------------- | -------------------------------------- |
| `read_file`   | Đọc file         | Chỉ file trong workspace              |
| `write_file`  | Ghi file         | Chỉ file trong workspace              |
| `list_dir`    | Liệt kê thư mục | Chỉ thư mục trong workspace           |
| `edit_file`   | Sửa file         | Chỉ file trong workspace              |
| `append_file` | Nối vào file     | Chỉ file trong workspace              |
| `exec`        | Thực thi lệnh   | Đường dẫn lệnh phải trong workspace   |

#### Bảo Vệ Exec Bổ Sung

Ngay cả khi `restrict_to_workspace: false`, công cụ `exec` chặn các lệnh nguy hiểm sau:

* `rm -rf`, `del /f`, `rmdir /s` — Xóa hàng loạt
* `format`, `mkfs`, `diskpart` — Định dạng đĩa
* `dd if=` — Tạo ảnh đĩa
* Ghi vào `/dev/sd[a-z]` — Ghi trực tiếp đĩa
* `shutdown`, `reboot`, `poweroff` — Tắt hệ thống
* Fork bomb `:(){ :|:& };:`

### Kiểm Soát Truy Cập File

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.allow_read_paths` | string[] | `[]` | Additional paths allowed for reading outside workspace |
| `tools.allow_write_paths` | string[] | `[]` | Additional paths allowed for writing outside workspace |

### Bảo Mật Exec

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Allow exec tool from remote channels (Telegram/Discord etc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Enable dangerous command interception |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Custom regex patterns to block |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Custom regex patterns to allow |

> **Lưu ý Bảo Mật:** Bảo vệ symlink được bật mặc định — tất cả đường dẫn file được giải quyết qua `filepath.EvalSymlinks` trước khi so khớp whitelist, ngăn chặn tấn công thoát qua symlink.

#### Hạn Chế Đã Biết: Tiến Trình Con Từ Công Cụ Build

Guard bảo mật exec chỉ kiểm tra dòng lệnh mà Piconomous khởi chạy trực tiếp. Nó không kiểm tra đệ quy các tiến trình con được tạo bởi công cụ phát triển được phép như `make`, `go run`, `cargo`, `npm run`, hoặc script build tùy chỉnh.

Điều này có nghĩa là lệnh cấp cao nhất vẫn có thể biên dịch hoặc khởi chạy binary khác sau khi vượt qua kiểm tra guard ban đầu. Trong thực tế, hãy coi script build, Makefile, script package, và binary được tạo như mã thực thi cần cùng mức độ review như lệnh shell trực tiếp.

Cho môi trường rủi ro cao hơn:

* Review script build trước khi thực thi.
* Ưu tiên phê duyệt/review thủ công cho quy trình biên dịch và chạy.
* Chạy Piconomous trong container hoặc VM nếu bạn cần cách ly mạnh hơn guard tích hợp.

#### Ví Dụ Lỗi

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Tắt Giới Hạn (Rủi Ro Bảo Mật)

Nếu bạn cần agent truy cập đường dẫn ngoài workspace:

**Phương pháp 1: File cấu hình**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Phương pháp 2: Biến môi trường**

```bash
export PICONOMOUS_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Cảnh báo**: Tắt giới hạn này cho phép agent truy cập bất kỳ đường dẫn nào trên hệ thống. Chỉ sử dụng cẩn thận trong môi trường được kiểm soát.

#### Tính Nhất Quán Ranh Giới Bảo Mật

Cài đặt `restrict_to_workspace` áp dụng nhất quán trên tất cả đường dẫn thực thi:

| Đường Dẫn Thực Thi | Ranh Giới Bảo Mật          |
| -------------------- | ---------------------------- |
| Main Agent           | `restrict_to_workspace` ✅   |
| Subagent / Spawn     | Kế thừa cùng giới hạn ✅    |
| Heartbeat tasks      | Kế thừa cùng giới hạn ✅    |

Tất cả đường dẫn chia sẻ cùng giới hạn workspace — không có cách nào vượt qua ranh giới bảo mật qua subagent hoặc tác vụ lên lịch.

### Heartbeat (Tác Vụ Định Kỳ)

Piconomous có thể thực hiện tác vụ định kỳ tự động. Tạo file `HEARTBEAT.md` trong workspace:

```markdown
# Tác Vụ Định Kỳ

- Kiểm tra email cho tin nhắn quan trọng
- Xem lịch cho sự kiện sắp tới
- Kiểm tra dự báo thời tiết
```

Agent sẽ đọc file này mỗi 30 phút (có thể cấu hình) và thực thi các tác vụ sử dụng công cụ có sẵn.

#### Tác Vụ Bất Đồng Bộ Với Spawn

Cho tác vụ chạy lâu (tìm kiếm web, gọi API), sử dụng công cụ `spawn` để tạo **subagent**:

```markdown
# Tác Vụ Định Kỳ
```

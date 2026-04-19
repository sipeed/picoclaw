# ⚙️ Hướng Dẫn Cấu Hình

> Quay lại [README](../project/README.vi.md)

## ⚙️ Cấu Hình

File cấu hình: `~/.picoclaw/config.json`

### Biến Môi Trường

Bạn có thể ghi đè các đường dẫn mặc định bằng biến môi trường. Điều này hữu ích cho cài đặt portable, triển khai container, hoặc chạy picoclaw như dịch vụ hệ thống. Các biến này độc lập và kiểm soát các đường dẫn khác nhau.

| Biến              | Mô tả                                                                                                                             | Đường Dẫn Mặc Định       |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Ghi đè đường dẫn đến file cấu hình. Chỉ định trực tiếp cho picoclaw file `config.json` nào cần tải, bỏ qua tất cả vị trí khác. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Ghi đè thư mục gốc cho dữ liệu picoclaw. Thay đổi vị trí mặc định của `workspace` và các thư mục dữ liệu khác.          | `~/.picoclaw`             |

**Ví dụ:**

```bash
# Chạy picoclaw với file cấu hình cụ thể
# Đường dẫn workspace sẽ được đọc từ trong file cấu hình đó
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Chạy picoclaw với tất cả dữ liệu lưu tại /opt/picoclaw
# Cấu hình sẽ được tải từ mặc định ~/.picoclaw/config.json
# Workspace sẽ được tạo tại /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Sử dụng cả hai cho thiết lập tùy chỉnh hoàn toàn
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Mức Log của Gateway

`gateway.log_level` kiểm soát mức độ chi tiết của log Gateway, có thể cấu hình trong `config.json`:

```json
{
  "gateway": {
    "log_level": "warn"
  }
}
```

Giá trị mặc định là `warn`. Các giá trị được hỗ trợ: `debug`, `info`, `warn`, `error`, `fatal`.

Cũng có thể ghi đè bằng biến môi trường: `PICOCLAW_LOG_LEVEL=info`

### Bố Cục Workspace

PicoClaw lưu trữ dữ liệu trong workspace đã cấu hình (mặc định: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
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

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<đường-dẫn-nhúng-khi-build>/skills` (tích hợp)

Cho thiết lập nâng cao/test, bạn có thể ghi đè thư mục gốc skill builtin với:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Dung Skill va Lenh Tu Kenh Chat

Sau khi cai dat skill, ban co the xem va ep dung truc tiep tu kenh chat:

- `/list skills` hien ten cac skill da cai dat ma agent hien tai co the dung.
- `/use <skill> <message>` ep dung mot skill cho duy nhat mot yeu cau.
- `/use <skill>` dat san skill do cho tin nhan tiep theo trong cung cuoc tro chuyen.
- `/use clear` huy skill override dang cho duoc tao boi `/use <skill>`.
- `/btw <question>` dat cau hoi phu ngay lap tuc ma khong thay doi lich su phien hien tai. `/btw` duoc xu ly nhu mot truy van truc tiep khong dung cong cu va khong di vao luong thuc thi cong cu thong thuong.

Vi du:

```text
/list skills
/use git giai thich cach squash 3 commit cuoi
/btw nhac lai giup toi chung ta da chot gi cho ke hoach deploy
/use italiapersonalfinance
dammi le ultime news
```

### Chính Sách Thực Thi Lệnh Thống Nhất

- Lệnh slash chung được thực thi qua một đường dẫn duy nhất trong `pkg/agent/loop.go` qua `commands.Executor`.
- Adapter kênh không còn xử lý lệnh chung cục bộ; chúng chuyển tiếp văn bản đầu vào đến đường dẫn bus/agent. Telegram vẫn tự động đăng ký khi khởi động các lệnh được hỗ trợ như `/start`, `/help`, `/show`, `/list`, `/use`, va `/btw`.
- Lệnh slash không xác định (ví dụ `/foo`) được chuyển sang xử lý LLM bình thường.
- Lệnh đã đăng ký nhưng không được hỗ trợ trên kênh hiện tại (ví dụ `/show` trên WhatsApp) trả về lỗi rõ ràng cho người dùng và dừng xử lý tiếp.

### 🔒 Sandbox Bảo Mật

PicoClaw chạy trong môi trường sandbox mặc định. Agent chỉ có thể truy cập file và thực thi lệnh trong workspace đã cấu hình.

#### Cấu Hình Mặc Định

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Tùy chọn                | Mặc định                | Mô tả                                    |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Thư mục làm việc của agent               |
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

Guard bảo mật exec chỉ kiểm tra dòng lệnh mà PicoClaw khởi chạy trực tiếp. Nó không kiểm tra đệ quy các tiến trình con được tạo bởi công cụ phát triển được phép như `make`, `go run`, `cargo`, `npm run`, hoặc script build tùy chỉnh.

Điều này có nghĩa là lệnh cấp cao nhất vẫn có thể biên dịch hoặc khởi chạy binary khác sau khi vượt qua kiểm tra guard ban đầu. Trong thực tế, hãy coi script build, Makefile, script package, và binary được tạo như mã thực thi cần cùng mức độ review như lệnh shell trực tiếp.

Cho môi trường rủi ro cao hơn:

* Review script build trước khi thực thi.
* Ưu tiên phê duyệt/review thủ công cho quy trình biên dịch và chạy.
* Chạy PicoClaw trong container hoặc VM nếu bạn cần cách ly mạnh hơn guard tích hợp.

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
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
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

PicoClaw có thể thực hiện tác vụ định kỳ tự động. Tạo file `HEARTBEAT.md` trong workspace:

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

## Tác Vụ Nhanh (trả lời trực tiếp)

- Báo giờ hiện tại

## Tác Vụ Dài (dùng spawn cho bất đồng bộ)

- Tìm kiếm tin tức AI trên web và tóm tắt
- Kiểm tra email và báo cáo tin nhắn quan trọng
```

**Hành vi chính:**

| Tính năng        | Mô tả                                                              |
| ---------------- | ------------------------------------------------------------------ |
| **spawn**        | Tạo subagent bất đồng bộ, không chặn heartbeat                    |
| **Ngữ cảnh độc lập** | Subagent có ngữ cảnh riêng, không có lịch sử phiên             |
| **message tool** | Subagent giao tiếp trực tiếp với người dùng qua message tool      |
| **Không chặn**   | Sau khi spawn, heartbeat tiếp tục tác vụ tiếp theo                |

#### Luồng Giao Tiếp Của Subagent

```
Heartbeat kích hoạt
    ↓
Agent đọc HEARTBEAT.md
    ↓
Tác vụ dài: spawn subagent
    ↓                           ↓
Tiếp tục tác vụ tiếp theo  Subagent hoạt động độc lập
    ↓                           ↓
Hoàn thành tất cả tác vụ   Subagent dùng công cụ "message"
    ↓                           ↓
Trả lời HEARTBEAT_OK        Người dùng nhận kết quả trực tiếp
```

**Cấu hình:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Tùy chọn   | Mặc định | Mô tả                                  |
| ---------- | -------- | -------------------------------------- |
| `enabled`  | `true`   | Bật/tắt heartbeat                      |
| `interval` | `30`     | Khoảng thời gian kiểm tra tính bằng phút (tối thiểu: 5) |

**Biến môi trường:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` để tắt
* `PICOCLAW_HEARTBEAT_INTERVAL=60` để thay đổi khoảng thời gian

### Providers

> [!NOTE]
> Groq cung cấp chuyển đổi giọng nói thành văn bản miễn phí qua Whisper. Nếu được cấu hình, tin nhắn âm thanh từ bất kỳ kênh nào sẽ được tự động chuyển đổi ở cấp độ agent.

| Provider     | Mục đích                                | Lấy API Key                                                  |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM (Gemini trực tiếp)                  | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM (Zhipu trực tiếp)                   | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM (Volcengine trực tiếp)              | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM (khuyến nghị, truy cập tất cả mô hình) | [openrouter.ai](https://openrouter.ai)                   |
| `anthropic`  | LLM (Claude trực tiếp)                  | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM (GPT trực tiếp)                     | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM (DeepSeek trực tiếp)                | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM (Qwen trực tiếp)                    | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **Chuyển đổi giọng nói** (Whisper)| [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM (Cerebras trực tiếp)                | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM (Vivgrid trực tiếp)                 | [vivgrid.com](https://vivgrid.com)                           |

### Cấu Hình Mô Hình (model_list)

> **Tính năng mới:** PicoClaw hiện sử dụng cách tiếp cận **lấy mô hình làm trung tâm**. Chỉ cần chỉ định định dạng `vendor/model` (ví dụ: `zhipu/glm-4.7`) để thêm provider mới — **không cần thay đổi code!**

#### Tất Cả Vendor Được Hỗ Trợ

| Vendor                  | Tiền tố `model` | API Base mặc định                                   | Giao thức | API Key                                                          |
| ----------------------- | --------------- | --------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| **OpenAI**              | `openai/`       | `https://api.openai.com/v1`                         | OpenAI    | [Lấy](https://platform.openai.com)                               |
| **Anthropic**           | `anthropic/`    | `https://api.anthropic.com/v1`                      | Anthropic | [Lấy](https://console.anthropic.com)                             |
| **智谱 AI (GLM)**       | `zhipu/`        | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Lấy](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)     |
| **DeepSeek**            | `deepseek/`     | `https://api.deepseek.com/v1`                       | OpenAI    | [Lấy](https://platform.deepseek.com)                             |
| **Google Gemini**       | `gemini/`       | `https://generativelanguage.googleapis.com/v1beta`  | OpenAI    | [Lấy](https://aistudio.google.com/api-keys)                      |
| **Groq**                | `groq/`         | `https://api.groq.com/openai/v1`                    | OpenAI    | [Lấy](https://console.groq.com)                                  |
| **通义千问 (Qwen)**     | `qwen/`         | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Lấy](https://dashscope.console.aliyun.com)                      |
| **Ollama**              | `ollama/`       | `http://localhost:11434/v1`                         | OpenAI    | Cục bộ (không cần key)                                           |
| **OpenRouter**          | `openrouter/`   | `https://openrouter.ai/api/v1`                      | OpenAI    | [Lấy](https://openrouter.ai/keys)                                |
| **VolcEngine (Doubao)** | `volcengine/`   | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Lấy](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **Antigravity**         | `antigravity/`  | Google Cloud                                        | Custom    | Chỉ OAuth                                                        |

#### Cân Bằng Tải

Cấu hình nhiều endpoint cho cùng tên mô hình — PicoClaw sẽ tự động round-robin:

```json
{
  "model_list": [
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api1.example.com/v1", "api_keys": ["sk-key1"] },
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api2.example.com/v1", "api_keys": ["sk-key2"] }
  ]
}
```

#### Di Chuyển Từ Cấu Hình `providers` Cũ

Cấu hình `providers` cũ đã **bị deprecated** và đã được loại bỏ trong V2. Các cấu hình V0/V1 hiện có sẽ được tự động migrate. Xem [docs/migration/model-list-migration.md](../migration/model-list-migration.md).

### Kiến Trúc Provider

PicoClaw định tuyến provider theo họ giao thức:

- **Tương thích OpenAI**: OpenRouter, Groq, Zhipu, endpoint kiểu vLLM và hầu hết các provider khác.
- **Anthropic**: Hành vi API Claude gốc.
- **Codex/OAuth**: Tuyến xác thực OAuth/token OpenAI.

### Tác Vụ Đã Lên Lịch / Nhắc Nhở

PicoClaw hỗ trợ tác vụ theo lịch qua công cụ `cron`.

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5
    }
  }
}
```

Tác vụ đã lên lịch được lưu trữ bền vững sau khi khởi động lại tại `~/.picoclaw/workspace/cron/`.

### Chủ Đề Nâng Cao

| Chủ đề | Mô tả |
| ------ | ----- |
| [Hệ Thống Hook](../architecture/hooks/README.md) | Hook hướng sự kiện: observer, interceptor, approval hook |
| [Steering](../architecture/steering.md) | Chèn tin nhắn vào vòng lặp agent đang chạy |
| [SubTurn](../architecture/subturn.md) | Điều phối subagent, kiểm soát đồng thời, vòng đời |
| [Quản Lý Ngữ Cảnh](../architecture/agent-refactor/context.md) | Phát hiện ranh giới ngữ cảnh, nén |

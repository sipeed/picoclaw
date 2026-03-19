# 🔧 Cấu Hình Công Cụ

> Quay lại [README](../../README.vi.md)

Cấu hình công cụ của PicoClaw nằm trong trường `tools` của `config.json`.

## Cấu trúc thư mục

```json
{
  "tools": {
    "web": {
      ...
    },
    "mcp": {
      ...
    },
    "exec": {
      ...
    },
    "cron": {
      ...
    },
    "skills": {
      ...
    }
  }
}
```

## Công cụ Web

Các công cụ web được sử dụng để tìm kiếm và tải nội dung web.

### Web Fetcher
Cài đặt chung để tải và xử lý nội dung trang web.

| Cấu hình            | Kiểu   | Mặc định      | Mô tả                                                                                        |
|----------------------|--------|---------------|-----------------------------------------------------------------------------------------------|
| `enabled`            | bool   | true          | Bật khả năng tải trang web.                                                                   |
| `fetch_limit_bytes`  | int    | 10485760      | Kích thước tối đa của payload trang web cần tải, tính bằng byte (mặc định là 10MB).          |
| `format`             | string | "plaintext"   | Định dạng đầu ra của nội dung đã tải. Tùy chọn: `plaintext` hoặc `markdown` (khuyến nghị).   |

### Brave

| Cấu hình      | Kiểu   | Mặc định | Mô tả                     |
|----------------|--------|----------|----------------------------|
| `enabled`      | bool   | false    | Bật tìm kiếm Brave        |
| `api_key`      | string | -        | Khóa API Brave Search      |
| `max_results`  | int    | 5        | Số kết quả tối đa         |

### DuckDuckGo

| Cấu hình      | Kiểu | Mặc định | Mô tả                        |
|----------------|------|----------|-------------------------------|
| `enabled`      | bool | true     | Bật tìm kiếm DuckDuckGo      |
| `max_results`  | int  | 5        | Số kết quả tối đa            |

### Perplexity

| Cấu hình      | Kiểu   | Mặc định | Mô tả                        |
|----------------|--------|----------|-------------------------------|
| `enabled`      | bool   | false    | Bật tìm kiếm Perplexity      |
| `api_key`      | string | -        | Khóa API Perplexity           |
| `max_results`  | int    | 5        | Số kết quả tối đa            |

## Công cụ Exec

Công cụ exec được sử dụng để thực thi các lệnh shell.

| Cấu hình               | Kiểu  | Mặc định | Mô tả                                         |
|--------------------------|-------|----------|------------------------------------------------|
| `enable_deny_patterns`   | bool  | true     | Bật chặn lệnh nguy hiểm mặc định             |
| `custom_deny_patterns`   | array | []       | Mẫu từ chối tùy chỉnh (biểu thức chính quy)  |

### Chức năng

- **`enable_deny_patterns`**: Đặt thành `false` để tắt hoàn toàn các mẫu chặn lệnh nguy hiểm mặc định
- **`custom_deny_patterns`**: Thêm các mẫu regex từ chối tùy chỉnh; các lệnh khớp sẽ bị chặn

### Các mẫu lệnh bị chặn mặc định

Theo mặc định, PicoClaw chặn các lệnh nguy hiểm sau:

- Lệnh xóa: `rm -rf`, `del /f/q`, `rmdir /s`
- Thao tác đĩa: `format`, `mkfs`, `diskpart`, `dd if=`, ghi vào `/dev/sd*`
- Thao tác hệ thống: `shutdown`, `reboot`, `poweroff`
- Thay thế lệnh: `$()`, `${}`, dấu backtick
- Pipe đến shell: `| sh`, `| bash`
- Leo thang đặc quyền: `sudo`, `chmod`, `chown`
- Điều khiển tiến trình: `pkill`, `killall`, `kill -9`
- Thao tác từ xa: `curl | sh`, `wget | sh`, `ssh`
- Quản lý gói: `apt`, `yum`, `dnf`, `npm install -g`, `pip install --user`
- Container: `docker run`, `docker exec`
- Git: `git push`, `git force`
- Khác: `eval`, `source *.sh`

### Hạn chế kiến trúc đã biết

Bộ bảo vệ exec chỉ xác thực lệnh cấp cao nhất được gửi đến PicoClaw. Nó **không** kiểm tra đệ quy các tiến trình con được tạo bởi các công cụ build hoặc script sau khi lệnh đó bắt đầu chạy.

Ví dụ về các quy trình có thể bỏ qua bộ bảo vệ lệnh trực tiếp sau khi lệnh ban đầu được cho phép:

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

Điều này có nghĩa là bộ bảo vệ hữu ích để chặn các lệnh trực tiếp rõ ràng nguy hiểm, nhưng nó **không phải** là sandbox đầy đủ cho các pipeline build chưa được xem xét. Nếu mô hình mối đe dọa của bạn bao gồm mã không đáng tin cậy trong workspace, hãy sử dụng cách ly mạnh hơn như container, VM hoặc quy trình phê duyệt xung quanh các lệnh build và chạy.

### Ví dụ cấu hình

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ]
    }
  }
}
```

## Công cụ Cron

Công cụ cron được sử dụng để lên lịch các tác vụ định kỳ.

| Cấu hình               | Kiểu | Mặc định | Mô tả                                              |
|--------------------------|------|----------|-----------------------------------------------------|
| `exec_timeout_minutes`   | int  | 5        | Thời gian chờ thực thi tính bằng phút, 0 nghĩa là không giới hạn |

## Công cụ MCP

Công cụ MCP cho phép tích hợp với các máy chủ Model Context Protocol bên ngoài.

### Khám phá công cụ (tải chậm)

Khi kết nối với nhiều máy chủ MCP, việc hiển thị hàng trăm công cụ cùng lúc có thể làm cạn kiệt cửa sổ ngữ cảnh của LLM và tăng chi phí API. Tính năng **Discovery** giải quyết vấn đề này bằng cách giữ các công cụ MCP *ẩn* theo mặc định.

Thay vì tải tất cả các công cụ, LLM được cung cấp một công cụ tìm kiếm nhẹ (sử dụng khớp từ khóa BM25 hoặc Regex). Khi LLM cần một khả năng cụ thể, nó tìm kiếm trong thư viện ẩn. Các công cụ khớp sau đó được tạm thời "mở khóa" và đưa vào ngữ cảnh trong số lượt được cấu hình (`ttl`).

### Cấu hình toàn cục

| Cấu hình    | Kiểu   | Mặc định | Mô tả                                        |
|-------------|--------|----------|-----------------------------------------------|
| `enabled`   | bool   | false    | Bật tích hợp MCP toàn cục                    |
| `discovery` | object | `{}`     | Cấu hình khám phá công cụ (xem bên dưới)    |
| `servers`   | object | `{}`     | Ánh xạ tên máy chủ đến cấu hình máy chủ     |

### Cấu hình Discovery (`discovery`)

| Cấu hình             | Kiểu | Mặc định | Mô tả                                                                                                                            |
|----------------------|------|----------|-----------------------------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false    | Nếu true, các công cụ MCP bị ẩn và được tải theo yêu cầu qua tìm kiếm. Nếu false, tất cả công cụ được tải                      |
| `ttl`                | int  | 5        | Số lượt hội thoại mà một công cụ đã khám phá vẫn được mở khóa                                                                   |
| `max_search_results` | int  | 5        | Số công cụ tối đa được trả về cho mỗi truy vấn tìm kiếm                                                                         |
| `use_bm25`           | bool | true     | Bật công cụ tìm kiếm ngôn ngữ tự nhiên/từ khóa (`tool_search_tool_bm25`). **Cảnh báo**: tiêu tốn nhiều tài nguyên hơn tìm kiếm regex |
| `use_regex`          | bool | false    | Bật công cụ tìm kiếm mẫu regex (`tool_search_tool_regex`)                                                                        |

> **Lưu ý:** Nếu `discovery.enabled` là `true`, bạn **phải** bật ít nhất một công cụ tìm kiếm (`use_bm25` hoặc `use_regex`),
> nếu không ứng dụng sẽ không khởi động được.

### Cấu hình từng máy chủ

| Cấu hình   | Kiểu   | Bắt buộc | Mô tả                                     |
|------------|--------|----------|--------------------------------------------|
| `enabled`  | bool   | có       | Bật máy chủ MCP này                       |
| `type`     | string | không    | Loại truyền tải: `stdio`, `sse`, `http`   |
| `command`  | string | stdio    | Lệnh thực thi cho truyền tải stdio        |
| `args`     | array  | không    | Đối số lệnh cho truyền tải stdio          |
| `env`      | object | không    | Biến môi trường cho tiến trình stdio      |
| `env_file` | string | không    | Đường dẫn đến tệp môi trường cho tiến trình stdio |
| `url`      | string | sse/http | URL endpoint cho truyền tải `sse`/`http`  |
| `headers`  | object | không    | Header HTTP cho truyền tải `sse`/`http`   |

### Hành vi truyền tải

- Nếu bỏ qua `type`, truyền tải được tự động phát hiện:
    - `url` được đặt → `sse`
    - `command` được đặt → `stdio`
- `http` và `sse` đều sử dụng `url` + `headers` tùy chọn.
- `env` và `env_file` chỉ được áp dụng cho máy chủ `stdio`.

### Ví dụ cấu hình

#### 1) Máy chủ MCP Stdio

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-filesystem",
            "/tmp"
          ]
        }
      }
    }
  }
}
```

#### 2) Máy chủ MCP từ xa SSE/HTTP

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "remote-mcp": {
          "enabled": true,
          "type": "sse",
          "url": "https://example.com/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_TOKEN"
          }
        }
      }
    }
  }
}
```

#### 3) Thiết lập MCP quy mô lớn với khám phá công cụ được bật

*Trong ví dụ này, LLM chỉ thấy `tool_search_tool_bm25`. Nó sẽ tìm kiếm và mở khóa động các công cụ Github hoặc Postgres chỉ khi được người dùng yêu cầu.*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true,
        "use_regex": false
      },
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-github"
          ],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_TOKEN"
          }
        },
        "postgres": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-postgres",
            "postgresql://user:password@localhost/dbname"
          ]
        },
        "slack": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-slack"
          ],
          "env": {
            "SLACK_BOT_TOKEN": "YOUR_SLACK_BOT_TOKEN",
            "SLACK_TEAM_ID": "YOUR_SLACK_TEAM_ID"
          }
        }
      }
    }
  }
}
```

## Công cụ Skills

Công cụ skills cấu hình khám phá và cài đặt kỹ năng thông qua các registry như ClawHub.

### Registry

| Cấu hình                          | Kiểu   | Mặc định             | Mô tả                                       |
|------------------------------------|--------|-----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                  | Bật registry ClawHub                         |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai`  | URL cơ sở ClawHub                            |
| `registries.clawhub.auth_token`    | string | `""`                  | Token Bearer tùy chọn để có giới hạn tốc độ cao hơn |
| `registries.clawhub.search_path`   | string | `/api/v1/search`      | Đường dẫn API tìm kiếm                      |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`      | Đường dẫn API Skills                         |
| `registries.clawhub.download_path` | string | `/api/v1/download`    | Đường dẫn API tải xuống                      |

### Ví dụ cấu hình

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": "",
          "search_path": "/api/v1/search",
          "skills_path": "/api/v1/skills",
          "download_path": "/api/v1/download"
        }
      }
    }
  }
}
```

## Biến môi trường

Tất cả các tùy chọn cấu hình có thể được ghi đè qua biến môi trường với định dạng `PICOCLAW_TOOLS_<SECTION>_<KEY>`:

Ví dụ:

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

Lưu ý: Cấu hình kiểu map lồng nhau (ví dụ `tools.mcp.servers.<name>.*`) được cấu hình trong `config.json` thay vì qua biến môi trường.

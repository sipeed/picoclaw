# 🐳 Docker và Bắt Đầu Nhanh

> Quay lại [README](../../README.vi.md)

## 🐳 Docker Compose

Bạn cũng có thể chạy PicoClaw bằng Docker Compose mà không cần cài đặt gì trên máy.

```bash
# 1. Clone repo này
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Lần chạy đầu tiên — tự động tạo docker/data/config.json rồi thoát
#    (chỉ kích hoạt khi cả config.json và workspace/ đều không tồn tại)
docker compose -f docker/docker-compose.yml --profile gateway up
# Container hiển thị "First-run setup complete." và dừng lại.

# 3. Cấu hình API key của bạn
vim docker/data/config.json   # Set provider API keys, bot tokens, etc.

# 4. Khởi động
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Người dùng Docker**: Mặc định, Gateway lắng nghe trên `127.0.0.1`, không thể truy cập từ host. Nếu bạn cần truy cập các health endpoint hoặc mở port, hãy đặt `PICOCLAW_GATEWAY_HOST=0.0.0.0` trong môi trường hoặc cập nhật `config.json`.

```bash
# 5. Kiểm tra log
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Dừng
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Chế Độ Launcher (Web Console)

Image `launcher` bao gồm cả ba binary (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) và khởi động web console mặc định, cung cấp giao diện trình duyệt để cấu hình và chat.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Mở http://localhost:18800 trong trình duyệt. Launcher tự động quản lý tiến trình gateway.

> [!WARNING]
> Web console chưa hỗ trợ xác thực. Tránh để lộ ra internet công cộng.

### Chế Độ Agent (One-shot)

```bash
# Đặt câu hỏi
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "What is 2+2?"

# Chế độ tương tác
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Cập Nhật

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Bắt Đầu Nhanh

> [!TIP]
> Cấu hình API Key trong `~/.picoclaw/config.json`. Lấy API Key: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). Tìm kiếm web là tùy chọn — lấy miễn phí [Tavily API](https://tavily.com) (1000 truy vấn miễn phí/tháng) hoặc [Brave Search API](https://brave.com/search/api) (2000 truy vấn miễn phí/tháng).

**1. Khởi tạo**

```bash
picoclaw onboard
```

**2. Cấu hình** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model_name": "gpt-5.4",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key",
      "api_base":"https://ark.cn-beijing.volces.com/api/coding/v3"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "your-api-key",
      "request_timeout": 300
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "your-anthropic-key"
    }
  ],
  "tools": {
    "web": {
      "enabled": true,
      "fetch_limit_bytes": 10485760,
      "format": "plaintext",
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "tavily": {
        "enabled": false,
        "api_key": "YOUR_TAVILY_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://your-searxng-instance:8888",
        "max_results": 5
      }
    }
  }
}
```

> **Mới**: Định dạng cấu hình `model_list` cho phép thêm provider mà không cần thay đổi code. Xem [Cấu Hình Mô Hình](#cấu-hình-mô-hình-model_list) để biết chi tiết.
> `request_timeout` là tùy chọn và tính bằng giây. Nếu bỏ qua hoặc đặt `<= 0`, PicoClaw sử dụng timeout mặc định (120s).

**3. Lấy API Key**

* **Nhà cung cấp LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Tìm kiếm Web** (tùy chọn):
  * [Brave Search](https://brave.com/search/api) - Trả phí ($5/1000 truy vấn, ~$5-6/tháng)
  * [Perplexity](https://www.perplexity.ai) - Tìm kiếm bằng AI với giao diện chat
  * [SearXNG](https://github.com/searxng/searxng) - Công cụ tìm kiếm tổng hợp tự host (miễn phí, không cần API key)
  * [Tavily](https://tavily.com) - Tối ưu cho AI Agent (1000 yêu cầu/tháng)
  * DuckDuckGo - Fallback tích hợp (không cần API key)

> **Lưu ý**: Xem `config.example.json` để có mẫu cấu hình đầy đủ.

**4. Chat**

```bash
picoclaw agent -m "What is 2+2?"
```

Vậy là xong! Bạn có một trợ lý AI hoạt động trong 2 phút.

---

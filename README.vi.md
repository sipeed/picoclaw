<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Trợ lý AI Siêu Nhẹ viết bằng Go</h1>

<h3>Phần cứng $10 · RAM 10MB · Khởi động ms · Let's Go, PicoClaw!</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | **Tiếng Việt** | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [Malay](README.my.md) | [English](README.md)

</div>

---

> **PicoClaw** là một dự án mã nguồn mở độc lập do [Sipeed](https://sipeed.com) khởi xướng, được viết hoàn toàn bằng **Go** từ đầu — không phải fork của OpenClaw, NanoBot hay bất kỳ dự án nào khác.

**PicoClaw** là trợ lý AI cá nhân siêu nhẹ lấy cảm hứng từ [NanoBot](https://github.com/HKUDS/nanobot). Nó được xây dựng lại từ đầu bằng **Go** thông qua quá trình "tự khởi động" — chính AI Agent đã dẫn dắt quá trình di chuyển kiến trúc và tối ưu hóa mã nguồn.

**Chạy trên phần cứng $10 với <10MB RAM** — ít hơn 99% bộ nhớ so với OpenClaw và rẻ hơn 98% so với Mac mini!

<table align="center">
<tr align="center">
<td align="center" valign="top">
<p align="center">
<img src="assets/picoclaw_mem.gif" width="360" height="240">
</p>
</td>
<td align="center" valign="top">
<p align="center">
<img src="assets/licheervnano.png" width="400" height="240">
</p>
</td>
</tr>
</table>

> [!CAUTION]
> **Thông báo Bảo mật**
>
> * **KHÔNG CÓ CRYPTO:** PicoClaw **chưa** phát hành bất kỳ token hay tiền điện tử chính thức nào. Mọi thông tin trên `pump.fun` hoặc các nền tảng giao dịch khác đều là **lừa đảo**.
> * **DOMAIN CHÍNH THỨC:** Website chính thức **DUY NHẤT** là **[picoclaw.io](https://picoclaw.io)**, và website công ty là **[sipeed.com](https://sipeed.com)**
> * **CẢNH BÁO:** Nhiều domain `.ai/.org/.com/.net/...` đã bị bên thứ ba đăng ký. Đừng tin tưởng chúng.
> * **LƯU Ý:** PicoClaw đang trong giai đoạn phát triển nhanh. Có thể còn các vấn đề bảo mật chưa được giải quyết. Không triển khai lên môi trường production trước v1.0.
> * **LƯU Ý:** PicoClaw gần đây đã merge nhiều PR. Các bản build gần đây có thể dùng 10-20MB RAM. Tối ưu hóa tài nguyên được lên kế hoạch sau khi tính năng ổn định.

## 📢 Tin tức

2026-03-25 🚀 **v0.2.4 đã phát hành!** Tái cấu trúc kiến trúc Agent (SubTurn, Hooks, Steering, EventBus), tích hợp WeChat/WeCom, tăng cường bảo mật (.security.yml, lọc dữ liệu nhạy cảm), provider mới (AWS Bedrock, Azure, Xiaomi MiMo) và 35 bản vá lỗi. PicoClaw đã đạt **26K Stars**!

2026-03-17 🚀 **v0.2.3 đã phát hành!** Giao diện system tray (Windows & Linux), truy vấn trạng thái sub-agent (`spawn_status`), thử nghiệm Gateway hot-reload, bảo mật Cron, và 2 bản vá bảo mật. PicoClaw đã đạt **25K Stars**!

2026-03-09 🎉 **v0.2.1 — Bản cập nhật lớn nhất từ trước đến nay!** Hỗ trợ giao thức MCP, 4 Channel mới (Matrix/IRC/WeCom/Discord Proxy), 3 Provider mới (Kimi/Minimax/Avian), pipeline thị giác, bộ nhớ JSONL, định tuyến mô hình.

2026-02-28 📦 **v0.2.0** phát hành với hỗ trợ Docker Compose và Web UI Launcher.

<details>
<summary>Tin tức trước đó...</summary>

2026-02-26 🎉 PicoClaw đạt **20K Stars** chỉ trong 17 ngày! Tự động điều phối Channel và giao diện khả năng đã hoạt động.

2026-02-16 🎉 PicoClaw vượt 12K Stars trong một tuần! Vai trò người duy trì cộng đồng và [Lộ trình](ROADMAP.md) chính thức ra mắt.

2026-02-13 🎉 PicoClaw vượt 5000 Stars trong 4 ngày! Lộ trình dự án và nhóm nhà phát triển đang được xây dựng.

2026-02-09 🎉 **PicoClaw ra mắt!** Được xây dựng trong 1 ngày để đưa AI Agent lên phần cứng $10 với <10MB RAM. Let's Go, PicoClaw!

</details>

## ✨ Tính năng

🪶 **Siêu nhẹ**: Bộ nhớ lõi <10MB — nhỏ hơn 99% so với OpenClaw.*

💰 **Chi phí tối thiểu**: Đủ hiệu quả để chạy trên phần cứng $10 — rẻ hơn 98% so với Mac mini.

⚡️ **Khởi động cực nhanh**: Khởi động nhanh hơn 400 lần. Khởi động trong <1 giây ngay cả trên bộ xử lý đơn nhân 0.6GHz.

🌍 **Thực sự di động**: Một binary duy nhất cho các kiến trúc RISC-V, ARM, MIPS và x86. Một binary, chạy mọi nơi!

🤖 **Được AI khởi động**: Triển khai Go thuần túy — 95% mã lõi được tạo bởi Agent và tinh chỉnh qua quy trình human-in-the-loop.

🔌 **Hỗ trợ MCP**: Tích hợp [Model Context Protocol](https://modelcontextprotocol.io/) gốc — kết nối bất kỳ MCP server nào để mở rộng khả năng Agent.

👁️ **Pipeline thị giác**: Gửi hình ảnh và tệp trực tiếp đến Agent — tự động mã hóa base64 cho LLM đa phương thức.

🧠 **Định tuyến thông minh**: Định tuyến mô hình dựa trên quy tắc — các truy vấn đơn giản đến mô hình nhẹ, tiết kiệm chi phí API.

_*Các bản build gần đây có thể dùng 10-20MB do merge PR nhanh. Tối ưu hóa tài nguyên đang được lên kế hoạch. So sánh tốc độ khởi động dựa trên benchmark lõi đơn 0.8GHz (xem bảng bên dưới)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **Ngôn ngữ**                   | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **Thời gian khởi động**</br>(lõi 0.8GHz) | >500s         | >30s                     | **<1s**                                |
| **Chi phí**                    | Mac Mini $599 | Hầu hết board Linux ~$50 | **Bất kỳ board Linux**</br>**từ $10**  |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[Danh sách Tương thích Phần cứng](docs/vi/hardware-compatibility.md)** — Xem tất cả các board đã được kiểm tra, từ RISC-V $5 đến Raspberry Pi đến điện thoại Android. Board của bạn chưa có trong danh sách? Gửi PR!

<p align="center">
<img src="assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

## 🦾 Minh họa

### 🛠️ Quy trình Trợ lý Tiêu chuẩn

<table align="center">
<tr align="center">
<th><p align="center">Chế độ Kỹ sư Full-Stack</p></th>
<th><p align="center">Ghi nhật ký & Lập kế hoạch</p></th>
<th><p align="center">Tìm kiếm Web & Học tập</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Phát triển · Triển khai · Mở rộng</td>
<td align="center">Lên lịch · Tự động hóa · Ghi nhớ</td>
<td align="center">Khám phá · Thông tin · Xu hướng</td>
</tr>
</table>

### 🐜 Triển khai Sáng tạo với Dấu chân Nhỏ

PicoClaw có thể được triển khai trên hầu hết mọi thiết bị Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) phiên bản E(Ethernet) hoặc W(WiFi6), cho trợ lý gia đình tối giản
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), hoặc $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), cho vận hành máy chủ tự động
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) hoặc $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), cho giám sát thông minh

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Còn nhiều trường hợp triển khai đang chờ đón!

## 📦 Cài đặt

### Tải xuống từ picoclaw.io (Khuyến nghị)

Truy cập **[picoclaw.io](https://picoclaw.io)** — website chính thức tự động phát hiện nền tảng của bạn và cung cấp tải xuống một cú nhấp. Không cần chọn kiến trúc thủ công.

### Tải xuống binary đã biên dịch sẵn

Ngoài ra, tải binary cho nền tảng của bạn từ trang [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Xây dựng từ mã nguồn (để phát triển)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build core binary
make build

# Build Web UI Launcher (required for WebUI mode)
make build-launcher

# Build for multiple platforms
make build-all

# Build for Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build and install
make install
```

**Raspberry Pi Zero 2 W:** Sử dụng binary phù hợp với hệ điều hành của bạn: Raspberry Pi OS 32-bit -> `make build-linux-arm`; 64-bit -> `make build-linux-arm64`. Hoặc chạy `make build-pi-zero` để xây dựng cả hai.

## 🚀 Hướng dẫn Khởi động Nhanh

### 🌐 WebUI Launcher (Khuyến nghị cho Desktop)

WebUI Launcher cung cấp giao diện dựa trên trình duyệt để cấu hình và trò chuyện. Đây là cách dễ nhất để bắt đầu — không cần kiến thức dòng lệnh.

**Tùy chọn 1: Nhấp đúp (Desktop)**

Sau khi tải xuống từ [picoclaw.io](https://picoclaw.io), nhấp đúp vào `picoclaw-launcher` (hoặc `picoclaw-launcher.exe` trên Windows). Trình duyệt của bạn sẽ tự động mở tại `http://localhost:18800`.

**Tùy chọn 2: Dòng lệnh**

```bash
picoclaw-launcher
# Mở http://localhost:18800 trong trình duyệt của bạn
```

> [!TIP]
> **Truy cập từ xa / Docker / VM:** Thêm cờ `-public` để lắng nghe trên tất cả giao diện:
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Bắt đầu:**

Mở WebUI, sau đó: **1)** Cấu hình Provider (thêm API key LLM của bạn) -> **2)** Cấu hình Channel (ví dụ: Telegram) -> **3)** Khởi động Gateway -> **4)** Trò chuyện!

Để biết tài liệu WebUI chi tiết, xem [docs.picoclaw.io](https://docs.picoclaw.io).

<details>
<summary><b>Docker (thay thế)</b></summary>

```bash
# 1. Clone this repo
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. First run — auto-generates docker/data/config.json then exits
#    (only triggers when both config.json and workspace/ are missing)
docker compose -f docker/docker-compose.yml --profile launcher up
# The container prints "First-run setup complete." and stops.

# 3. Set your API keys
vim docker/data/config.json

# 4. Start
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Open http://localhost:18800
```

> **Người dùng Docker / VM:** Gateway lắng nghe trên `127.0.0.1` theo mặc định. Đặt `PICOCLAW_GATEWAY_HOST=0.0.0.0` hoặc dùng cờ `-public` để có thể truy cập từ host.

```bash
# Check logs
docker compose -f docker/docker-compose.yml logs -f

# Stop
docker compose -f docker/docker-compose.yml --profile launcher down

# Update
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

<details>
<summary><b>macOS — Cảnh báo bảo mật khi khởi chạy lần đầu</b></summary>

macOS có thể chặn `picoclaw-launcher` khi khởi chạy lần đầu vì nó được tải từ internet và chưa được công chứng qua Mac App Store.

**Bước 1:** Nhấp đúp vào `picoclaw-launcher`. Bạn sẽ thấy cảnh báo bảo mật:

<p align="center">
<img src="assets/macos-gatekeeper-warning.jpg" alt="Cảnh báo macOS Gatekeeper" width="400">
</p>

> *"picoclaw-launcher" Không Mở Được — Apple không thể xác minh "picoclaw-launcher" không chứa phần mềm độc hại có thể gây hại cho Mac hoặc xâm phạm quyền riêng tư của bạn.*

**Bước 2:** Mở **Cài đặt Hệ thống** → **Quyền riêng tư & Bảo mật** → cuộn xuống phần **Bảo mật** → nhấp **Vẫn Mở** → xác nhận bằng cách nhấp **Vẫn Mở** trong hộp thoại.

<p align="center">
<img src="assets/macos-gatekeeper-allow.jpg" alt="macOS Quyền riêng tư & Bảo mật — Vẫn Mở" width="600">
</p>

Sau bước này, `picoclaw-launcher` sẽ mở bình thường trong các lần khởi chạy tiếp theo.

</details>

### 💻 TUI Launcher (Khuyến nghị cho Headless / SSH)

TUI (Terminal UI) Launcher cung cấp giao diện terminal đầy đủ tính năng để cấu hình và quản lý. Lý tưởng cho máy chủ, Raspberry Pi và các môi trường headless khác.

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**Bắt đầu:**

Sử dụng menu TUI để: **1)** Cấu hình Provider -> **2)** Cấu hình Channel -> **3)** Khởi động Gateway -> **4)** Trò chuyện!

Để biết tài liệu TUI chi tiết, xem [docs.picoclaw.io](https://docs.picoclaw.io).

### 📱 Android

Hãy cho chiếc điện thoại cũ của bạn một cuộc sống mới! Biến nó thành Trợ lý AI thông minh với PicoClaw.

**Tùy chọn 1: Termux (có sẵn ngay)**

1. Cài đặt [Termux](https://github.com/termux/termux-app) (tải từ [GitHub Releases](https://github.com/termux/termux-app/releases), hoặc tìm kiếm trong F-Droid / Google Play)
2. Chạy các lệnh sau:

```bash
# Download the latest release
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot provides a standard Linux filesystem layout
```

Sau đó làm theo phần Terminal Launcher bên dưới để hoàn tất cấu hình.

<img src="assets/termux.jpg" alt="PicoClaw on Termux" width="512">

**Tùy chọn 2: Cài đặt APK (sắp ra mắt)**

Một APK Android độc lập với WebUI tích hợp đang được phát triển. Hãy đón chờ!

<details>
<summary><b>Terminal Launcher (cho môi trường hạn chế tài nguyên)</b></summary>

Đối với các môi trường tối giản chỉ có binary lõi `picoclaw` (không có Launcher UI), bạn có thể cấu hình mọi thứ qua dòng lệnh và tệp cấu hình JSON.

**1. Khởi tạo**

```bash
picoclaw onboard
```

Lệnh này tạo `~/.picoclaw/config.json` và thư mục workspace.

**2. Cấu hình** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-api-key"
    }
  ]
}
```

> Xem `config/config.example.json` trong repo để có mẫu cấu hình đầy đủ với tất cả các tùy chọn có sẵn.

**3. Trò chuyện**

```bash
# One-shot question
picoclaw agent -m "What is 2+2?"

# Interactive mode
picoclaw agent

# Start gateway for chat app integration
picoclaw gateway
```

</details>

## 🔌 Providers (LLM)

PicoClaw hỗ trợ 30+ Provider LLM thông qua cấu hình `model_list`. Sử dụng định dạng `protocol/model`:

| Provider | Protocol | API Key | Ghi chú |
|----------|----------|---------|---------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Bắt buộc | GPT-5.4, GPT-4o, o3, v.v. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Bắt buộc | Claude Opus 4.6, Sonnet 4.6, v.v. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Bắt buộc | Gemini 3 Flash, 2.5 Pro, v.v. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Bắt buộc | 200+ mô hình, API thống nhất |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Bắt buộc | GLM-4.7, GLM-5, v.v. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Bắt buộc | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Bắt buộc | Doubao, Ark models |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Bắt buộc | Qwen3, Qwen-Max, v.v. |
| [Groq](https://console.groq.com/keys) | `groq/` | Bắt buộc | Suy luận nhanh (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Bắt buộc | Kimi models |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Bắt buộc | MiniMax models |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Bắt buộc | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Bắt buộc | Mô hình do NVIDIA lưu trữ |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Bắt buộc | Suy luận nhanh |
| [Novita AI](https://novita.ai/) | `novita/` | Bắt buộc | Nhiều mô hình mở |
| [Xiaomi MiMo](https://platform.xiaomimimo.com/) | `mimo/` | Bắt buộc | Mô hình MiMo |
| [Ollama](https://ollama.com/) | `ollama/` | Không cần | Mô hình cục bộ, tự lưu trữ |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | Không cần | Triển khai cục bộ, tương thích OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Tùy | Proxy cho 100+ provider |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Bắt buộc | Triển khai Azure doanh nghiệp |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Đăng nhập bằng device code |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |

<details>
<summary><b>Triển khai cục bộ (Ollama, vLLM, v.v.)</b></summary>

**Ollama:**
```json
{
  "model_list": [
    {
      "model_name": "local-llama",
      "model": "ollama/llama3.1:8b",
      "api_base": "http://localhost:11434/v1"
    }
  ]
}
```

**vLLM:**
```json
{
  "model_list": [
    {
      "model_name": "local-vllm",
      "model": "vllm/your-model",
      "api_base": "http://localhost:8000/v1"
    }
  ]
}
```

Để biết chi tiết cấu hình provider đầy đủ, xem [Providers & Models](docs/vi/providers.md).

</details>

## 💬 Channels (Ứng dụng Chat)

Trò chuyện với PicoClaw của bạn qua 17+ nền tảng nhắn tin:

| Channel | Thiết lập | Protocol | Tài liệu |
|---------|-----------|----------|----------|
| **Telegram** | Dễ (bot token) | Long polling | [Hướng dẫn](docs/channels/telegram/README.vi.md) |
| **Discord** | Dễ (bot token + intents) | WebSocket | [Hướng dẫn](docs/channels/discord/README.vi.md) |
| **WhatsApp** | Dễ (quét QR hoặc bridge URL) | Native / Bridge | [Hướng dẫn](docs/vi/chat-apps.md#whatsapp) |
| **Weixin** | Dễ (quét QR gốc) | iLink API | [Hướng dẫn](docs/vi/chat-apps.md#weixin) |
| **QQ** | Dễ (AppID + AppSecret) | WebSocket | [Hướng dẫn](docs/channels/qq/README.vi.md) |
| **Slack** | Dễ (bot + app token) | Socket Mode | [Hướng dẫn](docs/channels/slack/README.vi.md) |
| **Matrix** | Trung bình (homeserver + token) | Sync API | [Hướng dẫn](docs/channels/matrix/README.vi.md) |
| **DingTalk** | Trung bình (client credentials) | Stream | [Hướng dẫn](docs/channels/dingtalk/README.vi.md) |
| **Feishu / Lark** | Trung bình (App ID + Secret) | WebSocket/SDK | [Hướng dẫn](docs/channels/feishu/README.vi.md) |
| **LINE** | Trung bình (credentials + webhook) | Webhook | [Hướng dẫn](docs/channels/line/README.vi.md) |
| **WeCom** | Dễ (đăng nhập QR hoặc thủ công) | WebSocket | [Hướng dẫn](docs/channels/wecom/README.md) |
| **IRC** | Trung bình (server + nick) | IRC protocol | [Hướng dẫn](docs/vi/chat-apps.md#irc) |
| **OneBot** | Trung bình (WebSocket URL) | OneBot v11 | [Hướng dẫn](docs/channels/onebot/README.vi.md) |
| **MaixCam** | Dễ (bật) | TCP socket | [Hướng dẫn](docs/channels/maixcam/README.vi.md) |
| **Pico** | Dễ (bật) | Native protocol | Tích hợp sẵn |
| **Pico Client** | Dễ (WebSocket URL) | WebSocket | Tích hợp sẵn |

> Tất cả các Channel dựa trên webhook dùng chung một Gateway HTTP server (`gateway.host`:`gateway.port`, mặc định `127.0.0.1:18790`). Feishu sử dụng chế độ WebSocket/SDK và không dùng HTTP server chung.

> Mức độ chi tiết log được kiểm soát bởi `gateway.log_level` (mặc định: `warn`). Các giá trị được hỗ trợ: `debug`, `info`, `warn`, `error`, `fatal`. Cũng có thể đặt qua `PICOCLAW_LOG_LEVEL`. Xem [Cấu hình](docs/vi/configuration.md#mức-log-của-gateway) để biết thêm chi tiết.

Để biết hướng dẫn thiết lập Channel chi tiết, xem [Cấu hình Ứng dụng Chat](docs/vi/chat-apps.md).

## 🔧 Tools

### 🔍 Tìm kiếm Web

PicoClaw có thể tìm kiếm web để cung cấp thông tin cập nhật. Cấu hình trong `tools.web`:

| Công cụ Tìm kiếm | API Key | Gói miễn phí | Liên kết |
|------------------|---------|--------------|----------|
| DuckDuckGo | Không cần | Không giới hạn | Dự phòng tích hợp sẵn |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Bắt buộc | 1000 truy vấn/ngày | AI, tối ưu cho tiếng Trung |
| [Tavily](https://tavily.com) | Bắt buộc | 1000 truy vấn/tháng | Tối ưu cho AI Agent |
| [Brave Search](https://brave.com/search/api) | Bắt buộc | 2000 truy vấn/tháng | Nhanh và riêng tư |
| [Perplexity](https://www.perplexity.ai) | Bắt buộc | Trả phí | Tìm kiếm hỗ trợ AI |
| [SearXNG](https://github.com/searxng/searxng) | Không cần | Tự lưu trữ | Metasearch engine miễn phí |
| [GLM Search](https://open.bigmodel.cn/) | Bắt buộc | Tùy | Tìm kiếm web Zhipu |

### ⚙️ Các Tools Khác

PicoClaw bao gồm các tool tích hợp sẵn cho thao tác tệp, thực thi mã, lên lịch và nhiều hơn nữa. Xem [Cấu hình Tools](docs/vi/tools_configuration.md) để biết chi tiết.

## 🎯 Skills

Skills là các khả năng mô-đun mở rộng Agent của bạn. Chúng được tải từ các tệp `SKILL.md` trong workspace của bạn.

**Cài đặt Skills từ ClawHub:**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**Cấu hình token ClawHub** (tùy chọn, để có giới hạn tốc độ cao hơn):

Thêm vào `config.json` của bạn:
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "your-clawhub-token"
        }
      }
    }
  }
}
```

Để biết thêm chi tiết, xem [Cấu hình Tools - Skills](docs/vi/tools_configuration.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

PicoClaw hỗ trợ [MCP](https://modelcontextprotocol.io/) gốc — kết nối bất kỳ MCP server nào để mở rộng khả năng Agent của bạn với các tool và nguồn dữ liệu bên ngoài.

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

Để biết cấu hình MCP đầy đủ (stdio, SSE, HTTP transports, Tool Discovery), xem [Cấu hình Tools - MCP](docs/vi/tools_configuration.md#mcp-tool).

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Tham gia Mạng xã hội Agent

Kết nối PicoClaw với Mạng xã hội Agent chỉ bằng cách gửi một tin nhắn duy nhất qua CLI hoặc bất kỳ Ứng dụng Chat nào đã tích hợp.

**Đọc `https://clawdchat.ai/skill.md` và làm theo hướng dẫn để tham gia [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Tham chiếu CLI

| Lệnh                      | Mô tả                                    |
| ------------------------- | ---------------------------------------- |
| `picoclaw onboard`        | Khởi tạo cấu hình & workspace           |
| `picoclaw auth weixin` | Kết nối tài khoản WeChat qua QR |
| `picoclaw agent -m "..."` | Trò chuyện với agent                     |
| `picoclaw agent`          | Chế độ trò chuyện tương tác             |
| `picoclaw gateway`        | Khởi động gateway                        |
| `picoclaw status`         | Hiển thị trạng thái                      |
| `picoclaw version`        | Hiển thị thông tin phiên bản            |
| `picoclaw model`          | Xem hoặc chuyển đổi mô hình mặc định   |
| `picoclaw cron list`      | Liệt kê tất cả công việc đã lên lịch   |
| `picoclaw cron add ...`   | Thêm công việc đã lên lịch             |
| `picoclaw cron disable`   | Vô hiệu hóa công việc đã lên lịch      |
| `picoclaw cron remove`    | Xóa công việc đã lên lịch              |
| `picoclaw skills list`    | Liệt kê các Skill đã cài đặt           |
| `picoclaw skills install` | Cài đặt một Skill                       |
| `picoclaw migrate`        | Di chuyển dữ liệu từ các phiên bản cũ  |
| `picoclaw auth login`     | Xác thực với các provider               |

### ⏰ Tác vụ Đã lên lịch / Nhắc nhở

PicoClaw hỗ trợ nhắc nhở đã lên lịch và tác vụ định kỳ thông qua tool `cron`:

* **Nhắc nhở một lần**: "Nhắc tôi sau 10 phút" -> kích hoạt một lần sau 10 phút
* **Tác vụ định kỳ**: "Nhắc tôi mỗi 2 giờ" -> kích hoạt mỗi 2 giờ
* **Biểu thức Cron**: "Nhắc tôi lúc 9 giờ sáng hàng ngày" -> sử dụng biểu thức cron

## 📚 Tài liệu

Để biết các hướng dẫn chi tiết ngoài README này:

| Chủ đề | Mô tả |
|--------|-------|
| [Docker & Khởi động Nhanh](docs/vi/docker.md) | Thiết lập Docker Compose, chế độ Launcher/Agent |
| [Ứng dụng Chat](docs/vi/chat-apps.md) | Hướng dẫn thiết lập 17+ Channel |
| [Cấu hình](docs/vi/configuration.md) | Biến môi trường, bố cục workspace, sandbox bảo mật |
| [Providers & Models](docs/vi/providers.md) | 30+ Provider LLM, định tuyến mô hình, cấu hình model_list |
| [Spawn & Tác vụ Bất đồng bộ](docs/vi/spawn-tasks.md) | Tác vụ nhanh, tác vụ dài với spawn, điều phối sub-agent bất đồng bộ |
| [Hooks](docs/hooks/README.md) | Hệ thống hook hướng sự kiện: observer, interceptor, approval hook |
| [Steering](docs/steering.md) | Chèn tin nhắn vào vòng lặp agent đang chạy |
| [SubTurn](docs/subturn.md) | Điều phối subagent, kiểm soát đồng thời, vòng đời |
| [Khắc phục sự cố](docs/vi/troubleshooting.md) | Các vấn đề thường gặp và giải pháp |
| [Cấu hình Tools](docs/vi/tools_configuration.md) | Bật/tắt từng tool, chính sách exec, MCP, Skills |
| [Tương thích Phần cứng](docs/vi/hardware-compatibility.md) | Các board đã kiểm tra, yêu cầu tối thiểu |

## 🤝 Đóng góp & Lộ trình

PR luôn được chào đón! Codebase được thiết kế nhỏ gọn và dễ đọc.

Xem [Lộ trình Cộng đồng](https://github.com/sipeed/picoclaw/issues/988) và [CONTRIBUTING.md](CONTRIBUTING.md) để biết hướng dẫn.

Nhóm nhà phát triển đang được xây dựng, tham gia sau khi PR đầu tiên của bạn được merge!

Nhóm Người dùng:

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="WeChat group QR code" width="512">

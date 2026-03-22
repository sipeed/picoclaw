<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Trợ lý AI Siêu Nhẹ viết bằng Go</h1>

  <h3>Phần cứng $10 · <10MB RAM · Khởi động <1 giây · Nào, xuất phát!</h3>
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

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | **Tiếng Việt** | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** là dự án mã nguồn mở độc lập được khởi xướng bởi [Sipeed](https://sipeed.com). Được viết hoàn toàn bằng **Go** — không phải là bản fork của OpenClaw, NanoBot hay bất kỳ dự án nào khác.

🦐 PicoClaw là trợ lý AI cá nhân siêu nhẹ, lấy cảm hứng từ [NanoBot](https://github.com/HKUDS/nanobot), được viết lại hoàn toàn bằng Go thông qua quá trình "tự khởi tạo" (self-bootstrapping) — nơi chính AI Agent đã tự dẫn dắt toàn bộ quá trình chuyển đổi kiến trúc và tối ưu hóa mã nguồn.

⚡️ Chạy trên phần cứng chỉ $10 với RAM <10MB: Tiết kiệm 99% bộ nhớ so với OpenClaw và rẻ hơn 98% so với Mac mini!

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
> **🚨 TUYÊN BỐ BẢO MẬT & KÊNH CHÍNH THỨC**
>
> * **KHÔNG CÓ CRYPTO:** PicoClaw **KHÔNG** có bất kỳ token/coin chính thức nào. Mọi thông tin trên `pump.fun` hoặc các sàn giao dịch khác đều là **LỪA ĐẢO**.
>
> * **DOMAIN CHÍNH THỨC:** Website chính thức **DUY NHẤT** là **[picoclaw.io](https://picoclaw.io)**, website công ty là **[sipeed.com](https://sipeed.com)**
> * **Cảnh báo:** Nhiều tên miền `.ai/.org/.com/.net/...` đã bị bên thứ ba đăng ký.
> * **Cảnh báo:** PicoClaw đang trong giai đoạn phát triển sớm và có thể còn các vấn đề bảo mật mạng chưa được giải quyết. Không nên triển khai lên môi trường production trước phiên bản v1.0.
> * **Lưu ý:** PicoClaw gần đây đã merge nhiều PR, dẫn đến bộ nhớ sử dụng có thể lớn hơn (10–20MB) ở các phiên bản mới nhất. Chúng tôi sẽ ưu tiên tối ưu tài nguyên khi bộ tính năng đã ổn định.

## 📢 Tin tức

2026-03-17 🚀 **v0.2.3 Phát hành!** Giao diện khay hệ thống (Windows & Linux), theo dõi trạng thái sub-agent (`spawn_status`), hot-reload gateway thử nghiệm, cổng bảo mật cron và 2 bản vá bảo mật. PicoClaw đạt **25K ⭐**!

2026-03-09 🎉 **v0.2.1 — Bản cập nhật lớn nhất!** Hỗ trợ giao thức MCP, 4 kênh mới (Matrix/IRC/WeCom/Discord Proxy), 3 nhà cung cấp mới (Kimi/Minimax/Avian), pipeline xử lý hình ảnh, bộ nhớ JSONL và định tuyến mô hình.

2026-02-28 📦 **v0.2.0** phát hành với hỗ trợ Docker Compose và launcher Web UI.

2026-02-26 🎉 PicoClaw đạt **20K stars** chỉ trong 17 ngày! Tự động điều phối kênh và giao diện năng lực đã được triển khai.

<details>
<summary>Tin tức cũ hơn...</summary>

2026-02-16 🎉 PicoClaw đạt 12K stars chỉ trong một tuần! Vai trò maintainer cộng đồng và [roadmap](ROADMAP.md) đã được công bố chính thức.

2026-02-13 🎉 PicoClaw đạt 5000 stars trong 4 ngày! Lộ trình dự án và Nhóm phát triển đang được thiết lập.

2026-02-09 🎉 **PicoClaw chính thức ra mắt!** Được xây dựng trong 1 ngày để mang AI Agent đến phần cứng $10 với RAM <10MB. 🦐 PicoClaw, Lên Đường!

</details>

## ✨ Tính năng nổi bật

🪶 **Siêu nhẹ**: Bộ nhớ sử dụng <10MB — nhỏ hơn 99% so với OpenClaw (chức năng cốt lõi).*

💰 **Chi phí tối thiểu**: Đủ hiệu quả để chạy trên phần cứng $10 — rẻ hơn 98% so với Mac mini.

⚡️ **Khởi động siêu nhanh**: Nhanh gấp 400 lần, khởi động trong <1 giây ngay cả trên CPU đơn nhân 0.6GHz.

🌍 **Di động thực sự**: Một file binary duy nhất chạy trên RISC-V, ARM, MIPS và x86. Một click là chạy!

🤖 **AI tự xây dựng**: Triển khai Go-native tự động — 95% mã nguồn cốt lõi được Agent tạo ra, với sự tinh chỉnh của con người.

🔌 **Hỗ trợ MCP**: Tích hợp [Model Context Protocol](https://modelcontextprotocol.io/) gốc — kết nối bất kỳ máy chủ MCP nào để mở rộng khả năng của agent.

👁️ **Pipeline Xử lý Hình ảnh**: Gửi hình ảnh và tệp trực tiếp cho agent — tự động mã hóa base64 cho các LLM đa phương thức.

🧠 **Định tuyến Thông minh**: Định tuyến mô hình dựa trên quy tắc — truy vấn đơn giản chuyển đến mô hình nhẹ, tiết kiệm chi phí API.

_*Các phiên bản gần đây có thể sử dụng 10–20MB do merge tính năng nhanh chóng. Tối ưu tài nguyên đang được lên kế hoạch. So sánh thời gian khởi động dựa trên benchmark đơn nhân 0.8GHz (xem bảng bên dưới)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Ngôn ngữ**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **Thời gian khởi động**</br>(CPU 0.8GHz) | >500s         | >30s                     | **<1s**                                   |
| **Chi phí**                   | Mac Mini $599 | Hầu hết SBC Linux ~$50  | **Mọi bo mạch Linux**</br>**Chỉ từ $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

> 📋 **[Danh Sách Tương Thích Phần Cứng](docs/hardware-compatibility.md)** — Xem tất cả các board đã được kiểm tra, từ RISC-V $5 đến Raspberry Pi và điện thoại Android. Board của bạn chưa có? Gửi PR!

## 🦾 Demo

### 🛠️ Quy trình trợ lý tiêu chuẩn

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Lập trình Full-Stack</p></th>
    <th><p align="center">🗂️ Quản lý Nhật ký & Kế hoạch</p></th>
    <th><p align="center">🔎 Tìm kiếm Web & Học hỏi</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Phát triển • Triển khai • Mở rộng</td>
    <td align="center">Lên lịch • Tự động hóa • Ghi nhớ</td>
    <td align="center">Khám phá • Phân tích • Xu hướng</td>
  </tr>
</table>

### 📱 Chạy trên điện thoại Android cũ

Hãy cho chiếc điện thoại cũ một cuộc sống mới! Biến nó thành trợ lý AI thông minh với PicoClaw. Bắt đầu nhanh:

1. **Cài đặt [Termux](https://github.com/termux/termux-app)** (Tải từ [GitHub Releases](https://github.com/termux/termux-app/releases), hoặc tìm trên F-Droid / Google Play).
2. **Chạy các lệnh**

```bash
# Tải phiên bản mới nhất từ https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot cung cấp bố cục hệ thống tệp Linux tiêu chuẩn
```

Sau đó làm theo hướng dẫn trong phần "Bắt đầu nhanh" để hoàn tất cấu hình!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Triển khai sáng tạo trên phần cứng tối thiểu

PicoClaw có thể triển khai trên hầu hết mọi thiết bị Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) phiên bản E(Ethernet) hoặc W(WiFi6), dùng làm Trợ lý Gia đình tối giản
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), hoặc $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) dùng cho quản trị Server tự động
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) hoặc $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) dùng cho Giám sát thông minh

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Nhiều hình thức triển khai hơn đang chờ bạn khám phá!

## 📦 Cài đặt

### Tải từ picoclaw.io (Khuyến nghị)

Truy cập **[picoclaw.io](https://picoclaw.io)** — trang web chính thức tự động phát hiện nền tảng của bạn và cung cấp tải xuống một cú nhấp. Không cần chọn kiến trúc thủ công.

### Tải binary đã biên dịch sẵn

Hoặc tải binary cho nền tảng của bạn từ trang [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Biên dịch từ mã nguồn (cho phát triển)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build (không cần cài đặt)
make build

# Build cho nhiều nền tảng
make build-all

# Build cho Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build và cài đặt
make install
```

**Raspberry Pi Zero 2 W:** Sử dụng binary phù hợp với hệ điều hành: Raspberry Pi OS 32-bit → `make build-linux-arm`; 64-bit → `make build-linux-arm64`. Hoặc chạy `make build-pi-zero` để build cả hai.

## 📚 Tài liệu

Để xem hướng dẫn chi tiết, tham khảo tài liệu bên dưới. README này chỉ bao gồm phần bắt đầu nhanh.

| Chủ đề | Mô tả |
|--------|-------|
| 🐳 [Docker & Bắt đầu nhanh](docs/vi/docker.md) | Thiết lập Docker Compose, chế độ Launcher/Agent, cấu hình Bắt đầu nhanh |
| 💬 [Ứng dụng Chat](docs/vi/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom và nhiều hơn |
| ⚙️ [Cấu hình](docs/vi/configuration.md) | Biến môi trường, cấu trúc workspace, nguồn skill, sandbox bảo mật, heartbeat |
| 🔌 [Nhà cung cấp & Mô hình](docs/vi/providers.md) | 20+ nhà cung cấp LLM, định tuyến mô hình, cấu hình model_list, kiến trúc nhà cung cấp |
| 🔄 [Spawn & Tác vụ bất đồng bộ](docs/vi/spawn-tasks.md) | Tác vụ nhanh, tác vụ dài với spawn, điều phối sub-agent bất đồng bộ |
| 🐛 [Xử lý sự cố](docs/vi/troubleshooting.md) | Các vấn đề thường gặp và giải pháp |
| 🔧 [Cấu hình Công cụ](docs/vi/tools_configuration.md) | Bật/tắt từng công cụ, chính sách thực thi |
| 📋 [Tương Thích Phần Cứng](docs/hardware-compatibility.md) | Các board đã kiểm tra, yêu cầu tối thiểu, cách thêm board |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Tham gia Mạng xã hội Agent

Kết nối PicoClaw với Mạng xã hội Agent chỉ bằng cách gửi một tin nhắn qua CLI hoặc bất kỳ ứng dụng Chat nào đã tích hợp.

**Đọc `https://clawdchat.ai/skill.md` và làm theo hướng dẫn để tham gia [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Tham chiếu CLI

| Lệnh                      | Mô tả                         |
| -------------------------- | ------------------------------ |
| `picoclaw onboard`         | Khởi tạo cấu hình & workspace |
| `picoclaw agent -m "..."`  | Trò chuyện với agent           |
| `picoclaw agent`           | Chế độ chat tương tác          |
| `picoclaw gateway`         | Khởi động gateway              |
| `picoclaw status`          | Hiển thị trạng thái            |
| `picoclaw version`         | Hiển thị thông tin phiên bản   |
| `picoclaw cron list`       | Liệt kê tất cả tác vụ định kỳ |
| `picoclaw cron add ...`    | Thêm tác vụ định kỳ           |
| `picoclaw cron disable`    | Tắt tác vụ định kỳ            |
| `picoclaw cron remove`     | Xóa tác vụ định kỳ            |
| `picoclaw skills list`     | Liệt kê các skill đã cài      |
| `picoclaw skills install`  | Cài đặt một skill              |
| `picoclaw migrate`         | Di chuyển dữ liệu từ phiên bản cũ |
| `picoclaw auth login`      | Xác thực với nhà cung cấp     |
| `picoclaw model`           | Xem hoặc chuyển đổi model mặc định |

### Tác vụ định kỳ / Nhắc nhở

PicoClaw hỗ trợ nhắc nhở theo lịch và tác vụ lặp lại thông qua công cụ `cron`:

* **Nhắc nhở một lần**: "Nhắc tôi sau 10 phút" → kích hoạt một lần sau 10 phút
* **Tác vụ lặp lại**: "Nhắc tôi mỗi 2 giờ" → kích hoạt mỗi 2 giờ
* **Biểu thức Cron**: "Nhắc tôi lúc 9 giờ sáng mỗi ngày" → sử dụng biểu thức cron

## 🤝 Đóng góp & Lộ trình

Chào đón mọi PR! Mã nguồn được thiết kế nhỏ gọn và dễ đọc. 🤗

Xem [Lộ trình Cộng đồng](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) đầy đủ.

Nhóm phát triển đang được xây dựng. Tham gia sau khi có PR đầu tiên được merge!

Nhóm người dùng:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

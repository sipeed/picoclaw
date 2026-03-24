<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Asisten AI Super Ringan berbasis Go</h1>

<h3>Perangkat Keras $10 · RAM 10MB · Boot ms · Let's Go, PicoClaw!</h3>
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

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [English](README.md) | **Bahasa Indonesia**

</div>

---

> **PicoClaw** adalah proyek open-source independen yang diinisiasi oleh [Sipeed](https://sipeed.com), ditulis sepenuhnya dalam **Go** — bukan fork dari OpenClaw, NanoBot, atau proyek lainnya.

**PicoClaw** adalah asisten AI pribadi yang super ringan, terinspirasi dari [NanoBot](https://github.com/HKUDS/nanobot). Dibangun ulang dari awal dalam **Go** melalui proses "self-bootstrapping" — AI Agent itu sendiri yang memandu migrasi arsitektur dan optimasi kode.

**Berjalan di perangkat keras $10 dengan RAM <10MB** — hemat 99% memori dibanding OpenClaw dan 98% lebih murah dari Mac mini!

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
> **Peringatan Keamanan**
>
> * **TANPA KRIPTO:** PicoClaw **tidak** menerbitkan token atau cryptocurrency resmi apa pun. Semua klaim di `pump.fun` atau platform trading lainnya adalah **penipuan**.
> * **DOMAIN RESMI:** Satu-satunya website resmi adalah **[picoclaw.io](https://picoclaw.io)**, dan website perusahaan adalah **[sipeed.com](https://sipeed.com)**
> * **WASPADA:** Banyak domain `.ai/.org/.com/.net/...` telah didaftarkan oleh pihak ketiga. Jangan percaya mereka.
> * **CATATAN:** PicoClaw masih dalam tahap pengembangan awal yang cepat. Mungkin ada masalah keamanan yang belum terselesaikan. Jangan deploy ke produksi sebelum v1.0.
> * **CATATAN:** PicoClaw baru-baru ini menggabungkan banyak PR. Build terbaru mungkin menggunakan RAM 10-20MB. Optimasi sumber daya direncanakan setelah fitur stabil.

## 📢 Berita

2026-03-17 🚀 **v0.2.3 Dirilis!** UI system tray (Windows & Linux), pelacakan status sub-agent (`spawn_status`), eksperimental Gateway hot-reload, gerbang keamanan Cron, dan 2 perbaikan keamanan. PicoClaw telah mencapai **25K Stars**!

2026-03-09 🎉 **v0.2.1 — Update terbesar sejauh ini!** Dukungan protokol MCP, 4 channel baru (Matrix/IRC/WeCom/Discord Proxy), 3 provider baru (Kimi/Minimax/Avian), pipeline vision, penyimpanan memori JSONL, routing model.

2026-02-28 📦 **v0.2.0** dirilis dengan dukungan Docker Compose dan Web UI Launcher.

2026-02-26 🎉 PicoClaw mencapai **20K Stars** hanya dalam 17 hari! Orkestrasi channel otomatis dan antarmuka kapabilitas kini aktif.

<details>
<summary>Berita sebelumnya...</summary>

2026-02-16 🎉 PicoClaw menembus 12K Stars dalam satu minggu! Peran maintainer komunitas dan [Roadmap](ROADMAP.md) resmi diluncurkan.

2026-02-13 🎉 PicoClaw menembus 5000 Stars dalam 4 hari! Roadmap proyek dan grup pengembang sedang dalam proses.

2026-02-09 🎉 **PicoClaw Diluncurkan!** Dibangun dalam 1 hari untuk menghadirkan AI Agent ke perangkat keras $10 dengan RAM <10MB. Let's Go, PicoClaw!

</details>

## ✨ Fitur

🪶 **Super Ringan**: Penggunaan memori inti <10MB — 99% lebih kecil dari OpenClaw.*

💰 **Biaya Minimal**: Cukup efisien untuk berjalan di perangkat keras $10 — 98% lebih murah dari Mac mini.

⚡️ **Boot Secepat Kilat**: Startup 400x lebih cepat. Boot dalam <1 detik bahkan di prosesor single-core 0,6GHz.

🌍 **Portabilitas Sejati**: Satu binary untuk RISC-V, ARM, MIPS, dan x86. Satu binary, jalan di mana saja!

🤖 **AI-Bootstrapped**: Implementasi Go native murni — 95% kode inti dihasilkan oleh Agent dengan penyempurnaan human-in-the-loop.

🔌 **Dukungan MCP**: Integrasi [Model Context Protocol](https://modelcontextprotocol.io/) native — hubungkan server MCP mana pun untuk memperluas kapabilitas Agent.

👁️ **Pipeline Vision**: Kirim gambar dan file langsung ke Agent — encoding base64 otomatis untuk LLM multimodal.

🧠 **Routing Cerdas**: Routing model berbasis aturan — kueri sederhana diarahkan ke model ringan, menghemat biaya API.

_*Build terbaru mungkin menggunakan 10-20MB karena penggabungan PR yang cepat. Optimasi sumber daya direncanakan. Perbandingan kecepatan boot berdasarkan benchmark single-core 0,8GHz (lihat tabel di bawah)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **Bahasa**                     | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **Waktu Boot**</br>(core 0,8GHz) | >500d       | >30d                     | **<1d**                                |
| **Biaya**                      | Mac Mini $599 | Kebanyakan board Linux ~$50 | **Board Linux mana pun**</br>**mulai $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[Daftar Kompatibilitas Hardware](docs/hardware-compatibility.md)** — Lihat semua board yang telah diuji, dari RISC-V $5 hingga Raspberry Pi hingga ponsel Android. Board Anda belum terdaftar? Kirim PR!

<p align="center">
<img src="assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

## 🦾 Demonstrasi

### 🛠️ Alur Kerja Asisten Standar

<table align="center">
<tr align="center">
<th><p align="center">Mode Full-Stack Engineer</p></th>
<th><p align="center">Pencatatan & Perencanaan</p></th>
<th><p align="center">Pencarian Web & Pembelajaran</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Develop · Deploy · Scale</td>
<td align="center">Jadwal · Otomasi · Ingat</td>
<td align="center">Temukan · Wawasan · Tren</td>
</tr>
</table>

### 🐜 Deploy Inovatif dengan Footprint Rendah

PicoClaw dapat di-deploy di hampir semua perangkat Linux!

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versi E(Ethernet) atau W(WiFi6), untuk home assistant minimal
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), atau $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), untuk operasi server otomatis
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) atau $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), untuk pengawasan cerdas

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Lebih Banyak Kasus Deploy Menanti!

## 📦 Instalasi

### Unduh dari picoclaw.io (Direkomendasikan)

Kunjungi **[picoclaw.io](https://picoclaw.io)** — website resmi mendeteksi platform Anda secara otomatis dan menyediakan unduhan satu klik. Tidak perlu memilih arsitektur secara manual.

### Unduh binary yang sudah dikompilasi

Atau, unduh binary untuk platform Anda dari halaman [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Build dari source (untuk pengembangan)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build binary inti
make build

# Build Web UI Launcher (diperlukan untuk mode WebUI)
make build-launcher

# Build untuk berbagai platform
make build-all

# Build untuk Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build dan instal
make install
```

**Raspberry Pi Zero 2 W:** Gunakan binary yang sesuai dengan OS Anda: Raspberry Pi OS 32-bit -> `make build-linux-arm`; 64-bit -> `make build-linux-arm64`. Atau jalankan `make build-pi-zero` untuk build keduanya.

## 🚀 Panduan Memulai Cepat

### 🌐 WebUI Launcher (Direkomendasikan untuk Desktop)

WebUI Launcher menyediakan antarmuka berbasis browser untuk konfigurasi dan chat. Ini adalah cara termudah untuk memulai — tidak perlu pengetahuan command-line.

**Opsi 1: Klik dua kali (Desktop)**

Setelah mengunduh dari [picoclaw.io](https://picoclaw.io), klik dua kali `picoclaw-launcher` (atau `picoclaw-launcher.exe` di Windows). Browser Anda akan terbuka otomatis di `http://localhost:18800`.

**Opsi 2: Command line**

```bash
picoclaw-launcher
# Buka http://localhost:18800 di browser Anda
```

> [!TIP]
> **Akses jarak jauh / Docker / VM:** Tambahkan flag `-public` untuk mendengarkan di semua antarmuka:
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Memulai:**

Buka WebUI, lalu: **1)** Konfigurasi Provider (tambahkan API key LLM Anda) -> **2)** Konfigurasi Channel (mis. Telegram) -> **3)** Mulai Gateway -> **4)** Chat!

Untuk dokumentasi WebUI lengkap, lihat [docs.picoclaw.io](https://docs.picoclaw.io).

<details>
<summary><b>Docker (alternatif)</b></summary>

```bash
# 1. Clone repo ini
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Jalankan pertama kali — otomatis membuat docker/data/config.json lalu keluar
#    (hanya terpicu ketika config.json dan workspace/ keduanya tidak ada)
docker compose -f docker/docker-compose.yml --profile launcher up
# Container mencetak "First-run setup complete." dan berhenti.

# 3. Atur API key Anda
vim docker/data/config.json

# 4. Mulai
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Buka http://localhost:18800
```

> **Pengguna Docker / VM:** Gateway mendengarkan di `127.0.0.1` secara default. Atur `PICOCLAW_GATEWAY_HOST=0.0.0.0` atau gunakan flag `-public` agar dapat diakses dari host.

```bash
# Cek log
docker compose -f docker/docker-compose.yml logs -f

# Hentikan
docker compose -f docker/docker-compose.yml --profile launcher down

# Update
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

### 💻 TUI Launcher (Direkomendasikan untuk Headless / SSH)

TUI (Terminal UI) Launcher menyediakan antarmuka terminal lengkap untuk konfigurasi dan manajemen. Ideal untuk server, Raspberry Pi, dan lingkungan headless lainnya.

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**Memulai:**

Gunakan menu TUI untuk: **1)** Konfigurasi Provider -> **2)** Konfigurasi Channel -> **3)** Mulai Gateway -> **4)** Chat!

Untuk dokumentasi TUI lengkap, lihat [docs.picoclaw.io](https://docs.picoclaw.io).

### 📱 Android

Berikan kehidupan kedua untuk ponsel lama Anda! Ubah menjadi Asisten AI pintar dengan PicoClaw.

**Opsi 1: Termux (tersedia sekarang)**

1. Instal [Termux](https://github.com/termux/termux-app) (unduh dari [GitHub Releases](https://github.com/termux/termux-app/releases), atau cari di F-Droid / Google Play)
2. Jalankan perintah berikut:

```bash
# Unduh rilis terbaru
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot menyediakan tata letak filesystem Linux standar
```

Kemudian ikuti bagian Terminal Launcher di bawah untuk menyelesaikan konfigurasi.

<img src="assets/termux.jpg" alt="PicoClaw on Termux" width="512">

**Opsi 2: Instal APK (segera hadir)**

APK Android mandiri dengan WebUI bawaan sedang dalam pengembangan. Pantau terus!

<details>
<summary><b>Terminal Launcher (untuk lingkungan dengan sumber daya terbatas)</b></summary>

Untuk lingkungan minimal di mana hanya binary inti `picoclaw` yang tersedia (tanpa Launcher UI), Anda dapat mengonfigurasi semuanya melalui command line dan file konfigurasi JSON.

**1. Inisialisasi**

```bash
picoclaw onboard
```

Ini membuat `~/.picoclaw/config.json` dan direktori workspace.

**2. Konfigurasi** (`~/.picoclaw/config.json`)

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

> Lihat `config/config.example.json` di repo untuk template konfigurasi lengkap dengan semua opsi yang tersedia.

**3. Chat**

```bash
# Pertanyaan satu kali
picoclaw agent -m "What is 2+2?"

# Mode interaktif
picoclaw agent

# Mulai gateway untuk integrasi aplikasi chat
picoclaw gateway
```

</details>

## 🔌 Providers (LLM)

PicoClaw mendukung 30+ provider LLM melalui konfigurasi `model_list`. Gunakan format `protocol/model`:

| Provider | Protocol | API Key | Catatan |
|----------|----------|---------|---------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Diperlukan | GPT-5.4, GPT-4o, o3, dll. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Diperlukan | Claude Opus 4.6, Sonnet 4.6, dll. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Diperlukan | Gemini 3 Flash, 2.5 Pro, dll. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Diperlukan | 200+ model, API terpadu |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Diperlukan | GLM-4.7, GLM-5, dll. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Diperlukan | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Diperlukan | Doubao, model Ark |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Diperlukan | Qwen3, Qwen-Max, dll. |
| [Groq](https://console.groq.com/keys) | `groq/` | Diperlukan | Inferensi cepat (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Diperlukan | Model Kimi |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Diperlukan | Model MiniMax |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Diperlukan | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Diperlukan | Model yang di-host NVIDIA |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Diperlukan | Inferensi cepat |
| [Novita AI](https://novita.ai/) | `novita/` | Diperlukan | Berbagai model open |
| [Ollama](https://ollama.com/) | `ollama/` | Tidak perlu | Model lokal, self-hosted |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | Tidak perlu | Deploy lokal, kompatibel OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Bervariasi | Proxy untuk 100+ provider |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Diperlukan | Deploy Azure enterprise |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Login dengan device code |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |

<details>
<summary><b>Deploy lokal (Ollama, vLLM, dll.)</b></summary>

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

Untuk detail konfigurasi provider lengkap, lihat [Providers & Models](docs/providers.md).

</details>

## 💬 Channels (Aplikasi Chat)

Bicara dengan PicoClaw Anda melalui 17+ platform pesan:

| Channel | Pengaturan | Protocol | Dokumentasi |
|---------|------------|----------|-------------|
| **Telegram** | Mudah (bot token) | Long polling | [Panduan](docs/channels/telegram/README.md) |
| **Discord** | Mudah (bot token + intents) | WebSocket | [Panduan](docs/channels/discord/README.md) |
| **WhatsApp** | Mudah (scan QR atau bridge URL) | Native / Bridge | [Panduan](docs/chat-apps.md#whatsapp) |
| **Weixin** | Mudah (scan QR native) | iLink API | [Panduan](docs/chat-apps.md#weixin) |
| **QQ** | Mudah (AppID + AppSecret) | WebSocket | [Panduan](docs/channels/qq/README.md) |
| **Slack** | Mudah (bot + app token) | Socket Mode | [Panduan](docs/channels/slack/README.md) |
| **Matrix** | Sedang (homeserver + token) | Sync API | [Panduan](docs/channels/matrix/README.md) |
| **DingTalk** | Sedang (client credentials) | Stream | [Panduan](docs/channels/dingtalk/README.md) |
| **Feishu / Lark** | Sedang (App ID + Secret) | WebSocket/SDK | [Panduan](docs/channels/feishu/README.md) |
| **LINE** | Sedang (credentials + webhook) | Webhook | [Panduan](docs/channels/line/README.md) |
| **WeCom Bot** | Sedang (webhook URL) | Webhook | [Panduan](docs/channels/wecom/wecom_bot/README.md) |
| **WeCom App** | Sedang (corp credentials) | Webhook | [Panduan](docs/channels/wecom/wecom_app/README.md) |
| **WeCom AI Bot** | Sedang (token + AES key) | WebSocket / Webhook | [Panduan](docs/channels/wecom/wecom_aibot/README.md) |
| **IRC** | Sedang (server + nick) | IRC protocol | [Panduan](docs/chat-apps.md#irc) |
| **OneBot** | Sedang (WebSocket URL) | OneBot v11 | [Panduan](docs/channels/onebot/README.md) |
| **MaixCam** | Mudah (aktifkan) | TCP socket | [Panduan](docs/channels/maixcam/README.md) |
| **Pico** | Mudah (aktifkan) | Native protocol | Bawaan |
| **Pico Client** | Mudah (WebSocket URL) | WebSocket | Bawaan |

> Semua channel berbasis webhook berbagi satu server HTTP Gateway (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`). Feishu menggunakan mode WebSocket/SDK dan tidak menggunakan server HTTP bersama.

Untuk instruksi pengaturan channel lengkap, lihat [Konfigurasi Aplikasi Chat](docs/chat-apps.md).

## 🔧 Tools

### 🔍 Pencarian Web

PicoClaw dapat mencari web untuk memberikan informasi terkini. Konfigurasi di `tools.web`:

| Mesin Pencari | API Key | Tier Gratis | Tautan |
|--------------|---------|-------------|--------|
| DuckDuckGo | Tidak perlu | Tidak terbatas | Fallback bawaan |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Diperlukan | 1000 kueri/hari | Bertenaga AI, dioptimalkan untuk bahasa Mandarin |
| [Tavily](https://tavily.com) | Diperlukan | 1000 kueri/bulan | Dioptimalkan untuk AI Agent |
| [Brave Search](https://brave.com/search/api) | Diperlukan | 2000 kueri/bulan | Cepat dan privat |
| [Perplexity](https://www.perplexity.ai) | Diperlukan | Berbayar | Pencarian bertenaga AI |
| [SearXNG](https://github.com/searxng/searxng) | Tidak perlu | Self-hosted | Mesin metasearch gratis |
| [GLM Search](https://open.bigmodel.cn/) | Diperlukan | Bervariasi | Pencarian web Zhipu |

### ⚙️ Tools Lainnya

PicoClaw menyertakan tools bawaan untuk operasi file, eksekusi kode, penjadwalan, dan lainnya. Lihat [Konfigurasi Tools](docs/tools_configuration.md) untuk detail.

## 🎯 Skills

Skills adalah kapabilitas modular yang memperluas Agent Anda. Dimuat dari file `SKILL.md` di workspace Anda.

**Instal skills dari ClawHub:**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**Konfigurasi token ClawHub** (opsional, untuk rate limit lebih tinggi):

Tambahkan ke `config.json` Anda:
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

Untuk detail lebih lanjut, lihat [Konfigurasi Tools - Skills](docs/tools_configuration.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

PicoClaw mendukung [MCP](https://modelcontextprotocol.io/) secara native — hubungkan server MCP mana pun untuk memperluas kapabilitas Agent Anda dengan tools dan sumber data eksternal.

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

Untuk konfigurasi MCP lengkap (transport stdio, SSE, HTTP, Tool Discovery), lihat [Konfigurasi Tools - MCP](docs/tools_configuration.md#mcp-tool).

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Bergabung dengan Jaringan Sosial Agent

Hubungkan PicoClaw ke Jaringan Sosial Agent hanya dengan mengirim satu pesan melalui CLI atau Aplikasi Chat terintegrasi mana pun.

**Baca `https://clawdchat.ai/skill.md` dan ikuti instruksi untuk bergabung dengan [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Referensi CLI

| Perintah                   | Deskripsi                        |
| -------------------------- | -------------------------------- |
| `picoclaw onboard`         | Inisialisasi konfigurasi & workspace |
| `picoclaw onboard weixin` | Hubungkan akun WeChat via QR |
| `picoclaw agent -m "..."` | Chat dengan agent                |
| `picoclaw agent`           | Mode chat interaktif             |
| `picoclaw gateway`         | Mulai gateway                    |
| `picoclaw status`          | Tampilkan status                 |
| `picoclaw version`         | Tampilkan info versi             |
| `picoclaw model`           | Lihat atau ganti model default   |
| `picoclaw cron list`       | Daftar semua tugas terjadwal     |
| `picoclaw cron add ...`    | Tambah tugas terjadwal           |
| `picoclaw cron disable`    | Nonaktifkan tugas terjadwal      |
| `picoclaw cron remove`     | Hapus tugas terjadwal            |
| `picoclaw skills list`     | Daftar skill yang terinstal      |
| `picoclaw skills install`  | Instal skill                     |
| `picoclaw migrate`         | Migrasi data dari versi lama     |
| `picoclaw auth login`      | Autentikasi dengan provider      |

### ⏰ Tugas Terjadwal / Pengingat

PicoClaw mendukung pengingat terjadwal dan tugas berulang melalui tool `cron`:

* **Pengingat satu kali**: "Ingatkan saya dalam 10 menit" -> terpicu sekali setelah 10 menit
* **Tugas berulang**: "Ingatkan saya setiap 2 jam" -> terpicu setiap 2 jam
* **Ekspresi cron**: "Ingatkan saya jam 9 pagi setiap hari" -> menggunakan ekspresi cron

## 📚 Dokumentasi

Untuk panduan lengkap di luar README ini:

| Topik | Deskripsi |
|-------|-----------|
| [Docker & Panduan Cepat](docs/docker.md) | Pengaturan Docker Compose, mode Launcher/Agent |
| [Aplikasi Chat](docs/chat-apps.md) | Semua 17+ panduan pengaturan channel |
| [Konfigurasi](docs/configuration.md) | Variabel environment, tata letak workspace, sandbox keamanan |
| [Providers & Models](docs/providers.md) | 30+ provider LLM, routing model, konfigurasi model_list |
| [Spawn & Tugas Async](docs/spawn-tasks.md) | Tugas cepat, tugas panjang dengan spawn, orkestrasi sub-agent async |
| [Hooks](docs/hooks/README.md) | Sistem hook berbasis event: observer, interceptor, approval hook |
| [Steering](docs/steering.md) | Menyuntikkan pesan ke dalam loop agent yang sedang berjalan |
| [SubTurn](docs/subturn.md) | Koordinasi subagent, kontrol konkurensi, siklus hidup |
| [Pemecahan Masalah](docs/troubleshooting.md) | Masalah umum dan solusinya |
| [Konfigurasi Tools](docs/tools_configuration.md) | Aktifkan/nonaktifkan per-tool, kebijakan exec, MCP, Skills |
| [Kompatibilitas Hardware](docs/hardware-compatibility.md) | Board yang telah diuji, persyaratan minimum |

## 🤝 Kontribusi & Roadmap

PR sangat diterima! Codebase sengaja dibuat kecil dan mudah dibaca.

Lihat [Roadmap Komunitas](https://github.com/sipeed/picoclaw/issues/988) dan [CONTRIBUTING.md](CONTRIBUTING.md) untuk panduan.

Grup pengembang sedang dibangun, bergabunglah setelah PR pertama Anda di-merge!

Grup Pengguna:

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="Kode QR grup WeChat" width="512">


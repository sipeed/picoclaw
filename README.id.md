<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Asisten AI Super Ringan berbasis Go</h1>

  <h3>Perangkat Keras $10 · RAM <10MB · Boot <1 Detik · Ayo, Berangkat!</h3>
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

> **PicoClaw** adalah proyek open-source independen yang diinisiasi oleh [Sipeed](https://sipeed.com). Ditulis sepenuhnya dalam **Go** — bukan fork dari OpenClaw, NanoBot, atau proyek lainnya.

🦐 PicoClaw adalah asisten AI pribadi yang super ringan, terinspirasi dari [NanoBot](https://github.com/HKUDS/nanobot), ditulis ulang sepenuhnya dalam Go melalui proses "self-bootstrapping" — di mana AI Agent itu sendiri yang memandu seluruh migrasi arsitektur dan optimasi kode.

⚡️ Berjalan di perangkat keras $10 dengan RAM <10MB: Hemat 99% memori dibanding OpenClaw dan 98% lebih murah dibanding Mac mini!

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
> **🚨 KEAMANAN & SALURAN RESMI**
>
> * **TANPA KRIPTO:** PicoClaw **TIDAK** memiliki token/koin resmi. Semua klaim di `pump.fun` atau platform trading lainnya adalah **PENIPUAN**.
>
> * **DOMAIN RESMI:** Satu-satunya website resmi adalah **[picoclaw.io](https://picoclaw.io)**, dan website perusahaan adalah **[sipeed.com](https://sipeed.com)**
> * **Peringatan:** Banyak domain `.ai/.org/.com/.net/...` yang didaftarkan oleh pihak ketiga.
> * **Peringatan:** PicoClaw masih dalam tahap pengembangan awal dan mungkin memiliki masalah keamanan jaringan yang belum teratasi. Jangan deploy ke lingkungan produksi sebelum rilis v1.0.
> * **Catatan:** PicoClaw baru-baru ini menggabungkan banyak PR, yang mungkin mengakibatkan penggunaan memori lebih besar (10–20MB) pada versi terbaru. Kami berencana untuk memprioritaskan optimasi sumber daya segera setelah fitur saat ini mencapai kondisi stabil.

## 📢 Berita

2026-03-17 🚀 **v0.2.3 Dirilis!** UI system tray (Windows & Linux), pelacakan status sub-agent (`spawn_status`), eksperimental gateway hot-reload, gerbang keamanan cron, dan 2 perbaikan keamanan. PicoClaw kini di **25K ⭐**!

2026-03-09 🎉 **v0.2.1 — Update terbesar!** Dukungan protokol MCP, 4 channel baru (Matrix/IRC/WeCom/Discord Proxy), 3 provider baru (Kimi/Minimax/Avian), pipeline vision, penyimpanan memori JSONL, dan routing model.

2026-02-28 📦 **v0.2.0** dirilis dengan dukungan Docker Compose dan launcher Web UI.

2026-02-26 🎉 PicoClaw mencapai **20K bintang** hanya dalam 17 hari! Orkestrasi channel otomatis dan antarmuka kapabilitas diluncurkan.

<details>
<summary>Berita lama...</summary>

2026-02-16 🎉 PicoClaw mencapai 12K bintang dalam satu minggu! Peran maintainer komunitas dan [roadmap](ROADMAP.md) resmi diposting.

2026-02-13 🎉 PicoClaw mencapai 5000 bintang dalam 4 hari! Roadmap Proyek dan pengaturan Grup Pengembang sedang berjalan.

2026-02-09 🎉 **PicoClaw Diluncurkan!** Dibangun dalam 1 hari untuk menghadirkan AI Agent ke perangkat keras $10 dengan RAM <10MB. 🦐 PicoClaw, Ayo Berangkat!

</details>

## ✨ Fitur

🪶 **Super Ringan**: Penggunaan memori <10MB — 99% lebih kecil dari fungsionalitas inti OpenClaw.*

💰 **Biaya Minimal**: Cukup efisien untuk berjalan di perangkat keras $10 — 98% lebih murah dari Mac mini.

⚡️ **Secepat Kilat**: Waktu startup 400X lebih cepat, boot dalam <1 detik bahkan di prosesor single core 0,6GHz.

🌍 **Portabilitas Sejati**: Satu binary mandiri untuk RISC-V, ARM, MIPS, dan x86, Satu Klik Langsung Jalan!

🤖 **AI-Bootstrapped**: Implementasi Go-native secara otonom — 95% kode inti dihasilkan oleh Agent dengan penyempurnaan human-in-the-loop.

🔌 **Dukungan MCP**: Integrasi [Model Context Protocol](https://modelcontextprotocol.io/) native — hubungkan server MCP mana pun untuk memperluas kapabilitas agent.

👁️ **Pipeline Vision**: Kirim gambar dan file langsung ke agent — encoding base64 otomatis untuk LLM multimodal.

🧠 **Routing Cerdas**: Routing model berbasis aturan — kueri sederhana diarahkan ke model ringan, menghemat biaya API.

_*Versi terbaru mungkin menggunakan 10–20MB karena penggabungan fitur yang cepat. Optimasi sumber daya direncanakan. Perbandingan startup berdasarkan benchmark prosesor single-core 0,8GHz (lihat tabel di bawah)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Bahasa**                    | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **Startup**</br>(0,8GHz core) | >500d         | >30d                     | **<1d**                                   |
| **Biaya**                     | Mac Mini $599 | Kebanyakan Linux SBC </br>~$50 | **Semua Board Linux**</br>**Mulai dari $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demonstrasi

### 🛠️ Alur Kerja Asisten Standar

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Full-Stack Engineer</p></th>
    <th><p align="center">🗂️ Pencatatan & Manajemen Perencanaan</p></th>
    <th><p align="center">🔎 Pencarian Web & Pembelajaran</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Develop • Deploy • Scale</td>
    <td align="center">Jadwal • Otomasi • Memori</td>
    <td align="center">Penemuan • Wawasan • Tren</td>
  </tr>
</table>

### 📱 Jalankan di HP Android Lama

Berikan kehidupan kedua untuk HP lama Anda! Ubah menjadi Asisten AI pintar dengan PicoClaw. Panduan Cepat:

1. **Instal [Termux](https://github.com/termux/termux-app)** (Unduh dari [GitHub Releases](https://github.com/termux/termux-app/releases), atau cari di F-Droid / Google Play).
2. **Jalankan perintah**

```bash
# Unduh rilis terbaru dari https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard
```

Kemudian ikuti instruksi di bagian "Panduan Cepat" untuk menyelesaikan konfigurasi!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Deploy Inovatif dengan Footprint Rendah

PicoClaw dapat di-deploy di hampir semua perangkat Linux!

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versi E(Ethernet) atau W(WiFi6), untuk Home Assistant Minimal
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), atau $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) untuk Pemeliharaan Server Otomatis
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) atau $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) untuk Pemantauan Cerdas

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Lebih Banyak Kasus Deploy Menanti!

## 📦 Instalasi

### Instal dengan binary yang sudah dikompilasi

Unduh binary untuk platform Anda dari halaman [Releases](https://github.com/sipeed/picoclaw/releases).

### Instal dari source (fitur terbaru, disarankan untuk pengembangan)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, tidak perlu instal
make build

# Build untuk berbagai platform
make build-all

# Build untuk Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build dan Instal
make install
```

**Raspberry Pi Zero 2 W:** Gunakan binary yang sesuai dengan OS Anda: Raspberry Pi OS 32-bit → `make build-linux-arm`; 64-bit → `make build-linux-arm64`. Atau jalankan `make build-pi-zero` untuk build keduanya.

## 📚 Dokumentasi

Untuk panduan lengkap, lihat dokumen di bawah. README ini hanya berisi panduan cepat.

| Topik | Deskripsi |
|-------|-----------|
| 🐳 [Docker & Panduan Cepat](docs/docker.md) | Pengaturan Docker Compose, mode Launcher/Agent, konfigurasi Panduan Cepat |
| 💬 [Aplikasi Chat](docs/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom, dan lainnya |
| ⚙️ [Konfigurasi](docs/configuration.md) | Variabel environment, tata letak workspace, sumber skill, sandbox keamanan, heartbeat |
| 🔌 [Provider & Model](docs/providers.md) | 20+ provider LLM, routing model, konfigurasi model_list, arsitektur provider |
| 🔄 [Spawn & Tugas Async](docs/spawn-tasks.md) | Tugas cepat, tugas panjang dengan spawn, orkestrasi sub-agent async |
| 🐛 [Pemecahan Masalah](docs/troubleshooting.md) | Masalah umum dan solusinya |
| 🔧 [Konfigurasi Tools](docs/tools_configuration.md) | Aktifkan/nonaktifkan tool, kebijakan exec |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Bergabung dengan Jaringan Sosial Agent

Hubungkan Picoclaw ke Jaringan Sosial Agent hanya dengan mengirim satu pesan melalui CLI atau Aplikasi Chat terintegrasi.

**Baca `https://clawdchat.ai/skill.md` dan ikuti instruksi untuk bergabung dengan [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Referensi CLI

| Perintah                  | Deskripsi                        |
| ------------------------- | -------------------------------- |
| `picoclaw onboard`        | Inisialisasi konfigurasi & workspace |
| `picoclaw agent -m "..."` | Chat dengan agent                |
| `picoclaw agent`          | Mode chat interaktif             |
| `picoclaw gateway`        | Mulai gateway                    |
| `picoclaw status`         | Tampilkan status                 |
| `picoclaw version`        | Tampilkan info versi             |
| `picoclaw cron list`      | Daftar semua tugas terjadwal     |
| `picoclaw cron add ...`   | Tambah tugas terjadwal           |
| `picoclaw cron disable`   | Nonaktifkan tugas terjadwal      |
| `picoclaw cron remove`    | Hapus tugas terjadwal            |
| `picoclaw skills list`    | Daftar skill yang terinstal      |
| `picoclaw skills install` | Instal skill                     |
| `picoclaw migrate`        | Migrasi data dari versi lama     |
| `picoclaw auth login`     | Autentikasi dengan provider      |

### Tugas Terjadwal / Pengingat

PicoClaw mendukung pengingat terjadwal dan tugas berulang melalui tool `cron`:

* **Pengingat satu kali**: "Ingatkan saya dalam 10 menit" → terpicu sekali setelah 10 menit
* **Tugas berulang**: "Ingatkan saya setiap 2 jam" → terpicu setiap 2 jam
* **Ekspresi cron**: "Ingatkan saya jam 9 pagi setiap hari" → menggunakan ekspresi cron

## 🤝 Kontribusi & Roadmap

PR sangat diterima! Codebase sengaja dibuat kecil dan mudah dibaca. 🤗

Lihat [Roadmap Komunitas](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) lengkap kami.

Grup pengembang sedang dibangun, bergabunglah setelah PR pertama Anda di-merge!

Grup Pengguna:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

# 💬 Konfigurasi Aplikasi Sembang

> Kembali ke [README](../../README.my.md)

## 💬 Aplikasi Sembang

Berbual dengan picoclaw anda melalui Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot, MaixCam, atau Pico (protokol asli)

> **Nota**: Semua saluran berasaskan webhook (LINE, WeCom, dan sebagainya) diservis pada satu pelayan HTTP Gateway yang dikongsi (`gateway.host`:`gateway.port`, lalai `127.0.0.1:18790`). Tiada port khusus per saluran untuk dikonfigurasikan. Nota: Feishu menggunakan mod WebSocket/SDK dan tidak menggunakan pelayan HTTP webhook yang dikongsi.

| Saluran          | Penyediaan                                 |
| ---------------- | ------------------------------------------ |
| **Telegram**     | Mudah (hanya token)                        |
| **Discord**      | Mudah (token bot + intents)                |
| **WhatsApp**     | Mudah (asli: imbas QR; atau bridge URL)    |
| **Matrix**       | Sederhana (homeserver + access token bot)  |
| **QQ**           | Mudah (AppID + AppSecret)                  |
| **DingTalk**     | Sederhana (kelayakan aplikasi)             |
| **LINE**         | Sederhana (kelayakan + webhook URL)        |
| **WeCom AI Bot** | Sederhana (Token + kunci AES)              |
| **Feishu**       | Sederhana (App ID + Secret, mod WebSocket) |
| **Slack**        | Sederhana (Bot token + App token)          |
| **IRC**          | Sederhana (pelayan + konfigurasi TLS)      |
| **OneBot**       | Sederhana (QQ melalui protokol OneBot)     |
| **MaixCam**      | Mudah (integrasi perkakasan Sipeed)        |
| **Pico**         | Protokol PicoClaw asli                     |

<details>
<summary><b>Telegram</b> (Disyorkan)</summary>

**1. Cipta bot**

* Buka Telegram, cari `@BotFather`
* Hantar `/newbot`, ikut arahan
* Salin token

**2. Konfigurasi**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": false,
    }
  }
}
```

> Dapatkan user ID anda daripada `@userinfobot` di Telegram.

**3. Jalankan**

```bash
picoclaw gateway
```

**4. Menu arahan Telegram (auto-register semasa startup)**

PicoClaw kini menyimpan definisi arahan dalam satu registry bersama. Semasa startup, Telegram akan mendaftarkan arahan bot yang disokong secara automatik (contohnya `/start`, `/help`, `/show`, `/list`) supaya menu arahan dan tingkah laku runtime sentiasa selari.
Pendaftaran menu arahan Telegram kekal sebagai UX penemuan setempat saluran; pelaksanaan arahan generik dikendalikan secara berpusat dalam gelung agen melalui commands executor.

Jika pendaftaran arahan gagal (ralat sementara rangkaian/API), saluran tetap akan bermula dan PicoClaw akan mencuba semula pendaftaran di latar belakang.

**4. Pemformatan Lanjutan**
Anda boleh menetapkan `use_markdown_v2: true` untuk mengaktifkan pilihan pemformatan yang lebih maju. Ini membolehkan bot menggunakan keseluruhan set ciri Telegram MarkdownV2, termasuk gaya bersarang, spoiler, dan blok lebar tetap tersuai.

</details>

<details>
<summary><b>Discord</b></summary>

**1. Cipta bot**

* Pergi ke <https://discord.com/developers/applications>
* Cipta aplikasi → Bot → Add Bot
* Salin token bot

**2. Aktifkan intents**

* Dalam tetapan Bot, aktifkan **MESSAGE CONTENT INTENT**
* (Pilihan) Aktifkan **SERVER MEMBERS INTENT** jika anda bercadang menggunakan allow list berasaskan data ahli

**3. Dapatkan User ID anda**
* Discord Settings → Advanced → aktifkan **Developer Mode**
* Klik kanan avatar anda → **Copy User ID**

**4. Konfigurasi**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Jemput bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Buka URL jemputan yang dijana dan tambahkan bot ke pelayan anda

**Pilihan: Mod trigger kumpulan**

Secara lalai bot membalas semua mesej dalam saluran pelayan. Untuk mengehadkan balasan kepada @mention sahaja, tambah:

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Anda juga boleh mencetuskan dengan awalan kata kunci (contohnya `!bot`):

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Jalankan**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b> (asli melalui whatsmeow)</summary>

PicoClaw boleh menyambung ke WhatsApp dalam dua cara:

- **Asli (disyorkan):** Dalam proses menggunakan [whatsmeow](https://github.com/tulir/whatsmeow). Tiada bridge berasingan. Tetapkan `"use_native": true` dan biarkan `bridge_url` kosong. Pada larian pertama, imbas kod QR dengan WhatsApp (Linked Devices). Sesi disimpan di bawah workspace anda (contohnya `workspace/whatsapp/`). Saluran asli ini adalah **pilihan** untuk memastikan binari lalai kekal kecil; bina dengan `-tags whatsapp_native` (contohnya `make build-whatsapp-native` atau `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Sambung ke bridge WebSocket luaran. Tetapkan `bridge_url` (contohnya `ws://localhost:3001`) dan biarkan `use_native` sebagai false.

**Konfigurasi (asli)**

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

Jika `session_store_path` kosong, sesi akan disimpan dalam `<workspace>/whatsapp/`. Jalankan `picoclaw gateway`; pada larian pertama, imbas kod QR yang dipaparkan dalam terminal menggunakan WhatsApp → Linked Devices.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Cipta bot**

- Pergi ke [QQ Open Platform](https://q.qq.com/#)
- Cipta aplikasi → Dapatkan **AppID** dan **AppSecret**

**2. Konfigurasi**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

> Tetapkan `allow_from` kepada kosong untuk membenarkan semua pengguna, atau nyatakan nombor QQ untuk mengehadkan akses.

**3. Jalankan**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Cipta bot**

* Pergi ke [Open Platform](https://open.dingtalk.com/)
* Cipta aplikasi dalaman
* Salin Client ID dan Client Secret

**2. Konfigurasi**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Tetapkan `allow_from` kepada kosong untuk membenarkan semua pengguna, atau nyatakan user ID DingTalk untuk mengehadkan akses.

**3. Jalankan**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Sediakan akaun bot**

* Gunakan homeserver pilihan anda (contohnya `https://matrix.org` atau self-hosted)
* Cipta pengguna bot dan dapatkan access tokennya

**2. Konfigurasi**

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. Jalankan**

```bash
picoclaw gateway
```

Untuk pilihan penuh (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), lihat [Panduan Konfigurasi Saluran Matrix](docs/channels/matrix/README.md).

</details>

<details>
<summary><b>LINE</b></summary>

**1. Cipta Akaun Rasmi LINE**

- Pergi ke [LINE Developers Console](https://developers.line.biz/)
- Cipta provider → Cipta saluran Messaging API
- Salin **Channel Secret** dan **Channel Access Token**

**2. Konfigurasi**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

> Webhook LINE diservis pada pelayan Gateway yang dikongsi (`gateway.host`:`gateway.port`, lalai `127.0.0.1:18790`).

**3. Tetapkan Webhook URL**

LINE memerlukan HTTPS untuk webhook. Gunakan reverse proxy atau tunnel:

```bash
# Contoh dengan ngrok (port lalai gateway ialah 18790)
ngrok http 18790
```

Kemudian tetapkan Webhook URL dalam LINE Developers Console kepada `https://your-domain/webhook/line` dan aktifkan **Use webhook**.

**4. Jalankan**

```bash
picoclaw gateway
```

> Dalam sembang kumpulan, bot hanya membalas apabila @disebut. Balasan akan memetik mesej asal.

</details>

<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw menyokong tiga jenis integrasi WeCom:

**Pilihan 1: WeCom Bot (Bot)** - Penyediaan lebih mudah, menyokong sembang kumpulan
**Pilihan 2: WeCom App (Custom App)** - Lebih banyak ciri, pemesejan proaktif, sembang peribadi sahaja
**Pilihan 3: WeCom AI Bot (AI Bot)** - AI Bot rasmi, balasan streaming, menyokong sembang kumpulan & peribadi

Lihat [Panduan Konfigurasi WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) untuk arahan penyediaan terperinci.

**Quick Setup - WeCom Bot:**

**1. Cipta bot**

* Pergi ke WeCom Admin Console → Group Chat → Add Group Bot
* Salin webhook URL (format: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Konfigurasi**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

> Webhook WeCom diservis pada pelayan Gateway yang dikongsi (`gateway.host`:`gateway.port`, lalai `127.0.0.1:18790`).

**Quick Setup - WeCom App:**

**1. Cipta aplikasi**

* Pergi ke WeCom Admin Console → App Management → Create App
* Salin **AgentId** dan **Secret**
* Pergi ke halaman "My Company", salin **CorpID**

**2. Konfigurasi penerimaan mesej**

* Dalam butiran aplikasi, klik "Receive Message" → "Set API"
* Tetapkan URL kepada `http://your-server:18790/webhook/wecom-app`
* Jana **Token** dan **EncodingAESKey**

**3. Konfigurasi**

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": []
    }
  }
}
```

**4. Jalankan**

```bash
picoclaw gateway
```

> **Nota**: Callback webhook WeCom diservis pada port Gateway (lalai 18790). Gunakan reverse proxy untuk HTTPS.

**Quick Setup - WeCom AI Bot:**

**1. Cipta AI Bot**

* Pergi ke WeCom Admin Console → App Management → AI Bot
* Dalam tetapan AI Bot, konfigurasikan callback URL: `http://your-server:18791/webhook/wecom-aibot`
* Salin **Token** dan klik "Random Generate" untuk **EncodingAESKey**

**2. Konfigurasi**

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "Hello! How can I help you?"
    }
  }
}
```

**3. Jalankan**

```bash
picoclaw gateway
```

> **Nota**: WeCom AI Bot menggunakan protokol streaming pull — tiada isu timeout balasan. Tugasan panjang (>30 saat) akan bertukar secara automatik kepada penghantaran push `response_url`.

</details>

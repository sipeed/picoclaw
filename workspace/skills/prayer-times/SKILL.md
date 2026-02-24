---
name: prayer-times
description: Jadwal sholat otomatis — reminder waktu sholat, sahur, dan buka puasa. Data dari jadwalsholatorg.
metadata: {"nanobot":{"emoji":"🕌"}}
---

# Prayer Times / Jadwal Sholat

Reminder otomatis waktu sholat. Data dari [jadwalsholatorg](https://github.com/lakuapik/jadwalsholatorg).
Notifikasi dikirim ke **Telegram + ntfy**.

## Script

Lokasi: `skills/prayer-times/scripts/prayer_notify.sh`

```bash
chmod +x skills/prayer-times/scripts/prayer_notify.sh
```

| Command | Fungsi |
|---------|--------|
| `setup <city>` | Set kota + fetch data awal **(wajib pertama kali)** |
| `fetch` | Download JSON bulan ini |
| `today` | Tampilkan jadwal hari ini |
| `schedule [prayers...]` | Output `nama\|HH:MM\|detik` untuk sholat yang belum lewat |
| `notify <prayer> <time>` | Kirim ke ntfy + cetak pesan untuk Telegram |
| `status` | Tampilkan config dan status data |

**Auto-fetch**: `schedule` dan `today` otomatis fetch jika data bulan ini belum ada.
Jadi meski device mati saat awal bulan, data tetap ter-update saat dipakai.

## Trigger Phrases

- "jadwal sholat" / "waktu sholat" / "prayer times"
- "reminder sholat" / "aktifkan adzan"
- "reminder sahur" / "bangun sahur"
- "reminder buka puasa" / "iftar"
- "stop/nonaktifkan reminder sholat"

## One-Time Setup

Saat user **pertama kali** minta reminder sholat:

### 1. Tanya kota dan ntfy topic
- Kota: samarinda, dumai, pekanbaru, jakarta-pusat, surabaya, dll
- ntfy topic: URL ntfy.sh user (opsional, bisa ditambah nanti)

### 2. Setup kota (via exec tool, BUKAN cron)
```bash
bash skills/prayer-times/scripts/prayer_notify.sh setup samarinda
```
Ini akan save config ke `~/.picoclaw/prayer-times/config` dan fetch data bulan ini.

### 3. Setup ntfy (opsional, via exec tool)
Jika user memberikan ntfy URL, tambahkan ke config:
```bash
sed -i 's|NTFY_TOPIC=".*"|NTFY_TOPIC="https://ntfy.sh/USER_TOPIC"|' ~/.picoclaw/prayer-times/config
```
Simpan juga ke MEMORY.md agar tidak lupa.

### 4. Setup monthly fetch cron (tanggal 1, jam 01:00)
```json
{"action": "add", "message": "Monthly prayer fetch", "command": "bash skills/prayer-times/scripts/prayer_notify.sh fetch", "cron_expr": "0 1 1 * *"}
```

### 5. Setup daily scheduler (jam 01:30 setiap hari)
```json
{
  "action": "add",
  "message": "Baca jadwal sholat hari ini. Jalankan: bash skills/prayer-times/scripts/prayer_notify.sh schedule. Untuk setiap baris output (format: nama|waktu|detik), buat 2 cron job one-time: (1) deliver:true untuk Telegram dengan pesan emoji, (2) command dengan: bash skills/prayer-times/scripts/prayer_notify.sh notify <nama> <waktu> untuk ntfy.",
  "cron_expr": "30 1 * * *",
  "deliver": false
}
```

## Config File

Disimpan di `~/.picoclaw/prayer-times/config` — **milik skill ini sendiri, tidak shared**:
```
CITY="samarinda"
SAHUR_MINS="30"
IFTAR_MINS="10"
PRAYERS="shubuh dzuhur ashr magrib isya"
NTFY_TOPIC="https://ntfy.sh/user-topic-here"
```

User bisa minta ubah via exec tool:
- Tambah sahur+iftar: edit PRAYERS di config lalu restart daily scheduler
- Ganti kota: `prayer_notify.sh setup <kota_baru>`
- Ganti ntfy: edit NTFY_TOPIC di config

## Cara Agent Memproses Daily Scheduler

Saat daily scheduler trigger (jam 01:30), agent HARUS:

### Step 1: Jalankan script via exec tool
```bash
bash skills/prayer-times/scripts/prayer_notify.sh schedule
```

Output contoh:
```
shubuh|05:06|12360
dzuhur|12:27|52020
ashr|15:42|63720
magrib|18:30|73800
isya|19:39|77940
```

### Step 2: Untuk SETIAP baris, buat 2 cron job

**Telegram** (deliver=true):
```json
{"action": "add", "message": "🌅 Waktu Shubuh (05:06) — Saatnya menunaikan sholat shubuh.", "at_seconds": 12360}
```

**ntfy** (command):
```json
{"action": "add", "message": "ntfy: shubuh", "command": "bash skills/prayer-times/scripts/prayer_notify.sh notify shubuh 05:06", "at_seconds": 12360}
```

### Step 3: Konfirmasi (satu pesan ringkasan)
Kirim ringkasan ke user: "✅ Reminder sholat hari ini sudah diset: Shubuh 05:06, Dzuhur 12:27, ..."

## Waktu yang Tersedia

| Nama | Keterangan | Perlu diminta user? |
|------|------------|---------------------|
| `shubuh` | Waktu Shubuh | Default aktif |
| `dzuhur` | Waktu Dzuhur | Default aktif |
| `ashr` | Waktu Ashar | Default aktif |
| `magrib` | Waktu Maghrib | Default aktif |
| `isya` | Waktu Isya | Default aktif |
| `sahur` | 30 menit sebelum Shubuh | Ya — untuk Ramadan |
| `iftar` | 10 menit sebelum Maghrib | Ya — untuk Ramadan |

## Rules

1. **Tanya kota** saat pertama kali — jangan asumsi. Jalankan `setup <kota>` sebelum apapun.
2. **JANGAN hardcode waktu sholat** — selalu baca dari script output. Waktu berubah setiap hari.
3. **JANGAN buat cron_expr untuk waktu sholat** — karena waktu berubah harian, gunakan daily scheduler + at_seconds.
4. **Parse output dengan benar** — format: `nama|HH:MM|detik`. Kolom ke-3 = `at_seconds`.
5. **Dual delivery** — setiap reminder: Telegram (deliver:true) + ntfy (via notify command).
6. **at_seconds auto-delete** — one-time job otomatis hilang setelah trigger.
7. **Auto-fetch** — script otomatis download data jika file bulan ini belum ada. Aman jika device mati saat awal bulan.

# 🔄 Spawn & Tugasan Async

> Kembali ke [README](../../README.my.md)

## Tugasan Cepat (balas terus)

- Laporkan masa semasa

## Tugasan Panjang (guna spawn untuk async)

- Cari berita AI di web dan ringkaskan
- Semak e-mel dan laporkan mesej penting
```

**Tingkah laku utama:**

| Feature                 | Description                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Mencipta sub-agen async, tidak menyekat heartbeat         |
| **Independent context** | Sub-agen mempunyai konteks sendiri, tiada sejarah sesi    |
| **message tool**        | Sub-agen berkomunikasi terus dengan pengguna melalui message tool |
| **Non-blocking**        | Selepas spawn, heartbeat terus ke tugasan seterusnya      |

#### Cara Komunikasi Sub-agen Berfungsi

```
Heartbeat dicetuskan
    ↓
Agen membaca HEARTBEAT.md
    ↓
Untuk tugasan panjang: spawn sub-agen
    ↓                           ↓
Terus ke tugasan seterusnya  Sub-agen bekerja secara bebas
    ↓                           ↓
Semua tugasan selesai     Sub-agen menggunakan tool "message"
    ↓                           ↓
Balas HEARTBEAT_OK        Pengguna menerima hasil secara terus
```

Sub-agen mempunyai akses kepada tools (message, web_search, dan sebagainya) dan boleh berkomunikasi dengan pengguna secara bebas tanpa melalui agen utama.

**Konfigurasi:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Default | Description                              |
| ---------- | ------- | ---------------------------------------- |
| `enabled`  | `true`  | Hidupkan/matikan heartbeat               |
| `interval` | `30`    | Selang semakan dalam minit (minimum: 5) |

**Pemboleh ubah persekitaran:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` untuk nyahaktifkan
* `PICOCLAW_HEARTBEAT_INTERVAL=60` untuk menukar selang

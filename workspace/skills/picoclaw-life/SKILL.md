---
name: picoclaw-life
description: Setup, manage, dan monitor PicoClaw gateway lifecycle — install service, start/stop/restart, lihat status dan log.
metadata: {"nanobot":{"emoji":"🦀"}}
---

# PicoClaw Lifecycle Manager

Manage PicoClaw gateway lifecycle: install sebagai systemd service, start/stop/restart, monitor status dan log.

## When to use (trigger phrases)

- "start/stop/restart picoclaw"
- "status picoclaw" / "picoclaw jalan gak?"
- "install picoclaw service" / "setup picoclaw"
- "update picoclaw manager"
- "log picoclaw" / "lihat log picoclaw"
- "uninstall picoclaw service"

## Arsitektur

```
systemd → picoclaw_manager.py (port 8321) → picoclaw gateway
```

- `picoclaw_manager.py` jalan sebagai systemd service (`picoclaw-manager`)
- REST API untuk kontrol gateway (start/stop/restart/status)
- Auto-start gateway saat service dimulai

## API Endpoints

| Method | Endpoint | Fungsi |
|--------|----------|--------|
| `GET` | `/api/health` | Health check manager |
| `GET` | `/api/picoclaw/status` | Status gateway (running, pid, uptime, recent logs) |
| `POST` | `/api/picoclaw/start` | Start gateway |
| `POST` | `/api/picoclaw/stop` | Stop gateway |
| `POST` | `/api/picoclaw/restart` | Restart gateway |

## Setup (Pertama Kali)

Script didownload langsung dari repo agar selalu versi terbaru.

### 1. Download & install via exec tool

```bash
curl -fsSL https://raw.githubusercontent.com/muava12/picoclaw-fork/main/setup_picoclaw_manager.sh -o /tmp/setup_picoclaw_manager.sh && curl -fsSL https://raw.githubusercontent.com/muava12/picoclaw-fork/main/picoclaw_manager.py -o /tmp/picoclaw_manager.py && bash /tmp/setup_picoclaw_manager.sh install
```

Ini akan:
- Copy `picoclaw_manager.py` ke `/opt/picoclaw/`
- Buat systemd service `picoclaw-manager`
- Enable auto-start on boot
- Start service

### 2. Verifikasi

```bash
curl -s http://localhost:8321/api/health
```

## Update Manager

Untuk update ke versi terbaru, download ulang dan jalankan update:

```bash
curl -fsSL https://raw.githubusercontent.com/muava12/picoclaw-fork/main/setup_picoclaw_manager.sh -o /tmp/setup_picoclaw_manager.sh && curl -fsSL https://raw.githubusercontent.com/muava12/picoclaw-fork/main/picoclaw_manager.py -o /tmp/picoclaw_manager.py && bash /tmp/setup_picoclaw_manager.sh update
```

## Perintah Harian

### Cek status

```bash
curl -s http://localhost:8321/api/picoclaw/status
```

### Restart gateway

```bash
curl -s -X POST http://localhost:8321/api/picoclaw/restart
```

### Start/stop gateway

```bash
curl -s -X POST http://localhost:8321/api/picoclaw/start
curl -s -X POST http://localhost:8321/api/picoclaw/stop
```

### Lihat log terakhir (via systemd)

```bash
journalctl -u picoclaw-manager --no-pager -n 30
```

### Follow live log (via systemd)

> ⚠️ **JANGAN jalankan dari exec tool** — streaming tidak akan selesai.

```bash
journalctl -u picoclaw-manager -f
```

## Rules

1. **Prefer curl API** — untuk start/stop/restart/status, gunakan curl ke `localhost:8321` karena lebih cepat dan tidak butuh sudo.
2. **Gunakan `journalctl -n N`** untuk log — jangan pakai `-f` (streaming) dari exec tool.
3. **Install dulu** — jika baru pertama kali, jalankan setup download+install.
4. **Selalu terbaru** — script didownload dari GitHub saat install/update, jadi cukup update repo untuk propagate perubahan.

## Config

- **Binary PicoClaw**: `~/.local/bin/picoclaw`
- **Config**: `~/.picoclaw/config.json`
- **Port Manager API**: `8321`
- **Install dir**: `/opt/picoclaw/`
- **Service**: `picoclaw-manager.service`

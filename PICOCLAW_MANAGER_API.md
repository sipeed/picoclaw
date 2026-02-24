# PicoClaw Manager API

HTTP API server untuk mengontrol lifecycle PicoClaw gateway process.

Base URL: `http://{host}:{port}` (default: `http://localhost:8321`)

## Endpoints

### `GET /api/health`
Health check server. Tidak perlu auth.
- Response: `{"status": "ok", "service": "picoclaw-manager", "timestamp": "..."}`

### `GET /api/picoclaw/status`
Cek status PicoClaw gateway: apakah running, PID, uptime, dan 20 baris log terakhir.
- Response: `{"running": true, "pid": 1234, "started_at": "...", "uptime_seconds": 3600, "recent_logs": [...]}`

### `POST /api/picoclaw/start`
Jalankan PicoClaw gateway. Gagal jika sudah berjalan.
- Response sukses: `{"success": true, "message": "...", "pid": 1234}`
- Response gagal: `{"success": false, "message": "PicoClaw gateway sudah berjalan"}`

### `POST /api/picoclaw/stop`
Hentikan PicoClaw gateway (SIGTERM → SIGKILL fallback).
- Response: `{"success": true, "message": "PicoClaw gateway berhasil dihentikan (PID: 1234)"}`

### `POST /api/picoclaw/restart`
Stop lalu start ulang PicoClaw gateway. Bisa dipanggil meskipun gateway sedang tidak berjalan.
- Response: `{"success": true, "message": "...", "pid": 5678}`

## Contoh Request
```bash
curl http://localhost:8321/api/picoclaw/status
curl -X POST http://localhost:8321/api/picoclaw/start
curl -X POST http://localhost:8321/api/picoclaw/restart
```

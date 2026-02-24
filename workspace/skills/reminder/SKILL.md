---
name: reminder
description: Schedule reminders, recurring tasks, and system monitoring jobs using the built-in cron tool.
metadata: {"nanobot":{"emoji":"⏰"}}
---

# Reminder & Scheduling

Manage reminders and scheduled tasks using the `cron` tool. All schedules persist across restarts.

## When to use (trigger phrases)

Use this skill immediately when the user says any of:
- "ingatkan aku" / "remind me" / "kasih reminder"
- "jadwalkan" / "schedule" / "set alarm"
- "setiap X menit/jam" / "every X minutes/hours"
- "jam 8 pagi" / "at 9am" / "besok pagi"
- "cek disk setiap jam" / "monitor CPU"
- "batalkan reminder" / "hapus jadwal" / "cancel"
- "lihat jadwal" / "list reminders"

## Tool: `cron`

### Actions

| Action    | Purpose                        | Required Params         |
|-----------|--------------------------------|-------------------------|
| `add`     | Create new reminder/task       | `message` + schedule    |
| `list`    | Show all active schedules      | —                       |
| `remove`  | Delete a schedule              | `job_id`                |
| `enable`  | Re-enable a disabled schedule  | `job_id`                |
| `disable` | Pause without deleting         | `job_id`                |

### Schedule Types (pick exactly ONE)

#### 1. `at_seconds` — One-time reminder

Triggers once, then auto-deletes. Value = seconds from now.

| User says                  | `at_seconds` value |
|----------------------------|--------------------|
| "dalam 5 menit"           | `300`              |
| "dalam 30 menit"          | `1800`             |
| "dalam 1 jam"             | `3600`             |
| "dalam 2 jam"             | `7200`             |
| "besok jam 8" (±15 jam)   | `54000`            |

```json
{
  "action": "add",
  "message": "Waktunya meeting standup!",
  "at_seconds": 1800
}
```

#### 2. `every_seconds` — Recurring interval

Repeats indefinitely at fixed intervals.

| User says               | `every_seconds` value |
|-------------------------|-----------------------|
| "setiap 5 menit"       | `300`                 |
| "setiap 30 menit"      | `1800`                |
| "setiap 1 jam"          | `3600`                |
| "setiap 2 jam"          | `7200`                |
| "setiap 6 jam"          | `21600`               |
| "setiap hari" (24 jam)  | `86400`               |

```json
{
  "action": "add",
  "message": "Jangan lupa minum air!",
  "every_seconds": 3600
}
```

#### 3. `cron_expr` — Cron expression (complex schedules)

Standard 5-field cron: `minute hour day-of-month month day-of-week`

| User says                    | `cron_expr`       |
|------------------------------|-------------------|
| "setiap hari jam 8 pagi"    | `0 8 * * *`       |
| "setiap hari jam 6 sore"    | `0 18 * * *`      |
| "Senin-Jumat jam 9 pagi"    | `0 9 * * 1-5`     |
| "setiap Senin jam 10"       | `0 10 * * 1`      |
| "setiap tanggal 1 jam 9"    | `0 9 1 * *`       |
| "setiap 15 menit"           | `*/15 * * * *`    |
| "jam 8 dan jam 20"          | `0 8,20 * * *`    |

```json
{
  "action": "add",
  "message": "Saatnya review laporan harian",
  "cron_expr": "0 17 * * 1-5"
}
```

### Parameter `deliver`

Controls how the message is sent when triggered:

| Value   | Behavior                                                    | Use for                          |
|---------|-------------------------------------------------------------|----------------------------------|
| `true`  | Send message directly to chat (default)                     | Simple reminders, notifications  |
| `false` | Route message through agent (agent processes it as a task)  | Complex tasks needing AI action  |

```json
{
  "action": "add",
  "message": "Cek cuaca hari ini dan beri rekomendasi pakaian",
  "cron_expr": "0 6 * * *",
  "deliver": false
}
```

### Parameter `command`

Execute a shell command instead of sending a message. Output is sent to chat. Sets `deliver=false` automatically.

```json
{
  "action": "add",
  "message": "Cek penggunaan disk",
  "command": "df -h / | tail -1",
  "every_seconds": 3600
}
```

```json
{
  "action": "add",
  "message": "Monitor memory usage",
  "command": "free -h | grep Mem",
  "cron_expr": "*/30 * * * *"
}
```

## Rules

1. **DUAL DELIVERY (WAJIB)** — Setiap reminder harus membuat **2 cron job** dengan schedule yang sama:
   - Job 1: `deliver: true` → kirim ke Telegram (channel aktif)
   - Job 2: `command` dengan curl ke ntfy → push notification ke HP
2. **Always confirm** after creating — tell the user what was scheduled and when it will trigger.
3. **Use `at_seconds`** for one-time reminders. Do NOT use `every_seconds` or `cron_expr` for one-time tasks.
4. **Use `every_seconds`** for simple recurring intervals. Use `cron_expr` only when the user needs day-of-week or specific time-of-day control.
5. **Calculate `at_seconds` correctly** — convert the user's time reference to seconds from now. If the user says "in 2 hours", use `7200`, not a timestamp.
6. **Timezone awareness** — the system uses the configured timezone (`Asia/Makassar` = WITA, UTC+8). All cron expressions run in this timezone.
7. **List before remove** — when the user wants to cancel a reminder, call `list` first to get the `job_id`, then `remove`. Remove BOTH the Telegram and ntfy jobs.
8. **Never hallucinate job IDs** — always get real IDs from `list`.
9. **Message should be actionable** — write the reminder message as what the user needs to see/do, not what the tool parameters are.

## Setup (One-Time)

Saat user pertama kali minta reminder dengan ntfy, pastikan ntfy topic dikonfigurasi.

Config disimpan di `skills/reminder/data/ntfy.conf` — **milik skill ini sendiri, tidak shared**.

### 1. Tanya ntfy topic URL ke user
### 2. Simpan ke config (via exec tool)
```bash
mkdir -p skills/reminder/data && echo 'NTFY_TOPIC="https://ntfy.sh/USER_TOPIC"' > skills/reminder/data/ntfy.conf
```
### 3. Simpan juga ke MEMORY.md agar tidak lupa

> Jika user tidak mau ntfy, skip setup ini. Reminder tetap dikirim ke Telegram.

## ntfy Push Notification

Gunakan helper script milik skill ini. **WAJIB source config sendiri sebelum panggil ntfy_send.sh**:

```bash
# Semua perintah ntfy harus diawali source config
source skills/reminder/data/ntfy.conf 2>/dev/null; NTFY_TOPIC="$NTFY_TOPIC" bash skills/reminder/scripts/ntfy_send.sh "MESSAGE" --title "JUDUL" --tags alarm_clock
```

> **JANGAN pakai curl langsung** ke ntfy. Selalu gunakan `ntfy_send.sh` agar URL dibaca dari config.

## Common Patterns

### One-time reminder (2 jobs)

**Job 1 — Telegram:**
```json
{"action": "add", "message": "⏰ Meeting dengan tim marketing!", "at_seconds": 600}
```
**Job 2 — ntfy push:**
```json
{"action": "add", "message": "ntfy: meeting", "command": "source skills/reminder/data/ntfy.conf 2>/dev/null; NTFY_TOPIC=\"$NTFY_TOPIC\" bash skills/reminder/scripts/ntfy_send.sh 'Meeting dengan tim marketing!' --title Reminder --tags alarm_clock", "at_seconds": 600}
```

### Recurring reminder (2 jobs)

**Job 1 — Telegram:**
```json
{"action": "add", "message": "💧 Jangan lupa minum air!", "every_seconds": 3600}
```
**Job 2 — ntfy push:**
```json
{"action": "add", "message": "ntfy: minum air", "command": "source skills/reminder/data/ntfy.conf 2>/dev/null; NTFY_TOPIC=\"$NTFY_TOPIC\" bash skills/reminder/scripts/ntfy_send.sh 'Jangan lupa minum air!' --title Hydration --tags droplet", "every_seconds": 3600}
```

### Daily cron reminder (2 jobs)

**Job 1 — Telegram:**
```json
{"action": "add", "message": "📋 Saatnya review laporan harian", "cron_expr": "0 17 * * 1-5"}
```
**Job 2 — ntfy push:**
```json
{"action": "add", "message": "ntfy: daily review", "command": "source skills/reminder/data/ntfy.conf 2>/dev/null; NTFY_TOPIC=\"$NTFY_TOPIC\" bash skills/reminder/scripts/ntfy_send.sh 'Saatnya review laporan harian' --title 'Daily Review' --tags memo", "cron_expr": "0 17 * * 1-5"}
```

### Daily report via agent (1 job only, no ntfy)
```json
{"action": "add", "message": "Rangkum log sistem hari ini dan kirim hasilnya", "cron_expr": "0 22 * * *", "deliver": false}
```

### System health check (1 job only, output to chat)
```json
{"action": "add", "message": "Health check", "command": "uptime && free -h | grep Mem && df -h / | tail -1", "every_seconds": 21600}
```

### Cancel a reminder
```json
{"action": "list"}
```
Then remove BOTH paired jobs (Telegram + ntfy) with their respective job_ids:
```json
{"action": "remove", "job_id": "telegram_job_id_here"}
```
```json
{"action": "remove", "job_id": "ntfy_job_id_here"}
```


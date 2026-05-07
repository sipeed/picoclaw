# ⚙️ Panduan Konfigurasi

> Kembali ke [README](../project/README.ms.md)

## ⚙️ Konfigurasi

Fail konfigurasi: `~/.picoclaw/config.json`

### Pemboleh Ubah Persekitaran

Anda boleh menggantikan laluan lalai menggunakan pemboleh ubah persekitaran. Ini berguna untuk pemasangan mudah alih, deployment dalam container, atau menjalankan picoclaw sebagai system service. Pemboleh ubah ini saling bebas dan mengawal laluan yang berbeza.

| Pemboleh Ubah     | Penerangan                                                                                                                                          | Laluan Lalai              |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `PICOCLAW_CONFIG` | Menindih laluan ke fail konfigurasi. Ini memberitahu picoclaw secara terus fail `config.json` yang perlu dimuatkan, dengan mengabaikan lokasi lain. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Menindih direktori root untuk data picoclaw. Ini mengubah lokasi lalai bagi `workspace` dan direktori data lain.                                    | `~/.picoclaw`             |

**Contoh:**

```bash
# Jalankan picoclaw menggunakan fail config tertentu
# Laluan workspace akan dibaca daripada fail config tersebut
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Jalankan picoclaw dengan semua data disimpan di /opt/picoclaw
# Config akan dimuatkan dari lalai ~/.picoclaw/config.json
# Workspace akan dicipta di /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Gunakan kedua-duanya untuk setup yang disesuaikan sepenuhnya
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Susun Atur Workspace

PicoClaw menyimpan data dalam workspace yang dikonfigurasikan (lalai: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sesi perbualan dan sejarah
├── memory/            # Memori jangka panjang (MEMORY.md)
├── state/             # Keadaan persisten (saluran terakhir, dll.)
├── cron/              # Pangkalan data job berjadual
├── skills/            # Skill tersuai
├── AGENTS.md          # Panduan tingkah laku agen
├── HEARTBEAT.md       # Prompt tugasan berkala (disemak setiap 30 minit)
├── IDENTITY.md        # Identiti agen
├── SOUL.md            # Jiwa agen
└── USER.md            # Keutamaan pengguna
```

### Sumber Skill

Secara lalai, skill dimuatkan daripada:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<current-working-directory>/skills` (builtin)

Untuk setup lanjutan/ujian, anda boleh menindih root builtin skills dengan:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Menggunakan Skill dan Arahan Dari Saluran Chat

Selepas skill dipasang, anda boleh menyemak dan memaksanya terus dari saluran chat:

- `/list skills` memaparkan nama skill dipasang yang kelihatan kepada agen semasa.
- `/use <skill> <message>` memaksa satu skill untuk satu permintaan sahaja.
- `/use <skill>` menyediakan skill itu untuk mesej anda yang seterusnya dalam chat yang sama.
- `/use clear` membatalkan skill override tertunda yang dibuat melalui `/use <skill>`.
- `/btw <question>` bertanya soalan sampingan segera tanpa mengubah sejarah sesi semasa. `/btw` dikendalikan sebagai pertanyaan langsung tanpa tool dan tidak memasuki aliran pelaksanaan tool biasa.

Contoh:

```text
/list skills
/use git terangkan cara squash 3 commit terakhir
/btw ingatkan saya semula apa keputusan tadi untuk pelan deploy
/use italiapersonalfinance
dammi le ultime news
```

### Polisi Pelaksanaan Arahan Bersepadu

- Generic slash command dilaksanakan melalui satu laluan dalam `pkg/agent/loop.go` melalui `commands.Executor`.
- Adapter saluran tidak lagi menggunakan generic command secara setempat; ia memajukan teks masuk ke laluan bus/agent. Telegram masih auto-register arahan yang disokong semasa startup seperti `/start`, `/help`, `/show`, `/list`, `/use`, dan `/btw`.
- Slash command yang tidak dikenali (contohnya `/foo`) akan diteruskan ke pemprosesan LLM biasa.
- Arahan yang didaftarkan tetapi tidak disokong pada saluran semasa (contohnya `/show` di WhatsApp) akan memulangkan ralat yang jelas kepada pengguna dan menghentikan pemprosesan lanjut.

### 🔒 Security Sandbox

PicoClaw berjalan dalam persekitaran bersandbox secara lalai. Agen hanya boleh mengakses fail dan melaksanakan arahan dalam workspace yang dikonfigurasikan.

#### Konfigurasi Lalai

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Option                  | Default                 | Description                               |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Direktori kerja untuk agen                |
| `restrict_to_workspace` | `true`                  | Hadkan akses fail/arahan kepada workspace |

#### Tools yang Dilindungi

Apabila `restrict_to_workspace: true`, tools berikut disandboxkan:

| Tool          | Fungsi            | Sekatan                             |
| ------------- | ----------------- | ----------------------------------- |
| `read_file`   | Baca fail         | Hanya fail dalam workspace          |
| `write_file`  | Tulis fail        | Hanya fail dalam workspace          |
| `list_dir`    | Senarai direktori | Hanya direktori dalam workspace     |
| `edit_file`   | Edit fail         | Hanya fail dalam workspace          |
| `append_file` | Tambah ke fail    | Hanya fail dalam workspace          |
| `exec`        | Jalankan arahan   | Laluan arahan mesti dalam workspace |

#### Perlindungan Exec Tambahan

Walaupun dengan `restrict_to_workspace: false`, tool `exec` menyekat arahan berbahaya berikut:

* `rm -rf`, `del /f`, `rmdir /s` — Pemadaman pukal
* `format`, `mkfs`, `diskpart` — Pemformatan cakera
* `dd if=` — Pengimejan cakera
* Menulis ke `/dev/sd[a-z]` — Tulis terus ke cakera
* `shutdown`, `reboot`, `poweroff` — Penutupan sistem
* Fork bomb `:(){ :|:& };:`

### Kawalan Akses Fail

| Kunci Config              | Jenis    | Lalai | Penerangan                                                      |
| ------------------------- | -------- | ----- | --------------------------------------------------------------- |
| `tools.allow_read_paths`  | string[] | `[]`  | Laluan tambahan yang dibenarkan untuk dibaca di luar workspace  |
| `tools.allow_write_paths` | string[] | `[]`  | Laluan tambahan yang dibenarkan untuk ditulis di luar workspace |

### Keselamatan Exec

| Kunci Config                       | Jenis    | Lalai   | Penerangan                                                   |
| ---------------------------------- | -------- | ------- | ------------------------------------------------------------ |
| `tools.exec.allow_remote`          | bool     | `false` | Benarkan tool exec dari saluran jauh (Telegram/Discord dll.) |
| `tools.exec.enable_deny_patterns`  | bool     | `true`  | Aktifkan pemintasan arahan berbahaya                         |
| `tools.exec.custom_deny_patterns`  | string[] | `[]`    | Corak regex tersuai untuk disekat                            |
| `tools.exec.custom_allow_patterns` | string[] | `[]`    | Corak regex tersuai untuk dibenarkan                         |

> **Nota Keselamatan:** Perlindungan symlink diaktifkan secara lalai — semua laluan fail akan diselesaikan melalui `filepath.EvalSymlinks` sebelum dipadankan dengan whitelist, bagi mengelakkan serangan melarikan diri melalui symlink.

#### Had yang Diketahui: Proses Anak Daripada Build Tools

Pengawal keselamatan exec hanya memeriksa baris arahan yang PicoClaw lancarkan secara terus. Ia tidak memeriksa secara rekursif proses anak yang dilancarkan oleh tools pembangun yang dibenarkan seperti `make`, `go run`, `cargo`, `npm run`, atau skrip build tersuai.

Ini bermakna arahan peringkat atas masih boleh mengkompil atau melancarkan binari lain selepas ia melepasi semakan awal pengawal. Dalam amalan, anggap build script, Makefile, package script, dan binari terjana sebagai kod boleh laksana yang memerlukan tahap semakan yang sama seperti arahan shell terus.

Untuk persekitaran yang lebih berisiko:

* Semak build script sebelum pelaksanaan.
* Utamakan kelulusan/semakan manual untuk aliran kerja compile-and-run.
* Jalankan PicoClaw dalam container atau VM jika anda memerlukan pengasingan yang lebih kuat daripada pengawal terbina dalam.

#### Contoh Ralat

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Menyahaktifkan Sekatan (Risiko Keselamatan)

Jika anda perlu membenarkan agen mengakses laluan di luar workspace:

**Kaedah 1: Fail config**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Kaedah 2: Pemboleh ubah persekitaran**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Amaran**: Menyahaktifkan sekatan ini membenarkan agen mengakses mana-mana laluan pada sistem anda. Gunakan dengan berhati-hati hanya dalam persekitaran terkawal.

#### Ketekalan Sempadan Keselamatan

Tetapan `restrict_to_workspace` digunakan secara konsisten merentas semua laluan pelaksanaan:

| Execution Path   | Security Boundary           |
| ---------------- | --------------------------- |
| Main Agent       | `restrict_to_workspace` ✅   |
| Subagent / Spawn | Inherits same restriction ✅ |
| Heartbeat tasks  | Inherits same restriction ✅ |

Semua laluan berkongsi sekatan workspace yang sama — tiada cara untuk memintas sempadan keselamatan melalui subagent atau tugasan berjadual.

### Heartbeat (Tugasan Berkala)

PicoClaw boleh melaksanakan tugasan berkala secara automatik. Cipta fail `HEARTBEAT.md` dalam workspace anda:

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast
```

Agen akan membaca fail ini setiap 30 minit (boleh dikonfigurasi) dan melaksanakan sebarang tugasan menggunakan tools yang tersedia.

#### Tugasan Async dengan Spawn

Untuk tugasan yang berjalan lama (carian web, panggilan API), gunakan tool `spawn` untuk mencipta **subagent**:

```markdown
# Periodic Tasks

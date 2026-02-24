#!/bin/bash
# prayer_notify.sh — Fetch, schedule, and send prayer time notifications
# Usage:
#   prayer_notify.sh fetch                  — Download current month's JSON
#   prayer_notify.sh today                  — Show today's prayer times
#   prayer_notify.sh schedule [prayers...]  — Output seconds-from-now for each prayer
#   prayer_notify.sh notify <prayer> <time> — Send notification via ntfy + stdout
#   prayer_notify.sh setup <city>           — Set city and fetch initial data
#   prayer_notify.sh status                 — Show current config and data status
#
# Config file: skills/prayer-times/data/config (co-located with skill)
# Auto-fetch: schedule command auto-fetches if data is missing or stale

set -euo pipefail

# Resolve paths relative to this script's location
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DATA_DIR="${PRAYER_DATA_DIR:-$SKILL_DIR/data}"
CONFIG_FILE="$DATA_DIR/config"

# ntfy_send.sh lives alongside this script
NTFY_SEND="${SCRIPT_DIR}/ntfy_send.sh"

mkdir -p "$DATA_DIR"

# ---- Load config ----

load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        # Source config file (contains CITY=, SAHUR_MINS=, IFTAR_MINS=, PRAYERS=)
        . "$CONFIG_FILE"
    fi
    CITY="${CITY:-samarinda}"
    SAHUR_MINS="${SAHUR_MINS:-30}"
    IFTAR_MINS="${IFTAR_MINS:-10}"
    PRAYERS="${PRAYERS:-shubuh dzuhur ashr magrib isya}"
    NTFY_TOPIC="${NTFY_TOPIC:-}"
}

save_config() {
    cat > "$CONFIG_FILE" << EOF
CITY="$CITY"
SAHUR_MINS="$SAHUR_MINS"
IFTAR_MINS="$IFTAR_MINS"
PRAYERS="$PRAYERS"
NTFY_TOPIC="$NTFY_TOPIC"
EOF
    echo "Config saved to $CONFIG_FILE"
}

# ---- Helper functions ----

get_json_path() {
    local year month
    year=$(date +%Y)
    month=$(date +%m)
    echo "$DATA_DIR/${CITY}_${year}_${month}.json"
}

is_data_current() {
    local json_path
    json_path=$(get_json_path)
    [ -f "$json_path" ]
}

ensure_data() {
    # Auto-fetch if data for current month is missing
    if ! is_data_current; then
        echo "Data bulan ini belum ada, auto-fetching..." >&2
        cmd_fetch
    fi
}

now_epoch() {
    date +%s
}

today_date() {
    date +%Y-%m-%d
}

# ---- Commands ----

cmd_setup() {
    local city="${1:-}"
    if [ -z "$city" ]; then
        echo "Usage: prayer_notify.sh setup <city>"
        echo ""
        echo "Contoh kota: samarinda, dumai, pekanbaru, jakarta-pusat, surabaya"
        echo "Lihat daftar di: https://github.com/lakuapik/jadwalsholatorg/tree/master/adzan"
        exit 1
    fi

    CITY="$city"
    save_config

    echo "Kota diset ke: $CITY"
    echo "Mengambil jadwal sholat..."
    cmd_fetch

    echo ""
    echo "Setup selesai! Gunakan:"
    echo "  prayer_notify.sh today      — lihat jadwal hari ini"
    echo "  prayer_notify.sh schedule   — hitung waktu reminder"
}

cmd_fetch() {
    local year month json_path url
    year=$(date +%Y)
    month=$(date +%m)
    json_path=$(get_json_path)
    url="https://raw.githubusercontent.com/lakuapik/jadwalsholatorg/master/adzan/${CITY}/${year}/${month}.json"

    echo "Fetching: ${CITY} ${year}-${month}..."
    if curl -sf "$url" -o "$json_path"; then
        local count
        count=$(python3 -c "import json; print(len(json.load(open('$json_path'))))" 2>/dev/null || echo "?")
        echo "OK: $json_path ($count hari)"
    else
        echo "ERROR: Gagal fetch dari $url"
        echo "Pastikan nama kota benar. Cek: https://github.com/lakuapik/jadwalsholatorg/tree/master/adzan"
        exit 1
    fi
}

cmd_today() {
    ensure_data
    local json_path today
    json_path=$(get_json_path)
    today=$(today_date)

    python3 -c "
import json, sys
data = json.load(open('$json_path'))
for entry in data:
    if entry['tanggal'] == '$today':
        print(f'📅 Jadwal Sholat $CITY ({entry[\"tanggal\"]})')
        print(f'  Imsyak  : {entry[\"imsyak\"]}')
        print(f'  Shubuh  : {entry[\"shubuh\"]}')
        print(f'  Terbit  : {entry[\"terbit\"]}')
        print(f'  Dhuha   : {entry[\"dhuha\"]}')
        print(f'  Dzuhur  : {entry[\"dzuhur\"]}')
        print(f'  Ashar   : {entry[\"ashr\"]}')
        print(f'  Maghrib : {entry[\"magrib\"]}')
        print(f'  Isya    : {entry[\"isya\"]}')
        sys.exit(0)
print('Tidak ada data untuk $today')
sys.exit(1)
"
}

cmd_schedule() {
    ensure_data
    local json_path today now_ts
    json_path=$(get_json_path)
    today=$(today_date)
    now_ts=$(now_epoch)

    # Filter: use args if provided, otherwise use config
    local filter="${*:-$PRAYERS}"

    python3 -c "
import json, sys, time
from datetime import datetime

data = json.load(open('$json_path'))
today = '$today'
now_ts = int('$now_ts')
sahur_mins = int('$SAHUR_MINS')
iftar_mins = int('$IFTAR_MINS')
filter_arg = '$filter'.strip()

entry = None
for e in data:
    if e['tanggal'] == today:
        entry = e
        break

if not entry:
    print('ERROR: Tidak ada data untuk ' + today, file=sys.stderr)
    sys.exit(1)

def hhmm_to_epoch(hhmm):
    h, m = map(int, hhmm.split(':'))
    dt = datetime.strptime(today + f' {h:02d}:{m:02d}:00', '%Y-%m-%d %H:%M:%S')
    return int(dt.timestamp())

schedule = []

# Sahur: N min before Shubuh
shubuh_epoch = hhmm_to_epoch(entry['shubuh'])
sahur_epoch = shubuh_epoch - (sahur_mins * 60)
sahur_time = time.strftime('%H:%M', time.localtime(sahur_epoch))
schedule.append(('sahur', sahur_time, sahur_epoch))

# 5 waktu wajib
for name in ['shubuh', 'dzuhur', 'ashr', 'magrib', 'isya']:
    schedule.append((name, entry[name], hhmm_to_epoch(entry[name])))

# Iftar: N min before Maghrib
magrib_epoch = hhmm_to_epoch(entry['magrib'])
iftar_epoch = magrib_epoch - (iftar_mins * 60)
iftar_time = time.strftime('%H:%M', time.localtime(iftar_epoch))
schedule.append(('iftar', iftar_time, iftar_epoch))

# Filter
wanted = set(f.strip().lower() for f in filter_arg.split())
schedule = [s for s in schedule if s[0] in wanted]

# Output future prayers only
found = False
for name, display_time, epoch in sorted(schedule, key=lambda x: x[2]):
    diff = epoch - now_ts
    if diff > 0:
        print(f'{name}|{display_time}|{diff}')
        found = True

if not found:
    print('INFO: Semua waktu sholat hari ini sudah lewat.', file=sys.stderr)
"
}

cmd_notify() {
    local prayer="$1"
    local prayer_time="${2:-}"

    local emoji tag title
    case "$prayer" in
        sahur)  emoji="🌙"; tag="crescent_moon"; title="Waktu Sahur";;
        shubuh) emoji="🌅"; tag="sunrise"; title="Waktu Shubuh";;
        dzuhur) emoji="☀️"; tag="sun"; title="Waktu Dzuhur";;
        ashr)   emoji="🌤️"; tag="sun_behind_cloud"; title="Waktu Ashar";;
        magrib) emoji="🌇"; tag="city_sunset"; title="Waktu Maghrib";;
        isya)   emoji="🌃"; tag="night_with_stars"; title="Waktu Isya";;
        iftar)  emoji="🍽️"; tag="fork_and_knife"; title="Persiapan Buka Puasa";;
        *)      emoji="🕌"; tag="mosque"; title="Waktu Sholat";;
    esac

    local message
    if [ "$prayer" = "sahur" ]; then
        message="${emoji} ${title} (${prayer_time}) — Ayo bangun sahur! ${SAHUR_MINS} menit lagi waktu Imsyak."
    elif [ "$prayer" = "iftar" ]; then
        message="${emoji} ${title} (${prayer_time}) — ${IFTAR_MINS} menit lagi waktu berbuka puasa!"
    else
        message="${emoji} ${title} (${prayer_time}) — Saatnya menunaikan sholat ${prayer}."
    fi

    # Send to ntfy via shared helper (export NTFY_TOPIC from our own config)
    if [ -f "$NTFY_SEND" ]; then
        NTFY_TOPIC="$NTFY_TOPIC" bash "$NTFY_SEND" "$message" --title "$title" --tags "$tag" --priority high 2>/dev/null || true
    fi

    # Output for PicoClaw to send to Telegram
    echo "$message"
}

cmd_status() {
    echo "=== Prayer Times Config ==="
    echo "Kota     : $CITY"
    echo "Sahur    : $SAHUR_MINS menit sebelum Shubuh"
    echo "Iftar    : $IFTAR_MINS menit sebelum Maghrib"
    echo "Prayers  : $PRAYERS"
    echo "Data dir : $DATA_DIR"
    echo ""
    # Show ntfy status from shared config
    if [ -x "$NTFY_SEND" ]; then
        bash "$NTFY_SEND" status
    else
        echo "ntfy: ⚠️  ntfy_send.sh not found"
    fi
    echo ""

    local json_path
    json_path=$(get_json_path)
    if [ -f "$json_path" ]; then
        local count
        count=$(python3 -c "import json; print(len(json.load(open('$json_path'))))" 2>/dev/null || echo "?")
        echo "Data bulan ini: ✅ Ada ($count hari)"
        echo "File: $json_path"
    else
        echo "Data bulan ini: ❌ Belum ada"
        echo "Jalankan: prayer_notify.sh fetch"
    fi
}

# ---- Main ----

load_config

case "${1:-help}" in
    setup)    shift; cmd_setup "$@" ;;
    fetch)    cmd_fetch ;;
    today)    cmd_today ;;
    schedule) shift; cmd_schedule "$@" ;;
    notify)   shift; cmd_notify "$@" ;;
    status)   cmd_status ;;
    help|*)
        echo "Usage: prayer_notify.sh {setup|fetch|today|schedule|notify|status}"
        echo ""
        echo "Commands:"
        echo "  setup <city>           Set kota dan fetch data awal"
        echo "  fetch                  Download jadwal bulan ini"
        echo "  today                  Tampilkan jadwal hari ini"
        echo "  schedule [prayers...]  Hitung detik-dari-sekarang untuk reminder"
        echo "  notify <prayer> <time> Kirim notifikasi (ntfy + stdout)"
        echo "  status                 Tampilkan config dan status data"
        echo ""
        echo "Config: $CONFIG_FILE"
        echo "Kota saat ini: $CITY"
        ;;
esac

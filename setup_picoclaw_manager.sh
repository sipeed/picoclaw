#!/bin/bash
# ═══════════════════════════════════════════════════
#  PicoClaw Manager — Installer & Service Manager
#  Untuk Armbian / Debian / Ubuntu
# ═══════════════════════════════════════════════════
set -e

SERVICE_NAME="picoclaw-manager"
INSTALL_DIR="/opt/picoclaw"
SCRIPT_NAME="picoclaw_manager.py"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
PICOCLAW_BIN="$HOME/.local/bin/picoclaw"
RUN_USER="$(whoami)"

# ── Warna ─────────────────────────────────────────
R='\033[0;31m' G='\033[0;32m' B='\033[0;34m'
Y='\033[1;33m' C='\033[0;36m' W='\033[1;37m' X='\033[0m'

banner() {
  echo ""
  echo -e "  ${C}┌──────────────────────────────────────┐${X}"
  echo -e "  ${C}│${W}   🦀 PicoClaw Service Manager       ${C}│${X}"
  echo -e "  ${C}└──────────────────────────────────────┘${X}"
  echo ""
}

info()    { echo -e "  ${B}▸${X} $1"; }
success() { echo -e "  ${G}✓${X} $1"; }
warn()    { echo -e "  ${Y}!${X} $1"; }
err()     { echo -e "  ${R}✗${X} $1"; }

# ── Install ───────────────────────────────────────
cmd_install() {
  banner
  info "Installing ${SERVICE_NAME}..."
  echo ""

  # Cari picoclaw_api.py di folder yang sama dengan script ini
  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  SOURCE="${SCRIPT_DIR}/${SCRIPT_NAME}"

  if [ ! -f "$SOURCE" ]; then
    err "${SCRIPT_NAME} tidak ditemukan di ${SCRIPT_DIR}"
    exit 1
  fi

  # Copy script
  info "Copying ${SCRIPT_NAME} → ${INSTALL_DIR}/"
  sudo mkdir -p "$INSTALL_DIR"
  sudo cp "$SOURCE" "${INSTALL_DIR}/${SCRIPT_NAME}"
  sudo chmod +x "${INSTALL_DIR}/${SCRIPT_NAME}"
  success "Script copied"

  # Copy .env jika ada
  if [ -f "${SCRIPT_DIR}/.env" ]; then
    sudo cp "${SCRIPT_DIR}/.env" "${INSTALL_DIR}/.env"
    success ".env copied"
  fi

  # Buat systemd service
  info "Creating systemd service..."
  sudo tee "$SERVICE_FILE" > /dev/null << UNIT
[Unit]
Description=PicoClaw Manager — Process Lifecycle Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=/usr/bin/python3 ${INSTALL_DIR}/${SCRIPT_NAME} --auto-start --picoclaw-bin ${PICOCLAW_BIN}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
Environment=PYTHONUNBUFFERED=1

[Install]
WantedBy=multi-user.target
UNIT
  success "Service file created: ${SERVICE_FILE}"

  # Reload & enable
  sudo systemctl daemon-reload
  sudo systemctl enable "$SERVICE_NAME"
  success "Service enabled (auto-start on boot)"

  # Start
  sudo systemctl start "$SERVICE_NAME"
  success "Service started"

  echo ""
  info "Cek status: ${W}$0 status${X}"
  info "Lihat log:  ${W}$0 logs${X}"
  echo ""
}

# ── Uninstall ─────────────────────────────────────
cmd_uninstall() {
  banner
  warn "Uninstalling ${SERVICE_NAME}..."
  echo ""

  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    sudo systemctl stop "$SERVICE_NAME"
    success "Service stopped"
  fi

  if [ -f "$SERVICE_FILE" ]; then
    sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    sudo rm -f "$SERVICE_FILE"
    sudo systemctl daemon-reload
    success "Service file removed"
  fi

  if [ -d "$INSTALL_DIR" ]; then
    read -p "  Hapus ${INSTALL_DIR}? [y/N] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      sudo rm -rf "$INSTALL_DIR"
      success "Install directory removed"
    fi
  fi

  success "Uninstall selesai"
  echo ""
}

# ── Update ────────────────────────────────────────
cmd_update() {
  banner
  info "Updating ${SCRIPT_NAME}..."

  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  SOURCE="${SCRIPT_DIR}/${SCRIPT_NAME}"

  if [ ! -f "$SOURCE" ]; then
    err "${SCRIPT_NAME} tidak ditemukan di ${SCRIPT_DIR}"
    exit 1
  fi

  sudo cp "$SOURCE" "${INSTALL_DIR}/${SCRIPT_NAME}"
  sudo chmod +x "${INSTALL_DIR}/${SCRIPT_NAME}"
  success "Script updated"

  sudo systemctl restart "$SERVICE_NAME"
  success "Service restarted"
  echo ""
}

# ── Service Commands ──────────────────────────────
cmd_start()   { sudo systemctl start "$SERVICE_NAME"   && success "Started";   }
cmd_stop()    { sudo systemctl stop "$SERVICE_NAME"     && success "Stopped";   }
cmd_restart() { sudo systemctl restart "$SERVICE_NAME"  && success "Restarted"; }

cmd_status() {
  banner
  echo -e "  ${W}Systemd Status:${X}"
  echo ""
  sudo systemctl status "$SERVICE_NAME" --no-pager -l 2>/dev/null || warn "Service not found"
  echo ""

  # Cek API health
  if command -v curl &> /dev/null; then
    echo -e "  ${W}API Health Check:${X}"
    echo ""
    RESPONSE=$(curl -s http://localhost:8321/api/health 2>/dev/null || echo "unreachable")
    if echo "$RESPONSE" | grep -q '"ok"'; then
      success "API is responding: ${RESPONSE}"
    else
      warn "API not responding"
    fi
    echo ""

    echo -e "  ${W}PicoClaw Gateway:${X}"
    echo ""
    GW_STATUS=$(curl -s http://localhost:8321/api/picoclaw/status 2>/dev/null || echo "{}")
    RUNNING=$(echo "$GW_STATUS" | grep -o '"running": *[a-z]*' | head -1 | awk '{print $2}')
    PID=$(echo "$GW_STATUS" | grep -o '"pid": *[0-9]*' | head -1 | awk '{print $2}')
    if [ "$RUNNING" = "true" ]; then
      success "Gateway running (PID: ${PID})"
    else
      warn "Gateway not running"
    fi
    echo ""
  fi
}

cmd_logs() {
  journalctl -u "$SERVICE_NAME" -f --no-pager
}

cmd_logs_history() {
  journalctl -u "$SERVICE_NAME" --no-pager -n "${1:-50}"
}

# ── Usage ─────────────────────────────────────────
cmd_help() {
  banner
  echo -e "  ${W}Usage:${X} $0 <command>"
  echo ""
  echo -e "  ${C}Setup${X}"
  echo "    install       Install & enable service"
  echo "    uninstall     Remove service & files"
  echo "    update        Update script & restart"
  echo ""
  echo -e "  ${C}Service${X}"
  echo "    start         Start the API server"
  echo "    stop          Stop the API server"
  echo "    restart       Restart the API server"
  echo "    status        Show status & health check"
  echo ""
  echo -e "  ${C}Logs${X}"
  echo "    logs          Follow live logs"
  echo "    logs-history  Show last 50 log lines"
  echo ""
}

# ── Dispatch ──────────────────────────────────────
case "${1:-help}" in
  install)      cmd_install      ;;
  uninstall)    cmd_uninstall    ;;
  update)       cmd_update       ;;
  start)        cmd_start        ;;
  stop)         cmd_stop         ;;
  restart)      cmd_restart      ;;
  status)       cmd_status       ;;
  logs)         cmd_logs         ;;
  logs-history) cmd_logs_history "$2" ;;
  help|--help|-h) cmd_help       ;;
  *)
    err "Unknown command: $1"
    cmd_help
    exit 1
    ;;
esac

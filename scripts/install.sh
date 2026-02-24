#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY_NAME="picoclaw"
BUILD_DIR="build"
INSTALL_PREFIX="${HOME}/.local"
PICOCLAW_HOME="${HOME}/.picoclaw"

# ── parse args ───────────────────────────────────────────────
ACTION="install"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --prefix)       INSTALL_PREFIX="$2"; shift 2 ;;
        --uninstall)    ACTION="uninstall"; shift ;;
        --uninstall-all) ACTION="uninstall-all"; shift ;;
        *)              echo "Unknown option: $1"; exit 1 ;;
    esac
done

INSTALL_BIN_DIR="${INSTALL_PREFIX}/bin"

# ── install ──────────────────────────────────────────────────
do_install() {
    "${SCRIPT_DIR}/build.sh"

    echo "Installing ${BINARY_NAME}..."
    mkdir -p "$INSTALL_BIN_DIR"

    # Atomic install: copy to temp, then rename
    cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_BIN_DIR}/${BINARY_NAME}.new"
    chmod +x "${INSTALL_BIN_DIR}/${BINARY_NAME}.new"
    mv -f "${INSTALL_BIN_DIR}/${BINARY_NAME}.new" "${INSTALL_BIN_DIR}/${BINARY_NAME}"

    echo "Installed to ${INSTALL_BIN_DIR}/${BINARY_NAME}"
}

# ── uninstall ────────────────────────────────────────────────
do_uninstall() {
    echo "Uninstalling ${BINARY_NAME}..."
    rm -f "${INSTALL_BIN_DIR}/${BINARY_NAME}"
    echo "Removed binary from ${INSTALL_BIN_DIR}/${BINARY_NAME}"
    echo "Note: config and workspace preserved. Use --uninstall-all to remove everything."
}

# ── uninstall-all ────────────────────────────────────────────
do_uninstall_all() {
    do_uninstall
    echo "Removing workspace and config..."
    rm -rf "$PICOCLAW_HOME"
    echo "Removed ${PICOCLAW_HOME}"
    echo "Complete uninstallation done!"
}

# ── main ─────────────────────────────────────────────────────
case "$ACTION" in
    install)       do_install       ;;
    uninstall)     do_uninstall     ;;
    uninstall-all) do_uninstall_all ;;
esac

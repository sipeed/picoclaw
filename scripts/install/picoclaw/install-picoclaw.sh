#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  printf 'Error: this installer must run with GNU bash (ash/busybox are not supported).\n' >&2
  exit 1
fi

if [ "${BASH_VERSINFO[0]}" -lt 4 ]; then
  printf 'Error: GNU bash 4+ is required. Current version: %s\n' "$BASH_VERSION" >&2
  exit 1
fi

if ! bash --version 2>/dev/null | head -n 1 | grep -qi 'gnu bash'; then
  printf 'Error: this installer requires a full GNU bash runtime.\n' >&2
  exit 1
fi

set -euo pipefail

INSTALL_MODE="user"
SOURCE="github"
ARCH_OVERRIDE=""

BASE_GITHUB="https://github.com/sipeed/picoclaw/releases/latest/download/"
BASE_CDN="https://picoclaw-downloads.tos-cn-beijing.volces.com/latest/"

EXPECTED_BINS=("picoclaw" "picoclaw-launcher" "picoclaw-launcher-tui")

usage() {
  cat <<'EOF'
Usage:
  install-picoclaw.sh [--mode system|user] [--source github|cdn] [--arch ARCH]

Examples:
  ./install-picoclaw.sh --mode user
  sudo ./install-picoclaw.sh --mode system --source cdn
EOF
}

log() {
  printf '%s\n' "$*"
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

create_secure_tmp_dir() {
  local prefix="${1:-picoclaw}"
  local d

  if d="$(mktemp -d "/tmp/${prefix}.XXXXXX" 2>/dev/null)"; then
    printf '%s\n' "$d"
    return 0
  fi

  local i candidate
  for i in 1 2 3 4 5 6 7 8 9 10; do
    candidate="/tmp/${prefix}.$$.${RANDOM}.$(date +%s)"
    if mkdir -m 700 "$candidate" 2>/dev/null; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  return 1
}

detect_os() {
  local u
  u="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$u" in
    linux) echo "linux" ;;
    darwin) echo "macos" ;;
    freebsd) echo "freebsd" ;;
    netbsd) echo "netbsd" ;;
    mingw*|msys*|cygwin*)
      die "Windows shell detected ($u). Please run the PowerShell installer instead: pwsh ./static/scripts/picoclaw/install-picoclaw.ps1 -InstallMode user"
      ;;
    *) die "Unsupported OS: $u" ;;
  esac
}

detect_arch() {
  if [[ -n "$ARCH_OVERRIDE" ]]; then
    echo "$ARCH_OVERRIDE"
    return
  fi

  local m
  m="$(uname -m | tr '[:upper:]' '[:lower:]')"
  case "$m" in
    x86_64|amd64) echo "x86_64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l|armv7) echo "armv7" ;;
    armv6l|armv6) echo "armv6" ;;
    riscv64) echo "riscv64" ;;
    mipsel|mipsle) echo "mipsle" ;;
    s390x) echo "s390x" ;;
    *) die "Unsupported architecture: $m (use --arch to override)" ;;
  esac
}

assert_privileges() {
  if [[ "$INSTALL_MODE" != "system" ]]; then
    return
  fi

  if [[ "$(id -u)" -ne 0 ]]; then
    die "System install requires root privileges (sudo/root)."
  fi
}

download_file() {
  local url="$1"
  local out="$2"

  log "Downloading: $url"
  if command_exists curl; then
    curl -fsSL "$url" -o "$out"
    return
  fi
  if command_exists wget; then
    wget -qO "$out" "$url"
    return
  fi
  die "Neither curl nor wget is available."
}

base_url() {
  case "$SOURCE" in
    github) echo "$BASE_GITHUB" ;;
    cdn) echo "$BASE_CDN" ;;
    *) die "Invalid source: $SOURCE" ;;
  esac
}

asset_for() {
  local os="$1"
  local arch="$2"

  TAR_FILE=""
  DEB_FILE=""
  RPM_FILE=""

  case "$os/$arch" in
    windows/x86_64) TAR_FILE="picoclaw_Windows_x86_64.zip" ;;
    windows/arm64) TAR_FILE="picoclaw_Windows_arm64.zip" ;;

    linux/x86_64)
      TAR_FILE="picoclaw_Linux_x86_64.tar.gz"
      DEB_FILE="picoclaw_x86_64.deb"
      RPM_FILE="picoclaw_x86_64.rpm"
      ;;
    linux/arm64)
      TAR_FILE="picoclaw_Linux_arm64.tar.gz"
      DEB_FILE="picoclaw_aarch64.deb"
      RPM_FILE="picoclaw_aarch64.rpm"
      ;;
    linux/armv7)
      TAR_FILE="picoclaw_Linux_armv7.tar.gz"
      DEB_FILE="picoclaw_armv7.deb"
      RPM_FILE="picoclaw_armv7.rpm"
      ;;
    linux/armv6)
      TAR_FILE="picoclaw_Linux_armv6.tar.gz"
      DEB_FILE="picoclaw_armv6.deb"
      RPM_FILE="picoclaw_armv6.rpm"
      ;;
    linux/riscv64)
      TAR_FILE="picoclaw_Linux_riscv64.tar.gz"
      DEB_FILE="picoclaw_riscv64.deb"
      RPM_FILE="picoclaw_riscv64.rpm"
      ;;
    linux/mipsle)
      TAR_FILE="picoclaw_Linux_mipsle.tar.gz"
      DEB_FILE="picoclaw_mipsle.deb"
      RPM_FILE="picoclaw_mipsle.rpm"
      ;;
    linux/s390x)
      TAR_FILE="picoclaw_Linux_s390x.tar.gz"
      DEB_FILE="picoclaw_s390x.deb"
      RPM_FILE="picoclaw_s390x.rpm"
      ;;

    macos/x86_64) TAR_FILE="picoclaw_Darwin_x86_64.tar.gz" ;;
    macos/arm64) TAR_FILE="picoclaw_Darwin_arm64.tar.gz" ;;

    freebsd/x86_64) TAR_FILE="picoclaw_Freebsd_x86_64.tar.gz" ;;
    freebsd/arm64) TAR_FILE="picoclaw_Freebsd_arm64.tar.gz" ;;
    freebsd/armv7) TAR_FILE="picoclaw_Freebsd_armv7.tar.gz" ;;

    netbsd/x86_64) TAR_FILE="picoclaw_Netbsd_x86_64.tar.gz" ;;
    netbsd/arm64) TAR_FILE="picoclaw_Netbsd_arm64.tar.gz" ;;

    *) die "No asset mapping for $os/$arch" ;;
  esac
}

resolve_unix_system_layout() {
  LAYOUT_MODE="symlink"
  if [[ -d "/usr/local/share" ]]; then
    INSTALL_ROOT="/usr/local/share/picoclaw"
    LINK_DIR="/usr/local/bin"
    return
  fi

  if [[ -d "/usr/share" ]]; then
    INSTALL_ROOT="/usr/share/picoclaw"
    LINK_DIR="/usr/bin"
    return
  fi

  INSTALL_ROOT="$(create_secure_tmp_dir "picoclaw-install" || true)"
  [[ -n "$INSTALL_ROOT" ]] || die "Failed to create secure temporary install directory under /tmp"
  LINK_DIR="/usr/bin"
  LAYOUT_MODE="copy-executables"
}

resolve_layout() {
  local os="$1"

  if [[ "$INSTALL_MODE" == "user" ]]; then
    INSTALL_ROOT="$HOME/.local/share/picoclaw"
    LINK_DIR="$HOME/.local/bin"
    LAYOUT_MODE="user"
    return
  fi

  case "$os" in
    linux|macos|freebsd|netbsd)
      resolve_unix_system_layout
      ;;
    *)
      die "Bash installer does not support this OS: $os"
      ;;
  esac
}

install_via_package_manager_linux() {
  local tmp_dir="$1"
  local base
  base="$(base_url)"

  if [[ -n "$DEB_FILE" ]] && command_exists dpkg; then
    local deb_path="$tmp_dir/$DEB_FILE"
    download_file "${base}${DEB_FILE}" "$deb_path"
    dpkg -i "$deb_path"
    PM_RESULT="dpkg:$deb_path"
    return 0
  fi

  if [[ -n "$RPM_FILE" ]] && command_exists rpm; then
    local rpm_path="$tmp_dir/$RPM_FILE"
    download_file "${base}${RPM_FILE}" "$rpm_path"
    rpm -Uvh --replacepkgs "$rpm_path"
    PM_RESULT="rpm:$rpm_path"
    return 0
  fi

  PM_RESULT=""
  return 1
}

extract_archive() {
  local archive="$1"
  local dst="$2"
  mkdir -p "$dst"

  case "$archive" in
    *.tar.gz)
      tar -xzf "$archive" -C "$dst"
      ;;
    *.zip)
      if command_exists unzip; then
        unzip -o "$archive" -d "$dst" >/dev/null
      else
        die "unzip is required for zip archives"
      fi
      ;;
    *)
      die "Unsupported archive: $archive"
      ;;
  esac
}

resolve_flat_binary_path() {
  local root="$1"
  local name="$2"
  local candidate="$root/$name"

  if [[ -f "$candidate" ]]; then
    printf '%s\n' "$candidate"
    return 0
  fi

  return 1
}

link_expected_bins() {
  local root="$1"
  local link_dir="$2"

  mkdir -p "$link_dir"
  LINKED_PATHS=()

  local name bin target
  for name in "${EXPECTED_BINS[@]}"; do
    bin="$(resolve_flat_binary_path "$root" "$name" || true)"
    if [[ -z "$bin" ]]; then
      continue
    fi
    target="$link_dir/$name"
    rm -f "$target"
    ln -s "$bin" "$target"
    LINKED_PATHS+=("$target")
  done
}

copy_executables_to_bin() {
  local root="$1"
  local bin_dir="$2"

  mkdir -p "$bin_dir"
  COPIED_PATHS=()

  local name file dest
  for name in "${EXPECTED_BINS[@]}"; do
    file="$(resolve_flat_binary_path "$root" "$name" || true)"
    if [[ -z "$file" ]]; then
      continue
    fi
    if [[ ! -x "$file" ]]; then
      continue
    fi

    dest="$bin_dir/$name"
    cp -f "$file" "$dest"
    chmod +x "$dest" || true
    COPIED_PATHS+=("$dest")
  done
}

ensure_user_path_export() {
  local line='export PATH="$HOME/.local/bin:$PATH"'
  local profile

  if [[ -f "$HOME/.bashrc" ]]; then
    profile="$HOME/.bashrc"
  elif [[ -f "$HOME/.bash_profile" ]]; then
    profile="$HOME/.bash_profile"
  else
    profile="$HOME/.profile"
  fi

  touch "$profile"
  if ! grep -Fq "$line" "$profile"; then
    printf '\n%s\n' "$line" >> "$profile"
  fi
  USER_PROFILE="$profile"
}

warn_missing_user_local() {
  if [[ "$INSTALL_MODE" != "user" ]]; then
    return
  fi
  if [[ ! -d "$HOME/.local" ]]; then
    printf '\033[33mWarning: %s/.local does not exist. The installer will create it automatically.\033[0m\n' "$HOME"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      INSTALL_MODE="${2:-}"
      shift 2
      ;;
    --source)
      SOURCE="${2:-}"
      shift 2
      ;;
    --arch)
      ARCH_OVERRIDE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

[[ "$INSTALL_MODE" == "system" || "$INSTALL_MODE" == "user" ]] || die "--mode must be system or user"
[[ "$SOURCE" == "github" || "$SOURCE" == "cdn" ]] || die "--source must be github or cdn"

OS_ID="$(detect_os)"
ARCH_ID="$(detect_arch)"
assert_privileges
warn_missing_user_local

asset_for "$OS_ID" "$ARCH_ID"
resolve_layout "$OS_ID"

TMP_DIR="$(create_secure_tmp_dir "picoclaw-download")"
[[ -n "$TMP_DIR" ]] || die "Failed to create temporary working directory"
trap 'rm -rf "$TMP_DIR"' EXIT

log "Install mode: $INSTALL_MODE"
log "OS/Arch: $OS_ID/$ARCH_ID"

if [[ "$OS_ID" == "linux" && "$INSTALL_MODE" == "system" ]]; then
  if install_via_package_manager_linux "$TMP_DIR"; then
    log "Installed via package manager: ${PM_RESULT%%:*}"
    log "Package file: ${PM_RESULT#*:}"
    log "Install layout is managed by the package manager."
    exit 0
  fi
fi

ARCHIVE_PATH="$TMP_DIR/$TAR_FILE"
download_file "$(base_url)${TAR_FILE}" "$ARCHIVE_PATH"

mkdir -p "$INSTALL_ROOT"
extract_archive "$ARCHIVE_PATH" "$INSTALL_ROOT"

if [[ "$LAYOUT_MODE" == "copy-executables" ]]; then
  copy_executables_to_bin "$INSTALL_ROOT" "$LINK_DIR"
  log "Extracted to: $INSTALL_ROOT"
  log "Copied executable files to: $LINK_DIR"
  for p in "${COPIED_PATHS[@]:-}"; do
    [[ -n "$p" ]] && log " - $p"
  done
elif [[ "$LAYOUT_MODE" == "symlink" || "$LAYOUT_MODE" == "user" ]]; then
  link_expected_bins "$INSTALL_ROOT" "$LINK_DIR"
  log "Extracted to: $INSTALL_ROOT"
  log "Created links in: $LINK_DIR"
  for p in "${LINKED_PATHS[@]:-}"; do
    [[ -n "$p" ]] && log " - $p"
  done
fi

if [[ "$INSTALL_MODE" == "user" ]]; then
  ensure_user_path_export
  log "PATH export ensured in: $USER_PROFILE"
fi

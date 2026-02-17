#!/bin/bash

# PicoClaw Manager for Termux
# A simple script to install, configure, and manage PicoClaw

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

GITHUB_REPO="sipeed/picoclaw"
CONFIG_FILE="$HOME/.picoclaw/config.json"
DATA_DIR="$HOME/.picoclaw"
LOG_FILE="$DATA_DIR/picoclaw.log"
BIN_PATH="$PREFIX/bin/picoclaw"

show_header() {
    clear
    echo -e "${BLUE}"
    cat << "EOF"
██████╗ ██╗ ██████╗ ██████╗  ██████╗██╗      █████╗ ██╗    ██╗
██╔══██╗██║██╔════╝██╔═══██╗██╔════╝██║     ██╔══██╗██║    ██║
██████╔╝██║██║     ██║   ██║██║     ██║     ███████║██║ █╗ ██║
██╔═══╝ ██║██║     ██║   ██║██║     ██║     ██╔══██║██║███╗██║
██║     ██║╚██████╗╚██████╔╝╚██████╗███████╗██║  ██║╚███╔███╔╝
╚═╝     ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ 
                                                                                                                                                      
EOF
    echo -e "       🦞 PicoClaw Termux Manager"
    echo -e "==========================================${NC}"
}

is_android() {
    [[ "$(uname -o 2>/dev/null)" == "Android" ]] || [ -n "$ANDROID_ROOT" ] || [ -d "/data/data/com.termux" ]
}

check_dependencies() {
    echo -e "${YELLOW}[*] Checking dependencies...${NC}"

    deps=("curl" "jq" "tmux" "tar")

    to_install=()
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            to_install+=("$dep")
        fi
    done

    if [ ${#to_install[@]} -ne 0 ]; then
        echo -e "${YELLOW}[!] Installing missing packages: ${to_install[*]}${NC}"
        pkg update && pkg install -y "${to_install[@]}"
    else
        echo -e "${GREEN}[✓] All dependencies found.${NC}"
    fi
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        aarch64|arm64) echo "arm64" ;;
        armv7l|armv6l) echo "armv6" ;;
        x86_64|amd64)  echo "x86_64" ;;
        *)
            echo -e "${RED}[!] Unsupported architecture: $arch${NC}" >&2
            return 1
            ;;
    esac
}

get_latest_version() {
    curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | jq -r '.tag_name'
}

# Download prebuilt binary from GitHub Releases
install_from_release() {
    local version="$1"
    local arch="$2"

    # Android/Termux uses a dedicated Android binary (GOOS=android, PIE).
    # Standard Linux uses the regular Linux binary.
    local os_name="Linux"
    if is_android; then
        os_name="Android"
    fi

    local filename="picoclaw_${os_name}_${arch}.tar.gz"
    local url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${filename}"
    local tmpdir
    tmpdir=$(mktemp -d -p "${TMPDIR:-/tmp}")

    echo -e "${YELLOW}[*] Downloading ${filename}...${NC}"
    if ! curl -fSL --progress-bar "$url" -o "${tmpdir}/${filename}"; then
        echo -e "${RED}[!] Download failed. URL: ${url}${NC}"
        rm -rf "$tmpdir"
        return 1
    fi

    echo -e "${YELLOW}[*] Extracting binary...${NC}"
    tar -xzf "${tmpdir}/${filename}" -C "$tmpdir"

    if [ ! -f "${tmpdir}/picoclaw" ]; then
        echo -e "${RED}[!] Binary not found in archive.${NC}"
        rm -rf "$tmpdir"
        return 1
    fi

    mkdir -p "$(dirname "$BIN_PATH")"
    cp "${tmpdir}/picoclaw" "$BIN_PATH"
    chmod +x "$BIN_PATH"
    rm -rf "$tmpdir"
    return 0
}

install_picoclaw() {
    check_dependencies

    echo -e "${YELLOW}[*] Detecting architecture...${NC}"
    local arch
    arch=$(detect_arch)
    if [ $? -ne 0 ] || [ -z "$arch" ]; then
        echo -e "${RED}[!] Could not detect architecture. Aborting.${NC}"
        read -p "Press Enter to return to menu..."
        return
    fi
    echo -e "${GREEN}[✓] Architecture: $arch${NC}"

    echo -e "${YELLOW}[*] Fetching latest release...${NC}"
    local version
    version=$(get_latest_version)
    if [ -z "$version" ] || [ "$version" = "null" ]; then
        echo -e "${RED}[!] Could not fetch latest version. Check your internet connection.${NC}"
        read -p "Press Enter to return to menu..."
        return
    fi
    echo -e "${GREEN}[✓] Latest version: $version${NC}"

    install_from_release "$version" "$arch"

    if [ $? -ne 0 ]; then
        echo -e "${RED}[!] Installation failed.${NC}"
        read -p "Press Enter to return to menu..."
        return
    fi

    if ! command -v picoclaw &> /dev/null; then
        echo -e "${RED}[!] Installation failed. Is $PREFIX/bin in your PATH?${NC}"
    else
        echo -e "${GREEN}[✓] PicoClaw ${version} installed successfully!${NC}"
        echo -e "${YELLOW}[*] Running initial setup...${NC}"
        picoclaw onboard
    fi
    read -p "Press Enter to return to menu..."
}

configure_picoclaw() {
    if [ ! -f "$CONFIG_FILE" ]; then
        echo -e "${RED}[!] Config file not found. Please install first.${NC}"
        read -p "Press Enter to return to menu..."
        return
    fi

    echo -e "${BLUE}--- Configuration ---${NC}"
    
    # LLM Settings
    read -p "Enter Gemini API Key (or press enter to skip): " gemini_key
    if [ ! -z "$gemini_key" ]; then
        jq ".providers.gemini.api_key = \"$gemini_key\" | .agents.defaults.provider = \"gemini\" | .agents.defaults.model = \"gemini-2.0-flash-lite\"" "$CONFIG_FILE" > "$CONFIG_FILE.tmp" && mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
        echo -e "${GREEN}[✓] Gemini configured.${NC}"
    fi

    # Telegram Settings
    read -p "Enter Telegram Bot Token (or press enter to skip): " tg_token
    if [ ! -z "$tg_token" ]; then
        jq ".channels.telegram.enabled = true | .channels.telegram.token = \"$tg_token\" | .channels.telegram.allow_from = []" "$CONFIG_FILE" > "$CONFIG_FILE.tmp" && mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
        echo -e "${GREEN}[✓] Telegram configured.${NC}"
    fi

    echo -e "${GREEN}[✓] Configuration updated!${NC}"
    read -p "Press Enter to return to menu..."
}

manage_service() {
    echo -e "${BLUE}--- Service Management ---${NC}"
    echo "1) Start Gateway (in Tmux)"
    echo "2) Stop Gateway"
    echo "3) Restart Gateway"
    echo "4) View Logs"
    echo "5) Back"
    read -p "Select option: " subopt

    case $subopt in
        1)
            if tmux has-session -t picoclaw 2>/dev/null; then
                echo -e "${YELLOW}[!] Session 'picoclaw' already exists.${NC}"
            else
                mkdir -p "$DATA_DIR"
                tmux new-session -d -s picoclaw "picoclaw gateway 2>&1 | tee $LOG_FILE"
                echo -e "${GREEN}[✓] Gateway started in tmux session 'picoclaw'.${NC}"
            fi
            ;;
        2)
            pkill -9 picoclaw
            tmux kill-session -t picoclaw 2>/dev/null
            echo -e "${YELLOW}[*] Gateway stopped.${NC}"
            ;;
        3)
            pkill -9 picoclaw
            tmux kill-session -t picoclaw 2>/dev/null
            sleep 1
            mkdir -p "$DATA_DIR"
            tmux new-session -d -s picoclaw "picoclaw gateway 2>&1 | tee $LOG_FILE"
            echo -e "${GREEN}[✓] Gateway restarted.${NC}"
            ;;
        4)
            if [ -f "$LOG_FILE" ]; then
                tail -f "$LOG_FILE"
            else
                echo -e "${YELLOW}[!] No log file found. Start the gateway first.${NC}"
            fi
            ;;
        *) return ;;
    esac
    read -p "Press Enter to return to menu..."
}

uninstall_picoclaw() {
    read -p "Are you sure you want to uninstall and delete all data? (y/N): " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        pkill -9 picoclaw
        tmux kill-session -t picoclaw 2>/dev/null
        rm -f "$BIN_PATH"
        rm -rf "$DATA_DIR"
        echo -e "${GREEN}[✓] PicoClaw uninstalled.${NC}"
    fi
    read -p "Press Enter to return to menu..."
}

network_diagnostics() {
    echo -e "${BLUE}--- Network Diagnostics ---${NC}"
    echo -e "${YELLOW}[*] Testing DNS resolution...${NC}"
    if nslookup google.com &> /dev/null; then
        echo -e "${GREEN}[✓] DNS resolution working.${NC}"
    else
        echo -e "${RED}[!] DNS resolution failed.${NC}"
        echo -e "${YELLOW}[TIP] Try running: echo \"nameserver 8.8.8.8\" > \$PREFIX/etc/resolv.conf${NC}"
        read -p "Would you like to apply this fix now? (y/N): " fix_dns
        if [[ "$fix_dns" == "y" || "$fix_dns" == "Y" ]]; then
            mkdir -p "$PREFIX/etc"
            echo "nameserver 8.8.8.8" > "$PREFIX/etc/resolv.conf"
            echo -e "${GREEN}[✓] DNS fix applied.${NC}"
        fi
    fi

    echo -e "\n${YELLOW}[*] Testing API connectivity...${NC}"
    if curl -s --connect-timeout 5 https://generativelanguage.googleapis.com &> /dev/null; then
        echo -e "${GREEN}[✓] Google API reachable.${NC}"
    else
        echo -e "${RED}[!] Google API unreachable.${NC}"
    fi

    if curl -s --connect-timeout 5 https://api.openai.com &> /dev/null; then
        echo -e "${GREEN}[✓] OpenAI API reachable.${NC}"
    else
        echo -e "${RED}[!] OpenAI API unreachable.${NC}"
    fi

    if curl -s --connect-timeout 5 https://open.bigmodel.cn &> /dev/null; then
        echo -e "${GREEN}[✓] Zhipu (BigModel) API reachable.${NC}"
    else
        echo -e "${RED}[!] Zhipu (BigModel) API unreachable.${NC}"
    fi

    if curl -s --connect-timeout 5 https://api.moonshot.cn &> /dev/null; then
        echo -e "${GREEN}[✓] Moonshot API reachable.${NC}"
    else
        echo -e "${RED}[!] Moonshot API unreachable.${NC}"
    fi

    echo -e "\n${YELLOW}[TIP] If you are in a restricted network, configure a proxy in option 2.${NC}"
    read -p "Press Enter to return to menu..."
}

while true; do
    show_header
    echo "1) Install / Update PicoClaw"
    echo "2) Configure (API Keys / Telegram)"
    echo "3) Start / Stop / Logs"
    echo "4) Show Status"
    echo "5) Network Diagnostics"
    echo "6) Uninstall"
    echo "7) Exit"
    read -p "Select option: " opt

    case $opt in
        1) install_picoclaw ;;
        2) configure_picoclaw ;;
        3) manage_service ;;
        4) picoclaw status; read -p "Press Enter to return to menu..." ;;
        5) network_diagnostics ;;
        6) uninstall_picoclaw ;;
        7) exit 0 ;;
        *) echo -e "${RED}[!] Invalid option${NC}"; sleep 1 ;;
    esac
done

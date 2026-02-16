#!/bin/bash

# PicoClaw Manager for Termux
# A simple script to install, configure, and manage PicoClaw

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CONFIG_FILE="$HOME/.picoclaw/config.json"
REPO_DIR="$HOME/.picoclaw-repo"
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

check_dependencies() {
    echo -e "${YELLOW}[*] Checking dependencies...${NC}"
    deps=("golang" "git" "make" "jq" "tmux")
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

install_picoclaw() {
    check_dependencies
    
    echo -e "${YELLOW}[*] Cloning repository...${NC}"
    if [ -d "$REPO_DIR" ]; then
        cd "$REPO_DIR" && git pull
    else
        git clone https://github.com/sipeed/picoclaw.git "$REPO_DIR"
        cd "$REPO_DIR"
    fi

    echo -e "${YELLOW}[*] Building PicoClaw...${NC}"
    # Adjust Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f1,2,3)
    sed -i "s/go 1.25.7/go $GO_VERSION/" go.mod
    
    export CGO_ENABLED=0
    make deps
    make build

    echo -e "${YELLOW}[*] Installing to bin...${NC}"
    cp build/picoclaw-linux-arm64 "$BIN_PATH"
    chmod +x "$BIN_PATH"
    
    if ! command -v picoclaw &> /dev/null; then
        echo -e "${RED}[!] Installation failed. Bin not in PATH?${NC}"
    else
        echo -e "${GREEN}[✓] PicoClaw installed successfully!${NC}"
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
                tmux new-session -d -s picoclaw "picoclaw gateway | tee $REPO_DIR/picoclaw.log"
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
            tmux new-session -d -s picoclaw "picoclaw gateway | tee $REPO_DIR/picoclaw.log"
            echo -e "${GREEN}[✓] Gateway restarted.${NC}"
            ;;
        4)
            tail -f "$REPO_DIR/picoclaw.log"
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
        rm "$BIN_PATH"
        rm -rf "$REPO_DIR"
        rm -rf "$HOME/.picoclaw"
        echo -e "${GREEN}[✓] PicoClaw uninstalled.${NC}"
    fi
    read -p "Press Enter to return to menu..."

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

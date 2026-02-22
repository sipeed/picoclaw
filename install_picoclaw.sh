#!/bin/bash
set -e

INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="picoclaw"
URL="https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
BOLD='\033[1m'
RESET='\033[0m'

print_step() {
    echo -e "${BLUE}==>${RESET} ${BOLD}$1${RESET}"
}

print_success() {
    echo -e "${GREEN}✓${RESET} $1"
}

print_error() {
    echo -e "${RED}✗${RESET} $1"
}

get_latest_version() {
    local repo="$1"
    curl -sI "https://github.com/${repo}/releases/latest" 2>/dev/null | \
        grep -i "location:" | sed 's|.*/tag/||' | tr -d '\r\n'
}

echo ""
echo -e "${BOLD}  PicoClaw Installer${RESET}"
echo "  ─────────────────────"
echo ""

# Ask user which version to install
echo -e "${BOLD}Select version to install:${RESET}"
echo "  1) Original (sipeed/picoclaw)"
echo "  2) Fork (muava12/picoclaw-fork)"
echo ""
read -p "Choose [1-2] (default: 2): " version_choice < /dev/tty

if [[ "$version_choice" == "1" ]]; then
    URL="https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz"
    REPO="sipeed/picoclaw"
    echo ""
    print_step "Selected: ${BOLD}Original version${RESET}"
else
    URL="https://github.com/muava12/picoclaw-fork/releases/latest/download/picoclaw_Linux_arm64.tar.gz"
    REPO="muava12/picoclaw-fork"
    echo ""
    print_step "Selected: ${BOLD}Fork version${RESET}"
fi

print_step "Checking for latest version..."
VERSION=$(get_latest_version "$REPO")
if [ -n "$VERSION" ]; then
    echo "    Latest: $VERSION"
else
    echo "    (unable to fetch version)"
fi

# Check if already installed
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>/dev/null | sed -n 's/.*\(v\?[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*[^ ]*\).*/\1/p' | head -1)
    # Strip 'v' prefix if present
    INSTALLED_VERSION=${INSTALLED_VERSION#v}
    if [ -n "$INSTALLED_VERSION" ]; then
        print_success "Already installed: $BINARY_NAME v$INSTALLED_VERSION"
        
        if [ -n "$VERSION" ]; then
            # Strip 'v' prefix for comparison
            VERSION_NUM=${VERSION#v}
            if [ "$INSTALLED_VERSION" = "$VERSION_NUM" ]; then
                echo ""
                print_success "You already have the latest version!"
                echo ""
                echo -e "Run: ${BOLD}picoclaw onboard${RESET} to get started"
                echo ""
                exit 0
            else
                echo ""
                echo -e "${BLUE}!${RESET} Update available: v$INSTALLED_VERSION → $VERSION"
                echo ""
                if [ -t 0 ]; then
                    read -p "Do you want to update? [y/N] " -n 1 -r
                else
                    read -p "Do you want to update? [y/N] " -n 1 -r < /dev/tty
                fi
                echo ""
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    print_step "Update cancelled"
                    echo ""
                    exit 0
                fi
            fi
        fi
    fi
fi

mkdir -p "$INSTALL_DIR"
print_success "Install directory: $INSTALL_DIR"

print_step "Downloading $BINARY_NAME (arm64)..."
cd /tmp
if command -v pv &> /dev/null; then
    curl -L "$URL" | pv -b -p -e -r > picoclaw.tar.gz
else
    curl -L --progress-bar "$URL" -o picoclaw.tar.gz
fi

print_step "Extracting..."
tar -xzf picoclaw.tar.gz "$BINARY_NAME"
chmod +x "$BINARY_NAME"

print_step "Installing..."
mv -f "$BINARY_NAME" "$INSTALL_DIR/"
rm -f picoclaw.tar.gz

print_success "Installed to $INSTALL_DIR/$BINARY_NAME"
print_success "Version: $VERSION"

echo ""
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    print_step "Fixing PATH..."
    if ! grep -q "$INSTALL_DIR" ~/.bashrc 2>/dev/null; then
        echo '' >> ~/.bashrc
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
        print_success "Added to ~/.bashrc"
    else
        print_success "Already configured in ~/.bashrc"
    fi
    export PATH="$INSTALL_DIR:$PATH"
    print_success "PATH updated for current session"
else
    print_success "Already in PATH"
fi

echo ""
echo -e "Run: ${BOLD}picoclaw onboard${RESET} to get started"
echo ""

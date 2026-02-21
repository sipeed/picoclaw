#!/bin/bash

set -e

# PicoClaw Installer
# Downloads and installs PicoClaw from GitHub releases

REPO_OWNER="sipeed"
REPO_NAME="picoclaw"
GITHUB_API="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}"
BINARY_NAME="picoclaw"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print usage
usage() {
    cat << EOF
PicoClaw Installer - Download and install PicoClaw from GitHub releases

Usage: $0 [OPTIONS] [VERSION]

Arguments:
    VERSION              Version to install/download (e.g., v0.1.2). Default: latest

Options:
    -h, --help           Show this help message
    -l, --list-versions  List all available versions
    -a, --list-assets    List assets for a specific version (or latest if not specified)
    -i, --install-dir    Installation directory (default: ~/.local/bin)
    -d, --download       Download only (do not install), saves to current directory
    -y, --yes            Skip confirmation prompt

Examples:
    $0                          Install latest version
    $0 v0.1.2                   Install specific version
    $0 -l                       List all available versions
    $0 -a                       List assets for latest version
    $0 -a v0.1.2                List assets for specific version
    $0 -i /usr/local/bin        Install latest to custom directory
    $0 -i /usr/local/bin v0.1.2 Install specific version to custom directory
    $0 -d                       Download latest version (current directory)
    $0 -d v0.1.2                Download specific version (current directory)

Environment Variables:
    INSTALL_DIR           Override default installation directory

EOF
    exit 0
}

# Print colored message
msg() {
    local color="$1"
    shift
    printf "${color}%s${NC}\n" "$*"
}

# Check for required dependencies
check_dependencies() {
    local missing=()

    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        missing+=("curl or wget")
    fi

    if ! command -v jq >/dev/null 2>&1 && ! command -v grep >/dev/null 2>&1; then
        missing+=("jq or grep")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        msg "$RED" "Error: Missing required dependencies: ${missing[*]}"
        exit 1
    fi
}

# Download function that works with both curl and wget
download_file() {
    local url="$1"
    local output="$2"

    if command -v curl >/dev/null 2>&1; then
        if [ -n "$output" ]; then
            curl -fsSL -o "$output" "$url"
        else
            curl -fsSL "$url"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [ -n "$output" ]; then
            wget -q -O "$output" "$url"
        else
            wget -q -O - "$url"
        fi
    else
        return 1
    fi
}

# Get JSON value (using jq if available, otherwise grep)
get_json_value() {
    local json="$1"
    local query="$2"

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r "$query"
    else
        # Fallback using grep for simple queries
        echo "$json" | grep -m1 "\"$query\"" | cut -d'"' -f4
    fi
}

# Get latest version tag
get_latest_version() {
    local json
    json=$(download_file "${GITHUB_API}/releases/latest")

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r '.tag_name'
    else
        echo "$json" | grep -m1 '"tag_name"' | cut -d'"' -f4
    fi
}

# List all available versions
list_versions() {
    msg "$BLUE" "Fetching available versions..."

    local json
    json=$(download_file "${GITHUB_API}/releases")

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r '.[] | "\(.tag_name)\t\(.published_at)\t\(.name // "")"' | while IFS=$'\t' read -r tag date name; do
            date_short=$(echo "$date" | cut -d'T' -f1)
            printf "  ${GREEN}%-12s${NC} %s" "$tag" "$date_short"
            if [ -n "$name" ] && [ "$name" != "null" ]; then
                printf " - %s" "$name"
            fi
            printf "\n"
        done
    else
        # Fallback: extract version tags using grep
        echo "$json" | grep '"tag_name"' | cut -d'"' -f4 | head -20
    fi
}

# List assets for a version
list_assets() {
    local version="$1"

    if [ -z "$version" ]; then
        version=$(get_latest_version)
        msg "$BLUE" "Listing assets for latest version: $version"
    else
        msg "$BLUE" "Listing assets for version: $version"
    fi

    local json
    if [ "$version" = "latest" ]; then
        json=$(download_file "${GITHUB_API}/releases/latest")
    else
        json=$(download_file "${GITHUB_API}/releases/tags/${version}")
    fi

    if [ -z "$json" ] || echo "$json" | grep -q '"message": "Not Found"'; then
        msg "$RED" "Error: Version '$version' not found"
        exit 1
    fi

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r '.assets[] | "\(.name)\t\(.size)\t\(.browser_download_url)"' | while IFS=$'\t' read -r name size url; do
            # Convert size to human readable
            if [ "$size" -gt 1048576 ]; then
                size_fmt="$((size / 1048576))MB"
            elif [ "$size" -gt 1024 ]; then
                size_fmt="$((size / 1024))KB"
            else
                size_fmt="${size}B"
            fi
            printf "  ${GREEN}%-30s${NC} %8s\n" "$name" "$size_fmt"
        done
    else
        # Fallback: extract asset names using grep
        echo "$json" | grep '"name"' | head -20 | while read -r line; do
            name=$(echo "$line" | cut -d'"' -f4)
            printf "  %s\n" "$name"
        done
    fi
}

# Detect platform (matches goreleaser asset naming: Darwin_arm64, Linux_x86_64, etc.)
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Darwin) os="Darwin" ;;
        Linux) os="Linux" ;;
        *) msg "$RED" "Error: Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        armv6|armv7l|armhf) arch="armv6" ;;
        riscv64) arch="riscv64" ;;
        mips64) arch="mips64" ;;
        s390x) arch="s390x" ;;
        *) msg "$RED" "Error: Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    # Detect Rosetta 2 on macOS - use native arm64 binary instead of x86_64
    if [ "$os" = "Darwin" ] && [ "$arch" = "x86_64" ]; then
        if [ "$(sysctl -n sysctl.proc_translated 2>/dev/null)" = "1" ]; then
            arch="arm64"
        fi
    fi

    echo "${os}_${arch}"
}

# Find matching asset for platform
find_asset_url() {
    local json="$1"
    local platform="$2"

    if command -v jq >/dev/null 2>&1; then
        # Try exact match first
        local url
        url=$(echo "$json" | jq -r --arg platform "$platform" '.assets[] | select(.name | contains($platform)) | .browser_download_url' | head -1)
        if [ -n "$url" ]; then
            echo "$url"
            return 0
        fi
    else
        # Fallback using grep
        local url
        url=$(echo "$json" | grep -i "$platform" | grep 'browser_download_url' | head -1 | cut -d'"' -f4)
        if [ -n "$url" ]; then
            echo "$url"
            return 0
        fi
    fi

    return 1
}

# Download only (no install) - saves to current directory
download_picoclaw() {
    local version="$1"
    local skip_confirm="$2"

    # Default to latest if no version specified
    if [ -z "$version" ]; then
        version="latest"
    fi

    # Detect platform
    local platform
    platform=$(detect_platform)
    msg "$BLUE" "Detected platform: $platform"

    # Get release info
    local json
    if [ "$version" = "latest" ]; then
        msg "$BLUE" "Fetching latest version..."
        json=$(download_file "${GITHUB_API}/releases/latest")
        version=$(echo "$json" | grep -m1 '"tag_name"' | cut -d'"' -f4)
        if command -v jq >/dev/null 2>&1; then
            version=$(echo "$json" | jq -r '.tag_name')
        fi
    else
        msg "$BLUE" "Fetching version $version..."
        json=$(download_file "${GITHUB_API}/releases/tags/${version}")
    fi

    if [ -z "$json" ] || echo "$json" | grep -q '"message": "Not Found"'; then
        msg "$RED" "Error: Version '$version' not found"
        exit 1
    fi

    # Find asset URL
    local asset_url
    asset_url=$(find_asset_url "$json" "$platform")

    if [ -z "$asset_url" ]; then
        msg "$RED" "Error: No asset found for platform '$platform'"
        msg "$YELLOW" "Available assets:"
        list_assets "$version"
        exit 1
    fi

    local asset_name
    asset_name=$(basename "$asset_url")

    msg "$GREEN" "Found: $asset_name"

    # Show download details
    local dest_dir
    dest_dir="$(pwd)"
    local dest_path="${dest_dir}/${asset_name}"

    echo ""
    msg "$BLUE" "Download Details:"
    echo "  Version:    $version"
    echo "  Platform:   $platform"
    echo "  Asset:      $asset_name"
    echo "  Save to:    $dest_path"
    echo ""

    # Confirm download
    if [ "$skip_confirm" != "true" ]; then
        read -p "Proceed with download? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ -n $REPLY ]]; then
            msg "$YELLOW" "Download cancelled"
            exit 0
        fi
    fi

    # Check if file already exists
    if [ -f "$dest_path" ]; then
        msg "$YELLOW" "File already exists: $dest_path"
        if [ "$skip_confirm" != "true" ]; then
            read -p "Overwrite? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                msg "$YELLOW" "Download cancelled"
                exit 0
            fi
        fi
    fi

    # Download
    msg "$BLUE" "Downloading ${asset_name}..."

    if ! download_file "$asset_url" "$dest_path"; then
        msg "$RED" "Error: Download failed"
        rm -f "$dest_path"
        exit 1
    fi

    # Show success message
    local file_size
    file_size=$(ls -lh "$dest_path" | awk '{print $5}')

    echo ""
    msg "$GREEN" "✓ Download complete!"
    echo ""
    echo "  File:   $dest_path"
    echo "  Size:   $file_size"
    echo ""

    # If archive, show extraction hint
    if [[ "$asset_name" == *.tar.gz ]] || [[ "$asset_name" == *.tgz ]]; then
        msg "$BLUE" "To extract: tar -xzf $asset_name"
    elif [[ "$asset_name" == *.zip ]]; then
        msg "$BLUE" "To extract: unzip $asset_name"
    fi
}

# Install picoclaw
install_picoclaw() {
    local version="$1"
    local skip_confirm="$2"

    # Default to latest if no version specified
    if [ -z "$version" ]; then
        version="latest"
    fi

    # Detect platform
    local platform
    platform=$(detect_platform)
    msg "$BLUE" "Detected platform: $platform"

    # Get release info
    local json
    if [ "$version" = "latest" ]; then
        msg "$BLUE" "Fetching latest version..."
        json=$(download_file "${GITHUB_API}/releases/latest")
        version=$(echo "$json" | grep -m1 '"tag_name"' | cut -d'"' -f4)
        if command -v jq >/dev/null 2>&1; then
            version=$(echo "$json" | jq -r '.tag_name')
        fi
    else
        msg "$BLUE" "Fetching version $version..."
        json=$(download_file "${GITHUB_API}/releases/tags/${version}")
    fi

    if [ -z "$json" ] || echo "$json" | grep -q '"message": "Not Found"'; then
        msg "$RED" "Error: Version '$version' not found"
        exit 1
    fi

    # Find asset URL
    local asset_url
    asset_url=$(find_asset_url "$json" "$platform")

    if [ -z "$asset_url" ]; then
        msg "$RED" "Error: No asset found for platform '$platform'"
        msg "$YELLOW" "Available assets:"
        list_assets "$version"
        exit 1
    fi

    local asset_name
    asset_name=$(basename "$asset_url")

    msg "$GREEN" "Found: $asset_name"

    # Show installation details
    echo ""
    msg "$BLUE" "Installation Details:"
    echo "  Version:      $version"
    echo "  Platform:     $platform"
    echo "  Asset:        $asset_name"
    echo "  Install Dir:  $INSTALL_DIR"
    echo ""

    # Confirm installation
    if [ "$skip_confirm" != "true" ]; then
        read -p "Proceed with installation? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ -n $REPLY ]]; then
            msg "$YELLOW" "Installation cancelled"
            exit 0
        fi
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Download binary
    local tmp_file
    tmp_file=$(mktemp)
    local binary_path="${INSTALL_DIR}/${BINARY_NAME}"

    msg "$BLUE" "Downloading ${asset_name}..."

    if ! download_file "$asset_url" "$tmp_file"; then
        msg "$RED" "Error: Download failed"
        rm -f "$tmp_file"
        exit 1
    fi

    # Handle tar.gz archives
    if [[ "$asset_name" == *.tar.gz ]] || [[ "$asset_name" == *.tgz ]]; then
        msg "$BLUE" "Extracting archive..."
        local extract_dir
        extract_dir=$(mktemp -d)
        tar -xzf "$tmp_file" -C "$extract_dir"
        mv "${extract_dir}/${BINARY_NAME}" "$binary_path" 2>/dev/null || \
            mv "${extract_dir}/"*"/${BINARY_NAME}" "$binary_path" 2>/dev/null || \
            mv "${extract_dir}/"* "$binary_path" 2>/dev/null
        rm -rf "$extract_dir" "$tmp_file"
    elif [[ "$asset_name" == *.zip ]]; then
        msg "$BLUE" "Extracting archive..."
        local extract_dir
        extract_dir=$(mktemp -d)
        unzip -q "$tmp_file" -d "$extract_dir"
        mv "${extract_dir}/${BINARY_NAME}" "$binary_path" 2>/dev/null || \
            mv "${extract_dir}/"*"/${BINARY_NAME}" "$binary_path" 2>/dev/null || \
            mv "${extract_dir}/"* "$binary_path" 2>/dev/null
        rm -rf "$extract_dir" "$tmp_file"
    else
        # Assume it's a raw binary
        mv "$tmp_file" "$binary_path"
    fi

    # Make executable
    chmod +x "$binary_path"

    # Verify installation
    if ! command -v "$binary_path" >/dev/null 2>&1; then
        msg "$YELLOW" "Note: $INSTALL_DIR is not in your PATH"
        msg "$YELLOW" "Add it to your PATH by adding this to your shell profile:"
        echo ""
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
    fi

    # Show success message
    local installed_version
    installed_version=$("$binary_path" --version 2>/dev/null || echo "$version")

    echo ""
    msg "$GREEN" "✓ PicoClaw installed successfully!"
    echo ""
    echo "  Version:  $installed_version"
    echo "  Location: $binary_path"
    echo ""

    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        msg "$YELLOW" "To use picoclaw, add $INSTALL_DIR to your PATH:"
        echo ""
        echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        msg "$YELLOW" "Or move it to a directory in your PATH:"
        echo ""
        echo "    mv $binary_path /usr/local/bin/"
        echo ""
    fi
}

# Parse command line arguments
POSITIONAL_ARGS=()
LIST_VERSIONS=false
LIST_ASSETS=""
SKIP_CONFIRM=false
DOWNLOAD_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        -l|--list-versions)
            LIST_VERSIONS=true
            shift
            ;;
        -a|--list-assets)
            LIST_ASSETS="true"
            shift
            # Check if next arg is a version (not another flag)
            if [[ -n "${1:-}" ]] && [[ ! "$1" =~ ^- ]]; then
                LIST_ASSETS="$1"
                shift
            fi
            ;;
        -i|--install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -d|--download)
            DOWNLOAD_ONLY=true
            shift
            ;;
        -y|--yes)
            SKIP_CONFIRM=true
            shift
            ;;
        -*)
            msg "$RED" "Error: Unknown option: $1"
            echo ""
            usage
            ;;
        *)
            POSITIONAL_ARGS+=("$1")
            shift
            ;;
    esac
done

# Restore positional args
set -- "${POSITIONAL_ARGS[@]}"

# Check dependencies
check_dependencies

# Handle list commands
if [ "$LIST_VERSIONS" = true ]; then
    list_versions
    exit 0
fi

if [ -n "$LIST_ASSETS" ]; then
    if [ "$LIST_ASSETS" = "true" ]; then
        list_assets
    else
        list_assets "$LIST_ASSETS"
    fi
    exit 0
fi

# Download or install
VERSION="${1:-}"
if [ "$DOWNLOAD_ONLY" = true ]; then
    download_picoclaw "$VERSION" "$SKIP_CONFIRM"
else
    install_picoclaw "$VERSION" "$SKIP_CONFIRM"
fi

#!/usr/bin/env bash
set -euo pipefail

# ── project constants ────────────────────────────────────────
BINARY_NAME="picoclaw"
CMD_DIR="cmd/${BINARY_NAME}"
BUILD_DIR="build"

# ── Go flags ─────────────────────────────────────────────────
GO="${GO:-go}"
GOFLAGS="${GOFLAGS:--v -tags stdjson}"

# ── version info (injected via ldflags) ──────────────────────
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
GIT_COMMIT="$(git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")"
BUILD_TIME="$(date +%FT%T%z)"
GO_VERSION="$($GO version | awk '{print $3}')"
LDFLAGS="-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.goVersion=${GO_VERSION} -s -w"

# ── platform detection ───────────────────────────────────────
detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$arch" in
        x86_64)     arch="amd64" ;;
        aarch64)    arch="arm64" ;;
        loongarch64) arch="loong64" ;;
    esac

    echo "${os} ${arch}"
}

# ── generate ─────────────────────────────────────────────────
generate() {
    echo "Run generate..."
    rm -rf "./${CMD_DIR}/workspace" 2>/dev/null || true
    $GO generate ./...
    echo "Run generate complete"
}

# ── build single platform ────────────────────────────────────
build_one() {
    local goos="$1" goarch="$2"
    local suffix="${BINARY_NAME}-${goos}-${goarch}"
    [[ "$goos" == "windows" ]] && suffix+=".exe"

    echo "Building ${BINARY_NAME} for ${goos}/${goarch}..."
    mkdir -p "$BUILD_DIR"
    GOOS="$goos" GOARCH="$goarch" $GO build $GOFLAGS -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${suffix}" "./${CMD_DIR}"
    echo "Build complete: ${BUILD_DIR}/${suffix}"
}

# ── build current platform ───────────────────────────────────
build_current() {
    generate
    read -r os arch <<< "$(detect_platform)"
    build_one "$os" "$arch"
    ln -sf "${BINARY_NAME}-${os}-${arch}" "${BUILD_DIR}/${BINARY_NAME}"
}

# ── build all platforms ──────────────────────────────────────
build_all() {
    generate
    echo "Building for multiple platforms..."
    mkdir -p "$BUILD_DIR"
    build_one linux   amd64
    build_one linux   arm64
    build_one linux   loong64
    build_one linux   riscv64
    build_one darwin  arm64
    build_one windows amd64
    echo "All builds complete"
}

# ── clean ────────────────────────────────────────────────────
clean() {
    echo "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    echo "Clean complete"
}

# ── main ─────────────────────────────────────────────────────
case "${1:-}" in
    --all)   build_all   ;;
    --clean) clean       ;;
    *)       build_current ;;
esac

#!/usr/bin/env bash
# Build PicoClaw core and web launcher without using make.
# Usage: ./scripts/build-without-make.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." >/dev/null 2>&1 && pwd)"
cd "${REPO_ROOT}"

GO_TAGS="goolm,stdjson"
EXE_EXT=""
UNAME_S="$(uname -s 2>/dev/null || echo unknown)"
case "${UNAME_S}" in
    *MINGW*|*MSYS*|*CYGWIN*|*Windows_NT*)
        EXE_EXT=".exe"
        ;;
esac

command -v go >/dev/null 2>&1 || {
    echo "ERROR: go is not installed or not on PATH."
    exit 1
}
command -v pnpm >/dev/null 2>&1 || {
    echo "ERROR: pnpm is not installed or not on PATH."
    exit 1
}

mkdir -p "${REPO_ROOT}/build"

echo "=== Generating Go code ==="
go generate ./...

echo ""
echo "=== Building PicoClaw core binary ==="
go build -tags "${GO_TAGS}" -o "${REPO_ROOT}/build/picoclaw${EXE_EXT}" ./cmd/picoclaw

echo ""
echo "=== Building PicoClaw web frontend assets ==="
cd "${REPO_ROOT}/web/frontend"
pnpm install --frozen-lockfile
pnpm build:backend

echo ""
echo "=== Building PicoClaw web launcher ==="
cd "${REPO_ROOT}"
go build -tags "${GO_TAGS}" -o "${REPO_ROOT}/build/picoclaw-launcher${EXE_EXT}" ./web/backend

echo ""
echo "=== Build complete ==="
echo "Core binary: ${REPO_ROOT}/build/picoclaw${EXE_EXT}"
echo "Web launcher: ${REPO_ROOT}/build/picoclaw-launcher${EXE_EXT}"

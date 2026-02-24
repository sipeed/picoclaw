#!/usr/bin/env bash
set -euo pipefail

GO="${GO:-go}"
GOLANGCI_LINT="${GOLANGCI_LINT:-golangci-lint}"

# ── individual checks ────────────────────────────────────────
do_deps() {
    echo "==> Downloading dependencies..."
    $GO mod download
    $GO mod verify
}

do_fmt() {
    echo "==> Formatting..."
    $GOLANGCI_LINT fmt
}

do_vet() {
    echo "==> Running vet..."
    $GO vet ./...
}

do_test() {
    echo "==> Running tests..."
    $GO test ./...
}

do_lint() {
    echo "==> Running linter..."
    $GOLANGCI_LINT run
}

do_all() {
    do_deps
    do_fmt
    do_vet
    do_test
    echo ""
    echo "All checks passed."
}

# ── main ─────────────────────────────────────────────────────
case "${1:-}" in
    test)  do_test ;;
    lint)  do_lint ;;
    fmt)   do_fmt  ;;
    vet)   do_vet  ;;
    deps)  do_deps ;;
    "")    do_all  ;;
    *)
        echo "Usage: $(basename "$0") [test|lint|fmt|vet|deps]"
        echo "  (no argument runs all checks)"
        exit 1
        ;;
esac

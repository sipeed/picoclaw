#!/usr/bin/env bash
set -euo pipefail

REQUIRED_GO_MAJOR=1
REQUIRED_GO_MINOR=25
PICOCLAW_HOME="${HOME}/.picoclaw"

passed=0
failed=0

check() {
    local name="$1" ok="$2" msg="$3"
    if [[ "$ok" == "true" ]]; then
        echo "  [ok] ${name}: ${msg}"
        passed=$((passed + 1))
    else
        echo "  [!!] ${name}: ${msg}"
        failed=$((failed + 1))
    fi
}

echo "PicoClaw Environment Setup"
echo "=========================="
echo ""

# ── Go ───────────────────────────────────────────────────────
echo "Checking dependencies..."
if command -v go &>/dev/null; then
    go_ver="$(go version | awk '{print $3}' | sed 's/go//')"
    go_major="${go_ver%%.*}"
    go_minor="${go_ver#*.}"
    go_minor="${go_minor%%.*}"

    if [[ "$go_major" -gt "$REQUIRED_GO_MAJOR" ]] || \
       { [[ "$go_major" -eq "$REQUIRED_GO_MAJOR" ]] && [[ "$go_minor" -ge "$REQUIRED_GO_MINOR" ]]; }; then
        check "Go" "true" "go${go_ver}"
    else
        check "Go" "false" "go${go_ver} (need >= ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR})"
    fi
else
    check "Go" "false" "not installed (https://go.dev/dl/)"
fi

# ── golangci-lint ────────────────────────────────────────────
if command -v golangci-lint &>/dev/null; then
    lint_ver="$(golangci-lint version --format short 2>/dev/null || echo "unknown")"
    check "golangci-lint" "true" "v${lint_ver}"
else
    check "golangci-lint" "false" "not installed (https://golangci-lint.run/welcome/install/)"
fi

# ── git ──────────────────────────────────────────────────────
if command -v git &>/dev/null; then
    git_ver="$(git --version | awk '{print $3}')"
    check "git" "true" "${git_ver}"
else
    check "git" "false" "not installed"
fi

# ── curl (for health checks) ────────────────────────────────
if command -v curl &>/dev/null; then
    check "curl" "true" "available"
else
    check "curl" "false" "not installed (needed for deploy health checks)"
fi

echo ""

# ── download dependencies ────────────────────────────────────
if command -v go &>/dev/null; then
    echo "Downloading Go dependencies..."
    go mod download
    go mod verify
    echo "  Dependencies ready."
    echo ""
fi

# ── picoclaw onboard ────────────────────────────────────────
if [[ ! -d "$PICOCLAW_HOME" ]]; then
    if command -v picoclaw &>/dev/null; then
        echo "Running picoclaw onboard..."
        picoclaw onboard
        echo ""
    else
        echo "Note: Run 'picoclaw onboard' after first install to initialize workspace."
        echo ""
    fi
else
    echo "Workspace: ${PICOCLAW_HOME} (exists)"
    echo ""
fi

# ── summary ──────────────────────────────────────────────────
echo "=========================="
if [[ "$failed" -eq 0 ]]; then
    echo "All checks passed (${passed}/${passed})."
    echo "Ready to build: scripts/build.sh"
else
    echo "${passed} passed, ${failed} failed."
    echo "Fix the issues above, then re-run this script."
fi

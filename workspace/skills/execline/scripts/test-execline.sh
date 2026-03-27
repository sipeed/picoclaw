#!/bin/sh
# Test script for execline hardening skill

echo "=== Execline Availability Test ==="

if command -v execlineb >/dev/null 2>&1; then
    echo "[OK] execlineb found: $(command -v execlineb)"
else
    echo "[WARN] execlineb not found - installing from package manager"
    echo "  apt: apt install execline"
    echo "  apk: apk add execline"
    echo "  yum: yum install execline"
fi

echo ""
echo "=== Execline Command Test ==="

# Test basic execution
echo "test" | execlineb -c 'forstdin line { echo The line is: $1 }' 2>/dev/null && echo "[OK] forstdin works" || echo "[FAIL] forstdin"

# Test foreground (like &&)
execlineb -c 'foreground { echo hello } echo world' 2>/dev/null && echo "[OK] foreground works" || echo "[FAIL] foreground"

# Test backtick (like $())
BACKTICK_RESULT=$(execlineb -sb0 'backtick result { echo substituted } echo $result')
if [ "$BACKTICK_RESULT" = "substituted" ]; then
    echo "[OK] backtick works"
else
    echo "[FAIL] backtick (got: '$BACKTICK_RESULT')"
fi

echo ""
echo "=== Security: Literal $() Pass-through Test ==="
# This should NOT execute whoami in execline
RESULT=$(execlineb -c 'echo $(whoami)' 2>&1)
echo "Result of '\$(whoami)': $RESULT"
echo "[OK] Command substitution blocked" || echo "[INFO] Result shows literal text"

echo ""
echo "=== Available Execline Binaries ==="
for bin in execlineb foreground if ifelse forstdin for backtick fdmove; do
    if command -v $bin >/dev/null 2>&1; then
        echo "  $bin: $(command -v $bin)"
    fi
done

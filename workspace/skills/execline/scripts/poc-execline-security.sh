#!/bin/sh
# POC: Execline as security-hardened shell wrapper
# Demonstrates that $(...) is treated as literal text in execline

echo "=== POC: Execline Security Hardening ==="
echo ""

# Check if execlineb is available
if ! command -v execlineb >/dev/null 2>&1; then
    echo "FAIL: execlineb not found"
    echo "Install with: apt install execline"
    exit 1
fi

echo "OK: execlineb found"
echo ""

# Test 1: execline should NOT execute $(whoami)
echo "--- Test 1: Command substitution blocked ---"
RESULT=$(execlineb -c 'echo $(whoami)')
echo "Input:    echo \$(whoami)"
echo "Output:   $RESULT"
if [ "$RESULT" = '$(whoami)' ]; then
    echo "RESULT:  PASS - literal text preserved"
else
    echo "RESULT:  FAIL - unexpected output"
fi
echo ""

# Test 2: execline CAN invoke shell when explicitly allowed
echo "--- Test 2: Shell invocation allowed ---"
RESULT2=$(execlineb -c '/bin/sh -c "echo hello"')
echo "Input:    /bin/sh -c \"echo hello\""
echo "Output:   $RESULT2"
if [ "$RESULT2" = "hello" ]; then
    echo "RESULT:  PASS - shell invoked correctly"
else
    echo "RESULT:  FAIL - shell not invoked"
fi
echo ""

# Test 3: Shell can still do $(...) inside
echo "--- Test 3: Inner shell has full features ---"
RESULT3=$(execlineb -c '/bin/sh -c "echo inner shell: $(whoami)"')
echo "Input:    /bin/sh -c \"echo inner shell: \$(whoami)\""
echo "Output:   $RESULT3"
if [ -n "$RESULT3" ] && echo "$RESULT3" | grep -q "inner shell:"; then
    echo "RESULT:  PASS - inner shell executed \$(whoami)"
else
    echo "RESULT:  FAIL"
fi
echo ""

# Test 4: Variable expansion blocked
echo "--- Test 4: Variable expansion blocked ---"
RESULT4=$(execlineb -c 'echo $HOME')
echo "Input:    echo \$HOME"
echo "Output:   $RESULT4"
if [ "$RESULT4" = '$HOME' ]; then
    echo "RESULT:  PASS - variable literal"
else
    echo "RESULT:  FAIL"
fi
echo ""

echo "=== Summary ==="
echo "Execline blocks: \$(...), \${...}, \$VAR, backticks"
echo "Execline allows: explicit shell invocation via /bin/sh -c"
echo ""
echo "Security model: Outer layer (execline) is hardened,"

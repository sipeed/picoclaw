#!/bin/bash
set -e
cd /home/picoclaw/.picoclaw/workspace
echo "Installing @playwright/test if needed..."
npm install @playwright/test --save-dev 2>&1 | grep -E "(added|up to date)" || true
echo ""
echo "Running Playwright test..."
echo "=================================================="
npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts --reporter=list 2>&1
echo "=================================================="

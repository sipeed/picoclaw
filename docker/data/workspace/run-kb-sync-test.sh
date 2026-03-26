#!/bin/bash
set -e

cd /home/picoclaw/.picoclaw/workspace

echo "=========================================="
echo "Running Playwright Test: Schedule KB Full Sync"
echo "=========================================="
echo ""

npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts --reporter=list

echo ""
echo "=========================================="
echo "Test execution completed"
echo "=========================================="

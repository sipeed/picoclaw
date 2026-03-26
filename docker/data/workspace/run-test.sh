#!/bin/bash
cd /home/picoclaw/.picoclaw/workspace
npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts --reporter=list 2>&1

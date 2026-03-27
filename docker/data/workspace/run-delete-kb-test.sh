#!/bin/bash
cd /home/picoclaw/.picoclaw/workspace
npx playwright test tests/knowledge-base/delete-kb.spec.ts --reporter=list

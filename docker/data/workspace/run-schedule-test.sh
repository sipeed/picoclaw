#!/bin/bash
cd /home/picoclaw/.picoclaw/workspace
npx playwright test tests/knowledge-base/edit-kb-schedule.spec.ts --reporter=list

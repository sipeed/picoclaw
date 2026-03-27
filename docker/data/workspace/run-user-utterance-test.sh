#!/bin/bash
cd /home/picoclaw/.picoclaw/workspace
npx playwright test tests/flow-designer/create-new-flow-user-utterance-node.spec.ts --reporter=list

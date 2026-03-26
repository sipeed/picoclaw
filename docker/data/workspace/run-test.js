#!/usr/bin/env node

const { execSync } = require('child_process');
const path = require('path');

const workspaceDir = '/home/picoclaw/.picoclaw/workspace';
process.chdir(workspaceDir);

console.log('📦 Installing @playwright/test...');
try {
  execSync('npm install @playwright/test --save-dev', { stdio: 'inherit' });
} catch (e) {
  console.error('Failed to install @playwright/test');
}

console.log('\n🚀 Running Playwright test...\n');
try {
  const result = execSync('npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts --reporter=list', { 
    stdio: 'pipe',
    encoding: 'utf-8'
  });
  console.log(result);
} catch (e) {
  console.log(e.stdout);
  console.log(e.stderr);
  process.exit(e.status || 1);
}

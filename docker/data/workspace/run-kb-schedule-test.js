#!/usr/bin/env node

const { execSync } = require('child_process');
const path = require('path');

const workspaceDir = '/home/picoclaw/.picoclaw/workspace';
process.chdir(workspaceDir);

console.log('📦 Checking Playwright installation...');
try {
  execSync('npm list @playwright/test --depth=0', { stdio: 'pipe' });
  console.log('✅ @playwright/test already installed\n');
} catch (e) {
  console.log('📥 Installing @playwright/test...');
  try {
    execSync('npm install @playwright/test --save-dev', { stdio: 'inherit' });
  } catch (err) {
    console.error('Failed to install @playwright/test');
  }
}

console.log('\n🚀 Running Playwright test: tests/knowledge-base/edit-kb-schedule.spec.ts\n');
console.log('='.repeat(70));

try {
  const result = execSync('npx playwright test tests/knowledge-base/edit-kb-schedule.spec.ts --reporter=list', { 
    stdio: 'inherit',
    encoding: 'utf-8'
  });
} catch (e) {
  process.exit(e.status || 1);
}

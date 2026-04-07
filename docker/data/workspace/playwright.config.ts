import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 120000,
  retries: 0,
  workers: 1,
  reporter: 'line',
  use: {
    headless: true,
    launchOptions: {
      args: ['--disable-dev-shm-usage', '--no-sandbox'],
    },
  },
});

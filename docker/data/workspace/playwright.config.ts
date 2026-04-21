import { defineConfig } from '@playwright/test';

export default defineConfig({
  use: {
    baseURL: 'https://dashboard.int3nt.info',
  },
  timeout: 60000,
  workers: 1, // Run tests sequentially to avoid state conflicts
});

import { defineConfig } from '@playwright/test';

export default defineConfig({
  use: {
    baseURL: process.env.BASE_URL || 'https://dashboard.int3nt.info',
  },
  timeout: 60000,
});

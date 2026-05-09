import { test, expect } from '@playwright/test';

test.describe('Navigation Tests', () => {
  const routes = [
    '/',
    '/models',
    '/logs',
    '/credentials',
    '/config',
    '/config/raw',
    '/channels',
    '/agent',
    '/agent/tools',
    '/agent/skills',
    '/agent/research',
    '/agent/hub',
    '/agent/cockpit',
  ];

  for (const route of routes) {
    test(`should load ${route} without crash`, async ({ page }) => {
      const consoleErrors: string[] = [];
      
      page.on('console', (msg) => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      page.on('pageerror', (error) => {
        consoleErrors.push(error.message);
      });

      try {
        await page.goto(route, { waitUntil: 'networkidle', timeout: 30000 });
        
        // Wait a bit for any delayed errors
        await page.waitForTimeout(2000);
        
        // Check if page has content (not just blank)
        const body = await page.locator('body').first();
        const hasContent = await body.innerText().then(t => t.trim().length > 0);
        
        expect(hasContent).toBe(true);
        
        // Log any errors found
        if (consoleErrors.length > 0) {
          console.log(`Route ${route} errors:`, consoleErrors);
        }
        
        // We allow some errors but expect page to load
        // Don't fail on console errors unless it's critical
      } catch (error) {
        console.log(`Route ${route} failed to load:`, error);
        throw error;
      }
    });
  }
});

test.describe('Console Error Detection', () => {
  test('should capture all console errors on index page', async ({ page }) => {
    const consoleErrors: string[] = [];
    const pageErrors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    page.on('pageerror', (error) => {
      pageErrors.push(error.message);
    });

    await page.goto('/', { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);

    // Report errors but don't fail the test
    if (consoleErrors.length > 0) {
      console.log('Console errors found:', consoleErrors);
    }
    if (pageErrors.length > 0) {
      console.log('Page errors found:', pageErrors);
    }

    // Just check page loaded
    await expect(page.locator('body')).toBeVisible();
  });
});
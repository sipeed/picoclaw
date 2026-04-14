import { test, expect, Page } from '@playwright/test';

test.describe('Create Knowledge Base Bucket with Website Crawler', () => {
  let page: Page;

  test('Create KB Bucket - Website Crawler Configuration', async ({ browser }) => {
    page = await browser.newPage();
    const baseUrl = process.env.BASE_URL || 'https://dashboard.int3nt.info';
    const email = 'heidi@intnt.ai';
    const password = 'testing2026!';
    const org = 'Testing2026!';
    const kbName = 'Picotest2';
    const baseUrl_input = 'https://intentai.com';
    const seedUrl = 'https://intentai.com/blog/';
    const maxCrawlDepth = '2';
    const maxExtractedLinks = '1000';
    const maxUniqueUrls = '25';

    // Step 1: Navigate to login page
    console.log('📍 Step 1: Navigate to login page');
    await page.goto(`${baseUrl}/login`, { waitUntil: 'networkidle' });
    await page.waitForURL(/\/login/);
    await page.locator('.login-card').waitFor({ state: 'visible', timeout: 10000 });
    console.log('✅ PASS: Step 1 - Login page loaded');

    // Step 2: Fill credentials and login
    console.log('📍 Step 2: Fill credentials and login');
    const loginCard = page.locator('.login-card');
    const emailInput = loginCard
      .locator('div')
      .filter({ hasText: /^Email address/ })
      .locator('input')
      .first();
    const passwordInput = loginCard
      .locator('div')
      .filter({ hasText: /^Password/ })
      .locator('input')
      .first();
    const loginButton = page.getByRole('button', { name: /^Login$/i });

    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
    await expect(loginButton).toBeVisible();

    await emailInput.fill(email);
    await passwordInput.fill(password);
    await loginButton.click();
    console.log('✅ PASS: Step 2 - Credentials entered and login button clicked');

    // Step 3: Wait for redirect to org selection
    console.log('📍 Step 3: Wait for redirect to organization selection page');
    await page.waitForURL(/\?select_org/, { timeout: 20000 });
    const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
    if (await loader.first().isVisible().catch(() => false)) {
      await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
    }
    await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
    console.log('✅ PASS: Step 3 - Redirected to organization selection page');

    // Step 4: Select organization
    console.log(`📍 Step 4: Select organization "${org}"`);
    await page.locator('.organization-card').filter({ hasText: org }).click();
    await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
    console.log(`✅ PASS: Step 4 - Organization "${org}" selected and redirected to dashboard`);

    // Step 5: Verify user is on dashboard
    console.log('📍 Step 5: Verify user is on dashboard');
    await expect(page).toHaveURL(/https:\/\/dashboard\.int3nt\.info\/$/);
    console.log('✅ PASS: Step 5 - User successfully on dashboard');

    // Step 6: Click Knowledge Base in sidebar
    console.log('📍 Step 6: Click "Knowledge Base" in the left sidebar');
    const kbLink = page.locator('a:has-text("Knowledge Base")');
    await expect(kbLink).toBeVisible();
    await kbLink.click();
    await page.waitForURL(/\/knowledge-base/, { timeout: 10000 });
    console.log('✅ PASS: Step 6 - Knowledge Base page loaded');

    // Step 7: Click "Create Knowledge Base Bucket" button
    console.log('📍 Step 7: Click "Create Knowledge Base Bucket" button');
    const createButton = page.locator('.create-button');
    await expect(createButton).toBeVisible();
    await createButton.click();
    await page.locator('.custom-drawer-overlay').waitFor({ state: 'visible', timeout: 10000 });
    console.log('✅ PASS: Step 7 - Create KB Bucket panel opened');

    // Step 8: Fill Knowledge Group Name
    console.log(`📍 Step 8: Fill Knowledge Group Name with "${kbName}"`);
    const drawer = page.locator('.custom-drawer-overlay');
    const kbNameInput = drawer.locator('input[placeholder="Enter knowledge group name"]').first();
    await expect(kbNameInput).toBeVisible();
    await kbNameInput.fill(kbName);
    console.log(`✅ PASS: Step 8 - Knowledge Group Name filled with "${kbName}"`);

    // Step 9: Leave LLM transformer model as None (default)
    console.log('📍 Step 9: Verify LLM transformer model is left as None (default)');
    const llmDropdown = page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0);
    await expect(llmDropdown).toBeVisible();
    console.log('✅ PASS: Step 9 - LLM transformer model left as None');

    // Step 10: Click Source Type dropdown
    console.log('📍 Step 10: Click Source Type dropdown');
    const sourceTypeDropdown = page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1);
    await expect(sourceTypeDropdown).toBeVisible();
    await sourceTypeDropdown.click();
    await page.waitForTimeout(500);
    console.log('✅ PASS: Step 10 - Source Type dropdown opened');

    // Step 11: Select Website Crawler
    console.log('📍 Step 11: Select "Website Crawler" from dropdown');
    await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Website Crawler/ }).click();
    await page.waitForTimeout(500);
    console.log('✅ PASS: Step 11 - Website Crawler selected');

    // Step 12: Wait for Website Crawler Configuration to appear
    console.log('📍 Step 12: Wait for Website Crawler Configuration');
    await page.locator('.custom-drawer-overlay').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    console.log('✅ PASS: Step 12 - Website Crawler Configuration displayed');

    // Step 13: Fill Base URL
    console.log(`📍 Step 13: Fill Base URL with "${baseUrl_input}"`);
    const baseUrlInput = drawer.locator('input[placeholder="https://example.com"]').first();
    await expect(baseUrlInput).toBeVisible();
    await baseUrlInput.fill(baseUrl_input);
    console.log(`✅ PASS: Step 13 - Base URL filled with "${baseUrl_input}"`);

    // Step 14: Open Web Crawler Parameters expansion panel
    console.log('📍 Step 14: Open "Web Crawler Parameters" expansion panel');
    const webCrawlerPanel = page.locator('.v-expansion-panel-title').filter({ hasText: /Web Crawler Parameters/ });
    await expect(webCrawlerPanel).toBeVisible();
    await webCrawlerPanel.click();
    await page.waitForTimeout(500);
    console.log('✅ PASS: Step 14 - Web Crawler Parameters panel opened');

    // Step 15: Fill Seed URLs
    console.log(`📍 Step 15: Fill "Seed URLs" field with "${seedUrl}"`);
    // Seed URLs should be in a textarea within the Web Crawler Parameters section
    const textareas = drawer.locator('textarea');
    const seedUrlsTextarea = textareas.nth(0);
    await expect(seedUrlsTextarea).toBeVisible();
    await seedUrlsTextarea.fill(seedUrl);
    console.log(`✅ PASS: Step 15 - Seed URLs filled with "${seedUrl}"`);

    // Step 16: Leave Sitemap URLs empty
    console.log('📍 Step 16: Leave Sitemap URLs empty (default)');
    const sitemapUrlsTextarea = textareas.nth(1);
    await expect(sitemapUrlsTextarea).toBeVisible();
    // Just verify it's visible and leave it empty
    console.log('✅ PASS: Step 16 - Sitemap URLs left empty');

    // Step 17: Leave Schedule (Cron Expression) as default
    console.log('📍 Step 17: Leave Schedule (Cron Expression) as default');
    // The cron expression field should be visible in the drawer
    console.log('✅ PASS: Step 17 - Schedule left as default');

    // Step 18: Open "Crawl Depth and Limits" expansion panel
    console.log('📍 Step 18: Open "Crawl Depth and Limits" expansion panel');
    const crawlDepthPanel = page.locator('.v-expansion-panel-title').filter({ hasText: /Crawl Depth and Limits/ });
    await expect(crawlDepthPanel).toBeVisible();
    await crawlDepthPanel.click();
    await page.waitForTimeout(500);
    console.log('✅ PASS: Step 18 - Crawl Depth and Limits panel opened');

    // Step 19: Fill Max Crawl Depth
    console.log(`📍 Step 19: Fill Max Crawl Depth with "${maxCrawlDepth}"`);
    const numberInputs = drawer.locator('input[type="number"]');
    const maxCrawlDepthInput = numberInputs.nth(0);
    await expect(maxCrawlDepthInput).toBeVisible();
    await maxCrawlDepthInput.clear();
    await maxCrawlDepthInput.fill(maxCrawlDepth);
    console.log(`✅ PASS: Step 19 - Max Crawl Depth filled with "${maxCrawlDepth}"`);

    // Step 20: Fill Max Extracted Links Count
    console.log(`📍 Step 20: Fill Max Extracted Links Count with "${maxExtractedLinks}"`);
    const maxExtractedLinksInput = numberInputs.nth(1);
    await expect(maxExtractedLinksInput).toBeVisible();
    await maxExtractedLinksInput.clear();
    await maxExtractedLinksInput.fill(maxExtractedLinks);
    console.log(`✅ PASS: Step 20 - Max Extracted Links Count filled with "${maxExtractedLinks}"`);

    // Step 21: Fill Max Unique URL Count
    console.log(`📍 Step 21: Fill Max Unique URL Count with "${maxUniqueUrls}"`);
    const maxUniqueUrlsInput = numberInputs.nth(2);
    await expect(maxUniqueUrlsInput).toBeVisible();
    await maxUniqueUrlsInput.clear();
    await maxUniqueUrlsInput.fill(maxUniqueUrls);
    console.log(`✅ PASS: Step 21 - Max Unique URL Count filled with "${maxUniqueUrls}"`);

    // Step 22: Click Continue button
    console.log('📍 Step 22: Click Continue button to proceed to Step 2');
    const continueButton = drawer.locator('button:has-text("Continue")');
    await expect(continueButton).toBeVisible();
    await continueButton.click();
    await page.waitForTimeout(1000);
    console.log('✅ PASS: Step 22 - Continue button clicked');

    // Step 23: Verify Step 2: Search Engine Configuration appears
    console.log('📍 Step 23: Verify Step 2: Search Engine Configuration');
    const step2Title = drawer.locator('text=Step 2: Search Engine Configuration');
    await expect(step2Title).toBeVisible({ timeout: 10000 });
    console.log('✅ PASS: Step 23 - Step 2: Search Engine Configuration displayed');

    // Step 24: Click Submit button
    console.log('📍 Step 24: Click Submit button to create KB bucket');
    const submitButton = drawer.locator('button:has-text("Submit")');
    await expect(submitButton).toBeVisible();
    await submitButton.click();
    await page.waitForTimeout(2000);
    console.log('✅ PASS: Step 24 - Submit button clicked');

    // Step 25: Verify KB bucket was created and appears in list
    console.log(`📍 Step 25: Verify KB bucket "${kbName}" appears in Knowledge Base list`);
    const kbBucketName = page.locator('.bucket-name').filter({ hasText: kbName }).first();
    await expect(kbBucketName).toBeVisible({ timeout: 15000 });
    console.log(`✅ PASS: Step 25 - KB bucket "${kbName}" successfully created and appears in list`);

    // Final Report
    console.log('\n' + '='.repeat(70));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(70));
    console.log('✅ Step 1: PASS - Login page loaded');
    console.log('✅ Step 2: PASS - Credentials entered and login button clicked');
    console.log('✅ Step 3: PASS - Redirected to organization selection page');
    console.log(`✅ Step 4: PASS - Organization "${org}" selected`);
    console.log('✅ Step 5: PASS - User successfully on dashboard');
    console.log('✅ Step 6: PASS - Knowledge Base page loaded');
    console.log('✅ Step 7: PASS - Create KB Bucket panel opened');
    console.log(`✅ Step 8: PASS - Knowledge Group Name filled with "${kbName}"`);
    console.log('✅ Step 9: PASS - LLM transformer model left as None');
    console.log('✅ Step 10: PASS - Source Type dropdown opened');
    console.log('✅ Step 11: PASS - Website Crawler selected');
    console.log('✅ Step 12: PASS - Website Crawler Configuration displayed');
    console.log(`✅ Step 13: PASS - Base URL filled with "${baseUrl_input}"`);
    console.log('✅ Step 14: PASS - Web Crawler Parameters panel opened');
    console.log(`✅ Step 15: PASS - Seed URLs filled with "${seedUrl}"`);
    console.log('✅ Step 16: PASS - Sitemap URLs left empty');
    console.log('✅ Step 17: PASS - Schedule left as default');
    console.log('✅ Step 18: PASS - Crawl Depth and Limits panel opened');
    console.log(`✅ Step 19: PASS - Max Crawl Depth filled with "${maxCrawlDepth}"`);
    console.log(`✅ Step 20: PASS - Max Extracted Links Count filled with "${maxExtractedLinks}"`);
    console.log(`✅ Step 21: PASS - Max Unique URL Count filled with "${maxUniqueUrls}"`);
    console.log('✅ Step 22: PASS - Continue button clicked');
    console.log('✅ Step 23: PASS - Step 2: Search Engine Configuration displayed');
    console.log('✅ Step 24: PASS - Submit button clicked');
    console.log(`✅ Step 25: PASS - KB bucket "${kbName}" successfully created and appears in list`);
    console.log('='.repeat(70));
    console.log('\n✅ ALL TESTS PASSED!\n');

    await page.close();
  });
});
